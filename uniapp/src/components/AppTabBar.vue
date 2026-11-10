<script setup lang="ts">
import { goTab, type TabKey } from '@/utils/nav'

defineProps<{ active: TabKey }>()

const tabs: { key: TabKey; label: string; icon: string }[] = [
  { key: 'home', label: '首页', icon: '🏠' },
  { key: 'transactions', label: '记账', icon: '📒' },
  { key: 'statistics', label: '统计', icon: '📊' },
  { key: 'settings', label: '设置', icon: '⚙️' },
  { key: 'help', label: '帮助', icon: '❓' },
]
</script>

<template>
  <view class="tabbar">
    <view
      v-for="tab in tabs"
      :key="tab.key"
      class="tabbar__item"
      :class="{ 'tabbar__item--active': tab.key === active }"
      @click="goTab(tab.key)"
    >
      <text class="tabbar__icon">{{ tab.icon }}</text>
      <text class="tabbar__label">{{ tab.label }}</text>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.tabbar {
  flex-shrink: 0;
  display: flex;
  align-items: stretch;
  background-color: $c-card;
  border-top: 2rpx solid $c-border;
  padding-bottom: env(safe-area-inset-bottom);
  z-index: 20;
}

.tabbar__item {
  flex: 1;
  height: 108rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  color: $c-text-3;
}

.tabbar__item--active {
  color: $c-primary;
}

.tabbar__icon {
  font-size: 40rpx;
  line-height: 1;
}

.tabbar__label {
  font-size: 20rpx;
  line-height: 1.2;
}
</style>
