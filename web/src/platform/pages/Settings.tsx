import { useEffect, useState } from 'react'
import { platformAdminApi, type PlatformConfigItem } from '../services/api'

export default function PlatformSettings() {
  const [configs, setConfigs] = useState<PlatformConfigItem[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)

  useEffect(() => {
    loadConfigs()
  }, [])

  function loadConfigs() {
    setLoading(true)
    platformAdminApi.getConfigs()
      .then(res => setConfigs(res.data.data?.items || []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  function handleUpdate(key: string, value: string) {
    setSaving(key)
    platformAdminApi.updateConfig(key, value)
      .then(() => {
        setConfigs(prev => prev.map(c => c.key === key ? { ...c, value } : c))
      })
      .catch(() => alert('更新失败'))
      .finally(() => setSaving(null))
  }

  if (loading) {
    return (
      <div className="p-6">
        <div className="space-y-4">
          {[1, 2, 3, 4].map(i => (
            <div key={i} className="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
              <div className="h-4 bg-gray-100 rounded animate-pulse w-32 mb-3" />
              <div className="h-8 bg-gray-100 rounded animate-pulse w-full" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  // 缓存类型配置项
  const cacheTypeConfig = configs.find(c => c.key === 'cache_type')
  const ttlConfig = configs.find(c => c.key === 'cache_ttl_minutes')
  const ocrEnabledConfig = configs.find(c => c.key === 'ocr_cache_enabled')
  const llmEnabledConfig = configs.find(c => c.key === 'llm_cache_enabled')

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold text-gray-800 mb-6">平台配置</h2>

      <div className="space-y-6">
        {/* 缓存配置 */}
        <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
          <h3 className="text-lg font-semibold text-gray-700 mb-4">缓存设置</h3>

          {/* 缓存类型 */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-600 mb-2">缓存类型</label>
            <div className="flex gap-2">
              <button
                onClick={() => handleUpdate('cache_type', 'file')}
                disabled={saving === 'cache_type' || cacheTypeConfig?.value === 'file'}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  cacheTypeConfig?.value === 'file'
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                } ${saving === 'cache_type' ? 'opacity-50' : ''}`}
              >
                文件缓存 (SHA256)
              </button>
              <button
                onClick={() => handleUpdate('cache_type', 'text')}
                disabled={saving === 'cache_type' || cacheTypeConfig?.value === 'text'}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  cacheTypeConfig?.value === 'text'
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                } ${saving === 'cache_type' ? 'opacity-50' : ''}`}
              >
                文本缓存 (OCR内容)
              </button>
            </div>
            <p className="text-xs text-gray-400 mt-1">
              文件缓存：相同图片复用缓存 · 文本缓存：相同票据内容复用缓存（推荐）
            </p>
          </div>

          {/* 缓存时间 */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-600 mb-2">缓存有效期（分钟）</label>
            <div className="flex items-center gap-3">
              <input
                type="number"
                min="1"
                max="1440"
                value={ttlConfig?.value || '30'}
                onChange={e => {
                  const value = e.target.value
                  setConfigs(prev => prev.map(c => c.key === 'cache_ttl_minutes' ? { ...c, value } : c))
                }}
                className="w-20 px-3 py-2 border border-gray-200 rounded-lg text-sm"
              />
              <button
                onClick={() => handleUpdate('cache_ttl_minutes', ttlConfig?.value || '30')}
                disabled={saving === 'cache_ttl_minutes'}
                className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {saving === 'cache_ttl_minutes' ? '保存中...' : '保存'}
              </button>
            </div>
            <p className="text-xs text-gray-400 mt-1">默认 30 分钟，最大 1440 分钟（24小时）</p>
          </div>

          {/* 缓存开关 */}
          <div className="flex gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-2">OCR 缓存</label>
              <button
                onClick={() => handleUpdate('ocr_cache_enabled', ocrEnabledConfig?.value === 'true' ? 'false' : 'true')}
                disabled={saving === 'ocr_cache_enabled'}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  ocrEnabledConfig?.value === 'true'
                    ? 'bg-green-500 text-white'
                    : 'bg-gray-200 text-gray-600'
                } ${saving === 'ocr_cache_enabled' ? 'opacity-50' : ''}`}
              >
                {ocrEnabledConfig?.value === 'true' ? '已开启' : '已关闭'}
              </button>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-2">LLM 缓存</label>
              <button
                onClick={() => handleUpdate('llm_cache_enabled', llmEnabledConfig?.value === 'true' ? 'false' : 'true')}
                disabled={saving === 'llm_cache_enabled'}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  llmEnabledConfig?.value === 'true'
                    ? 'bg-green-500 text-white'
                    : 'bg-gray-200 text-gray-600'
                } ${saving === 'llm_cache_enabled' ? 'opacity-50' : ''}`}
              >
                {llmEnabledConfig?.value === 'true' ? '已开启' : '已关闭'}
              </button>
            </div>
          </div>
        </div>

        {/* 所有配置列表 */}
        <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
          <h3 className="text-lg font-semibold text-gray-700 mb-4">全部配置</h3>
          <div className="space-y-3">
            {configs.map(config => (
              <div key={config.key} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0">
                <div>
                  <p className="text-sm font-medium text-gray-700">{config.key}</p>
                  <p className="text-xs text-gray-400">{config.description}</p>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-sm font-medium text-blue-600">{config.value}</span>
                  <span className="text-xs text-gray-400">{config.updated_at}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}