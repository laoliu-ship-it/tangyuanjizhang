<script setup lang="ts">
import { computed } from 'vue'
import { money } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    amount: number
    type?: 'income' | 'expense'
    size?: 'sm' | 'md' | 'lg'
    signed?: boolean
  }>(),
  { type: undefined, size: 'md', signed: false },
)

const text = computed(() => {
  const value = Number.isFinite(props.amount) ? props.amount : 0
  if (props.type) {
    return `${props.type === 'income' ? '+' : '-'}¥${money(Math.abs(value))}`
  }
  if (props.signed) {
    return `${value < 0 ? '-' : '+'}¥${money(Math.abs(value))}`
  }
  return `${value < 0 ? '-' : ''}¥${money(Math.abs(value))}`
})

const tone = computed(() => {
  if (props.type) return props.type
  if (props.signed) return props.amount < 0 ? 'expense' : 'income'
  return 'neutral'
})

const sizeClass = computed(() => `amount--${props.size}`)
</script>

<template>
  <text class="amount num" :class="[sizeClass, `amount--${tone}`]">{{ text }}</text>
</template>

<style lang="scss" scoped>
.amount {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.amount--sm {
  font-size: $fs-sm;
}
.amount--md {
  font-size: $fs-lg;
}
.amount--lg {
  font-size: $fs-xl;
  font-weight: 700;
}

.amount--income {
  color: $c-income;
}
.amount--expense {
  color: $c-expense;
}
.amount--neutral {
  color: $c-text;
}
</style>
