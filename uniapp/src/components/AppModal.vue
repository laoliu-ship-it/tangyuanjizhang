<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    visible: boolean
    title?: string
    position?: 'bottom' | 'center'
    closeOnMask?: boolean
    showClose?: boolean
  }>(),
  {
    title: '',
    position: 'bottom',
    closeOnMask: true,
    showClose: true,
  },
)

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'close'): void
}>()

function close(): void {
  emit('update:visible', false)
  emit('close')
}

function onMaskTap(): void {
  if (props.closeOnMask) close()
}
</script>

<template>
  <view v-if="visible" class="modal-mask" @touchmove.stop.prevent @click="onMaskTap">
    <view
      class="modal-panel"
      :class="position === 'bottom' ? 'modal-panel--bottom' : 'modal-panel--center'"
      @click.stop
    >
      <view v-if="title || showClose" class="modal-head">
        <text class="modal-title">{{ title }}</text>
        <view v-if="showClose" class="modal-close" @click="close">✕</view>
      </view>
      <scroll-view scroll-y class="modal-scroll">
        <view class="modal-body">
          <slot />
        </view>
      </scroll-view>
      <view v-if="$slots.footer" class="modal-footer">
        <slot name="footer" />
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.modal-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 150;
  display: flex;
  background-color: rgba(17, 24, 39, 0.48);
  animation: mask-in 0.18s ease-out;
}

.modal-panel {
  display: flex;
  flex-direction: column;
  background-color: $c-card;
  overflow: hidden;
}

.modal-panel--bottom {
  margin-top: auto;
  width: 100%;
  max-height: 85vh;
  border-radius: $radius-lg $radius-lg 0 0;
  padding-bottom: env(safe-area-inset-bottom);
  animation: slide-up 0.24s ease-out;
}

.modal-panel--center {
  margin: auto;
  width: 640rpx;
  max-width: 90vw;
  max-height: 82vh;
  border-radius: $radius-lg;
  animation: pop-in 0.2s ease-out;
}

.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  padding: 28rpx 32rpx 20rpx;
  border-bottom: 2rpx solid $c-border;
}

.modal-title {
  font-size: $fs-lg;
  font-weight: 600;
  color: $c-text;
}

.modal-close {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32rpx;
  color: $c-text-3;
  border-radius: 50%;
}

.modal-scroll {
  flex: 1;
  min-height: 0;
  max-height: 70vh;
}

.modal-body {
  padding: 24rpx 32rpx;
}

.modal-footer {
  flex-shrink: 0;
  padding: 20rpx 32rpx 28rpx;
  border-top: 2rpx solid $c-border;
}

@media (min-width: 960px) {
  .modal-panel--bottom {
    width: 750px;
    margin: auto auto 0;
  }

  .modal-panel--center {
    width: 480px;
  }
}

@keyframes mask-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slide-up {
  from {
    transform: translateY(100%);
  }
  to {
    transform: translateY(0);
  }
}

@keyframes pop-in {
  from {
    transform: scale(0.94);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}
</style>
