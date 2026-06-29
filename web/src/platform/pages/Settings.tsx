import { useEffect, useState } from 'react'
import { platformAdminApi, type PlatformConfigItem, type PlatformLLMConfig } from '../services/api'

export default function PlatformSettings() {
  const [configs, setConfigs] = useState<PlatformConfigItem[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)

  const [llmConfig, setLlmConfig] = useState<PlatformLLMConfig | null>(null)
  const [llmForm, setLlmForm] = useState({
    enabled: false,
    provider: 'openai',
    base_url: '',
    api_key: '',
    model: 'gpt-4o',
    mode: 'ocr_text',
  })
  const [llmSaving, setLlmSaving] = useState(false)

  useEffect(() => {
    loadConfigs()
    loadLLMConfig()
  }, [])

  function loadConfigs() {
    setLoading(true)
    platformAdminApi.getConfigs()
      .then(res => setConfigs(res.data.data?.items || []))
      .catch(() => alert('加载配置失败，请刷新页面重试'))
      .finally(() => setLoading(false))
  }

  function loadLLMConfig() {
    platformAdminApi.getPlatformLLMConfig()
      .then(res => {
        const cfg = res.data.data
        if (cfg) {
          setLlmConfig(cfg)
          setLlmForm({
            enabled: cfg.enabled,
            provider: cfg.provider || 'openai',
            base_url: cfg.base_url || '',
            api_key: '',
            model: cfg.model || 'gpt-4o',
            mode: cfg.mode || 'ocr_text',
          })
        }
      })
      .catch(() => alert('加载 LLM 配置失败，请刷新页面重试'))
  }

  function handleSaveLLM() {
    setLlmSaving(true)
    platformAdminApi.updatePlatformLLMConfig(llmForm)
      .then(res => {
        const cfg = res.data.data
        if (cfg) {
          setLlmConfig(cfg)
          setLlmForm(prev => ({ ...prev, api_key: '' }))
        }
        alert('保存成功')
      })
      .catch((err: unknown) => {
        const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
        alert(msg ? `保存失败：${msg}` : '保存失败，请检查网络或重新登录')
      })
      .finally(() => setLlmSaving(false))
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

  // 积分计费配置项
  const creditConfigs = [
    { key: 'credits_per_upload', label: '上传图片（积分/张）', desc: '上传一张新图片扣除的积分' },
    { key: 'credits_per_ocr', label: 'OCR 识别（积分/次）', desc: '每次 OCR 识别扣除的积分' },
    { key: 'credits_per_llm', label: 'AI 整理（积分/次）', desc: '每次 AI 整理分析扣除的积分' },
    { key: 'credits_per_gb_per_day', label: '存储费用（积分/G/天）', desc: '每 GB 存储每天扣除积分（预留）' },
    { key: 'credits_new_user_gift', label: '新用户赠送积分', desc: '新用户注册时自动赠送的积分数量' },
  ]

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

        {/* 平台 LLM 配置 */}
        <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
          <h3 className="text-lg font-semibold text-gray-700 mb-4">平台 AI 分析配置</h3>
          <p className="text-sm text-gray-400 mb-5">配置平台默认 LLM，租户选择"使用平台 AI"时将使用此配置</p>

          <div className="space-y-4">
            {/* 启用开关 */}
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-700">启用平台 AI</p>
                <p className="text-xs text-gray-400">关闭后租户只能使用自己的 API Key</p>
              </div>
              <button
                disabled={llmSaving}
                onClick={() => {
                  const next = { ...llmForm, enabled: !llmForm.enabled }
                  setLlmForm(next)
                  setLlmSaving(true)
                  platformAdminApi.updatePlatformLLMConfig(next)
                    .then(res => {
                      const cfg = res.data.data
                      if (cfg) setLlmConfig(cfg)
                    })
                    .catch(() => {
                      setLlmForm(prev => ({ ...prev, enabled: !prev.enabled }))
                      alert('切换失败，请重试')
                    })
                    .finally(() => setLlmSaving(false))
                }}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  llmForm.enabled ? 'bg-green-500 text-white' : 'bg-gray-200 text-gray-600'
                } ${llmSaving ? 'opacity-50' : ''}`}
              >
                {llmForm.enabled ? '已启用' : '已禁用'}
              </button>
            </div>

            {/* Provider */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Provider</label>
              <select
                value={llmForm.provider}
                onChange={e => setLlmForm(prev => ({ ...prev, provider: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm"
              >
                <option value="openai">OpenAI</option>
                <option value="azure">Azure OpenAI</option>
                <option value="deepseek">DeepSeek</option>
                <option value="custom">自定义</option>
              </select>
            </div>

            {/* Base URL */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Base URL（可选）</label>
              <input
                type="text"
                placeholder="https://api.openai.com/v1"
                value={llmForm.base_url}
                onChange={e => setLlmForm(prev => ({ ...prev, base_url: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm"
              />
              <p className="text-xs text-gray-400 mt-1">留空使用 Provider 默认端点</p>
            </div>

            {/* API Key */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">
                API Key
                {llmConfig?.api_key_mask && (
                  <span className="ml-2 text-xs text-blue-500 font-normal">当前：{llmConfig.api_key_mask}</span>
                )}
              </label>
              <input
                type="password"
                placeholder="留空不修改已保存的 Key"
                value={llmForm.api_key}
                onChange={e => setLlmForm(prev => ({ ...prev, api_key: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm"
              />
            </div>

            {/* Model */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">模型</label>
              <input
                type="text"
                placeholder="gpt-4o"
                value={llmForm.model}
                onChange={e => setLlmForm(prev => ({ ...prev, model: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm"
              />
            </div>

            {/* Mode */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-2">调用模式</label>
              <div className="flex gap-2">
                <button
                  onClick={() => setLlmForm(prev => ({ ...prev, mode: 'ocr_text' }))}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    llmForm.mode === 'ocr_text' ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  OCR 文本
                </button>
                <button
                  onClick={() => setLlmForm(prev => ({ ...prev, mode: 'image' }))}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    llmForm.mode === 'image' ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  直接识图
                </button>
              </div>
              <p className="text-xs text-gray-400 mt-1">OCR 文本：先 OCR 再让 LLM 分析文字；直接识图：图片直接发给 LLM（需视觉模型）</p>
            </div>

            <div className="pt-2">
              <button
                onClick={handleSaveLLM}
                disabled={llmSaving}
                className="px-6 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {llmSaving ? '保存中...' : '保存 LLM 配置'}
              </button>
            </div>
          </div>
        </div>

        {/* 积分计费配置 */}
        <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
          <h3 className="text-lg font-semibold text-gray-700 mb-4">积分计费设置</h3>
          <p className="text-sm text-gray-400 mb-4">设置各类操作消耗的积分数，设为 0 则该操作免费</p>
          <div className="space-y-4">
            {creditConfigs.map(({ key, label, desc }) => {
              const cfg = configs.find(c => c.key === key)
              return (
                <div key={key} className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-700">{label}</p>
                    <p className="text-xs text-gray-400">{desc}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min="0"
                      value={cfg?.value ?? ''}
                      onChange={e => {
                        const value = e.target.value
                        setConfigs(prev => prev.map(c => c.key === key ? { ...c, value } : c))
                      }}
                      className="w-24 px-3 py-2 border border-gray-200 rounded-lg text-sm text-right"
                    />
                    <button
                      onClick={() => handleUpdate(key, cfg?.value ?? '0')}
                      disabled={saving === key}
                      className="px-3 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
                    >
                      {saving === key ? '保存中...' : '保存'}
                    </button>
                  </div>
                </div>
              )
            })}
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