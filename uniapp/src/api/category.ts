import { request } from './request'
import type { Category, TxType } from './types'

export const categoryApi = {
  /** GET /categories */
  list(): Promise<Category[]> {
    return request<Category[]>({ url: '/categories' })
  },

  /** POST /categories */
  create(data: { name: string; type: TxType; icon: string }): Promise<Category> {
    return request<Category>({ url: '/categories', method: 'POST', data })
  },

  /** PUT /categories/:id */
  update(id: number, data: { name: string; type: TxType; icon: string }): Promise<Category> {
    return request<Category>({ url: `/categories/${id}`, method: 'PUT', data })
  },

  /** DELETE /categories/:id */
  remove(id: number): Promise<null> {
    return request<null>({ url: `/categories/${id}`, method: 'DELETE' })
  },
}
