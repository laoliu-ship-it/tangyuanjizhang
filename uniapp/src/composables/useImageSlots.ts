/**
 * 记账核心状态机：多图队列 -> 压缩 -> SHA256 -> OCR -> LLM -> 草稿。
 * 队列串行执行，避免并发打爆 OCR / LLM 限流。
 */
import { computed, onUnmounted, ref, type ComputedRef, type Ref } from 'vue'
import { MAX_IMAGE_COUNT } from '@/config'
import { llmApi } from '@/api/llm'
import { ocrApi } from '@/api/ocr'
import { useMetaStore } from '@/store/meta'
import type { LLMSuggestion, OcrUploadResp, TxType } from '@/api/types'
import { extractDatePart, extractTimeFromOcrDate, nowLocal } from '@/utils/datetime'
import {
  compressImage,
  pickImages,
  revokePreview,
  sha256,
  type PickedImage,
} from '@/utils/image'
import { checkImage, isSecCheckAvailable } from '@/utils/seccheck'
import { toast } from '@/utils/ui'

export type SlotStatus =
  | 'pending'
  | 'compressing'
  | 'ocr_recognizing'
  | 'llm_recognizing'
  | 'done'
  | 'error'

export interface SlotForm {
  type: TxType
  amount: string
  categoryId: number
  merchantId: number
  merchantName: string
  date: string
  note: string
}

export interface ImageSlot {
  id: string
  previewUrl: string
  status: SlotStatus
  error?: string
  ocrId: number
  imagePath: string
  rawTexts: string[]
  ocrAmount: number
  ocrDate: string
  ocrMerchant: string
  saved: boolean
  form: SlotForm
  llm: LLMSuggestion[]
  llmError: string
}

export interface SlotDraft {
  id: string
  type: TxType
  amount: number
  categoryId: number
  categoryName: string
  merchantId: number
  merchantName: string
  date: string
  note: string
  imagePath: string
  llmIndex?: number
}

export interface UseImageSlotsReturn {
  slots: Ref<ImageSlot[]>
  activeIndex: Ref<number>
  activeSlot: ComputedRef<ImageSlot | undefined>
  drafts: ComputedRef<SlotDraft[]>
  isRecognizing: ComputedRef<boolean>
  llmEnabled: Ref<boolean>
  totalDrafts: ComputedRef<number>
  form: ComputedRef<SlotForm>
  noSlotForm: Ref<SlotForm>
  loadLlmConfig: () => Promise<void>
  chooseAndAdd: () => Promise<void>
  addImages: (images: PickedImage[]) => Promise<void>
  retry: (slotId: string) => void
  retryLlm: (slotId: string) => void
  removeSlot: (slotId: string) => void
  select: (index: number) => void
  patchForm: (patch: Partial<SlotForm>) => void
  applySuggestion: (sug: LLMSuggestion, index: number) => void
  addDraft: (draft: SlotDraft) => void
  removeDraft: (index: number) => void
  updateDraft: (index: number, draft: SlotDraft) => void
  setDrafts: (list: SlotDraft[]) => void
  markSaved: (slotId: string) => void
  reset: () => void
}

let slotSeed = 0
function newSlotId(): string {
  slotSeed += 1
  return `slot_${Date.now()}_${slotSeed}`
}

export function defaultSlotForm(): SlotForm {
  return {
    type: 'expense',
    amount: '',
    categoryId: 0,
    merchantId: 0,
    merchantName: '',
    date: nowLocal(),
    note: '',
  }
}

function errMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message) {
    if (err.message.includes('超时') || err.message.includes('timeout')) return '识别超时，请重试'
    return err.message
  }
  return fallback
}

/** LLM 返回日期 → "YYYY-MM-DDTHH:mm"（无时间时使用 OCR 时间） */
function suggestionDate(value: string | undefined, ocrDate: string): string {
  if (!value) return nowLocal()
  const datePart = extractDatePart(value)
  const m = value.match(/(\d{1,2}):(\d{2})/)
  const time = m ? `${m[1].padStart(2, '0')}:${m[2]}` : extractTimeFromOcrDate(ocrDate)
  return `${datePart}T${time}`
}

