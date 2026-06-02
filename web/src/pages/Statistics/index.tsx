import { useState, useEffect, useCallback } from 'react'
import dayjs from 'dayjs'
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  PieChart, Pie, Cell, Tooltip as PieTooltip
} from 'recharts'
import { statisticsApi, type MonthlyStatistics, type YearlyStatistics, type RangeStatistics } from '../../services/api'

const COLORS = [
  '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#14b8a6',
]

type ViewMode = 'monthly' | 'yearly' | 'range'

interface CommonData {
  total_income: number
  total_expense: number
  net_amount: number
  categories: { category_id: number; category_name: string; category_icon: string; type: string; total: number }[]
  barData: { label: string; 收入: number; 支出: number }[]
  emptyMessage: string
}

function extractCommonData(
  result: MonthlyStatistics | YearlyStatistics | RangeStatistics | null,
  view: ViewMode
): CommonData {
  if (!result) {
    return { total_income: 0, total_expense: 0, net_amount: 0, categories: [], barData: [], emptyMessage: '暂无数据' }
  }

  const total = 'total' in result ? result.total : result
  const total_income = total?.total_income ?? 0
  const total_expense = total?.total_expense ?? 0
  const net_amount = total?.net_amount ?? (total_income - total_expense)
  const categories = 'categories' in result ? (result.categories ?? []) : []

  let barData: { label: string; 收入: number; 支出: number }[] = []
  let emptyMessage = '暂无数据'

  if (view === 'monthly') {
    const monthly = result as MonthlyStatistics
    barData = (monthly.daily ?? []).map(d => ({
      label: dayjs(d.date).format('DD'),
      收入: d.total_income,
      支出: d.total_expense,
    }))
    emptyMessage = '本月暂无数据'
  } else if (view === 'yearly') {
    const yearly = result as YearlyStatistics
    barData = (yearly.monthly ?? []).map(m => ({
      label: `${m.month}月`,
      收入: m.total_income,
      支出: m.total_expense,
    }))
    emptyMessage = '本年暂无数据'
  } else if (view === 'range') {
    const range = result as RangeStatistics
    barData = (range.daily ?? []).map(d => ({
      label: dayjs(d.date).format('MM-DD'),
      收入: d.total_income,
      支出: d.total_expense,
    }))
    emptyMessage = '所选范围内暂无数据'
  }

  return { total_income, total_expense, net_amount, categories, barData, emptyMessage }
}

const VIEW_OPTIONS: { key: ViewMode; label: string }[] = [
  { key: 'monthly', label: '月视图' },
  { key: 'yearly', label: '年视图' },
  { key: 'range', label: '日期范围' },
]

