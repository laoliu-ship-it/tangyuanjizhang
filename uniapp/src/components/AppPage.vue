<script setup lang="ts">
import { computed } from 'vue'
import AppHeader from '@/components/AppHeader.vue'
import AppTabBar from '@/components/AppTabBar.vue'
import type { TabKey } from '@/utils/nav'

const props = withDefaults(
  defineProps<{
    title?: string
    back?: boolean
    tabbar?: boolean | TabKey
    showCredits?: boolean
    showTenant?: boolean
    padded?: boolean
  }>(),
  {
    title: '',
    back: false,
    tabbar: false,
    showCredits: false,
    showTenant: false,
    padded: true,
  },
)

const emit = defineEmits<{
  (e: 'scrolltolower'): void
  (e: 'scroll', detail: unknown): void
}>()

/** 路由 -> Tab key */
const ROUTE_TABS: Record<string, TabKey> = {
  'pages/home/index': 'home',
  'pages/transactions/index': 'transactions',
  'pages/statistics/index': 'statistics',
  'pages/settings/index': 'settings',
  'pages/help/index': 'help',
}

const activeTab = computed<TabKey | null>(() => {
  if (!props.tabbar) return null
  if (typeof props.tabbar === 'string') return props.tabbar
  try {
    const pages = getCurrentPages()
    const current = pages[pages.length - 1] as unknown as { route?: string }
    const route = current?.route ?? ''
    return ROUTE_TABS[route] ?? null
  } catch (e) {
    return null
  }
})
</script>

<template>
  <view class="app-page">
    <AppHeader
      :title="title"
      :back="back"
      :show-credits="showCredits"
      :show-tenant="showTenant"
    >
      <template #right>
        <slot name="right" />
      </template>
    </AppHeader>

    <scroll-view
      class="app-page__body"
      scroll-y
      :lower-threshold="120"
      @scrolltolower="emit('scrolltolower')"
      @scroll="emit('scroll', $event)"
    >
      <view class="app-page__content" :class="{ 'app-page__content--padded': padded }">
        <slot />
      </view>
    </scroll-view>

    <!-- 全屏弹层/固定操作条放在滚动区之外，避免小程序 scroll-view 原生层裁剪 -->
    <slot name="overlay" />

    <AppTabBar v-if="activeTab" :active="activeTab" />
  </view>
</template>

<style lang="scss" scoped>
.app-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background-color: $c-bg;
}

.app-page__body {
  flex: 1;
  height: 0;
  min-height: 0;
}

.app-page__content {
  min-height: 100%;
  padding-bottom: calc(32rpx + env(safe-area-inset-bottom));
}

.app-page__content--padded {
  padding: 24rpx;
  padding-bottom: calc(48rpx + env(safe-area-inset-bottom));
}

/* H5 桌面端居中限宽，避免 rpx 放大后内容过宽 */
@media (min-width: 960px) {
  .app-page {
    width: 750px;
    margin: 0 auto;
    box-shadow: 0 0 40rpx rgba(17, 24, 39, 0.08);
  }
}
</style>
