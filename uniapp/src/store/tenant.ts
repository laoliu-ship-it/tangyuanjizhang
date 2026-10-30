import { defineStore } from 'pinia'
import { STORAGE_KEYS } from '@/config'
import { getJSON, setJSON, remove } from '@/utils/storage'
import type { Tenant } from '@/api/types'

interface TenantSnapshot {
  tenants: Tenant[]
  currentTenantId: number | null
}

export const useTenantStore = defineStore('tenant', {
  state: (): TenantSnapshot => ({ tenants: [], currentTenantId: null }),

  getters: {
    currentTenant(state): Tenant | null {
      return state.tenants.find((t) => t.id === state.currentTenantId) ?? null
    },
    currentTenantName(): string {
      return this.currentTenant?.name ?? ''
    },
  },

  actions: {
    restore() {
      const saved = getJSON<TenantSnapshot>(STORAGE_KEYS.tenant, {
        tenants: [],
        currentTenantId: null,
      })
      this.tenants = Array.isArray(saved.tenants) ? saved.tenants : []
      this.currentTenantId =
        saved.currentTenantId ??
        (this.tenants.length > 0 ? this.tenants[0].id : null)
    },

    setTenants(list: Tenant[]) {
      this.tenants = list
      const stillExists = list.some((t) => t.id === this.currentTenantId)
      if (!stillExists) {
        this.currentTenantId = list.length > 0 ? list[0].id : null
      }
      this.persist()
    },

    switchTenant(id: number) {
      if (id === this.currentTenantId) return
      this.currentTenantId = id
      this.persist()
    },

    updateTenantName(id: number, name: string) {
      const target = this.tenants.find((t) => t.id === id)
      if (target) target.name = name
      this.persist()
    },

    reset() {
      this.tenants = []
      this.currentTenantId = null
      remove(STORAGE_KEYS.tenant)
    },

    persist() {
      setJSON(STORAGE_KEYS.tenant, {
        tenants: this.tenants,
        currentTenantId: this.currentTenantId,
      } satisfies TenantSnapshot)
    },
  },
})
