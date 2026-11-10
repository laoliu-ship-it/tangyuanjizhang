<script setup lang="ts">
import { computed } from 'vue'
import AmountText from '@/components/AmountText.vue'
import type { Category, Merchant, TxType } from '@/api/types'
import type { SlotForm } from '@/composables/useImageSlots'

const props = withDefaults(
  defineProps<{
    form: SlotForm
    categories: Category[]
    merchants?: Merchant[]
    disabled?: boolean
    readonly?: boolean
  }>(),
  { merchants: () => [], disabled: false, readonly: false },
)

const emit = defineEmits<{
  (e: 'patch', patch: Partial<SlotForm>): void
}>()

const filteredCategories = computed(() =>
  props.categories.filter((category) => category.type === props.form.type),
)

const datePart = computed(() => (props.form.date || '').slice(0, 10) || '')
const timePart = computed(() =>
  props.form.date && props.form.date.length >= 16 ? props.form.date.slice(11, 16) : '00:00',
)

const currentCategory = computed(() =>
  props.categories.find((category) => category.id === props.form.categoryId),
)

const suggestions = computed(() => {
  const keyword = props.form.merchantName.trim().toLowerCase()
  if (!keyword || props.form.merchantId) return []
  return props.merchants
    .filter((merchant) => merchant.name.toLowerCase().includes(keyword))
    .slice(0, 5)
})

function eventValue(e: Event): string {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  return detail?.value === undefined ? '' : String(detail.value)
}

function onType(type: TxType): void {
  if (props.disabled || props.readonly) return
  if (props.form.type === type) return
  emit('patch', { type, categoryId: 0 })
}

function onAmount(e: Event): void {
  emit('patch', { amount: eventValue(e) })
}

function onMerchant(e: Event): void {
  emit('patch', { merchantName: eventValue(e), merchantId: 0 })
}

function onNote(e: Event): void {
  emit('patch', { note: eventValue(e) })
}

function onDate(e: Event): void {
  emit('patch', { date: `${eventValue(e)}T${timePart.value}` })
}

function onTime(e: Event): void {
  emit('patch', { date: `${datePart.value}T${eventValue(e)}` })
}

function onCategory(category: Category): void {
  if (props.disabled || props.readonly) return
  emit('patch', { categoryId: category.id })
}

function onPickMerchant(merchant: Merchant): void {
  emit('patch', { merchantId: merchant.id, merchantName: merchant.name })
}
</script>

<template>
  <view v-if="readonly" class="readonly">
    <view class="readonly__amount">
      <AmountText :amount="Number(form.amount || 0)" :type="form.type" size="lg" />
      <text class="readonly__type">{{ form.type === 'income' ? '收入' : '支出' }}</text>
    </view>
    <view class="readonly__row">
      <text class="readonly__label">分类</text>
      <text class="readonly__value">
        {{ currentCategory?.icon ?? '📌' }} {{ currentCategory?.name ?? '未分类' }}
      </text>
    </view>
    <view v-if="form.merchantName" class="readonly__row">
      <text class="readonly__label">商户</text>
      <text class="readonly__value">{{ form.merchantName }}</text>
    </view>
    <view class="readonly__row">
      <text class="readonly__label">日期</text>
      <text class="readonly__value">{{ form.date.replace('T', ' ') }}</text>
    </view>
    <view v-if="form.note" class="readonly__row">
      <text class="readonly__label">备注</text>
      <text class="readonly__value readonly__value--note">{{ form.note }}</text>
    </view>
  </view>

  <view v-else class="editor">
    <!-- 支出 / 收入 -->
    <view class="type-toggle">
      <view
        class="type-toggle__item"
        :class="{ 'type-toggle__item--expense': form.type === 'expense' }"
        @click="onType('expense')"
      >
        支出
      </view>
      <view
        class="type-toggle__item"
        :class="{ 'type-toggle__item--income': form.type === 'income' }"
        @click="onType('income')"
      >
        收入
      </view>
    </view>

    <!-- 金额 -->
    <view class="amount-box">
      <text class="amount-box__currency">¥</text>
      <input
        class="amount-box__input"
        type="digit"
        :value="form.amount"
        :disabled="disabled"
        placeholder="0.00"
        placeholder-class="amount-box__placeholder"
        @input="onAmount"
      />
    </view>

    <!-- 日期时间 -->
    <view class="field">
      <text class="field__label">日期时间</text>
      <view class="date-row">
        <picker mode="date" :value="datePart" :disabled="disabled" @change="onDate">
          <view class="picker-box">{{ datePart }}</view>
        </picker>
        <picker mode="time" :value="timePart" :disabled="disabled" @change="onTime">
          <view class="picker-box">{{ timePart }}</view>
        </picker>
      </view>
    </view>

    <!-- 分类 -->
    <view class="field">
      <text class="field__label">分类</text>
      <view class="cat-grid">
        <view
          v-for="category in filteredCategories"
          :key="category.id"
          class="cat-item"
          :class="{ 'cat-item--active': form.categoryId === category.id }"
          @click="onCategory(category)"
        >
          <text class="cat-item__icon">{{ category.icon || '📌' }}</text>
          <text class="cat-item__name">{{ category.name }}</text>
        </view>
        <view v-if="filteredCategories.length === 0" class="cat-empty">
          暂无{{ form.type === 'expense' ? '支出' : '收入' }}分类，请到「设置 - 分类」添加
        </view>
      </view>
    </view>

    <!-- 商户 -->
    <view class="field field--relative">
      <text class="field__label">商户</text>
      <input
        class="input"
        :value="form.merchantName"
        :disabled="disabled"
        placeholder="输入商户名称（可选）"
        @input="onMerchant"
      />
      <view v-if="suggestions.length" class="suggest">
        <view
          v-for="merchant in suggestions"
          :key="merchant.id"
          class="suggest__item"
          @click="onPickMerchant(merchant)"
        >
          {{ merchant.name }}
        </view>
      </view>
    </view>

    <!-- 备注 -->
    <view class="field">
      <text class="field__label">备注</text>
      <input
        class="input"
        :value="form.note"
        :disabled="disabled"
        placeholder="添加备注（可选）"
        maxlength="200"
        @input="onNote"
      />
    </view>
  </view>
