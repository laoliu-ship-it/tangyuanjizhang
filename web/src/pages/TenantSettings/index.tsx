import { useState, useEffect } from 'react'
import Members from './Members'
import Categories from './Categories'
import LLMSettings from './LLMSettings'
import Roles from './Roles'
import { tenantApi } from '../../services/api'
import { useTenantStore } from '../../store/tenant'

type Tab = 'members' | 'roles' | 'categories' | 'llm'

const SUFFIX = '的记账本'

function TenantRename() {
  const { currentTenantId, tenants, setTenants } = useTenantStore()
  const currentTenant = tenants.find(t => t.id === currentTenantId)

  const currentPrefix = currentTenant?.name?.endsWith(SUFFIX)
    ? currentTenant.name.slice(0, -SUFFIX.length)
    : (currentTenant?.name ?? '')

  const [prefix, setPrefix] = useState(currentPrefix)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setPrefix(currentPrefix)
  }, [currentTenantId, currentPrefix])

  async function handleSave() {
    const trimmed = prefix.trim()
    if (!trimmed) { alert('账本名称不能为空'); return }
    if (!currentTenantId) return
    setSaving(true)
    try {
      await tenantApi.update(currentTenantId, { name: trimmed + SUFFIX })
      setTenants(tenants.map(t => t.id === currentTenantId ? { ...t, name: trimmed + SUFFIX } : t))
    } catch {
      alert('保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100 mb-6">
      <h3 className="text-sm font-semibold text-gray-700 mb-3">账本名称</h3>
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={prefix}
          onChange={e => setPrefix(e.target.value)}
          placeholder="输入账本名称前缀"
          className="flex-1 px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <span className="text-sm text-gray-400 whitespace-nowrap px-1">{SUFFIX}</span>
        <button
          onClick={handleSave}
          disabled={saving || prefix.trim() === currentPrefix}
          className="px-4 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-200 disabled:text-gray-400 text-white text-sm font-medium rounded-xl transition-colors whitespace-nowrap"
        >
          {saving ? '保存中…' : '保存'}
        </button>
      </div>
      <p className="mt-2 text-xs text-gray-400">预览：<span className="text-gray-600 font-medium">{(prefix.trim() || '…') + SUFFIX}</span></p>
    </div>
  )
}

export default function TenantSettings() {
  const [activeTab, setActiveTab] = useState<Tab>('members')

  return (
    <div className="p-4 md:p-6 max-w-3xl mx-auto">
      <h1 className="text-xl md:text-2xl font-bold text-gray-800 mb-5">账本设置</h1>
      <TenantRename />

      {/* Tab 切换 */}
      <div className="flex bg-gray-100 rounded-xl p-1 mb-6 w-fit">
        <button
          onClick={() => setActiveTab('members')}
          className={`px-6 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'members'
              ? 'bg-white text-gray-800 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          成员管理
        </button>
        <button
          onClick={() => setActiveTab('roles')}
          className={`px-6 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'roles'
              ? 'bg-white text-gray-800 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          角色权限
        </button>
        <button
          onClick={() => setActiveTab('categories')}
          className={`px-6 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'categories'
              ? 'bg-white text-gray-800 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          分类管理
        </button>
        <button
          onClick={() => setActiveTab('llm')}
          className={`px-6 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'llm'
              ? 'bg-white text-gray-800 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          AI 设置
        </button>
      </div>

      {activeTab === 'members' && <Members />}
      {activeTab === 'roles' && <Roles />}
      {activeTab === 'categories' && <Categories />}
      {activeTab === 'llm' && <LLMSettings />}
    </div>
  )
}
