import { request } from './request'
import type { CreditLogsPageResp, CreditsResp } from './types'

export const creditApi = {
  /** GET /credits */
  getBalance(): Promise<CreditsResp> {
    return request<CreditsResp>({ url: '/credits' })
  },

  /** GET /credits/logs?page=&size=&type= */
  getLogs(page: number, size: number, type?: 'income' | 'expense'): Promise<CreditLogsPageResp> {
    const params: Record<string, string | number> = { page, size }
    if (type) params.type = type
    return request<CreditLogsPageResp>({ url: '/credits/logs', data: params })
  },
}
