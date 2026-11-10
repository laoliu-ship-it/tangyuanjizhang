<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AppModal from '@/components/AppModal.vue'
import { excelApi, type ExcelFile } from '@/api/excel'
import type { ColumnMapping, ParseHeadersResult } from '@/api/types'
import { revokePreview } from '@/utils/image'

const props = defineProps<{
  visible: boolean
  file: ExcelFile | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'imported'): void
}>()

const visibleModel = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

type Step = 'mapping' | 'preview' | 'done'

interface PreviewState {
  valid_count: number
  skipped_count: number
  issues: string[]
}

interface DoneState {
  imported: number
  skipped_count: number
  issues: string[]
  failed?: boolean
}

const FIELD_DEFS: { key: keyof ColumnMapping; label: string; required: boolean }[] = [
  { key: 'date', label: '日期', required: true },
  { key: 'type', label: '类型', required: true },
  { key: 'amount', label: '金额', required: true },
  { key: 'category', label: '分类', required: true },
  { key: 'merchant', label: '商户', required: false },
  { key: 'note', label: '备注', required: false },
]

function emptyMapping(): ColumnMapping {
  return { date: -1, type: -1, amount: -1, category: -1, merchant: -1, note: -1 }
}

const step = ref<Step>('mapping')
const parseResult = ref<ParseHeadersResult | null>(null)
const mapping = ref<ColumnMapping>(emptyMapping())
const sheetIndex = ref(0)
const parseLoading = ref(false)
const parseError = ref('')
const previewLoading = ref(false)
const importLoading = ref(false)
const preview = ref<PreviewState | null>(null)
const done = ref<DoneState | null>(null)

const headerOptions = computed(() => {
  const headers = parseResult.value?.headers ?? []
  return ['— 不导入 —', ...headers.map((header, index) => header || `列${index + 1}`)]
})

const canPreview = computed(
  () =>
    mapping.value.date >= 0 &&
    mapping.value.type >= 0 &&
    mapping.value.amount >= 0 &&
    mapping.value.category >= 0,
)

function resetState(): void {
  step.value = 'mapping'
  parseResult.value = null
  mapping.value = emptyMapping()
  sheetIndex.value = 0
  parseLoading.value = false
  parseError.value = ''
  previewLoading.value = false
  importLoading.value = false
  preview.value = null
  done.value = null
}

function errorText(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback
}

async function parse(sheet?: number): Promise<void> {
  const file = props.file
  if (!file) return
  parseLoading.value = true
  parseError.value = ''
  try {
    const result = await excelApi.parseHeaders(file, sheet)
    parseResult.value = result
    sheetIndex.value = result.sheet_index ?? 0
    mapping.value = { ...emptyMapping(), ...result.suggestions }
  } catch (err) {
    parseError.value = errorText(err, '解析文件失败，请检查文件格式')
  } finally {
    parseLoading.value = false
  }
}

function openSheet(index: number): void {
  if (index === sheetIndex.value && parseResult.value) return
  sheetIndex.value = index
  void parse(index)
}

function pickerValue(key: keyof ColumnMapping): number {
  return mapping.value[key] >= 0 ? mapping.value[key] + 1 : 0
}

function onMappingChange(key: keyof ColumnMapping, e: Event): void {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  const index = Number(detail?.value ?? 0) - 1
  mapping.value = { ...mapping.value, [key]: index }
}

function sampleText(key: keyof ColumnMapping): string {
  const result = parseResult.value
  const index = mapping.value[key]
  if (!result || index < 0) return ''
  const values = result.sample_rows
    .slice(0, 2)
    .map((row) => row[index])
    .filter((value) => value !== undefined && value !== null && value !== '')
  return values.length ? values.join(' / ') : '（空）'
}

async function handlePreview(): Promise<void> {
  if (!props.file || !canPreview.value) return
  previewLoading.value = true
  try {
    const result = await excelApi.importExcel(
      props.file,
      mapping.value,
      sheetIndex.value,
      true,
    )
    preview.value = {
      valid_count: result.valid_count ?? 0,
      skipped_count: result.skipped_count ?? 0,
      issues: result.issues ?? [],
    }
    step.value = 'preview'
  } catch (err) {
    parseError.value = errorText(err, '预检失败，请检查文件内容')
  } finally {
    previewLoading.value = false
  }
}

