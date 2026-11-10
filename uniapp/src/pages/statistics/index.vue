<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { onShow } from '@dcloudio/uni-app'
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'
import ChartBar from '@/components/charts/ChartBar.vue'
import ChartPie from '@/components/charts/ChartPie.vue'
import { CHART_COLORS } from '@/components/charts/useChartCanvas'
import { statisticsApi } from '@/api/statistics'
import type {
  CategoryStat,
  MerchantStat,
  MerchantStatsResp,
  MonthlyStatistics,
  RangeStatistics,
  YearlyStatistics,
} from '@/api/types'
import { ensureSessionReady, useAuthGuard } from '@/composables/useAuthGuard'
import { daysInMonth } from '@/utils/datetime'
import { money, percent } from '@/utils/format'
import { goMerchantDetail } from '@/utils/nav'
import { toast, toastError } from '@/utils/ui'

useAuthGuard()

type ViewMode = 'monthly' | 'yearly' | 'range'

const VIEW_OPTIONS: { key: ViewMode; label: string }[] = [
  { key: 'monthly', label: '月视图' },
  { key: 'yearly', label: '年视图' },
  { key: 'range', label: '日期范围' },
]

const view = ref<ViewMode>('monthly')
const year = ref(dayjs().year())
const month = ref(dayjs().month() + 1)
const yearOnly = ref(dayjs().year())
const startDate = ref(dayjs().startOf('month').format('YYYY-MM-DD'))
const endDate = ref(dayjs().format('YYYY-MM-DD'))

const ready = ref(false)
const loading = ref(false)
const errorText = ref('')
const refreshing = ref(false)
let requestId = 0

interface ViewState<T> {
  key: string
  data: T | null
}

const monthlyState = reactive<ViewState<MonthlyStatistics>>({ key: '', data: null })
const yearlyState = reactive<ViewState<YearlyStatistics>>({ key: '', data: null })
const rangeState = reactive<ViewState<RangeStatistics>>({ key: '', data: null })

const monthlyKey = computed(() => `${year.value}-${month.value}`)
const yearlyKey = computed(() => `${yearOnly.value}`)
const rangeKey = computed(() => `${startDate.value}~${endDate.value}`)

const currentKey = computed(() => {
  if (view.value === 'monthly') return monthlyKey.value
  if (view.value === 'yearly') return yearlyKey.value
  return rangeKey.value
})

const currentData = computed<
  MonthlyStatistics | YearlyStatistics | RangeStatistics | null
>(() => {
  if (view.value === 'monthly') {
    return monthlyState.key === monthlyKey.value ? monthlyState.data : null
  }
  if (view.value === 'yearly') {
    return yearlyState.key === yearlyKey.value ? yearlyState.data : null
  }
  return rangeState.key === rangeKey.value ? rangeState.data : null
})

const stateForView = computed(() => {
  if (view.value === 'monthly') return monthlyState
  if (view.value === 'yearly') return yearlyState
  return rangeState
})

const totalIncome = computed(() => currentData.value?.total?.total_income ?? 0)
const totalExpense = computed(() => currentData.value?.total?.total_expense ?? 0)
const netAmount = computed(
  () => currentData.value?.total?.net_amount ?? totalIncome.value - totalExpense.value,
)
const categories = computed<CategoryStat[]>(() => currentData.value?.categories ?? [])
const merchants = computed<MerchantStat[]>(() => currentData.value?.top_merchants ?? [])

const maxMerchantTotal = computed(() => merchants.value[0]?.total ?? 1)

const expensePie = computed(() =>
  categories.value
    .filter((item) => item.type === 'expense')
    .map((item, index) => ({
      name: item.category_name,
      value: item.total,
      icon: item.category_icon,
      color: CHART_COLORS[index % CHART_COLORS.length],
    })),
)

const incomePie = computed(() =>
  categories.value
    .filter((item) => item.type === 'income')
    .map((item, index) => ({
      name: item.category_name,
      value: item.total,
      icon: item.category_icon,
      color: CHART_COLORS[index % CHART_COLORS.length],
    })),
)

const expenseTotal = computed(() => expensePie.value.reduce((sum, item) => sum + item.value, 0))
const incomeTotal = computed(() => incomePie.value.reduce((sum, item) => sum + item.value, 0))

