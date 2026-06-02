import { useAuthStore } from '../store/auth'
import { useTenantStore } from '../store/tenant'

export function useAuth() {
  const { token, userId, username, setAuth, logout: storeLogout } = useAuthStore()
  const { reset: resetTenant } = useTenantStore()

  const isLoggedIn = !!token

  const logout = () => {
    storeLogout()
    resetTenant()
  }

  return { token, userId, username, isLoggedIn, setAuth, logout }
}