async function handleImport(): Promise<void> {
  if (!props.file) return
  importLoading.value = true
  try {
    const result = await excelApi.importExcel(
      props.file,
      mapping.value,
      sheetIndex.value,
      false,
    )
    done.value = {
      imported: result.imported ?? 0,
      skipped_count: result.skipped_count ?? 0,
      issues: result.issues ?? [],
    }
    step.value = 'done'
    emit('imported')
  } catch (err) {
    done.value = {
      imported: 0,
      skipped_count: 0,
      issues: [errorText(err, '导入失败，请检查文件内容或网络连接')],
      failed: true,
    }
    step.value = 'done'
  } finally {
    importLoading.value = false
  }
}

function goBack(): void {
  if (step.value === 'mapping') {
    visibleModel.value = false
    return
  }
  step.value = 'mapping'
  preview.value = null
  done.value = null
}

function close(): void {
  visibleModel.value = false
}

watch(
  [() => props.visible, () => props.file],
  ([visible]) => {
    if (visible && props.file) {
      resetState()
      void parse(0)
    } else if (!visible) {
      if (props.file) revokePreview(props.file.path)
      resetState()
    }
  },
)
</script>

<template>
  <AppModal v-model:visible="visibleModel" title="Excel 导入" position="bottom">
    <view class="import">
      <!-- 步骤 -->
      <view class="steps">
        <view class="steps__item" :class="{ 'steps__item--active': step === 'mapping', 'steps__item--done': step !== 'mapping' }">
          <text class="steps__dot">{{ step !== 'mapping' ? '✓' : '1' }}</text>
          <text>列映射</text>
        </view>
        <view class="steps__line" />
        <view class="steps__item" :class="{ 'steps__item--active': step === 'preview', 'steps__item--done': step === 'done' }">
          <text class="steps__dot">{{ step === 'done' ? '✓' : '2' }}</text>
          <text>数据预检</text>
        </view>
        <view class="steps__line" />
        <view class="steps__item" :class="{ 'steps__item--active': step === 'done' }">
          <text class="steps__dot">3</text>
          <text>导入结果</text>
        </view>
      </view>

      <view v-if="file" class="import__file">
        <text class="import__file-name">📄 {{ file.name }}</text>
      </view>

      <!-- 步骤 1：列映射 -->
      <template v-if="step === 'mapping'">
        <view v-if="parseResult && parseResult.sheets.length > 1" class="sheets">
          <text class="sheets__label">工作表：</text>
          <view
            v-for="(sheet, index) in parseResult.sheets"
            :key="index"
            class="sheets__item"
            :class="{ 'sheets__item--active': sheetIndex === index }"
            @click="openSheet(index)"
          >
            {{ sheet }}
          </view>
        </view>

        <view v-if="parseLoading" class="import__loading">解析文件中…</view>
        <view v-else-if="parseError" class="import__error">{{ parseError }}</view>

        <template v-else-if="parseResult">
          <text class="import__hint">
            已识别 <text class="import__strong">{{ parseResult.headers.length }}</text> 列，请确认字段映射（
            <text class="import__required">*</text> 为必填）
          </text>

          <view class="mapping">
            <view v-for="field in FIELD_DEFS" :key="field.key" class="mapping__row">
              <view class="mapping__label">
                <text :class="{ 'mapping__label--required': field.required }">{{ field.label }}</text>
                <text v-if="field.required" class="import__required">*</text>
              </view>
              <picker
                mode="selector"
                :range="headerOptions"
                :value="pickerValue(field.key)"
                @change="onMappingChange(field.key, $event)"
              >
                <view
                  class="mapping__picker"
                  :class="{ 'mapping__picker--missing': field.required && mapping[field.key] < 0 }"
                >
                  {{ headerOptions[pickerValue(field.key)] }}
                </view>
              </picker>
              <text class="mapping__sample">{{ sampleText(field.key) }}</text>
            </view>
          </view>

          <view v-if="parseResult.sample_rows.length" class="preview-table-wrap">
            <text class="preview-table__title">数据预览（前 {{ parseResult.sample_rows.length }} 行）</text>
            <scroll-view scroll-x class="preview-table">
              <view class="preview-table__inner">
                <view class="preview-table__row preview-table__row--head">
                  <text
                    v-for="(header, index) in parseResult.headers"
                    :key="index"
                    class="preview-table__cell"
                  >
                    {{ header || `列${index + 1}` }}
                  </text>
                </view>
                <view
                  v-for="(row, rowIndex) in parseResult.sample_rows"
                  :key="rowIndex"
                  class="preview-table__row"
                >
                  <text
                    v-for="(cell, cellIndex) in row"
                    :key="cellIndex"
                    class="preview-table__cell"
                  >
                    {{ cell }}
                  </text>
                </view>
              </view>
            </scroll-view>
          </view>
        </template>
      </template>

      <!-- 步骤 2：预检 -->
      <template v-else-if="step === 'preview' && preview">
        <view class="preview-stats">
          <view class="preview-stat preview-stat--ok">
            <text class="preview-stat__value">{{ preview.valid_count }}</text>
            <text class="preview-stat__label">可导入行数</text>
          </view>
          <view class="preview-stat" :class="{ 'preview-stat--bad': preview.skipped_count > 0 }">
            <text class="preview-stat__value">{{ preview.skipped_count }}</text>
            <text class="preview-stat__label">有问题将跳过</text>
          </view>
        </view>

        <view v-if="preview.issues.length" class="issues">
          <text class="issues__title">
            发现 {{ preview.issues.length }} 处问题（可修改 Excel 后重新上传，或直接跳过这些行继续导入）：
          </text>
          <scroll-view scroll-y class="issues__list">
            <view v-for="(issue, index) in preview.issues" :key="index" class="issues__item">
              {{ index + 1 }}. {{ issue }}
            </view>
          </scroll-view>
        </view>
        <view v-else class="import__ok">数据检查通过，全部 {{ preview.valid_count }} 行均可正常导入 🎉</view>

        <view v-if="preview.valid_count === 0" class="import__error">
          所有行均有问题，没有可导入的数据。请修改 Excel 文件后重新上传。
        </view>
      </template>

      <!-- 步骤 3：结果 -->
      <template v-else-if="step === 'done' && done">
        <view v-if="!done.failed" class="done-ok">
          <text class="done-ok__value">{{ done.imported }}</text>
          <text class="done-ok__label">条记录导入成功</text>
          <text v-if="done.skipped_count > 0" class="done-ok__skip">
            另有 {{ done.skipped_count }} 行因问题已跳过
          </text>
        </view>
        <view v-else class="import__error">
          <text v-for="(issue, index) in done.issues" :key="index">{{ issue }}</text>
        </view>

        <view v-if="!done.failed && done.issues.length" class="issues">
          <text class="issues__title">跳过的行（供参考）：</text>
          <view class="issues__list issues__list--plain">
            <text v-for="(issue, index) in done.issues" :key="index" class="issues__item">
              {{ issue }}
            </text>
          </view>
        </view>
      </template>
    </view>

    <template #footer>
      <view class="import__footer">
        <view class="btn import__btn" @click="step === 'mapping' ? close() : goBack()">
          {{ step === 'mapping' ? '取消' : '← 返回' }}
        </view>
        <view
          v-if="step === 'mapping'"
          class="btn btn-primary import__btn"
          :class="{ 'btn-disabled': !canPreview || parseLoading || previewLoading }"
          @click="handlePreview"
        >
          {{ previewLoading ? '检查中…' : '下一步：预检' }}
        </view>
        <view
          v-else-if="step === 'preview' && preview && preview.valid_count > 0"
          class="btn btn-primary import__btn"
          :class="{ 'btn-disabled': importLoading }"
          @click="handleImport"
        >
          {{ importLoading ? '导入中…' : `确认导入 ${preview.valid_count} 条` }}
        </view>
        <view v-else-if="step === 'done'" class="btn btn-primary import__btn" @click="close">
          完成
        </view>
      </view>
    </template>
  </AppModal>
