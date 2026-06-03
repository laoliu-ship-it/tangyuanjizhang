import { useEffect, useState } from 'react'
import { platformAdminApi, type PlatformDashboardResp } from '../services/api'

export default function PlatformDashboard() {
  const [stats, setStats] = useState<PlatformDashboardResp | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    platformAdminApi.getDashboard()
      .then(res => setStats(res.data.data))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

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

  const cards = [
    { label: '注册用户数', value: stats?.total_users ?? 0, color: 'text-blue-600', bg: 'bg-blue-50' },
    { label: '注册账本数', value: stats?.total_tenants ?? 0, color: 'text-green-600', bg: 'bg-green-50' },
    { label: '记账总数', value: stats?.total_transactions ?? 0, color: 'text-purple-600', bg: 'bg-purple-50' },
  ]

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold text-gray-800 mb-6">平台概览</h2>
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
