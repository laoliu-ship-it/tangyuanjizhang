import { request } from './request'
import type { Merchant } from './types'

export const merchantApi = {
  /** GET /merchants */
  list(): Promise<Merchant[]> {
    return request<Merchant[]>({ url: '/merchants' })
  },

  /** POST /merchants */
  create(name: string): Promise<Merchant> {
    return request<Merchant>({ url: '/merchants', method: 'POST', data: { name } })
  },

  /** PUT /merchants/:id */
  update(id: number, name: string): Promise<Merchant> {
    return request<Merchant>({ url: `/merchants/${id}`, method: 'PUT', data: { name } })
  },

  /** DELETE /merchants/:id */
  remove(id: number): Promise<null> {
    return request<null>({ url: `/merchants/${id}`, method: 'DELETE' })
  },
}
