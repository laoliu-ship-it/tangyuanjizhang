/**
 * 虚拟支付购买能力纯函数：页面与 node:test 共用，避免 computed Ref 判断出错。
 * - Android：具备微信虚拟支付接口、管理员、非演示、已启用、有可售套餐即可购买。
 * - iOS：还必须已开通 IAP，且微信版本 >= 8.0.68。
 * - H5/App：supportsVirtualPayment=false，不可拉起支付。
 */
export interface BillingPurchaseCapabilityInput {
  enabled: boolean
  platform: string
  iapEnabled: boolean
  wechatVersion: string
  supportsVirtualPayment: boolean
  isAdmin: boolean
  isDemo: boolean
  packageCount: number
}

export function compareVersion(v1: string, v2: string): number {
  const a = String(v1 || '')
    .split('.')
    .map((v) => Number.parseInt(v, 10) || 0)
  const b = String(v2 || '')
    .split('.')
    .map((v) => Number.parseInt(v, 10) || 0)
  const len = Math.max(a.length, b.length)
  for (let i = 0; i < len; i += 1) {
    const x = a[i] ?? 0
    const y = b[i] ?? 0
    if (x > y) return 1
    if (x < y) return -1
  }
  return 0
}

export function isIOSWeChatSupported(version: string): boolean {
  return compareVersion(version || '0', '8.0.68') >= 0
}

export function canPurchaseBilling(input: BillingPurchaseCapabilityInput): boolean {
  if (!input.enabled || input.isDemo || !input.isAdmin) return false
  if (!input.supportsVirtualPayment || input.packageCount <= 0) return false
  if (String(input.platform).toLowerCase() === 'ios') {
    return input.iapEnabled && isIOSWeChatSupported(input.wechatVersion)
  }
  return true
}
