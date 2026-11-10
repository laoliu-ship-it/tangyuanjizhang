<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'
import TransactionCard from '@/components/transactions/TransactionCard.vue'
import TransactionForm from '@/components/transaction/TransactionForm.vue'
import { transactionApi } from '@/api/transaction'
import type { Transaction } from '@/api/types'
import { ensureSessionReady, useAuthGuard } from '@/composables/useAuthGuard'
import { useMetaStore } from '@/store/meta'
import { money } from '@/utils/format'
import { toastError } from '@/utils/ui'

useAuthGuard()

const PAGE_SIZE = 20

const meta = useMetaStore()

const merchantId = ref(0)
const merchantName = ref('商户明细')
const startDate = ref('')
const endDate = ref('')

const list = ref<Transaction[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const errorText = ref('')
const detailTarget = ref<Transaction | null>(null)

const categories = computed(() => meta.categories)
const finished = computed(() => list.value.length >= total.value)
const loadedTotal = computed(() =>
  list.value.reduce((sum, tx) => sum + (Number(tx.amount) || 0), 0),
)

const detailVisible = computed({
  get: () => detailTarget.value !== null,
  set: (value: boolean) => {
    if (!value) detailTarget.value = null
  },
})

const rangeText = computed(() => {
  if (startDate.value && endDate.value) return `${startDate.value} 至 ${endDate.value}`
  if (startDate.value) return `${startDate.value} 起`
  if (endDate.value) return `截至 ${endDate.value}`
  return '全部时间'
})

function safeDecode(value?: string): string {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch (e) {
    return value
  }
}

function categoryOf(tx: Transaction) {
  return categories.value.find((item) => item.id === tx.category_id)
}

async function loadPage(targetPage: number, append: boolean): Promise<void> {
  if (!merchantId.value || loading.value) return
  loading.value = true
  errorText.value = ''
  try {
    const result = await transactionApi.list({
      page: targetPage,
      page_size: PAGE_SIZE,
      merchant_id: merchantId.value,
      start_date: startDate.value || undefined,
      end_date: endDate.value || undefined,
      sort_by: 'transaction_date',
    })
    const items = Array.isArray(result?.items) ? result.items : []
    list.value = append ? [...list.value, ...items] : items
    total.value = Number(result?.total ?? items.length)
    page.value = targetPage
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载商户交易失败'
  } finally {
    loading.value = false
  }
}

function loadMore(): void {
  if (loading.value || finished.value) return
  void loadPage(page.value + 1, true)
}

function reload(): void {
  void loadPage(1, false)
}

function openDetail(tx: Transaction): void {
  detailTarget.value = tx
}

onLoad((options) => {
  const id = Number(options?.id ?? 0)
  merchantId.value = Number.isFinite(id) ? id : 0
  merchantName.value = safeDecode(options?.name) || '商户明细'
  startDate.value = safeDecode(options?.start)
  endDate.value = safeDecode(options?.end)
  if (!merchantId.value) {
    toastError(new Error('商户信息缺失'))
  }
})

onShow(() => {
  void (async () => {
    if (!(await ensureSessionReady())) return
    if (!merchantId.value) return
    void meta.ensureCategories().catch(() => {})
    reload()
  })()
})
</script>

<template>
  <AppPage :title="merchantName" back @scrolltolower="loadMore">
    <view class="merchant-page">
      <view class="summary">
        <view class="summary__main">
          <text class="summary__name">{{ merchantName }}</text>
          <text class="summary__range">{{ rangeText }}</text>
        </view>
        <view class="summary__amount">
          <text class="summary__amount-label">合计</text>
          <text class="summary__amount-value num">¥{{ money(loadedTotal) }}</text>
        </view>
      </view>

      <text class="summary__loaded">已加载 {{ list.length }} 笔，共 {{ total }} 笔</text>

      <view v-if="errorText" class="error-bar" @click="reload">
        <text>{{ errorText }}</text>
        <text class="error-bar__retry">点击重试</text>
      </view>

      <view v-if="loading && list.length === 0" class="state">加载中…</view>
      <EmptyState v-else-if="!loading && list.length === 0" icon="🏪" text="该商户暂无交易记录" />

      <view v-else class="tx-list">
        <TransactionCard
          v-for="tx in list"
          :key="tx.id"
          :tx="tx"
          :category="categoryOf(tx)"
          @click="openDetail"
        />
      </view>

      <view v-if="list.length > 0" class="footer">
        <text v-if="loading" class="footer__text">加载中…</text>
        <text v-else-if="finished" class="footer__text">已全部加载</text>
        <text v-else class="footer__text">上拉加载更多</text>
      </view>
    </view>

    <template #overlay>
      <TransactionForm
        v-model:visible="detailVisible"
        mode="detail"
        :initial-data="detailTarget ?? undefined"
      />
    </template>
  </AppPage>
</template>

<style lang="scss" scoped>
.merchant-page {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  padding: 28rpx;
  border-radius: $radius;
  background: linear-gradient(135deg, #eff6ff, #ffffff);
  border: 2rpx solid #dbeafe;
}

.summary__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.summary__name {
  font-size: $fs-lg;
  font-weight: 700;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.summary__range {
  font-size: $fs-xs;
  color: $c-text-3;
}

.summary__amount {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
}

.summary__amount-label {
  font-size: 20rpx;
  color: $c-text-3;
}

.summary__amount-value {
  font-size: $fs-lg;
  font-weight: 700;
  color: $c-expense;
}

.summary__loaded {
  font-size: 20rpx;
  color: $c-text-3;
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

.state {
  padding: 80rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-3;
}

.tx-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.footer {
  display: flex;
  justify-content: center;
  padding: 20rpx 0;
}

.footer__text {
  font-size: $fs-xs;
  color: $c-text-3;
}
</style>
