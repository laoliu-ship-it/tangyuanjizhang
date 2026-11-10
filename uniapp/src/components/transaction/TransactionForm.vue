<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import dayjs from 'dayjs'
import AppToast from '@/components/AppToast.vue'
import AppModal from '@/components/AppModal.vue'
import SlotEditor from './SlotEditor.vue'
import ImageSlotStrip from './ImageSlotStrip.vue'
import LlmSuggestionList from './LlmSuggestionList.vue'
import OcrTextPanel from './OcrTextPanel.vue'
import DraftList from './DraftList.vue'
import ReviewSheet from './ReviewSheet.vue'
import { useImageSlots, defaultSlotForm, type ImageSlot, type SlotDraft, type SlotForm } from '@/composables/useImageSlots'
import { useMetaStore } from '@/store/meta'
import { useTenantStore } from '@/store/tenant'
import { tenantApi } from '@/api/tenant'
import { transactionApi } from '@/api/transaction'
import type { LLMSuggestion, Transaction, TransactionCreatePayload } from '@/api/types'
import { resolveUpload } from '@/config'
import { extractDatePart, extractTimeFromOcrDate, fromApiDateTime, toApiDateTime } from '@/utils/datetime'
import { money } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    visible: boolean
    mode?: 'create' | 'edit' | 'detail'
    initialData?: Transaction
  }>(),
  { mode: 'create', initialData: undefined },
)

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const visibleModel = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const isCreate = computed(() => props.mode === 'create')
const isEdit = computed(() => props.mode === 'edit')
const isDetail = computed(() => props.mode === 'detail')

const meta = useMetaStore()
const tenantStore = useTenantStore()

const {
  slots,
  activeIndex,
  activeSlot,
  drafts,
  isRecognizing,
  form,
  loadLlmConfig,
  chooseAndAdd,
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
} = useImageSlots()

const categories = computed(() => meta.categories)
const merchants = computed(() => meta.merchants)

const requireExpenseImage = ref(true)
const metaLoading = ref(false)
const metaError = ref('')
const submitting = ref(false)
const toastState = ref<{ text: string; error: boolean } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | null = null

const editForm = ref<SlotForm>(defaultSlotForm())
const editImagePath = ref('')
const detailImages = ref<string[]>([])

const reviewVisible = ref(false)
const draftEditorVisible = ref(false)
const editingDraftIndex = ref(-1)
const editingDraftForm = ref<SlotForm | null>(null)

const systemInfo = uni.getSystemInfoSync()
const statusBarHeight = Number(systemInfo.statusBarHeight ?? 0)
const headerPaddingTop = statusBarHeight > 0 ? `${statusBarHeight}px` : 'env(safe-area-inset-top)'
const capsulePadding = ref(0)
// #ifdef MP-WEIXIN
try {
  const rect = uni.getMenuButtonBoundingClientRect()
  if (rect && rect.left) {
    capsulePadding.value = Math.max(12, Number(systemInfo.windowWidth) - rect.left + 8)
  }
} catch (e) {
  capsulePadding.value = 0
}
// #endif

const displayForm = computed<SlotForm>(() =>
  isCreate.value ? form.value : editForm.value,
)

const title = computed(() => {
  if (isDetail.value) return '交易详情'
  if (isEdit.value) return '编辑记账'
  if (slots.value.length > 1) return `新增记账（${activeIndex.value + 1}/${slots.value.length}）`
  return '新增记账'
})

const savedCount = computed(() => slots.value.filter((slot) => slot.saved).length)

const hasOcrInfo = computed(() => {
  const slot = activeSlot.value
  return !!slot && (slot.ocrAmount > 0 || !!slot.ocrDate || !!slot.ocrMerchant)
})

const appliedLlmIndices = computed<number[]>(() =>
  drafts.value
    .map((draft) => draft.llmIndex)
    .filter((index): index is number => index !== undefined),
)

