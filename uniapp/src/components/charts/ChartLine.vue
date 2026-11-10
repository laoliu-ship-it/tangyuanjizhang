<script setup lang="ts">
import { onMounted, watch } from 'vue'
import uCharts from '@qiun/ucharts'
import { CHART_COLORS, useChartCanvas } from './useChartCanvas'

interface ChartSeries {
  name: string
  data: number[]
  color?: string
}

const props = withDefaults(
  defineProps<{
    categories: string[]
    series: ChartSeries[]
    height?: number
  }>(),
  { height: 220 },
)

const { canvasId, wrapId, failed, render } = useChartCanvas(() => props.height)

const seriesTotal = (data: number[]): number =>
  data.reduce((sum, value) => sum + (Number.isFinite(value) ? value : 0), 0)

function draw(): void {
  void render((ctx, width, height) => {
    const colors = props.series.map((item, index) => item.color ?? CHART_COLORS[index % CHART_COLORS.length])
    new uCharts({
      type: 'line',
      context: ctx,
      width,
      height,
      categories: props.categories,
      series: props.series.map((item, index) => ({
        name: item.name,
        data: item.data,
        color: item.color ?? colors[index],
      })),
      animation: false,
      background: '#FFFFFF',
      color: colors,
      padding: [16, 12, 6, 4],
      legend: {
        show: true,
        position: 'top',
        float: 'right',
        fontSize: 11,
        lineHeight: 11,
        fontColor: '#6b7280',
        itemGap: 14,
      },
      xAxis: {
        disableGrid: true,
        fontSize: 10,
        fontColor: '#9ca3af',
        axisLineColor: '#e5e7eb',
        labelCount: props.categories.length > 16 ? 6 : props.categories.length,
        marginTop: 6,
      },
      yAxis: {
        gridColor: '#f3f4f6',
        dashLength: 4,
        fontColor: '#9ca3af',
        fontSize: 10,
        data: [{ min: 0 }],
      },
      extra: {
        line: {
          type: 'curve',
          width: 2,
          activeType: 'hollow',
        },
      },
    })
  })
}

onMounted(draw)

watch(
  () => [props.categories, props.series],
  () => draw(),
  { deep: true },
)
</script>

<template>
  <view :id="wrapId" class="chart">
    <canvas
      v-if="!failed"
      :id="canvasId"
      :canvas-id="canvasId"
      class="chart__canvas"
      :style="{ height: `${height}px` }"
    />
    <view v-else class="chart__fallback">
      <text class="chart__fallback-title">图表暂不可用</text>
      <view v-for="item in series" :key="item.name" class="chart__fallback-row">
        <text class="chart__fallback-label">{{ item.name }}</text>
        <text class="chart__fallback-value num">{{ seriesTotal(item.data).toFixed(2) }}</text>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.chart {
  width: 100%;
}

.chart__canvas {
  width: 100%;
}

.chart__fallback {
  padding: 20rpx 12rpx;
  border-radius: $radius-sm;
  background-color: $c-bg;
}

.chart__fallback-title {
  display: block;
  font-size: $fs-xs;
  color: $c-text-3;
  margin-bottom: 12rpx;
}

.chart__fallback-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8rpx 0;
  font-size: $fs-sm;
  color: $c-text-2;
}

.chart__fallback-value {
  font-weight: 600;
  color: $c-text;
}
</style>
