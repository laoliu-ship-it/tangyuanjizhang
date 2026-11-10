<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    total: number
    loading?: boolean
    finished?: boolean
  }>(),
  { loading: false, finished: false },
)

const emit = defineEmits<{
  (e: 'loadmore'): void
}>()

function onTap(): void {
  if (props.loading || props.finished) return
  emit('loadmore')
}
</script>

<template>
  <view class="pager" @click="onTap">
    <text v-if="loading" class="pager__text">加载中…</text>
    <text v-else-if="total === 0" class="pager__text">暂无记录</text>
    <text v-else-if="finished" class="pager__text">共 {{ total }} 条，已全部加载</text>
    <text v-else class="pager__text">上拉或点击加载更多</text>
  </view>
</template>

<style lang="scss" scoped>
.pager {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 28rpx 0 12rpx;
}

.pager__text {
  font-size: $fs-xs;
  color: $c-text-3;
}
</style>