const usedLines = computed<number[]>(() => {
  const slot = activeSlot.value
  if (!slot) return []
  const lines = new Set<number>()
  appliedLlmIndices.value.forEach((index) => {
    const suggestion = slot.llm[index]
    suggestion?.source_lines?.forEach((line) => lines.add(line))
  })
  return Array.from(lines)
})

/* ------------------------------ 初始化 ------------------------------ */

function showToast(text: string, error = false): void {
  toastState.value = { text, error }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastState.value = null
  }, 3500)
}

async function loadMeta(force = false): Promise<void> {
  metaLoading.value = true
  metaError.value = ''
  try {
    await Promise.all([meta.ensureCategories(force), meta.ensureMerchants(force)])
  } catch (err) {
    metaError.value = err instanceof Error ? err.message : '加载分类和商户失败'
  } finally {
    metaLoading.value = false
  }
}

async function loadSettings(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId) return
  try {
    const settings = await tenantApi.getSettings(tenantId)
    requireExpenseImage.value = settings?.require_expense_image ?? true
  } catch (e) {
    requireExpenseImage.value = true
  }
}

function populateFromTransaction(): void {
  const data = props.initialData
  if (!data) return
  editForm.value = {
    type: data.type,
    amount: String(data.amount ?? ''),
    categoryId: data.category_id ?? 0,
    merchantId: data.merchant_id ?? 0,
    merchantName: data.merchant_name ?? '',
    date: fromApiDateTime(data.transaction_date),
    note: data.note ?? '',
  }
  const first = data.images?.[0]
  editImagePath.value = first?.image_path ?? ''
  detailImages.value = (data.images ?? [])
    .map((image) => resolveUpload(image.image_path))
    .filter(Boolean)
}

function init(): void {
  toastState.value = null
  reviewVisible.value = false
  draftEditorVisible.value = false
  editingDraftIndex.value = -1
  editingDraftForm.value = null
  submitting.value = false
  metaError.value = ''
  void loadMeta()
  if (isCreate.value) {
    reset()
    void loadLlmConfig()
    void loadSettings()
  } else {
    populateFromTransaction()
  }
}

function cleanup(): void {
  reviewVisible.value = false
  draftEditorVisible.value = false
  toastState.value = null
  if (isCreate.value) reset()
}

watch(
  () => props.visible,
  (value) => {
    if (value) init()
    else cleanup()
  },
)

onUnmounted(() => {
  if (toastTimer) clearTimeout(toastTimer)
})

/* ------------------------------ 交互 ------------------------------ */

function close(): void {
  if (submitting.value) return
  visibleModel.value = false
}

function onEditPatch(patch: Partial<SlotForm>): void {
  editForm.value = { ...editForm.value, ...patch }
}

function onDraftPatch(patch: Partial<SlotForm>): void {
  if (!editingDraftForm.value) return
  editingDraftForm.value = { ...editingDraftForm.value, ...patch }
}

function previewActive(): void {
  const url = activeSlot.value?.previewUrl
  if (!url) return
  uni.previewImage({ urls: [url], current: url })
}

function previewEdit(): void {
  const url = resolveUpload(editImagePath.value)
  if (!url) return
  uni.previewImage({ urls: [url], current: url })
}

function previewDetail(index: number): void {
  const urls = detailImages.value
  if (!urls.length) return
  uni.previewImage({ urls, current: urls[index] ?? urls[0] })
}

function onRetry(slot: ImageSlot): void {
  retry(slot.id)
}

/* --------------------------- 表单校验 / 草稿 --------------------------- */

function validateForm(value: SlotForm, imagePath: string): string {
  if (!value.amount || Number(value.amount) <= 0) return '请输入有效金额'
  if (!value.categoryId) return '请选择分类'
  if (requireExpenseImage.value && value.type === 'expense' && !imagePath) {
    return '支出记录需要上传截图'
  }
  return ''
}

