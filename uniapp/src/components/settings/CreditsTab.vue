<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import { creditApi } from '@/api/credit'
import type { CreditLogItem, CreditLotItem } from '@/api/types'
import { humanDateTime } from '@/utils/datetime'
import { goBilling } from '@/utils/nav'
import { toastError } from '@/utils/ui'

const ACTION_LABEL: Record<string, string> = {
  upload: '上传图片',
  ocr: 'OCR识别',
  llm: 'AI分析',
  orc2bill: 'AI整理',
  gift: '注册赠送',
  admin_adjust: '管理员调整',
  storage: '存储扣费',
  expire: '到期作废',
}

const PAGE_SIZE = 10

type AmountType = 'all' | 'income' | 'expense'

const balance = ref(0)
const unlimited = ref(false)
const lots = ref<CreditLotItem[]>([])
const balanceLoading = ref(true)

const logs = ref<CreditLogItem[]>([])
const total = ref(0)
const amountType = ref<AmountType>('all')
const loading = ref(true)

const finished = computed(() => logs.value.length >= total.value)

let page = 1

function actionLabel(action: string): string {
  return ACTION_LABEL[action] ?? action
}

async function loadBalance(): Promise<void> {
  balanceLoading.value = true
  try {
    const data = await creditApi.getBalance()
    balance.value = data?.balance ?? 0
    unlimited.value = !!data?.unlimited
    lots.value = data?.lots ?? []
  } catch (err) {
    toastError(err, '加载积分余额失败')
  } finally {
    balanceLoading.value = false
  }
}

async function loadLogs(reset = false): Promise<void> {
  if (loading.value && !reset) return
  loading.value = true
  const target = reset ? 1 : page
  try {
    const data = await creditApi.getLogs(
      target,
      PAGE_SIZE,
      amountType.value === 'all' ? undefined : amountType.value,
    )
    const list = data?.logs ?? []
    logs.value = reset ? list : [...logs.value, ...list]
    total.value = data?.total ?? logs.value.length
    page = target + 1
  } catch (err) {
    toastError(err, '加载积分明细失败')
  } finally {
    loading.value = false
  }
}

function switchType(type: AmountType): void {
  if (amountType.value === type) return
  amountType.value = type
  page = 1
  void loadLogs(true)
}

onMounted(() => {
  void loadBalance()
  void loadLogs(true)
})
</script>

