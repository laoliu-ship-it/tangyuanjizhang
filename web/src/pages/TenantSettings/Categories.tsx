import { useState, useEffect, useCallback } from 'react'
import { categoryApi, type Category } from '../../services/api'

const COMMON_ICONS = ['🍽️', '🛒', '🚗', '🏠', '💊', '🎮', '👕', '✈️', '📚', '💡', '🎵', '💰', '💳', '🎁', '🏋️']

export default function Categories() {
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [addType, setAddType] = useState<'income' | 'expense'>('expense')
  const [addName, setAddName] = useState('')
  const [addIcon, setAddIcon] = useState('📌')
  const [adding, setAdding] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [editIcon, setEditIcon] = useState('')
  const [editType, setEditType] = useState<'income' | 'expense'>('expense')
  const [updating, setUpdating] = useState(false)

  const loadCategories = useCallback(async () => {
    setLoading(true)
    try {
      const res = await categoryApi.list()
      setCategories(res.data.data ?? [])
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      if (status === 403) {
        alert('权限不足：您没有权限查看分类')
      } else if (status === 401) {
        alert('登录已过期，请重新登录')
      } else {
        alert(msg || `加载分类失败（HTTP ${status || '未知'}）`)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadCategories()
  }, [loadCategories])

  function handleTypeChange(type: 'income' | 'expense') {
    setAddType(type)
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    if (!addName.trim()) {
      alert('请输入分类名称')
      return
    }
    setAdding(true)
    try {
      await categoryApi.create({ name: addName.trim(), type: addType, icon: addIcon })
      setAddName('')
      setAddIcon('📌')
      loadCategories()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '创建分类失败')
    } finally {
      setAdding(false)
    }
  }

  async function handleDelete(id: number) {
    if (!confirm('确认删除该分类？删除后相关交易将无法显示分类信息。')) return
    setDeletingId(id)
    try {
      await categoryApi.delete(id)
      loadCategories()
    } catch {
      alert('删除失败')
    } finally {
      setDeletingId(null)
    }
  }

  function startEdit(category: Category) {
    setEditingId(category.id)
    setEditName(category.name)
    setEditIcon(category.icon || '📌')
    setEditType(category.type)
  }

  function cancelEdit() {
    setEditingId(null)
    setEditName('')
    setEditIcon('')
    setEditType('expense')
  }

  async function handleUpdate(id: number) {
    if (!editName.trim()) {
      alert('请输入分类名称')
      return
    }
    setUpdating(true)
    try {
      await categoryApi.update(id, { name: editName.trim(), type: editType, icon: editIcon })
      cancelEdit()
      loadCategories()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
      alert(msg || '更新分类失败')
    } finally {
      setUpdating(false)
    }
  }

  const expenseCategories = categories.filter(c => c.type === 'expense')
  const incomeCategories = categories.filter(c => c.type === 'income')

  return (
    <div className="space-y-5">
      {/* 新增分类表单 */}
      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <h3 className="text-sm font-semibold text-gray-700 mb-3">新增分类</h3>
        <form onSubmit={handleAdd} className="space-y-3">
          {/* 类型 */}
          <div className="flex rounded-xl overflow-hidden border border-gray-200">
            <button
              type="button"
              onClick={() => handleTypeChange('expense')}
              className={`flex-1 py-2 text-sm font-medium transition-colors ${
                addType === 'expense' ? 'bg-red-500 text-white' : 'bg-white text-gray-500 hover:bg-gray-50'
              }`}
            >
              支出
            </button>
            <button
              type="button"
              onClick={() => handleTypeChange('income')}
              className={`flex-1 py-2 text-sm font-medium transition-colors ${
                addType === 'income' ? 'bg-green-500 text-white' : 'bg-white text-gray-500 hover:bg-gray-50'
              }`}
            >
              收入
            </button>
          </div>

          {/* 常用图标选择 */}
          <div>
            <p className="text-xs text-gray-500 mb-2">选择图标</p>
            <div className="flex flex-wrap gap-2 mb-2">
              {COMMON_ICONS.map(icon => (
                <button
                  key={icon}
                  type="button"
                  onClick={() => setAddIcon(icon)}
                  className={`w-9 h-9 rounded-lg text-lg flex items-center justify-center border transition-colors ${
                    addIcon === icon ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:bg-gray-50'
                  }`}
                >
                  {icon}
                </button>
              ))}
            </div>
            <input
              type="text"
              value={addIcon}
              onChange={e => setAddIcon(e.target.value)}
              placeholder="或输入 emoji"
              maxLength={4}
              className="w-24 px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex gap-3">
            <input
              type="text"
              value={addName}
              onChange={e => setAddName(e.target.value)}
              placeholder="分类名称"
              className="flex-1 px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={adding}
              className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white text-sm font-medium rounded-xl transition-colors"
            >
              {adding ? '添加中...' : '添加'}
            </button>
          </div>
        </form>
      </div>

      {loading ? (
        <div className="text-center py-10 text-gray-400">加载中...</div>
      ) : (
        <>
          {/* 当前类型的分类列表 */}
          <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
            <div className={`px-5 py-3 border-b ${addType === 'expense' ? 'bg-red-50 border-red-100' : 'bg-green-50 border-green-100'}`}>
              <span className={`text-sm font-medium ${addType === 'expense' ? 'text-red-600' : 'text-green-600'}`}>
                {addType === 'expense' ? '支出' : '收入'}分类（{addType === 'expense' ? expenseCategories.length : incomeCategories.length}）
              </span>
            </div>
            {(addType === 'expense' ? expenseCategories : incomeCategories).length === 0 ? (
              <p className="text-center py-6 text-gray-400 text-sm">暂无{addType === 'expense' ? '支出' : '收入'}分类</p>
            ) : (
              <ul className="divide-y divide-gray-50">
                {(addType === 'expense' ? expenseCategories : incomeCategories).map(c => (
                  <li key={c.id} className="px-5 py-3">
                    {editingId === c.id ? (
                      <form onSubmit={(e) => { e.preventDefault(); handleUpdate(c.id) }} className="space-y-3">
                        {/* 类型切换 */}
                        <div className="flex rounded-xl overflow-hidden border border-gray-200">
                          <button
                            type="button"
                            onClick={() => setEditType('expense')}
                            className={`flex-1 py-2 text-sm font-medium transition-colors ${
                              editType === 'expense' ? 'bg-red-500 text-white' : 'bg-white text-gray-500 hover:bg-gray-50'
                            }`}
                          >
                            支出
                          </button>
                          <button
                            type="button"
                            onClick={() => setEditType('income')}
                            className={`flex-1 py-2 text-sm font-medium transition-colors ${
                              editType === 'income' ? 'bg-green-500 text-white' : 'bg-white text-gray-500 hover:bg-gray-50'
                            }`}
                          >
                            收入
                          </button>
                        </div>

                        {/* 图标选择 */}
                        <div>
                          <p className="text-xs text-gray-500 mb-2">选择图标</p>
                          <div className="flex flex-wrap gap-2 mb-2">
                            {COMMON_ICONS.map(icon => (
                              <button
                                key={icon}
                                type="button"
                                onClick={() => setEditIcon(icon)}
                                className={`w-9 h-9 rounded-lg text-lg flex items-center justify-center border transition-colors ${
                                  editIcon === icon ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:bg-gray-50'
                                }`}
                              >
                                {icon}
                              </button>
                            ))}
                          </div>
                          <input
                            type="text"
                            value={editIcon}
                            onChange={e => setEditIcon(e.target.value)}
                            placeholder="或输入 emoji"
                            maxLength={4}
                            className="w-24 px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </div>

                        {/* 名称输入和操作按钮 */}
                        <div className="flex gap-3">
                          <input
                            type="text"
                            value={editName}
                            onChange={e => setEditName(e.target.value)}
                            placeholder="分类名称"
                            className="flex-1 px-4 py-2.5 border border-gray-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                          <button
                            type="submit"
                            disabled={updating}
                            className="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white text-sm font-medium rounded-xl transition-colors"
                          >
                            {updating ? '保存中...' : '保存'}
                          </button>
                          <button
                            type="button"
                            onClick={cancelEdit}
                            disabled={updating}
                            className="px-5 py-2.5 bg-gray-100 hover:bg-gray-200 disabled:bg-gray-50 text-gray-700 text-sm font-medium rounded-xl transition-colors"
                          >
                            取消
                          </button>
                        </div>
                      </form>
                    ) : (
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <span className="text-xl">{c.icon || '📌'}</span>
                          <span className="text-sm font-medium text-gray-700">{c.name}</span>
                        </div>
                        <div className="flex items-center gap-3">
                          <button
                            onClick={() => startEdit(c)}
                            className="text-sm text-blue-500 hover:text-blue-700 transition-colors"
                          >
                            编辑
                          </button>
                          <button
                            onClick={() => handleDelete(c.id)}
                            disabled={deletingId === c.id}
                            className="text-sm text-red-500 hover:text-red-700 disabled:opacity-50 transition-colors"
                          >
                            {deletingId === c.id ? '删除中' : '删除'}
                          </button>
                        </div>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}
    </div>
  )
}
