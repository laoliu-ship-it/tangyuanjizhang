import { useState, useEffect, useCallback } from 'react'
import dayjs from 'dayjs'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer
} from 'recharts'
import { statisticsApi, type DailyStatistics, type MonthlyStatistics } from '../../services/api'
import TransactionForm from '../../components/TransactionForm'

export default function Dashboard() {
  const today = dayjs().format('YYYY-MM-DD')
  const year = dayjs().year()
  const month = dayjs().month() + 1

  const [daily, setDaily] = useState<DailyStatistics | null>(null)
  const [monthly, setMonthly] = useState<MonthlyStatistics | null>(null)
  const [loadingDaily, setLoadingDaily] = useState(true)
  const [loadingMonthly, setLoadingMonthly] = useState(true)
  const [formOpen, setFormOpen] = useState(false)

  const loadDaily = useCallback(async () => {
    setLoadingDaily(true)
    try {
      const res = await statisticsApi.daily(today)
      setDaily(res.data.data)
    } catch {
      // 忽略统计加载错误
    } finally {
      setLoadingDaily(false)
    }
  }, [today])

  const loadMonthly = useCallback(async () => {
    setLoadingMonthly(true)
    try {
      const res = await statisticsApi.monthly(year, month)
      setMonthly(res.data.data)
    } catch {
      // 忽略统计加载错误
    } finally {
      setLoadingMonthly(false)
    }
  }, [year, month])

  useEffect(() => {
    loadDaily()
    loadMonthly()
  }, [loadDaily, loadMonthly])

  function handleFormSuccess() {
    loadDaily()
    loadMonthly()
  }

  const chartData = monthly?.daily?.map(d => ({
    date: dayjs(d.date).format('DD'),
    收入: d.total_income,
    支出: d.total_expense,
  })) ?? []

  return (
    <div className="p-4 md:p-6 max-w-5xl mx-auto">
      {/* 顶部标题栏 */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl md:text-2xl font-bold text-gray-800">今日概览</h1>
          <p className="text-sm text-gray-400 mt-0.5">{dayjs().format('YYYY年MM月DD日')}</p>
        </div>
        <button
          onClick={() => setFormOpen(true)}
          className="flex items-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-xl transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          记一笔
        </button>
      </div>

      {/* 今日收支卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-6">
        {/* 收入 */}
        <div className="bg-white rounded-xl p-3.5 shadow-sm border border-gray-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-full bg-green-50 flex items-center justify-center">
              <svg className="w-3.5 h-3.5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
              </svg>
            </div>
            <span className="text-sm text-gray-500">今日收入</span>
          </div>
          {loadingDaily ? (
            <div className="h-6 w-20 bg-gray-100 rounded animate-pulse" />
          ) : (
            <span className="text-lg font-bold text-green-600">
              ¥{(daily?.total_income ?? 0).toFixed(2)}
            </span>
          )}
        </div>

        {/* 支出 */}
        <div className="bg-white rounded-xl p-3.5 shadow-sm border border-gray-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-full bg-red-50 flex items-center justify-center">
              <svg className="w-3.5 h-3.5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
              </svg>
            </div>
            <span className="text-sm text-gray-500">今日支出</span>
          </div>
          {loadingDaily ? (
            <div className="h-6 w-20 bg-gray-100 rounded animate-pulse" />
          ) : (
            <span className="text-lg font-bold text-red-600">
              ¥{(daily?.total_expense ?? 0).toFixed(2)}
            </span>
          )}
        </div>

        {/* 净结余 */}
        <div className="bg-white rounded-xl p-3.5 shadow-sm border border-gray-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-full bg-blue-50 flex items-center justify-center">
              <svg className="w-3.5 h-3.5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
              </svg>
            </div>
            <span className="text-sm text-gray-500">净结余</span>
          </div>
          {loadingDaily ? (
            <div className="h-6 w-20 bg-gray-100 rounded animate-pulse" />
          ) : (
            <span className={`text-lg font-bold ${(daily?.total_income ?? 0) - (daily?.total_expense ?? 0) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {((daily?.total_income ?? 0) - (daily?.total_expense ?? 0) >= 0) ? '+' : '-'}¥{Math.abs((daily?.total_income ?? 0) - (daily?.total_expense ?? 0)).toFixed(2)}
            </span>
          )}
        </div>
      </div>

      {/* 月趋势折线图 */}
      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-gray-800">
            {dayjs().format('MM')}月收支趋势
          </h2>
          {monthly && (
            <div className="flex gap-4 text-xs text-gray-500">
              <span>
                收入 <span className="font-semibold text-green-500">¥{(monthly?.total?.total_income ?? 0).toFixed(2)}</span>
              </span>
              <span>
                支出 <span className="font-semibold text-red-500">¥{(monthly?.total?.total_expense ?? 0).toFixed(2)}</span>
              </span>
            </div>
          )}
        </div>

        {loadingMonthly ? (
          <div className="h-56 bg-gray-50 rounded-xl animate-pulse" />
        ) : chartData.length === 0 ? (
          <div className="h-56 flex items-center justify-center text-gray-400 text-sm">
            本月暂无数据
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={chartData} margin={{ top: 5, right: 10, left: -10, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip
                formatter={(value: number) => [`¥${value.toFixed(2)}`, '']}
                labelFormatter={label => `${label}日`}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="收入"
                stroke="#22c55e"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
              />
              <Line
                type="monotone"
                dataKey="支出"
                stroke="#ef4444"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>

      <TransactionForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSuccess={handleFormSuccess}
      />
    </div>
  )
}