export function useImageSlots(): UseImageSlotsReturn {
  const meta = useMetaStore()

  const slots = ref<ImageSlot[]>([])
  const activeIndex = ref(0)
  const draftMap = ref<Record<string, SlotDraft[]>>({})
  const noSlotDrafts = ref<SlotDraft[]>([])
  const noSlotForm = ref<SlotForm>(defaultSlotForm())
  const llmEnabled = ref(false)

  // 非响应式私有状态
  const imagesMap = new Map<string, PickedImage>()
  const queue: string[] = []
  let processing = false
  // 表单关闭/重置时递增；旧的异步队列看到代际变化后立即退出，避免与新一轮队列并发
  let generation = 0

  const activeSlot = computed<ImageSlot | undefined>(() => slots.value[activeIndex.value])
  const drafts = computed<SlotDraft[]>(() => {
    const slot = activeSlot.value
    return slot ? draftMap.value[slot.id] ?? [] : noSlotDrafts.value
  })
  const isRecognizing = computed<boolean>(() => {
    const status = activeSlot.value?.status
    return status === 'compressing' || status === 'ocr_recognizing' || status === 'llm_recognizing'
  })
  const totalDrafts = computed<number>(() => {
    let total = noSlotDrafts.value.length
    Object.keys(draftMap.value).forEach((key) => {
      total += draftMap.value[key]?.length ?? 0
    })
    return total
  })
  const form = computed<SlotForm>(() => activeSlot.value?.form ?? noSlotForm.value)

  function findSlot(id: string): ImageSlot | undefined {
    return slots.value.find((s) => s.id === id)
  }

  function setSlot(id: string, patch: Partial<ImageSlot>): void {
    const index = slots.value.findIndex((s) => s.id === id)
    if (index < 0) return
    slots.value[index] = { ...slots.value[index], ...patch }
  }

  function setDraftList(slotId: string, list: SlotDraft[]): void {
    const next = { ...draftMap.value }
    if (list.length) next[slotId] = list
    else delete next[slotId]
    draftMap.value = next
  }

  async function loadLlmConfig(): Promise<void> {
    try {
      const config = await llmApi.getConfig()
      llmEnabled.value = !!config?.enabled
    } catch (e) {
      llmEnabled.value = false
    }
  }

  /**
   * 根据 OCR 结果预填表单（只覆盖 OCR 能识别的字段，保留用户已输入内容）。
   */
  function patchFormByOcr(id: string, ocr: OcrUploadResp): void {
    const slot = findSlot(id)
    if (!slot) return
    const nextForm: SlotForm = { ...slot.form }
    if (ocr.ai_mode) {
      if ((ocr.amount ?? 0) > 0) nextForm.amount = String(ocr.amount)
      if (ocr.date) {
        nextForm.date = `${extractDatePart(ocr.date)}T${extractTimeFromOcrDate(ocr.date)}`
      }
      if (ocr.merchant_id) {
        nextForm.merchantId = ocr.merchant_id
        nextForm.merchantName = ocr.merchant_name ?? ''
      }
    }
    setSlot(id, {
      ocrId: ocr.ocr_id ?? 0,
      imagePath: ocr.image_path ?? '',
      rawTexts: Array.isArray(ocr.raw_texts) ? ocr.raw_texts : [],
      ocrAmount: Number(ocr.amount ?? 0),
      ocrDate: ocr.date ?? '',
      ocrMerchant: ocr.merchant_name ?? '',
      form: nextForm,
    })
  }

  function buildDrafts(suggestions: LLMSuggestion[], slot: ImageSlot): SlotDraft[] {
    return suggestions.map((sug, index) => {
      const categoryId = sug.category_id ?? 0
      const matched = categoryId > 0 ? meta.categories.find((c) => c.id === categoryId) : undefined
      const categoryName = matched?.name || sug.category_hint || (categoryId > 0 ? '' : '未匹配')
      const merchant = meta.merchants.find((m) => m.name === sug.merchant_name)
      return {
        id: `draft_${Date.now()}_${index}_${slot.id}`,
        type: sug.type === 'income' ? 'income' : 'expense',
        amount: Number(sug.amount ?? 0),
        categoryId,
        categoryName: categoryName || '未匹配',
        merchantId: merchant?.id ?? 0,
        merchantName: sug.merchant_name ?? '',
        date: suggestionDate(sug.date, slot.ocrDate),
        note: sug.note ?? '',
        imagePath: slot.imagePath,
        llmIndex: index,
      }
    })
  }

  function applyLlmResult(
    id: string,
    suggestions: LLMSuggestion[],
    error: string,
  ): void {
    const slot = findSlot(id)
    if (!slot) return
    setSlot(id, {
      status: 'done',
      llm: suggestions,
      llmError: error || '',
    })
    if (suggestions.length > 1) {
      setDraftList(id, buildDrafts(suggestions, slot))
      toast(`AI 识别出 ${suggestions.length} 笔，已生成草稿，请确认后提交`)
    }
  }

  async function processSlot(id: string): Promise<void> {
    const slot = findSlot(id)
    if (!slot) return
    const raw = imagesMap.get(id)
    if (!raw) {
      setSlot(id, { status: 'error', error: '图片已失效，请重新选择' })
      return
    }

    try {
      setSlot(id, { status: 'compressing', error: '' })
      let image = raw
      try {
        const compressed = await compressImage(raw)
        if (compressed.path && compressed.path !== raw.path) {
          revokePreview(raw.path)
          image = compressed
          imagesMap.set(id, compressed)
          setSlot(id, { previewUrl: compressed.path })
        }
      } catch (e) {
        console.warn('[image-slots] 压缩失败，使用原图', e)
      }

      if (!findSlot(id)) return
      const hash = await sha256(image)
      setSlot(id, { status: 'ocr_recognizing' })
      const ocr = await ocrApi.upload({
        filePath: image.path,
        file: image.file,
        hash,
      })

      if (!findSlot(id)) {
        imagesMap.delete(id)
        return
      }
      patchFormByOcr(id, ocr)
      imagesMap.delete(id)

      if (llmEnabled.value && (ocr.ocr_id ?? 0) > 0) {
        setSlot(id, { status: 'llm_recognizing' })
        try {
          await Promise.allSettled([meta.ensureCategories(), meta.ensureMerchants()])
          const result = await llmApi.analyzeByOcrId({ ocr_id: ocr.ocr_id })
          const suggestions = Array.isArray(result?.suggestions) ? result.suggestions : []
          applyLlmResult(id, suggestions, result?.error ?? '')
        } catch (err) {
          const msg = errMessage(err, 'AI 分析失败')
          setSlot(id, { status: 'done', llmError: msg })
          toast(`AI 分析失败：${msg}`)
        }
      } else {
        setSlot(id, { status: 'done' })
      }
    } catch (err) {
      const msg = errMessage(err, '识别失败')
      setSlot(id, { status: 'error', error: msg })
      toast(msg)
    }
  }

  async function processQueue(): Promise<void> {
    if (processing) return
    processing = true
    const current = generation
    try {
      while (queue.length > 0 && current === generation) {
        const id = queue[0]
        try {
          await processSlot(id)
        } finally {
          if (current === generation) queue.shift()
        }
      }
    } finally {
      if (current === generation) processing = false
    }
  }

  async function addImages(images: PickedImage[]): Promise<void> {
    if (!images.length) return
    const remain = Math.max(0, MAX_IMAGE_COUNT - slots.value.length)
    if (remain <= 0) {
      toast(`最多上传 ${MAX_IMAGE_COUNT} 张图片`)
      return
    }
    const accepted = images.slice(0, remain)
    const startIndex = slots.value.length
    const created: ImageSlot[] = []

    accepted.forEach((image) => {
      const id = newSlotId()
      imagesMap.set(id, image)
      const slot: ImageSlot = {
        id,
        previewUrl: image.path,
        status: 'pending',
        ocrId: 0,
        imagePath: '',
        rawTexts: [],
        ocrAmount: 0,
        ocrDate: '',
        ocrMerchant: '',
        saved: false,
        form: defaultSlotForm(),
        llm: [],
        llmError: '',
      }
      slots.value.push(slot)
      created.push(slot)
      queue.push(id)
    })

    if (created.length > 0) {
      activeIndex.value = startIndex
      void processQueue()
    }
    if (accepted.length < images.length) {
      toast(`最多上传 ${MAX_IMAGE_COUNT} 张图片，已忽略多余图片`)
    }
  }

  /**
   * 选图后先过小管家内容安全检测。
   * 只有检测通过的图片才会进入 slots / 渲染预览，未通过的图片全程不展示。
   */
  async function filterBySecCheck(images: PickedImage[]): Promise<PickedImage[]> {
    if (!images.length || !isSecCheckAvailable()) return images

    const passed: PickedImage[] = []
    let rejected = 0
    let unavailable = 0

    for (let i = 0; i < images.length; i++) {
      const image = images[i]
      const progress = images.length > 1 ? `处理中 ${i + 1}/${images.length}` : '处理中'
      uni.showLoading({ title: progress, mask: true })
      let result: Awaited<ReturnType<typeof checkImage>>
      try {
        result = await checkImage(image.path, () => {
          uni.showLoading({ title: progress, mask: true })
        })
      } finally {
        uni.hideLoading()
      }

      if (result.ok) {
        passed.push(image)
      } else {
        console.warn('[seccheck] 图片未通过', result.status, result.reason || '')
        if (result.status === 'failed') unavailable += 1
        else rejected += 1
      }
    }

    if (rejected > 0) {
      toast('图片检测不通过！请更换图片！')
    } else if (unavailable > 0) {
      toast('图片检测暂不可用，请稍后重试')
    }
    return passed
  }

  async function chooseAndAdd(): Promise<void> {
    const remain = MAX_IMAGE_COUNT - slots.value.length
    if (remain <= 0) {
      toast(`最多上传 ${MAX_IMAGE_COUNT} 张图片`)
      return
    }
    try {
      const images = await pickImages(remain)
      const checked = await filterBySecCheck(images)
      if (!checked.length) return
      await addImages(checked)
    } catch (err) {
      if (err instanceof Error && err.message.includes('取消')) return
      toast(err instanceof Error ? err.message : '选择图片失败')
    }
  }

  function retry(slotId: string): void {
    const slot = findSlot(slotId)
    if (!slot) return
    if (!imagesMap.has(slotId)) {
      toast('图片已失效，请重新选择')
      return
    }
    setSlot(slotId, { status: 'pending', error: '' })
    if (!queue.includes(slotId)) queue.push(slotId)
    void processQueue()
  }

  function retryLlm(slotId: string): void {
    const slot = findSlot(slotId)
    if (!slot || !slot.ocrId || slot.status !== 'done') return
    setSlot(slotId, { status: 'llm_recognizing', llmError: '' })
    void (async () => {
      try {
        await Promise.allSettled([meta.ensureCategories(), meta.ensureMerchants()])
        const result = await llmApi.analyzeByOcrId({ ocr_id: slot.ocrId })
        applyLlmResult(slotId, result.suggestions ?? [], result.error ?? '')
      } catch (err) {
        const msg = errMessage(err, 'AI 分析失败')
        setSlot(slotId, { status: 'done', llmError: msg })
        toast(`AI 分析失败：${msg}`)
      }
    })()
  }

  function removeSlot(slotId: string): void {
    const index = slots.value.findIndex((s) => s.id === slotId)
    if (index < 0) return
    const slot = slots.value[index]
    revokePreview(slot.previewUrl)
    imagesMap.delete(slotId)
    const queueIndex = queue.indexOf(slotId)
    if (queueIndex >= 0) queue.splice(queueIndex, 1)
    slots.value = slots.value.filter((s) => s.id !== slotId)
    setDraftList(slotId, [])
    if (slots.value.length === 0) {
      activeIndex.value = 0
    } else if (index < activeIndex.value) {
      activeIndex.value -= 1
    } else if (activeIndex.value >= slots.value.length) {
      activeIndex.value = slots.value.length - 1
    }
  }

  function select(index: number): void {
    if (index < 0 || index >= slots.value.length) return
    activeIndex.value = index
  }

  function patchForm(patch: Partial<SlotForm>): void {
    const slot = activeSlot.value
    if (slot) {
      setSlot(slot.id, { form: { ...slot.form, ...patch } })
    } else {
      noSlotForm.value = { ...noSlotForm.value, ...patch }
    }
  }

  function addDraft(draft: SlotDraft): void {
    const slot = activeSlot.value
    if (slot) {
      setDraftList(slot.id, [...(draftMap.value[slot.id] ?? []), draft])
    } else {
      noSlotDrafts.value = [...noSlotDrafts.value, draft]
    }
  }

  function removeDraft(index: number): void {
    const slot = activeSlot.value
    if (slot) {
      const list = (draftMap.value[slot.id] ?? []).filter((_, i) => i !== index)
      setDraftList(slot.id, list)
    } else {
      noSlotDrafts.value = noSlotDrafts.value.filter((_, i) => i !== index)
    }
    patchForm({ amount: '', note: '', merchantId: 0, merchantName: '' })
  }

  function updateDraft(index: number, draft: SlotDraft): void {
    const slot = activeSlot.value
    if (slot) {
      const list = (draftMap.value[slot.id] ?? []).map((item, i) => (i === index ? draft : item))
      setDraftList(slot.id, list)
    } else {
      noSlotDrafts.value = noSlotDrafts.value.map((item, i) => (i === index ? draft : item))
    }
  }

  function setDrafts(list: SlotDraft[]): void {
    const slot = activeSlot.value
    if (slot) setDraftList(slot.id, list)
    else noSlotDrafts.value = list
  }

  function applySuggestion(sug: LLMSuggestion, index: number): void {
    const slot = activeSlot.value
    if (!slot) return
    const patch: Partial<SlotForm> = {}
    if (sug.type === 'income' || sug.type === 'expense') patch.type = sug.type
    if ((sug.amount ?? 0) > 0) patch.amount = String(sug.amount)
    if (sug.date) patch.date = suggestionDate(sug.date, slot.ocrDate)
    if (sug.merchant_name) {
      const matched = meta.merchants.find((m) => m.name === sug.merchant_name)
      patch.merchantName = sug.merchant_name
      patch.merchantId = matched?.id ?? 0
    }
    if (sug.note) patch.note = sug.note

    const type = patch.type ?? slot.form.type
    if (sug.category_id > 0) {
      patch.categoryId = sug.category_id
    } else if (sug.category_hint) {
      const matched = meta.categories.find(
        (c) => c.type === type && c.name.includes(sug.category_hint),
      )
      if (matched) patch.categoryId = matched.id
    }

    patchForm(patch)

    if (!drafts.value.some((d) => d.llmIndex === index)) {
      const merged = { ...slot.form, ...patch }
      const category = meta.categories.find((c) => c.id === merged.categoryId)
      addDraft({
        id: `draft_${Date.now()}_${index}_${slot.id}`,
        type: merged.type,
        amount: Number(merged.amount || 0),
        categoryId: merged.categoryId,
        categoryName:
          category?.name ||
          sug.category_hint ||
          (merged.categoryId > 0 ? '' : '未匹配') ||
          '未匹配',
        merchantId: merged.merchantId,
        merchantName: merged.merchantName,
        date: merged.date,
        note: merged.note,
        imagePath: slot.imagePath,
        llmIndex: index,
      })
    }
    toast('已填入表单并暂存')
  }

  function markSaved(slotId: string): void {
    setSlot(slotId, { saved: true })
  }

  function reset(): void {
    generation += 1
    slots.value.forEach((slot) => revokePreview(slot.previewUrl))
    slots.value = []
    draftMap.value = {}
    noSlotDrafts.value = []
    noSlotForm.value = defaultSlotForm()
    activeIndex.value = 0
    llmEnabled.value = false
    imagesMap.clear()
    queue.splice(0, queue.length)
    processing = false
  }

  onUnmounted(() => {
    reset()
  })

  return {
    slots,
    activeIndex,
    activeSlot,
    drafts,
    isRecognizing,
    llmEnabled,
    totalDrafts,
    form,
    noSlotForm,
    loadLlmConfig,
    chooseAndAdd,
    addImages,
    retry,
    retryLlm,
    removeSlot,
    select,
    patchForm,
    applySuggestion,
    addDraft,
    removeDraft,
    updateDraft,
    setDrafts,
    markSaved,
    reset,
  }
}
