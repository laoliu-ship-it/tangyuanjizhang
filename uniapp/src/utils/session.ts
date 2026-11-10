import { tenantApi } from '@/api/tenant'
import type { AuthResult } from '@/api/types'
import { useAuthStore } from '@/store/auth'
import { useMetaStore } from '@/store/meta'
import { useTenantStore } from '@/store/tenant'
import { goHome } from '@/utils/nav'

/**
 * 登录/注册/微信登录成功后统一落库：
 * 写入 auth → 拉取账本列表（失败时用接口返回的 tenants 兜底）→ 重置 meta → 回首页。
 */
export async function applyAuthResult(result: AuthResult, options?: { go?: boolean }): Promise<void> {
  const authStore = useAuthStore()
  const tenantStore = useTenantStore()
  const metaStore = useMetaStore()

  authStore.setAuth(result.token, result.user_id, result.username, !!result.demo)

  try {
    const list = await tenantApi.list()
    tenantStore.setTenants(Array.isArray(list) ? list : [])
  } catch (e) {
    const fallback = (result.tenants ?? []).map((tenant) => ({
      id: tenant.id,
      name: tenant.name,
      owner_id: 0,
    }))
    tenantStore.setTenants(fallback)
  }
  metaStore.reset()
  if (options?.go !== false) goHome()
}
