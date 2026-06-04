import axios from 'axios'
import { useAuthStore } from '../store/auth'
import { useTenantStore } from '../store/tenant'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use(config => {
  const token = useAuthStore.getState().token
  const tenantId = useTenantStore.getState().currentTenantId
  if (token) config.headers.Authorization = `Bearer ${token}`
  if (tenantId) config.headers['X-Tenant-ID'] = String(tenantId)
  return config
})

api.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api

// ---- 类型定义 ----

export interface ApiResponse<T> {
  code: number
  data: T
  message?: string
}

export interface User {
  user_id: number
  username: string
  email: string
}

export interface Tenant {
  id: number
  name: string
  owner_id: number
}

export interface TenantMember {
  user_id: number
  username: string
  email: string
  role: string
  role_id?: number
  joined_at: string
}

export interface PermissionEntry {
  resource: string
  action: string
}

export interface TenantRole {
  id: number
  name: string
  is_system: boolean
  permissions: PermissionEntry[]
}

export interface Category {
  id: number
  name: string
  type: 'income' | 'expense'
  icon: string
}

export interface TransactionImage {
  image_path: string
  ocr_amount: number
  ocr_date: string
  ocr_merchant: string
  ocr_raw_texts: string
}

export interface Transaction {
  id: number
  type: 'income' | 'expense'
  amount: number
  category_id: number
  category_name?: string
  merchant_id?: number
  merchant_name?: string
  transaction_date: string
  note: string
  images?: TransactionImage[]
  created_at?: string
}

export interface Merchant {
  id: number
  name: string
  created_at?: string
}

export interface TransactionCreatePayload {
  type: 'income' | 'expense'
  amount: number
  category_id: number
  merchant_id?: number
  merchant_name?: string
  transaction_date: string
  note: string
  image_path?: string
  ocr_amount?: number
  ocr_date?: string
  ocr_merchant?: string
  ocr_raw_texts?: string
}

export interface DailyStatistics {
  date: string
  total_income: number
  total_expense: number
}

export interface MonthlyStatistics {
  year: number
  month: number
  total: {
    total_income: number
    total_expense: number
    net_amount: number
  }
  daily: DailyStatistics[]
  categories: CategoryStat[]
}

export interface YearlyStatistics {
  year: number
  total: {
    total_income: number
    total_expense: number
    net_amount: number
  }
  monthly: MonthSummary[]
  categories: CategoryStat[]
}

export interface MonthSummary {
  month: number
  total_income: number
  total_expense: number
}

export interface CategoryStat {
  category_id: number
  category_name: string
  category_icon: string
  type: 'income' | 'expense'
  total: number
}

export interface RangeStatistics {
  total: {
    total_income: number
    total_expense: number
    net_amount: number
  }
  daily: DailyStatistics[]
  categories: CategoryStat[]
}

export interface OcrResult {
  ocr_id: number        // 服务端 ocr_records.id，供后续 LLM 调用
  image_path: string
  ai_mode: boolean
  amount: number       // ai_mode=false 时为 0
  date: string         // ai_mode=false 时为 ""
  merchant_id: number  // 自动创建的商户ID
  merchant_name: string // 商户名称
  raw_texts: string[]  // 始终有值
}

export interface LLMSuggestion {
  type: 'income' | 'expense'
  amount: number
  merchant_name: string
  date: string          // "YYYY-MM-DD"
  category_id: number   // 匹配到的分类 ID，0 表示未匹配
  category_hint: string // 中文分类提示，如"餐饮"
  note: string
  source_lines: number[]  // 对应 OCR 原始文字的行号（从 0 开始）
}

export interface LLMAnalyzeRequest {
  image_path: string
  raw_texts: string[]
  categories: { id: number; name: string; type: string }[]
}

// 新流程：只传 ocr_id，服务端从 ocr_records 取内容再调 LLM
export interface LLMAnalyzeByOcrIdRequest {
  ocr_id: number
}

export interface LLMAnalyzeResponse {
  suggestions: LLMSuggestion[]
  error?: string
}

export interface OcrAnalyzeResult extends OcrResult {
  llm?: LLMSuggestion[]
  llm_error?: string
}

export interface TenantSettings {
  require_expense_image: boolean
}

