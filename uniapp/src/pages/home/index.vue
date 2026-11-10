<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import dayjs from 'dayjs'
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'
import ChartLine from '@/components/charts/ChartLine.vue'
import TransactionForm from '@/components/transaction/TransactionForm.vue'
import { statisticsApi } from '@/api/statistics'
import type { DailyStatistics, MonthlyStatistics } from '@/api/types'
import { ensureSessionReady, useAuthGuard } from '@/composables/useAuthGuard'
import { useRole } from '@/composables/useRole'
import { money } from '@/utils/format'

useAuthGuard()

const { canEdit, load: loadRole } = useRole()

const daily = ref<DailyStatistics | null>(null)
const monthly = ref<MonthlyStatistics | null>(null)
const loadingDaily = ref(true)
const loadingMonthly = ref(true)
const errorText = ref('')
const formVisible = ref(false)

const today = computed(() => dayjs().format('YYYY-MM-DD'))
const todayLabel = computed(() => dayjs().format('YYYY年MM月DD日'))

const netToday = computed(
  () => (daily.value?.total_income ?? 0) - (daily.value?.total_expense ?? 0),
)

const chartCategories = computed(
  () => (monthly.value?.daily ?? []).map((item) => dayjs(item.date).format('DD')),
)

const chartSeries = computed(() => [
  {
    name: '收入',
    color: '#22c55e',
    data: (monthly.value?.daily ?? []).map((item) => item.total_income ?? 0),
  },
  {
    name: '支出',
    color: '#ef4444',
    data: (monthly.value?.daily ?? []).map((item) => item.total_expense ?? 0),
  },
])

const hasMonthlyData = computed(() => (monthly.value?.daily ?? []).length > 0)

async function loadDaily(): Promise<void> {
  loadingDaily.value = true
  try {
    daily.value = await statisticsApi.daily(today.value)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载今日数据失败'
  } finally {
    loadingDaily.value = false
  }
}

async function loadMonthly(): Promise<void> {
  loadingMonthly.value = true
  try {
    monthly.value = await statisticsApi.monthly(dayjs().year(), dayjs().month() + 1)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载月度数据失败'
  } finally {
    loadingMonthly.value = false
  }
}

function reload(): void {
  errorText.value = ''
  void loadDaily()
  void loadMonthly()
}

function onRecordSuccess(): void {
  reload()
}

onShow(() => {
  void (async () => {
    if (!(await ensureSessionReady())) return
    void loadRole()
    reload()
  })()
})
</script>

<template>
  <AppPage
    title="汤圆记账"
    tabbar="home"
    show-credits
    show-tenant
  >
    <view class="home">
      <!-- 顶部 -->
      <view class="home__hero">
        <view class="home__hero-text">
          <text class="home__date">{{ todayLabel }}</text>
          <text class="home__greeting">今天也要好好记账呀</text>
        </view>
        <view v-if="canEdit" class="home__record-btn" @click="formVisible = true">
          <text class="home__record-plus">＋</text>
          <text>记一笔</text>
        </view>
      </view>

      <view v-if="errorText" class="home__error" @click="reload">
        <text>{{ errorText }}</text>
        <text class="home__error-retry">点击重试</text>
      </view>

      <!-- 今日概览 -->
      <view class="stat-grid">
        <view class="stat-card">
          <text class="stat-card__label">今日收入</text>
          <text v-if="loadingDaily" class="stat-card__skeleton" />
          <text v-else class="stat-card__value num stat-card__value--income">
            ¥{{ money(daily?.total_income ?? 0) }}
          </text>
        </view>
        <view class="stat-card">
          <text class="stat-card__label">今日支出</text>
          <text v-if="loadingDaily" class="stat-card__skeleton" />
          <text v-else class="stat-card__value num stat-card__value--expense">
            ¥{{ money(daily?.total_expense ?? 0) }}
          </text>
        </view>
        <view class="stat-card stat-card--net">
          <text class="stat-card__label">净结余</text>
          <text v-if="loadingDaily" class="stat-card__skeleton" />
          <text
            v-else
            class="stat-card__value num"
            :class="netToday >= 0 ? 'stat-card__value--income' : 'stat-card__value--expense'"
          >
            {{ netToday >= 0 ? '+' : '-' }}¥{{ money(Math.abs(netToday)) }}
          </text>
        </view>
      </view>

      <!-- 月趋势 -->
      <view class="card trend">
        <view class="trend__head">
          <view class="trend__title-wrap">
            <text class="trend__title">{{ dayjs().format('MM') }}月收支趋势</text>
            <text class="trend__sub">按日统计</text>
          </view>
          <view v-if="monthly" class="trend__totals">
            <text class="trend__total trend__total--in">
              收 ¥{{ money(monthly.total?.total_income ?? 0) }}
            </text>
            <text class="trend__total trend__total--out">
              支 ¥{{ money(monthly.total?.total_expense ?? 0) }}
            </text>
          </view>
        </view>

        <view v-if="loadingMonthly" class="trend__loading" />
        <EmptyState v-else-if="!hasMonthlyData" icon="📈" text="本月暂无数据" hint="记录第一笔交易后，这里会出现趋势图" />
        <ChartLine
          v-else
          :categories="chartCategories"
          :series="chartSeries"
          :height="220"
        />
      </view>

      <view v-if="!canEdit" class="home__readonly">
        当前角色为只读成员，如需记账请联系账本管理员
      </view>
    </view>

    <template #overlay>
      <TransactionForm
        v-model:visible="formVisible"
        mode="create"
        @success="onRecordSuccess"
      />
    </template>
  </AppPage>
