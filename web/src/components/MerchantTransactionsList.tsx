import { useState, useEffect, useCallback } from 'react'
import dayjs from 'dayjs'
import { transactionApi, categoryApi, type Transaction, type Category } from '../services/api'
import CategoryIcon from './CategoryIcon'
import TransactionForm from './TransactionForm'

const PAGE_SIZE = 10

interface MerchantTransactionsListProps {
  merchantId: number
  startDate?: string
  endDate?: string
}

export default function MerchantTransactionsList({ merchantId, startDate, endDate }: MerchantTransactionsListProps) {
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [categories, setCategories] = useState<Category[]>([])
  const [detailTarget, setDetailTarget] = useState<Transaction | null>(null)

  useEffect(() => {
    categoryApi.list().then(res => setCategories(res.data.data)).catch(() => {})
  }, [])

  const load = useCallback(async (p: number) => {
    setLoading(true)
    try {
      const res = await transactionApi.list({
        page: p,
        page_size: PAGE_SIZE,
        merchant_id: merchantId,
        start_date: startDate || undefined,
        end_date: endDate || undefined,
        sort_by: 'transaction_date',
      })
      setTransactions(res.data.data.items)
      setTotal(res.data.data.total)
    } catch {
      alert('加载商户交易记录失败')
    } finally {
      setLoading(false)
    }
  }, [merchantId, startDate, endDate])

  useEffect(() => {
    setPage(1)
    load(1)
  }, [load])

  const categoryMap = new Map(categories.map(c => [c.id, c]))
  const totalPages = Math.ceil(total / PAGE_SIZE)

  function goPage(p: number) {
    setPage(p)
    load(p)
  }

  return (
    <div>
      {loading && (
        <div className="text-center py-10 text-gray-400 text-sm">加载中...</div>
      )}

      {!loading && transactions.length === 0 && (
        <div className="text-center py-10 text-gray-400 text-sm">暂无交易记录</div>
      )}

      {!loading && transactions.length > 0 && (
        <div className="space-y-3">
          {transactions.map(t => {
            const cat = categoryMap.get(t.category_id)
            return (
              <button
                key={t.id}
                onClick={() => setDetailTarget(t)}
                className="w-full bg-gray-50 hover:bg-gray-100 rounded-xl p-3 flex items-center gap-3 text-left transition-colors"
              >
                <CategoryIcon icon={cat?.icon ?? '📌'} name={cat?.name} size="md" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-800 truncate">
                      {t.category_name ?? cat?.name ?? '-'}
                    </span>
                    <span className={`text-sm font-bold ml-2 flex-shrink-0 ${
                      t.type === 'income' ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {t.type === 'income' ? '+' : '-'}¥{t.amount.toFixed(2)}
                    </span>
                  </div>
                  <div className="flex items-center justify-between mt-1">
                    <span className="text-xs text-gray-400 truncate">
                      {dayjs(t.transaction_date.slice(0, 16).replace(' ', 'T')).format('YYYY-MM-DD HH:mm')}
                      {t.note && ` · ${t.note}`}
                    </span>
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      )}

      {!loading && totalPages > 1 && (
        <div className="flex items-center justify-center gap-1.5 mt-4">
          <button
            onClick={() => goPage(Math.max(1, page - 1))}
            disabled={page === 1}
            className="px-3 py-1.5 text-sm bg-white border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-40 transition-colors"
          >
            ‹
          </button>
          <span className="text-xs text-gray-500 px-2">{page} / {totalPages}</span>
          <button
            onClick={() => goPage(Math.min(totalPages, page + 1))}
            disabled={page === totalPages}
            className="px-3 py-1.5 text-sm bg-white border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-40 transition-colors"
          >
            ›
          </button>
        </div>
      )}

      <TransactionForm
        open={detailTarget !== null}
        onClose={() => setDetailTarget(null)}
        initialData={detailTarget ?? undefined}
        mode="detail"
      />
    </div>
  )
}
