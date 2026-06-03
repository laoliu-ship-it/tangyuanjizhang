import { useState, useEffect, useCallback } from 'react'
import { tenantApi, rbacApi, type TenantMember, type TenantRole } from '../../services/api'
import { useTenantStore } from '../../store/tenant'
import { useAuthStore } from '../../store/auth'

const ROLE_LABELS: Record<string, string> = {
  admin: '管理员',
  finance: '财务',
  partner: '合伙人',
}

export default function Members() {
  const { currentTenantId, tenants } = useTenantStore()
  const userId = useAuthStore(s => s.userId)
  const currentTenant = tenants.find(t => t.id === currentTenantId)

  const [members, setMembers] = useState<TenantMember[]>([])
  const [roles, setRoles] = useState<TenantRole[]>([])
  const [loading, setLoading] = useState(true)
  const [inviteUsername, setInviteUsername] = useState('')
  const [inviteRole, setInviteRole] = useState('finance')
  const [inviting, setInviting] = useState(false)
  const [removingId, setRemovingId] = useState<number | null>(null)
  const [updatingRoleId, setUpdatingRoleId] = useState<number | null>(null)

  // 当前用户是否是管理员（admin 角色可以管理成员）
  const currentUserRole = members.find(m => m.user_id === userId)?.role
  const isAdmin = currentUserRole === 'admin'
  // 当前用户是否是账本所有者（只有所有者能修改角色）
  const isOwner = currentTenant?.owner_id === userId

  const loadData = useCallback(async () => {
    if (!currentTenantId) return
    setLoading(true)
    try {
      const [membersRes, rolesRes] = await Promise.all([
        tenantApi.getMembers(currentTenantId),
        rbacApi.listRoles(currentTenantId),
      ])
      setMembers(membersRes.data.data ?? [])
      setRoles(rolesRes.data.data ?? [])
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      if (status === 403) {
        alert('权限不足：您没有权限查看成员列表')
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

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault()
    if (!inviteUsername.trim()) {
      alert('请输入用户名')
      return
    }
    if (!currentTenantId) return
    setInviting(true)
    try {
      await tenantApi.addMember(currentTenantId, { username: inviteUsername.trim(), role: inviteRole })
      setInviteUsername('')
      loadData()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '邀请失败')
    } finally {
      setInviting(false)
    }
  }

  async function handleRemove(memberId: number) {
    if (!confirm('确认移除该成员？')) return
    if (!currentTenantId) return
    setRemovingId(memberId)
    try {
      await tenantApi.removeMember(currentTenantId, memberId)
      loadData()
    } catch {
      alert('移除失败')
    } finally {
      setRemovingId(null)
    }
  }

  async function handleRoleChange(memberId: number, newRole: string) {
    if (!currentTenantId) return
    setUpdatingRoleId(memberId)
    try {
      await tenantApi.updateMemberRole(currentTenantId, memberId, newRole)
      setMembers(prev => prev.map(m => m.user_id === memberId ? { ...m, role: newRole } : m))
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '修改角色失败')
    } finally {
      setUpdatingRoleId(null)
    }
  }

  return (
    <div>
      {/* 邀请成员表单（管理员可用） */}
      {isAdmin && (
        <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100 mb-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">邀请成员</h3>
          <form onSubmit={handleInvite} className="flex flex-col sm:flex-row gap-3">
            <input
              type="text"
              value={inviteUsername}
              onChange={e => setInviteUsername(e.target.value)}
              placeholder="输入用户名"
              className="flex-1 px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select
              value={inviteRole}
              onChange={e => setInviteRole(e.target.value)}
              className="px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {roles.map(r => (
                <option key={r.id} value={r.name}>{ROLE_LABELS[r.name] || r.name}</option>
              ))}
            </select>
            <button
              type="submit"
              disabled={inviting}
              className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white text-sm font-medium rounded-xl transition-colors"
            >
              {inviting ? '邀请中...' : '邀请'}
            </button>
          </form>
        </div>
      )}

      {/* 成员列表 */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
        <div className="px-5 py-3 bg-gray-50 border-b border-gray-100">
          <span className="text-sm font-medium text-gray-500">共 {members.length} 位成员</span>
        </div>

        {loading ? (
          <div className="py-10 text-center text-gray-400">加载中...</div>
        ) : (
          <ul className="divide-y divide-gray-50">
            {members.map(m => {
              const isSelf = m.user_id === userId
              const isMemberOwner = currentTenant?.owner_id === m.user_id
              const canChangeRole = isOwner && !isSelf && !isMemberOwner
              const canRemove = isAdmin && !isSelf && !isMemberOwner

              return (
                <li key={m.user_id} className="flex items-center justify-between px-5 py-4 gap-3">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="w-9 h-9 rounded-full bg-blue-100 flex items-center justify-center text-blue-600 font-semibold text-sm shrink-0">
                      {m.username.charAt(0).toUpperCase()}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-gray-800 truncate">
                        {m.username}
                        {isSelf && <span className="ml-1 text-xs text-gray-400">（我）</span>}
                        {isMemberOwner && <span className="ml-1 text-xs text-amber-500">所有者</span>}
                      </p>
                      <p className="text-xs text-gray-400">
                        加入于 {m.joined_at?.slice(0, 10)}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    {/* 角色：所有者可改，其他只读 */}
                    {canChangeRole ? (
                      <select
                        value={m.role}
                        disabled={updatingRoleId === m.user_id}
                        onChange={e => handleRoleChange(m.user_id, e.target.value)}
                        className="px-3 py-1.5 border border-gray-200 rounded-lg text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white disabled:opacity-50"
                      >
                        {roles.map(r => (
                          <option key={r.id} value={r.name}>{ROLE_LABELS[r.name] || r.name}</option>
                        ))}
                      </select>
                    ) : (
                      <span className={`px-2.5 py-1 rounded-full text-xs font-medium ${
                        m.role === 'admin'
                          ? 'bg-blue-50 text-blue-600'
                          : m.role === 'finance'
                          ? 'bg-purple-50 text-purple-600'
                          : m.role === 'editor'
                          ? 'bg-green-50 text-green-600'
                          : m.role === 'partner'
                          ? 'bg-amber-50 text-amber-600'
                          : 'bg-gray-100 text-gray-500'
                      }`}>
                        {ROLE_LABELS[m.role] ?? m.role}
                      </span>
                    )}

                    {canRemove && (
                      <button
                        onClick={() => handleRemove(m.user_id)}
                        disabled={removingId === m.user_id}
                        className="text-sm text-red-500 hover:text-red-700 disabled:opacity-50 transition-colors"
                      >
                        {removingId === m.user_id ? '移除中' : '移除'}
                      </button>
                    )}
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
