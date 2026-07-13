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
        alert('登录已过期，请重新登录')
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
  last_login_at: string
  last_transaction_at: string
  credit_balance: number
  credit_unlimited: boolean
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

// ========== 积分系统 ==========

export interface CreditLogItem {
  id: number
  amount: number
  action: string
  balance_after: number
  ref_type: string
  ref_id: number
  note: string
  created_at: string
}

export interface TenantCreditsDetail {
  tenant_id: number
  tenant_name: string
  balance: number
  unlimited: boolean
  logs?: CreditLogItem[]
}

export interface TenantCreditsListResp {
  items: TenantCreditsDetail[]
}

// ========== 平台 LLM 配置 ==========

export interface PlatformLLMConfig {
  enabled: boolean
  provider: string
  base_url: string
  api_key_mask: string
  model: string
  mode: string
}

// ========== 提示词管理 ==========

export interface TenantPromptResp {
  custom_prompt: string
  default_prompt: string
  placeholder: string
}

// ========== 定时任务管理 ==========

export interface CronJobItem {
  name: string
  description: string
  schedule: string
  running: boolean
  next_run_at: string
  last_run_at: string
  last_duration: string
  last_error: string
  run_count: number
  fail_count: number
}

export interface CronJobListResp {
  jobs: CronJobItem[]
}

export interface CronTriggerResp {
  name: string
  triggered: boolean
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

  // 积分管理
  listUserTenantCredits: (userId: number) =>
    platformApi.get<ApiResponse<TenantCreditsListResp>>(`/users/${userId}/tenants/credits`),

  getTenantCredits: (tenantId: number) =>
    platformApi.get<ApiResponse<TenantCreditsDetail>>(`/tenants/${tenantId}/credits`),

  adjustTenantCredits: (tenantId: number, delta: number, note: string) =>
    platformApi.post<ApiResponse<TenantCreditsDetail>>(`/tenants/${tenantId}/credits/adjust`, { delta, note }),

  setTenantUnlimited: (tenantId: number, unlimited: boolean) =>
    platformApi.put<ApiResponse<TenantCreditsDetail>>(`/tenants/${tenantId}/credits/unlimited`, { unlimited }),

  getTenantPrompt: (tenantId: number) =>
    platformApi.get<ApiResponse<TenantPromptResp>>(`/tenants/${tenantId}/prompt`),

  updateTenantPrompt: (tenantId: number, customPrompt: string) =>
    platformApi.put<ApiResponse<TenantPromptResp>>(`/tenants/${tenantId}/prompt`, { custom_prompt: customPrompt }),

  // 平台 LLM 配置
  getPlatformLLMConfig: () =>
    platformApi.get<ApiResponse<PlatformLLMConfig>>('/llm-config'),

  updatePlatformLLMConfig: (data: {
    enabled: boolean
    provider: string
    base_url: string
    api_key: string
    model: string
    mode: string
  }) =>
    platformApi.put<ApiResponse<PlatformLLMConfig>>('/llm-config', data),

  // 定时任务管理
  listCronJobs: () =>
    platformApi.get<ApiResponse<CronJobListResp>>('/cron/jobs'),

  triggerCronJob: (name: string) =>
    platformApi.post<ApiResponse<CronTriggerResp>>(`/cron/jobs/${name}/run`),
}

export default platformApi
