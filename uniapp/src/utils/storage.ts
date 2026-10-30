/** uni storage 的安全封装：统一 JSON 序列化/反序列化，读取失败不抛异常 */

export function getJSON<T>(key: string, fallback: T): T {
  try {
    const raw = uni.getStorageSync(key)
    if (raw === '' || raw === null || raw === undefined) return fallback
    if (typeof raw === 'object') return raw as T
    return JSON.parse(String(raw)) as T
  } catch (e) {
    console.warn('[storage] 读取失败', key, e)
    return fallback
  }
}

export function setJSON(key: string, value: unknown): void {
  try {
    uni.setStorageSync(key, JSON.stringify(value))
  } catch (e) {
    console.warn('[storage] 写入失败', key, e)
  }
}

export function remove(key: string): void {
  try {
    uni.removeStorageSync(key)
  } catch (e) {
    console.warn('[storage] 删除失败', key, e)
  }
}