const barCategories = computed<string[]>(() => {
  const data = currentData.value
  if (!data) return []
  if (view.value === 'monthly' && 'daily' in data) {
    return (data.daily ?? []).map((item) => dayjs(item.date).format('DD'))
  }
  if (view.value === 'yearly' && 'monthly' in data) {
    return (data.monthly ?? []).map((item) => `${item.month}月`)
  }
  if ('daily' in data) {
    return (data.daily ?? []).map((item) => dayjs(item.date).format('MM-DD'))
  }
  return []
})

const barSeries = computed(() => {
  const data = currentData.value
  if (!data) {
    return [
      { name: '收入', data: [], color: '#22c55e' },
      { name: '支出', data: [], color: '#ef4444' },
    ]
  }
  if (view.value === 'monthly' && 'daily' in data) {
    return [
      { name: '收入', data: (data.daily ?? []).map((item) => item.total_income ?? 0), color: '#22c55e' },
      { name: '支出', data: (data.daily ?? []).map((item) => item.total_expense ?? 0), color: '#ef4444' },
    ]
  }
  if (view.value === 'yearly' && 'monthly' in data) {
    return [
      { name: '收入', data: (data.monthly ?? []).map((item) => item.total_income ?? 0), color: '#22c55e' },
      { name: '支出', data: (data.monthly ?? []).map((item) => item.total_expense ?? 0), color: '#ef4444' },
    ]
  }
  if ('daily' in data) {
    return [
      { name: '收入', data: (data.daily ?? []).map((item) => item.total_income ?? 0), color: '#22c55e' },
      { name: '支出', data: (data.daily ?? []).map((item) => item.total_expense ?? 0), color: '#ef4444' },
    ]
  }
  return [
    { name: '收入', data: [], color: '#22c55e' },
    { name: '支出', data: [], color: '#ef4444' },
  ]
})

const barTitle = computed(() => {
  if (view.value === 'yearly') return '每月收支'
  return '每日收支'
})

const isCurrentMonth = computed(
  () => year.value === dayjs().year() && month.value === dayjs().month() + 1,
)
const isCurrentYear = computed(() => yearOnly.value === dayjs().year())

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

function setView(value: ViewMode): void {
  view.value = value
}

function prevMonth(): void {
  if (month.value === 1) {
    year.value -= 1
    month.value = 12
  } else {
    month.value -= 1
  }
}

function nextMonth(): void {
  if (isCurrentMonth.value) return
  if (month.value === 12) {
    year.value += 1
    month.value = 1
  } else {
    month.value += 1
  }
}

function prevYear(): void {
  yearOnly.value -= 1
}

function nextYear(): void {
  if (isCurrentYear.value) return
  yearOnly.value += 1
}

function pickerValue(e: Event): string {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  return detail?.value === undefined ? '' : String(detail.value)
}

function onStartDate(e: Event): void {
  startDate.value = pickerValue(e)
  if (endDate.value && endDate.value < startDate.value) endDate.value = startDate.value
}

function onEndDate(e: Event): void {
  endDate.value = pickerValue(e)
  if (startDate.value && endDate.value < startDate.value) startDate.value = endDate.value
}

async function load(): Promise<void> {
  const id = ++requestId
  loading.value = true
  errorText.value = ''
  try {
    if (view.value === 'monthly') {
      const data = await statisticsApi.monthly(year.value, month.value)
      if (id !== requestId) return
      monthlyState.key = monthlyKey.value
      monthlyState.data = data
    } else if (view.value === 'yearly') {
      const data = await statisticsApi.yearly(yearOnly.value)
      if (id !== requestId) return
      yearlyState.key = yearlyKey.value
      yearlyState.data = data
    } else {
      const data = await statisticsApi.range(startDate.value, endDate.value)
      if (id !== requestId) return
      rangeState.key = rangeKey.value
      rangeState.data = data
    }
  } catch (err) {
    if (id === requestId) {
      errorText.value = err instanceof Error ? err.message : '加载统计数据失败'
    }
  } finally {
    if (id === requestId) loading.value = false
  }
}

watch([view, year, month, yearOnly, startDate, endDate], () => {
  if (ready.value) void load()
})

onShow(() => {
  void (async () => {
    if (!(await ensureSessionReady())) return
    ready.value = true
    void load()
  })()
})

function currentDateRange(): { start: string; end: string } {
  if (view.value === 'monthly') {
    const lastDay = daysInMonth(year.value, month.value)
    return {
      start: `${year.value}-${pad(month.value)}-01`,
      end: `${year.value}-${pad(month.value)}-${pad(lastDay)}`,
    }
  }
  if (view.value === 'yearly') {
    return { start: `${yearOnly.value}-01-01`, end: `${yearOnly.value}-12-31` }
  }
  return { start: startDate.value, end: endDate.value }
}

