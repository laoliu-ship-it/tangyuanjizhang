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
    // 小程序端会把 undefined 序列化成 "undefined" 拼进 query，这里显式只保留有值字段
    const data: TransactionListParams = {}
    if (params.page !== undefined) data.page = params.page
    if (params.page_size !== undefined) data.page_size = params.page_size
    if (params.type) data.type = params.type
    if (params.category_id) data.category_id = params.category_id
    if (params.merchant_id) data.merchant_id = params.merchant_id
    if (params.start_date) data.start_date = params.start_date
    if (params.end_date) data.end_date = params.end_date
    if (params.sort_by) data.sort_by = params.sort_by
    return request<TransactionListResponse>({ url: '/transactions', data })
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
