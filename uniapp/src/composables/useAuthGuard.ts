/**
 * 页面级登录守卫：无 token / 无租户 -> reLaunch 登录页。
 *
 * 用法一（只守卫）：setup 中调用 useAuthGuard()
 * 用法二（守卫 + 等会话就绪再拉数据）：
 *   onShow(async () => {
 *     if (!(await ensureSessionReady())) return
 *     await loadData()
 *   })
 */
import { onShow } from '@dcloudio/uni-app'
import { tenantApi } from '@/api/tenant'
import { useAuthStore } from '@/store/auth'
import { useMetaStore } from '@/store/meta'
import { useTenantStore } from '@/store/tenant'
import { goLogin } from '@/utils/nav'
import { toastError } from '@/utils/ui'

let sessionPromise: Promise<boolean> | null = null
let redirectingToLogin = false

function redirectLogin(): void {
  if (redirectingToLogin) return
  redirectingToLogin = true
  goLogin(true)
  setTimeout(() => {
    redirectingToLogin = false
  }, 1000)
}

/** 确保登录态和当前租户可用；不可用时跳转登录页并返回 false */
export function ensureSessionReady(): Promise<boolean> {
  const auth = useAuthStore()
  const tenant = useTenantStore()

  if (!auth.token) {
    redirectLogin()
    return Promise.resolve(false)
  }
  if (tenant.currentTenantId) return Promise.resolve(true)
  if (sessionPromise) return sessionPromise

  sessionPromise = (async () => {
    try {
      const list = await tenantApi.list()
      if (!Array.isArray(list) || list.length === 0) {
        auth.logout()
        tenant.reset()
        useMetaStore().reset()
        redirectLogin()
        return false
      }
      tenant.setTenants(list)
      return true
    } catch (err) {
      // 网络异常时不强制退出，避免误判；下次 onShow 会重试
      console.warn('[auth-guard] 拉取租户失败', err)
      toastError(err, '获取账本信息失败')
      return false
    } finally {
      sessionPromise = null
    }
  })()

  return sessionPromise
}

export function useAuthGuard(): void {
  onShow(() => {
    void ensureSessionReady()
  })
}