export interface TenantLLMConfig {
  id: number
  tenant_id: number
  enabled: boolean
  use_platform: boolean
  platform_enabled: boolean // 平台是否已配置（未配置时前端给提示）
  provider: string
  base_url: string
  api_key_mask: string
  model: string
  mode: 'vision' | 'ocr_text'
}

export interface TransactionListResponse {
  total: number
  items: Transaction[]
}

// ---- API 函数 ----

// 认证
export const authApi = {
  register: (data: { username: string; email: string; password: string }) =>
    api.post<ApiResponse<{ token: string; user_id: number; username: string; email: string; tenants: { id: number; name: string }[] }>>('/auth/register', data),
  login: (data: { email: string; password: string }) =>
    api.post<ApiResponse<{ token: string; user_id: number; username: string; email: string; tenants: { id: number; name: string }[] }>>('/auth/login', data),
}

// 租户
export const tenantApi = {
  list: () => api.get<ApiResponse<Tenant[]>>('/tenants'),
  create: (data: { name: string }) => api.post<ApiResponse<Tenant>>('/tenants', data),
  update: (id: number, data: { name: string }) => api.put<ApiResponse<Tenant>>(`/tenants/${id}`, data),
  addMember: (id: number, data: { username: string; role: string }) =>
    api.post<ApiResponse<TenantMember>>(`/tenants/${id}/members`, data),
  removeMember: (id: number, userId: number) =>
    api.delete<ApiResponse<null>>(`/tenants/${id}/members/${userId}`),
  updateMemberRole: (id: number, userId: number, role: string) =>
    api.put<ApiResponse<null>>(`/tenants/${id}/members/${userId}`, { role }),
  getMembers: (id: number) =>
    api.get<ApiResponse<TenantMember[]>>(`/tenants/${id}/members`),
  getSettings: (id: number) =>
    api.get<ApiResponse<TenantSettings>>(`/tenants/${id}/settings`),
  updateSettings: (id: number, data: TenantSettings) =>
    api.put<ApiResponse<TenantSettings>>(`/tenants/${id}/settings`, data),
}

// 分类
export const categoryApi = {
  list: () => api.get<ApiResponse<Category[]>>('/categories'),
  create: (data: { name: string; type: 'income' | 'expense'; icon: string }) =>
    api.post<ApiResponse<Category>>('/categories', data),
  update: (id: number, data: { name: string; type: 'income' | 'expense'; icon: string }) =>
    api.put<ApiResponse<Category>>(`/categories/${id}`, data),
  delete: (id: number) => api.delete<ApiResponse<null>>(`/categories/${id}`),
}

// 商户
export const merchantApi = {
  list: () => api.get<ApiResponse<Merchant[]>>('/merchants'),
  create: (name: string) => api.post<ApiResponse<Merchant>>('/merchants', { name }),
  update: (id: number, name: string) => api.put<ApiResponse<Merchant>>(`/merchants/${id}`, { name }),
  delete: (id: number) => api.delete<ApiResponse<null>>(`/merchants/${id}`),
}

// 交易
export const transactionApi = {
  list: (params: {
    page?: number
    page_size?: number
    type?: string
    category_id?: number
    start_date?: string
    end_date?: string
    sort_by?: 'transaction_date' | 'created_at'
  }) => api.get<ApiResponse<TransactionListResponse>>('/transactions', { params }),
  create: (data: TransactionCreatePayload) =>
    api.post<ApiResponse<Transaction>>('/transactions', data),
  batchCreate: (data: TransactionCreatePayload[]) =>
    api.post<ApiResponse<Transaction[]>>('/transactions/batch', data),
  update: (id: number, data: TransactionCreatePayload) =>
    api.put<ApiResponse<Transaction>>(`/transactions/${id}`, data),
  delete: (id: number) => api.delete<ApiResponse<null>>(`/transactions/${id}`),
}

// 统计
export const statisticsApi = {
  daily: (date: string) =>
    api.get<ApiResponse<DailyStatistics>>('/statistics/daily', { params: { date } }),
  monthly: (year: number, month: number) =>
    api.get<ApiResponse<MonthlyStatistics>>('/statistics/monthly', { params: { year, month } }),
  yearly: (year: number) =>
    api.get<ApiResponse<YearlyStatistics>>('/statistics/yearly', { params: { year } }),
  range: (start: string, end: string) =>
    api.get<ApiResponse<RangeStatistics>>('/statistics/range', { params: { start, end } }),
}

