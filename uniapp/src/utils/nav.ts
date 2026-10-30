/**
 * 路由表 + 跳转封装。
 * 业务代码禁止直接拼 '/pages/xxx'，统一走这里（后续换 hash 模式只需改一处）。
 */

export const routes = {
  home: '/pages/home/index',
  login: '/pages/auth/login',
  register: '/pages/auth/register',
  transactions: '/pages/transactions/index',
  statistics: '/pages/statistics/index',
  settings: '/pages/settings/index',
  help: '/pages/help/index',
  merchantDetail: '/pages/merchant/detail',
  upload: '/pages/upload/index',
} as const

export type TabKey = 'home' | 'transactions' | 'statistics' | 'settings' | 'help'

/** 回首页（清空页面栈） */
export function goHome(): void {
  uni.reLaunch({ url: routes.home })
}

/**
 * 去登录页。
 * @param replace true = reLaunch（清栈，用于退出登录 / 失效跳转），false = navigateTo（注册返回等场景）
 */
export function goLogin(replace = false): void {
  if (replace) {
    uni.reLaunch({ url: routes.login })
    return
  }
  uni.navigateTo({ url: routes.login })
}

/** 切 tab（本项目用自定义 TabBar，页面非原生 tabBar，统一 reLaunch 保证栈干净） */
export function goTab(key: TabKey): void {
  uni.reLaunch({ url: routes[key] })
}

/** 返回上一页；无栈可退时回退到 fallback */
export function goBack(fallback: string = routes.home): void {
  const pages = typeof getCurrentPages === 'function' ? getCurrentPages() : []
  if (pages.length > 1) {
    uni.navigateBack({ delta: 1 })
    return
  }
  uni.reLaunch({ url: fallback })
}

/** 打开商户明细 */
export function goMerchantDetail(params: {
  id: number
  name: string
  start?: string
  end?: string
}): void {
  const query: string[] = [`id=${params.id}`, `name=${encodeURIComponent(params.name)}`]
  if (params.start) query.push(`start=${encodeURIComponent(params.start)}`)
  if (params.end) query.push(`end=${encodeURIComponent(params.end)}`)
  uni.navigateTo({ url: `${routes.merchantDetail}?${query.join('&')}` })
}
