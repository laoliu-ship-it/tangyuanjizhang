/** uCharts 画布测量与渲染辅助（H5 / 小程序通用） */
import { getCurrentInstance, nextTick, ref, type Ref } from 'vue'

export const CHART_COLORS = [
  '#3b82f6',
  '#ef4444',
  '#22c55e',
  '#f59e0b',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
  '#84cc16',
  '#f97316',
  '#14b8a6',
]

let seed = 0

export interface UseChartCanvasReturn {
  canvasId: string
  wrapId: string
  failed: Ref<boolean>
  render: (
    builder: (ctx: UniNamespace.CanvasContext, width: number, height: number) => void,
  ) => Promise<void>
}

export function useChartCanvas(height: () => number): UseChartCanvasReturn {
  const instance = getCurrentInstance()
  seed += 1
  const canvasId = `chart_canvas_${Date.now()}_${seed}`
  const wrapId = `chart_wrap_${Date.now()}_${seed}`
  const widthValue = ref(0)
  const failed = ref(false)
  let retryCount = 0

  function measure(): Promise<number> {
    return new Promise((resolve) => {
      const query = uni.createSelectorQuery()
      if (instance?.proxy) query.in(instance.proxy)
      query
        .select(`#${wrapId}`)
        .boundingClientRect()
        .exec((res) => {
          const rect = (Array.isArray(res) ? res[0] : null) as
            | { width?: number; height?: number }
            | null
          const width = Number(rect?.width ?? 0)
          if (width > 0) {
            widthValue.value = width
            retryCount = 0
            resolve(width)
            return
          }
          if (retryCount < 6) {
            retryCount += 1
            setTimeout(() => resolve(measure()), 80)
            return
          }
          resolve(0)
        })
    })
  }

  async function render(
    builder: (ctx: UniNamespace.CanvasContext, width: number, height: number) => void,
  ): Promise<void> {
    const proxy = instance?.proxy
    if (!proxy) {
      failed.value = true
      return
    }
    try {
      // 先确保 canvas 节点存在（失败兜底时 canvas 被 v-if 移除，数据恢复后需重新挂载）
      failed.value = false
      await nextTick()
      const width = widthValue.value || (await measure())
      if (!width) {
        failed.value = true
        return
      }
      const context = uni.createCanvasContext(canvasId, proxy)
      builder(context, width, height())
      failed.value = false
    } catch (e) {
      console.error('[chart] 渲染失败', e)
      failed.value = true
    }
  }

  return {
    canvasId,
    wrapId,
    failed,
    render,
  }
}
