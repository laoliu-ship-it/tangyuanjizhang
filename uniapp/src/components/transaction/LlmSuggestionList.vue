<script setup lang="ts">
import { computed } from 'vue'
import type { LLMSuggestion } from '@/api/types'
import { money } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    suggestions: LLMSuggestion[]
    error?: string
    loading?: boolean
    appliedIndices?: number[]
  }>(),
  { error: '', loading: false, appliedIndices: () => [] },
)

const emit = defineEmits<{
  (e: 'apply', suggestion: LLMSuggestion, index: number): void
}>()

const visibleItems = computed(() =>
  props.suggestions
    .map((suggestion, index) => ({ suggestion, index }))
    .filter((item) => !props.appliedIndices.includes(item.index)),
)
</script>

<template>
  <view class="llm">
    <view v-if="loading" class="llm__loading">
      <view class="llm__spinner" />
      <text class="llm__loading-text">AI 正在分析账单…</text>
    </view>

    <view v-else-if="visibleItems.length" class="llm__list">
      <view class="llm__head">
        <text class="llm__title">AI 建议 · {{ visibleItems.length }} 笔</text>
        <text class="llm__hint">点击「接受」填入表单并暂存</text>
      </view>

      <view v-for="item in visibleItems" :key="item.index" class="llm-card">
        <view class="llm-card__icon">
          {{ item.suggestion.type === 'income' ? '💰' : '💸' }}
        </view>
        <view class="llm-card__body">
          <view class="llm-card__line">
            <text class="llm-card__category">
              {{ item.suggestion.category_hint || '未分类' }}
            </text>
            <text
              class="llm-card__amount num"
              :class="item.suggestion.type === 'income' ? 'llm-card__amount--in' : 'llm-card__amount--out'"
            >
              {{ item.suggestion.type === 'income' ? '+' : '-' }}¥{{
                money(Math.abs(item.suggestion.amount ?? 0))
              }}
            </text>
          </view>
          <view class="llm-card__line">
            <text class="llm-card__meta">
              {{ item.suggestion.merchant_name || '未识别商户' }}
              <text v-if="item.suggestion.date"> · {{ item.suggestion.date }}</text>
            </text>
            <view class="llm-card__accept" @click="emit('apply', item.suggestion, item.index)">
              接受
            </view>
          </view>
          <text v-if="item.suggestion.note" class="llm-card__note">
            {{ item.suggestion.note }}
          </text>
        </view>
      </view>
    </view>

    <text v-else-if="error" class="llm__error">AI 分析不可用：{{ error }}</text>
  </view>
</template>

<style lang="scss" scoped>
.llm {
  width: 100%;
}

.llm__loading {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 20rpx 8rpx;
}

.llm__spinner {
  width: 26rpx;
  height: 26rpx;
  border: 3rpx solid #c4b5fd;
  border-top-color: $c-purple;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.llm__loading-text {
  font-size: $fs-xs;
  color: $c-purple;
}

.llm__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12rpx;
  margin-bottom: 14rpx;
}

.llm__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.llm__hint {
  font-size: 20rpx;
  color: $c-text-3;
}

.llm__list {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}

.llm-card {
  display: flex;
  gap: 16rpx;
  padding: 20rpx;
  border: 2rpx solid #ede9fe;
  border-radius: $radius-sm;
  background-color: $c-purple-bg;
}

.llm-card__icon {
  flex-shrink: 0;
  width: 60rpx;
  height: 60rpx;
  border-radius: 50%;
  background-color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32rpx;
}

.llm-card__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.llm-card__line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.llm-card__category {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.llm-card__amount {
  flex-shrink: 0;
  font-size: $fs-sm;
  font-weight: 700;
}

.llm-card__amount--in {
  color: $c-income;
}
.llm-card__amount--out {
  color: $c-expense;
}

.llm-card__meta {
  font-size: $fs-xs;
  color: $c-text-2;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.llm-card__accept {
  flex-shrink: 0;
  padding: 6rpx 20rpx;
  border-radius: 999rpx;
  background-color: $c-purple;
  color: #fff;
  font-size: $fs-xs;
}

.llm-card__accept:active {
  opacity: 0.8;
}

.llm-card__note {
  font-size: $fs-xs;
  color: $c-text-3;
}

.llm__error {
  font-size: $fs-xs;
  color: $c-text-3;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
