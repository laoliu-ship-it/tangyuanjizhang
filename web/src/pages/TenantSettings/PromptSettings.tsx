import { useState, useEffect, useCallback, useRef } from 'react'
import { llmApi, type TenantPrompt } from '../../services/api'

export default function PromptSettings() {
  const [prompt, setPrompt] = useState<TenantPrompt | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState(false)
  const [saving, setSaving] = useState(false)
  const [text, setText] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const isCustom = !!prompt?.custom_prompt
  const placeholder = prompt?.placeholder ?? '{{分类列表}}'

  const loadPrompt = useCallback(async () => {
    setLoading(true)
    setLoadError(false)
    try {
      const res = await llmApi.getPrompt()
      const p = res.data.data
      setPrompt(p)
      // 未自定义时以平台默认为编辑底稿，方便在其基础上小修小改
      setText(p.custom_prompt || p.default_prompt || '')
    } catch (err: unknown) {
      console.error('Prompt load failed:', (err as { message?: string })?.message)
      setLoadError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadPrompt()
  }, [loadPrompt])

  async function handleSave() {
    const trimmed = text.trim()
    if (!trimmed) {
      alert('提示词不能为空，如需恢复平台默认请点击"恢复平台默认"')
      return
    }
    setSaving(true)
    try {
      const res = await llmApi.savePrompt({ custom_prompt: trimmed })
      const p = res.data.data
      setPrompt(p)
      setText(p.custom_prompt || p.default_prompt || '')
      alert(p.custom_prompt ? '已保存自定义提示词' : '内容与平台默认一致，继续使用平台默认提示词')
    } catch (err: unknown) {
      alert((err as { message?: string })?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  async function handleReset() {
    if (!confirm('确定恢复平台默认提示词吗？自定义内容将被清除。')) return
    setSaving(true)
    try {
      const res = await llmApi.savePrompt({ custom_prompt: '' })
      const p = res.data.data
      setPrompt(p)
      setText(p.default_prompt || '')
      alert('已恢复平台默认提示词')
    } catch (err: unknown) {
      alert((err as { message?: string })?.message || '操作失败')
    } finally {
      setSaving(false)
    }
  }

  function insertPlaceholder() {
    const ta = textareaRef.current
    if (!ta) return
    const start = ta.selectionStart ?? text.length
    const end = ta.selectionEnd ?? text.length
    const next = text.slice(0, start) + placeholder + text.slice(end)
    setText(next)
    requestAnimationFrame(() => {
      ta.focus()
      ta.selectionStart = ta.selectionEnd = start + placeholder.length
    })
  }

  if (loading) {
    return <div className="py-10 text-center text-gray-400">加载中...</div>
  }

  if (loadError) {
    return (
      <div className="py-10 text-center">
        <p className="text-red-500 mb-2 font-medium">加载失败</p>
        <p className="text-gray-500 mb-4 text-sm">无法获取提示词配置（仅管理员可用）</p>
        <button
          type="button"
          onClick={loadPrompt}
          className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-xl transition-colors"
        >
          重试
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* 状态与说明 */}
      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <div className="flex items-center justify-between mb-2">
          <p className="text-sm font-semibold text-gray-700">识别提示词</p>
          <span
            className={`text-xs px-2.5 py-1 rounded-full font-medium ${
              isCustom ? 'bg-blue-50 text-blue-600' : 'bg-gray-100 text-gray-500'
            }`}
          >
            {isCustom ? '当前生效：自定义提示词' : '当前生效：平台默认'}
          </span>
        </div>
        <p className="text-xs text-gray-400 leading-5">
          截图识别时发给 AI 的指令，决定如何从票据中提取金额、商户、时间、分类等标准字段。
          可在平台默认提示词的基础上小修小改（例如补充你常用商户、备注对应的分类规则），保存后即用你的版本。
        </p>
        <div className="mt-3 p-3 bg-blue-50 border border-blue-100 rounded-lg">
          <p className="text-xs text-blue-800 font-medium mb-1">分类列表占位符</p>
          <p className="text-xs text-blue-700 leading-5">
            提示词中的 <code className="px-1 py-0.5 bg-white rounded border border-blue-200">{placeholder}</code>
            {' '}会在每次识别时自动替换为本账本的分类列表（id / 名称 / 收支类型），AI 根据每笔账的商户或备注从中推算
            category_id。若提示词中没有该占位符，分类列表会自动追加到提示词末尾。
          </p>
        </div>
      </div>

      {/* 编辑区 */}
      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <div className="flex items-center justify-between mb-2">
          <label className="text-xs text-gray-500">提示词内容</label>
          <button
            type="button"
            onClick={insertPlaceholder}
            className="text-xs px-2.5 py-1 rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600 transition-colors"
          >
            插入 {placeholder}
          </button>
        </div>
        <textarea
          ref={textareaRef}
          value={text}
          onChange={e => setText(e.target.value)}
          rows={20}
          spellCheck={false}
          className="w-full px-3 py-2.5 border border-gray-300 rounded-xl text-xs font-mono leading-5 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-y"
        />
        <p className="mt-1 text-xs text-gray-400 text-right">{text.length} 字</p>
      </div>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="flex-1 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-200 disabled:text-gray-400 text-white text-sm font-medium rounded-xl transition-colors"
        >
          {saving ? '保存中…' : '保存为自定义提示词'}
        </button>
        <button
          type="button"
          onClick={handleReset}
          disabled={saving || !isCustom}
          className="px-5 py-3 bg-white hover:bg-gray-50 disabled:text-gray-300 text-gray-600 text-sm font-medium rounded-xl border border-gray-200 transition-colors whitespace-nowrap"
        >
          恢复平台默认
        </button>
      </div>
    </div>
  )
}
