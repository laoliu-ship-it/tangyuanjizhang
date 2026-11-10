/**
 * 微信虚拟支付运行时获取。
 * 小程序环境可能只注入 `wx` 自由变量（不一定挂到 globalThis），因此沿用 seccheck 的策略：
 * 优先真实的 `typeof wx !== 'undefined'`，再安全回退 `globalThis.wx`。
 * H5/App 通过条件编译直接返回 undefined，不参与支付能力判断。
 */
export interface WxVirtualPaymentOptions {
  signData: string
  mode: string
  paySig: string
  signature: string
  success: () => void
  fail: (err: { errMsg?: string }) => void
}

export interface WxRuntime {
  requestVirtualPayment?: (opts: WxVirtualPaymentOptions) => void
}

declare const wx: WxRuntime | undefined

export function getWxRuntime(): WxRuntime | undefined {
  // #ifdef MP-WEIXIN
  if (typeof wx !== 'undefined' && wx) return wx
  if (typeof globalThis !== 'undefined') {
    const runtime = (globalThis as unknown as { wx?: WxRuntime }).wx
    if (runtime) return runtime
  }
  // #endif
  return undefined
}

export function supportsVirtualPayment(): boolean {
  const runtime = getWxRuntime()
  return !!runtime && typeof runtime.requestVirtualPayment === 'function'
}
