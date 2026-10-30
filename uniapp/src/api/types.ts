/**
 * 后端 DTO 类型（字段名与后端 JSON 保持一致，全部 snake_case）
 * 唯一来源：web/src/services/api.ts + server/internal/dto
 */

export interface ApiResponse<T> {
  code: number
  data: T
  message?: string
}

// ── 用户 / 租户 ────────────────────────────────────────────────

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

export interface TenantSettings {
  require_expense_image: boolean
}

export interface AuthResult {
  token: string
  user_id: number
  username: string
  email: string
  tenants: { id: number; name: string }[]
}

// ── 分类 / 商户 ────────────────────────────────────────────────

export type TxType = 'income' | 'expense'

export interface Category {
  id: number
  name: string
  type: TxType
  icon: string
}

export interface Merchant {
  id: number
  name: string
  created_at?: string
}

// ── 交易 ──────────────────────────────────────────────────────

export interface TransactionImage {
  image_path: string
  ocr_amount: number
  ocr_date: string
  ocr_merchant: string
  ocr_raw_texts: string
}

export interface Transaction {
  id: number
  type: TxType
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

export interface TransactionCreatePayload {
  type: TxType
  amount: number
  category_id: number
  merchant_id?: number
  merchant_name?: string
  /** "YYYY-MM-DD HH:mm:00" */
  transaction_date: string
  note: string
  image_path?: string
  ocr_amount?: number
  ocr_date?: string
  ocr_merchant?: string
  ocr_raw_texts?: string
}

export interface TransactionListParams {
  page?: number
  page_size?: number
  type?: string
  category_id?: number
  merchant_id?: number
  start_date?: string
  end_date?: string
  sort_by?: 'transaction_date' | 'created_at'
}

export interface TransactionListResponse {
  total: number
  items: Transaction[]
}

// ── 统计 ──────────────────────────────────────────────────────

export interface DailyStatistics {
  date: string
  total_income: number
  total_expense: number
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
  type: TxType
  total: number
}

export interface MerchantStat {
  merchant_id: number
  merchant_name: string
  total: number
  tx_count: number
}

export interface StatsTotal {
  total_income: number
  total_expense: number
  net_amount: number
}

export interface MerchantStatsResp {
  top_merchants: MerchantStat[]
  computed_at?: string
  from_cache: boolean
}

export interface MonthlyStatistics {
  year: number
  month: number
  total: StatsTotal
  daily: DailyStatistics[]
  categories: CategoryStat[]
  top_merchants: MerchantStat[]
}

export interface YearlyStatistics {
  year: number
  total: StatsTotal
  monthly: MonthSummary[]
  categories: CategoryStat[]
  top_merchants: MerchantStat[]
}

export interface RangeStatistics {
  total: StatsTotal
  daily: DailyStatistics[]
  categories: CategoryStat[]
  top_merchants: MerchantStat[]
}

export type StatisticsResult = MonthlyStatistics | YearlyStatistics | RangeStatistics

// ── OCR / LLM ─────────────────────────────────────────────────

export interface OcrResult {
  /** 服务端 ocr_records.id，供后续 LLM 调用 */
  ocr_id: number
  image_path: string
  ai_mode: boolean
  /** ai_mode=false 时无意义 */
  amount: number
  /** ai_mode=false 时为空串 */
  date: string
  merchant_id: number
  merchant_name: string
  raw_texts: string[]
}

export interface OcrUploadResp extends OcrResult {
  duplicate?: boolean
  file_name?: string
}

export interface LLMSuggestion {
  type: TxType
  amount: number
  merchant_name: string
  /** "YYYY-MM-DD" 或 "YYYY-MM-DD HH:mm" */
  date: string
  /** 匹配到的分类 ID，0 表示未匹配 */
  category_id: number
  /** 中文分类提示，如「餐饮」 */
  category_hint: string
  note: string
  /** 对应 OCR 原始文字的行号（从 0 开始） */
  source_lines: number[]
}

export interface LLMAnalyzeResponse {
  suggestions: LLMSuggestion[]
  error?: string
}

export interface TenantLLMConfig {
  id: number
  tenant_id: number
  enabled: boolean
  use_platform: boolean
  platform_enabled: boolean
  provider: string
  base_url: string
  api_key_mask: string
  model: string
  mode: 'vision' | 'ocr_text'
}

export interface TenantPrompt {
  custom_prompt: string
  default_prompt: string
  placeholder: string
}

// ── 积分 ──────────────────────────────────────────────────────

export interface CreditLogItem {
  id: number
  /** 正=充值，负=消耗 */
  amount: number
  /** upload/ocr/llm/gift/admin_adjust */
  action: string
  /** -1 表示不限 */
  balance_after: number
  ref_type: string
  ref_id: number
  note: string
  created_at: string
}

export interface CreditLotItem {
  remaining: number
  expires_at: string
}

export interface CreditsResp {
  tenant_id: number
  balance: number
  unlimited: boolean
  lots: CreditLotItem[]
  logs: CreditLogItem[]
}

export interface CreditLogsPageResp {
  total: number
  page: number
  size: number
  logs: CreditLogItem[]
}

// ── Excel 导入导出 ────────────────────────────────────────────

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

export interface ImportResult {
  dry_run: boolean
  valid_count: number
  skipped_count: number
  imported: number
  issues: string[]
}