</template>

<style lang="scss" scoped>
.import {
  display: flex;
  flex-direction: column;
  gap: 22rpx;
}

.steps {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.steps__item {
  display: flex;
  align-items: center;
  gap: 6rpx;
  font-size: 20rpx;
  color: $c-text-3;
}

.steps__item--active {
  color: $c-primary;
  font-weight: 600;
}

.steps__item--done {
  color: $c-income;
}

.steps__dot {
  width: 32rpx;
  height: 32rpx;
  border-radius: 50%;
  background-color: #e5e7eb;
  color: $c-text-3;
  font-size: 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.steps__item--active .steps__dot {
  background-color: $c-primary;
  color: #fff;
}

.steps__item--done .steps__dot {
  background-color: $c-income;
  color: #fff;
}

.steps__line {
  flex: 1;
  height: 2rpx;
  background-color: $c-border-2;
}

.import__file {
  padding: 14rpx 20rpx;
  border-radius: $radius-sm;
  background-color: $c-primary-bg;
}

.import__file-name {
  font-size: $fs-xs;
  color: $c-primary;
}

.import__loading {
  padding: 48rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-3;
}

.import__error {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  padding: 20rpx 24rpx;
  border-radius: $radius-sm;
  background-color: $c-expense-bg;
  color: $c-expense;
  font-size: $fs-sm;
  line-height: 1.5;
}

.import__hint {
  font-size: $fs-xs;
  color: $c-text-2;
  line-height: 1.6;
}

.import__strong {
  color: $c-text;
  font-weight: 600;
}

.import__required {
  color: $c-expense;
  margin-left: 2rpx;
}

.sheets {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10rpx;
}

.sheets__label {
  font-size: $fs-xs;
  color: $c-text-2;
}

.sheets__item {
  padding: 8rpx 18rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 999rpx;
  font-size: $fs-xs;
  color: $c-text-2;
}

.sheets__item--active {
  border-color: $c-primary;
  background-color: $c-primary;
  color: #fff;
}

.mapping {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}

.mapping__row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.mapping__label {
  width: 82rpx;
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  gap: 2rpx;
  font-size: $fs-sm;
  color: $c-text-2;
}

.mapping__label--required {
  color: $c-text;
  font-weight: 500;
}

.mapping__picker {
  width: 250rpx;
  flex-shrink: 0;
  height: 68rpx;
  padding: 0 18rpx;
  display: flex;
  align-items: center;
  border: 2rpx solid $c-border-2;
  border-radius: 10rpx;
  font-size: $fs-xs;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.mapping__picker--missing {
  border-color: #fca5a5;
  background-color: $c-expense-bg;
}

.mapping__sample {
  flex: 1;
  min-width: 0;
  font-size: 20rpx;
  color: $c-text-3;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.preview-table-wrap {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.preview-table__title {
  font-size: 20rpx;
  color: $c-text-3;
}

.preview-table {
  width: 100%;
  border: 2rpx solid $c-border;
  border-radius: $radius-sm;
}

.preview-table__inner {
  display: inline-flex;
  flex-direction: column;
  min-width: 100%;
}

.preview-table__row {
  display: flex;
  border-bottom: 2rpx solid $c-border;
}

.preview-table__row:last-child {
  border-bottom: none;
}

.preview-table__row--head {
  background-color: $c-bg;
}

.preview-table__cell {
  width: 200rpx;
  flex-shrink: 0;
  padding: 12rpx 16rpx;
  font-size: 20rpx;
  color: $c-text-2;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.preview-table__row--head .preview-table__cell {
  color: $c-text;
  font-weight: 500;
}

.preview-stats {
  display: flex;
  gap: 16rpx;
}

.preview-stat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: 24rpx;
  border-radius: $radius-sm;
  background-color: #f3f4f6;
}

.preview-stat--ok {
  background-color: $c-income-bg;
}

.preview-stat--bad {
  background-color: $c-expense-bg;
}

.preview-stat__value {
  font-size: 44rpx;
  font-weight: 700;
  color: $c-text;
}

.preview-stat--ok .preview-stat__value {
  color: $c-income;
}

.preview-stat--bad .preview-stat__value {
  color: $c-expense;
}

.preview-stat__label {
  font-size: $fs-xs;
  color: $c-text-3;
}

.issues {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.issues__title {
  font-size: $fs-xs;
  color: $c-text-2;
  line-height: 1.5;
}

.issues__list {
  max-height: 360rpx;
  padding: 16rpx 20rpx;
  border-radius: $radius-sm;
  background-color: $c-warn-bg;
}

.issues__list--plain {
  background-color: $c-bg;
}

.issues__item {
  display: block;
  font-size: 20rpx;
  color: $c-warn;
  line-height: 1.6;
}

.issues__list--plain .issues__item {
  color: $c-text-3;
}

.import__ok {
  padding: 20rpx;
  border-radius: $radius-sm;
  background-color: $c-income-bg;
  color: $c-income;
  font-size: $fs-sm;
  text-align: center;
}

.done-ok {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6rpx;
  padding: 32rpx;
  border-radius: $radius;
  background-color: $c-income-bg;
}

.done-ok__value {
  font-size: 64rpx;
  font-weight: 700;
  color: $c-income;
}

.done-ok__label {
  font-size: $fs-sm;
  color: $c-income;
}

.done-ok__skip {
  font-size: $fs-xs;
  color: $c-text-3;
}

.import__footer {
  display: flex;
  gap: 16rpx;
}

.import__btn {
  flex: 1;
  height: 84rpx;
}
</style>
