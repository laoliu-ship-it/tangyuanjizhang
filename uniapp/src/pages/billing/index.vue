<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'
import { billingApi, type BillingEstimates, type BillingIntent, type BillingPackage, type BillingSummary } from '@/api/billing'
import { creditApi } from '@/api/credit'
import type { CreditLogItem } from '@/api/types'
import { useAuthStore } from '@/store/auth'
import { useAuthGuard } from '@/composables/useAuthGuard'
import { useRole } from '@/composables/useRole'
import { useTenantStore } from '@/store/tenant'
import { canPurchaseBilling, isIOSWeChatSupported } from '@/utils/billingCapability'
import { humanDateTime } from '@/utils/datetime'
import { toastError } from '@/utils/ui'
import { getWxRuntime, supportsVirtualPayment } from '@/utils/wxVirtualPayment'

const authStore = useAuthStore()
const tenantStore = useTenantStore()
const { canEdit: canManage, load: loadRole } = useRole()
useAuthGuard()

const loading = ref(true)
const buying = ref('')
const summary = ref<BillingSummary | null>(null)
const intents = ref<BillingIntent[]>([])
const logs = ref<CreditLogItem[]>([])
const message = ref('')
const canPay = ref(false)

// #ifdef MP-WEIXIN
canPay.value = supportsVirtualPayment()
// #endif

const systemInfo = uni.getSystemInfoSync()
const platform = String(systemInfo.platform || '').toLowerCase()
const wechatVersion = String(systemInfo.version || '')
const isIOS = computed(() => platform === 'ios')
const isDemo = computed(() => !!authStore.demo)
const isAdmin = computed(() => canManage.value)
const iapEnabled = computed(() => !!summary.value?.iap_enabled)
const packages = computed(() => (summary.value?.packages ?? []).filter((p) => p.status === 'listed'))
const canPurchase = computed(() =>
  canPurchaseBilling({
    enabled: !!summary.value?.enabled,
    platform,
    iapEnabled: iapEnabled.value,
    wechatVersion,
    supportsVirtualPayment: canPay.value,
    isAdmin: isAdmin.value,
    isDemo: isDemo.value,
    packageCount: packages.value.length,
  }),
)
const iosTerminalSupported = computed(
  () => !isIOS.value || (iapEnabled.value && isIOSWeChatSupported(wechatVersion)),
)

const fmtFen = (v: number) => `¥${(v / 100).toFixed(2)}`
const statusLabel = (status: string): string => {
  const map: Record<string, string> = {
    created: '待支付',
    paid: '已支付',
    delivered: '已到账',
    cancelled: '已取消',
    refunded: '已退款',
  }
  return map[status] ?? status
}
const ACTION_LABEL: Record<string, string> = {
  purchase: '购买积分到账',
  refund: '退款回收',
  upload: '上传图片',
  ocr: 'OCR识别',
  llm: 'AI分析',
  orc2bill: 'AI整理',
  gift: '注册赠送',
  admin_adjust: '管理员调整',
  storage: '存储扣费',
  expire: '到期作废',
}
const actionLabel = (action: string) => ACTION_LABEL[action] ?? action

function estimateTextOf(e: BillingEstimates | undefined): string {
  if (!e) return ''
  const parts: string[] = []
  if (e.upload_ocr_times !== null && e.upload_ocr_times !== undefined) {
    parts.push(`上传+OCR 约 ${e.upload_ocr_times} 次`)
  }
  if (e.ai_times !== null && e.ai_times !== undefined) parts.push(`AI 整理约 ${e.ai_times} 次`)
  if (e.full_chain_times !== null && e.full_chain_times !== undefined) {
    parts.push(`全链路约 ${e.full_chain_times} 次`)
  }
  if (!parts.length) return '当前扣费单价为 0 或未配置，暂无法给出有限次数估算'
  return `${parts.join(' / ')}（共享同一积分，不可叠加；按当前扣费配置）`
}

const estimateText = computed(() => estimateTextOf(summary.value?.estimates))
const packageEstimateText = (e: BillingEstimates | undefined) => estimateTextOf(e)
const hasPositiveUnit = computed(() => {
  const u = summary.value?.unit_prices
  return !!u && (u.upload > 0 || u.ocr > 0 || u.ai > 0)
})
const balanceHint = computed(() => {
  if (!summary.value || summary.value.unlimited || summary.value.credit_balance !== 0) return ''
  return hasPositiveUnit.value ? '积分不足，OCR/AI 将无法使用' : '当前扣费单价为 0 或未配置，暂不扣积分'
})
const unitPriceText = computed(() => {
  const u = summary.value?.unit_prices
  if (!u) return ''
  return `单价：上传 ${u.upload} / OCR ${u.ocr} / AI ${u.ai} 积分`
})

