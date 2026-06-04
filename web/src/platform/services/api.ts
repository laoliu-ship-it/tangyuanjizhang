import axios from 'axios'
import { usePlatformAuthStore } from '../store/auth'

const platformApi = axios.create({ baseURL: '/api/platform' })

platformApi.interceptors.request.use(config => {
  const token = usePlatformAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

platformApi.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      const url = err.config?.url || ''
      if (!url.includes('/auth/login')) {
        usePlatformAuthStore.getState().logout()
        window.location.href = '/platform/login'
      }
    }
    return Promise.reject(err)
  }
)

export interface ApiResponse<T> {
  code: number
  data: T
  message?: string
}

export interface PlatformLoginResp {
  token: string
  id: number
  name: string
  email: string
}

export interface PlatformDashboardResp {
  total_users: number
  total_tenants: number
  total_transactions: number
}

export interface PlatformUserItem {
  id: number
  username: string
  email: string
  created_at: string
}

export interface PlatformUserListResp {
  total: number
  page: number
  size: number
  items: PlatformUserItem[]
}

export interface PlatformUserDetailResp {
  user_id: number
  username: string
  email: string
  tenant_count: number
  transaction_count: number
  media_count: number
}

export interface PlatformConfigItem {
  key: string
  value: string
  description: string
  updated_at: string
}

export interface PlatformConfigListResp {
  items: PlatformConfigItem[]
}

export const platformAdminApi = {
  login: (data: { email: string; password: string }) =>
    platformApi.post<ApiResponse<PlatformLoginResp>>('/auth/login', data),

  getDashboard: () =>
    platformApi.get<ApiResponse<PlatformDashboardResp>>('/dashboard'),

  listUsers: (params: { keyword?: string; page?: number; page_size?: number }) =>
    platformApi.get<ApiResponse<PlatformUserListResp>>('/users', { params }),

  getUserDetail: (userId: number) =>
    platformApi.get<ApiResponse<PlatformUserDetailResp>>(`/users/${userId}`),

  // 配置管理
  getConfigs: () =>
    platformApi.get<ApiResponse<PlatformConfigListResp>>('/configs'),

  updateConfig: (key: string, value: string) =>
    platformApi.put<ApiResponse<PlatformConfigItem>>(`/configs/${key}`, { value }),
}

export default platformApi