function buildDraftFromForm(): SlotDraft | null {
  const slot = activeSlot.value
  const value = form.value
  const imagePath = slot?.imagePath ?? ''
  const message = validateForm(value, imagePath)
  if (message) {
    showToast(message, true)
    return null
  }
  const category = categories.value.find((item) => item.id === value.categoryId)
  return {
    id: `draft_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
    type: value.type,
    amount: Number(value.amount),
    categoryId: value.categoryId,
    categoryName: category?.name ?? '',
    merchantId: value.merchantId,
    merchantName: value.merchantName,
    date: value.date,
    note: value.note,
    imagePath,
  }
}

function handleAddDraft(): void {
  if (isRecognizing.value) {
    showToast('正在识别，请稍候')
    return
  }
  const draft = buildDraftFromForm()
  if (!draft) return
  addDraft(draft)
  patchForm({ amount: '', note: '', merchantId: 0, merchantName: '' })
  showToast('已加入暂存，可继续记账')
}

function handleComplete(): void {
  if (isRecognizing.value) {
    showToast('正在识别，请稍候')
    return
  }
  if (drafts.value.length === 0) {
    const draft = buildDraftFromForm()
    if (!draft) return
    setDrafts([draft])
  }
  reviewVisible.value = true
}

/* ----------------------------- 草稿编辑 ----------------------------- */

function openDraftEditor(index: number): void {
  const draft = drafts.value[index]
  if (!draft) return
  editingDraftIndex.value = index
  editingDraftForm.value = {
    type: draft.type,
    amount: String(draft.amount),
    categoryId: draft.categoryId,
    merchantId: draft.merchantId,
    merchantName: draft.merchantName,
    date: draft.date,
    note: draft.note,
  }
  draftEditorVisible.value = true
}

function saveDraftEditor(): void {
  const value = editingDraftForm.value
  const index = editingDraftIndex.value
  if (!value || index < 0) return
  if (!value.amount || Number(value.amount) <= 0) {
    showToast('请输入有效金额', true)
    return
  }
  if (!value.categoryId) {
    showToast('请选择分类', true)
    return
  }
  const original = drafts.value[index]
  if (!original) return
  const category = categories.value.find((item) => item.id === value.categoryId)
  updateDraft(index, {
    ...original,
    type: value.type,
    amount: Number(value.amount),
    categoryId: value.categoryId,
    categoryName: category?.name ?? '',
    merchantId: value.merchantId,
    merchantName: value.merchantName,
    date: value.date,
    note: value.note,
  })
  draftEditorVisible.value = false
  editingDraftForm.value = null
  editingDraftIndex.value = -1
  showToast('草稿已更新')
}

function onReviewEdit(index: number): void {
  reviewVisible.value = false
  openDraftEditor(index)
}

function onReviewRemove(index: number): void {
  removeDraft(index)
  if (drafts.value.length === 0) reviewVisible.value = false
}

/* ------------------------------ 提交 ------------------------------ */

function buildPayload(draft: SlotDraft, slot?: ImageSlot): TransactionCreatePayload {
  return {
    type: draft.type,
    amount: draft.amount,
    category_id: draft.categoryId,
    merchant_id: draft.merchantId || undefined,
    merchant_name: draft.merchantName || undefined,
    transaction_date: toApiDateTime(draft.date),
    note: draft.note,
    image_path: draft.imagePath || undefined,
    ocr_amount: slot?.ocrAmount ?? 0,
    ocr_date: slot?.ocrDate ?? '',
    ocr_merchant: slot?.ocrMerchant ?? '',
    ocr_raw_texts: slot && slot.rawTexts.length ? JSON.stringify(slot.rawTexts) : '',
  }
}

async function handleReviewConfirm(): Promise<void> {
  if (submitting.value) return
  const list = drafts.value
  if (!list.length) return

  for (let i = 0; i < list.length; i++) {
    const draft = list[i]
    const label = list.length > 1 ? `第${i + 1}笔` : '该笔'
    if (!draft.categoryId || draft.categoryId <= 0) {
      showToast(`${label}记录「${draft.categoryName || '未分类'}」未匹配有效分类，请编辑后再提交`, true)
      return
    }
    if (requireExpenseImage.value && draft.type === 'expense' && !draft.imagePath) {
      showToast(`${label}支出记录需要上传截图`, true)
      return
    }
  }

  submitting.value = true
  const slot = activeSlot.value
  try {
    for (const draft of list) {
      await transactionApi.create(buildPayload(draft, slot))
    }
    emit('success')
    showToast(`成功提交 ${list.length} 笔交易`)
    reviewVisible.value = false

    if (slot) {
      markSaved(slot.id)
      // 防止用户回到已保存槽位重复提交：清掉该槽位的草稿与表单金额
      setDrafts([])
      patchForm({ amount: '', note: '', merchantId: 0, merchantName: '' })
      const nextIndex = slots.value.findIndex(
        (item, index) => index !== activeIndex.value && !item.saved,
      )
      setTimeout(() => {
        if (nextIndex >= 0) {
          select(nextIndex)
          toastState.value = null
        } else {
          visibleModel.value = false
        }
      }, 900)
    } else {
      setDrafts([])
      setTimeout(() => {
        visibleModel.value = false
      }, 900)
    }
  } catch (err) {
    showToast(err instanceof Error ? err.message : '保存失败，请稍后重试', true)
  } finally {
    submitting.value = false
  }
}

async function handleEditSave(): Promise<void> {
  if (submitting.value || !props.initialData) return
  const value = editForm.value
  const message = validateForm(value, editImagePath.value)
  if (message) {
    showToast(message, true)
    return
  }
  submitting.value = true
  try {
    const image = props.initialData.images?.[0]
    const payload: TransactionCreatePayload = {
      type: value.type,
      amount: Number(value.amount),
      category_id: value.categoryId,
      merchant_id: value.merchantId || undefined,
      merchant_name: value.merchantName || undefined,
      transaction_date: toApiDateTime(value.date),
      note: value.note,
      image_path: editImagePath.value || undefined,
      ocr_amount: image?.ocr_amount ?? 0,
      ocr_date: image?.ocr_date ?? '',
      ocr_merchant: image?.ocr_merchant ?? '',
      ocr_raw_texts: image?.ocr_raw_texts ?? '',
    }
    await transactionApi.update(props.initialData.id, payload)
    emit('success')
    showToast('保存成功')
    setTimeout(() => {
      visibleModel.value = false
    }, 700)
  } catch (err) {
    showToast(err instanceof Error ? err.message : '保存失败，请稍后重试', true)
  } finally {
    submitting.value = false
  }
}

/* --------------------------- OCR 文本填入 --------------------------- */

function fillOcrSummary(): void {
  const slot = activeSlot.value
  if (!slot) return
  const patch: Partial<SlotForm> = {}
  if (slot.ocrAmount > 0) patch.amount = String(slot.ocrAmount)
  if (slot.ocrDate) {
    patch.date = `${extractDatePart(slot.ocrDate)}T${extractTimeFromOcrDate(slot.ocrDate)}`
  }
  if (slot.ocrMerchant) {
    patch.merchantName = slot.ocrMerchant
    const matched = merchants.value.find((item) => item.name === slot.ocrMerchant)
    if (matched) patch.merchantId = matched.id
  }
  patchForm(patch)
  showToast('已填入 OCR 结果')
}

type FillAction = 'amount' | 'merchant' | 'date' | 'note'

function fillFromOcrText(payload: { action: FillAction; text: string }): void {
  const text = payload.text
  if (payload.action === 'amount') {
    const value = parseFloat(text.replace(/[^\d.]/g, ''))
    if (!Number.isNaN(value) && value > 0) patchForm({ amount: String(value) })
    return
  }
  if (payload.action === 'merchant') {
    const matched = merchants.value.find((item) => item.name === text)
    patchForm({ merchantName: matched?.name ?? text, merchantId: matched?.id ?? 0 })
    return
  }
  if (payload.action === 'note') {
    const current = form.value.note
    patchForm({ note: current ? `${current}、${text}` : text })
    return
  }

  // 日期
  const full = text.match(
    /(\d{4})[年\-/](\d{1,2})[月\-/](\d{1,2})[日]?\s*(\d{1,2}):(\d{2})/,
  )
  if (full) {
    patchForm({
      date: `${full[1]}-${full[2].padStart(2, '0')}-${full[3].padStart(2, '0')}T${full[4].padStart(2, '0')}:${full[5]}`,
    })
    return
  }
  const dayOnly = text.match(/(\d{4})[年\-/](\d{1,2})[月\-/](\d{1,2})/)
  if (dayOnly) {
    patchForm({
      date: `${dayOnly[1]}-${dayOnly[2].padStart(2, '0')}-${dayOnly[3].padStart(2, '0')}T${dayjs().format('HH:mm')}`,
    })
    return
  }
  const monthDayTime = text.match(/(\d{1,2})[月\-/](\d{1,2})[日]?\s*(\d{1,2}):(\d{2})/)
  if (monthDayTime) {
    patchForm({
      date: `${dayjs().year()}-${monthDayTime[1].padStart(2, '0')}-${monthDayTime[2].padStart(2, '0')}T${monthDayTime[3].padStart(2, '0')}:${monthDayTime[4]}`,
    })
    return
  }
  const monthDay = text.match(/(\d{1,2})[月\-/](\d{1,2})/)
  if (monthDay) {
    patchForm({
      date: `${dayjs().year()}-${monthDay[1].padStart(2, '0')}-${monthDay[2].padStart(2, '0')}T${dayjs().format('HH:mm')}`,
    })
    return
  }
  const timeOnly = text.match(/^(\d{1,2}):(\d{2})/)
  if (timeOnly) {
    const currentDate = form.value.date.slice(0, 10) || dayjs().format('YYYY-MM-DD')
    patchForm({ date: `${currentDate}T${timeOnly[1].padStart(2, '0')}:${timeOnly[2]}` })
  }
}

function onApplySuggestion(suggestion: LLMSuggestion, index: number): void {
  applySuggestion(suggestion, index)
}
</script>

<template>
  <view v-if="visible" class="txn">
    <!-- 顶栏 -->
    <view class="txn__head" :style="{ paddingTop: headerPaddingTop }">
      <view
        class="txn__bar"
        :style="{ paddingRight: capsulePadding ? `${capsulePadding}px` : '24rpx' }"
      >
        <view class="txn__close" @click="close">✕</view>
        <text class="txn__title">{{ title }}</text>
        <text v-if="isCreate && slots.length > 1" class="txn__progress">
          {{ savedCount }}/{{ slots.length }}
        </text>
        <view v-else class="txn__placeholder" />
      </view>
    </view>

    <scroll-view scroll-y class="txn__body">
      <view class="txn__content">
        <AppToast :text="toastState?.text ?? ''" :error="toastState?.error" />

        <!-- 详情模式 -->
        <template v-if="isDetail">
          <view v-if="detailImages.length" class="detail-images">
            <image
              v-for="(url, index) in detailImages"
              :key="url"
              class="detail-images__item"
              :src="url"
              mode="aspectFill"
              @click="previewDetail(index)"
            />
          </view>
          <view class="card">
            <SlotEditor :form="displayForm" :categories="categories" readonly />
          </view>
          <view class="txn__tip">图片可点击全屏查看</view>
        </template>

        <!-- 编辑模式 -->
        <template v-else-if="isEdit">
          <view v-if="editImagePath" class="image-preview" @click="previewEdit">
            <image class="image-preview__img" :src="resolveUpload(editImagePath)" mode="aspectFit" />
          </view>
          <view class="card">
            <SlotEditor
              :form="displayForm"
              :categories="categories"
              :merchants="merchants"
              :disabled="submitting"
              @patch="onEditPatch"
            />
          </view>
        </template>

        <!-- 新增模式 -->
        <template v-else>
          <view class="create-strip">
            <ImageSlotStrip
              :slots="slots"
              :active-index="activeIndex"
              :max="9"
              @select="select"
              @remove="removeSlot"
              @retry="onRetry"
              @add="chooseAndAdd"
            />
          </view>

          <view v-if="activeSlot?.previewUrl" class="image-preview" @click="previewActive">
            <image class="image-preview__img" :src="activeSlot?.previewUrl" mode="aspectFit" />
            <view v-if="isRecognizing" class="image-preview__overlay">
              <view class="spinner" />
              <text class="image-preview__overlay-text">
                {{
                  activeSlot?.status === 'llm_recognizing'
                    ? 'AI 分析中…'
                    : activeSlot?.status === 'compressing'
                      ? '图片处理中…'
                      : 'OCR 识别中…'
                }}
              </text>
            </view>
            <view
              v-else-if="activeSlot?.status === 'error'"
              class="image-preview__overlay image-preview__overlay--error"
              @click.stop="activeSlot && retry(activeSlot.id)"
            >
              <text class="image-preview__retry">↻ 重试识别</text>
              <text class="image-preview__error-text">{{ activeSlot?.error }}</text>
            </view>
            <view v-if="activeSlot?.saved" class="image-preview__saved">✓ 已保存</view>
          </view>

          <view v-else class="upload-empty" @click="chooseAndAdd">
            <text class="upload-empty__icon">📷</text>
            <text class="upload-empty__title">上传票据截图，自动识别</text>
            <text class="upload-empty__hint">
              支持拍照 / 相册，可一次上传多张；不传图也可手动记一笔
            </text>
          </view>

          <view v-if="slots.length > 1" class="strip-progress">
            已保存 {{ savedCount }} / {{ slots.length }} 张
          </view>

          <!-- OCR 摘要 -->
          <view v-if="hasOcrInfo" class="card ocr-summary">
            <view class="ocr-summary__head">
              <text class="ocr-summary__title">OCR 识别结果</text>
              <text class="ocr-summary__fill" @click="fillOcrSummary">一键填入</text>
            </view>
            <view class="ocr-summary__rows">
              <text v-if="(activeSlot?.ocrAmount ?? 0) > 0" class="ocr-summary__row">
                💰 ¥{{ money(activeSlot?.ocrAmount ?? 0) }}
              </text>
              <text v-if="activeSlot?.ocrDate" class="ocr-summary__row">
                📅 {{ activeSlot?.ocrDate }}
              </text>
              <text v-if="activeSlot?.ocrMerchant" class="ocr-summary__row">
                🏪 {{ activeSlot?.ocrMerchant }}
              </text>
            </view>
          </view>

          <!-- AI 建议 -->
          <view
            v-if="
              activeSlot &&
              ((activeSlot?.llm.length ?? 0) > 0 ||
                !!activeSlot?.llmError ||
                activeSlot?.status === 'llm_recognizing')
            "
            class="card"
          >
            <LlmSuggestionList
              :suggestions="activeSlot?.llm ?? []"
              :error="activeSlot?.llmError ?? ''"
              :loading="activeSlot?.status === 'llm_recognizing'"
              :applied-indices="appliedLlmIndices"
              @apply="onApplySuggestion"
            />
            <view
              v-if="
                activeSlot?.status === 'done' &&
                !!activeSlot?.llmError &&
                (activeSlot?.llm.length ?? 0) === 0
              "
              class="retry-ai"
              @click="activeSlot && retryLlm(activeSlot.id)"
            >
              ↻ 重试 AI 分析
            </view>
          </view>

          <!-- OCR 原文 -->
          <view v-if="activeSlot?.rawTexts.length" class="card">
            <OcrTextPanel
              :raw-texts="activeSlot?.rawTexts ?? []"
              :used-lines="usedLines"
              @fill="fillFromOcrText"
            />
          </view>

          <!-- 表单 -->
          <view class="card">
            <view v-if="metaLoading && !categories.length" class="meta-state">加载分类中…</view>
            <view v-else-if="metaError" class="meta-state">
              <text>{{ metaError }}</text>
              <text class="meta-state__retry" @click="loadMeta(true)">重试</text>
            </view>
            <SlotEditor
              v-else
              :form="form"
              :categories="categories"
              :merchants="merchants"
              :disabled="isRecognizing || submitting"
              @patch="patchForm"
            />
          </view>

          <!-- 暂存草稿 -->
          <view v-if="drafts.length" class="card">
            <DraftList
              :drafts="drafts"
              :categories="categories"
              @edit="openDraftEditor"
              @remove="removeDraft"
            />
          </view>
        </template>
      </view>
    </scroll-view>

    <!-- 底部操作栏 -->
    <view class="txn__footer">
      <view v-if="isDetail" class="btn btn-primary btn-block" @click="close">关闭</view>
      <view
        v-else-if="isEdit"
        class="btn btn-primary btn-block"
        :class="{ 'btn-disabled': submitting }"
        @click="handleEditSave"
      >
        {{ submitting ? '保存中…' : '保存' }}
      </view>
      <view v-else class="txn__footer-btns">
        <view
          class="btn txn__footer-btn txn__footer-btn--secondary"
          :class="{ 'btn-disabled': isRecognizing || submitting }"
          @click="handleAddDraft"
        >
          记一笔
        </view>
        <view
          class="btn btn-primary txn__footer-btn"
          :class="{ 'btn-disabled': isRecognizing || submitting }"
          @click="handleComplete"
        >
          {{ drafts.length ? `完成 ${drafts.length} 笔` : '完成' }}
        </view>
      </view>
    </view>

    <!-- 提交确认 -->
    <ReviewSheet
      v-model:visible="reviewVisible"
      :drafts="drafts"
      :categories="categories"
      :submitting="submitting"
      @confirm="handleReviewConfirm"
      @edit="onReviewEdit"
      @remove="onReviewRemove"
    />

    <!-- 草稿编辑 -->
    <AppModal
      v-model:visible="draftEditorVisible"
      title="编辑草稿"
      position="center"
      :close-on-mask="false"
    >
      <view v-if="editingDraftForm" class="draft-editor">
        <SlotEditor
          :form="editingDraftForm"
          :categories="categories"
          :merchants="merchants"
          @patch="onDraftPatch"
        />
      </view>
      <template #footer>
        <view class="draft-editor__footer">
          <view class="btn draft-editor__btn" @click="draftEditorVisible = false">取消</view>
          <view class="btn btn-primary draft-editor__btn" @click="saveDraftEditor">保存</view>
        </view>
      </template>
    </AppModal>
  </view>
</template>

<style lang="scss" scoped>
.txn {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 120;
  display: flex;
  flex-direction: column;
  background-color: $c-bg;
}

.txn__head {
  flex-shrink: 0;
  background-color: $c-card;
  border-bottom: 2rpx solid $c-border;
}

.txn__bar {
  display: flex;
  align-items: center;
  height: 96rpx;
  padding-left: 24rpx;
  gap: 12rpx;
}

.txn__close {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 32rpx;
  color: $c-text-2;
}

.txn__close:active {
  background-color: $c-border;
}

.txn__title {
  flex: 1;
  min-width: 0;
  font-size: $fs-lg;
  font-weight: 600;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.txn__progress {
  flex-shrink: 0;
  font-size: $fs-xs;
  color: $c-text-3;
}

.txn__placeholder {
  width: 64rpx;
  flex-shrink: 0;
}

.txn__body {
  flex: 1;
  height: 0;
  min-height: 0;
}

.txn__content {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: 24rpx;
  padding-bottom: 40rpx;
}

.create-strip {
  padding: 4rpx 0;
}

.image-preview {
  position: relative;
  width: 100%;
  height: 380rpx;
  border-radius: $radius;
  overflow: hidden;
  background-color: #111827;
}

.image-preview__img {
  width: 100%;
  height: 100%;
}

.image-preview__overlay {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14rpx;
  background-color: rgba(17, 24, 39, 0.48);
}

.image-preview__overlay--error {
  background-color: rgba(17, 24, 39, 0.62);
}

.image-preview__overlay-text {
  font-size: $fs-sm;
  color: #fff;
}

.image-preview__retry {
  padding: 14rpx 28rpx;
  border-radius: 999rpx;
  background-color: #fff;
  color: $c-text;
  font-size: $fs-sm;
  font-weight: 600;
}

.image-preview__error-text {
  max-width: 80%;
  font-size: $fs-xs;
  color: #fecaca;
  text-align: center;
}

.image-preview__saved {
  position: absolute;
  top: 16rpx;
  left: 16rpx;
  padding: 6rpx 18rpx;
  border-radius: 999rpx;
  background-color: $c-income;
  color: #fff;
  font-size: $fs-xs;
}

.spinner {
  width: 44rpx;
  height: 44rpx;
  border: 4rpx solid #fff;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.upload-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
  padding: 48rpx 32rpx;
  border: 3rpx dashed $c-border-2;
  border-radius: $radius;
  background-color: #fff;
}

.upload-empty__icon {
  font-size: 64rpx;
  line-height: 1;
}

.upload-empty__title {
  font-size: $fs;
  font-weight: 500;
  color: $c-text;
}

.upload-empty__hint {
  font-size: $fs-xs;
  color: $c-text-3;
  text-align: center;
  line-height: 1.5;
}

.strip-progress {
  text-align: center;
  font-size: $fs-xs;
  color: $c-text-3;
}

.ocr-summary__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14rpx;
}

.ocr-summary__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.ocr-summary__fill {
  font-size: $fs-xs;
  color: $c-primary;
  padding: 4rpx 8rpx;
}

.ocr-summary__rows {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.ocr-summary__row {
  font-size: $fs-sm;
  color: $c-text-2;
}

.retry-ai {
  margin-top: 16rpx;
  padding: 16rpx;
  text-align: center;
  font-size: $fs-sm;
  color: $c-purple;
  background-color: $c-purple-bg;
  border-radius: $radius-sm;
}

.meta-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  padding: 40rpx 0;
  font-size: $fs-sm;
  color: $c-text-3;
}

.meta-state__retry {
  color: $c-primary;
}

.detail-images {
  display: flex;
  gap: 14rpx;
  overflow-x: auto;
}

.detail-images__item {
  width: 180rpx;
  height: 220rpx;
  flex-shrink: 0;
  border-radius: $radius-sm;
  background-color: $c-border;
}

.txn__tip {
  text-align: center;
  font-size: $fs-xs;
  color: $c-text-3;
}

.txn__footer {
  flex-shrink: 0;
  padding: 20rpx 24rpx;
  padding-bottom: calc(20rpx + env(safe-area-inset-bottom));
  background-color: $c-card;
  border-top: 2rpx solid $c-border;
}

.txn__footer-btns {
  display: flex;
  gap: 16rpx;
}

.txn__footer-btn {
  flex: 1;
  height: 88rpx;
}

.txn__footer-btn--secondary {
  background-color: #fff;
  border: 2rpx solid #bfdbfe;
  color: $c-primary;
}

.draft-editor {
  padding-bottom: 8rpx;
}

.draft-editor__footer {
  display: flex;
  gap: 16rpx;
}

.draft-editor__btn {
  flex: 1;
  height: 84rpx;
}

@media (min-width: 960px) {
  .txn {
    width: 750px;
    margin: 0 auto;
    box-shadow: 0 0 60rpx rgba(17, 24, 39, 0.16);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
