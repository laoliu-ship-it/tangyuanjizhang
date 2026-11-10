<script setup lang="ts">
import CategoryIcon from '@/components/CategoryIcon.vue'
import AmountText from '@/components/AmountText.vue'
import type { Category, Transaction } from '@/api/types'
import { humanDateTime } from '@/utils/datetime'

const props = withDefaults(
  defineProps<{
    tx: Transaction
    category?: Category
    selectable?: boolean
    selected?: boolean
    selectMode?: boolean
  }>(),
  { category: undefined, selectable: false, selected: false, selectMode: false },
)

const emit = defineEmits<{
  (e: 'click', tx: Transaction): void
  (e: 'longpress', tx: Transaction): void
  (e: 'toggle', tx: Transaction): void
}>()

function categoryName(): string {
  return props.tx.category_name ?? props.category?.name ?? '未分类'
}

function onTap(): void {
  if (props.selectMode) emit('toggle', props.tx)
  else emit('click', props.tx)
}
</script>

<template>
  <view
    class="tx-card"
    :class="{ 'tx-card--selected': selected }"
    @click="onTap"
    @longpress="emit('longpress', tx)"
  >
    <view v-if="selectable && selectMode" class="tx-card__check" :class="{ 'tx-card__check--on': selected }">
      <text v-if="selected" class="tx-card__check-icon">✓</text>
    </view>

    <CategoryIcon :icon="category?.icon ?? '📌'" :name="categoryName()" size="md" />

    <view class="tx-card__body">
      <view class="tx-card__line">
        <text class="tx-card__category">{{ categoryName() }}</text>
        <AmountText :amount="tx.amount" :type="tx.type" size="sm" />
      </view>
      <view class="tx-card__meta">
        <text class="tx-card__time">{{ humanDateTime(tx.transaction_date) }}</text>
        <text v-if="tx.note" class="tx-card__note">· {{ tx.note }}</text>
      </view>
      <view class="tx-card__sub">
        <text class="tx-card__created">
          创建 {{ tx.created_at ? humanDateTime(tx.created_at) : '—' }}
        </text>
        <view class="tx-card__sub-right">
          <text v-if="tx.merchant_name" class="tx-card__merchant">{{ tx.merchant_name }}</text>
          <view class="tx-card__actions">
            <slot name="actions" />
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.tx-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 22rpx;
  border-radius: $radius;
  border: 2rpx solid $c-border;
  background-color: $c-card;
  box-shadow: $shadow-card;
}

.tx-card--selected {
  border-color: $c-primary;
  background-color: $c-primary-bg;
}

.tx-card__check {
  width: 40rpx;
  height: 40rpx;
  flex-shrink: 0;
  border-radius: 50%;
  border: 3rpx solid $c-border-2;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #fff;
}

.tx-card__check--on {
  border-color: $c-primary;
  background-color: $c-primary;
}

.tx-card__check-icon {
  font-size: 24rpx;
  color: #fff;
  font-weight: 700;
}

.tx-card__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.tx-card__line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.tx-card__category {
  font-size: $fs;
  font-weight: 500;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.tx-card__meta {
  display: flex;
  align-items: center;
  gap: 4rpx;
  min-width: 0;
}

.tx-card__time {
  flex-shrink: 0;
  font-size: $fs-xs;
  color: $c-text-3;
}

.tx-card__note {
  font-size: $fs-xs;
  color: $c-text-3;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.tx-card__sub {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.tx-card__created {
  font-size: 20rpx;
  color: #d1d5db;
}

.tx-card__sub-right {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}

.tx-card__merchant {
  font-size: 20rpx;
  color: $c-text-3;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.tx-card__actions {
  display: flex;
  align-items: center;
  gap: 16rpx;
}
</style>
