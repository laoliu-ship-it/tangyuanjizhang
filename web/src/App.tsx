import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import { useTenantStore } from './store/tenant'
import { usePlatformAuthStore } from './platform/store/auth'
import AppLayout from './layouts/AppLayout'
import PlatformLayout from './platform/layouts/PlatformLayout'
import Login from './pages/Auth/Login'
import Register from './pages/Auth/Register'
import Dashboard from './pages/Dashboard/index'
import Transactions from './pages/Transactions/index'
import UploadOCR from './pages/UploadOCR/index'
import Statistics from './pages/Statistics/index'
import TenantSettings from './pages/TenantSettings/index'
import Help from './pages/Help/index'
import PlatformLogin from './platform/pages/Login'
import PlatformDashboard from './platform/pages/Dashboard'
import PlatformUsers from './platform/pages/Users'
import PlatformUserDetail from './platform/pages/UserDetail'
import PlatformSettings from './platform/pages/Settings'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore(s => s.token)
  const currentTenantId = useTenantStore(s => s.currentTenantId)

  if (!token) {
    return <Navigate to="/login" replace />
  }
  if (!currentTenantId) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function GuestOnly({ children }: { children: React.ReactNode }) {
  const token = useAuthStore(s => s.token)
  const currentTenantId = useTenantStore(s => s.currentTenantId)

  if (token && currentTenantId) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

function RequirePlatformAuth({ children }: { children: React.ReactNode }) {
  const token = usePlatformAuthStore(s => s.token)

  if (!token) {
    return <Navigate to="/platform/login" replace />
  }
  return <>{children}</>
}

function GuestOnlyPlatform({ children }: { children: React.ReactNode }) {
  const token = usePlatformAuthStore(s => s.token)

  if (token) {
    return <Navigate to="/platform" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* 租户端认证路由 */}
        <Route
          path="/login"
          element={
            <GuestOnly>
              <Login />
            </GuestOnly>
          }
        />
        <Route
          path="/register"
          element={
            <GuestOnly>
              <Register />
            </GuestOnly>
          }
        />

        {/* 平台管理端认证路由 */}
        <Route
          path="/platform/login"
          element={
            <GuestOnlyPlatform>
              <PlatformLogin />
            </GuestOnlyPlatform>
          }
        />

        {/* 租户端主路由 */}
        <Route
          path="/"
          element={
            <RequireAuth>
              <AppLayout />
            </RequireAuth>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="transactions" element={<Transactions />} />
          <Route path="upload" element={<UploadOCR />} />
          <Route path="statistics" element={<Statistics />} />
          <Route path="settings" element={<TenantSettings />} />
          <Route path="help" element={<Help />} />
        </Route>

        {/* 平台管理端主路由 */}
        <Route
          path="/platform"
          element={
            <RequirePlatformAuth>
              <PlatformLayout />
            </RequirePlatformAuth>
          }
        >
          <Route index element={<PlatformDashboard />} />
          <Route path="users" element={<PlatformUsers />} />
          <Route path="users/:id" element={<PlatformUserDetail />} />
          <Route path="settings" element={<PlatformSettings />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
