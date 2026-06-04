import { useState, useRef, useCallback, useEffect } from 'react'
import dayjs from 'dayjs'
import { uploadApi, categoryApi, llmApi, type OcrResult, type LLMSuggestion, type Category } from '../../services/api'
import { useResponsive } from '../../hooks/useResponsive'
import { toWebP, fileSHA256 } from '../../utils/imageUtils'

const OCR_TIMEOUT = 25000 // 25 秒超时，略短于服务端 30 秒

interface ImageItem {
  file: File
  webpFile?: File
  preview: string
  hash: string
  ocrResult?: OcrResult
  llmResult?: LLMSuggestion[]
  llmError?: string
  status: 'pending' | 'compressing' | 'ocr_loading' | 'llm_loading' | 'success' | 'timeout' | 'error'
  error?: string
}

export default function UploadOCR() {
  const { isMobile } = useResponsive()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [images, setImages] = useState<ImageItem[]>([])
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [categories, setCategories] = useState<Category[]>([])
  const [llmEnabled, setLlmEnabled] = useState(false)

  // 编辑字段
  const [editAmount, setEditAmount] = useState('')
  const [editDate, setEditDate] = useState('')
  const [editMerchant, setEditMerchant] = useState('')
  const [editType, setEditType] = useState<'income' | 'expense'>('expense')
  const [editCategoryId, setEditCategoryId] = useState<number | ''>('')
  const [editNote, setEditNote] = useState('')
  const [saving, setSaving] = useState(false)

  // 加载 LLM 配置
  useEffect(() => {
    llmApi.getConfig().then(r => {
      setLlmEnabled(r.data.data?.enabled ?? false)
    }).catch(() => {
      setLlmEnabled(false)
    })
  }, [])

  // 加载分类
  useEffect(() => {
    if (images.length === 0) return
    categoryApi.list().then(res => setCategories(res.data.data)).catch(() => {})
  }, [images.length])

  const selectedImage = images[selectedIndex]

  function applyLLM(llm: LLMSuggestion, cats?: Category[]) {
    const catList = cats ?? categories
    if (llm.type === 'income' || llm.type === 'expense') setEditType(llm.type)
    if ((llm.amount ?? 0) > 0) setEditAmount(String(llm.amount))
    if (llm.date) setEditDate(llm.date)
    if (llm.merchant_name) setEditMerchant(llm.merchant_name)
    if (llm.note) setEditNote(llm.note)
    if (llm.category_hint) {
      const type = llm.type ?? 'expense'
      const matched = catList.find(c => c.type === type && c.name.includes(llm.category_hint))
      if (matched) setEditCategoryId(matched.id)
    }
  }

  async function processFile(img: ImageItem, index: number, quality = 0.85) {
    console.log(`[OCR] processFile 开始，index=${index}, quality=${quality}`)
    if (!img) {
      console.error('[OCR] img 为空，退出')
      return
    }

    updateImage(index, { status: 'compressing' })
    console.log('[OCR] 状态更新为 compressing')

    try {
      console.log('[OCR] 开始计算 SHA256')
      const originalHash = await fileSHA256(img.file)
      console.log(`[OCR] SHA256: ${originalHash}`)

      console.log('[OCR] 开始图片压缩')
      const result = await toWebP(img.file, quality)
      console.log(`[OCR] 压缩完成, oversized=${result.oversized}, size=${result.file?.size}`)

      // 图片超过 1MB，压缩重试
      if (result.oversized) {
        const nextQuality = Math.max(0.1, quality - 0.1)
        if (nextQuality >= 0.3) {
          console.log(`[OCR] 图片过大，降低质量重试: ${nextQuality}`)
          return processFile(img, index, nextQuality)
        }
      }

      updateImage(index, {
        webpFile: result.file,
        hash: originalHash,
        status: 'ocr_loading',
      })
      console.log('[OCR] 状态更新为 ocr_loading，准备发送 OCR 请求')

      // === 第一步：OCR 识别 ===
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), OCR_TIMEOUT)

      try {
        console.log('[OCR] 发送 ocr 请求')
        const res = await uploadApi.ocr(result.file!, originalHash, { signal: controller.signal })
        console.log('[OCR] ocr 响应:', res.data)
        clearTimeout(timeoutId)

        const ocr = res.data.data as OcrResult
        updateImage(index, { ocrResult: ocr })

        // === 第二步：LLM 分析（如果启用） ===
        let llmSuggestions: LLMSuggestion[] = []
        if (llmEnabled) {
          updateImage(index, { status: 'llm_loading' })
          console.log('[LLM] 状态更新为 llm_loading，准备发送 LLM 分析请求')

          // 加载分类列表
          let catItems: { id: number; name: string; type: string }[] = []
          if (categories.length === 0) {
            try {
              const catRes = await categoryApi.list()
              setCategories(catRes.data.data)
              catItems = catRes.data.data.map(c => ({ id: c.id, name: c.name, type: c.type }))
            } catch (e) {
              console.error('[LLM] 加载分类失败:', e)
            }
          } else {
            catItems = categories.map(c => ({ id: c.id, name: c.name, type: c.type }))
          }

          try {
            const llmRes = await llmApi.analyze({
              image_path: ocr.image_path,
              raw_texts: ocr.raw_texts,
              categories: catItems,
            })
            console.log('[LLM] analyze 响应:', llmRes.data)

            const llmData = llmRes.data.data
            if (llmData.error) {
              updateImage(index, { status: 'success', llmError: llmData.error })
            } else {
              llmSuggestions = llmData.suggestions || []
              updateImage(index, { status: 'success', llmResult: llmSuggestions })
            }
          } catch (llmErr: unknown) {
            console.error('[LLM] analyze 失败:', llmErr)
            const axErr = llmErr as { response?: { data?: { message?: string } } }
            const msg = axErr.response?.data?.message || 'AI 分析失败'
            updateImage(index, { status: 'success', llmError: msg })
          }
        } else {
          updateImage(index, { status: 'success' })
        }

        // 如果是当前选中的图片，自动填充表单（直接使用刚获取的数据）
        if (index === selectedIndex) {
          if (llmSuggestions.length > 0) {
            applyLLM(llmSuggestions[0])
          } else if (ocr) {
            setEditAmount(String(ocr.amount || ''))
            setEditDate(ocr.date || dayjs().format('YYYY-MM-DD'))
            setEditMerchant(ocr.merchant_name || '')
            setEditNote(ocr.merchant_name ? `商户：${ocr.merchant_name}` : '')
            setEditType('expense')
            setEditCategoryId('')
          }
        }
      } catch (err: unknown) {
        clearTimeout(timeoutId)
        const isTimeout = (err as DOMException)?.name === 'AbortError'
        const axErr = err as { response?: { status?: number; data?: { message?: string } } }
        const msg = axErr.response?.data?.message || ''

        updateImage(index, {
          status: isTimeout ? 'timeout' : 'error',
          error: isTimeout ? 'OCR识别超时，请重试' : msg || 'OCR识别失败，请重试',
        })
      }
    } catch (err: unknown) {
      updateImage(index, {
        status: 'error',
        error: '图片处理失败',
      })
    }
  }

  function updateImage(index: number, updates: Partial<ImageItem>) {
    setImages(prev => {
      const next = [...prev]
      next[index] = { ...next[index], ...updates }
      return next
    })
  }

  function handleRecognize(index: number) {
    const img = images[index]
    if (img) processFile(img, index)
  }

  function handleRetry(index: number) {
    const img = images[index]
    if (!img) return
    updateImage(index, { status: 'ocr_loading', error: undefined, ocrResult: undefined, llmResult: undefined, llmError: undefined })
    processFile(img, index)
  }

  function handleDeleteImage(index: number) {
    setImages(prev => {
      const next = prev.filter((_, i) => i !== index)
      // 释放预览 URL
      URL.revokeObjectURL(prev[index].preview)
      return next
    })
    // 调整选中索引
    if (selectedIndex >= images.length - 1) {
      setSelectedIndex(Math.max(0, images.length - 2))
    } else if (selectedIndex > index) {
      setSelectedIndex(selectedIndex - 1)
    }
  }

  function handleAddMore() {
    fileInputRef.current?.click()
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    console.log('[上传] 步骤1: handleFileChange 开始')
    const files = Array.from(e.target.files ?? [])
    console.log(`[上传] 步骤2: 选择了 ${files.length} 个文件`, files)
    if (files.length === 0) {
      console.log('[上传] 没有选择文件，退出')
      return
    }

    const startIndex = images.length
    const newImages: ImageItem[] = files.map(file => ({
      file,
      preview: URL.createObjectURL(file),
      hash: '',
      status: 'pending' as const,
    }))
    console.log(`[上传] 步骤3: 创建了 ${newImages.length} 个预览`)

    setImages(prev => {
      const all = [...prev, ...newImages]
      if (prev.length === 0) {
        setSelectedIndex(0)
      }
      console.log(`[上传] 步骤4: setImages 完成，共 ${all.length} 张图片`)
      return all
    })

    console.log(`[上传] 步骤5: 准备调用 processFile，startIndex=${startIndex}`)
    newImages.forEach((img, i) => {
      console.log(`[上传] 步骤6: 即将调用 processFile 第 ${i} 张`, img.file.name)
      setTimeout(() => processFile(img, startIndex + i), 100)
    })

    e.target.value = ''
    console.log('[上传] 步骤7: handleFileChange 结束')
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    const files = Array.from(e.dataTransfer.files).filter(f => f.type.startsWith('image/'))
    if (files.length === 0) return

    const startIndex = images.length
    const newImages: ImageItem[] = files.map(file => ({
      file,
      preview: URL.createObjectURL(file),
      hash: '',
      status: 'pending' as const,
    }))

    setImages(prev => {
      const all = [...prev, ...newImages]
      if (prev.length === 0) {
        setSelectedIndex(0)
      }
      return all
    })

    // 自动开始识别新添加的图片
    newImages.forEach((img, i) => {
      setTimeout(() => processFile(img, startIndex + i), 100)
    })
  }

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  function handleSelectImage(index: number) {
    setSelectedIndex(index)
    const img = images[index]
    if (img?.ocrResult) {
      // 已有识别结果，填充表单
      if (img.llmResult && img.llmResult.length > 0) {
        applyLLM(img.llmResult[0])
      } else {
        setEditAmount(String(img.ocrResult.amount || ''))
        setEditDate(img.ocrResult.date || dayjs().format('YYYY-MM-DD'))
        setEditMerchant(img.ocrResult.merchant_name || '')
        setEditNote(img.ocrResult.merchant_name ? `商户：${img.ocrResult.merchant_name}` : '')
        setEditType('expense')
        setEditCategoryId('')
      }
    } else {
      // 清空表单
      setEditAmount('')
      setEditDate('')
      setEditMerchant('')
      setEditNote('')
      setEditType('expense')
      setEditCategoryId('')
    }
  }

  async function handleSave() {
    if (!selectedImage?.ocrResult) {
      alert('请先识别图片')
      return
    }
    if (!editAmount || Number(editAmount) <= 0) {
      alert('请输入有效金额')
      return
    }
    if (editCategoryId === '') {
      alert('请选择分类')
      return
    }
    setSaving(true)
    try {
      // TODO: 实现保存逻辑
      alert('保存成功！')
    } catch {
      alert('保存失败，请重试')
    } finally {
      setSaving(false)
    }
  }

  function handleClearAll() {
    images.forEach(img => URL.revokeObjectURL(img.preview))
    setImages([])
    setSelectedIndex(0)
    setEditAmount('')
    setEditDate('')
    setEditMerchant('')
    setEditNote('')
    setEditCategoryId('')
  }

  const filteredCategories = categories.filter(c => c.type === editType)

  const hasImages = images.length > 0

  return (
    <div className="p-4 md:p-6 max-w-2xl mx-auto">
      <div className="mb-5">
        <h1 className="text-xl md:text-2xl font-bold text-gray-800">拍照识别</h1>
        <p className="text-sm text-gray-400 mt-0.5">上传票据图片，自动识别金额和日期</p>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        multiple
        onChange={handleFileChange}
        className="hidden"
      />

      {/* 上传区域（无图片时显示） */}
      {!hasImages && (
        <div
          onClick={() => fileInputRef.current?.click()}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          className="border-2 border-dashed rounded-2xl p-10 text-center cursor-pointer transition-colors border-gray-200 hover:border-blue-400 hover:bg-gray-50"
        >
          <div className="flex flex-col items-center gap-3">
            <div className="w-16 h-16 rounded-full bg-blue-50 flex items-center justify-center">
              <svg className="w-8 h-8 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                  d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                  d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </div>
            <div>
              <p className="text-gray-700 font-medium">{isMobile ? '拍照或选择图片' : '点击上传或拖拽图片'}</p>
              <p className="text-xs text-gray-400 mt-1">支持 JPG、PNG、WEBP 等格式，可多选</p>
            </div>
          </div>
        </div>
      )}

      {/* 有图片时显示 */}
      {hasImages && (
        <div className="space-y-4">
          {/* 顶部操作栏 */}
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-gray-800">
              已上传 {images.length} 张图片
            </h2>
            <div className="flex gap-2">
              <button
                onClick={handleAddMore}
                className="px-3 py-1.5 text-sm bg-blue-50 text-blue-600 hover:bg-blue-100 rounded-lg transition-colors"
              >
                + 添加图片
              </button>
              <button
                onClick={handleClearAll}
                className="px-3 py-1.5 text-sm bg-gray-100 text-gray-600 hover:bg-gray-200 rounded-lg transition-colors"
              >
                清空
              </button>
            </div>
          </div>

          {/* 缩略图列表 */}
          <div className="flex gap-2 overflow-x-auto pb-2">
            {images.map((img, idx) => (
              <div
                key={idx}
                onClick={() => handleSelectImage(idx)}
                className={`relative flex-shrink-0 w-20 h-20 rounded-xl overflow-hidden cursor-pointer border-2 transition-colors ${
                  idx === selectedIndex
                    ? 'border-blue-500'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <img src={img.preview} alt="" className="w-full h-full object-cover" />
                {/* 状态标识 */}
                <div className="absolute inset-0 flex items-center justify-center">
                  {img.status === 'ocr_loading' && (
                    <div className="absolute inset-0 bg-black/40 flex items-center justify-center">
                      <div className="w-6 h-6 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    </div>
                  )}
                  {img.status === 'llm_loading' && (
                    <div className="absolute inset-0 bg-purple-500/40 flex items-center justify-center">
                      <div className="w-6 h-6 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    </div>
                  )}
                  {img.status === 'compressing' && (
                    <div className="absolute inset-0 bg-black/40 flex items-center justify-center">
                      <div className="w-6 h-6 border-2 border-blue-300 border-t-transparent rounded-full animate-spin" />
                    </div>
                  )}
                  {img.status === 'success' && (
                    <div className="absolute inset-0 bg-black/20 flex items-center justify-center">
                      <span className="text-white text-lg drop-shadow-md">✓</span>
                    </div>
                  )}
                  {(img.status === 'timeout' || img.status === 'error') && (
                    <>
                      {/* 半透明遮罩 */}
                      <div className="absolute inset-0 bg-red-500/40" />
                      {/* 中心重试按钮 */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleRetry(idx)
                        }}
                        title={img.error ?? '点击重试'}
                        className="w-14 h-14 rounded-full bg-white shadow-lg hover:bg-gray-50 flex items-center justify-center transition-all hover:scale-110 border-2 border-red-500"
                      >
                        <svg className="w-7 h-7 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5}
                            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                      </button>
                    </>
                  )}
                  {img.status === 'pending' && (
                    <div className="absolute inset-0 bg-black/20 flex items-center justify-center">
                      <span className="text-gray-300 text-xs">待识别</span>
                    </div>
                  )}
                </div>
                {/* 删除按钮 - 错误状态时红色更显眼 */}
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDeleteImage(idx)
                  }}
                  className={`absolute -top-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center shadow-md transition-colors ${
                    img.status === 'timeout' || img.status === 'error'
                      ? 'bg-red-500 hover:bg-red-600'
                      : 'bg-gray-400/80 hover:bg-gray-500'
                  }`}
                >
                  <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
          </div>

          {/* 当前图片识别区域 */}
          {selectedImage && (
            <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5">
              {/* 识别按钮/结果 */}
              {selectedImage.status === 'pending' && (
                <div className="text-center py-6">
                  <p className="text-gray-500 mb-3">点击下方按钮开始识别</p>
                  <button
                    onClick={() => handleRecognize(selectedIndex)}
                    className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-xl transition-colors"
                  >
                    开始识别
                  </button>
                </div>
              )}

              {selectedImage.status === 'compressing' && (
                <div className="flex flex-col items-center gap-3 py-6">
                  <div className="w-10 h-10 border-3 border-blue-500 border-t-transparent rounded-full animate-spin" />
                  <p className="text-gray-500">图片压缩中...</p>
                </div>
              )}

              {selectedImage.status === 'ocr_loading' && (
                <div className="flex flex-col items-center gap-3 py-6">
                  <div className="w-10 h-10 border-3 border-blue-500 border-t-transparent rounded-full animate-spin" />
                  <p className="text-gray-500">OCR 识别中...</p>
                </div>
              )}

              {selectedImage.status === 'llm_loading' && (
                <div className="flex flex-col items-center gap-3 py-6">
                  <div className="w-10 h-10 border-3 border-purple-500 border-t-transparent rounded-full animate-spin" />
                  <p className="text-purple-500">AI 分析中...</p>
                </div>
              )}

              {(selectedImage.status === 'timeout' || selectedImage.status === 'error') && (
                <div className="text-center py-6">
                  <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-red-50 flex items-center justify-center">
                    <svg className="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                        d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
                    </svg>
                  </div>
                  <p className="text-red-500 mb-3">{selectedImage.error}</p>
                  <button
                    onClick={() => handleRetry(selectedIndex)}
                    className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-xl transition-colors"
                  >
                    重试
                  </button>
                </div>
              )}

              {/* OCR 识别结果 */}
              {selectedImage.status === 'success' && selectedImage.ocrResult && (
                <>
                  {/* LLM 建议卡片 */}
                  {selectedImage.llmResult && selectedImage.llmResult.length > 0 && (
                    <div className="mb-4 bg-purple-50 border border-purple-100 rounded-xl p-3 space-y-2">
                      <p className="text-xs text-purple-500 font-medium">
                        AI 建议 · {selectedImage.llmResult.length} 笔
                      </p>
                      {selectedImage.llmResult.map((llm, i) => (
                        <div key={i} className="bg-white rounded-lg p-2">
                          <div className="flex items-center justify-between gap-2 flex-wrap">
                            <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-sm text-gray-600">
                              {(llm.amount ?? 0) > 0 && (
                                <span>💰 <span className="font-semibold text-purple-700">¥{llm.amount}</span></span>
                              )}
                              {llm.date && <span>📅 {llm.date}</span>}
                              {llm.merchant_name && <span>🏪 {llm.merchant_name}</span>}
                              {llm.category_hint && <span>🏷 {llm.category_hint}</span>}
                            </div>
                            <button
                              type="button"
                              onClick={() => applyLLM(llm)}
                              className="text-xs text-purple-600 hover:text-purple-800 underline flex-shrink-0"
                            >
                              填入
                            </button>
                          </div>
                          {llm.note && <p className="text-xs text-gray-400 mt-0.5">{llm.note}</p>}
                        </div>
                      ))}
                    </div>
                  )}
                  {!selectedImage.llmResult && selectedImage.llmError && (
                    <div className="mb-4 bg-amber-50 border border-amber-200 rounded-xl p-3">
                      <div className="flex items-start gap-2">
                        <svg className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
                        </svg>
                        <div>
                          <p className="text-sm font-medium text-amber-800">AI 分析不可用</p>
                          <p className="text-xs text-amber-700 mt-1">{selectedImage.llmError}</p>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* 识别原始文字 */}
                  {selectedImage.ocrResult.raw_texts && selectedImage.ocrResult.raw_texts.length > 0 && (
                    <div className="mb-5 p-3 bg-gray-50 rounded-xl">
                      <p className="text-xs text-gray-400 mb-1">识别原文</p>
                      <p className="text-sm text-gray-600 leading-relaxed">
                        {selectedImage.ocrResult.raw_texts.join(' ')}
                      </p>
                    </div>
                  )}

                  <h2 className="text-base font-semibold text-gray-800 mb-4">确认并编辑信息</h2>

                  <div className="space-y-4">
                    {/* 类型切换 */}
                    <div className="flex rounded-xl overflow-hidden border border-gray-200">
                      <button
                        type="button"
                        onClick={() => { setEditType('expense'); setEditCategoryId('') }}
                        className={`flex-1 py-2.5 text-sm font-medium transition-colors ${
                          editType === 'expense' ? 'bg-red-500 text-white' : 'bg-white text-gray-500 hover:bg-gray-50'
                        }`}
                      >
                        支出
                      </button>
                      <button
                        type="button"
                        onClick={() => { setEditType('income'); setEditCategoryId('') }}
                        className={`flex-1 py-2.5 text-sm font-medium transition-colors ${
                          editType === 'income' ? 'bg-green-500 text-white' : 'bg-white text-gray-500 hover:bg-gray-50'
                        }`}
                      >
                        收入
                      </button>
                    </div>

                    {/* 金额 */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">金额</label>
                      <div className="relative">
                        <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">¥</span>
                        <input
                          type="number"
                          step="0.01"
                          min="0.01"
                          value={editAmount}
                          onChange={e => setEditAmount(e.target.value)}
                          className="w-full pl-8 pr-4 py-2.5 border border-gray-300 rounded-xl text-lg font-semibold focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                    </div>

                    {/* 日期 */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">日期</label>
                      <input
                        type="date"
                        value={editDate}
                        onChange={e => setEditDate(e.target.value)}
                        className="w-full px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>

                    {/* 商户 */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">商户</label>
                      <input
                        type="text"
                        value={editMerchant}
                        onChange={e => setEditMerchant(e.target.value)}
                        placeholder="商户名称"
                        className="w-full px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>

                    {/* 备注 */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">备注</label>
                      <input
                        type="text"
                        value={editNote}
                        onChange={e => setEditNote(e.target.value)}
                        placeholder="备注信息"
                        className="w-full px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>

                    {/* 分类 */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">分类</label>
                      <div className="grid grid-cols-4 gap-2">
                        {filteredCategories.map(cat => (
                          <button
                            key={cat.id}
                            type="button"
                            onClick={() => setEditCategoryId(cat.id)}
                            className={`flex flex-col items-center gap-1 p-2 rounded-xl border transition-colors ${
                              editCategoryId === cat.id
                                ? 'border-blue-500 bg-blue-50 text-blue-600'
                                : 'border-gray-200 hover:bg-gray-50 text-gray-600'
                            }`}
                          >
                            <span className="text-xl">{cat.icon || '📌'}</span>
                            <span className="text-xs truncate w-full text-center">{cat.name}</span>
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>

                  <div className="flex gap-3 mt-6">
                    <button
                      onClick={() => handleRetry(selectedIndex)}
                      className="flex-1 py-3 bg-gray-100 hover:bg-gray-200 text-gray-600 font-medium rounded-xl transition-colors"
                    >
                      重新识别
                    </button>
                    <button
                      onClick={handleSave}
                      disabled={saving}
                      className="flex-1 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white font-semibold rounded-xl transition-colors"
                    >
                      {saving ? '保存中...' : '保存记账'}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
