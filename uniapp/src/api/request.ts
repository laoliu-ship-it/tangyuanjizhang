/**
 * 唯一网络出口：uni.request 封装 + 上传封装。
 * 规范见 ARCHITECTURE.md §6：
 *  - 注入 Authorization / X-Tenant-ID
 *  - 业务错误（HTTP 200 且 code !== 0）→ throw new Error(message)
 *  - HTTP 401 → 清 auth/tenant → reLaunch 登录页 → throw
 *  - 网络错误/超时 → throw new Error('网络异常，请稍后重试')
 */
import { API_BASE, REQUEST_TIMEOUT } from '@/config'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { routes } from '@/utils/nav'
import type { ApiResponse } from './types'

export interface RequestOptions {
  url: string
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: Record<string, any> | any[] | null
  timeout?: number
  header?: Record<string, string>
}

export interface UploadOptions {
  url: string
  filePath?: string
  /** 仅 H5 可传（File / Blob），非 H5 忽略 */
  file?: File
  /** form 字段名，默认 'file' */
  name?: string
  formData?: Record<string, any>
  timeout?: number
}

const MSG_FAIL = '请求失败'
const MSG_NETWORK = '网络异常，请稍后重试'
const MSG_UNAUTHORIZED = '登录已过期'

/** 拼接完整 URL（API_BASE 已在 config 中统一） */
export function fullUrl(url: string): string {
  if (!url) return API_BASE
  if (/^https?:\/\//i.test(url)) return url
  return `${API_BASE}${url.startsWith('/') ? url : `/${url}`}`
}

/** 构造鉴权头（供 excel.ts / uploadFile 复用） */
export function buildAuthHeaders(extra?: Record<string, string>): Record<string, string> {
  const headers: Record<string, string> = { ...(extra ?? {}) }
  try {
    const auth = useAuthStore()
    if (auth.token) headers.Authorization = `Bearer ${auth.token}`
  } catch (e) {
    console.warn('[request] auth store 尚未就绪', e)
  }
  try {
    const tenant = useTenantStore()
    if (tenant.currentTenantId) headers['X-Tenant-ID'] = String(tenant.currentTenantId)
  } catch (e) {
    console.warn('[request] tenant store 尚未就绪', e)
  }
  return headers
}

let redirecting = false

/** 401 统一处理：清缓存 + 回登录页（多次并发 401 只跳一次） */
export function handleUnauthorized(): void {
  try {
    useAuthStore().logout()
    useTenantStore().reset()
  } catch (e) {
    console.warn('[request] 清理登录态失败', e)
  }
  if (redirecting) return
  redirecting = true
  uni.reLaunch({
    url: routes.login,
    complete: () => {
      setTimeout(() => {
        redirecting = false
      }, 1000)
    },
  })
}

function isUnauthorized(status: number): boolean {
  return status === 401
}

function isOk(status: number): boolean {
  return status >= 200 && status < 300
}

/** 安全 JSON.parse：失败返回 null */
function safeJson(raw: unknown): unknown {
  if (raw === null || raw === undefined) return null
  if (typeof raw !== 'string') return raw
  const text = raw.trim()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

/**
 * 解析 { code, data, message } 包裹体。
 * 业务失败抛 Error(message)；非包裹体原样返回（容错）。
 */
function unwrap<T>(raw: unknown): T {
  const body = safeJson(raw)
  if (body && typeof body === 'object' && 'code' in (body as Record<string, unknown>)) {
    const res = body as ApiResponse<T>
    if (res.code !== 0) throw new Error(res.message || MSG_FAIL)
    return res.data
  }
  return body as T
}

/** 从错误体里尽量取 message */
function errorMessage(raw: unknown, fallback: string): string {
  const body = safeJson(raw)
  if (body && typeof body === 'object') {
    const msg = (body as { message?: unknown }).message
    if (typeof msg === 'string' && msg) return msg
  }
  return fallback
}

export function request<T>(opt: RequestOptions): Promise<T> {
  const { url, method = 'GET', data = null, timeout = REQUEST_TIMEOUT, header } = opt

  return new Promise<T>((resolve, reject) => {
    uni.request({
      url: fullUrl(url),
      method,
      data: data === null ? undefined : (data as UniNamespace.RequestOptions['data']),
      timeout,
      header: buildAuthHeaders(header),
      success: (res) => {
        const status = res.statusCode
        if (isUnauthorized(status)) {
          handleUnauthorized()
          reject(new Error(MSG_UNAUTHORIZED))
          return
        }
        if (!isOk(status)) {
          console.error('[request] HTTP 异常', status, url, res.data)
          reject(new Error(errorMessage(res.data, MSG_NETWORK)))
          return
        }
        try {
          resolve(unwrap<T>(res.data))
        } catch (e) {
          reject(e instanceof Error ? e : new Error(MSG_FAIL))
        }
      },
      fail: (err) => {
        console.error('[request] 网络错误', url, err)
        reject(new Error(MSG_NETWORK))
      },
    })
  })
}

/** 把 FormData 的 value 统一转成字符串（uni.uploadFile 的 formData 只支持基础类型） */
function stringifyFormData(formData?: Record<string, any>): Record<string, string> {
  const out: Record<string, string> = {}
  if (!formData) return out
  Object.keys(formData).forEach((key) => {
    const value = formData[key]
    if (value === undefined || value === null) return
    out[key] = typeof value === 'string' ? value : String(value)
  })
  return out
}

/* ---------------- 非 H5：uni.uploadFile ---------------- */

function uploadFileUni<T>(opt: UploadOptions): Promise<T> {
  const { url, filePath = '', name = 'file', formData, timeout = REQUEST_TIMEOUT } = opt
  return new Promise<T>((resolve, reject) => {
    uni.uploadFile({
      url: fullUrl(url),
      filePath,
      name,
      formData: stringifyFormData(formData),
      timeout,
      header: buildAuthHeaders(),
      success: (res) => {
        if (isUnauthorized(res.statusCode)) {
          handleUnauthorized()
          reject(new Error(MSG_UNAUTHORIZED))
          return
        }
        if (!isOk(res.statusCode)) {
          console.error('[uploadFile] HTTP 异常', res.statusCode, url, res.data)
          reject(new Error(errorMessage(res.data, MSG_NETWORK)))
          return
        }
        // uni.uploadFile 返回的是字符串，需要 parse 后再判定 code
        try {
          resolve(unwrap<T>(res.data))
        } catch (e) {
          reject(e instanceof Error ? e : new Error(MSG_FAIL))
        }
      },
      fail: (err) => {
        console.error('[uploadFile] 网络错误', url, err)
        reject(new Error(MSG_NETWORK))
      },
    })
  })
}

/* ---------------- H5：XHR + FormData（支持 File / Blob） ---------------- */

// #ifdef H5
function baseName(path: string): string {
  const clean = path.split('?')[0].split('#')[0]
  const idx = clean.lastIndexOf('/')
  return idx >= 0 ? clean.slice(idx + 1) : clean
}

function uploadFileH5<T>(opt: UploadOptions): Promise<T> {
  const { url, name = 'file', formData, timeout = REQUEST_TIMEOUT } = opt
  return new Promise<T>((resolve, reject) => {
    const form = new FormData()
    const filePart: Blob | null = opt.file ?? null
    if (filePart) {
      const filename = (filePart as File).name || baseName(opt.filePath || '') || 'file'
      form.append(name, filePart, filename)
    } else if (opt.filePath) {
      // H5 下没有 File 实例时（极少数情况）退化为 blob 拉取
      reject(new Error('文件读取失败，请重新选择'))
      return
    }
    const fields = stringifyFormData(formData)
    Object.keys(fields).forEach((key) => form.append(key, fields[key]))

    const xhr = new XMLHttpRequest()
    xhr.open('POST', fullUrl(url), true)
    const headers = buildAuthHeaders()
    Object.keys(headers).forEach((key) => xhr.setRequestHeader(key, headers[key]))
    xhr.timeout = timeout
    xhr.onload = () => {
      if (isUnauthorized(xhr.status)) {
        handleUnauthorized()
        reject(new Error(MSG_UNAUTHORIZED))
        return
      }
      if (!isOk(xhr.status)) {
        console.error('[uploadFile] HTTP 异常', xhr.status, url, xhr.responseText)
        reject(new Error(errorMessage(xhr.responseText, MSG_NETWORK)))
        return
      }
      try {
        resolve(unwrap<T>(xhr.responseText))
      } catch (e) {
        reject(e instanceof Error ? e : new Error(MSG_FAIL))
      }
    }
    xhr.onerror = () => {
      console.error('[uploadFile] 网络错误', url)
      reject(new Error(MSG_NETWORK))
    }
    xhr.ontimeout = () => {
      console.error('[uploadFile] 请求超时', url, timeout)
      reject(new Error(MSG_NETWORK))
    }
    xhr.send(form)
  })
}
// #endif

/**
 * 上传（H5 用 XHR + FormData 以支持 File/Blob；非 H5 用 uni.uploadFile）
 */
export function uploadFile<T>(opt: UploadOptions): Promise<T> {
  // #ifdef H5
  return uploadFileH5<T>(opt)
  // #endif
  // #ifndef H5
  return uploadFileUni<T>(opt)
  // #endif
}
