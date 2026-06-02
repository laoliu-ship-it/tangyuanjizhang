import { useState, useEffect, useCallback, useRef } from 'react'
import dayjs from 'dayjs'
import { transactionApi, categoryApi, importApi, tenantApi, type Transaction, type Category } from '../../services/api'
import { useResponsive } from '../../hooks/useResponsive'
import { useTenantStore } from '../../store/tenant'
import { useAuthStore } from '../../store/auth'
import TransactionForm from '../../components/TransactionForm'
import CategoryIcon from '../../components/CategoryIcon'
import ImportMappingDialog from '../../components/ImportMappingDialog'

const PAGE_SIZE = 10

export default function Transactions() {
  const { isMobile } = useResponsive()
  const currentTenantId = useTenantStore(s => s.currentTenantId)
  const userId = useAuthStore(s => s.userId)

  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)

  const [filterType, setFilterType] = useState('')
  const [filterCategoryId, setFilterCategoryId] = useState<number | ''>('')
  const [filterStartDate, setFilterStartDate] = useState('')
  const [filterEndDate, setFilterEndDate] = useState('')
  const [sortBy, setSortBy] = useState<'transaction_date' | 'created_at'>('transaction_date')

  const [categories, setCategories] = useState<Category[]>([])
  const [currentRole, setCurrentRole] = useState<string>('member')
  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Transaction | undefined>()
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null)
  const [detailTarget, setDetailTarget] = useState<Transaction | null>(null)

  const fileImportRef = useRef<HTMLInputElement>(null)
  const [importDialogOpen, setImportDialogOpen] = useState(false)
  const [importFile, setImportFile] = useState<File | null>(null)

  const canEdit = currentRole === 'owner' || currentRole === 'admin'

  useEffect(() => {
    categoryApi.list().then(res => setCategories(res.data.data)).catch(() => {})
  }, [])

  useEffect(() => {
    if (!currentTenantId || !userId) return
    tenantApi.getMembers(currentTenantId)
      .then(res => {
        const me = res.data.data.find(m => m.user_id === userId)
        setCurrentRole(me?.role ?? 'member')
      })
      .catch(() => setCurrentRole('member'))
  }, [currentTenantId, userId])

  const loadTransactions = useCallback(async (p = page) => {
    setLoading(true)
    try {
      const res = await transactionApi.list({
        page: p,
        page_size: PAGE_SIZE,
        type: filterType || undefined,
        category_id: filterCategoryId || undefined,
        start_date: filterStartDate || undefined,
        end_date: filterEndDate || undefined,
        sort_by: sortBy,
      })
      setTransactions(res.data.data.items)
      setTotal(res.data.data.total)
    } catch {
      alert('加载交易列表失败')
    } finally {
      setLoading(false)
    }
  }, [page, filterType, filterCategoryId, filterStartDate, filterEndDate, sortBy])

  useEffect(() => {
    loadTransactions(page)
  }, [loadTransactions, page])

  function handleSearch() {
    setPage(1)
    loadTransactions(1)
  }

  function handleReset() {
    setFilterType('')
    setFilterCategoryId('')
    setFilterStartDate('')
    setFilterEndDate('')
    setPage(1)
  }

  async function handleDelete(id: number) {
    setDeleteConfirmId(id)
  }

  async function confirmDelete() {
    if (deleteConfirmId === null) return
    const id = deleteConfirmId
    setDeleteConfirmId(null)
    setDeletingId(id)
    try {
      await transactionApi.delete(id)
      loadTransactions(page)
    } catch {
      alert('删除失败')
    } finally {
      setDeletingId(null)
    }
  }

  function handleEdit(t: Transaction) {
    setEditTarget(t)
    setFormOpen(true)
  }

  function handleAddNew() {
    setEditTarget(undefined)
    setFormOpen(true)
  }

  async function handleDownloadTemplate() {
    try {
      const res = await importApi.downloadTemplate()
      const url = URL.createObjectURL(new Blob([res.data]))
      const a = document.createElement('a')
      a.href = url
      a.download = '导入模板.xlsx'
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      alert('下载失败')
    }
  }

  function handleImport(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ''
    setImportFile(file)
    setImportDialogOpen(true)
  }

  const totalPages = Math.ceil(total / PAGE_SIZE)

  const categoryMap = new Map(categories.map(c => [c.id, c]))

  return (
    <div className="p-4 md:p-6 max-w-5xl mx-auto">
      {/* 隐藏的文件导入 input */}
      <input
        type="file"
        accept=".xlsx,.xls"
        ref={fileImportRef}
        onChange={handleImport}
        className="hidden"
      />

      <div className="flex items-center justify-between mb-5">
        <h1 className="text-xl md:text-2xl font-bold text-gray-800">交易记录</h1>
        {!isMobile && canEdit && (
          <div className="flex items-center gap-2">
            <button
              onClick={handleDownloadTemplate}
              className="flex items-center gap-1.5 px-3 py-2 bg-gray-100 hover:bg-gray-200 text-gray-600 text-sm font-medium rounded-xl transition-colors"
            >
              📥 下载模板
            </button>
            <button
              onClick={() => fileImportRef.current?.click()}
              className="flex items-center gap-1.5 px-3 py-2 bg-gray-100 hover:bg-gray-200 text-gray-600 text-sm font-medium rounded-xl transition-colors"
            >
              📤 导入
            </button>
            <button
              onClick={handleAddNew}
              className="flex items-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-xl transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              新增记账
            </button>
          </div>
        )}
      </div>

      {/* 筛选栏 */}
      <div className="bg-white rounded-2xl p-4 mb-5 shadow-sm border border-gray-100">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <select
            value={filterType}
            onChange={e => setFilterType(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">全部类型</option>
            <option value="income">收入</option>
            <option value="expense">支出</option>
          </select>

          <select
            value={filterCategoryId}
            onChange={e => setFilterCategoryId(e.target.value ? Number(e.target.value) : '')}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">全部分类</option>
            {categories.map(c => (
              <option key={c.id} value={c.id}>{c.icon} {c.name}</option>
            ))}
          </select>

          <input
            type="date"
            value={filterStartDate}
            onChange={e => setFilterStartDate(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="开始日期"
          />

          <input
            type="date"
            value={filterEndDate}
            onChange={e => setFilterEndDate(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="结束日期"
          />
        </div>
        <div className="flex items-center justify-between mt-3">
          <div className="flex gap-3">
            <button
              onClick={handleSearch}
              className="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
            >
              查询
            </button>
            <button
              onClick={handleReset}
              className="px-5 py-2 bg-gray-100 hover:bg-gray-200 text-gray-600 text-sm font-medium rounded-lg transition-colors"
            >
              重置
            </button>
          </div>
          <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
            <button
              onClick={() => { setSortBy('transaction_date'); setPage(1) }}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                sortBy === 'transaction_date'
                  ? 'bg-white text-blue-600 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              按交易时间
            </button>
            <button
              onClick={() => { setSortBy('created_at'); setPage(1) }}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                sortBy === 'created_at'
                  ? 'bg-white text-blue-600 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              按创建时间
            </button>
          </div>
        </div>
      </div>

      {/* 加载状态 */}
      {loading && (
        <div className="text-center py-10 text-gray-400">加载中...</div>
      )}

      {/* 无数据 */}
      {!loading && transactions.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <svg className="w-12 h-12 mx-auto mb-3 text-gray-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          暂无交易记录
        </div>
      )}

      {/* PC 表格 */}
      {!loading && transactions.length > 0 && !isMobile && (
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-100">
                <th className="text-left px-5 py-3 font-medium text-gray-500">创建时间</th>
                <th className="text-left px-5 py-3 font-medium text-gray-500">类型</th>
                <th className="text-left px-5 py-3 font-medium text-gray-500">分类</th>
                <th className="text-right px-5 py-3 font-medium text-gray-500">金额</th>
                <th className="text-left px-5 py-3 font-medium text-gray-500">备注</th>
                <th className="text-left px-5 py-3 font-medium text-gray-500">交易日期</th>
                <th className="text-center px-5 py-3 font-medium text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {transactions.map(t => {
                const cat = categoryMap.get(t.category_id)
                return (
                  <tr key={t.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-5 py-3.5 text-gray-500 text-xs">
                      {t.created_at ? dayjs(t.created_at).format('YYYY-MM-DD HH:mm') : '-'}
                    </td>
                    <td className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        t.type === 'income'
                          ? 'bg-green-50 text-green-700'
                          : 'bg-red-50 text-red-700'
                      }`}>
                        {t.type === 'income' ? '收入' : '支出'}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-2">
                        {cat && <span>{cat.icon}</span>}
                        <span className="text-gray-700">{t.category_name ?? cat?.name ?? '-'}</span>
                      </div>
                    </td>
                    <td className={`px-5 py-3.5 text-right font-semibold ${
                      t.type === 'income' ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {t.type === 'income' ? '+' : '-'}¥{t.amount.toFixed(2)}
                    </td>
                    <td className="px-5 py-3.5 text-gray-500 max-w-xs truncate">
                      {t.note || '-'}
                    </td>
                    <td className="px-5 py-3.5 text-gray-600">
                      {dayjs(t.transaction_date).format('YYYY-MM-DD HH:mm')}
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center justify-center gap-3">
                        <button
                          onClick={() => setDetailTarget(t)}
                          className="text-gray-500 hover:text-gray-700 text-sm transition-colors"
                        >
                          详情
                        </button>
                        {canEdit && (
                          <>
                            <button
                              onClick={() => handleEdit(t)}
                              className="text-blue-500 hover:text-blue-700 text-sm transition-colors"
                            >
                              编辑
                            </button>
                            <button
                              onClick={() => handleDelete(t.id)}
                              disabled={deletingId === t.id}
                              className="text-red-500 hover:text-red-700 text-sm transition-colors disabled:opacity-50"
                            >
                              {deletingId === t.id ? '删除中' : '删除'}
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* 移动端卡片列表 */}
      {!loading && transactions.length > 0 && isMobile && (
        <div className="space-y-3">
          {transactions.map(t => {
            const cat = categoryMap.get(t.category_id)
            return (
              <div
                key={t.id}
                className="bg-white rounded-2xl p-4 shadow-sm border border-gray-100 flex items-center gap-3"
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
                    <span className="text-xs text-gray-400">
                      {dayjs(t.transaction_date).format('MM-DD HH:mm')}
                      {t.note && ` · ${t.note}`}
                    </span>
                    <span className="text-xs text-gray-300">
                      {t.created_at ? dayjs(t.created_at).format('MM-DD HH:mm') : ''}
                    </span>
                    <div className="flex gap-3">
                      <button
                        onClick={() => setDetailTarget(t)}
                        className="text-xs text-gray-500 hover:text-gray-700"
                      >
                        详情
                      </button>
                      {canEdit && (
                        <>
                          <button
                            onClick={() => handleEdit(t)}
                            className="text-xs text-blue-500 hover:text-blue-700"
                          >
                            编辑
                          </button>
                          <button
                            onClick={() => handleDelete(t.id)}
                            disabled={deletingId === t.id}
                            className="text-xs text-red-500 hover:text-red-700 disabled:opacity-50"
                          >
                            删除
                          </button>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* 分页 */}
      {!loading && totalPages > 1 && (
        <div className="flex items-center justify-center gap-1.5 mt-6 flex-wrap">
          {/* 上一页 */}
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="px-3 py-2 text-sm bg-white border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-40 transition-colors"
          >
            ‹
          </button>

          {/* 页码 */}
          {(() => {
            const pages: (number | '...')[] = []
            if (totalPages <= 7) {
              for (let i = 1; i <= totalPages; i++) pages.push(i)
            } else {
              pages.push(1)
              if (page > 3) pages.push('...')
              for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) pages.push(i)
              if (page < totalPages - 2) pages.push('...')
              pages.push(totalPages)
            }
            return pages.map((p, idx) =>
              p === '...' ? (
                <span key={`e${idx}`} className="px-2 py-2 text-sm text-gray-400 select-none">…</span>
              ) : (
                <button
                  key={p}
                  onClick={() => setPage(p)}
                  className={`min-w-[36px] px-3 py-2 text-sm rounded-lg border transition-colors ${
                    p === page
                      ? 'bg-blue-600 text-white border-blue-600 font-medium'
                      : 'bg-white border-gray-200 text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  {p}
                </button>
              )
            )
          })()}

          {/* 下一页 */}
          <button
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            className="px-3 py-2 text-sm bg-white border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-40 transition-colors"
          >
            ›
          </button>

          <span className="text-xs text-gray-400 ml-2">共 {total} 条</span>
        </div>
      )}

      {/* 移动端悬浮 + 按钮（仅管理员可见） */}
      {isMobile && canEdit && (
        <button
          onClick={handleAddNew}
          className="fixed bottom-20 right-5 w-14 h-14 bg-blue-600 hover:bg-blue-700 text-white rounded-full shadow-lg flex items-center justify-center transition-colors z-10"
        >
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        </button>
      )}

      {/* 删除确认弹窗 */}
      {deleteConfirmId !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={() => setDeleteConfirmId(null)}>
          <div className="bg-white rounded-2xl p-6 shadow-xl w-80 mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-base font-semibold text-gray-800 mb-2">确认删除</h3>
            <p className="text-sm text-gray-500 mb-5">此操作不可撤销，确定要删除这条记录吗？</p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setDeleteConfirmId(null)}
                className="px-4 py-2 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
              >
                取消
              </button>
              <button
                onClick={confirmDelete}
                className="px-4 py-2 text-sm text-white bg-red-500 hover:bg-red-600 rounded-lg transition-colors"
              >
                删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 详情弹窗（复用 TransactionForm 的 detail 模式） */}
      <TransactionForm
        open={detailTarget !== null}
        onClose={() => setDetailTarget(null)}
        initialData={detailTarget ?? undefined}
        mode="detail"
      />

      {/* 编辑弹窗 */}
      <TransactionForm
        open={formOpen}
        onClose={() => { setFormOpen(false); setEditTarget(undefined) }}
        onSuccess={() => loadTransactions(page)}
        initialData={editTarget}
        mode="edit"
      />

      {/* 导入列映射弹窗 */}
      <ImportMappingDialog
        open={importDialogOpen}
        file={importFile}
        onClose={() => setImportDialogOpen(false)}
        onImported={() => { loadTransactions(1); setPage(1) }}
      />
    </div>
  )
}