<template>
  <view class="credits">
    <view class="credit-card">
      <text class="credit-card__label">当前积分余额</text>
      <text v-if="balanceLoading" class="credit-card__value credit-card__value--muted">…</text>
      <text v-else-if="unlimited" class="credit-card__value">无限制</text>
      <text v-else class="credit-card__value num">{{ balance.toLocaleString() }}</text>
      <text v-if="unlimited" class="credit-card__hint">已开启白名单，不受积分限制</text>
      <text v-else-if="!balanceLoading && lots.length" class="credit-card__warn">
        最早 {{ lots[0]?.remaining?.toLocaleString() ?? 0 }} 积分将于
        {{ lots[0]?.expires_at?.slice(0, 10) ?? '—' }} 过期，将优先扣除
      </text>
      <view class="credit-card__billing" @click="goBilling">
        <text>积分充值</text>
        <text class="credit-card__billing-hint">购买积分 / 查看订单 ›</text>
      </view>
    </view>

    <view class="card list-card">
      <view class="list-card__head">
        <text class="list-card__title">积分明细</text>
        <view class="credits__tabs">
          <view
            v-for="tab in [
              { key: 'all', label: '全部' },
              { key: 'income', label: '收入' },
              { key: 'expense', label: '支出' },
            ]"
            :key="tab.key"
            class="credits__tab"
            :class="{ 'credits__tab--active': amountType === tab.key }"
            @click="switchType(tab.key as AmountType)"
          >
            {{ tab.label }}
          </view>
        </view>
      </view>

      <view v-if="loading && logs.length === 0" class="list-state">加载中…</view>
      <EmptyState v-else-if="logs.length === 0" icon="🎫" text="暂无积分记录" />
      <view v-else class="credit-list">
        <view v-for="log in logs" :key="log.id" class="credit-item">
          <view class="credit-item__main">
            <text class="credit-item__action">{{ actionLabel(log.action) }}</text>
            <text v-if="log.note" class="credit-item__note">{{ log.note }}</text>
            <text class="credit-item__time">{{ humanDateTime(log.created_at) }}</text>
          </view>
          <view class="credit-item__side">
            <text
              class="credit-item__amount num"
              :class="log.amount >= 0 ? 'credit-item__amount--in' : 'credit-item__amount--out'"
            >
              {{ log.amount >= 0 ? '+' : '' }}{{ log.amount.toLocaleString() }}
            </text>
            <text class="credit-item__balance">
              余额：{{ log.balance_after === -1 ? '∞' : log.balance_after.toLocaleString() }}
            </text>
          </view>
        </view>
      </view>

      <view v-if="logs.length > 0" class="credit-more">
        <text v-if="finished" class="credit-more__text">共 {{ total }} 条，已全部加载</text>
        <view v-else class="btn btn-ghost btn-sm" @click="loadLogs(false)">
          {{ loading ? '加载中…' : '加载更多' }}
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.credits {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.credit-card {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  padding: 32rpx;
  border-radius: $radius;
  background: linear-gradient(135deg, #eff6ff, #eef2ff);
  border: 2rpx solid #dbeafe;
}

.credit-card__billing {
  margin-top: 20rpx;
  padding-top: 20rpx;
  border-top: 1rpx solid rgba(79, 70, 229, 0.15);
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #4338ca;
  font-size: 26rpx;
}
.credit-card__billing-hint {
  color: #6d28d9;
  font-weight: 600;
}

.credit-card__label {
  font-size: $fs-sm;
  color: $c-text-2;
}

.credit-card__value {
  font-size: 56rpx;
  font-weight: 700;
  color: $c-primary;
  line-height: 1.2;
}

.credit-card__value--muted {
  color: #93c5fd;
}

.credit-card__hint {
  font-size: $fs-xs;
  color: #60a5fa;
}

.credit-card__warn {
  font-size: $fs-xs;
  color: $c-warn;
}

.list-card {
  padding: 0;
  overflow: hidden;
}

.list-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 20rpx 24rpx;
  border-bottom: 2rpx solid $c-border;
}

.list-card__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.credits__tabs {
  display: flex;
  gap: 4rpx;
  padding: 4rpx;
  border-radius: 10rpx;
  background-color: #f3f4f6;
}

.credits__tab {
  padding: 8rpx 16rpx;
  border-radius: 8rpx;
  font-size: 20rpx;
  color: $c-text-2;
}

.credits__tab--active {
  background-color: #fff;
  color: $c-text;
  font-weight: 600;
  box-shadow: 0 2rpx 6rpx rgba(17, 24, 39, 0.08);
}

.list-state {
  padding: 60rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-3;
}

.credit-list {
  display: flex;
  flex-direction: column;
}

.credit-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20rpx;
  padding: 22rpx 24rpx;
  border-bottom: 2rpx solid $c-border;
}

.credit-item__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.credit-item__action {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
}

.credit-item__note {
  font-size: $fs-xs;
  color: $c-text-3;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.credit-item__time {
  font-size: $fs-xs;
  color: $c-text-3;
}

.credit-item__side {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
}

.credit-item__amount {
  font-size: $fs;
  font-weight: 600;
}

.credit-item__amount--in {
  color: $c-income;
}

.credit-item__amount--out {
  color: $c-expense;
}

.credit-item__balance {
  font-size: $fs-xs;
  color: $c-text-3;
}

.credit-more {
  display: flex;
  justify-content: center;
  padding: 20rpx;
}

.credit-more__text {
  font-size: $fs-xs;
  color: $c-text-3;
}
</style>