</template>

<style lang="scss" scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.home__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  padding: 8rpx 4rpx;
}

.home__hero-text {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.home__date {
  font-size: $fs-xl;
  font-weight: 700;
  color: $c-text;
}

.home__greeting {
  font-size: $fs-xs;
  color: $c-text-3;
}

.home__record-btn {
  display: flex;
  align-items: center;
  gap: 6rpx;
  padding: 16rpx 28rpx;
  border-radius: 999rpx;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: #fff;
  font-size: $fs-sm;
  font-weight: 600;
  box-shadow: 0 8rpx 20rpx rgba(37, 99, 235, 0.28);
}

.home__record-plus {
  font-size: 32rpx;
  line-height: 1;
}

.home__error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18rpx 22rpx;
  border-radius: $radius-sm;
  background-color: $c-expense-bg;
  color: $c-expense;
  font-size: $fs-xs;
}

.home__error-retry {
  color: $c-primary;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14rpx;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
  padding: 22rpx 18rpx;
  border-radius: $radius;
  background-color: $c-card;
  border: 2rpx solid $c-border;
  box-shadow: $shadow-card;
}

.stat-card--net {
  background: linear-gradient(135deg, #eff6ff, #ffffff);
}

.stat-card__label {
  font-size: 20rpx;
  color: $c-text-3;
}

.stat-card__value {
  font-size: 30rpx;
  font-weight: 700;
}

.stat-card__value--income {
  color: $c-income;
}

.stat-card__value--expense {
  color: $c-expense;
}

.stat-card__skeleton {
  height: 36rpx;
  border-radius: 8rpx;
  background-color: #f3f4f6;
}

.trend {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.trend__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16rpx;
}

.trend__title-wrap {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.trend__title {
  font-size: $fs;
  font-weight: 600;
  color: $c-text;
}

.trend__sub {
  font-size: 20rpx;
  color: $c-text-3;
}

.trend__totals {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
}

.trend__total {
  font-size: 20rpx;
  font-weight: 500;
}

.trend__total--in {
  color: $c-income;
}

.trend__total--out {
  color: $c-expense;
}

.trend__loading {
  height: 260rpx;
  border-radius: $radius-sm;
  background-color: $c-bg;
  animation: pulse 1.2s ease-in-out infinite;
}

.home__readonly {
  padding: 18rpx 22rpx;
  border-radius: $radius-sm;
  background-color: $c-warn-bg;
  color: $c-warn;
  font-size: $fs-xs;
  text-align: center;
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
