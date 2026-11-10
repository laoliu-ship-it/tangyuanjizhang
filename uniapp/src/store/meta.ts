import { defineStore } from 'pinia'
import { categoryApi } from '@/api/category'
import { merchantApi } from '@/api/merchant'
import { tenantApi } from '@/api/tenant'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import type { Category, Merchant } from '@/api/types'

export type TenantRoleKey = 'owner' | 'admin' | 'member'

interface MetaState {
  categories: Category[]
  merchants: Merchant[]
  role: TenantRoleKey
  loadedAt: number
  categoriesAt: number
  merchantsAt: number
  roleAt: number
}

/** 缓存有效期 5 分钟 */
const TTL = 5 * 60 * 1000

// 模块级去重：并发调用只发一次请求
let categoriesPromise: Promise<Category[]> | null = null
let merchantsPromise: Promise<Merchant[]> | null = null
let rolePromise: Promise<TenantRoleKey> | null = null

export const useMetaStore = defineStore('meta', {
  state: (): MetaState => ({
    categories: [],
    merchants: [],
    role: 'member',
    loadedAt: 0,
    categoriesAt: 0,
    merchantsAt: 0,
    roleAt: 0,
  }),

  actions: {
    /** 分类列表（跨页缓存） */
    async ensureCategories(force = false): Promise<Category[]> {
      const tenant = useTenantStore()
      if (!tenant.currentTenantId) return []
      if (!force && this.categories.length > 0 && Date.now() - this.categoriesAt < TTL) {
        return this.categories
      }
      if (categoriesPromise) return categoriesPromise
      categoriesPromise = categoryApi
        .list()
        .then((list) => {
          this.categories = Array.isArray(list) ? list : []
          this.categoriesAt = Date.now()
          this.loadedAt = Date.now()
          return this.categories
        })
        .finally(() => {
          categoriesPromise = null
        })
      return categoriesPromise
    },

    /** 商户列表（跨页缓存） */
    async ensureMerchants(force = false): Promise<Merchant[]> {
      const tenant = useTenantStore()
      if (!tenant.currentTenantId) return []
      if (!force && this.merchants.length > 0 && Date.now() - this.merchantsAt < TTL) {
        return this.merchants
      }
      if (merchantsPromise) return merchantsPromise
      merchantsPromise = merchantApi
        .list()
        .then((list) => {
          this.merchants = Array.isArray(list) ? list : []
          this.merchantsAt = Date.now()
          this.loadedAt = Date.now()
          return this.merchants
        })
        .finally(() => {
          merchantsPromise = null
        })
      return merchantsPromise
    },

    /** 当前用户在账本中的角色（owner/admin/member） */
    async ensureRole(force = false): Promise<TenantRoleKey> {
      const tenant = useTenantStore()
      const auth = useAuthStore()
      if (!tenant.currentTenantId || !auth.userId) return 'member'
      if (!force && this.roleAt > 0 && Date.now() - this.roleAt < TTL) {
        return this.role
      }
      if (rolePromise) return rolePromise
      rolePromise = tenantApi
        .getMembers(tenant.currentTenantId)
        .then((members) => {
          const me = members.find((m) => m.user_id === auth.userId)
          const raw = me?.role ?? 'member'
          const role: TenantRoleKey =
            raw === 'owner' ? 'owner' : raw === 'admin' ? 'admin' : 'member'
          this.role = role
          this.roleAt = Date.now()
          return role
        })
        .finally(() => {
          rolePromise = null
        })
      return rolePromise
    },

    setRole(role: TenantRoleKey) {
      this.role = role
      this.roleAt = Date.now()
    },

    /** 局部更新分类缓存（设置页增删改后调用，避免再发一次请求） */
    setCategories(list: Category[]) {
      this.categories = list
      this.categoriesAt = Date.now()
    },

    /** 局部更新商户缓存 */
    setMerchants(list: Merchant[]) {
      this.merchants = list
      this.merchantsAt = Date.now()
    },

    reset() {
      this.categories = []
      this.merchants = []
      this.role = 'member'
      this.loadedAt = 0
      this.categoriesAt = 0
      this.merchantsAt = 0
      this.roleAt = 0
      categoriesPromise = null
      merchantsPromise = null
      rolePromise = null
    },
  },
})
