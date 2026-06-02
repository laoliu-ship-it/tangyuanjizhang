import { useState, useEffect, useCallback } from 'react'
import { llmApi, type TenantLLMConfig } from '../../services/api'

const PROVIDERS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'azure', label: 'Azure OpenAI' },
  { value: 'ollama', label: 'Ollama（本地）' },
]

const MODES = [
  { value: 'ocr_text', label: '文字模式（将 OCR 文本发给 AI）' },
  { value: 'vision', label: '视觉模式（将图片直接发给 AI）' },
]

export default function LLMSettings() {
  const [config, setConfig] = useState<TenantLLMConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState(false)
  const [saving, setSaving] = useState(false)

  // 表单状态
  const [enabled, setEnabled] = useState(false)
  const [usePlatform, setUsePlatform] = useState(true)
  const [provider, setProvider] = useState('openai')
  const [baseURL, setBaseURL] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState('gpt-4o')
  const [mode, setMode] = useState<'vision' | 'ocr_text'>('ocr_text')

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setLoadError(false)
    try {
      const res = await llmApi.getConfig()
      const cfg = res.data.data
      setConfig(cfg)
      setEnabled(cfg.enabled)
      setUsePlatform(cfg.use_platform ?? true)
      setProvider(cfg.provider || 'openai')
      setBaseURL(cfg.base_url || '')
      setModel(cfg.model || 'gpt-4o')
      setMode((cfg.mode as 'vision' | 'ocr_text') || 'ocr_text')
    } catch {
      setLoadError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    // 选了平台但平台未配置，阻止保存并提示
    if (enabled && usePlatform && config && !config.platform_enabled) {
      alert('平台暂未开放 AI 分析功能，请联系管理员，或切换为"使用自定义配置"。')
      return
    }
    setSaving(true)
    try {
      const payload: Parameters<typeof llmApi.saveConfig>[0] = {
        enabled,
        use_platform: usePlatform,
        provider,
        base_url: baseURL,
        model,
        mode,
      }
      if (apiKey.trim()) {
        payload.api_key = apiKey.trim()
      }
      const res = await llmApi.saveConfig(payload)
      setConfig(res.data.data)
      setApiKey('') // 保存成功后清空明文 key
      alert('保存成功')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <div className="py-10 text-center text-gray-400">加载中...</div>
  }

  if (loadError) {
    return (
      <div className="py-10 text-center">
        <p className="text-gray-500 mb-4">加载 AI 配置失败，请重试</p>
        <button
          type="button"
          onClick={loadConfig}
          className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-xl transition-colors"
        >
          重试
        </button>
      </div>
    )
  }

  const platformUnavailable = enabled && usePlatform && config && !config.platform_enabled

  return (
    <form onSubmit={handleSave} className="space-y-5">
      {/* 启用开关 */}
      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-semibold text-gray-700">启用 AI 分析</p>
            <p className="text-xs text-gray-400 mt-0.5">上传票据时自动调用大模型识别金额、商户、分类</p>
          </div>
          <button
            type="button"
            onClick={() => setEnabled(v => !v)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              enabled ? 'bg-blue-600' : 'bg-gray-200'
            }`}
          >
            <span className={`inline-block h-4 w-4 rounded-full bg-white shadow transition-transform ${
              enabled ? 'translate-x-6' : 'translate-x-1'
            }`} />
          </button>
        </div>
      </div>

      {enabled && (
        <>
          {/* 使用平台 or 自定义 */}
          <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
            <p className="text-sm font-semibold text-gray-700 mb-3">API 来源</p>
            <div className="space-y-2">
              <label className="flex items-start gap-3 cursor-pointer">
                <input
                  type="radio"
                  name="source"
                  checked={usePlatform}
                  onChange={() => setUsePlatform(true)}
                  className="mt-0.5"
                />
                <div>
                  <p className="text-sm font-medium text-gray-700">使用平台提供的 AI</p>
                  <p className="text-xs text-gray-400">无需配置，直接使用平台统一服务</p>
                  {/* 平台未配置时的警告 */}
                  {usePlatform && config && !config.platform_enabled && (
                    <p className="text-xs text-amber-600 bg-amber-50 rounded-lg px-3 py-2 mt-2 border border-amber-200">
                      平台暂未开放 AI 分析功能，请联系管理员，或切换为自定义配置。
                    </p>
                  )}
                </div>
              </label>
              <label className="flex items-start gap-3 cursor-pointer">
                <input
                  type="radio"
                  name="source"
                  checked={!usePlatform}
                  onChange={() => setUsePlatform(false)}
                  className="mt-0.5"
                />
                <div>
                  <p className="text-sm font-medium text-gray-700">使用自定义 API Key</p>
                  <p className="text-xs text-gray-400">使用自己的 OpenAI / DeepSeek 等账号</p>
                </div>
              </label>
            </div>
          </div>

          {/* 自定义配置 */}
          {!usePlatform && (
            <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100 space-y-4">
              <p className="text-sm font-semibold text-gray-700">自定义配置</p>

              {/* Provider */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">Provider</label>
                <select
                  value={provider}
                  onChange={e => setProvider(e.target.value)}
                  className="w-full px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {PROVIDERS.map(p => (
                    <option key={p.value} value={p.value}>{p.label}</option>
                  ))}
                </select>
              </div>

              {/* Base URL */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">
                  API Base URL
                  <span className="text-gray-400 ml-1">（OpenAI 兼容接口留空即可）</span>
                </label>
                <input
                  type="url"
                  value={baseURL}
                  onChange={e => setBaseURL(e.target.value)}
                  placeholder="https://api.openai.com/v1"
                  className="w-full px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* API Key */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">
                  API Key
                  {config?.api_key_mask && (
                    <span className="ml-2 text-gray-400">当前：{config.api_key_mask}</span>
                  )}
                </label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  placeholder={config?.api_key_mask ? '不修改请留空' : '输入 API Key'}
                  className="w-full px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* Model */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">模型名称</label>
                <input
                  type="text"
                  value={model}
                  onChange={e => setModel(e.target.value)}
                  placeholder="gpt-4o"
                  className="w-full px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          )}

          {/* 识别模式 */}
          <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
            <p className="text-sm font-semibold text-gray-700 mb-3">识别模式</p>
            <div className="space-y-2">
              {MODES.map(m => (
                <label key={m.value} className="flex items-center gap-3 cursor-pointer">
                  <input
                    type="radio"
                    name="mode"
                    value={m.value}
                    checked={mode === m.value}
                    onChange={() => setMode(m.value as 'vision' | 'ocr_text')}
                  />
                  <span className="text-sm text-gray-700">{m.label}</span>
                </label>
              ))}
            </div>
            {mode === 'vision' && (
              <p className="text-xs text-amber-600 mt-2">视觉模式需要模型支持图片输入（如 gpt-4o、claude-3 等）</p>
            )}
          </div>
        </>
      )}

      <button
        type="submit"
        disabled={saving || !!platformUnavailable}
        className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-200 disabled:text-gray-400 text-white text-sm font-medium rounded-xl transition-colors"
      >
        {saving ? '保存中…' : '保存配置'}
      </button>
    </form>
  )
}
