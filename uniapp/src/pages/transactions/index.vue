<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'
import TransactionCard from '@/components/transactions/TransactionCard.vue'
import Pagination from '@/components/transactions/Pagination.vue'
import ImportSheet from '@/components/transactions/ImportSheet.vue'
import TransactionForm from '@/components/transaction/TransactionForm.vue'
import { excelApi, type ExcelFile } from '@/api/excel'
import { transactionApi } from '@/api/transaction'
import type { Transaction } from '@/api/types'
import { ensureSessionReady, useAuthGuard } from '@/composables/useAuthGuard'
import { useRole } from '@/composables/useRole'
import { useMetaStore } from '@/store/meta'
import {
  createEmptyFilter,
  type TransactionFilter,
} from '@/components/transactions/filter'
import { todayStr } from '@/utils/datetime'
import { confirm, haptic, hideLoading, showLoading, toast, toastError } from '@/utils/ui'

useAuthGuard()

const PAGE_SIZE = 20

const meta = useMetaStore()
const { canEdit, load: loadRole } = useRole()

const list = ref<Transaction[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const errorText = ref('')
const filter = ref<TransactionFilter>(createEmptyFilter())
const filterDraft = ref<TransactionFilter>(createEmptyFilter())
const exporting = ref(false)

const selectMode = ref(false)
const selectedIds = ref<number[]>([])

const createVisible = ref(false)
const detailTarget = ref<Transaction | null>(null)
const editTarget = ref<Transaction | null>(null)

const importVisible = ref(false)
const importFile = ref<ExcelFile | null>(null)
const busy = ref(false)
let lastLongPress = 0
let metaLoaded = false

const categories = computed(() => meta.categories)
const merchants = computed(() => meta.merchants)
const finished = computed(() => list.value.length >= total.value)
const allSelected = computed(
  () => list.value.length > 0 && selectedIds.value.length === list.value.length,
)

const detailVisible = computed({
  get: () => detailTarget.value !== null,
  set: (value: boolean) => {
    if (!value) detailTarget.value = null
  },
})

const editVisible = computed({
  get: () => editTarget.value !== null,
  set: (value: boolean) => {
    if (!value) editTarget.value = null
  },
})

const TYPE_OPTIONS: { value: '' | 'income' | 'expense'; label: string }[] = [
  { value: '', label: '全部类型' },
  { value: 'income', label: '收入' },
  { value: 'expense', label: '支出' },
]

const categoryOptions = computed(() => [
  '全部分类',
  ...categories.value.map((category) => `${category.icon || '📌'} ${category.name}`),
])

const typeIndex = computed(() => {
  const index = TYPE_OPTIONS.findIndex((option) => option.value === filterDraft.value.type)
  return index >= 0 ? index : 0
})

const categoryIndex = computed(() => {
  if (!filterDraft.value.categoryId) return 0
  const index = categories.value.findIndex((item) => item.id === filterDraft.value.categoryId)
  return index >= 0 ? index + 1 : 0
})

const merchantSuggestions = computed(() => {
  const keyword = filterDraft.value.merchantName.trim().toLowerCase()
  if (!keyword || filterDraft.value.merchantId) return []
  return merchants.value
    .filter((merchant) => merchant.name.toLowerCase().includes(keyword))
    .slice(0, 20)
})

const hasActiveFilter = computed(() => {
  const value = filter.value
  return !!(value.type || value.categoryId || value.merchantId || value.startDate || value.endDate)
})

const dateLabel = computed(() =>
  filterDraft.value.sortBy === 'created_at' ? '创建时间' : '交易时间',
)

function categoryOf(tx: Transaction) {
  return categories.value.find((item) => item.id === tx.category_id)
}

function eventValue(e: Event): string {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  return detail?.value === undefined ? '' : String(detail.value)
}

function setSortBy(value: 'transaction_date' | 'created_at'): void {
  filterDraft.value.sortBy = value
}

function onTypeChange(e: Event): void {
  const option = TYPE_OPTIONS[Number(eventValue(e))]
  filterDraft.value.type = option?.value ?? ''
}

function onCategoryChange(e: Event): void {
  const index = Number(eventValue(e))
  const category = categories.value[index - 1]
  filterDraft.value.categoryId = category?.id ?? 0
}

function onMerchantInput(e: Event): void {
  filterDraft.value.merchantName = eventValue(e)
  filterDraft.value.merchantId = 0
}

function clearMerchant(): void {
  filterDraft.value.merchantName = ''
  filterDraft.value.merchantId = 0
}

function pickMerchant(merchant: { id: number; name: string }): void {
  filterDraft.value.merchantName = merchant.name
  filterDraft.value.merchantId = merchant.id
}

function onStartDate(e: Event): void {
  filterDraft.value.startDate = eventValue(e)
  if (filterDraft.value.endDate && filterDraft.value.endDate < filterDraft.value.startDate) {
    filterDraft.value.endDate = filterDraft.value.startDate
  }
}

function onEndDate(e: Event): void {
  filterDraft.value.endDate = eventValue(e)
  if (filterDraft.value.startDate && filterDraft.value.endDate < filterDraft.value.startDate) {
    filterDraft.value.startDate = filterDraft.value.endDate
  }
}

function handleQuery(): void {
  filter.value = { ...filterDraft.value }
  reloadFirstPage()
}

function handleReset(): void {
  const empty = createEmptyFilter()
  filter.value = { ...empty }
  filterDraft.value = { ...empty }
  reloadFirstPage()
}

function queryParams(targetPage: number) {
  return {
    page: targetPage,
    page_size: PAGE_SIZE,
    type: filter.value.type || undefined,
    category_id: filter.value.categoryId || undefined,
    merchant_id: filter.value.merchantId || undefined,
    start_date: filter.value.startDate || undefined,
    end_date: filter.value.endDate || undefined,
    sort_by: filter.value.sortBy,
  }
}

async function loadPage(targetPage: number, append: boolean): Promise<void> {
  if (loading.value) return
  loading.value = true
  errorText.value = ''
  if (!append) {
    selectedIds.value = []
    selectMode.value = false
  }
  try {
    const result = await transactionApi.list(queryParams(targetPage))
    const items = Array.isArray(result?.items) ? result.items : []
    list.value = append ? [...list.value, ...items] : items
    total.value = Number(result?.total ?? items.length)
    page.value = targetPage
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载交易列表失败'
  } finally {
    loading.value = false
  }
}

function loadMore(): void {
  if (loading.value || finished.value) return
  void loadPage(page.value + 1, true)
}

function reloadFirstPage(): void {
  void loadPage(1, false)
}

function enterSelection(tx: Transaction): void {
  if (!canEdit.value) return
  haptic()
  selectMode.value = true
  if (!selectedIds.value.includes(tx.id)) selectedIds.value = [tx.id]
}

function onLongPress(tx: Transaction): void {
  lastLongPress = Date.now()
  if (selectMode.value) {
    toggleSelect(tx.id)
    return
  }
  enterSelection(tx)
}

function toggleSelect(id: number): void {
  if (selectedIds.value.includes(id)) {
    selectedIds.value = selectedIds.value.filter((item) => item !== id)
  } else {
    selectedIds.value = [...selectedIds.value, id]
  }
}

function onToggleCard(tx: Transaction): void {
  toggleSelect(tx.id)
}

function selectAll(): void {
  selectedIds.value = allSelected.value ? [] : list.value.map((item) => item.id)
}

function cancelSelection(): void {
  selectMode.value = false
  selectedIds.value = []
}

function openDetail(tx: Transaction): void {
  if (Date.now() - lastLongPress < 500) return
  detailTarget.value = tx
}

function openEdit(tx: Transaction): void {
  editTarget.value = tx
}

function openCreate(): void {
  createVisible.value = true
}

async function removeTransaction(tx: Transaction): Promise<void> {
  const ok = await confirm('此操作不可撤销，确定要删除这条记录吗？', '确认删除')
  if (!ok) return
  try {
    await transactionApi.remove(tx.id)
    toast('已删除', 'success')
    const target = list.value.length === 1 && page.value > 1 ? page.value - 1 : page.value
    await loadPage(target, false)
  } catch (err) {
    toastError(err, '删除失败')
  }
}

async function batchRemove(): Promise<void> {
  if (!selectedIds.value.length) return
  const ok = await confirm(
    `此操作不可撤销，确定要删除选中的 ${selectedIds.value.length} 条记录吗？`,
    '确认批量删除',
  )
  if (!ok) return
  try {
    await transactionApi.batchRemove(selectedIds.value)
    toast('批量删除成功', 'success')
    cancelSelection()
    await loadPage(1, false)
  } catch (err) {
    toastError(err, '批量删除失败')
  }
}

async function downloadTemplate(): Promise<void> {
  if (busy.value) return
  busy.value = true
  showLoading('下载中…')
  try {
    await excelApi.downloadTemplate()
    toast('模板已下载', 'success')
  } catch (err) {
    toastError(err, '下载失败')
  } finally {
    hideLoading()
    busy.value = false
  }
}

async function exportExcel(): Promise<void> {
  if (busy.value) return
  busy.value = true
  exporting.value = true
  showLoading('导出中…')
  try {
    await excelApi.exportExcel({
      start_date: filterDraft.value.startDate || undefined,
      end_date: filterDraft.value.endDate || undefined,
      type: filterDraft.value.type || undefined,
    })
    toast('导出完成', 'success')
  } catch (err) {
    toastError(err, '导出失败')
  } finally {
    hideLoading()
    exporting.value = false
    busy.value = false
  }
}

async function chooseImport(): Promise<void> {
  if (busy.value) return
  busy.value = true
  try {
    const file = await excelApi.chooseExcelFile()
    importFile.value = file
    importVisible.value = true
  } catch (err) {
    if (err instanceof Error && err.message.includes('取消')) return
    toastError(err, '选择文件失败')
  } finally {
    busy.value = false
  }
}

function onImported(): void {
  reloadFirstPage()
}

watch(importVisible, (value) => {
  if (!value) importFile.value = null
})

onShow(() => {
  void (async () => {
    if (!(await ensureSessionReady())) return
    void loadRole()
    if (!metaLoaded) {
      metaLoaded = true
      void meta.ensureCategories().catch(() => {})
      void meta.ensureMerchants().catch(() => {})
    }
    reloadFirstPage()
  })()
})
</script>

<template>
  <AppPage title="交易记录" tabbar="transactions" @scrolltolower="loadMore">
    <template #right>
      <view
        v-if="canEdit"
        class="export-btn"
        :class="{ 'export-btn--busy': busy }"
        @click="exportExcel"
      >
        <text class="export-btn__icon">📤</text>
        <text>{{ exporting ? '导出中…' : '导出' }}</text>
      </view>
    </template>

    <view class="txn-page">
      <!-- 筛选卡片 -->
      <view class="filter-card">
        <view class="filter-head">
          <text class="filter-head__label">时间类型（日期筛选与列表排序）</text>
          <view class="time-switch">
            <view
              class="time-switch__item"
              :class="{ 'time-switch__item--active': filterDraft.sortBy === 'transaction_date' }"
              @click="setSortBy('transaction_date')"
            >
              交易时间
            </view>
            <view
              class="time-switch__item"
              :class="{ 'time-switch__item--active': filterDraft.sortBy === 'created_at' }"
              @click="setSortBy('created_at')"
            >
              创建时间
            </view>
          </view>
        </view>

        <view class="filter-grid">
          <view class="filter-field">
            <text class="filter-field__label">类型</text>
            <picker
              mode="selector"
              :range="TYPE_OPTIONS"
              range-key="label"
              :value="typeIndex"
              @change="onTypeChange"
            >
              <view class="filter-select">
                <text>{{ TYPE_OPTIONS[typeIndex]?.label ?? '全部类型' }}</text>
                <text class="filter-arrow">⌄</text>
              </view>
            </picker>
          </view>

          <view class="filter-field">
            <text class="filter-field__label">分类</text>
            <picker
              mode="selector"
              :range="categoryOptions"
              :value="categoryIndex"
              @change="onCategoryChange"
            >
              <view class="filter-select">
                <text class="filter-select__text">{{ categoryOptions[categoryIndex] ?? '全部分类' }}</text>
                <text class="filter-arrow">⌄</text>
              </view>
            </picker>
          </view>

          <view class="filter-field filter-field--relative">
            <text class="filter-field__label">商户</text>
            <view class="merchant-wrap">
              <input
                class="filter-input"
                :value="filterDraft.merchantName"
                placeholder="搜索商户"
                @input="onMerchantInput"
              />
              <text v-if="filterDraft.merchantName" class="merchant-clear" @click="clearMerchant">
                ×
              </text>
            </view>
            <view v-if="merchantSuggestions.length" class="merchant-suggest">
              <view
                v-for="merchant in merchantSuggestions"
                :key="merchant.id"
                class="merchant-suggest__item"
                @click="pickMerchant(merchant)"
              >
                {{ merchant.name }}
              </view>
            </view>
          </view>

          <view class="filter-field">
            <text class="filter-field__label">
              {{ filterDraft.sortBy === 'created_at' ? '创建日期（起）' : '交易日期（起）' }}
            </text>
            <picker
              mode="date"
              :value="filterDraft.startDate"
              :end="filterDraft.endDate || todayStr()"
              @change="onStartDate"
            >
              <view class="filter-select filter-select--date">
                <text>{{ filterDraft.startDate || '年 / 月 / 日' }}</text>
                <text class="filter-date-icon">📅</text>
              </view>
            </picker>
          </view>

          <view class="filter-field">
            <text class="filter-field__label">
              {{ filterDraft.sortBy === 'created_at' ? '创建日期（止）' : '交易日期（止）' }}
            </text>
            <picker
              mode="date"
              :value="filterDraft.endDate"
              :start="filterDraft.startDate"
              :end="todayStr()"
              @change="onEndDate"
            >
              <view class="filter-select filter-select--date">
                <text>{{ filterDraft.endDate || '年 / 月 / 日' }}</text>
                <text class="filter-date-icon">📅</text>
              </view>
            </picker>
          </view>
        </view>

        <text class="filter-hint">
          日期筛选按「{{ dateLabel }}」，切换上方时间类型后筛选范围会同步生效，需重新点击查询
        </text>

        <view class="filter-actions">
          <view class="btn btn-primary filter-action" @click="handleQuery">查询</view>
          <view class="btn filter-action" @click="handleReset">重置</view>
          <view v-if="canEdit" class="filter-links">
            <text class="filter-link" @click="downloadTemplate">下载模板</text>
            <text class="filter-link-sep">|</text>
            <text class="filter-link" @click="chooseImport">导入 Excel</text>
          </view>
        </view>
      </view>

      <!-- 多选栏 -->
      <view v-if="selectMode" class="select-bar">
        <text class="select-bar__count">已选 {{ selectedIds.length }} 项</text>
        <view class="select-bar__actions">
          <text class="select-bar__action" @click="selectAll">
            {{ allSelected ? '取消全选' : '全选本页' }}
          </text>
          <text class="select-bar__action" @click="cancelSelection">退出多选</text>
        </view>
      </view>

      <view v-if="errorText" class="error-bar" @click="reloadFirstPage">
        <text>{{ errorText }}</text>
        <text class="error-bar__retry">点击重试</text>
      </view>

      <!-- 加载 / 空态 -->
      <view v-if="loading && list.length === 0" class="list-state">
        <view class="list-state__spinner" />
        <text>加载中…</text>
      </view>
      <EmptyState
        v-else-if="!loading && list.length === 0"
        icon="🧾"
        text="暂无交易记录"
        :hint="hasActiveFilter ? '当前筛选条件下没有记录' : '点击右下角 + 开始记一笔'"
      >
        <view v-if="hasActiveFilter" class="btn btn-ghost btn-sm" @click="handleReset">
          重置筛选
        </view>
      </EmptyState>

      <!-- 列表 -->
      <view v-else class="tx-list">
        <view v-for="tx in list" :key="tx.id" class="tx-row">
          <TransactionCard
            :tx="tx"
            :category="categoryOf(tx)"
            :selectable="canEdit"
            :select-mode="selectMode"
            :selected="selectedIds.includes(tx.id)"
            @click="openDetail"
            @longpress="onLongPress"
            @toggle="onToggleCard"
          >
            <template #actions>
              <template v-if="canEdit && !selectMode">
                <text class="tx-action" @click.stop="openEdit(tx)">编辑</text>
                <text class="tx-action tx-action--danger" @click.stop="removeTransaction(tx)">
                  删除
                </text>
              </template>
            </template>
          </TransactionCard>
        </view>
      </view>

      <Pagination
        v-if="list.length > 0"
        :total="total"
        :loading="loading"
        :finished="finished"
        @loadmore="loadMore"
      />
    </view>

    <template #overlay>
      <!-- 批量操作栏 -->
      <view v-if="selectMode && selectedIds.length > 0" class="batch-bar">
        <text class="batch-bar__text">已选 {{ selectedIds.length }} 项</text>
        <view class="btn btn-danger btn-sm" @click="batchRemove">批量删除</view>
      </view>

      <!-- 浮动新增 -->
      <view v-if="canEdit && !selectMode" class="fab" @click="openCreate">＋</view>

      <TransactionForm v-model:visible="createVisible" mode="create" @success="reloadFirstPage" />
      <TransactionForm
        v-model:visible="detailVisible"
        mode="detail"
        :initial-data="detailTarget ?? undefined"
      />
      <TransactionForm
        v-model:visible="editVisible"
        mode="edit"
        :initial-data="editTarget ?? undefined"
        @success="reloadFirstPage"
      />

      <ImportSheet v-model:visible="importVisible" :file="importFile" @imported="onImported" />
    </template>
  </AppPage>
</template>

<style lang="scss" scoped>
.export-btn {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 12rpx 22rpx;
  border-radius: 999rpx;
  background-color: #f3f4f6;
  color: $c-text;
  font-size: $fs-sm;
  font-weight: 500;
}

.export-btn__icon {
  font-size: 24rpx;
}

.export-btn--busy {
  opacity: 0.55;
}

.export-btn:active {
  background-color: $c-border-2;
}

.txn-page {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.filter-card {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: 28rpx;
  border-radius: $radius;
  background-color: $c-card;
  border: 2rpx solid $c-border;
  box-shadow: $shadow-card;
}

.filter-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.filter-head__label {
  flex: 1;
  font-size: $fs-xs;
  color: $c-text-2;
  line-height: 1.5;
}

.time-switch {
  display: flex;
  flex-shrink: 0;
  padding: 5rpx;
  border-radius: 10rpx;
  background-color: #f3f4f6;
}

.time-switch__item {
  padding: 10rpx 18rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  color: $c-text-2;
  white-space: nowrap;
}

.time-switch__item--active {
  background-color: #fff;
  color: $c-primary;
  font-weight: 600;
  box-shadow: 0 2rpx 6rpx rgba(17, 24, 39, 0.08);
}

.filter-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18rpx 16rpx;
}

.filter-field {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.filter-field--relative {
  position: relative;
}

.filter-field__label {
  font-size: 22rpx;
  color: $c-text-3;
}

.filter-select {
  height: 80rpx;
  padding: 0 20rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 12rpx;
  background-color: #fff;
  font-size: $fs-sm;
  color: $c-text;
}

.filter-select__text {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.filter-arrow {
  flex-shrink: 0;
  font-size: 24rpx;
  color: $c-text-3;
}

.filter-select--date {
  justify-content: space-between;
  font-size: 24rpx;
}

.filter-date-icon {
  flex-shrink: 0;
  font-size: 24rpx;
}

.merchant-wrap {
  position: relative;
}

.filter-input {
  width: 100%;
  height: 80rpx;
  padding: 0 50rpx 0 20rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 12rpx;
  background-color: #fff;
  font-size: $fs-sm;
  color: $c-text;
  box-sizing: border-box;
}

.merchant-clear {
  position: absolute;
  top: 50%;
  right: 16rpx;
  transform: translateY(-50%);
  width: 40rpx;
  height: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28rpx;
  color: $c-text-3;
}

.merchant-suggest {
  position: absolute;
  top: 112rpx;
  left: 0;
  right: 0;
  z-index: 30;
  max-height: 340rpx;
  overflow-y: auto;
  border: 2rpx solid $c-border-2;
  border-radius: 12rpx;
  background-color: #fff;
  box-shadow: $shadow-float;
}

.merchant-suggest__item {
  padding: 20rpx 22rpx;
  font-size: $fs-sm;
  color: $c-text;
  border-bottom: 2rpx solid $c-border;
}

.merchant-suggest__item:last-child {
  border-bottom: none;
}

.merchant-suggest__item:active {
  background-color: $c-primary-bg;
  color: $c-primary;
}

.filter-hint {
  font-size: 20rpx;
  color: $c-text-3;
  line-height: 1.6;
}

.filter-actions {
  display: flex;
  align-items: center;
  gap: 16rpx;
  flex-wrap: wrap;
}

.filter-action {
  min-width: 140rpx;
  height: 76rpx;
  padding: 0 28rpx;
  font-size: $fs-sm;
}

.filter-links {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.filter-link {
  font-size: 20rpx;
  color: $c-primary;
  padding: 4rpx;
}

.filter-link-sep {
  font-size: 20rpx;
  color: $c-border-2;
}

.select-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16rpx 22rpx;
  border-radius: $radius-sm;
  background-color: $c-primary-bg;
  border: 2rpx solid #bfdbfe;
}

.select-bar__count {
  font-size: $fs-xs;
  font-weight: 600;
  color: $c-primary;
}

.select-bar__actions {
  display: flex;
  gap: 24rpx;
}

.select-bar__action {
  font-size: $fs-xs;
  color: $c-primary;
}

.error-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18rpx 22rpx;
  border-radius: $radius-sm;
  background-color: $c-expense-bg;
  color: $c-expense;
  font-size: $fs-xs;
}

.error-bar__retry {
  color: $c-primary;
}

.list-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14rpx;
  padding: 80rpx 0;
  font-size: $fs-sm;
  color: $c-text-3;
}

.list-state__spinner {
  width: 30rpx;
  height: 30rpx;
  border: 3rpx solid $c-border-2;
  border-top-color: $c-primary;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.tx-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.tx-action {
  font-size: 20rpx;
  color: $c-primary;
  padding: 4rpx;
}

.tx-action--danger {
  color: $c-expense;
}

.batch-bar {
  position: fixed;
  left: 24rpx;
  right: 24rpx;
  bottom: calc(108rpx + env(safe-area-inset-bottom) + 16rpx);
  z-index: 45;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18rpx 24rpx;
  border-radius: $radius;
  background-color: #fff;
  box-shadow: $shadow-float;
  border: 2rpx solid $c-border;
}

.batch-bar__text {
  font-size: $fs-sm;
  color: $c-text-2;
}

@media (min-width: 960px) {
  .batch-bar {
    left: calc((100vw - 750px) / 2 + 24rpx);
    right: calc((100vw - 750px) / 2 + 24rpx);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