async function load(refresh = false): Promise<void> {
  if (!tenantStore.currentTenantId) return
  loading.value = true
  message.value = ''
  try {
    const [s, i, l] = await Promise.all([
      billingApi.summary(refresh),
      billingApi.intents().catch(() => ({ total: 0, page: 1, size: 20, items: [] })),
      creditApi.getLogs(1, 20).catch(() => ({ total: 0, page: 1, size: 20, logs: [] })),
    ])
    summary.value = s
    intents.value = i.items ?? []
    logs.value = l.logs ?? []
    if (s.stale) message.value = '展示的是最近一次同步结果，请点击刷新获取最新价格/上下架状态'
    if (isIOS.value && (!s.iap_enabled || !isIOSWeChatSupported(wechatVersion))) {
      message.value = '当前终端暂不可用，请稍后再试'
    }
  } catch (err) {
    toastError(err, '加载积分商品失败')
  } finally {
    loading.value = false
  }
}

function refresh(): void {
  void load(true)
}

function loginCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (res) => (res.code ? resolve(res.code) : reject(new Error('获取微信支付凭证失败'))),
      fail: () => reject(new Error('获取微信支付凭证失败')),
    })
  })
}

async function pollIntent(ref: string): Promise<void> {
  for (let i = 0; i < 15; i += 1) {
    await new Promise((r) => setTimeout(r, 2000))
    const data = await billingApi.intentsByRef(ref).catch(() => null)
    if (!data) continue
    if (data.status === 'delivered' || data.status === 'refunded') {
      message.value = data.status === 'delivered' ? '支付成功，积分已到账' : '订单已退款'
      await load()
      return
    }
  }
  message.value = '支付结果确认中，请稍后在订单列表查看'
  await load()
}

async function buy(pkg: BillingPackage): Promise<void> {
  if (isDemo.value) {
    message.value = '演示账号不可购买'
    return
  }
  if (!isAdmin.value) {
    message.value = '仅账本管理员可购买'
    return
  }
  if (!canPay.value) {
    message.value = '当前环境暂不支持购买，请在微信中打开小程序'
    return
  }
  if (isIOS.value && !iosTerminalSupported.value) {
    message.value = '当前终端暂不可用，请稍后再试'
    return
  }
  if (buying.value) return
  buying.value = pkg.sku_code
  message.value = ''
  try {
    const intent = await billingApi.createIntent(pkg.sku_code, pkg.price_fen, platform || 'unknown')
    const code = await loginCode()
    const pay = await billingApi.prepare(intent.business_ref, code)
    await new Promise<void>((resolve, reject) => {
      const wxRuntime = getWxRuntime()
      if (!wxRuntime?.requestVirtualPayment) {
        reject(new Error('当前终端暂不支持支付'))
        return
      }
      wxRuntime.requestVirtualPayment({
        signData: pay.signData,
        mode: pay.mode,
        paySig: pay.paySig,
        signature: pay.signature,
        success: () => resolve(),
        fail: (err: { errMsg?: string }) =>
          reject(new Error(err?.errMsg?.includes('cancel') ? '已取消支付' : err?.errMsg || '支付失败')),
      })
    })
    message.value = '支付已提交，正在确认到账…'
    await pollIntent(intent.business_ref)
  } catch (err) {
    toastError(err, '购买失败')
  } finally {
    buying.value = ''
  }
}

onShow(async () => {
  await loadRole()
  await load()
})
</script>