// OCR
export const uploadApi = {
  ocr: (file: File, originalHash?: string, config?: { signal?: AbortSignal }) => {
    const form = new FormData()
    form.append('file', file)
    if (originalHash) form.append('original_hash', originalHash)
    return api.post<ApiResponse<OcrResult & { duplicate?: boolean; file_name?: string }>>('/upload/ocr', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      signal: config?.signal,
    })
  },
  ocrAnalyze: (file: File, originalHash?: string, config?: { signal?: AbortSignal }) => {
    const form = new FormData()
    form.append('file', file)
    if (originalHash) form.append('original_hash', originalHash)
    return api.post<ApiResponse<OcrAnalyzeResult & { duplicate?: boolean; file_name?: string }>>('/upload/ocr/analyze', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      signal: config?.signal,
    })
  },
}

// LLM 配置
export const llmApi = {
  getConfig: () => api.get<ApiResponse<TenantLLMConfig>>('/llm/config'),
  saveConfig: (data: {
    enabled: boolean
    use_platform: boolean
    provider?: string
    base_url?: string
    api_key?: string
    model?: string
    mode?: 'vision' | 'ocr_text'
  }) => api.put<ApiResponse<TenantLLMConfig>>('/llm/config', data),
  analyze: (data: LLMAnalyzeRequest) =>
    api.post<ApiResponse<LLMAnalyzeResponse>>('/llm/analyze', data),
  analyzeByOcrId: (data: LLMAnalyzeByOcrIdRequest) =>
    api.post<ApiResponse<LLMAnalyzeResponse>>('/llm/analyze', data),
}

// 导出
export const exportApi = {
  excel: (params: { start_date?: string; end_date?: string; type?: string }) =>
    api.get('/export/excel', { params, responseType: 'blob' }),
}

// 导入

export interface ColumnMapping {
  date: number
  type: number
  amount: number
  category: number
  merchant: number
  note: number
}

export interface ParseHeadersResult {
  sheets: string[]
  sheet_index: number
  headers: string[]
  sample_rows: string[][]
  suggestions: ColumnMapping
}

export const importApi = {
  downloadTemplate: () =>
    api.get('/import/template', { responseType: 'blob' }),
  parseHeaders: (file: File, sheetIndex?: number) => {
    const form = new FormData()
    form.append('file', file)
    const params = sheetIndex !== undefined ? { sheet: String(sheetIndex) } : {}
    return api.post<ApiResponse<ParseHeadersResult>>('/import/parse-headers', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      params,
    })
  },
  importExcel: (file: File, mapping?: ColumnMapping, sheetIndex?: number, dryRun?: boolean) => {
    const form = new FormData()
    form.append('file', file)
    if (mapping) form.append('mapping', JSON.stringify(mapping))
    if (sheetIndex !== undefined) form.append('sheet', String(sheetIndex))
    if (dryRun) form.append('dry_run', 'true')
    return api.post<ApiResponse<{
      dry_run: boolean
      valid_count: number
      skipped_count: number
      imported: number
      issues: string[]
    }>>('/import/excel', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}

// RBAC
export const rbacApi = {
  listPermissions: () =>
    api.get<ApiResponse<PermissionEntry[]>>('/permissions'),
  listRoles: (tenantId: number) =>
    api.get<ApiResponse<TenantRole[]>>(`/tenants/${tenantId}/roles`),
  createRole: (tenantId: number, data: { name: string; permissions: PermissionEntry[] }) =>
    api.post<ApiResponse<TenantRole>>(`/tenants/${tenantId}/roles`, data),
  updateRole: (tenantId: number, roleId: number, data: { permissions: PermissionEntry[] }) =>
    api.put<ApiResponse<TenantRole>>(`/tenants/${tenantId}/roles/${roleId}`, data),
  deleteRole: (tenantId: number, roleId: number) =>
    api.delete<ApiResponse<null>>(`/tenants/${tenantId}/roles/${roleId}`),
}
