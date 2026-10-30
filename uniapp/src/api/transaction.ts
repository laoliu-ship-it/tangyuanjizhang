import { request } from './request'
import type {
  Transaction,
  TransactionCreatePayload,
  TransactionListParams,
  TransactionListResponse,
} from './types'

export const transactionApi = {
  /** GET /transactions?page=&page_size=&type=&category_id=&merchant_id=&start_date=&end_date=&sort_by= */
  list(params: TransactionListParams): Promise<TransactionListResponse> {
    return request<TransactionListResponse>({ url: '/transactions', data: params })
  },

  /** POST /transactions */
  create(data: TransactionCreatePayload): Promise<Transaction> {
    return request<Transaction>({ url: '/transactions', method: 'POST', data })
  },

  /** PUT /transactions/:id */
  update(id: number, data: TransactionCreatePayload): Promise<Transaction> {
    return request<Transaction>({ url: `/transactions/${id}`, method: 'PUT', data })
  },

  /** DELETE /transactions/:id */
  remove(id: number): Promise<null> {
    return request<null>({ url: `/transactions/${id}`, method: 'DELETE' })
  },

  /** DELETE /transactions/batch（body: { ids }），返回删除条数 */
  batchRemove(ids: number[]): Promise<number> {
    return request<number>({ url: '/transactions/batch', method: 'DELETE', data: { ids } })
  },
}
