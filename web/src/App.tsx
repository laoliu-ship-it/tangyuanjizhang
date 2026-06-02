import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import { useTenantStore } from './store/tenant'
import AppLayout from './layouts/AppLayout'
import Login from './pages/Auth/Login'
import Register from './pages/Auth/Register'
import Dashboard from './pages/Dashboard/index'
import Transactions from './pages/Transactions/index'
import UploadOCR from './pages/UploadOCR/index'
import Statistics from './pages/Statistics/index'
import TenantSettings from './pages/TenantSettings/index'

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

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
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
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
