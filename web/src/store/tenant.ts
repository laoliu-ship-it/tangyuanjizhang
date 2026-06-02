import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface Tenant {
  id: number
  name: string
  owner_id?: number
}

interface TenantState {
  currentTenantId: number | null
  tenants: Tenant[]
  setTenants: (tenants: Tenant[]) => void
  switchTenant: (id: number) => void
  reset: () => void
}

export const useTenantStore = create<TenantState>()(
  persist(
    set => ({
      currentTenantId: null,
      tenants: [],
      setTenants: (tenants) => set({ tenants, currentTenantId: tenants[0]?.id ?? null }),
      switchTenant: (id) => set({ currentTenantId: id }),
      reset: () => set({ currentTenantId: null, tenants: [] }),
    }),
    { name: 'tenant' }
  )
)
