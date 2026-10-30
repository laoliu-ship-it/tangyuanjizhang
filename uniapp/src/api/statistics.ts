import { request } from './request'
import type {
  DailyStatistics,
  MerchantStatsResp,
  MonthlyStatistics,
  RangeStatistics,
  YearlyStatistics,
} from './types'

/** 把查询参数对象拼成 query string（服务端 refresh 接口从 query 读参） */
function toQuery(params: Record<string, string | number | undefined>): string {
  const parts: string[] = []
  Object.keys(params).forEach((key) => {
    const value = params[key]
    if (value === undefined || value === null || value === '') return
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
  })
  return parts.length ? `?${parts.join('&')}` : ''
}

export const statisticsApi = {
  /** GET /statistics/daily?date= */
  daily(date: string): Promise<DailyStatistics> {
    return request<DailyStatistics>({ url: `/statistics/daily${toQuery({ date })}` })
  },

  /** GET /statistics/monthly?year=&month= */
  monthly(year: number, month: number): Promise<MonthlyStatistics> {
    return request<MonthlyStatistics>({ url: `/statistics/monthly${toQuery({ year, month })}` })
  },

  /** GET /statistics/yearly?year= */
  yearly(year: number): Promise<YearlyStatistics> {
    return request<YearlyStatistics>({ url: `/statistics/yearly${toQuery({ year })}` })
  },

  /** GET /statistics/range?start=&end= */
  range(start: string, end: string): Promise<RangeStatistics> {
    return request<RangeStatistics>({ url: `/statistics/range${toQuery({ start, end })}` })
  },

  /** POST /statistics/merchants/refresh?view=&year=&month=|start=&end= */
  refreshMerchants(params:
    | { view: 'monthly'; year: number; month: number }
    | { view: 'yearly'; year: number }
    | { view: 'range'; start: string; end: string }
  ): Promise<MerchantStatsResp> {
    const query = toQuery(params as unknown as Record<string, string | number | undefined>)
    return request<MerchantStatsResp>({
      url: `/statistics/merchants/refresh${query}`,
      method: 'POST',
    })
  },
}
