import { useState, useEffect, useCallback } from 'react'
import { platformAdminApi, type CronJobItem } from '../services/api'

export default function CronJobs() {
  const [jobs, setJobs] = useState<CronJobItem[]>([])
  const [loading, setLoading] = useState(true)
  const [triggering, setTriggering] = useState<string | null>(null)
  const [message, setMessage] = useState('')

  const fetchJobs = useCallback(() => {
    setLoading(true)
    platformAdminApi.listCronJobs()
      .then(res => setJobs(res.data.data?.jobs ?? []))
      .catch(err => setMessage(err instanceof Error ? err.message : '加载失败'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  async function handleTrigger(name: string) {
    setTriggering(name)
    setMessage('')
    try {
      await platformAdminApi.triggerCronJob(name)
      setMessage(`任务 ${name} 已触发`)
      // 稍等片刻再刷新，让执行状态落库
      setTimeout(fetchJobs, 1000)
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '触发失败')
    } finally {
      setTriggering(null)
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">定时任务</h2>
          <p className="text-sm text-gray-500 mt-1">所有后台周期任务的统一管理与手动触发</p>
        </div>
        <button
          onClick={fetchJobs}
          className="px-4 py-2 text-sm rounded-xl border border-gray-200 text-gray-600 hover:bg-gray-50 transition-colors"
        >
          刷新
        </button>
      </div>

      {message && (
        <div className="mb-4 px-4 py-2.5 rounded-xl bg-blue-50 border border-blue-100 text-sm text-blue-700">
          {message}
        </div>
      )}

      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        {loading ? (
          <p className="text-sm text-gray-400 text-center py-10">加载中…</p>
        ) : jobs.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-10">暂无定时任务</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50 text-left text-xs text-gray-500">
                  <th className="px-4 py-3 font-medium">任务</th>
                  <th className="px-4 py-3 font-medium">调度</th>
                  <th className="px-4 py-3 font-medium">状态</th>
                  <th className="px-4 py-3 font-medium">下次执行</th>
                  <th className="px-4 py-3 font-medium">上次执行</th>
                  <th className="px-4 py-3 font-medium">耗时</th>
                  <th className="px-4 py-3 font-medium">执行/失败</th>
                  <th className="px-4 py-3 font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {jobs.map(job => (
                  <tr key={job.name} className="hover:bg-gray-50/50">
                    <td className="px-4 py-3">
                      <p className="font-medium text-gray-800">{job.name}</p>
                      <p className="text-xs text-gray-400 mt-0.5">{job.description}</p>
                    </td>
                    <td className="px-4 py-3 text-gray-600 whitespace-nowrap">{job.schedule}</td>
                    <td className="px-4 py-3">
                      {job.running ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-blue-50 text-blue-600">执行中</span>
                      ) : job.last_error ? (
                        <span
                          className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-red-50 text-red-600"
                          title={job.last_error}
                        >
                          上次失败
                        </span>
                      ) : (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-green-50 text-green-600">正常</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-500 whitespace-nowrap">{job.next_run_at || '—'}</td>
                    <td className="px-4 py-3 text-gray-500 whitespace-nowrap">{job.last_run_at || '从未执行'}</td>
                    <td className="px-4 py-3 text-gray-500 whitespace-nowrap">{job.last_duration || '—'}</td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className="text-gray-600">{job.run_count}</span>
                      <span className="text-gray-300 mx-1">/</span>
                      <span className={job.fail_count > 0 ? 'text-red-500' : 'text-gray-400'}>{job.fail_count}</span>
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleTrigger(job.name)}
                        disabled={triggering === job.name || job.running}
                        className="px-3 py-1.5 text-xs rounded-lg bg-slate-800 text-white hover:bg-slate-700 disabled:opacity-40 transition-colors whitespace-nowrap"
                      >
                        {triggering === job.name ? '触发中…' : '立即执行'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {jobs.some(j => j.last_error) && (
        <div className="mt-4 space-y-2">
          {jobs.filter(j => j.last_error).map(j => (
            <div key={j.name} className="px-4 py-2.5 rounded-xl bg-red-50 border border-red-100 text-xs text-red-600">
              <span className="font-medium">{j.name}</span> 上次执行失败：{j.last_error}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
