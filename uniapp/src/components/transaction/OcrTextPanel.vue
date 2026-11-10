<script setup lang="ts">
import { computed } from 'vue'
import { haptic, toast } from '@/utils/ui'

const props = withDefaults(
  defineProps<{
    rawTexts: string[]
    usedLines?: number[]
  }>(),
  { usedLines: () => [] },
)

const emit = defineEmits<{
  (e: 'fill', payload: { action: 'amount' | 'merchant' | 'date' | 'note'; text: string }): void
}>()

const usedLineSet = computed(() => new Set(props.usedLines ?? []))

const ACTIONS: { action: 'amount' | 'merchant' | 'date' | 'note'; label: string }[] = [
  { action: 'amount', label: '填入金额' },
  { action: 'merchant', label: '填入商户' },
  { action: 'date', label: '填入日期' },
  { action: 'note', label: '填入备注' },
]

let lastLongPress = 0

function isNumericToken(token: string): boolean {
  return /^[+-]?\d+(\.\d+)?$/.test(token)
}

function openMenu(text: string): void {
  if (!text) return
  haptic()
  uni.showActionSheet({
    itemList: ACTIONS.map((item) => item.label),
    success: (res) => {
      const target = ACTIONS[res.tapIndex]
      if (target) emit('fill', { action: target.action, text })
    },
    fail: () => {
      // 用户取消
    },
  })
}

function onLongPress(text: string): void {
  lastLongPress = Date.now()
  openMenu(text)
}

function onTap(text: string): void {
  if (Date.now() - lastLongPress < 700) return
  openMenu(text)
}

function copyAll(): void {
  const text = props.rawTexts
    .flatMap((line) => line.split(/\s+/).filter(Boolean))
    .join(' ')
  if (!text) return
  uni.setClipboardData({
    data: text,
    success: () => toast('已复制识别内容'),
  })
}

function tokensOf(line: string): string[] {
  return line.split(/\s+/).filter(Boolean)
}
</script>

<template>
  <view v-if="rawTexts.length" class="ocr-panel">
    <view class="ocr-panel__head">
      <text class="ocr-panel__title">识别原文</text>
      <text class="ocr-panel__copy" @click="copyAll">📋 复制</text>
    </view>
    <text class="ocr-panel__hint">长按词条可填入金额 / 商户 / 日期 / 备注</text>

    <view class="ocr-panel__lines">
      <view v-for="(line, lineIndex) in rawTexts" :key="lineIndex" class="ocr-line">
        <view
          v-for="(token, tokenIndex) in tokensOf(line)"
          :key="`${lineIndex}_${tokenIndex}`"
          class="ocr-token"
          :class="{
            'ocr-token--used': usedLineSet.has(lineIndex),
            'ocr-token--num': isNumericToken(token),
          }"
          @longpress="onLongPress(token)"
          @click="onTap(token)"
        >
          <text>{{ token }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.ocr-panel {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  padding: 20rpx;
  border-radius: $radius-sm;
  background-color: #fff;
  border: 2rpx solid $c-border;
}

.ocr-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.ocr-panel__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.ocr-panel__copy {
  font-size: $fs-xs;
  color: $c-primary;
  padding: 4rpx 8rpx;
}

.ocr-panel__hint {
  font-size: 20rpx;
  color: $c-text-3;
}

.ocr-panel__lines {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
  max-height: 420rpx;
  overflow-y: auto;
}

.ocr-line {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
}

.ocr-token {
  display: inline-flex;
  align-items: center;
  padding: 4rpx 10rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 8rpx;
  font-size: $fs-xs;
  color: $c-text-2;
  background-color: $c-bg;
}

.ocr-token--num {
  border-color: #bfdbfe;
  background-color: $c-primary-bg;
  color: $c-primary;
}

.ocr-token--used {
  text-decoration: line-through;
  color: $c-text-3;
  background-color: #f3f4f6;
  border-color: $c-border;
}
</style>