</template>

<style lang="scss" scoped>
.editor {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.type-toggle {
  display: flex;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  overflow: hidden;
}

.type-toggle__item {
  flex: 1;
  padding: 18rpx 0;
  text-align: center;
  font-size: $fs;
  font-weight: 500;
  color: $c-text-2;
  background-color: #fff;
}

.type-toggle__item--expense {
  background-color: $c-expense;
  color: #fff;
}

.type-toggle__item--income {
  background-color: $c-income;
  color: #fff;
}

.amount-box {
  display: flex;
  align-items: center;
  gap: 10rpx;
  padding: 20rpx 24rpx;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
}

.amount-box__currency {
  font-size: 36rpx;
  font-weight: 600;
  color: $c-text-2;
}

.amount-box__input {
  flex: 1;
  font-size: 52rpx;
  font-weight: 700;
  color: $c-text;
  height: 72rpx;
  line-height: 72rpx;
}

.amount-box__placeholder {
  color: $c-text-3;
  font-weight: 400;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.field--relative {
  position: relative;
}

.field__label {
  font-size: $fs-sm;
  color: $c-text-2;
}

.date-row {
  display: flex;
  gap: 16rpx;
}

.picker-box {
  min-width: 200rpx;
  height: 76rpx;
  padding: 0 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
  font-size: $fs;
  color: $c-text;
}

.cat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12rpx;
}

.cat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: 14rpx 6rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 14rpx;
  background-color: #fff;
}

.cat-item--active {
  border-color: $c-primary;
  background-color: $c-primary-bg;
}

.cat-item__icon {
  font-size: 38rpx;
  line-height: 1.1;
}

.cat-item__name {
  width: 100%;
  text-align: center;
  font-size: 20rpx;
  color: $c-text-2;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.cat-item--active .cat-item__name {
  color: $c-primary;
}

.cat-empty {
  grid-column: span 4;
  padding: 24rpx 0;
  text-align: center;
  font-size: $fs-xs;
  color: $c-text-3;
}

.suggest {
  position: absolute;
  top: 116rpx;
  left: 0;
  right: 0;
  z-index: 10;
  max-height: 320rpx;
  overflow-y: auto;
  background-color: #fff;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  box-shadow: $shadow-float;
}

.suggest__item {
  padding: 20rpx 24rpx;
  font-size: $fs-sm;
  color: $c-text;
  border-bottom: 2rpx solid $c-border;
}

.suggest__item:last-child {
  border-bottom: none;
}

.suggest__item:active {
  background-color: $c-primary-bg;
  color: $c-primary;
}

.readonly {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.readonly__amount {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6rpx;
  padding: 32rpx;
  border-radius: $radius;
  background-color: $c-bg;
}

.readonly__type {
  font-size: $fs-xs;
  color: $c-text-3;
}

.readonly__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24rpx;
  padding-bottom: 18rpx;
  border-bottom: 2rpx solid $c-border;
}

.readonly__label {
  flex-shrink: 0;
  font-size: $fs-sm;
  color: $c-text-3;
}

.readonly__value {
  font-size: $fs-sm;
  color: $c-text;
  text-align: right;
}

.readonly__value--note {
  word-break: break-all;
}
</style>