<template>
  <AppPage title="积分充值" back>
    <view v-if="isDemo" class="demo-banner">演示账号（只读）：不能购买、不能修改账本数据</view>
    <view v-if="message" class="notice">{{ message }}</view>

    <view v-if="loading" class="loading">加载中…</view>
    <template v-else>
      <view class="balance-card">
        <view class="balance-item">
          <text class="balance-label">当前积分余额</text>
          <text class="balance-value">{{ summary?.unlimited ? '∞' : (summary?.credit_balance ?? 0).toLocaleString() }}</text>
        </view>
        <view class="balance-meta">
          <text v-if="balanceHint" class="warn">{{ balanceHint }}</text>
          <text v-if="summary?.synced_at">商品同步：{{ humanDateTime(summary.synced_at) }}</text>
        </view>
        <button class="refresh-btn" size="mini" @tap="refresh">刷新价格/上下架</button>
      </view>

      <view class="estimate-card">
        <text class="estimate-title">积分可用于</text>
        <text class="estimate-text">{{ estimateText }}</text>
        <text class="estimate-price">{{ unitPriceText }}</text>
        <text class="estimate-hint">OCR 估算包含上传+识别；AI 为单独整理步骤。不同用途互斥，不是独立配额。</text>
      </view>

      <view class="section-title">积分商品</view>
      <EmptyState v-if="!packages.length" text="暂无可售商品，请稍后再试" />
      <view v-for="pkg in packages" :key="pkg.sku_code" class="pkg-card">
        <view class="pkg-main">
          <text class="pkg-name">{{ pkg.name }}</text>
          <text class="pkg-desc">
            到账 {{ pkg.credit_amount.toLocaleString() }} 积分 · 有效期 {{ pkg.validity_days }} 天
          </text>
          <text class="pkg-estimate">{{ packageEstimateText(pkg.estimates) }}</text>
        </view>
        <view class="pkg-right">
          <text class="pkg-price">{{ fmtFen(pkg.price_fen) }}</text>
          <button
            class="buy-btn"
            size="mini"
            :disabled="!!buying || !canPurchase"
            :loading="buying === pkg.sku_code"
            @tap="buy(pkg)"
          >
            购买
          </button>
        </view>
      </view>

      <view class="section-title">我的订单</view>
      <EmptyState v-if="!intents.length" text="暂无订单" />
      <view v-for="o in intents" :key="o.business_ref" class="row-card">
        <view>
          <text class="row-title">{{ o.sku_code }}</text>
          <text class="row-sub">{{ statusLabel(o.status) }} · 到账 {{ o.credit_amount }} 积分</text>
        </view>
        <text class="row-amount">{{ fmtFen(o.amount_fen) }}</text>
      </view>

      <view class="section-title">积分流水</view>
      <EmptyState v-if="!logs.length" text="暂无积分流水" />
      <view v-for="l in logs" :key="l.id" class="row-card">
        <view>
          <text class="row-title">{{ actionLabel(l.action) }}</text>
          <text class="row-sub">{{ l.note }} · {{ humanDateTime(l.created_at) }}</text>
        </view>
        <text class="row-amount" :class="{ minus: l.amount < 0 }">{{ l.amount > 0 ? '+' : '' }}{{ l.amount.toLocaleString() }}</text>
      </view>
    </template>
  </AppPage>
</template>

<style lang="scss" scoped>
.demo-banner {
  background: #fef3c7;
  color: #92400e;
  border-radius: 16rpx;
  padding: 20rpx;
  margin-bottom: 20rpx;
  font-size: 26rpx;
}
.notice {
  background: #eef2ff;
  color: #3730a3;
  border-radius: 16rpx;
  padding: 20rpx;
  margin-bottom: 20rpx;
  font-size: 26rpx;
}
.loading { color: #9ca3af; padding: 40rpx 0; text-align: center; }
.balance-card {
  background: #fff;
  border-radius: 24rpx;
  padding: 28rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 24rpx;
  align-items: center;
}
.balance-item { display: flex; flex-direction: column; min-width: 220rpx; }
.balance-label { color: #6b7280; font-size: 24rpx; }
.balance-value { font-size: 44rpx; font-weight: 700; color: #111827; }
.balance-meta { flex: 1; display: flex; flex-direction: column; color: #6b7280; font-size: 22rpx; gap: 6rpx; }
.balance-meta .warn { color: #dc2626; }
.refresh-btn { margin-left: auto; }
.estimate-card {
  background: #f5f3ff;
  border-radius: 20rpx;
  padding: 24rpx;
  margin-top: 20rpx;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}
.estimate-title { font-size: 26rpx; font-weight: 600; color: #5b21b6; }
.estimate-text { font-size: 26rpx; color: #4c1d95; }
.estimate-price { font-size: 22rpx; color: #6d28d9; }
.estimate-hint { font-size: 20rpx; color: #7c3aed; }
.section-title { font-size: 28rpx; font-weight: 600; color: #374151; margin: 32rpx 4rpx 16rpx; }
.pkg-card, .row-card {
  background: #fff;
  border-radius: 20rpx;
  padding: 24rpx;
  margin-bottom: 16rpx;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16rpx;
}
.pkg-name { display: block; font-size: 30rpx; font-weight: 600; color: #111827; }
.pkg-desc, .row-sub { display: block; color: #6b7280; font-size: 22rpx; margin-top: 8rpx; }
.pkg-estimate { display: block; color: #6d28d9; font-size: 20rpx; margin-top: 6rpx; }
.pkg-right { display: flex; flex-direction: column; align-items: flex-end; gap: 10rpx; }
.pkg-price { color: #dc2626; font-size: 32rpx; font-weight: 700; }
.buy-btn { background: #4f46e5; color: #fff; border: none; }
.row-title { font-size: 28rpx; color: #111827; }
.row-amount { font-size: 28rpx; color: #059669; font-weight: 600; }
.row-amount.minus { color: #dc2626; }
</style>
