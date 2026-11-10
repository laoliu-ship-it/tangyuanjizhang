<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    icon?: string
    name?: string
    size?: 'sm' | 'md' | 'lg'
  }>(),
  { icon: '📌', name: '', size: 'md' },
)

const sizeClass = computed(() => `cat-icon--${props.size}`)
</script>

<template>
  <view class="cat-icon" :class="sizeClass">
    <text class="cat-icon__emoji">{{ icon || '📌' }}</text>
  </view>
</template>

<style lang="scss" scoped>
.cat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 50%;
  background-color: $c-primary-bg;
}

.cat-icon--sm {
  width: 56rpx;
  height: 56rpx;
}
.cat-icon--md {
  width: 72rpx;
  height: 72rpx;
}
.cat-icon--lg {
  width: 96rpx;
  height: 96rpx;
}

.cat-icon__emoji {
  line-height: 1;
}

.cat-icon--sm .cat-icon__emoji {
  font-size: 28rpx;
}
.cat-icon--md .cat-icon__emoji {
  font-size: 36rpx;
}
.cat-icon--lg .cat-icon__emoji {
  font-size: 48rpx;
}
</style>
