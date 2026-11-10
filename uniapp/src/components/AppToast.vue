<script setup lang="ts">
withDefaults(
  defineProps<{
    text: string
    error?: boolean
  }>(),
  { error: false },
)
</script>

<template>
  <view v-if="text" class="app-toast" :class="{ 'app-toast--error': error }">
    <text class="app-toast__icon">{{ error ? '✕' : '✓' }}</text>
    <text class="app-toast__text">{{ text }}</text>
  </view>
</template>

<style lang="scss" scoped>
.app-toast {
  position: fixed;
  top: calc(env(safe-area-inset-top) + 32rpx);
  left: 50%;
  transform: translateX(-50%);
  z-index: 200;
  display: flex;
  align-items: center;
  gap: 12rpx;
  max-width: 620rpx;
  padding: 20rpx 32rpx;
  border-radius: 999rpx;
  background-color: rgba(17, 24, 39, 0.92);
  box-shadow: $shadow-float;
  pointer-events: none;
}

.app-toast--error {
  background-color: rgba(220, 38, 38, 0.95);
}

.app-toast__icon {
  font-size: $fs-sm;
  color: #fff;
  font-weight: 700;
}

.app-toast__text {
  font-size: $fs-sm;
  color: #fff;
  line-height: 1.4;
}
</style>