function onMerchantTap(merchant: MerchantStat): void {
  const range = currentDateRange()
  goMerchantDetail({
    id: merchant.merchant_id,
    name: merchant.merchant_name,
    start: range.start,
    end: range.end,
  })
}

async function refreshMerchants(): Promise<void> {
  if (refreshing.value) return
  refreshing.value = true
  try {
    let result: MerchantStatsResp
    if (view.value === 'monthly') {
      result = await statisticsApi.refreshMerchants({
        view: 'monthly',
        year: year.value,
        month: month.value,
      })
    } else if (view.value === 'yearly') {
      result = await statisticsApi.refreshMerchants({ view: 'yearly', year: yearOnly.value })
    } else {
      result = await statisticsApi.refreshMerchants({
        view: 'range',
        start: startDate.value,
        end: endDate.value,
      })
    }
    const state = stateForView.value
    if (state.key === currentKey.value && state.data) {
      state.data = { ...state.data, top_merchants: result.top_merchants ?? [] }
    }
    toast('商户统计已刷新', 'success')
  } catch (err) {
    toastError(err, '刷新失败')
  } finally {
    refreshing.value = false
  }
}

function merchantBarColor(index: number): string {
  return CHART_COLORS[index % CHART_COLORS.length]
}
</script>

<template>
  <AppPage title="统计分析" tabbar="statistics">
    <view class="stats">
      <!-- 视图切换 -->
      <view class="seg">
        <view
          v-for="option in VIEW_OPTIONS"
          :key="option.key"
          class="seg__item"
          :class="{ 'seg__item--active': view === option.key }"
          @click="setView(option.key)"
        >
          {{ option.label }}
        </view>
      </view>

      <!-- 期间选择 -->
      <view class="period">
        <template v-if="view === 'monthly'">
          <view class="period__arrow" @click="prevMonth">‹</view>
          <text class="period__label">{{ year }}年{{ pad(month) }}月</text>
          <view class="period__arrow" :class="{ 'period__arrow--disabled': isCurrentMonth }" @click="nextMonth">
            ›
          </view>
        </template>
        <template v-else-if="view === 'yearly'">
          <view class="period__arrow" @click="prevYear">‹</view>
          <text class="period__label">{{ yearOnly }}年</text>
          <view class="period__arrow" :class="{ 'period__arrow--disabled': isCurrentYear }" @click="nextYear">
            ›
          </view>
        </template>
        <template v-else>
          <picker mode="date" :value="startDate" :end="endDate" @change="onStartDate">
            <view class="period__date">{{ startDate || '开始日期' }}</view>
          </picker>
          <text class="period__sep">至</text>
          <picker mode="date" :value="endDate" :start="startDate" :end="dayjs().format('YYYY-MM-DD')" @change="onEndDate">
            <view class="period__date">{{ endDate || '结束日期' }}</view>
          </picker>
        </template>
        <text v-if="loading && currentData" class="period__loading">更新中…</text>
      </view>

      <view v-if="errorText" class="stats__error" @click="load">
        <text>{{ errorText }}</text>
        <text class="stats__error-retry">点击重试</text>
      </view>

      <!-- 加载骨架 -->
      <view v-if="!currentData && loading" class="stats__skeleton">
        <view class="skeleton-row">
          <view class="skeleton-card" />
          <view class="skeleton-card" />
          <view class="skeleton-card" />
        </view>
        <view class="skeleton-block" />
      </view>

      <EmptyState
        v-else-if="!currentData && !loading"
        icon="📊"
        text="暂无统计数据"
        hint="换个时间范围试试，或先记录几笔交易"
      >
        <view class="btn btn-primary btn-sm" @click="load">重新加载</view>
      </EmptyState>

      <template v-else-if="currentData">
        <!-- 总额 -->
        <view class="total-grid">
          <view class="total-card">
            <text class="total-card__label">收入</text>
            <text class="total-card__value total-card__value--income num">
              ¥{{ money(totalIncome) }}
            </text>
          </view>
          <view class="total-card">
            <text class="total-card__label">支出</text>
            <text class="total-card__value total-card__value--expense num">
              ¥{{ money(totalExpense) }}
            </text>
          </view>
          <view class="total-card total-card--net">
            <text class="total-card__label">净结余</text>
            <text
              class="total-card__value num"
              :class="netAmount >= 0 ? 'total-card__value--net-in' : 'total-card__value--net-out'"
            >
              {{ netAmount >= 0 ? '+' : '-' }}¥{{ money(Math.abs(netAmount)) }}
            </text>
          </view>
        </view>

        <!-- 柱状图 -->
        <view class="card">
          <view class="card__head">
            <text class="card__title">{{ barTitle }}</text>
            <text class="card__sub">共 {{ barCategories.length }} 个数据点</text>
          </view>
          <EmptyState
            v-if="barCategories.length === 0"
            icon="📊"
            :text="view === 'yearly' ? '本年暂无数据' : '所选范围暂无数据'"
          />
          <ChartBar
            v-else
            :categories="barCategories"
            :series="barSeries"
            :height="220"
          />
        </view>

        <!-- 分类占比 -->
        <view class="card">
          <view class="card__head">
            <text class="card__title">支出分类</text>
            <text class="card__sub">共 ¥{{ money(expenseTotal) }}</text>
          </view>
          <EmptyState v-if="expensePie.length === 0" icon="🍽️" text="暂无支出分类数据" />
          <template v-else>
            <ChartPie :items="expensePie" :height="200" />
            <view class="legend">
              <view v-for="item in expensePie" :key="item.name" class="legend__row">
                <view class="legend__name-wrap">
                  <view class="legend__dot" :style="{ backgroundColor: item.color }" />
                  <text class="legend__name">{{ item.icon }} {{ item.name }}</text>
                </view>
                <view class="legend__values">
                  <text class="legend__pct">{{ percent(item.value, expenseTotal) }}</text>
                  <text class="legend__value num">¥{{ money(item.value) }}</text>
                </view>
              </view>
            </view>
          </template>
        </view>

        <view class="card">
          <view class="card__head">
            <text class="card__title">收入分类</text>
            <text class="card__sub">共 ¥{{ money(incomeTotal) }}</text>
          </view>
          <EmptyState v-if="incomePie.length === 0" icon="💰" text="暂无收入分类数据" />
          <template v-else>
            <ChartPie :items="incomePie" :height="200" />
            <view class="legend">
              <view v-for="item in incomePie" :key="item.name" class="legend__row">
                <view class="legend__name-wrap">
                  <view class="legend__dot" :style="{ backgroundColor: item.color }" />
                  <text class="legend__name">{{ item.icon }} {{ item.name }}</text>
                </view>
                <view class="legend__values">
                  <text class="legend__pct">{{ percent(item.value, incomeTotal) }}</text>
                  <text class="legend__value num">¥{{ money(item.value) }}</text>
                </view>
              </view>
            </view>
          </template>
        </view>

        <!-- 商户 Top10 -->
        <view class="card">
          <view class="card__head">
            <text class="card__title">商户支出 Top10</text>
            <view
              class="refresh-btn"
              :class="{ 'refresh-btn--loading': refreshing }"
              @click="refreshMerchants"
            >
              ↻ {{ refreshing ? '刷新中' : '刷新' }}
            </view>
          </view>

          <EmptyState v-if="merchants.length === 0" icon="🏪" text="暂无商户支出数据" />
          <view v-else class="merchant-list">
            <view
              v-for="(merchant, index) in merchants"
              :key="merchant.merchant_id"
              class="merchant"
              @click="onMerchantTap(merchant)"
            >
              <view
                class="merchant__rank"
                :class="{
                  'merchant__rank--1': index === 0,
                  'merchant__rank--2': index === 1,
                  'merchant__rank--3': index === 2,
                }"
              >
                {{ index + 1 }}
              </view>
              <view class="merchant__body">
                <view class="merchant__line">
                  <text class="merchant__name">{{ merchant.merchant_name }}</text>
                  <view class="merchant__values">
                    <text class="merchant__count">{{ merchant.tx_count }}笔</text>
                    <text class="merchant__amount num">¥{{ money(merchant.total) }}</text>
                  </view>
                </view>
                <view class="merchant__bar">
                  <view
                    class="merchant__bar-fill"
                    :style="{
                      width: `${Math.max(4, Math.round((merchant.total / (maxMerchantTotal || 1)) * 100))}%`,
                      backgroundColor: merchantBarColor(index),
                    }"
                  />
                </view>
              </view>
            </view>
          </view>
        </view>
      </template>
    </view>
  </AppPage>
