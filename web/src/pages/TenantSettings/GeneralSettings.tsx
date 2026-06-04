import { useState, useEffect } from 'react'
import { tenantApi } from '../../services/api'
import { useTenantStore } from '../../store/tenant'

export default function GeneralSettings() {
  const { currentTenantId } = useTenantStore()
  const [requireExpenseImage, setRequireExpenseImage] = useState(true)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!currentTenantId) return
    setLoading(true)
    tenantApi.getSettings(currentTenantId).then(r => {
      setRequireExpenseImage(r.data.data?.require_expense_image ?? true)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [currentTenantId])

  async function handleSave() {
    if (!currentTenantId) return
    setSaving(true)
    try {
      const r = await tenantApi.updateSettings(currentTenantId, { require_expense_image: requireExpenseImage })
      setRequireExpenseImage(r.data.data.require_expense_image)
      alert('保存成功')
    } catch {
      alert('保存失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="py-10 text-center text-gray-400">加载中...</div>

  return (
    <div className="space-y-5">
      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-semibold text-gray-700">支出记录必须上传图片</p>
            <p className="text-xs text-gray-400 mt-0.5">开启后，新增支出时若未上传凭证图片将无法保存</p>
          </div>
          <button
            type="button"
            onClick={() => setRequireExpenseImage(v => !v)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              requireExpenseImage ? 'bg-blue-600' : 'bg-gray-200'
            }`}
          >
            <span className={`inline-block h-4 w-4 rounded-full bg-white shadow transition-transform ${
              requireExpenseImage ? 'translate-x-6' : 'translate-x-1'
            }`} />
          </button>
        </div>
      </div>

      <button
        type="button"
        onClick={handleSave}
        disabled={saving}
        className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-200 disabled:text-gray-400 text-white text-sm font-medium rounded-xl transition-colors"
      >
        {saving ? '保存中…' : '保存配置'}
      </button>
    </div>
  )
}