export default function Statistics() {
  const [view, setView] = useState<ViewMode>('monthly')

  // 月视图状态
  const [year, setYear] = useState(dayjs().year())
  const [month, setMonth] = useState(dayjs().month() + 1)

  // 年视图状态
  const [yearOnly, setYearOnly] = useState(dayjs().year())

  // 范围视图状态
  const [startDate, setStartDate] = useState(dayjs().startOf('month').format('YYYY-MM-DD'))
  const [endDate, setEndDate] = useState(dayjs().format('YYYY-MM-DD'))

  const [data, setData] = useState<MonthlyStatistics | YearlyStatistics | RangeStatistics | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      let res
      if (view === 'monthly') {
        res = await statisticsApi.monthly(year, month)
      } else if (view === 'yearly') {
        res = await statisticsApi.yearly(yearOnly)
      } else {
        res = await statisticsApi.range(startDate, endDate)
      }
      setData(res.data.data)
    } catch {
      alert('加载统计数据失败')
    } finally {
      setLoading(false)
    }
  }, [view, year, month, yearOnly, startDate, endDate])

  useEffect(() => {
    load()
  }, [load])

  // 月视图导航
  function prevMonth() {
    if (month === 1) {
      setYear(y => y - 1)
      setMonth(12)
    } else {
      setMonth(m => m - 1)
    }
  }

  function nextMonth() {
    const now = dayjs()
    if (year > now.year() || (year === now.year() && month >= now.month() + 1)) return
    if (month === 12) {
      setYear(y => y + 1)
      setMonth(1)
    } else {
      setMonth(m => m + 1)
    }
  }

  const isCurrentMonth = year === dayjs().year() && month === dayjs().month() + 1

  // 年视图导航
  function prevYear() {
    setYearOnly(y => y - 1)
  }

  function nextYear() {
    const nowYear = dayjs().year()
    if (yearOnly >= nowYear) return
    setYearOnly(y => y + 1)
  }

  const isCurrentYear = yearOnly === dayjs().year()

  const { total_income, total_expense, net_amount, categories, barData, emptyMessage } =
    extractCommonData(data, view)

  const expensePie = categories
    .filter(c => c.type === 'expense')
    .map(c => ({ name: c.category_name, value: c.total, icon: c.category_icon }))

  const incomePie = categories
    .filter(c => c.type === 'income')
    .map(c => ({ name: c.category_name, value: c.total, icon: c.category_icon }))

  return (
    <div className="p-4 md:p-6 max-w-5xl mx-auto">
      <h1 className="text-xl md:text-2xl font-bold text-gray-800 mb-5">统计分析</h1>

      {/* 视图切换器 */}
      <div className="flex justify-center mb-5">
        <div className="inline-flex bg-gray-100 rounded-xl p-1">
          {VIEW_OPTIONS.map(opt => (
            <button
              key={opt.key}
              onClick={() => setView(opt.key)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                view === opt.key
                  ? 'bg-white text-blue-600 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* 月视图导航 */}
      {view === 'monthly' && (
        <div className="flex items-center justify-center gap-6 mb-6">
          <button
            onClick={prevMonth}
            className="w-9 h-9 rounded-full bg-white border border-gray-200 hover:bg-gray-50 flex items-center justify-center transition-colors"
          >
            <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h2 className="text-lg font-semibold text-gray-800 w-28 text-center">
            {year}年{String(month).padStart(2, '0')}月
          </h2>
          <button
            onClick={nextMonth}
            disabled={isCurrentMonth}
            className="w-9 h-9 rounded-full bg-white border border-gray-200 hover:bg-gray-50 disabled:opacity-30 flex items-center justify-center transition-colors"
          >
            <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      )}

      {/* 年视图导航 */}
      {view === 'yearly' && (
        <div className="flex items-center justify-center gap-6 mb-6">
          <button
            onClick={prevYear}
            className="w-9 h-9 rounded-full bg-white border border-gray-200 hover:bg-gray-50 flex items-center justify-center transition-colors"
          >
            <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h2 className="text-lg font-semibold text-gray-800 w-24 text-center">
            {yearOnly}年
          </h2>
          <button
            onClick={nextYear}
            disabled={isCurrentYear}
            className="w-9 h-9 rounded-full bg-white border border-gray-200 hover:bg-gray-50 disabled:opacity-30 flex items-center justify-center transition-colors"
          >
            <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      )}

      {/* 日期范围选择器 */}
      {view === 'range' && (
        <div className="flex items-center justify-center gap-3 mb-6">
          <input
            type="date"
            value={startDate}
            onChange={e => setStartDate(e.target.value)}
            max={endDate}
            className="px-3 py-2 rounded-lg border border-gray-200 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <span className="text-gray-400">至</span>
          <input
            type="date"
            value={endDate}
            onChange={e => setEndDate(e.target.value)}
            min={startDate}
            max={dayjs().format('YYYY-MM-DD')}
            className="px-3 py-2 rounded-lg border border-gray-200 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      )}

      {loading ? (
        <div className="text-center py-20 text-gray-400">加载中...</div>
      ) : (
        <>
          {/* 汇总卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-6">
            <div className="bg-white rounded-2xl p-4 shadow-sm border border-gray-100 flex items-center justify-between">
              <p className="text-sm text-gray-500">收入</p>
              <p className="text-xl font-bold text-green-500">
                ¥{total_income.toFixed(2)}
              </p>
            </div>
            <div className="bg-white rounded-2xl p-4 shadow-sm border border-gray-100 flex items-center justify-between">
              <p className="text-sm text-gray-500">支出</p>
              <p className="text-xl font-bold text-red-500">
                ¥{total_expense.toFixed(2)}
              </p>
            </div>
            <div className="bg-white rounded-2xl p-4 shadow-sm border border-gray-100 flex items-center justify-between">
              <p className="text-sm text-gray-500">净结余</p>
              <p className={`text-xl font-bold ${net_amount >= 0 ? 'text-blue-600' : 'text-orange-500'}`}>
                {net_amount >= 0 ? '+' : ''}¥{net_amount.toFixed(2)}
              </p>
            </div>
          </div>

          {/* 柱图 */}
          <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100 mb-6">
            <h3 className="text-base font-semibold text-gray-800 mb-4">
              {view === 'monthly' ? '每日收支' : view === 'yearly' ? '每月收支' : '每日收支'}
            </h3>
            {barData.length === 0 ? (
              <div className="h-48 flex items-center justify-center text-gray-400 text-sm">{emptyMessage}</div>
            ) : (
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={barData} margin={{ top: 5, right: 10, left: -10, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="label" tick={{ fontSize: 11 }} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip formatter={(value: number) => [`¥${value.toFixed(2)}`, '']} />
                  <Legend />
                  <Bar dataKey="收入" fill="#22c55e" radius={[3, 3, 0, 0]} />
                  <Bar dataKey="支出" fill="#ef4444" radius={[3, 3, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>

          {/* 分类饼图 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {/* 支出分类 */}
            <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
              <h3 className="text-base font-semibold text-gray-800 mb-4">支出分类</h3>
              {expensePie.length === 0 ? (
                <div className="h-48 flex items-center justify-center text-gray-400 text-sm">暂无支出</div>
              ) : (
                <>
                  <ResponsiveContainer width="100%" height={200}>
                    <PieChart>
                      <Pie
                        data={expensePie}
                        cx="50%"
                        cy="50%"
                        innerRadius={50}
                        outerRadius={80}
                        paddingAngle={2}
                        dataKey="value"
                      >
                        {expensePie.map((_, index) => (
                          <Cell key={index} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Pie>
                      <PieTooltip formatter={(value: number) => [`¥${value.toFixed(2)}`, '']} />
                    </PieChart>
                  </ResponsiveContainer>
                  <div className="mt-3 space-y-2">
                    {expensePie.map((item, index) => (
                      <div key={index} className="flex items-center justify-between text-sm">
                        <div className="flex items-center gap-2">
                          <span
                            className="w-3 h-3 rounded-full flex-shrink-0"
                            style={{ backgroundColor: COLORS[index % COLORS.length] }}
                          />
                          <span className="text-gray-600">{item.icon} {item.name}</span>
                        </div>
                        <span className="font-medium text-gray-800">¥{item.value.toFixed(2)}</span>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>

            {/* 收入分类 */}
            <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
              <h3 className="text-base font-semibold text-gray-800 mb-4">收入分类</h3>
              {incomePie.length === 0 ? (
                <div className="h-48 flex items-center justify-center text-gray-400 text-sm">暂无收入</div>
              ) : (
                <>
                  <ResponsiveContainer width="100%" height={200}>
                    <PieChart>
                      <Pie
                        data={incomePie}
                        cx="50%"
                        cy="50%"
                        innerRadius={50}
                        outerRadius={80}
                        paddingAngle={2}
                        dataKey="value"
                      >
                        {incomePie.map((_, index) => (
                          <Cell key={index} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Pie>
                      <PieTooltip formatter={(value: number) => [`¥${value.toFixed(2)}`, '']} />
                    </PieChart>
                  </ResponsiveContainer>
                  <div className="mt-3 space-y-2">
                    {incomePie.map((item, index) => (
                      <div key={index} className="flex items-center justify-between text-sm">
                        <div className="flex items-center gap-2">
                          <span
                            className="w-3 h-3 rounded-full flex-shrink-0"
                            style={{ backgroundColor: COLORS[index % COLORS.length] }}
                          />
                          <span className="text-gray-600">{item.icon} {item.name}</span>
                        </div>
                        <span className="font-medium text-gray-800">¥{item.value.toFixed(2)}</span>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
