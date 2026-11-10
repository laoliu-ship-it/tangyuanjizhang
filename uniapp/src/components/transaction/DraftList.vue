<script setup lang="ts">
import CategoryIcon from '@/components/CategoryIcon.vue'
import AmountText from '@/components/AmountText.vue'
import type { Category } from '@/api/types'
import type { SlotDraft } from '@/composables/useImageSlots'
import { humanDateTime } from '@/utils/datetime'

const props = defineProps<{
  drafts: SlotDraft[]
  categories: Category[]
}>()

const emit = defineEmits<{
  (e: 'edit', index: number): void
  (e: 'remove', index: number): void
}>()

function categoryOf(draft: SlotDraft): Category | undefined {
  return props.categories.find((category) => category.id === draft.categoryId)
}

function isInvalid(draft: SlotDraft): boolean {
  return !draft.categoryId || draft.categoryId <= 0
}
</script>

<template>
  <view v-if="drafts.length" class="drafts">
    <view class="drafts__head">
      <text class="drafts__title">暂存记录（{{ drafts.length }}）</text>
      <text class="drafts__hint">确认后统一提交</text>
    </view>

    <view
      v-for="(draft, index) in drafts"
      :key="draft.id"
      class="draft-card"
      :class="{ 'draft-card--invalid': isInvalid(draft) }"
    >
      <CategoryIcon :icon="categoryOf(draft)?.icon ?? '📌'" :name="draft.categoryName" size="sm" />
      <view class="draft-card__body">
        <view class="draft-card__line">
          <text class="draft-card__category">
            {{ draft.categoryName || '未分类' }}
            <text v-if="isInvalid(draft)" class="draft-card__warn">· 需选择分类</text>
          </text>
          <AmountText :amount="draft.amount" :type="draft.type" size="sm" />
        </view>
        <view class="draft-card__meta">
          <text class="draft-card__time">{{ humanDateTime(draft.date) }}</text>
          <text v-if="draft.merchantName" class="draft-card__merchant">
            · {{ draft.merchantName }}
          </text>
        </view>
        <text v-if="draft.note" class="draft-card__note">{{ draft.note }}</text>
      </view>
      <view class="draft-card__actions">
        <text class="draft-card__action" @click="emit('edit', index)">编辑</text>
        <text class="draft-card__action draft-card__action--danger" @click="emit('remove', index)">
          删除
        </text>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.drafts {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}

.drafts__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.drafts__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.drafts__hint {
  font-size: 20rpx;
  color: $c-text-3;
}

.draft-card {
  display: flex;
  align-items: center;
  gap: 14rpx;
  padding: 18rpx;
  border: 2rpx solid $c-border;
  border-radius: $radius-sm;
  background-color: #fff;
}

.draft-card--invalid {
  border-color: #fca5a5;
  background-color: #fef2f2;
}

.draft-card__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.draft-card__line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.draft-card__category {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.draft-card__warn {
  font-size: 20rpx;
  color: $c-expense;
}

.draft-card__meta {
  display: flex;
  align-items: center;
  gap: 4rpx;
  font-size: $fs-xs;
  color: $c-text-3;
}

.draft-card__merchant {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.draft-card__note {
  font-size: $fs-xs;
  color: $c-text-2;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.draft-card__actions {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.draft-card__action {
  font-size: $fs-xs;
  color: $c-primary;
}

.draft-card__action--danger {
  color: $c-text-3;
}
</style>
