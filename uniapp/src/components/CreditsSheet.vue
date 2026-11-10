<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AppModal from '@/components/AppModal.vue'
import EmptyState from '@/components/EmptyState.vue'
import { creditApi } from '@/api/credit'
import type { CreditLogItem, CreditLotItem } from '@/api/types'
import { humanDateTime } from '@/utils/datetime'
import { toastError } from '@/utils/ui'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'close'): void
}>()

const visibleModel = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

type AmountType = 'all' | 'income' | 'expense'

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

const balance = ref(0)
const unlimited = ref(false)
const lots = ref<CreditLotItem[]>([])
const balanceLoading = ref(false)

const logs = ref<CreditLogItem[]>([])
const total = ref(0)
const amountType = ref<AmountType>('all')
const loading = ref(false)
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
  } catch (e) {
    // 积分展示失败不阻塞主流程
  } finally {
    balanceLoading.value = false
  }
}

async function loadLogs(reset = false): Promise<void> {
  if (loading.value) return
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

function init(): void {
  balance.value = 0
  unlimited.value = false
  lots.value = []
  logs.value = []
  total.value = 0
  amountType.value = 'all'
  page = 1
  void loadBalance()
  void loadLogs(true)
}

function switchType(type: AmountType): void {
  if (amountType.value === type) return
  amountType.value = type
  page = 1
  void loadLogs(true)
}

watch(
  () => props.visible,
  (value) => {
    if (value) init()
  },
)
</script>

<template>
  <AppModal v-model:visible="visibleModel" title="积分明细" @close="emit('close')">
    <!-- 余额卡 -->
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
    </view>

    <!-- 筛选 -->
    <view class="credit-tabs">
      <view
        v-for="tab in [
          { key: 'all', label: '全部' },
          { key: 'income', label: '收入' },
          { key: 'expense', label: '支出' },
        ]"
        :key="tab.key"
        class="credit-tabs__item"
        :class="{ 'credit-tabs__item--active': amountType === tab.key }"
        @click="switchType(tab.key as AmountType)"
      >
        {{ tab.label }}
      </view>
    </view>

    <!-- 流水 -->
    <view v-if="loading && logs.length === 0" class="credit-loading">加载中…</view>
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
  </AppModal>
</template>

<style lang="scss" scoped>
.credit-card {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  padding: 32rpx;
  border-radius: $radius;
  background: linear-gradient(135deg, #eff6ff, #eef2ff);
  border: 2rpx solid #dbeafe;
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

.credit-tabs {
  display: flex;
  gap: 8rpx;
  padding: 8rpx;
  margin: 24rpx 0 8rpx;
  border-radius: $radius-sm;
  background-color: #f3f4f6;
}

.credit-tabs__item {
  flex: 1;
  text-align: center;
  padding: 12rpx 0;
  border-radius: 10rpx;
  font-size: $fs-sm;
  color: $c-text-2;
}

.credit-tabs__item--active {
  background-color: #fff;
  color: $c-text;
  font-weight: 600;
  box-shadow: 0 2rpx 6rpx rgba(17, 24, 39, 0.08);
}

.credit-loading {
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
  padding: 22rpx 0;
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
  padding: 24rpx 0 8rpx;
  display: flex;
  justify-content: center;
}

.credit-more__text {
  font-size: $fs-xs;
  color: $c-text-3;
}
</style>
