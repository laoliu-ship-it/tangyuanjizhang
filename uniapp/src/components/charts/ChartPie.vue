<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import uCharts from '@qiun/ucharts'
import { CHART_COLORS, useChartCanvas } from './useChartCanvas'
import { compactMoney } from '@/utils/format'

interface PieItem {
  name: string
  value: number
  color?: string
}

const props = withDefaults(
  defineProps<{
    items: PieItem[]
    height?: number
  }>(),
  { height: 200 },
)

const { canvasId, wrapId, failed, render } = useChartCanvas(() => props.height)

const total = computed(() =>
  props.items.reduce((sum, item) => sum + (Number.isFinite(item.value) ? item.value : 0), 0),
)

function draw(): void {
  void render((ctx, width, height) => {
    new uCharts({
      type: 'ring',
      context: ctx,
      width,
      height,
      series: [
        {
          data: props.items.map((item, index) => ({
            name: item.name,
            value: Number(item.value) || 0,
            color: item.color ?? CHART_COLORS[index % CHART_COLORS.length],
          })),
        },
      ],
      animation: false,
      background: '#FFFFFF',
      color: CHART_COLORS,
      padding: [0, 0, 0, 0],
      dataLabel: false,
      legend: { show: false },
      title: {
        name: `¥${compactMoney(total.value)}`,
        fontSize: 15,
        color: '#1f2937',
      },
      subtitle: {
        name: '合计',
        fontSize: 10,
        color: '#9ca3af',
      },
      extra: {
        ring: {
          ringWidth: 16,
          activeOpacity: 0.5,
          activeRadius: 6,
          offsetAngle: 0,
          labelWidth: 10,
          border: true,
          borderWidth: 2,
          borderColor: '#FFFFFF',
        },
      },
    })
  })
}

onMounted(draw)

watch(
  () => [props.items, total.value],
  () => draw(),
  { deep: true },
)
</script>

<template>
  <view :id="wrapId" class="pie">
    <canvas
      v-if="!failed"
      :id="canvasId"
      :canvas-id="canvasId"
      class="pie__canvas"
      :style="{ height: `${height}px` }"
    />
    <view v-else class="pie__fallback">
      <text class="pie__fallback-title">图表暂不可用</text>
      <view v-for="(item, index) in items" :key="item.name" class="pie__fallback-row">
        <view class="pie__legend-name">
          <view
            class="pie__dot"
            :style="{ backgroundColor: item.color ?? CHART_COLORS[index % CHART_COLORS.length] }"
          />
          <text>{{ item.name }}</text>
        </view>
        <text class="pie__fallback-value num">{{ Number(item.value || 0).toFixed(2) }}</text>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.pie {
  width: 100%;
}

.pie__canvas {
  width: 100%;
}

.pie__fallback {
  padding: 12rpx;
  border-radius: $radius-sm;
  background-color: $c-bg;
}

.pie__fallback-title {
  display: block;
  font-size: $fs-xs;
  color: $c-text-3;
  margin-bottom: 8rpx;
}

.pie__fallback-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6rpx 0;
  font-size: $fs-sm;
  color: $c-text-2;
}

.pie__legend-name {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}

.pie__dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 50%;
  flex-shrink: 0;
}

.pie__fallback-value {
  font-weight: 600;
  color: $c-text;
}
</style>
