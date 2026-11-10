<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import CreditsSheet from '@/components/CreditsSheet.vue'
import TenantSwitcher from '@/components/TenantSwitcher.vue'
import { creditApi } from '@/api/credit'
import { useTenantStore } from '@/store/tenant'
import { compactMoney } from '@/utils/format'
import { goBack } from '@/utils/nav'

withDefaults(
  defineProps<{
    title: string
    back?: boolean
    showCredits?: boolean
    showTenant?: boolean
  }>(),
  { back: false, showCredits: false, showTenant: false },
)

const tenantStore = useTenantStore()

const systemInfo = uni.getSystemInfoSync()
const statusBarHeight = Number(systemInfo.statusBarHeight ?? 0)
const headerPaddingTop = statusBarHeight > 0 ? `${statusBarHeight}px` : 'env(safe-area-inset-top)'

/** 微信小程序右上角胶囊按钮占位，避免头部内容被遮挡 */
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

const isMp = ref(false)
// #ifdef MP-WEIXIN
isMp.value = true
// #endif

const balance = ref<number | null>(null)
const unlimited = ref(false)
const creditsVisible = ref(false)

const creditText = computed(() => {
  if (unlimited.value) return '∞'
  if (balance.value === null) return ''
  return balance.value >= 10000 ? compactMoney(balance.value) : balance.value.toLocaleString()
})

async function loadCredits(): Promise<void> {
  if (!tenantStore.currentTenantId) return
  try {
    const data = await creditApi.getBalance()
    balance.value = data?.balance ?? 0
    unlimited.value = !!data?.unlimited
  } catch (e) {
    // 头部积分展示失败时静默
  }
}

watch(
  () => tenantStore.currentTenantId,
  () => {
    balance.value = null
    unlimited.value = false
    void loadCredits()
  },
)

onMounted(() => {
  void loadCredits()
})
</script>

<template>
  <view class="header" :style="{ paddingTop: headerPaddingTop }">
    <view
      class="header__bar"
      :style="{ paddingRight: capsulePadding ? `${capsulePadding}px` : '24rpx' }"
    >
      <view class="header__left">
        <view v-if="back" class="header__back" @click="goBack()">‹</view>
        <view v-else class="header__logo">汤</view>
      </view>

      <view class="header__center">
        <text class="header__title">{{ title }}</text>
      </view>

      <view class="header__right">
        <slot name="right" />
        <template v-if="!(isMp && showCredits && showTenant)">
          <view v-if="showCredits && creditText" class="header__credits" @click="creditsVisible = true">
            <text class="header__coin">C</text>
            <text class="header__credit-text">{{ creditText }}</text>
          </view>
          <TenantSwitcher v-if="showTenant" compact />
        </template>
      </view>
    </view>

    <!-- 微信小程序：胶囊按钮占用右侧空间，积分与账本切换放第二行，避免挤压标题 -->
    <view v-if="isMp && (showCredits || showTenant)" class="header__sub">
      <TenantSwitcher v-if="showTenant" compact />
      <view v-if="showCredits && creditText" class="header__credits" @click="creditsVisible = true">
        <text class="header__coin">C</text>
        <text class="header__credit-text">{{ creditText }}</text>
      </view>
    </view>

    <CreditsSheet v-model:visible="creditsVisible" />
  </view>
</template>

<style lang="scss" scoped>
.header {
  flex-shrink: 0;
  background-color: $c-card;
  border-bottom: 2rpx solid $c-border;
}

.header__bar {
  display: flex;
  align-items: center;
  height: 96rpx;
  padding-left: 24rpx;
  gap: 12rpx;
}

.header__left {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.header__back {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 52rpx;
  line-height: 1;
  color: $c-text;
  border-radius: 50%;
}

.header__back:active {
  background-color: $c-border;
}

.header__logo {
  width: 56rpx;
  height: 56rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 28rpx;
  font-weight: 700;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
}

.header__center {
  flex: 1;
  min-width: 0;
  display: flex;
  justify-content: center;
}

.header__title {
  max-width: 100%;
  font-size: $fs-lg;
  font-weight: 700;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.header__right {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.header__credits {
  display: flex;
  align-items: center;
  gap: 4rpx;
  padding: 6rpx 12rpx;
  border-radius: 999rpx;
  background-color: #fffbeb;
}

.header__coin {
  width: 26rpx;
  height: 26rpx;
  border-radius: 50%;
  background-color: #f59e0b;
  color: #fff;
  font-size: 18rpx;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.header__credit-text {
  font-size: $fs-xs;
  color: $c-warn;
  font-weight: 600;
}

.header__sub {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 0 24rpx 16rpx;
}
</style>
