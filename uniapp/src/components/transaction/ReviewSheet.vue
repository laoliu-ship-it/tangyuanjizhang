<script setup lang="ts">
import { computed } from 'vue'
import AppModal from '@/components/AppModal.vue'
import CategoryIcon from '@/components/CategoryIcon.vue'
import AmountText from '@/components/AmountText.vue'
import type { Category } from '@/api/types'
import type { SlotDraft } from '@/composables/useImageSlots'
import { humanDateTime } from '@/utils/datetime'
import { money } from '@/utils/format'

const props = defineProps<{
  visible: boolean
  drafts: SlotDraft[]
  categories: Category[]
  submitting?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'confirm'): void
  (e: 'edit', index: number): void
  (e: 'remove', index: number): void
}>()

const visibleModel = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const invalidCount = computed(
  () => props.drafts.filter((draft) => !draft.categoryId || draft.categoryId <= 0).length,
)

const net = computed(() =>
  props.drafts.reduce(
    (sum, draft) => sum + (draft.type === 'expense' ? -draft.amount : draft.amount),
    0,
  ),
)

function categoryOf(draft: SlotDraft): Category | undefined {
  return props.categories.find((category) => category.id === draft.categoryId)
}
</script>

<template>
  <AppModal v-model:visible="visibleModel" title="确认提交" position="bottom">
    <view class="review">
      <view class="review__summary">
        <text class="review__count">已暂存 {{ drafts.length }} 笔</text>
        <text class="review__net num" :class="net >= 0 ? 'review__net--in' : 'review__net--out'">
          {{ net >= 0 ? '+' : '-' }}¥{{ money(Math.abs(net)) }}
        </text>
      </view>

      <view class="review__list">
        <view
          v-for="(draft, index) in drafts"
          :key="draft.id"
          class="review-item"
          :class="{ 'review-item--invalid': !draft.categoryId || draft.categoryId <= 0 }"
        >
          <CategoryIcon :icon="categoryOf(draft)?.icon ?? '📌'" size="sm" />
          <view class="review-item__body">
            <view class="review-item__line">
              <text class="review-item__category">
                {{ draft.categoryName || '未分类' }}
              </text>
              <AmountText :amount="draft.amount" :type="draft.type" size="sm" />
            </view>
            <view class="review-item__meta">
              <text>{{ humanDateTime(draft.date) }}</text>
              <text v-if="draft.merchantName"> · {{ draft.merchantName }}</text>
              <text v-if="draft.note"> · {{ draft.note }}</text>
            </view>
            <text v-if="!draft.categoryId || draft.categoryId <= 0" class="review-item__warn">
              ⚠ 需要选择有效分类
            </text>
          </view>
          <view class="review-item__actions">
            <text class="review-item__action" @click="emit('edit', index)">编辑</text>
            <text class="review-item__action review-item__action--danger" @click="emit('remove', index)">
              删除
            </text>
          </view>
        </view>
      </view>

      <text v-if="invalidCount > 0" class="review__warn">
        ⚠ 有 {{ invalidCount }} 笔未匹配有效分类，请编辑后再提交
      </text>
    </view>

    <template #footer>
      <view class="review__footer">
        <view class="btn review__btn" @click="visibleModel = false">继续记账</view>
        <view
          class="btn btn-primary review__btn"
          :class="{ 'btn-disabled': invalidCount > 0 || submitting }"
          @click="emit('confirm')"
        >
          {{
            submitting
              ? '提交中…'
              : invalidCount > 0
                ? '请先修正分类'
                : `确认提交 ${drafts.length} 笔`
          }}
        </view>
      </view>
    </template>
  </AppModal>
</template>

<style lang="scss" scoped>
.review {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.review__summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 24rpx;
  border-radius: $radius-sm;
  background-color: $c-primary-bg;
}

.review__count {
  font-size: $fs-sm;
  color: $c-text-2;
}

.review__net {
  font-size: $fs-lg;
  font-weight: 700;
}

.review__net--in {
  color: $c-income;
}
.review__net--out {
  color: $c-expense;
}

.review__list {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.review-item {
  display: flex;
  align-items: center;
  gap: 14rpx;
  padding: 16rpx;
  border-radius: $radius-sm;
  background-color: $c-bg;
}

.review-item--invalid {
  background-color: $c-expense-bg;
  border: 2rpx solid #fca5a5;
}

.review-item__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.review-item__line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.review-item__category {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
}

.review-item__meta {
  font-size: $fs-xs;
  color: $c-text-3;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.review-item__warn {
  font-size: $fs-xs;
  color: $c-expense;
}

.review-item__actions {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.review-item__action {
  font-size: $fs-xs;
  color: $c-primary;
}

.review-item__action--danger {
  color: $c-text-3;
}

.review__warn {
  font-size: $fs-xs;
  color: $c-expense;
  text-align: center;
}

.review__footer {
  display: flex;
  gap: 16rpx;
}

.review__btn {
  flex: 1;
  height: 88rpx;
}
</style>
