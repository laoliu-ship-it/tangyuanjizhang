import { useState, useEffect } from 'react'
import { importApi, type ColumnMapping, type ParseHeadersResult } from '../services/api'
import type { AxiosError } from 'axios'

interface Props {
  open: boolean
  file: File | null
  onClose: () => void
  onImported: () => void
}

const FIELD_DEFS = [
  { key: 'date'     as keyof ColumnMapping, label: '日期', required: true  },
  { key: 'type'     as keyof ColumnMapping, label: '类型', required: true  },
  { key: 'amount'   as keyof ColumnMapping, label: '金额', required: true  },
  { key: 'category' as keyof ColumnMapping, label: '分类', required: true  },
  { key: 'merchant' as keyof ColumnMapping, label: '商户', required: false },
  { key: 'note'     as keyof ColumnMapping, label: '备注', required: false },
]

type Step = 'mapping' | 'preview' | 'done'

interface PreviewData {
  valid_count: number
  skipped_count: number
  issues: string[]
}

interface DoneData {
  imported: number
  skipped_count: number
  issues: string[]
}

export default function ImportMappingDialog({ open, file, onClose, onImported }: Props) {
  const [step, setStep] = useState<Step>('mapping')
  const [parseResult, setParseResult] = useState<ParseHeadersResult | null>(null)
  const [mapping, setMapping] = useState<ColumnMapping>({ date: -1, type: -1, amount: -1, category: -1, merchant: -1, note: -1 })
  const [sheetIndex, setSheetIndex] = useState(0)

  const [parseLoading, setParseLoading] = useState(false)
  const [parseError, setParseError] = useState('')
  const [previewLoading, setPreviewLoading] = useState(false)
  const [importLoading, setImportLoading] = useState(false)

  const [preview, setPreview] = useState<PreviewData | null>(null)
  const [done, setDone] = useState<DoneData | null>(null)

  // 解析表头
  useEffect(() => {
    if (!open || !file) return
    let cancelled = false
    setParseLoading(true)
    setParseError('')
    importApi.parseHeaders(file, sheetIndex)
      .then(res => {
        if (cancelled) return
        const r = res.data.data
        setParseResult(r)
        setSheetIndex(r.sheet_index)
        setMapping(r.suggestions)
      })
      .catch((err: AxiosError<{ message?: string }>) => {
        if (!cancelled) {
          setParseError(err.response?.data?.message ?? '解析文件失败，请检查文件格式')
        }
      })
      .finally(() => { if (!cancelled) setParseLoading(false) })
    return () => { cancelled = true }
  }, [open, file, sheetIndex])

  // 关闭时重置
  useEffect(() => {
    if (!open) {
      setStep('mapping')
      setParseResult(null)
      setSheetIndex(0)
      setMapping({ date: -1, type: -1, amount: -1, category: -1, merchant: -1, note: -1 })
      setParseError('')
      setPreview(null)
      setDone(null)
    }
  }, [open])

  const canPreview = mapping.date >= 0 && mapping.type >= 0 && mapping.amount >= 0 && mapping.category >= 0

  async function handlePreview() {
    if (!file || !canPreview) return
    setPreviewLoading(true)
    try {
      const res = await importApi.importExcel(file, mapping, sheetIndex, true)
      const d = res.data.data
      setPreview({ valid_count: d.valid_count, skipped_count: d.skipped_count, issues: d.issues ?? [] })
      setStep('preview')
    } catch (err: unknown) {
      const axiosErr = err as AxiosError<{ message?: string }>
      setParseError(axiosErr.response?.data?.message ?? '预检失败，请检查文件内容')
    } finally {
      setPreviewLoading(false)
    }
  }

  async function handleImport() {
    if (!file) return
    setImportLoading(true)
    try {
      const res = await importApi.importExcel(file, mapping, sheetIndex, false)
      const d = res.data.data
      setDone({ imported: d.imported, skipped_count: d.skipped_count, issues: d.issues ?? [] })
      setStep('done')
      onImported()
    } catch (err: unknown) {
      const axiosErr = err as AxiosError<{ message?: string }>
      const msg = axiosErr.response?.data?.message ?? '导入失败，请检查文件内容或网络连接'
      setDone({ imported: -1, skipped_count: 0, issues: [msg] })
      setStep('done')
    } finally {
      setImportLoading(false)
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/50 overflow-y-auto py-6" onClick={onClose}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl mx-4 my-auto" onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-gray-800">Excel 导入</h2>
            {/* 步骤指示 */}
            <div className="flex items-center gap-1 text-xs text-gray-400">
              <StepDot active={step === 'mapping'} done={step !== 'mapping'} label="列映射" />
              <span className="text-gray-200">—</span>
              <StepDot active={step === 'preview'} done={step === 'done'} label="数据预检" />
              <span className="text-gray-200">—</span>
              <StepDot active={step === 'done'} done={false} label="导入结果" />
            </div>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl leading-none">✕</button>
        </div>

        <div className="px-6 py-5 space-y-4">

          {/* ========== Step 1: 列映射 ========== */}
          {step === 'mapping' && (
            <>
              {/* Sheet 切换 */}
              {parseResult && parseResult.sheets.length > 1 && (
                <div className="flex items-center gap-3">
                  <span className="text-sm text-gray-500 whitespace-nowrap">工作表：</span>
                  <div className="flex flex-wrap gap-2">
                    {parseResult.sheets.map((s, i) => (
                      <button key={i} onClick={() => setSheetIndex(i)}
                        className={`px-3 py-1 text-sm rounded-lg border transition-colors ${
                          sheetIndex === i ? 'bg-blue-600 text-white border-blue-600' : 'bg-white text-gray-600 border-gray-300 hover:border-blue-400'
                        }`}
                      >{s}</button>
                    ))}
                  </div>
                </div>
              )}

              {parseLoading && <div className="text-center py-8 text-gray-400 text-sm">解析文件中…</div>}
              {parseError && <div className="bg-red-50 text-red-600 text-sm px-4 py-3 rounded-xl">{parseError}</div>}

              {!parseLoading && parseResult && (
                <>
                  <div>
                    <p className="text-sm text-gray-500 mb-3">
                      已识别 <b className="text-gray-700">{parseResult.headers.length}</b> 列，请确认字段映射（带 <span className="text-red-400">*</span> 为必填）：
                    </p>
                    <div className="space-y-2">
                      {FIELD_DEFS.map(field => (
                        <div key={field.key} className="flex items-center gap-3">
                          <div className="w-16 text-right text-sm">
                            <span className={field.required ? 'font-medium text-gray-800' : 'text-gray-500'}>{field.label}</span>
                            {field.required && <span className="text-red-400 ml-0.5">*</span>}
                          </div>
                          <select
                            value={mapping[field.key]}
                            onChange={e => setMapping(prev => ({ ...prev, [field.key]: Number(e.target.value) }))}
                            className={`flex-1 px-3 py-1.5 text-sm border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                              field.required && mapping[field.key] < 0 ? 'border-red-300 bg-red-50' : 'border-gray-300'
                            }`}
                          >
                            <option value={-1}>— 不导入 —</option>
                            {parseResult.headers.map((h, i) => (
                              <option key={i} value={i}>{h || `列${i + 1}`}</option>
                            ))}
                          </select>
                          <div className="w-32 text-xs text-gray-400 truncate">
                            {mapping[field.key] >= 0 && parseResult.sample_rows.length > 0
                              ? parseResult.sample_rows.slice(0, 2).map(r => r[mapping[field.key]]).filter(Boolean).join(' / ') || '（空）'
                              : ''}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* 数据预览表格 */}
                  {parseResult.sample_rows.length > 0 && (
                    <div>
                      <p className="text-xs text-gray-400 mb-1.5">数据预览（前 {parseResult.sample_rows.length} 行）</p>
                      <div className="overflow-x-auto rounded-lg border border-gray-100">
                        <table className="w-full text-xs">
                          <thead>
                            <tr className="bg-gray-50">
                              {parseResult.headers.map((h, i) => (
                                <th key={i} className="px-3 py-2 text-left font-medium text-gray-500 whitespace-nowrap">{h || `列${i+1}`}</th>
                              ))}
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-gray-50">
                            {parseResult.sample_rows.map((row, ri) => (
                              <tr key={ri}>
                                {row.map((cell, ci) => (
                                  <td key={ci} className="px-3 py-2 text-gray-600 whitespace-nowrap max-w-[120px] truncate">{cell}</td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}
                </>
              )}
            </>
          )}

          {/* ========== Step 2: 预检结果 ========== */}
          {step === 'preview' && preview && (
            <div className="space-y-4">
              {/* 汇总 */}
              <div className="grid grid-cols-2 gap-3">
                <div className="bg-green-50 rounded-xl px-4 py-3 text-center">
                  <p className="text-2xl font-bold text-green-600">{preview.valid_count}</p>
                  <p className="text-xs text-green-700 mt-0.5">可导入行数</p>
                </div>
                <div className={`rounded-xl px-4 py-3 text-center ${preview.skipped_count > 0 ? 'bg-red-50' : 'bg-gray-50'}`}>
                  <p className={`text-2xl font-bold ${preview.skipped_count > 0 ? 'text-red-500' : 'text-gray-400'}`}>{preview.skipped_count}</p>
                  <p className={`text-xs mt-0.5 ${preview.skipped_count > 0 ? 'text-red-600' : 'text-gray-400'}`}>有问题将跳过</p>
                </div>
              </div>

              {/* 问题列表 */}
              {preview.issues.length > 0 ? (
                <div>
                  <p className="text-sm font-medium text-gray-700 mb-2">
                    发现 {preview.issues.length} 处问题（可修改 Excel 后重新上传，或直接跳过这些行继续导入）：
                  </p>
                  <div className="max-h-56 overflow-y-auto rounded-xl bg-amber-50 border border-amber-100 px-4 py-3 space-y-1.5">
                    {preview.issues.map((issue, i) => (
                      <div key={i} className="flex gap-2 text-xs text-amber-900">
                        <span className="flex-shrink-0 text-amber-400 font-bold">{i + 1}.</span>
                        <span>{issue}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="bg-green-50 text-green-700 text-sm px-4 py-3 rounded-xl text-center">
                  数据检查通过，全部 {preview.valid_count} 行均可正常导入 🎉
                </div>
              )}

              {preview.valid_count === 0 && (
                <div className="bg-red-50 text-red-600 text-sm px-4 py-3 rounded-xl">
                  所有行均有问题，没有可导入的数据。请修改 Excel 文件后重新上传。
                </div>
              )}
            </div>
          )}

          {/* ========== Step 3: 导入完成 ========== */}
          {step === 'done' && done && (
            <div className="space-y-3">
              {done.imported >= 0 ? (
                <>
                  <div className="bg-green-50 rounded-xl px-4 py-4 text-center">
                    <p className="text-3xl font-bold text-green-600">{done.imported}</p>
                    <p className="text-sm text-green-700 mt-1">条记录导入成功</p>
                    {done.skipped_count > 0 && (
                      <p className="text-xs text-gray-500 mt-1">另有 {done.skipped_count} 行因问题已跳过</p>
                    )}
                  </div>
                  {done.issues.length > 0 && (
                    <div>
                      <p className="text-xs text-gray-500 mb-1.5">跳过的行（供参考）：</p>
                      <div className="max-h-40 overflow-y-auto rounded-lg bg-gray-50 px-3 py-2 space-y-1">
                        {done.issues.map((issue, i) => (
                          <p key={i} className="text-xs text-gray-500 font-mono">{issue}</p>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="bg-red-50 rounded-xl px-4 py-4">
                  <p className="text-sm font-medium text-red-700 mb-2">导入失败</p>
                  {done.issues.map((msg, i) => (
                    <p key={i} className="text-sm text-red-600">{msg}</p>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-gray-100">
          <button onClick={step === 'mapping' ? onClose : () => { setStep('mapping'); setPreview(null); setDone(null) }}
            className="px-4 py-2 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
          >
            {step === 'mapping' ? '取消' : '← 返回'}
          </button>

          <div className="flex gap-3">
            {step === 'mapping' && (
              <button
                onClick={handlePreview}
                disabled={!canPreview || parseLoading || previewLoading}
                className="px-5 py-2 text-sm text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-40 rounded-lg transition-colors"
              >
                {previewLoading ? '检查中…' : '下一步：预检数据 →'}
              </button>
            )}
            {step === 'preview' && preview && preview.valid_count > 0 && (
              <button
                onClick={handleImport}
                disabled={importLoading}
                className="px-5 py-2 text-sm text-white bg-green-600 hover:bg-green-700 disabled:opacity-40 rounded-lg transition-colors"
              >
                {importLoading ? '导入中…' : `确认导入 ${preview.valid_count} 条`}
              </button>
            )}
            {step === 'done' && (
              <button onClick={onClose}
                className="px-5 py-2 text-sm text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
              >
                完成
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function StepDot({ active, done, label }: { active: boolean; done: boolean; label: string }) {
  return (
    <span className={`flex items-center gap-1 ${active ? 'text-blue-600 font-medium' : done ? 'text-green-500' : 'text-gray-300'}`}>
      <span className={`w-4 h-4 rounded-full flex items-center justify-center text-[10px] ${
        active ? 'bg-blue-600 text-white' : done ? 'bg-green-500 text-white' : 'bg-gray-200 text-gray-400'
      }`}>{done ? '✓' : active ? '●' : '○'}</span>
      {label}
    </span>
  )
}
