import { useState, useEffect, useCallback } from 'react'
import { rbacApi, type TenantRole } from '../../services/api'
import { useTenantStore } from '../../store/tenant'

const PERMISSION_LABELS: Record<string, Record<string, string>> = {
  transaction: { read: '查看账目', write: '增删改账目' },
  category: { read: '查看分类', write: '管理分类' },
  merchant: { read: '查看商户', write: '管理商户' },
  statistics: { read: '查看统计' },
  export: { read: '导出数据' },
  tenant: { read: '查看租户', write: '管理租户' },
}

const ROLE_LABELS: Record<string, string> = {
  admin: '管理员',
  finance: '财务',
  partner: '合伙人',
}

export default function Roles() {
  const { currentTenantId } = useTenantStore()
  const [roles, setRoles] = useState<TenantRole[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [newRoleName, setNewRoleName] = useState('')
  const [newRolePerms, setNewRolePerms] = useState<Set<string>>(new Set())

  const loadData = useCallback(async () => {
    if (!currentTenantId) return
    setLoading(true)
    try {
      const rolesRes = await rbacApi.listRoles(currentTenantId)
      setRoles(rolesRes.data.data ?? [])
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      if (status === 403) {
        alert('权限不足：您没有权限查看角色权限（仅管理员可用）')
      } else if (status === 401) {
        alert('登录已过期，请重新登录')
      } else {
        alert(msg || `加载失败（HTTP ${status || '未知'}）`)
      }
    } finally {
      setLoading(false)
    }
  }, [currentTenantId])

  useEffect(() => {
    loadData()
  }, [loadData])

  async function handleCreate() {
    if (!newRoleName.trim() || !currentTenantId) return
    if (newRolePerms.size === 0) {
      alert('请至少选择一个权限')
      return
    }
    try {
      const perms = Array.from(newRolePerms).map(k => {
        const [resource, action] = k.split(':')
        return { resource, action }
      })
      await rbacApi.createRole(currentTenantId, { name: newRoleName.trim(), permissions: perms })
      setNewRoleName('')
      setNewRolePerms(new Set())
      setShowCreate(false)
      loadData()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '创建失败')
    }
  }

  async function handleDelete(role: TenantRole) {
    if (!confirm(`确认删除角色"${role.name}"？`)) return
    if (!currentTenantId) return
    try {
      await rbacApi.deleteRole(currentTenantId, role.id)
      loadData()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '删除失败')
    }
  }

  function permSummary(role: TenantRole) {
    return (role.permissions ?? [])
      .map(p => PERMISSION_LABELS[p.resource]?.[p.action] || `${p.resource}:${p.action}`)
      .join('、')
  }

  if (loading) return <div className="py-10 text-center text-gray-400">加载中...</div>

  return (
    <div>
      {/* 角色列表 */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden mb-5">
        <div className="px-5 py-3 bg-gray-50 border-b border-gray-100 flex justify-between items-center">
          <span className="text-sm font-medium text-gray-500">共 {roles.length} 个角色</span>
          <button
            onClick={() => setShowCreate(true)}
            className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-xl transition-colors"
          >
            创建角色
          </button>
        </div>

        <ul className="divide-y divide-gray-50">
          {roles.map(role => (
            <li key={role.id} className="flex items-center justify-between px-5 py-4 gap-3">
              <div className="min-w-0">
                <p className="text-sm font-medium text-gray-800">
                  {ROLE_LABELS[role.name] || role.name}
                  {role.is_system && (
                    <span className="ml-2 px-2 py-0.5 bg-gray-100 text-gray-500 text-xs rounded-full">
                      系统
                    </span>
                  )}
                </p>
                <p className="text-xs text-gray-400 mt-1 truncate">{permSummary(role)}</p>
              </div>
              {!role.is_system && (
                <button
                  onClick={() => handleDelete(role)}
                  className="text-sm text-red-500 hover:text-red-700 transition-colors shrink-0"
                >
                  删除
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>

      {/* 创建角色弹窗 */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg">
            <h3 className="text-lg font-semibold text-gray-800 mb-4">创建角色</h3>
            <input
              type="text"
              value={newRoleName}
              onChange={e => setNewRoleName(e.target.value)}
              placeholder="角色名称"
              className="w-full px-4 py-2.5 border border-gray-300 rounded-xl text-sm mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <div className="space-y-3 mb-4 max-h-64 overflow-y-auto">
              {Object.entries(PERMISSION_LABELS).map(([resource, actions]) => (
                <div key={resource}>
                  <p className="text-xs font-medium text-gray-500 mb-1">
                    {resource === 'transaction' ? '账目' : resource === 'category' ? '分类' : resource === 'merchant' ? '商户' : resource === 'statistics' ? '统计' : resource === 'export' ? '导出' : resource === 'tenant' ? '租户' : resource}
                  </p>
                  <div className="flex flex-wrap gap-3">
                    {Object.entries(actions).map(([action, label]) => {
                      const key = `${resource}:${action}`
                      const checked = newRolePerms.has(key)
                      return (
                        <label key={key} className="flex items-center gap-1.5 text-sm cursor-pointer">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => {
                              const next = new Set(newRolePerms)
                              checked ? next.delete(key) : next.add(key)
                              setNewRolePerms(next)
                            }}
                            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          />
                          {label}
                        </label>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => { setShowCreate(false); setNewRoleName(''); setNewRolePerms(new Set()) }}
                className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800"
              >
                取消
              </button>
              <button
                onClick={handleCreate}
                className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-xl transition-colors"
              >
                创建
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
