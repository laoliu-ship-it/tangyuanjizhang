import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { platformAdminApi, type PlatformUserDetailResp } from '../services/api'

export default function PlatformUserDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [user, setUser] = useState<PlatformUserDetailResp | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    platformAdminApi.getUserDetail(Number(id))
      .then(res => setUser(res.data.data))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="p-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <div key={i} className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
              <div className="h-4 bg-gray-100 rounded animate-pulse w-24 mb-3" />
              <div className="h-10 bg-gray-100 rounded animate-pulse w-32" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (!user) {
    return (
      <div className="p-6">
        <p className="text-gray-500">用户不存在</p>
        <button onClick={() => navigate('/platform/users')} className="mt-4 text-blue-600 hover:underline text-sm">
          返回用户列表
        </button>
      </div>
    )
  }

  const cards = [
    { label: '所属账本数', value: user.tenant_count, color: 'text-green-600' },
    { label: '记账数', value: user.transaction_count, color: 'text-blue-600' },
    { label: '图片资源量', value: user.media_count, color: 'text-purple-600' },
  ]

  return (
    <div className="p-6">
      <button onClick={() => navigate('/platform/users')} className="mb-4 text-blue-600 hover:underline text-sm">
        ← 返回用户列表
      </button>

      <h2 className="text-2xl font-bold text-gray-800 mb-2">{user.username}</h2>
      <p className="text-gray-500 mb-6">{user.email}</p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {cards.map(card => (
          <div key={card.label} className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
            <p className="text-sm text-gray-500 mb-2">{card.label}</p>
            <p className={`text-4xl font-bold ${card.color}`}>{card.value.toLocaleString()}</p>
          </div>
        ))}
      </div>
    </div>
  )
}
