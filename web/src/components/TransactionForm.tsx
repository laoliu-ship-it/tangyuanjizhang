import { useState, useEffect, useRef, useCallback } from 'react'
import dayjs from 'dayjs'
import {
  categoryApi, merchantApi, transactionApi, uploadApi, llmApi,
  type Category, type Merchant, type Transaction, type TransactionCreatePayload, type OcrAnalyzeResult, type OcrResult, type LLMSuggestion,
} from '../services/api'
import { useResponsive } from '../hooks/useResponsive'
import { toWebP, fileSHA256, getImageEngineStatus } from '../utils/imageUtils'

// ── 类型定义 ────────────────────────────────────────────────
interface SlotForm {
  type: 'income' | 'expense'
  amount: string
  categoryId: number | ''
  merchantId: number
  merchantName: string
  date: string
  note: string
}

type SlotStatus = 'pending' | 'recognizing' | 'done' | 'error'

interface ImageSlot {
  id: string
  file: File
  previewUrl: string
  status: SlotStatus
  ocrResult: OcrAnalyzeResult | null
  imagePath: string
  saved: boolean
  form: SlotForm
  originalHash?: string
}

interface SlotDraft {
  id: string
  type: 'income' | 'expense'
  amount: number
  categoryId: number
  categoryName: string
  merchantId: number
  merchantName: string
  date: string
  note: string
  imagePath: string
  llmIndex?: number  // 对应 ocrResult.llm 数组的下标，用于标记「已填入」
}

interface Props {
  open: boolean
  onClose: () => void
  onSuccess?: () => void
  initialData?: Transaction
  mode?: 'create' | 'edit' | 'detail'  // detail 模式只读展示
}

const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500'

/**
 * 从 OCR 日期字符串中提取 HH:mm 时间部分。
 * 兼容格式：
 *   "2024-05-20T14:30"      → "14:30"
 *   "2024-05-20 14:30"      → "14:30"
 *   "2024-05-20T14:30:00"   → "14:30"
 *   "2024-05-20 14:30:33"   → "14:30"
 *   "2024年5月20日 14:30"    → "14:30"
 *   "2024/5/20 14:30"       → "14:30"
 *   "2024-05-20"            → fallback（返回当前时间）
 */
function extractTimeFromOcrDate(ocrDate: string | undefined, fallback?: string): string {
  if (!ocrDate) return fallback ?? dayjs().format('HH:mm')
  // 匹配 HH:mm（支持前面有 T、空格、中文等分隔符）
  const m = ocrDate.match(/[\sT](\d{1,2}):(\d{2})/)
  if (m) {
    return `${m[1].padStart(2, '0')}:${m[2]}`
  }
  return fallback ?? dayjs().format('HH:mm')
}

function defaultForm(): SlotForm {
  return { type: 'expense', amount: '', categoryId: '', merchantId: 0, merchantName: '', date: dayjs().format('YYYY-MM-DDTHH:mm'), note: '' }
}

let _slotIdCounter = 0
function newSlotId() { return `slot_${Date.now()}_${_slotIdCounter++}` }

