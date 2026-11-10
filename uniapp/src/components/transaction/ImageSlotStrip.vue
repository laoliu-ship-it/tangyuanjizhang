<script setup lang="ts">
import type { ImageSlot } from '@/composables/useImageSlots'

withDefaults(
  defineProps<{
    slots: ImageSlot[]
    activeIndex: number
    max?: number
  }>(),
  { max: 9 },
)

const emit = defineEmits<{
  (e: 'select', index: number): void
  (e: 'remove', slotId: string): void
  (e: 'retry', slot: ImageSlot): void
  (e: 'add'): void
}>()

function onThumbTap(slot: ImageSlot, index: number): void {
  if (slot.status === 'error') {
    emit('retry', slot)
    return
  }
  emit('select', index)
}
</script>

<template>
  <scroll-view scroll-x class="strip" :show-scrollbar="false">
    <view class="strip__inner">
      <view
        v-for="(slot, index) in slots"
        :key="slot.id"
        class="thumb"
        :class="{
          'thumb--active': index === activeIndex,
          'thumb--saved': slot.saved,
        }"
        @click="onThumbTap(slot, index)"
      >
        <image class="thumb__img" :src="slot.previewUrl" mode="aspectFill" />

        <!-- 处理中 -->
        <view
          v-if="slot.status === 'compressing' || slot.status === 'ocr_recognizing'"
          class="thumb__overlay"
        >
          <view class="spinner" />
        </view>

        <!-- AI 分析中 -->
        <view v-else-if="slot.status === 'llm_recognizing'" class="thumb__overlay thumb__overlay--ai">
          <text class="thumb__badge-text">AI</text>
        </view>

        <!-- 等待 -->
        <view v-else-if="slot.status === 'pending'" class="thumb__overlay thumb__overlay--pending">
          <text class="thumb__badge-text">⏳</text>
        </view>

        <!-- 识别失败 -->
        <view v-else-if="slot.status === 'error'" class="thumb__overlay thumb__overlay--error">
          <text class="thumb__badge-text">重试</text>
        </view>

        <!-- AI 失败 -->
        <view
          v-else-if="slot.llmError && slot.llm.length === 0"
          class="thumb__overlay thumb__overlay--warn"
        >
          <text class="thumb__badge-text">AI!</text>
        </view>

        <!-- 已保存 -->
        <view v-if="slot.saved" class="thumb__saved">✓ 已保存</view>

        <!-- 删除 -->
        <view class="thumb__remove" @click.stop="emit('remove', slot.id)">✕</view>
      </view>

      <view v-if="slots.length < max" class="thumb thumb--add" @click="emit('add')">
        <text class="thumb__add-icon">+</text>
      </view>
    </view>
  </scroll-view>
</template>

<style lang="scss" scoped>
.strip {
  width: 100%;
  white-space: nowrap;
}

.strip__inner {
  display: inline-flex;
  align-items: center;
  gap: 14rpx;
  padding: 4rpx 2rpx;
}

.thumb {
  position: relative;
  width: 112rpx;
  height: 144rpx;
  flex-shrink: 0;
  border-radius: 14rpx;
  overflow: hidden;
  border: 4rpx solid transparent;
  background-color: $c-border;
}

.thumb--active {
  border-color: $c-primary;
  box-shadow: 0 6rpx 16rpx rgba(37, 99, 235, 0.22);
}

.thumb--saved {
  opacity: 0.82;
}

.thumb__img {
  width: 100%;
  height: 100%;
}

.thumb__overlay {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(17, 24, 39, 0.5);
}

.thumb__overlay--ai {
  background-color: rgba(124, 58, 237, 0.55);
}
.thumb__overlay--pending {
  background-color: rgba(17, 24, 39, 0.32);
}
.thumb__overlay--error {
  background-color: rgba(220, 38, 38, 0.6);
}
.thumb__overlay--warn {
  background-color: rgba(217, 119, 6, 0.6);
}

.thumb__badge-text {
  font-size: 20rpx;
  color: #fff;
  font-weight: 600;
}

.spinner {
  width: 30rpx;
  height: 30rpx;
  border: 4rpx solid #fff;
  border-top-color: transparent;
  border-radius: 50%;
  animation: rotate 0.8s linear infinite;
}

.thumb__saved {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 2rpx 0;
  text-align: center;
  font-size: 17rpx;
  color: #fff;
  background-color: rgba(22, 163, 74, 0.85);
}

.thumb__remove {
  position: absolute;
  top: 2rpx;
  right: 2rpx;
  width: 36rpx;
  height: 36rpx;
  border-radius: 50%;
  background-color: rgba(17, 24, 39, 0.55);
  color: #fff;
  font-size: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.thumb--add {
  border: 3rpx dashed $c-border-2;
  background-color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.thumb__add-icon {
  font-size: 56rpx;
  color: $c-text-3;
  line-height: 1;
}

@keyframes rotate {
  to {
    transform: rotate(360deg);
  }
}
</style>