</template>

<style lang="scss" scoped>
.stats {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.seg {
  display: flex;
  padding: 6rpx;
  border-radius: $radius-sm;
  background-color: #eef2f7;
}

.seg__item {
  flex: 1;
  padding: 14rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-2;
  border-radius: 10rpx;
}

.seg__item--active {
  background-color: #fff;
  color: $c-primary;
  font-weight: 600;
  box-shadow: 0 2rpx 8rpx rgba(17, 24, 39, 0.08);
}

.period {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20rpx;
}

.period__arrow {
  width: 58rpx;
  height: 58rpx;
  border-radius: 50%;
  border: 2rpx solid $c-border-2;
  background-color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32rpx;
  color: $c-text-2;
}

.period__arrow--disabled {
  opacity: 0.3;
}

.period__label {
  min-width: 220rpx;
  text-align: center;
  font-size: $fs;
  font-weight: 600;
  color: $c-text;
}

.period__date {
  min-width: 220rpx;
  height: 68rpx;
  padding: 0 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
  font-size: $fs-sm;
  color: $c-text;
}

.period__sep {
  font-size: $fs-xs;
  color: $c-text-3;
}

.period__loading {
  font-size: 20rpx;
  color: $c-text-3;
}

.stats__error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18rpx 22rpx;
  border-radius: $radius-sm;
  background-color: $c-expense-bg;
  color: $c-expense;
  font-size: $fs-xs;
}

