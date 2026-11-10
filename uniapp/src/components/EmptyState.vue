<script setup lang="ts">
withDefaults(
  defineProps<{
    icon?: string
    text?: string
    hint?: string
  }>(),
  { icon: '📭', text: '暂无数据', hint: '' },
)
</script>

<template>
  <view class="empty">
    <text class="empty__icon">{{ icon }}</text>
    <text class="empty__text">{{ text }}</text>
    <text v-if="hint" class="empty__hint">{{ hint }}</text>
    <view class="empty__action">
      <slot />
    </view>
  </view>
</template>

<style lang="scss" scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80rpx 40rpx;
}

.empty__icon {
  font-size: 72rpx;
  line-height: 1;
  margin-bottom: 20rpx;
  opacity: 0.8;
}

.empty__text {
  font-size: $fs-sm;
  color: $c-text-2;
}

.empty__hint {
  margin-top: 10rpx;
  font-size: $fs-xs;
  color: $c-text-3;
  text-align: center;
}

.empty__action {
  margin-top: 28rpx;
}
</style>
