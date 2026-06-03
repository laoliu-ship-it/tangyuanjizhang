import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { platformAdminApi, type PlatformUserItem } from '../services/api'

export default function PlatformUsers() {
  const navigate = useNavigate()
  const [users, setUsers] = useState<PlatformUserItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(true)

  const fetchUsers = useCallback(() => {
    setLoading(true)
    platformAdminApi.listUsers({ keyword: keyword || undefined, page, page_size: 20 })
      .then(res => {
        setUsers(res.data.data.items)
        setTotal(res.data.data.total)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [keyword, page])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  const totalPages = Math.ceil(total / 20)

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold text-gray-800 mb-6">用户管理</h2>

      {/* 搜索 */}
      <div className="mb-6">
        <input
          type="text"
          value={keyword}
          onChange={e => { setKeyword(e.target.value); setPage(1) }}
          placeholder="搜索用户名或邮箱"
          className="w-full max-w-md px-4 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
        />
      </div>

      {/* 表格 */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">ID</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">用户名</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">邮箱</th>
              <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase">注册时间</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">加载中...</td></tr>
            ) : users.length === 0 ? (
              <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">暂无用户</td></tr>
            ) : (
              users.map(user => (
                <tr
                  key={user.id}
                  onClick={() => navigate(`/platform/users/${user.id}`)}
                  className="border-b border-gray-50 hover:bg-blue-50 cursor-pointer transition-colors"
                >
                  <td className="px-6 py-4 text-sm text-gray-500">{user.id}</td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-800">{user.username}</td>
                  <td className="px-6 py-4 text-sm text-gray-600">{user.email}</td>
                  <td className="px-6 py-4 text-sm text-gray-500">{user.created_at}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-gray-500">
            共 {total} 条记录，第 {page} / {totalPages} 页
          </p>
          <div className="flex gap-2">
            <button
              disabled={page <= 1}
              onClick={() => setPage(p => p - 1)}
              className="px-4 py-2 text-sm border border-gray-300 rounded-xl disabled:opacity-50 hover:bg-gray-50"
            >
              上一页
            </button>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage(p => p + 1)}
              className="px-4 py-2 text-sm border border-gray-300 rounded-xl disabled:opacity-50 hover:bg-gray-50"
            >
              下一页
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
