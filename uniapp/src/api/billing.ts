import { request } from './request'

export interface BillingPackage {
  sku_code: string
  name: string
  wx_product_id: string
  price_fen: number
  credit_amount: number
  validity_days: number
  status: string
  synced_at?: string | null
  estimates: BillingEstimates
}

export interface BillingUnitPrices {
  upload: number
  ocr: number
  ai: number
}

export interface BillingEstimates {
  upload_ocr_times: number | null
  ai_times: number | null
  full_chain_times: number | null
}

export interface BillingSummary {
  enabled: boolean
  packages: BillingPackage[]
  credit_balance: number
  unlimited: boolean
  unit_prices: BillingUnitPrices
  estimates: BillingEstimates
  synced_at?: string | null
  stale: boolean
  iap_enabled: boolean
}

export interface BillingIntent {
  business_ref: string
  sku_code: string
  name?: string
  amount_fen: number
  credit_amount: number
  validity_days: number
  expires_at: string
  status: string
}

export interface BillingIntentPage {
  total: number
  page: number
  size: number
  items: BillingIntent[]
}

export interface BillingPrepare {
  out_trade_no: string
  mode: string
  signData: string
  paySig: string
  signature: string
  env: number
}

export const billingApi = {
  summary: (refresh = false) =>
    request<BillingSummary>({ url: `/billing/summary${refresh ? '?refresh=1' : ''}`, method: 'GET' }),
  createIntent: (skuCode: string, expectedPriceFen: number, platform: string) =>
    request<BillingIntent>({
      url: '/billing/purchase-intents',
      method: 'POST',
      data: { sku_code: skuCode, expected_price_fen: expectedPriceFen, platform },
    }),
  prepare: (ref: string, code: string) =>
    request<BillingPrepare>({ url: `/billing/purchase-intents/${ref}/prepare`, method: 'POST', data: { code } }),
  intents: (page = 1, size = 20) =>
    request<BillingIntentPage>({ url: `/billing/purchase-intents?page=${page}&size=${size}`, method: 'GET' }),
  intentsByRef: (ref: string) =>
    request<BillingIntent>({ url: `/billing/purchase-intents/${ref}`, method: 'GET' }),
}
