import { useState, useRef, useCallback } from 'react'
import dayjs from 'dayjs'
import { uploadApi, categoryApi, transactionApi, type OcrAnalyzeResult, type LLMSuggestion, type Category, type TransactionCreatePayload } from '../../services/api'
import { useResponsive } from '../../hooks/useResponsive'
import { toWebP } from '../../utils/imageUtils'

export default function UploadOCR() {
  const { isMobile } = useResponsive()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [dragOver, setDragOver] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [ocrResult, setOcrResult] = useState<OcrAnalyzeResult | null>(null)
  const [categories, setCategories] = useState<Category[]>([])

  // 编辑字段
  const [editAmount, setEditAmount] = useState('')
  const [editDate, setEditDate] = useState('')
  const [editMerchant, setEditMerchant] = useState('')
  const [editType, setEditType] = useState<'income' | 'expense'>('expense')
  const [editCategoryId, setEditCategoryId] = useState<number | ''>('')
  const [editNote, setEditNote] = useState('')
  const [saving, setSaving] = useState(false)

  async function processFile(file: File) {
    if (!file.type.startsWith('image/')) {
      alert('请上传图片文件')
      return
    }
    setUploading(true)
    setOcrResult(null)
    try {
      const webpFile = await toWebP(file)
      const [ocrRes, catRes] = await Promise.all([
        uploadApi.ocrAnalyze(webpFile),
        categoryApi.list(),
      ])
      const ocr = ocrRes.data.data
      setOcrResult(ocr)
      setCategories(catRes.data.data)

      // 优先用 LLM 结果填入，否则用 OCR
      if (ocr.llm && ocr.llm.length > 0) {
        applyLLM(ocr.llm[0], catRes.data.data)
      } else {
        setEditAmount(String(ocr.amount || ''))
        setEditDate(ocr.date || dayjs().format('YYYY-MM-DD'))
        setEditMerchant(ocr.merchant_name || '')
        setEditNote(ocr.merchant_name ? `商户：${ocr.merchant_name}` : '')
        setEditType('expense')
        setEditCategoryId('')
      }
    } catch (err: unknown) {
      const axErr = err as { response?: { status?: number; data?: { message?: string } } }
      const msg = axErr.response?.data?.message
      if (axErr.response?.status === 403) {
        alert(msg || '权限不足，无法上传图片')
      } else {
        alert(msg || 'OCR 识别失败，请重试')
      }
    } finally {
      setUploading(false)
    }
  }

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

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) processFile(file)
    e.target.value = ''
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files?.[0]
    if (file) processFile(file)
  }

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(true)
  }, [])

  const handleDragLeave = useCallback(() => {
    setDragOver(false)
  }, [])

  async function handleSave() {
    if (!editAmount || Number(editAmount) <= 0) {
      alert('请输入有效金额')
      return
    }
    if (editCategoryId === '') {
      alert('请选择分类')
      return
    }
    setSaving(true)
    const payload: TransactionCreatePayload = {
      type: editType,
      amount: Number(editAmount),
      category_id: editCategoryId as number,
      transaction_date: editDate,
      note: editNote,
    }
    try {
      await transactionApi.batchCreate([payload])
      alert('保存成功！')
      setOcrResult(null)
      setEditAmount('')
      setEditDate('')
      setEditMerchant('')
      setEditNote('')
      setEditCategoryId('')
    } catch {
      alert('保存失败，请重试')
    } finally {
      setSaving(false)
    }
  }

  const filteredCategories = categories.filter(c => c.type === editType)

  return (
    <div className="p-4 md:p-6 max-w-2xl mx-auto">
      <div className="mb-5">
        <h1 className="text-xl md:text-2xl font-bold text-gray-800">拍照识别</h1>
        <p className="text-sm text-gray-400 mt-0.5">上传票据图片，自动识别金额和日期</p>
      </div>

      {/* 上传区域 */}
      {!ocrResult && (
        <div
          onClick={() => fileInputRef.current?.click()}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          className={`
            border-2 border-dashed rounded-2xl p-10 text-center cursor-pointer transition-colors
            ${dragOver
              ? 'border-blue-500 bg-blue-50'
              : 'border-gray-200 hover:border-blue-400 hover:bg-gray-50'
            }
          `}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            onChange={handleFileChange}
            className="hidden"
          />

          {uploading ? (
            <div className="flex flex-col items-center gap-3">
              <div className="w-12 h-12 border-4 border-blue-500 border-t-transparent rounded-full animate-spin" />
              <p className="text-gray-500">识别中，请稍候...</p>
            </div>
          ) : (
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
                <p className="text-gray-700 font-medium">
                  {isMobile ? '拍照或选择图片' : '点击上传或拖拽图片'}
                </p>
                <p className="text-xs text-gray-400 mt-1">支持 JPG、PNG、WEBP 等格式</p>
              </div>
            </div>
          )}
        </div>
      )}

      {/* OCR 识别结果 + 编辑表单 */}
      {ocrResult && (
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5">
          {/* LLM 建议卡片 */}
          {ocrResult.llm && ocrResult.llm.length > 0 && (
            <div className="mb-4 bg-purple-50 border border-purple-100 rounded-xl p-3 space-y-2">
              <p className="text-xs text-purple-500 font-medium">
                AI 建议 · {ocrResult.llm.length} 笔
                {ocrResult.llm.length > 1 && <span className="ml-1 text-purple-400">（点击填入对应数据）</span>}
              </p>
              {ocrResult.llm.map((llm, i) => (
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
          {!ocrResult.llm && ocrResult.llm_error && (
            <p className="text-xs text-gray-400 mb-3">AI 分析不可用：{ocrResult.llm_error}</p>
          )}

          {/* 识别原始文字 */}
          {ocrResult.raw_texts && ocrResult.raw_texts.length > 0 && (
            <div className="mb-5 p-3 bg-gray-50 rounded-xl">
              <p className="text-xs text-gray-400 mb-1">识别原文</p>
              <p className="text-sm text-gray-600 leading-relaxed">
                {ocrResult.raw_texts.join(' ')}
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
              onClick={() => setOcrResult(null)}
              className="flex-1 py-3 bg-gray-100 hover:bg-gray-200 text-gray-600 font-medium rounded-xl transition-colors"
            >
              重新上传
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex-1 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white font-semibold rounded-xl transition-colors"
            >
              {saving ? '保存中...' : '保存记账'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
