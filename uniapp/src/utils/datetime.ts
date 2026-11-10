/** 日期时间工具：统一 "YYYY-MM-DDTHH:mm" 与后端 "YYYY-MM-DD HH:mm:ss" 的转换 */
import dayjs from 'dayjs'

/** 今天 YYYY-MM-DD */
export function todayStr(): string {
  return dayjs().format('YYYY-MM-DD')
}

/** 'YYYY-MM-DDTHH:mm' -> 'YYYY-MM-DD HH:mm:00' */
export function toApiDateTime(local: string): string {
  if (!local) return dayjs().format('YYYY-MM-DD HH:mm:00')
  const normalized = local.replace('T', ' ')
  const withSeconds = normalized.length === 16 ? `${normalized}:00` : normalized.slice(0, 19)
  return withSeconds
}

/** 'YYYY-MM-DD HH:mm:ss' -> 'YYYY-MM-DDTHH:mm' */
export function fromApiDateTime(api: string): string {
  if (!api) return dayjs().format('YYYY-MM-DDTHH:mm')
  const text = api.replace('T', ' ')
  if (text.length >= 16) return `${text.slice(0, 10)}T${text.slice(11, 16)}`
  if (text.length === 10) return `${text}T00:00`
  return text
}

/** MM-DD HH:mm（带容错） */
export function humanDateTime(s?: string | null): string {
  if (!s) return ''
  const text = String(s).replace('T', ' ')
  const m = text.match(/(\d{4})-(\d{2})-(\d{2})[ ](\d{2}):(\d{2})/)
  if (m) return `${m[2]}-${m[3]} ${m[4]}:${m[5]}`
  const d = text.match(/(\d{4})-(\d{2})-(\d{2})/)
  if (d) return `${d[2]}-${d[3]}`
  return text.slice(0, 16)
}

/** YYYY-MM-DD（带容错） */
export function humanDate(s?: string | null): string {
  if (!s) return ''
  const text = String(s).replace('T', ' ')
  const m = text.match(/(\d{4})-(\d{2})-(\d{2})/)
  if (m) return `${m[1]}-${m[2]}-${m[3]}`
  const parsed = dayjs(text)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD') : text.slice(0, 10)
}

/**
 * 从 OCR 日期字符串中提取 HH:mm。
 * 兼容 "2024-05-20T14:30"、"2024-05-20 14:30:00"、"2024年5月20日 14:30" 等；
 * 无时间部分时使用 fallback（默认当前时间）。
 */
export function extractTimeFromOcrDate(ocrDate?: string, fallback?: string): string {
  if (!ocrDate) return fallback ?? dayjs().format('HH:mm')
  const m = String(ocrDate).match(/[\sT](\d{1,2}):(\d{2})/)
  if (m) return `${m[1].padStart(2, '0')}:${m[2]}`
  return fallback ?? dayjs().format('HH:mm')
}

/** 从 OCR/LLM 日期中提取 YYYY-MM-DD；无法提取时返回 today */
export function extractDatePart(value?: string): string {
  if (!value) return todayStr()
  const m = String(value).match(/(\d{4})-(\d{1,2})-(\d{1,2})/)
  if (m) return `${m[1]}-${m[2].padStart(2, '0')}-${m[3].padStart(2, '0')}`
  const cn = String(value).match(/(\d{4})年(\d{1,2})月(\d{1,2})日/)
  if (cn) return `${cn[1]}-${cn[2].padStart(2, '0')}-${cn[3].padStart(2, '0')}`
  return todayStr()
}

/** 当前本地时间，格式 YYYY-MM-DDTHH:mm */
export function nowLocal(): string {
  return dayjs().format('YYYY-MM-DDTHH:mm')
}

/** 当前年 / 月 */
export function currentYear(): number {
  return dayjs().year()
}

export function currentMonth(): number {
  return dayjs().month() + 1
}

/** 月份天数 */
export function daysInMonth(year: number, month: number): number {
  return dayjs(`${year}-${String(month).padStart(2, '0')}-01`).daysInMonth()
}
