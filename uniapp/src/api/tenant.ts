import { request } from './request'
import type { Tenant, TenantMember, TenantSettings } from './types'

export const tenantApi = {
  /** GET /tenants */
  list(): Promise<Tenant[]> {
    return request<Tenant[]>({ url: '/tenants' })
  },

  /** POST /tenants */
  create(data: { name: string }): Promise<Tenant> {
    return request<Tenant>({ url: '/tenants', method: 'POST', data })
  },

  /** PUT /tenants/:id */
  update(id: number, data: { name: string }): Promise<Tenant> {
    return request<Tenant>({ url: `/tenants/${id}`, method: 'PUT', data })
  },

  /** GET /tenants/:id/members */
  getMembers(id: number): Promise<TenantMember[]> {
    return request<TenantMember[]>({ url: `/tenants/${id}/members` })
  },

  /** POST /tenants/:id/members */
  addMember(id: number, data: { username: string; role: string }): Promise<TenantMember> {
    return request<TenantMember>({ url: `/tenants/${id}/members`, method: 'POST', data })
  },

  /** DELETE /tenants/:id/members/:uid */
  removeMember(id: number, userId: number): Promise<null> {
    return request<null>({ url: `/tenants/${id}/members/${userId}`, method: 'DELETE' })
  },

  /** PUT /tenants/:id/members/:uid */
  updateMemberRole(id: number, userId: number, role: string): Promise<null> {
    return request<null>({
      url: `/tenants/${id}/members/${userId}`,
      method: 'PUT',
      data: { role },
    })
  },

  /** GET /tenants/:id/settings */
  getSettings(id: number): Promise<TenantSettings> {
    return request<TenantSettings>({ url: `/tenants/${id}/settings` })
  },

  /** PUT /tenants/:id/settings */
  updateSettings(id: number, data: TenantSettings): Promise<TenantSettings> {
    return request<TenantSettings>({ url: `/tenants/${id}/settings`, method: 'PUT', data })
  },
}