.stats__error-retry {
  color: $c-primary;
}

.stats__skeleton {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.skeleton-row {
  display: flex;
  gap: 14rpx;
}

.skeleton-card {
  flex: 1;
  height: 120rpx;
  border-radius: $radius;
  background-color: #eef2f7;
  animation: pulse 1.2s ease-in-out infinite;
}

.skeleton-block {
  height: 420rpx;
  border-radius: $radius;
  background-color: #eef2f7;
  animation: pulse 1.2s ease-in-out infinite;
}

.total-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14rpx;
}

.total-card {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
  padding: 22rpx 18rpx;
  border-radius: $radius;
  background-color: $c-card;
  border: 2rpx solid $c-border;
  box-shadow: $shadow-card;
}

.total-card--net {
  background: linear-gradient(135deg, #eff6ff, #ffffff);
}

.total-card__label {
  font-size: 20rpx;
  color: $c-text-3;
}

.total-card__value {
  font-size: 30rpx;
  font-weight: 700;
}

.total-card__value--income {
  color: $c-income;
}

.total-card__value--expense {
  color: $c-expense;
}

.total-card__value--net-in {
  color: $c-primary;
}

.total-card__value--net-out {
  color: $c-warn;
}

.card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  margin-bottom: 18rpx;
}

.card__title {
  font-size: $fs;
  font-weight: 600;
  color: $c-text;
}

.card__sub {
  font-size: 20rpx;
  color: $c-text-3;
}

.legend {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
  margin-top: 8rpx;
}

.legend__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.legend__name-wrap {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}

.legend__dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 50%;
  flex-shrink: 0;
}

.legend__name {
  font-size: $fs-sm;
  color: $c-text-2;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.legend__values {
  display: flex;
  align-items: center;
  gap: 14rpx;
  flex-shrink: 0;
}

.legend__pct {
  font-size: 20rpx;
  color: $c-text-3;
}

.legend__value {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.refresh-btn {
  padding: 8rpx 18rpx;
  border-radius: 999rpx;
  background-color: $c-primary-bg;
  color: $c-primary;
  font-size: 20rpx;
}

.refresh-btn--loading {
  opacity: 0.6;
}

.merchant-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.merchant {
  display: flex;
  align-items: center;
  gap: 14rpx;
}

.merchant__rank {
  width: 36rpx;
  height: 36rpx;
  border-radius: 50%;
  flex-shrink: 0;
  background-color: #f3f4f6;
  color: $c-text-3;
  font-size: 20rpx;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.merchant__rank--1 {
  background-color: #ef4444;
  color: #fff;
}

.merchant__rank--2 {
  background-color: #fb923c;
  color: #fff;
}

.merchant__rank--3 {
  background-color: #facc15;
  color: #fff;
}

.merchant__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.merchant__line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.merchant__name {
  font-size: $fs-sm;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.merchant__values {
  display: flex;
  align-items: center;
  gap: 12rpx;
  flex-shrink: 0;
}

.merchant__count {
  font-size: 20rpx;
  color: $c-text-3;
}

.merchant__amount {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-expense;
}

.merchant__bar {
  height: 10rpx;
  border-radius: 999rpx;
  background-color: #f3f4f6;
  overflow: hidden;
}

.merchant__bar-fill {
  height: 100%;
  border-radius: 999rpx;
  transition: width 0.4s ease;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.55;
  }
}
</style>
