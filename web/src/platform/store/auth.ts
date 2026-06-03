import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface PlatformAuthState {
  token: string | null
  adminId: number | null
  adminName: string | null
  setAuth: (token: string, adminId: number, adminName: string) => void
  logout: () => void
}

export const usePlatformAuthStore = create<PlatformAuthState>()(
  persist(
    set => ({
      token: null,
      adminId: null,
      adminName: null,
      setAuth: (token, adminId, adminName) => set({ token, adminId, adminName }),
      logout: () => set({ token: null, adminId: null, adminName: null }),
    }),
    { name: 'platform_auth' }
  )
)
