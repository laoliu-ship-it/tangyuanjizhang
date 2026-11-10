/** 金额 / 百分比格式化（全端统一） */

/** 拆分成整数与小数部分，非法值按 0 处理 */
function splitAmount(n?: number | null): { sign: string; int: string; dec: string } {
  const value = typeof n === 'number' && Number.isFinite(n) ? n : 0
  const sign = value < 0 ? '-' : ''
  const abs = Math.abs(value)
  const fixed = abs.toFixed(2)
  const [int, dec] = fixed.split('.')
  return { sign, int, dec }
}

function groupInt(int: string): string {
  return int.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

/** 1,234.56（无 ¥，负数带 -） */
export function money(n?: number | null): string {
  const { sign, int, dec } = splitAmount(n)
  return `${sign}${groupInt(int)}.${dec}`
}

/** +¥1,234.56 / -¥1,234.56（type 决定正负号；金额取绝对值） */
export function moneySigned(type: 'income' | 'expense', n?: number | null): string {
  const value = Math.abs(typeof n === 'number' && Number.isFinite(n) ? n : 0)
  return `${type === 'income' ? '+' : '-'}¥${money(value)}`
}

/** 1.2万 / 3,456（大额紧凑展示） */
export function compactMoney(n?: number | null): string {
  const value = typeof n === 'number' && Number.isFinite(n) ? n : 0
  const sign = value < 0 ? '-' : ''
  const abs = Math.abs(value)
  if (abs >= 1e8) {
    return `${sign}${trimZero((abs / 1e8).toFixed(2))}亿`
  }
  if (abs >= 1e4) {
    return `${sign}${trimZero((abs / 1e4).toFixed(1))}万`
  }
  return `${sign}${groupInt(String(Math.round(abs)))}`
}

/** 12.3%；total <= 0 时返回 0% */
export function percent(v: number, total: number): string {
  if (!Number.isFinite(v) || !Number.isFinite(total) || total <= 0) return '0%'
  return `${((v / total) * 100).toFixed(1)}%`
}

function trimZero(text: string): string {
  return text.replace(/\.0+$/, '').replace(/(\.\d*?)0+$/, '$1')
}