// ── 组件 ────────────────────────────────────────────────────
export default function TransactionForm({ open, onClose, onSuccess, initialData, mode = 'create' }: Props) {
  const { isMobile } = useResponsive()
  const isDetailMode = mode === 'detail'
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [slots, setSlots] = useState<ImageSlot[]>([])
  const [activeIdx, setActiveIdx] = useState(0)
  const [drafts, setDrafts] = useState<Map<string, SlotDraft[]>>(new Map())
  const [categories, setCategories] = useState<Category[]>([])
  const [merchants, setMerchants] = useState<Merchant[]>([])
  const [catLoading, setCatLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [lightboxOpen, setLightboxOpen] = useState(false)
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; text: string; tokenKey: string } | null>(null)
  const [selectedTokens, setSelectedTokens] = useState<Map<string, string>>(new Map()) // key为token原文, value为数值字符串
  const [usedTokens, setUsedTokens] = useState<Set<string>>(new Set()) // 已被右键填入的 OCR token
  const [showReview, setShowReview] = useState(false)
  const [editingDraftIndex, setEditingDraftIndex] = useState<number | null>(null)
  const [editingDraftForm, setEditingDraftForm] = useState<SlotDraft | null>(null)
  const [toast, setToast] = useState<{ text: string; error?: boolean } | null>(null)
  const [noSlotForm, setNoSlotForm] = useState<SlotForm>(defaultForm())
  const [noSlotDrafts, setNoSlotDrafts] = useState<SlotDraft[]>([])
  const llmEnabledRef = useRef(false)
  const submittingRef = useRef(false) // 防抖：防止重复提交

  // 当 activeIdx 变化时清空选中和已用标记
  const prevActiveIdxRef = useRef(activeIdx)
  useEffect(() => {
    if (prevActiveIdxRef.current !== activeIdx) {
      setSelectedTokens(new Map())
      setUsedTokens(new Set())
    }
    prevActiveIdxRef.current = activeIdx
  }, [activeIdx])

  // ── 多选计算器 ──────────────────────────────────────────
  function isNumericToken(t: string): boolean {
    // 匹配带符号的数字，如 -12, +5, 3.14, -8.00
    return /^[+-]?\d+(\.\d+)?$/.test(t)
  }

  function toggleToken(token: string) {
    setSelectedTokens(prev => {
      const next = new Map(prev)
      if (next.has(token)) {
        next.delete(token)
      } else if (isNumericToken(token)) {
        next.set(token, token)
      }
      return next
    })
  }

  const calcResult = (() => {
    const nums = Array.from(selectedTokens.values()).map(Number)
    if (nums.length < 2) return null
    const sum = nums.reduce((a, b) => a + b, 0)
    const formula = nums.map(n => (n >= 0 ? `+${n}` : `${n}`)).join('')
    return { formula: formula.replace(/^\+/, ''), result: sum }
  })()

  // OCR 队列：顺序处理，不阻塞用户操作
  const ocrQueueRef = useRef<string[]>([])
  const processingRef = useRef(false)
  const filesMapRef = useRef<Map<string, File>>(new Map())  // 同步存文件，避免等待 state 更新

  // 活跃 slot
  const activeSlot = slots[activeIdx] ?? null
  const currentDrafts = activeSlot ? (drafts.get(activeSlot.id) ?? []) : noSlotDrafts
  const savedCount = slots.filter(s => s.saved).length
  // 已填入草稿的 LLM 条目下标集合（用于 OCR 面板标记删除线/已填入）
  const filledLlmIndices = new Set(currentDrafts.map(d => d.llmIndex).filter((i): i is number => i !== undefined))

  // ── 初始化 ──────────────────────────────────────────────
  useEffect(() => {
    if (!open) return
    setActiveIdx(0)
    setDrafts(new Map())
    setUsedTokens(new Set())
    setShowReview(false)
    setNoSlotForm(defaultForm())
    setNoSlotDrafts([])
    setLightboxOpen(false)
    setCtxMenu(null)
    ocrQueueRef.current = []
    processingRef.current = false
    submittingRef.current = false // 重置提交防抖标志
    filesMapRef.current.clear()

    // 加载 LLM 配置，决定是否调用 AI 分析
    llmApi.getConfig().then(r => {
      llmEnabledRef.current = r.data.data?.enabled ?? false
    }).catch(() => {
      llmEnabledRef.current = false
    })

    if (initialData) {
      const raw = initialData.transaction_date ?? ''
      const date = raw.length >= 16 ? raw.slice(0, 16).replace(' ', 'T') : raw.slice(0, 10) + 'T00:00'
      const firstImage = initialData.images?.[0]
      const imagePath = firstImage?.image_path ?? ''
      // 从已有数据恢复 OCR 结果
      let restoredOCR: OcrAnalyzeResult | null = null
      if (firstImage?.ocr_raw_texts) {
        try {
          const rawTexts = JSON.parse(firstImage.ocr_raw_texts) as string[]
          restoredOCR = {
            image_path: imagePath,
            ai_mode: false,
            amount: firstImage.ocr_amount ?? 0,
            date: firstImage.ocr_date ?? '',
            merchant_id: 0,
            merchant_name: firstImage.ocr_merchant ?? '',
            raw_texts: rawTexts,
          }
        } catch { /* ignore parse error */ }
      }
      const editSlot: ImageSlot = {
        id: newSlotId(),
        file: new File([], ''),
        previewUrl: imagePath,
        status: 'done',
        ocrResult: restoredOCR,
        imagePath: imagePath,
        saved: false,
        form: {
          type: initialData.type,
          amount: String(initialData.amount),
          categoryId: initialData.category_id,
          merchantId: initialData.merchant_id ?? 0,
          merchantName: initialData.merchant_name ?? '',
          date,
          note: initialData.note ?? '',
        },
      }
      setSlots([editSlot])
    } else {
      setSlots([])
    }
    loadCategories()
    loadMerchants()
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  // 清理 blob URL：按 id 判断是否真正移除，避免 map() 产生新引用时误 revoke
  const prevSlotsRef = useRef<ImageSlot[]>([])
  useEffect(() => {
    const prev = prevSlotsRef.current
    const currentIds = new Set(slots.map(s => s.id))
    prev.forEach(s => {
      if (s.previewUrl.startsWith('blob:') && !currentIds.has(s.id)) {
        URL.revokeObjectURL(s.previewUrl)
      }
    })
    prevSlotsRef.current = slots
  }, [slots])

  async function loadCategories() {
    setCatLoading(true)
    try { const r = await categoryApi.list(); setCategories(r.data.data ?? []) }
    catch { /* ignore */ } finally { setCatLoading(false) }
  }

  async function loadMerchants() {
    try { const r = await merchantApi.list(); setMerchants(r.data.data ?? []) }
    catch { /* ignore */ }
  }

  // ── OCR 队列处理器 ────────────────────────────────────────
  const processQueue = useCallback(async () => {
    if (processingRef.current) return
    processingRef.current = true
    while (ocrQueueRef.current.length > 0) {
      const id = ocrQueueRef.current[0]
      // 标记为识别中
      setSlots(prev => prev.map(s => s.id === id ? { ...s, status: 'recognizing' as SlotStatus } : s))
      const file = filesMapRef.current.get(id) ?? null
      if (!file || file.size === 0) {
        ocrQueueRef.current = ocrQueueRef.current.slice(1)
        continue
      }
      try {
        let ocr: OcrAnalyzeResult | null = null
        const slot = slots.find(s => s.id === id)
        const originalHash = slot?.originalHash
        if (llmEnabledRef.current) {
          const res = await uploadApi.ocrAnalyze(file!, originalHash)
          ocr = res.data.data
        } else {
          const res = await uploadApi.ocr(file!, originalHash)
          const raw = res.data.data as OcrResult
          // 将 OcrResult 包装成 OcrAnalyzeResult 兼容后续处理
          ocr = {
            image_path: raw.image_path,
            ai_mode: raw.ai_mode,
            amount: raw.amount,
            date: raw.date,
            merchant_id: raw.merchant_id,
            merchant_name: raw.merchant_name ?? '',
            raw_texts: raw.raw_texts,
          }
        }
        setSlots(prev => prev.map(s => {
          if (s.id !== id) return s
          const form = { ...s.form }
          if (ocr?.ai_mode) {
            if ((ocr.amount ?? 0) > 0) form.amount = String(ocr.amount)
            if (ocr.date) {
              const datePart = ocr.date.match(/(\d{4}-\d{2}-\d{2})/)?.[1] ?? ocr.date.slice(0, 10)
              form.date = `${datePart}T${extractTimeFromOcrDate(ocr.date)}`
            }
            if (ocr.merchant_id) {
              form.merchantId = ocr.merchant_id
              form.merchantName = ocr.merchant_name ?? ''
            }
          }
          return { ...s, status: 'done' as SlotStatus, ocrResult: ocr ?? null, imagePath: ocr?.image_path ?? '', form }
        }))
        // 多条 LLM 建议 → 自动批量生成草稿
        if (ocr?.llm && ocr.llm.length > 1) {
          // 从 OCR 日期中提取时间（如有）
          const ocrTime = extractTimeFromOcrDate(ocr.date)
          setDrafts(prev => {
            const next = new Map(prev)
            const autoDrafts: SlotDraft[] = ocr.llm!.map((llm, idx) => {
              const catId = llm.category_id ?? 0
              const catName = catId
                ? (categories.find(c => c.id === catId)?.name ?? llm.category_hint ?? '')
                : (llm.category_hint ?? '')
              
              // 检查 LLM 返回的 date 是否已包含时间
              let formattedDate: string
              if (llm.date) {
                const hasTime = /\d{2}:\d{2}$/.test(llm.date.trim())
                if (hasTime) {
                  // "YYYY-MM-DD HH:mm" → "YYYY-MM-DDTHH:mm"
                  const parts = llm.date.trim().split(' ')
                  formattedDate = `${parts[0]}T${parts[1] || '00:00'}`
                } else {
                  formattedDate = `${llm.date}T${ocrTime}`
                }
              } else {
                formattedDate = dayjs().format('YYYY-MM-DDTHH:mm')
              }
              
              return {
                id: `draft_${Date.now()}_${idx}`,
                type: llm.type === 'income' ? 'income' as const : 'expense' as const,
                amount: llm.amount ?? 0,
                categoryId: catId,
                categoryName: catName,
                merchantId: 0,
                merchantName: llm.merchant_name ?? '',
                date: formattedDate,
                note: llm.note ?? '',
                imagePath: ocr.image_path ?? '',
                llmIndex: idx,
              }
            })
            next.set(id, autoDrafts)
            return next
          })
          setToast({ text: `AI 识别出 ${ocr.llm.length} 笔，已生成草稿，请确认后提交` })
          setTimeout(() => setToast(null), 4000)
        }
      } catch (err: unknown) {
        const axErr = err as { response?: { status?: number; data?: { message?: string } } }
        if (axErr.response?.status === 403) {
          const msg = axErr.response.data?.message || '权限不足，无法上传图片'
          setToast({ text: msg, error: true })
          setTimeout(() => setToast(null), 3500)
        }
        setSlots(prev => prev.map(s => s.id === id ? { ...s, status: 'error' as SlotStatus } : s))
      }
      filesMapRef.current.delete(id)
      ocrQueueRef.current = ocrQueueRef.current.slice(1)
    }
    processingRef.current = false
  }, [])

  // ── 文件选择 ──────────────────────────────────────────────

  /** 逐张转换，如果超限则弹窗确认后降低质量重试 */
  async function convertFilesWithConfirm(imageFiles: File[]): Promise<File[]> {
    const engineStatus = getImageEngineStatus()
    if (engineStatus === 'unavailable') {
      throw new Error('图片转换不可用：WebAssembly 加载失败')
    }

    const results: File[] = []
    for (const file of imageFiles) {
      let quality = 0.85
      let result = await toWebP(file, quality)
      while (result.oversized) {
        const sizeMB = ((result.size ?? 0) / 1024 / 1024).toFixed(1)
        const ok = confirm(`"${file.name}" 图片太大（${sizeMB}MB），是否自动压缩后重试？\n（质量将降低约10%）`)
        if (!ok) throw new Error(`用户取消压缩：${file.name}`)
        quality = Math.max(0.1, quality - 0.1)
        result = await toWebP(file, quality)
      }
      results.push(result.file!)
    }
    return results
  }

  async function handleFilesChange(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (files.length === 0) return
    const imageFiles = files.filter(f => f.type.startsWith('image/'))
    if (imageFiles.length === 0) { alert('请选择图片文件'); return }

    const hashes = await Promise.all(imageFiles.map(f => fileSHA256(f)))
    let webpFiles: File[]
    try {
      webpFiles = await convertFilesWithConfirm(imageFiles)
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : '转换失败')
      return
    }
    const newSlots: ImageSlot[] = webpFiles.map((f, i) => ({
      id: newSlotId(),
      file: f,
      previewUrl: URL.createObjectURL(f),
      status: 'pending' as SlotStatus,
      ocrResult: null,
      imagePath: '',
      saved: false,
      form: defaultForm(),
      originalHash: hashes[i],
    }))

    // 同步写入 files map，processQueue 能立即读到文件
    newSlots.forEach(s => filesMapRef.current.set(s.id, s.file))
    setSlots(prev => {
      const updated = [...prev, ...newSlots]
      // 自动切到第一张新图
      setActiveIdx(updated.length - newSlots.length)
      return updated
    })
    // 加入 OCR 队列
    newSlots.forEach(s => ocrQueueRef.current.push(s.id))
    processQueue()
  }

  async function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    const files = Array.from(e.dataTransfer.files)
    const imageFiles = files.filter(f => f.type.startsWith('image/'))
    if (imageFiles.length === 0) return
    const hashes = await Promise.all(imageFiles.map(f => fileSHA256(f)))
    let webpFiles: File[]
    try {
      webpFiles = await convertFilesWithConfirm(imageFiles)
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : '转换失败')
      return
    }
    const newSlots: ImageSlot[] = webpFiles.map((f, i) => ({
      id: newSlotId(), file: f, previewUrl: URL.createObjectURL(f),
      status: 'pending' as SlotStatus, ocrResult: null, imagePath: '', saved: false, form: defaultForm(),
      originalHash: hashes[i],
    }))
    newSlots.forEach(s => filesMapRef.current.set(s.id, s.file))
    setSlots(prev => {
      setActiveIdx(prev.length)
      return [...prev, ...newSlots]
    })
    newSlots.forEach(s => ocrQueueRef.current.push(s.id))
    processQueue()
  }

  // ── 表单更新（绑定到当前 slot）────────────────────────────
  function patchForm(patch: Partial<SlotForm>) {
    if (activeSlot) {
      setSlots(prev => prev.map((s, i) => i === activeIdx ? { ...s, form: { ...s.form, ...patch } } : s))
    } else {
      setNoSlotForm(prev => ({ ...prev, ...patch }))
    }
  }

  function applyLLMSuggestion(llm: LLMSuggestion, llmIndex: number) {
    const patch: Partial<SlotForm> = {}
    if (llm.type === 'income' || llm.type === 'expense') patch.type = llm.type
    if ((llm.amount ?? 0) > 0) patch.amount = String(llm.amount)
    if (llm.date) {
      // 检查 LLM 返回的 date 是否已包含时间（格式：YYYY-MM-DD HH:mm）
      const hasTime = /\d{2}:\d{2}$/.test(llm.date.trim())
      if (hasTime) {
        // 将 "YYYY-MM-DD HH:mm" 转换为 "YYYY-MM-DDTHH:mm"
        const parts = llm.date.trim().split(' ')
        patch.date = `${parts[0]}T${parts[1] || '00:00'}`
      } else {
        // 从 OCR 日期中提取时间（如有），否则用当前时间
        const ocrTime = extractTimeFromOcrDate(ocrResult?.date)
        patch.date = `${llm.date}T${ocrTime}`
      }
    }
    if (llm.merchant_name) {
      const m = merchants.find(x => x.name === llm.merchant_name)
      patch.merchantName = llm.merchant_name
      if (m) patch.merchantId = m.id
    }
    if (llm.note) patch.note = llm.note
    // 优先用 LLM 返回的 category_id，fallback 到 category_hint 模糊匹配
    if (llm.category_id) {
      patch.categoryId = llm.category_id
    } else if (llm.category_hint) {
      const type = patch.type ?? (activeSlot?.form.type ?? 'expense')
      const matched = categories.find(c => c.type === type && c.name.includes(llm.category_hint))
      if (matched) patch.categoryId = matched.id
    }
    const resolvedCatName = patch.categoryId
      ? (categories.find(c => c.id === patch.categoryId)?.name ?? llm.category_hint ?? '')
      : (llm.category_hint ?? '')
    patchForm(patch)

    // 标记 source_lines 对应的所有 OCR 词条为已使用（划线）
    if (llm.source_lines && llm.source_lines.length > 0) {
      setUsedTokens(prev => {
        const next = new Set(prev)
        for (const lineIdx of llm.source_lines) {
          // 每一行的所有 token 都标记
          const ocrResult = activeSlot?.ocrResult
          if (ocrResult?.raw_texts && lineIdx >= 0 && lineIdx < ocrResult.raw_texts.length) {
            const line = ocrResult.raw_texts[lineIdx]
            line.split(/\s+/).filter(Boolean).forEach((_, tokenIdx) => {
              next.add(`${lineIdx}_${tokenIdx}`)
            })
          }
        }
        return next
      })
    }

    // 直接暂存为草稿（如尚未填入过该条目）
    if (!filledLlmIndices.has(llmIndex)) {
      const slotId = activeSlot?.id
      const draft: SlotDraft = {
        id: `draft_${Date.now()}_${llmIndex}`,
        type: (patch.type ?? activeSlot?.form.type ?? 'expense') as 'income' | 'expense',
        amount: Number(patch.amount ?? 0),
        categoryId: (patch.categoryId as number) ?? 0,
        categoryName: resolvedCatName,
        merchantId: (patch.merchantId as number) ?? 0,
        merchantName: patch.merchantName ?? '',
        date: patch.date ?? dayjs().format('YYYY-MM-DDTHH:mm'),
        note: patch.note ?? '',
        imagePath: activeSlot?.imagePath ?? '',
        llmIndex,
      }
      if (slotId) {
        setDrafts(prev => {
          const next = new Map(prev)
          next.set(slotId, [...(next.get(slotId) ?? []), draft])
          return next
        })
      } else {
        setNoSlotDrafts(prev => [...prev, draft])
      }
    }
  }

  // ── 编辑模式：单笔保存 ─────────────────────────────────────
  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!activeSlot) return
    if (activeSlot.status === 'recognizing') { alert('正在识别，请稍候'); return }
    const { form, imagePath } = activeSlot
    if (!form.amount || Number(form.amount) <= 0) { alert('请输入有效金额'); return }
    if (form.categoryId === '') { alert('请选择分类'); return }
    if (form.type === 'expense' && !imagePath) { alert('支出记录需要上传截图'); return }

    setSaving(true)
    const ocr = activeSlot?.ocrResult
    const payload: TransactionCreatePayload = {
      type: form.type,
      amount: Number(form.amount),
      category_id: form.categoryId as number,
      merchant_id: form.merchantId || undefined,
      merchant_name: form.merchantName || undefined,
      transaction_date: form.date.replace('T', ' ') + ':00',
      note: form.note,
      image_path: imagePath || undefined,
      ocr_amount: ocr?.amount ?? 0,
      ocr_date: ocr?.date ?? '',
      ocr_merchant: ocr?.merchant_name ?? '',
      ocr_raw_texts: ocr ? JSON.stringify(ocr.raw_texts ?? []) : '',
    }
    try {
      await transactionApi.update(initialData!.id, payload)
      onSuccess?.(); onClose()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      setToast({ text: msg || '保存失败，请稍后重试', error: true })
    }
    finally { setSaving(false) }
  }

  // ── 新增模式：记一笔（本地暂存）─────────────────────────────
  function handleAddDraft() {
    if (!activeSlot && form.type === 'expense') { alert('支出记录需要上传截图'); return }
    if (!form.amount || Number(form.amount) <= 0) { alert('请输入有效金额'); return }
    if (form.categoryId === '') { alert('请选择分类'); return }
    if (form.type === 'expense' && !activeSlot!.imagePath && !activeSlot!.previewUrl) { alert('支出记录需要上传截图'); return }

    const cat = categories.find(c => c.id === form.categoryId)
    const draft: SlotDraft = {
      id: `draft_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
      type: form.type,
      amount: Number(form.amount),
      categoryId: form.categoryId as number,
      categoryName: cat?.name ?? '',
      merchantId: form.merchantId,
      merchantName: form.merchantName,
      date: form.date,
      note: form.note,
      imagePath: activeSlot?.imagePath || activeSlot?.ocrResult?.image_path || '',
    }
    if (activeSlot) {
      setDrafts(prev => {
        const next = new Map(prev)
        const list = next.get(activeSlot.id) ?? []
        next.set(activeSlot.id, [...list, draft])
        return next
      })
    } else {
      setNoSlotDrafts(prev => [...prev, draft])
    }
    // 清空金额、备注、商户，保留类型、分类、日期
    patchForm({ amount: '', note: '', merchantId: 0, merchantName: '' })
  }

  // ── 新增模式：批量保存当前 slot 的所有草稿（带防抖）──────────
  async function handleBatchSave() {
    // 防抖：如果正在提交中，直接返回
    if (submittingRef.current) return
    if (currentDrafts.length === 0) return
    
    submittingRef.current = true
    setSaving(true)
    const ocr = activeSlot?.ocrResult ?? null
    const draftsToSave = currentDrafts
    try {
      for (const draft of draftsToSave) {
        const payload: TransactionCreatePayload = {
          type: draft.type,
          amount: draft.amount,
          category_id: draft.categoryId,
          merchant_id: draft.merchantId || undefined,
          merchant_name: draft.merchantName || undefined,
          transaction_date: draft.date.replace('T', ' ') + ':00',
          note: draft.note,
          image_path: draft.imagePath || undefined,
          ocr_amount: ocr?.amount ?? 0,
          ocr_date: ocr?.date ?? '',
          ocr_merchant: ocr?.merchant_name ?? '',
          ocr_raw_texts: ocr ? JSON.stringify(ocr.raw_texts ?? []) : '',
        }
        await transactionApi.create(payload)
      }
      const count = draftsToSave.length
      setToast({ text: `✓ 成功提交 ${count} 笔交易` })
      onSuccess?.()
      // 手机端延迟关闭让 Toast 可见，PC 端也确保提示可见
      const closeDelay = isMobile ? 1500 : 2000
      if (!activeSlot) {
        setNoSlotDrafts([])
        setTimeout(() => { setToast(null); onClose() }, closeDelay)
      } else {
        setSlots(prev => prev.map((s, i) => i === activeIdx ? { ...s, saved: true } : s))
        const nextIdx = slots.findIndex((s, i) => i !== activeIdx && !s.saved)
        if (nextIdx >= 0) {
          setTimeout(() => setToast(null), closeDelay)
          setActiveIdx(nextIdx)
        } else {
          const allSaved = slots.every((s, i) => i === activeIdx || s.saved)
          if (allSaved) setTimeout(() => { setToast(null); onClose() }, closeDelay)
          else setTimeout(() => setToast(null), closeDelay)
        }
      }
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      setToast({ text: msg || '保存失败，请稍后重试', error: true })
    }
    finally { 
      setSaving(false)
      submittingRef.current = false
    }
  }

  // ── 新增模式：点击完成 ─────────────────────────────────────
  function handleCompleteClick() {
    if (currentDrafts.length === 0) return
    if (isMobile) {
      // 手机端：无论几笔都先弹底部核验抽屉
      setShowReview(true)
    } else if (activeSlot?.previewUrl && currentDrafts.length > 1) {
      setShowReview(true)
    } else {
      handleBatchSave()
    }
  }

  // ── 移除草稿（手机抽屉和 PC 草稿列均使用）────────────────────
  function removeDraft(i: number) {
    if (activeSlot) {
      setDrafts(prev => {
        const next = new Map(prev)
        const list = (next.get(activeSlot.id) ?? []).filter((_, j) => j !== i)
        if (list.length === 0) next.delete(activeSlot.id)
        else next.set(activeSlot.id, list)
        return next
      })
      setUsedTokens(new Set())
    } else {
      setNoSlotDrafts(prev => prev.filter((_, j) => j !== i))
    }
    // 删除草稿后清空表单金额/备注/商户，防止残留数据被"记一笔"重复提交
    patchForm({ amount: '', note: '', merchantId: 0, merchantName: '' })
  }

  // ── 编辑弹窗：打开 ────────────────────────────────────────
  function openEditDraft(index: number) {
    const draft = currentDrafts[index]
    if (!draft) return
    setEditingDraftIndex(index)
    setEditingDraftForm({ ...draft })
  }

  // ── 编辑弹窗：保存 ────────────────────────────────────────
  function saveEditDraft() {
    if (editingDraftForm === null || editingDraftIndex === null) return
    const idx = editingDraftIndex
    const form = editingDraftForm
    const cat = categories.find(c => c.id === form.categoryId)
    const updated: SlotDraft = {
      ...form,
      categoryName: cat?.name ?? form.categoryName,
    }
    if (activeSlot) {
      setDrafts(prev => {
        const next = new Map(prev)
        const list = (next.get(activeSlot.id) ?? []).map((d, i) => i === idx ? updated : d)
        next.set(activeSlot.id, list)
        return next
      })
    } else {
      setNoSlotDrafts(prev => prev.map((d, i) => i === idx ? updated : d))
    }
    closeEditDraft()
  }

  // ── 编辑弹窗：关闭 ────────────────────────────────────────
  function closeEditDraft() {
    setEditingDraftIndex(null)
    setEditingDraftForm(null)
  }

  // ── 右键菜单 — 只保留 ESC 关闭，关闭改用透明遮罩实现 ─────────
  useEffect(() => {
    if (!ctxMenu) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setCtxMenu(null) }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [ctxMenu])

  // ── Lightbox ESC 关闭 ─────────────────────────────────────
  useEffect(() => {
    if (!lightboxOpen) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setLightboxOpen(false) }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [lightboxOpen])

  function openCtxMenu(e: React.MouseEvent | React.TouchEvent, text: string, tokenKey: string) {
    e.preventDefault()
    const { clientX, clientY } = 'touches' in e ? e.touches[0] : (e as React.MouseEvent)
    setCtxMenu({ x: clientX, y: clientY, text, tokenKey })
  }

  function applyCtxFill(action: 'amount' | 'merchant' | 'date' | 'note') {
    const text = ctxMenu?.text ?? ''
    if (action === 'amount') {
      const n = parseFloat(text.replace(/[^\d.]/g, ''))
      if (!isNaN(n) && n > 0) patchForm({ amount: String(n) })
    } else if (action === 'merchant') {
      // 查找已有商户或仅填充名称
      const exist = merchants.find(m => m.name === text)
      if (exist) {
        patchForm({ merchantId: exist.id, merchantName: exist.name })
      } else {
        patchForm({ merchantName: text })
      }
    } else if (action === 'date') {
      // 完整日期时间：2026年5月27日14:05:33 / 2026-05-27 14:05
      const dtm = text.match(/(\d{4})[年\-/](\d{1,2})[月\-/](\d{1,2})[日]?\s*(\d{1,2}):(\d{2})/)
      if (dtm) {
        patchForm({ date: `${dtm[1]}-${dtm[2].padStart(2,'0')}-${dtm[3].padStart(2,'0')}T${dtm[4].padStart(2,'0')}:${dtm[5]}` })
      } else {
        // 仅日期：2026年5月27日 / 2026-05-27
        const dm = text.match(/(\d{4})[年\-/](\d{1,2})[月\-/](\d{1,2})/)
        if (dm) {
          patchForm({ date: `${dm[1]}-${dm[2].padStart(2,'0')}-${dm[3].padStart(2,'0')}T${dayjs().format('HH:mm')}` })
        } else {
          // 无年份日期时间：5月27日14:05 / 5-27 14:05 → 默认今年
          const mdtm = text.match(/(\d{1,2})[月\-/](\d{1,2})[日]?\s*(\d{1,2}):(\d{2})/)
          if (mdtm) {
            const year = dayjs().year()
            patchForm({ date: `${year}-${mdtm[1].padStart(2,'0')}-${mdtm[2].padStart(2,'0')}T${mdtm[3].padStart(2,'0')}:${mdtm[4]}` })
          } else {
            // 无年份仅日期：5月27日 / 5-27 → 默认今年
            const mdm = text.match(/(\d{1,2})[月\-/](\d{1,2})/)
            if (mdm) {
              const year = dayjs().year()
              patchForm({ date: `${year}-${mdm[1].padStart(2,'0')}-${mdm[2].padStart(2,'0')}T${dayjs().format('HH:mm')}` })
            } else {
              // 仅时间：14:05:33 / 14:05 → 保留当前日期只替换时间
              const tm = text.match(/^(\d{1,2}):(\d{2})/)
              if (tm) {
                const curDate = (activeSlot?.form.date ?? dayjs().format('YYYY-MM-DDTHH:mm')).slice(0, 10)
                patchForm({ date: `${curDate}T${tm[1].padStart(2,'0')}:${tm[2]}` })
              }
            }
          }
        }
      }
    } else if (action === 'note') {
      const cur = activeSlot?.form.note ?? ''
      patchForm({ note: cur ? `${cur}、${text}` : text })
    }
    // 标记该 OCR token 为已使用（按位置而非文本内容）
    if (ctxMenu?.tokenKey) {
      setUsedTokens(prev => new Set(prev).add(ctxMenu.tokenKey))
    }
    setCtxMenu(null)
  }

  if (!open) return null

  const form = activeSlot?.form ?? noSlotForm
  const ocrResult = activeSlot?.ocrResult ?? null
  const isRecognizing = activeSlot?.status === 'recognizing'
  const filteredCats = categories.filter(c => c.type === form.type)
  const isEditMode = !!initialData
  // 统一列表在 OCR 栏内，不再需要独立的草稿列
  const modalWidth = 'max-w-5xl'
  const colWidth = 'w-1/4'

  // ── 缩略图条 ─────────────────────────────────────────────
  const ThumbnailStrip = !isEditMode && (
    <div className="flex-shrink-0 flex items-center gap-1.5 px-2 py-2 bg-gray-50 border-t border-gray-100 overflow-x-auto">
      {slots.map((slot, i) => (
        <button
          key={slot.id}
          type="button"
          onClick={() => setActiveIdx(i)}
          className={`relative flex-shrink-0 w-11 h-14 rounded-lg overflow-hidden border-2 transition-all ${
            i === activeIdx ? 'border-blue-500 shadow-md' : 'border-transparent opacity-70 hover:opacity-100'
          }`}
        >
          <img src={slot.previewUrl} alt="" className="w-full h-full object-cover" />
          {/* 识别中 */}
          {slot.status === 'recognizing' && (
            <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
              <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
            </div>
          )}
          {/* 待识别 */}
          {slot.status === 'pending' && (
            <div className="absolute inset-0 bg-black/30 flex items-center justify-center">
              <span className="text-white text-xs">⏳</span>
            </div>
          )}
          {/* 识别失败 */}
          {slot.status === 'error' && (
            <div className="absolute bottom-0 inset-x-0 bg-red-500/90 text-center py-0.5">
              <span className="text-white text-xs">!</span>
            </div>
          )}
          {/* 已保存 */}
          {slot.saved && (
            <div className="absolute inset-0 bg-green-500/20 flex items-end justify-center pb-0.5">
              <span className="text-green-600 text-base font-bold drop-shadow">✓</span>
            </div>
          )}
          {/* 有草稿未保存 */}
          {!slot.saved && (drafts.get(slot.id) ?? []).length > 0 && (
            <div className="absolute top-0.5 right-0.5 bg-blue-500 text-white text-xs font-bold rounded-full w-5 h-5 flex items-center justify-center shadow">
              {(drafts.get(slot.id) ?? []).length}
            </div>
          )}
        </button>
      ))}
      {/* 添加更多 */}
      <button
        type="button"
        onClick={() => fileInputRef.current?.click()}
        className="flex-shrink-0 w-11 h-14 rounded-lg border-2 border-dashed border-gray-300 hover:border-blue-400 flex items-center justify-center transition-colors"
      >
        <span className="text-gray-400 text-xl leading-none">+</span>
      </button>
    </div>
  )

  // ── 图片栏 ────────────────────────────────────────────────
  const ImageCol = (
    <div className={isMobile ? 'w-full' : `${isDetailMode ? 'w-1/2' : colWidth} flex-shrink-0 flex flex-col border-r border-gray-100 bg-gray-50`}>
      {/* 主图区 */}
      <div
        className={isMobile ? 'relative w-full h-56 bg-gray-100 rounded-xl overflow-hidden' : 'relative flex-1 overflow-hidden'}
        style={isMobile ? undefined : { minHeight: 0 }}
      >
        {!activeSlot?.previewUrl ? (
          isDetailMode ? (
            <div className="w-full h-full flex flex-col items-center justify-center gap-2 text-gray-300">
              <svg className="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                  d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <p className="text-xs">无凭证图片</p>
            </div>
          ) : (
            <div
              onClick={() => fileInputRef.current?.click()}
              onDrop={handleDrop}
              onDragOver={e => e.preventDefault()}
              className="w-full h-full flex flex-col items-center justify-center gap-2 cursor-pointer hover:bg-gray-100 transition-colors"
            >
              <svg className="w-10 h-10 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                  d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <p className="text-xs text-gray-400 text-center px-2">点击或拖拽<br />支持多选</p>
            </div>
          )
        ) : (
          <>
            <img
              src={activeSlot.previewUrl}
              alt="凭证"
              onClick={() => setLightboxOpen(true)}
              className="w-full h-full object-contain cursor-zoom-in"
            />
            {isRecognizing && (
              <div className="absolute inset-0 bg-black/40 flex flex-col items-center justify-center gap-2">
                <div className="w-8 h-8 border-4 border-white border-t-transparent rounded-full animate-spin" />
                <p className="text-white text-xs">识别中…</p>
              </div>
            )}
            {!isDetailMode && activeSlot.saved && (
              <div className="absolute top-2 left-2 bg-green-500 text-white text-xs px-2 py-0.5 rounded-full">
                已保存
              </div>
            )}
          </>
        )}
      </div>
      {/* 缩略图条 - detail 模式不显示 */}
      {!isDetailMode && ThumbnailStrip}
      {/* 底部提示 */}
      {!isMobile && !isDetailMode && slots.length > 0 && (
        <div className="px-3 py-1.5 flex-shrink-0 flex items-center justify-between">
          <p className="text-xs text-gray-400">点击放大</p>
          {slots.length > 1 && (
            <p className="text-xs text-gray-400">{savedCount}/{slots.length} 已保存</p>
          )}
        </div>
      )}
    </div>
  )

  // ── OCR 结果栏 ────────────────────────────────────────────
  const OcrCol = (
    <div className={isMobile ? 'w-full' : `${colWidth} flex-shrink-0 flex flex-col border-r border-gray-100 overflow-hidden`}>
      {!isMobile && (
        <div className="px-4 pt-4 pb-2 flex-shrink-0 border-b border-gray-50 flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-700">识别结果</span>
            {ocrResult?.ai_mode && <span className="text-xs bg-green-100 text-green-600 px-1.5 py-0.5 rounded-full">AI</span>}
            {isRecognizing && <span className="text-xs text-gray-400">识别中…</span>}
          </div>
          {isEditMode && activeSlot?.previewUrl && activeSlot?.status !== 'recognizing' && (
            <button
              type="button"
              onClick={async () => {
                const slot = slots[activeIdx]
                if (!slot?.previewUrl) return
                setSlots(prev => prev.map((s, i) => i === activeIdx ? { ...s, status: 'recognizing' } : s))
                try {
                  const blob = await fetch(slot.previewUrl).then(r => r.blob())
                  const file = new File([blob], 'reocr.jpg', { type: blob.type || 'image/jpeg' })
                  const res = await uploadApi.ocrAnalyze(file, slot.originalHash)
                  const ocr = res.data.data
                  setSlots(prev => prev.map((s, i) => {
                    if (i !== activeIdx) return s
                    const form = { ...s.form }
                    if (ocr?.ai_mode) {
                      if ((ocr.amount ?? 0) > 0) form.amount = String(ocr.amount)
                      if (ocr.date) {
                        // 提取日期部分（YYYY-MM-DD），兼容 "2024-05-20T14:30" 和 "2024-05-20" 两种格式
                        const datePart = ocr.date.match(/(\d{4}-\d{2}-\d{2})/)?.[1] ?? ocr.date.slice(0, 10)
                        form.date = `${datePart}T${extractTimeFromOcrDate(ocr.date)}`
                      }
                      if (ocr.merchant_id) { form.merchantId = ocr.merchant_id; form.merchantName = ocr.merchant_name ?? '' }
                    }
                    return { ...s, status: 'done' as SlotStatus, ocrResult: ocr ?? null }
                  }))
                } catch {
                  setSlots(prev => prev.map((s, i) => i === activeIdx ? { ...s, status: 'error' } : s))
                }
              }}
              className="text-xs text-blue-500 hover:text-blue-600 transition-colors flex-shrink-0"
            >
              🔄 重新识别
            </button>
          )}
        </div>
      )}
      <div className={isMobile ? 'py-2' : 'flex-1 overflow-y-auto p-4'}>
        {!ocrResult && !isRecognizing && (
          <div className="h-full flex items-center justify-center">
            <p className="text-xs text-gray-300 text-center">
              {activeSlot?.status === 'pending' ? '等待识别…' : '上传图片后自动识别'}
            </p>
          </div>
        )}
        {isRecognizing && (
          <div className="h-full flex items-center justify-center">
            <p className="text-xs text-gray-400">识别中…</p>
          </div>
        )}
        {ocrResult && !isRecognizing && (
          <>
            {ocrResult.ai_mode && (
              <div className="mb-3 bg-green-50 rounded-xl p-3 space-y-1.5">
                {(ocrResult.amount ?? 0) > 0 && (
                  <p className="text-sm">💰 <span className="font-semibold text-green-600">¥{ocrResult.amount}</span></p>
                )}
                {ocrResult.date && <p className="text-sm">📅 {ocrResult.date}</p>}
                {ocrResult.merchant_name && <p className="text-sm">🏪 {ocrResult.merchant_name}</p>}
                <button
                  type="button"
                  onClick={() => {
                    const patch: Partial<SlotForm> = {}
                    if ((ocrResult.amount ?? 0) > 0) patch.amount = String(ocrResult.amount)
                    if (ocrResult.date) {
                      const datePart = ocrResult.date.match(/(\d{4}-\d{2}-\d{2})/)?.[1] ?? ocrResult.date.slice(0, 10)
                      patch.date = `${datePart}T${extractTimeFromOcrDate(ocrResult.date)}`
                    }
                    if (ocrResult.merchant_id) patch.merchantId = ocrResult.merchant_id
                    if (ocrResult.merchant_name) patch.merchantName = ocrResult.merchant_name
                    patchForm(patch)
                  }}
                  className="w-full text-xs bg-green-600 hover:bg-green-700 text-white py-1.5 rounded-lg transition-colors"
                >
                  一键填入
                </button>
              </div>
            )}
            {/* ── 统一列表：AI 建议 + 已暂存草稿 ──────────────────── */}
            {((ocrResult.llm && ocrResult.llm.length > 0) || currentDrafts.length > 0) && (
              <div className="mb-3">
                <div className="flex items-center justify-between px-1 mb-2">
                  <span className="text-sm font-semibold text-gray-700">
                    {currentDrafts.length > 0
                      ? `已暂存 ${currentDrafts.length} 笔`
                      : `AI 建议 · ${(ocrResult.llm ?? []).length} 笔`}
                  </span>
                  {currentDrafts.length > 0 && (
                    <span className="text-xs text-gray-400">点击编辑或确认</span>
                  )}
                </div>

                {/* 已暂存的草稿列表 */}
                {currentDrafts.map((draft, di) => {
                  const catIcon = categories.find(c => c.id === draft.categoryId)?.icon ?? (draft.type === 'income' ? '💰' : '💸')
                  return (
                    <div
                      key={draft.id}
                      className="flex items-center gap-2.5 bg-white rounded-xl p-3 shadow-sm border border-gray-100 mb-2 hover:border-blue-200 transition-colors"
                    >
                      <span className="text-xl flex-shrink-0">{catIcon}</span>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <div className="min-w-0">
                            <span className="text-sm font-medium text-gray-800 truncate">
                              {draft.categoryName || '未分类'}
                            </span>
                            {draft.merchantName && (
                              <span className="text-xs ml-1 text-gray-500">· {draft.merchantName}</span>
                            )}
                          </div>
                          <span className={`text-sm font-bold ml-2 flex-shrink-0 ${draft.type === 'income' ? 'text-green-600' : 'text-red-600'}`}>
                            {draft.type === 'income' ? '+' : '-'}¥{draft.amount.toFixed(2)}
                          </span>
                        </div>
                        <div className="flex items-center justify-between mt-0.5">
                          <span className="text-xs text-gray-400">
                            {draft.date.slice(0, 16).replace('T', ' ')}
                            {draft.note && ` · ${draft.note}`}
                          </span>
                          <div className="flex items-center gap-1 flex-shrink-0">
                            <button
                              type="button"
                              onClick={() => openEditDraft(di)}
                              className="text-xs px-2 py-1 rounded-lg bg-blue-50 text-blue-600 hover:bg-blue-100 transition-colors"
                            >
                              编辑
                            </button>
                            <button
                              type="button"
                              onClick={() => removeDraft(di)}
                              className="text-xs px-2 py-1 rounded-lg bg-gray-50 text-gray-500 hover:bg-red-50 hover:text-red-500 transition-colors"
                            >
                              删除
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>
                  )
                })}

                {/* 尚未接受的 AI 建议（仅当有 LLM 建议且存在未填入的条目时显示） */}
                {ocrResult.llm && ocrResult.llm.map((llm, i) => {
                  const filled = filledLlmIndices.has(i)
                  if (filled) return null  // 已接受的不再重复显示
                  const cat = categories.find(c => c.id === llm.category_id)
                  const catIcon = cat?.icon ?? (llm.type === 'income' ? '💰' : '💸')
                  return (
                    <div
                      key={`llm_${i}`}
                      className="flex items-center gap-2.5 bg-white rounded-xl p-3 shadow-sm border border-gray-100 mb-2 hover:border-indigo-200 transition-colors"
                    >
                      <span className="text-xl flex-shrink-0">{catIcon}</span>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <div className="min-w-0">
                            <span className="text-sm font-medium text-gray-800 truncate">
                              {cat?.name ?? llm.category_hint ?? '未分类'}
                            </span>
                            {llm.merchant_name && (
                              <span className="text-xs ml-1 text-gray-500">· {llm.merchant_name}</span>
                            )}
                          </div>
                          <span className={`text-sm font-bold ml-2 flex-shrink-0 ${llm.type === 'income' ? 'text-green-600' : 'text-red-600'}`}>
                            {llm.type === 'income' ? '+' : '-'}¥{(llm.amount ?? 0).toFixed(2)}
                          </span>
                        </div>
                        <div className="flex items-center justify-between mt-0.5">
                          <span className="text-xs text-gray-400">
                            {llm.date || ''}
                            {llm.note && ` · ${llm.note}`}
                          </span>
                          <button
                            type="button"
                            onClick={() => applyLLMSuggestion(llm, i)}
                            className="text-xs px-3 py-1 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white transition-colors"
                          >
                            接受
                          </button>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
            {!ocrResult.llm && ocrResult.llm_error && (
              <p className="text-xs text-gray-400 mb-2 px-1">AI 分析不可用：{ocrResult.llm_error}</p>
            )}
            {(ocrResult.raw_texts ?? []).length > 0 && (
              <div className="flex-1 flex flex-col min-h-0">
                <p className="text-xs text-gray-300 mb-1 flex-shrink-0">
                  {isMobile ? '点击词条选择填入字段' : '左键选中数字入算式 · 右键填入字段'}
                </p>
                <div className={isMobile ? 'bg-gray-50 rounded-xl p-3 max-h-48 overflow-y-auto space-y-1.5' : 'flex-1 overflow-y-auto space-y-1.5'}>
                  {(ocrResult.raw_texts ?? []).map((line, i) => (
                    <div key={i} className="flex flex-wrap gap-1">
                      {line.split(/\s+/).filter(Boolean).map((token, j) => {
                        const tokenKey = `${i}_${j}`
                        return (
                          <span
                            key={j}
                            onContextMenu={e => openCtxMenu(e, token, tokenKey)}
                            onClick={e => {
                              if (isMobile) {
                                // 移动端：直接点击弹出填写菜单
                                const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
                                setCtxMenu({
                                  x: rect.left,
                                  y: Math.min(rect.bottom + 4, window.innerHeight - 230),
                                  text: token,
                                  tokenKey,
                                })
                              } else {
                                toggleToken(token)
                              }
                            }}
                            className={`text-xs font-mono border rounded px-1.5 py-0.5 select-none transition-colors ${
                              usedTokens.has(tokenKey)
                                ? 'text-gray-400 bg-gray-100 border-gray-200 line-through cursor-default'
                                : isMobile
                                  ? 'text-gray-700 bg-gray-50 border-gray-200 active:bg-blue-50 active:border-blue-300 cursor-pointer'
                                  : selectedTokens.has(token)
                                    ? 'bg-blue-500 text-white border-blue-500 cursor-pointer'
                                    : isNumericToken(token)
                                      ? 'bg-blue-50 border-blue-200 text-blue-700 hover:bg-blue-100 cursor-pointer'
                                      : 'text-gray-700 bg-gray-50 border-gray-200 cursor-context-menu'
                            }`}
                          >
                            {token}
                          </span>
                        )
                      })}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* ── OCR Footer：计算器 ──────────────────────────────── */}
            {calcResult && (
              <div className="flex-shrink-0 border-t border-gray-100 bg-blue-50 p-3 space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-gray-400">算式</span>
                  <button
                    type="button"
                    onClick={() => setSelectedTokens(new Map())}
                    className="text-xs text-gray-400 hover:text-gray-600"
                  >
                    清空
                  </button>
                </div>
                <p className="text-xs font-mono text-gray-700 break-all">{calcResult.formula}</p>
                <div className="flex items-center justify-between">
                  <p className="text-base font-bold text-blue-600">= ¥{calcResult.result.toFixed(2)}</p>
                  <div className="flex gap-1.5">
                    <button
                      type="button"
                      onClick={() => {
                        const cur = parseFloat(form.amount) || 0
                        const sign = form.type === 'expense' ? -1 : 1
                        patchForm({ amount: String(Math.abs(cur + sign * calcResult.result)) })
                        setSelectedTokens(new Map())
                      }}
                      className="px-2.5 py-1 text-xs bg-white border border-blue-200 text-blue-600 hover:bg-blue-100 rounded-lg transition-colors"
                    >
                      累加
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        patchForm({ amount: String(Math.abs(calcResult.result)) })
                        setSelectedTokens(new Map())
                      }}
                      className="px-2.5 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
                    >
                      填入金额
                    </button>
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )

  // ── 分类字段（手机端移到备注之后，PC 保持金额之后）────────────
  const categoryFieldJsx = (
    <div>
      <p className="text-xs text-gray-500 mb-1.5">分类</p>
      {catLoading ? (
        <div className="grid grid-cols-4 gap-1.5">
          {Array.from({ length: 8 }).map((_, i) => <div key={i} className="h-14 rounded-xl bg-gray-100 animate-pulse" />)}
        </div>
      ) : (
        <div className="grid grid-cols-4 gap-1.5">
          {filteredCats.map(cat => (
            <button key={cat.id} type="button"
              onClick={() => patchForm({ categoryId: cat.id })}
              className={`flex flex-col items-center gap-0.5 p-1.5 rounded-xl border text-xs transition-colors ${
                form.categoryId === cat.id
                  ? 'border-blue-500 bg-blue-50 text-blue-600'
                  : 'border-gray-200 hover:bg-gray-50 text-gray-600'
              }`}
            >
              <span className="text-lg">{cat.icon || '📌'}</span>
              <span className="truncate w-full text-center" style={{ fontSize: '10px' }}>{cat.name}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )

  // ── 表单栏 ────────────────────────────────────────────────
  const FormCol = (
    <div className={isMobile ? 'w-full' : `${isDetailMode ? 'w-1/2' : 'flex-1'} flex flex-col overflow-hidden`}>
      {isDetailMode ? (
        // ── 详情模式：只读展示 ───────────────────────────────
        <div className={isMobile ? 'space-y-3' : 'flex-1 overflow-y-auto p-4 space-y-3'}>
          {/* 金额（突出显示） */}
          <div className="text-center py-4 bg-gray-50 rounded-xl">
            <p className={`text-2xl font-bold ${form.type === 'income' ? 'text-green-600' : 'text-red-600'}`}>
              {form.type === 'income' ? '+' : '-'}¥{(Number(form.amount) || 0).toFixed(2)}
            </p>
            <p className="text-xs text-gray-400 mt-1">{form.type === 'income' ? '收入' : '支出'}</p>
          </div>

          {/* 分类 */}
          <div className="flex items-center justify-between py-2.5 border-b border-gray-50">
            <span className="text-sm text-gray-400">分类</span>
            <span className="text-sm text-gray-800 font-medium">
              {categories.find(c => c.id === form.categoryId)?.icon ?? '📌'}{' '}
              {categories.find(c => c.id === form.categoryId)?.name ?? '-'}
            </span>
          </div>

          {/* 商户 */}
          {form.merchantName && (
            <div className="flex items-center justify-between py-2.5 border-b border-gray-50">
              <span className="text-sm text-gray-400">商户</span>
              <span className="text-sm text-gray-800">{form.merchantName}</span>
            </div>
          )}

          {/* 日期时间 */}
          <div className="flex items-center justify-between py-2.5 border-b border-gray-50">
            <span className="text-sm text-gray-400">日期</span>
            <span className="text-sm text-gray-800">{form.date.replace('T', ' ')}</span>
          </div>

          {/* 备注 */}
          {form.note && (
            <div className="flex items-start justify-between gap-4 py-2.5 border-b border-gray-50">
              <span className="text-sm text-gray-400 flex-shrink-0">备注</span>
              <span className="text-sm text-gray-800 text-right">{form.note}</span>
            </div>
          )}

          {/* 关闭按钮 */}
          <button
            type="button"
            onClick={onClose}
            className="w-full py-2.5 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors mt-4"
          >
            关闭
          </button>
        </div>
      ) : (
        // ── 编辑/新增模式：可编辑表单 ───────────────────────────
        <form
          onSubmit={isEditMode ? handleSave : (e) => e.preventDefault()}
          className={isMobile ? 'space-y-3' : 'flex-1 overflow-y-auto p-4 space-y-3'}
        >
          {/* 收入/支出 */}
          <div className="flex rounded-xl overflow-hidden border border-gray-200">
            {(['expense', 'income'] as const).map(t => (
              <button key={t} type="button"
                onClick={() => patchForm({ type: t, categoryId: '' })}
                className={`flex-1 py-2 text-sm font-medium transition-colors ${
                  form.type === t
                    ? t === 'expense' ? 'bg-red-500 text-white' : 'bg-green-500 text-white'
                    : 'bg-white text-gray-500 hover:bg-gray-50'
                }`}
              >
                {t === 'expense' ? '支出' : '收入'}
              </button>
            ))}
          </div>

          {/* 金额 */}
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 font-medium text-sm">¥</span>
            <input type="number" step="0.01" min="0.01"
              value={form.amount} onChange={e => patchForm({ amount: e.target.value })}
              placeholder="0.00" required
              className="w-full pl-7 pr-3 py-2.5 border border-gray-200 rounded-xl text-xl font-bold focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {/* 分类 — PC 在金额之后，手机端移到备注之后 */}
          {!isMobile && categoryFieldJsx}

          {/* 日期时间 */}
          <div>
            <p className="text-xs text-gray-500 mb-1">日期时间</p>
            <input type="datetime-local" value={form.date} onChange={e => patchForm({ date: e.target.value })} required className={inputCls} />
          </div>

          {/* 商户 */}
          <div className="relative">
            <p className="text-xs text-gray-500 mb-1">商户</p>
            <input
              type="text"
              value={form.merchantName}
              onChange={e => {
                const v = e.target.value
                patchForm({ merchantName: v, merchantId: 0 })
              }}
              onBlur={() => {
                // 失去焦点时尝试匹配已有商户
                if (form.merchantName && !form.merchantId) {
                  const match = merchants.find(m => m.name === form.merchantName)
                  if (match) patchForm({ merchantId: match.id })
                }
              }}
              placeholder="输入商户名称（可选）" className={inputCls}
            />
            {/* 自动补全下拉 */}
            {form.merchantName && !form.merchantId && (
              <div className="absolute z-10 left-0 right-0 top-full bg-white border border-gray-200 rounded-lg shadow-lg max-h-36 overflow-y-auto mt-0.5">
                {merchants
                  .filter(m => m.name.toLowerCase().includes(form.merchantName.toLowerCase()))
                  .slice(0, 5)
                  .map(m => (
                    <button
                      key={m.id}
                      type="button"
                      onClick={() => patchForm({ merchantId: m.id, merchantName: m.name })}
                      className="w-full text-left px-3 py-2 text-sm hover:bg-blue-50 hover:text-blue-600 transition-colors"
                    >
                      {m.name}
                    </button>
                  ))}
              </div>
            )}
          </div>

          {/* 备注 */}
          <div>
            <p className="text-xs text-gray-500 mb-1">备注</p>
            {form.note.split('、').filter(Boolean).length > 1 && (
              <div className="flex flex-wrap gap-1 mb-1.5">
                {form.note.split('、').filter(Boolean).map((seg, i, arr) => (
                  <span key={i} className="inline-flex items-center gap-0.5 text-xs bg-gray-100 text-gray-700 rounded-full px-2 py-0.5">
                    {seg}
                    <button
                      type="button"
                      onClick={() => patchForm({ note: arr.filter((_, j) => j !== i).join('、') })}
                      className="ml-0.5 text-gray-400 hover:text-red-500 leading-none"
                    >×</button>
                  </span>
                ))}
              </div>
            )}
            <input type="text" value={form.note} onChange={e => patchForm({ note: e.target.value })}
              placeholder="添加备注（可选）" className={inputCls} />
          </div>

          {/* 分类 — 手机端在备注之后 */}
          {isMobile && categoryFieldJsx}

          {/* ── 按钮区 ──────────────────────────────────────── */}
          {isEditMode ? (
            <button
              type="submit"
              disabled={saving || isRecognizing || !activeSlot}
              className="w-full py-2.5 font-semibold rounded-xl transition-colors text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white"
            >
              {saving ? '保存中…' : isRecognizing ? '识别中…' : '保存'}
            </button>
          ) : (
            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleAddDraft}
                disabled={saving || isRecognizing || (form.type === 'expense' && !activeSlot)}
                className="flex-1 py-2.5 font-semibold rounded-xl transition-colors text-sm bg-white border border-blue-200 text-blue-600 hover:bg-blue-50 disabled:border-gray-200 disabled:text-gray-400"
              >
                {isRecognizing ? '识别中…' : '记一笔'}
              </button>
              <button
                type="button"
                onClick={handleCompleteClick}
                disabled={saving || isRecognizing || currentDrafts.length === 0}
                className={`flex-1 py-2.5 font-semibold rounded-xl transition-colors text-sm text-white ${
                  currentDrafts.length > 0
                    ? 'bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300'
                    : 'bg-gray-300'
                }`}
              >
                {saving ? '提交中…' : currentDrafts.length > 0 ? `完成 ${currentDrafts.length} 笔` : '完成'}
              </button>
            </div>
          )}

          {/* 多图时显示进度 */}
          {!isEditMode && slots.length > 1 && (
            <p className="text-center text-xs text-gray-400">{savedCount} / {slots.length} 张已保存</p>
          )}
        </form>
      )}
    </div>
  )

  // ── 填入菜单（PC 右键 / 移动端点击）────────────────────────
  // 用透明遮罩关闭，避免 document.click 与 React 合成事件冲突
  const CtxMenuEl = ctxMenu ? (
    <>
      {/* 透明遮罩：点击任意空白处关闭 */}
      <div className="fixed inset-0 z-[79]" onClick={() => setCtxMenu(null)} />
      <div
        className="fixed z-[80] bg-white rounded-2xl shadow-2xl border border-gray-100 overflow-hidden"
        style={{ left: Math.min(ctxMenu.x + 4, window.innerWidth - 192), top: Math.min(ctxMenu.y + 4, window.innerHeight - 224), minWidth: 180 }}
        onClick={e => e.stopPropagation()}
      >
        <div className="px-3 py-2 bg-gray-50 border-b border-gray-100">
          <p className="text-xs text-gray-500 truncate max-w-40">「{ctxMenu.text}」</p>
        </div>
        {([
          { icon: '💰', label: '填入金额', action: 'amount' },
          { icon: '🏪', label: '填入商户', action: 'merchant' },
          { icon: '📅', label: '填入日期', action: 'date' },
          { icon: '📝', label: '填入备注', action: 'note' },
        ] as const).map(opt => (
          <button key={opt.action} onClick={() => applyCtxFill(opt.action)}
            className="w-full flex items-center gap-2.5 px-3 py-2.5 text-sm text-gray-700 hover:bg-blue-50 hover:text-blue-600 transition-colors text-left"
          >
            <span>{opt.icon}</span><span>{opt.label}</span>
          </button>
        ))}
      </div>
    </>
  ) : null

  // ── 手机端底部抽屉（核验草稿后提交）────────────────────────
  const MobileReviewSheet = isMobile && showReview ? (
    <div className="fixed inset-0 z-[70] flex flex-col justify-end">
      <div className="absolute inset-0 bg-black/40" onClick={() => setShowReview(false)} />
      <div className="relative bg-white rounded-t-2xl flex flex-col" style={{ maxHeight: '82vh' }}>
        {/* 顶部把手 + 标题 */}
        <div className="flex-shrink-0 pt-3 pb-0">
          <div className="w-10 h-1 bg-gray-200 rounded-full mx-auto mb-3" />
        </div>
        <div className="flex items-center justify-between px-4 pb-3 border-b border-gray-100 flex-shrink-0">
          <span className="text-sm font-semibold text-gray-800">
            已暂存 {currentDrafts.length} 笔
          </span>
          <button onClick={() => setShowReview(false)} className="text-gray-400 p-1 text-lg leading-none">✕</button>
        </div>
        {/* 草稿列表 */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          {currentDrafts.map((d, i) => (
            <div key={d.id} className="flex items-start gap-3 bg-gray-50 rounded-xl p-3">
              <div className="flex-1 min-w-0 space-y-0.5">
                <div className="flex items-center gap-2">
                  <span className={`text-base font-bold ${d.type === 'income' ? 'text-green-600' : 'text-red-600'}`}>
                    {d.type === 'income' ? '+' : '-'}¥{d.amount.toFixed(2)}
                  </span>
                  <span className="text-xs text-gray-500 bg-white px-2 py-0.5 rounded-full border border-gray-200">
                    {categories.find(c => c.id === d.categoryId)?.icon ?? ''} {d.categoryName}
                  </span>
                </div>
                {(d.merchantName || d.note) && (
                  <p className="text-xs text-gray-400 truncate">
                    {[d.merchantName, d.note].filter(Boolean).join(' · ')}
                  </p>
                )}
                <p className="text-xs text-gray-300">{d.date.slice(0, 16).replace('T', ' ')}</p>
              </div>
              <button
                type="button"
                onClick={() => removeDraft(i)}
                className="flex-shrink-0 w-7 h-7 flex items-center justify-center rounded-full text-gray-300 hover:text-red-500 hover:bg-red-50 transition-colors text-base"
              >✕</button>
            </div>
          ))}
        </div>
        {/* 总计 + 按钮 */}
        <div className="flex-shrink-0 border-t border-gray-100 px-4 py-4 space-y-3">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-500">合计</span>
            <span className="font-bold text-gray-800">
              {currentDrafts.reduce((s, d) => s + (d.type === 'expense' ? -d.amount : d.amount), 0) >= 0 ? '+' : ''}
              ¥{Math.abs(currentDrafts.reduce((s, d) => s + (d.type === 'expense' ? -d.amount : d.amount), 0)).toFixed(2)}
            </span>
          </div>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setShowReview(false)}
              className="flex-1 py-3 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors font-medium"
            >
              继续记账
            </button>
            <button
              type="button"
              onClick={() => { setShowReview(false); handleBatchSave() }}
              disabled={saving || currentDrafts.length === 0}
              className="flex-1 py-3 text-sm font-semibold bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white rounded-xl transition-colors"
            >
              {saving ? '提交中…' : `确认提交 ${currentDrafts.length} 笔`}
            </button>
          </div>
        </div>
      </div>
    </div>
  ) : null

  // ─ 草稿编辑弹窗 ────────────────────────────────────────────
  const DraftEditModal = editingDraftForm !== null && editingDraftIndex !== null ? (
    <div className="fixed inset-0 bg-black/60 z-[70] flex items-center justify-center p-6" onClick={closeEditDraft}>
      <div
        className="bg-white rounded-2xl shadow-2xl w-full max-w-md"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-gray-100">
          <h3 className="text-base font-semibold text-gray-800">编辑记账</h3>
          <button onClick={closeEditDraft} className="p-1.5 rounded-full hover:bg-gray-100 text-gray-500">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="p-5 space-y-3">
          {/* 收入/支出 */}
          <div className="flex rounded-xl overflow-hidden border border-gray-200">
            {(['expense', 'income'] as const).map(t => (
              <button key={t} type="button"
                onClick={() => setEditingDraftForm({ ...editingDraftForm, type: t, categoryId: 0 })}
                className={`flex-1 py-2 text-sm font-medium transition-colors ${
                  editingDraftForm.type === t
                    ? t === 'expense' ? 'bg-red-500 text-white' : 'bg-green-500 text-white'
                    : 'bg-white text-gray-500 hover:bg-gray-50'
                }`}
              >
                {t === 'expense' ? '支出' : '收入'}
              </button>
            ))}
          </div>

          {/* 金额 */}
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 font-medium text-sm">¥</span>
            <input type="number" step="0.01" min="0.01"
              value={editingDraftForm.amount}
              onChange={e => setEditingDraftForm({ ...editingDraftForm, amount: parseFloat(e.target.value) || 0 })}
              className="w-full pl-7 pr-3 py-2.5 border border-gray-200 rounded-xl text-lg font-bold focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {/* 商户 */}
          <div>
            <p className="text-xs text-gray-500 mb-1">商户</p>
            <input type="text"
              value={editingDraftForm.merchantName}
              onChange={e => setEditingDraftForm({ ...editingDraftForm, merchantName: e.target.value, merchantId: 0 })}
              placeholder="输入商户名称（可选）"
              className={inputCls}
            />
          </div>

          {/* 分类 */}
          <div>
            <p className="text-xs text-gray-500 mb-1.5">分类</p>
            <div className="grid grid-cols-4 gap-1.5">
              {categories.filter(c => c.type === editingDraftForm.type).map(cat => (
                <button key={cat.id} type="button"
                  onClick={() => setEditingDraftForm({ ...editingDraftForm, categoryId: cat.id })}
                  className={`flex flex-col items-center gap-0.5 p-1.5 rounded-xl border text-xs transition-colors ${
                    editingDraftForm.categoryId === cat.id
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:bg-gray-50 text-gray-600'
                  }`}
                >
                  <span className="text-lg">{cat.icon || '📌'}</span>
                  <span className="truncate w-full text-center" style={{ fontSize: '10px' }}>{cat.name}</span>
                </button>
              ))}
            </div>
          </div>

          {/* 备注 */}
          <div>
            <p className="text-xs text-gray-500 mb-1">备注</p>
            <input type="text"
              value={editingDraftForm.note}
              onChange={e => setEditingDraftForm({ ...editingDraftForm, note: e.target.value })}
              placeholder="添加备注（可选）"
              className={inputCls}
            />
          </div>

          {/* 日期时间 */}
          <div>
            <p className="text-xs text-gray-500 mb-1">日期时间</p>
            <input type="datetime-local"
              value={editingDraftForm.date}
              onChange={e => setEditingDraftForm({ ...editingDraftForm, date: e.target.value })}
              className={inputCls}
            />
          </div>
        </div>
        <div className="flex gap-3 px-5 py-3 border-t border-gray-100">
          <button type="button" onClick={closeEditDraft}
            className="flex-1 py-2.5 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors">
            取消
          </button>
          <button type="button" onClick={saveEditDraft}
            className="flex-1 py-2.5 text-sm font-semibold bg-blue-600 hover:bg-blue-700 text-white rounded-xl transition-colors">
            保存
          </button>
        </div>
      </div>
    </div>
  ) : null

  // ── PC 确认弹窗（多笔提交前）────────────────────────────────
  const ReviewModal = !isMobile && showReview && activeSlot ? (
    <div className="fixed inset-0 bg-black/60 z-[70] flex items-center justify-center">
      <div className="bg-white rounded-2xl shadow-2xl flex flex-col" style={{ width: '80vw', height: '80vh' }}>
        <div className="flex items-center justify-between px-5 py-3 border-b border-gray-100 flex-shrink-0">
          <h3 className="text-base font-semibold text-gray-800">确认提交 {currentDrafts.length} 笔</h3>
          <button onClick={() => setShowReview(false)} className="p-1.5 rounded-full hover:bg-gray-100 text-gray-500">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="flex flex-1 overflow-hidden">
          <div className="flex-1 min-w-0 border-r border-gray-100 bg-gray-50 overflow-hidden">
            {activeSlot.previewUrl ? (
              <img src={activeSlot.previewUrl} alt="凭证" onClick={() => setLightboxOpen(true)}
                className="w-full h-full object-contain cursor-zoom-in" />
            ) : (
              <div className="w-full h-full flex items-center justify-center">
                <p className="text-xs text-gray-300">无图片</p>
              </div>
            )}
          </div>
          <div className="flex-shrink-0 overflow-y-auto p-4">
            <table className="text-sm border-collapse">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left text-xs text-gray-400 font-normal pb-2 pr-4 whitespace-nowrap">类型</th>
                  <th className="text-right text-xs text-gray-400 font-normal pb-2 pr-4 whitespace-nowrap">金额</th>
                  <th className="text-left text-xs text-gray-400 font-normal pb-2 pr-4 whitespace-nowrap">分类</th>
                  <th className="text-left text-xs text-gray-400 font-normal pb-2 pr-4 whitespace-nowrap">商户</th>
                  <th className="text-left text-xs text-gray-400 font-normal pb-2 pr-4 whitespace-nowrap">日期</th>
                  <th className="text-left text-xs text-gray-400 font-normal pb-2 whitespace-nowrap">备注</th>
                </tr>
              </thead>
              <tbody>
                {currentDrafts.map(d => (
                  <tr key={d.id} className="border-b border-gray-50">
                    <td className={`py-2 pr-4 text-xs font-medium whitespace-nowrap ${d.type === 'income' ? 'text-green-600' : 'text-red-600'}`}>
                      {d.type === 'income' ? '收入' : '支出'}
                    </td>
                    <td className={`py-2 pr-4 text-right text-sm font-bold whitespace-nowrap ${d.type === 'income' ? 'text-green-600' : 'text-red-600'}`}>
                      {d.type === 'income' ? '+' : '-'}¥{d.amount.toFixed(2)}
                    </td>
                    <td className="py-2 pr-4 text-xs text-gray-700 whitespace-nowrap">
                      {categories.find(c => c.id === d.categoryId)?.icon ?? ''} {d.categoryName}
                    </td>
                    <td className="py-2 pr-4 text-xs text-gray-600 whitespace-nowrap">{d.merchantName || '-'}</td>
                    <td className="py-2 pr-4 text-xs text-gray-500 whitespace-nowrap">{d.date.replace('T', ' ')}</td>
                    <td className="py-2 text-xs text-gray-500 whitespace-nowrap">{d.note || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        <div className="flex gap-3 px-5 py-3 border-t border-gray-100 flex-shrink-0">
          <button type="button" onClick={() => setShowReview(false)}
            className="flex-1 py-2.5 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors">
            返回修改
          </button>
          <button type="button" onClick={() => { setShowReview(false); handleBatchSave() }}
            disabled={saving}
            className="flex-1 py-2.5 text-sm font-semibold bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white rounded-xl transition-colors">
            {saving ? '提交中…' : `确认提交 ${currentDrafts.length} 笔`}
          </button>
        </div>
      </div>
    </div>
  ) : null

  // ── Toast 提示 ───────────────────────────────────────────
  const Toast = toast ? (
    <div className={`fixed top-6 left-1/2 -translate-x-1/2 z-[90] flex items-center gap-2 text-white text-sm px-5 py-3 rounded-2xl shadow-xl pointer-events-none ${toast.error ? 'bg-red-500/90' : 'bg-gray-900/90'}`}>
      {toast.error ? (
        <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M6 18L18 6M6 6l12 12" />
        </svg>
      ) : (
        <svg className="w-4 h-4 text-green-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
        </svg>
      )}
      {toast.text}
    </div>
  ) : null

  // ── Lightbox ─────────────────────────────────────────────
  const Lightbox = lightboxOpen && activeSlot?.previewUrl ? (
    <div className="fixed inset-0 bg-black/90 z-[80] flex items-center justify-center p-4"
      onClick={() => setLightboxOpen(false)}>
      <button onClick={() => setLightboxOpen(false)}
        className="absolute top-4 right-4 text-white bg-black/40 hover:bg-black/60 rounded-full p-2">
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
      <img src={activeSlot.previewUrl} alt="凭证大图"
        onClick={e => e.stopPropagation()}
        className="max-w-full max-h-full object-contain rounded-lg" />
    </div>
  ) : null

  // ── 隐藏 file input（支持多选，移动端不强制相机）────────────
  const HiddenInput = (
    <input ref={fileInputRef} type="file" accept="image/*"
      multiple={!isEditMode}
      className="hidden"
      onChange={handleFilesChange}
    />
  )

  // ── 移动端布局 ────────────────────────────────────────────
  if (isMobile) {
    return (
      <>
        {Toast}{Lightbox}{CtxMenuEl}{MobileReviewSheet}{DraftEditModal}
        <div className="fixed inset-0 bg-white z-50 flex flex-col overflow-y-auto">
          {HiddenInput}
          <div className="flex items-center justify-between p-4 border-b border-gray-100 sticky top-0 bg-white z-10">
            <h2 className="text-base font-semibold text-gray-800">
              {isDetailMode ? '交易详情' : isEditMode ? '编辑记账' : slots.length > 1 ? `新增记账（${activeIdx + 1}/${slots.length}）` : '新增记账'}
            </h2>
            <button onClick={onClose} className="p-2 rounded-full hover:bg-gray-100 text-gray-500">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div className="flex flex-col gap-4 p-4 pb-8">
            {ImageCol}
            {!isDetailMode && ocrResult && OcrCol}
            {FormCol}
          </div>
        </div>
      </>
    )
  }

  // ── PC 详情布局（两栏，各半）─────────────────────────────────
  if (isDetailMode) {
    return (
      <>
        {Toast}{Lightbox}
        <div
          className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-6"
          onClick={e => { if (e.target === e.currentTarget) onClose() }}
        >
          <div className="bg-white rounded-2xl shadow-2xl flex flex-col w-full max-w-5xl"
            style={{ height: 'min(90vh, 680px)' }}>
            <div className="flex items-center justify-between px-5 py-3 border-b border-gray-100 flex-shrink-0">
              <h2 className="text-base font-semibold text-gray-800">交易详情</h2>
              <button onClick={onClose} className="p-1.5 rounded-full hover:bg-gray-100 text-gray-500">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex flex-1 overflow-hidden">
              {ImageCol}
              {FormCol}
            </div>
          </div>
        </div>
      </>
    )
  }

  // ── PC 新增/编辑布局（三/四栏：图片、OCR、[草稿]、表单）────────
  return (
    <>
      {Toast}{Lightbox}{CtxMenuEl}{ReviewModal}{DraftEditModal}
      <div
        className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-6"
        onClick={e => { if (e.target === e.currentTarget) onClose() }}
      >
        <div className={`bg-white rounded-2xl shadow-2xl flex flex-col w-full ${modalWidth}`}
          style={{ height: 'min(90vh, 680px)' }}>
          {HiddenInput}
          <div className="flex items-center justify-between px-5 py-3 border-b border-gray-100 flex-shrink-0">
            <h2 className="text-base font-semibold text-gray-800">
              {isEditMode ? '编辑记账' : slots.length > 1 ? `新增记账（${activeIdx + 1}/${slots.length}）` : '新增记账'}
            </h2>
            <button onClick={onClose} className="p-1.5 rounded-full hover:bg-gray-100 text-gray-500">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div className="flex flex-1 overflow-hidden">
            {ImageCol}
            {OcrCol}
            {FormCol}
          </div>
        </div>
      </div>
    </>
  )
}
