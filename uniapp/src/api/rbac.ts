import { request } from './request'
import type { PermissionEntry, TenantRole } from './types'

export const rbacApi = {
  /** GET /permissions */
  listPermissions(): Promise<PermissionEntry[]> {
    return request<PermissionEntry[]>({ url: '/permissions' })
  },

  /** GET /tenants/:id/roles */
  listRoles(tenantId: number): Promise<TenantRole[]> {
    return request<TenantRole[]>({ url: `/tenants/${tenantId}/roles` })
  },

  /** POST /tenants/:id/roles */
  createRole(
    tenantId: number,
    data: { name: string; permissions: PermissionEntry[] },
  ): Promise<TenantRole> {
    return request<TenantRole>({ url: `/tenants/${tenantId}/roles`, method: 'POST', data })
  },

  /** PUT /tenants/:id/roles/:rid */
  updateRole(
    tenantId: number,
    roleId: number,
    data: { permissions: PermissionEntry[] },
  ): Promise<TenantRole> {
    return request<TenantRole>({
      url: `/tenants/${tenantId}/roles/${roleId}`,
      method: 'PUT',
      data,
    })
  },

  /** DELETE /tenants/:id/roles/:rid */
  deleteRole(tenantId: number, roleId: number): Promise<null> {
    return request<null>({ url: `/tenants/${tenantId}/roles/${roleId}`, method: 'DELETE' })
  },
}
