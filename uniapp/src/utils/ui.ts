/** 统一交互封装：toast / confirm / loading（全端一致） */

export function toast(title: string, icon: 'none' | 'success' | 'error' = 'none'): void {
  if (!title) return
  uni.showToast({
    title,
    icon: icon === 'error' ? 'none' : icon,
    duration: 2000,
    mask: false,
  })
}

/** 统一错误提示：err instanceof Error -> message */
export function toastError(err: unknown, fallback = '操作失败'): void {
  const message = err instanceof Error ? err.message : fallback
  toast(message || fallback)
}

export function confirm(content: string, title = '提示'): Promise<boolean> {
  return new Promise((resolve) => {
    uni.showModal({
      title,
      content,
      confirmColor: '#2563eb',
      success: (res) => resolve(!!res.confirm),
      fail: () => resolve(false),
    })
  })
}

export function showLoading(title = '加载中'): void {
  uni.showLoading({ title, mask: true })
}

export function hideLoading(): void {
  uni.hideLoading()
}

/** 轻震动反馈（小程序 / App 生效，H5 端无感） */
export function haptic(): void {
  try {
    uni.vibrateShort({ type: 'light' })
  } catch (e) {
    // 忽略不支持振动的平台
  }
}
