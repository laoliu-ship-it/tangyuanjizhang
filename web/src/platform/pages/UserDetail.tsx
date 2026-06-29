import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { platformAdminApi, type PlatformUserDetailResp, type TenantCreditsDetail, type TenantPromptResp } from '../services/api'

export default function PlatformUserDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [user, setUser] = useState<PlatformUserDetailResp | null>(null)
  const [loading, setLoading] = useState(true)
  const [tenantCredits, setTenantCredits] = useState<TenantCreditsDetail[]>([])
  const [adjusting, setAdjusting] = useState<number | null>(null)
  const [adjustDelta, setAdjustDelta] = useState('')
  const [adjustNote, setAdjustNote] = useState('')
  const [adjustError, setAdjustError] = useState('')
  const [editingPrompt, setEditingPrompt] = useState<number | null>(null)
  const [promptData, setPromptData] = useState<TenantPromptResp | null>(null)
  const [promptText, setPromptText] = useState('')
  const [promptLoading, setPromptLoading] = useState(false)
  const [promptSaving, setPromptSaving] = useState(false)
  const [promptError, setPromptError] = useState('')

  useEffect(() => {
    if (!id) return
    platformAdminApi.getUserDetail(Number(id))
      .then(res => setUser(res.data.data))
      .catch(() => {})
      .finally(() => setLoading(false))

    platformAdminApi.listUserTenantCredits(Number(id))
      .then(res => setTenantCredits(res.data.data?.items || []))
      .catch(() => {})
  }, [id])

  function handleAdjust(tenantId: number) {
    const delta = parseInt(adjustDelta)
    if (isNaN(delta) || delta === 0) {
      setAdjustError('请输入有效的积分数（正数充值，负数扣减）')
      return
    }
    setAdjustError('')
    platformAdminApi.adjustTenantCredits(tenantId, delta, adjustNote || '管理员调整')
      .then(res => {
        setTenantCredits(prev => prev.map(t =>
          t.tenant_id === tenantId ? { ...t, balance: res.data.data?.balance ?? t.balance } : t
        ))
        setAdjusting(null)
        setAdjustDelta('')
        setAdjustNote('')
      })
      .catch(err => setAdjustError(err.message || '调整失败'))
  }

  async function handleOpenPrompt(tenantId: number) {
    setEditingPrompt(tenantId)
    setPromptData(null)
    setPromptText('')
    setPromptError('')
    setPromptLoading(true)
    try {
      const res = await platformAdminApi.getTenantPrompt(tenantId)
      const p = res.data.data
      setPromptData(p)
      setPromptText(p.custom_prompt || p.default_prompt || '')
    } catch {
      setPromptError('加载失败')
    } finally {
      setPromptLoading(false)
    }
  }

  async function handleSavePrompt(tenantId: number) {
    setPromptSaving(true)
    setPromptError('')
    try {
      const res = await platformAdminApi.updateTenantPrompt(tenantId, promptText.trim())
      const p = res.data.data
      setPromptData(p)
      setPromptText(p.custom_prompt || p.default_prompt || '')
    } catch {
      setPromptError('保存失败')
    } finally {
      setPromptSaving(false)
    }
  }

  async function handleResetPrompt(tenantId: number) {
    if (!confirm('确定恢复平台默认提示词吗？')) return
    setPromptSaving(true)
    setPromptError('')
    try {
      const res = await platformAdminApi.updateTenantPrompt(tenantId, '')
      const p = res.data.data
      setPromptData(p)
      setPromptText(p.default_prompt || '')
    } catch {
      setPromptError('操作失败')
    } finally {
      setPromptSaving(false)
    }
  }

  function handleToggleUnlimited(tenantId: number, current: boolean) {
    platformAdminApi.setTenantUnlimited(tenantId, !current)
      .then(res => {
        setTenantCredits(prev => prev.map(t =>
          t.tenant_id === tenantId ? { ...t, unlimited: res.data.data?.unlimited ?? !current } : t
        ))
      })
      .catch(() => alert('设置失败'))
  }

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

      {/* 账本积分管理 */}
      {tenantCredits.length > 0 && (
        <div className="mt-8 bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
          <h3 className="text-lg font-semibold text-gray-700 mb-4">账本积分管理</h3>
          <div className="space-y-4">
            {tenantCredits.map(t => (
              <div key={t.tenant_id} className="border border-gray-100 rounded-xl p-4">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <p className="font-medium text-gray-800">{t.tenant_name || `账本 #${t.tenant_id}`}</p>
                    <p className="text-sm text-gray-400">ID: {t.tenant_id}</p>
                  </div>
                  <div className="flex items-center gap-3">
                    {/* 无限制开关 */}
                    <button
                      onClick={() => handleToggleUnlimited(t.tenant_id, t.unlimited)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                        t.unlimited
                          ? 'bg-purple-100 text-purple-700 hover:bg-purple-200'
                          : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
                      }`}
                    >
                      {t.unlimited ? '无限制 ∞' : '有限制'}
                    </button>
                    {/* 余额 */}
                    <span className={`text-2xl font-bold ${t.unlimited ? 'text-purple-500' : 'text-blue-600'}`}>
                      {t.unlimited ? '∞' : t.balance.toLocaleString()}
                    </span>
                    <span className="text-sm text-gray-400">积分</span>
                    {/* 提示词按钮 */}
                    <button
                      onClick={() => editingPrompt === t.tenant_id ? setEditingPrompt(null) : handleOpenPrompt(t.tenant_id)}
                      className="px-3 py-1.5 bg-orange-50 text-orange-600 hover:bg-orange-100 rounded-lg text-sm font-medium"
                    >
                      提示词
                    </button>
                    {/* 调整按钮 */}
                    <button
                      onClick={() => { setAdjusting(t.tenant_id); setAdjustDelta(''); setAdjustNote(''); setAdjustError('') }}
                      className="px-3 py-1.5 bg-blue-50 text-blue-600 hover:bg-blue-100 rounded-lg text-sm font-medium"
                    >
                      调整
                    </button>
                  </div>
                </div>

                {/* 提示词内联编辑 */}
                {editingPrompt === t.tenant_id && (
                  <div className="mt-3 p-3 bg-orange-50 rounded-lg space-y-2">
                    {promptLoading ? (
                      <p className="text-sm text-gray-400">加载中...</p>
                    ) : (
                      <>
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-xs text-orange-700 font-medium">
                            {promptData?.custom_prompt ? '当前：自定义提示词' : '当前：平台默认提示词'}
                          </span>
                        </div>
                        <textarea
                          value={promptText}
                          onChange={e => setPromptText(e.target.value)}
                          rows={12}
                          spellCheck={false}
                          className="w-full px-3 py-2 border border-orange-200 rounded-lg text-xs font-mono leading-5 focus:outline-none focus:ring-2 focus:ring-orange-400 resize-y bg-white"
                        />
                        <p className="text-xs text-gray-400 text-right">{promptText.length} 字</p>
                        {promptError && <p className="text-xs text-red-500">{promptError}</p>}
                        <div className="flex gap-2">
                          <button
                            onClick={() => handleSavePrompt(t.tenant_id)}
                            disabled={promptSaving}
                            className="px-4 py-2 bg-orange-500 text-white rounded-lg text-sm font-medium hover:bg-orange-600 disabled:opacity-50"
                          >
                            {promptSaving ? '保存中…' : '保存'}
                          </button>
                          <button
                            onClick={() => handleResetPrompt(t.tenant_id)}
                            disabled={promptSaving || !promptData?.custom_prompt}
                            className="px-4 py-2 bg-white border border-orange-200 text-orange-600 rounded-lg text-sm hover:bg-orange-50 disabled:opacity-40"
                          >
                            恢复平台默认
                          </button>
                          <button
                            onClick={() => setEditingPrompt(null)}
                            className="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm hover:bg-gray-200 ml-auto"
                          >
                            关闭
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                )}

                {/* 内联调整表单 */}
                {adjusting === t.tenant_id && (
                  <div className="mt-3 p-3 bg-gray-50 rounded-lg space-y-2">
                    <div className="flex gap-2">
                      <input
                        type="number"
                        placeholder="积分数（正数充值，负数扣减）"
                        value={adjustDelta}
                        onChange={e => setAdjustDelta(e.target.value)}
                        className="flex-1 px-3 py-2 border border-gray-200 rounded-lg text-sm"
                      />
                      <input
                        type="text"
                        placeholder="备注（选填）"
                        value={adjustNote}
                        onChange={e => setAdjustNote(e.target.value)}
                        className="flex-1 px-3 py-2 border border-gray-200 rounded-lg text-sm"
                      />
                    </div>
                    {adjustError && <p className="text-xs text-red-500">{adjustError}</p>}
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleAdjust(t.tenant_id)}
                        className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600"
                      >
                        确认
                      </button>
                      <button
                        onClick={() => setAdjusting(null)}
                        className="px-4 py-2 bg-gray-200 text-gray-600 rounded-lg text-sm hover:bg-gray-300"
                      >
                        取消
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
