import { useState, useEffect, useCallback } from 'react'
import { creditsApi, type CreditLogItem } from '../../services/api'

const ACTION_LABEL: Record<string, string> = {
  upload: '图片上传',
  ocr: 'OCR识别',
  llm: 'AI整理',
  gift: '注册赠送',
  admin_adjust: '管理员调整',
  storage: '存储扣费',
}

function actionLabel(action: string) {
  return ACTION_LABEL[action] ?? action
}

type AmountType = 'all' | 'income' | 'expense'

const PAGE_SIZE = 20

export default function Credits() {
  const [balance, setBalance] = useState<number | null>(null)
  const [unlimited, setUnlimited] = useState(false)
  const [balanceLoading, setBalanceLoading] = useState(true)

  const [logs, setLogs] = useState<CreditLogItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [amountType, setAmountType] = useState<AmountType>('all')
  const [logsLoading, setLogsLoading] = useState(true)

  useEffect(() => {
    creditsApi.getBalance()
      .then(res => {
        const data = res.data.data
        setBalance(data?.balance ?? 0)
        setUnlimited(data?.unlimited ?? false)
      })
      .catch(() => {})
      .finally(() => setBalanceLoading(false))
  }, [])

  const fetchLogs = useCallback((p: number, type: AmountType) => {
    setLogsLoading(true)
    const apiType = type === 'all' ? undefined : type
    creditsApi.getLogs(p, PAGE_SIZE, apiType)
      .then(res => {
        const data = res.data.data
        setLogs(data?.logs ?? [])
        setTotal(data?.total ?? 0)
      })
      .catch(() => {})
      .finally(() => setLogsLoading(false))
  }, [])

  useEffect(() => {
    fetchLogs(page, amountType)
  }, [page, amountType, fetchLogs])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function handleTypeChange(t: AmountType) {
    setAmountType(t)
    setPage(1)
  }

  const tabs: { key: AmountType; label: string }[] = [
    { key: 'all', label: '全部' },
    { key: 'income', label: '收入' },
    { key: 'expense', label: '支出' },
  ]

  return (
    <div className="space-y-5">
      {/* 余额卡片 */}
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-2xl p-5 border border-blue-100">
        <p className="text-sm text-gray-500 mb-1">当前积分余额</p>
        {balanceLoading ? (
          <p className="text-3xl font-bold text-blue-300">…</p>
        ) : unlimited ? (
          <p className="text-3xl font-bold text-blue-600">无限制</p>
        ) : (
          <p className="text-3xl font-bold text-blue-600">{balance?.toLocaleString() ?? '—'}</p>
        )}
        {unlimited && (
          <p className="text-xs text-blue-400 mt-1">已开启白名单，不受积分限制</p>
        )}
      </div>

      {/* 流水列表 */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        {/* 标题栏 + Tab */}
        <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-700">积分明细</h3>
          <div className="flex gap-1 bg-gray-100 rounded-lg p-0.5">
            {tabs.map(t => (
              <button
                key={t.key}
                onClick={() => handleTypeChange(t.key)}
                className={`px-3 py-1 text-xs rounded-md transition-colors ${
                  amountType === t.key
                    ? 'bg-white text-gray-800 font-medium shadow-sm'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* 列表 */}
        {logsLoading ? (
          <p className="text-sm text-gray-400 text-center py-8">加载中…</p>
        ) : logs.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-8">暂无记录</p>
        ) : (
          <ul className="divide-y divide-gray-50">
            {logs.map(log => (
              <li key={log.id} className="flex items-center justify-between px-4 py-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-gray-800">{actionLabel(log.action)}</p>
                  {log.note && (
                    <p className="text-xs text-gray-400 truncate max-w-xs">{log.note}</p>
                  )}
                  <p className="text-xs text-gray-300 mt-0.5">{log.created_at}</p>
                </div>
                <div className="text-right ml-4 shrink-0">
                  <p className={`text-sm font-semibold ${log.amount >= 0 ? 'text-green-600' : 'text-red-500'}`}>
                    {log.amount >= 0 ? '+' : ''}{log.amount.toLocaleString()}
                  </p>
                  <p className="text-xs text-gray-400">
                    余额：{log.balance_after === -1 ? '∞' : log.balance_after.toLocaleString()}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        )}

        {/* 分页 */}
        {!logsLoading && total > PAGE_SIZE && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-gray-50">
            <p className="text-xs text-gray-400">共 {total} 条</p>
            <div className="flex items-center gap-2">
              <button
                disabled={page <= 1}
                onClick={() => setPage(p => p - 1)}
                className="px-2.5 py-1 text-xs rounded-md border border-gray-200 text-gray-600 disabled:opacity-30 hover:bg-gray-50 transition-colors"
              >
                上一页
              </button>
              <span className="text-xs text-gray-500">{page} / {totalPages}</span>
              <button
                disabled={page >= totalPages}
                onClick={() => setPage(p => p + 1)}
                className="px-2.5 py-1 text-xs rounded-md border border-gray-200 text-gray-600 disabled:opacity-30 hover:bg-gray-50 transition-colors"
              >
                下一页
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
