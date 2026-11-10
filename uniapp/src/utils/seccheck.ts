/**
 * 小管家「内容安检」客户端（微信小程序）
 *
 * 对接 xiaoguanjia/internal/modules/seccheck：
 *   POST /api/{app}/user/login              换取管家 JWT（typ=wx）
 *   POST /api/{app}/upload                  上传送检缩略图，拿 https 公网 URL
 *   POST /api/{app}/seccheck/image          提交微信 media_check_async
 *   GET  /api/{app}/seccheck/image/:traceId 轮询检测结果
 *
 * 设计要点：
 *  - 原图不出设备：本地生成 ~750px 缩略图送检，检测通过后才进入 OCR 流程；
 *  - fail-closed：risky / review / failed / 超时 一律按不通过处理；
 *  - 仅微信小程序启用，H5 / App 直接放行。
 */
import {
  SECCHECK_APP,
  SECCHECK_BASE,
  SECCHECK_ENABLED,
  SECCHECK_SCENE,
} from '@/config'

const TOKEN_KEY = 'ty_sg_token'
const POLL_INTERVAL = 1000
const POLL_MAX = 20
const MAX_SIDE = 750

export type SecCheckStatus = 'pass' | 'review' | 'risky' | 'failed' | 'pending'

export type SecCheckStep = 'compress' | 'upload' | 'submit' | 'poll'

export interface SecCheckResult {
  ok: boolean
  status: SecCheckStatus
  label?: number
  reason?: string
}

interface SecCheckRecord {
  status?: SecCheckStatus
  suggest?: string
  label?: number
  trace_id?: string
  err_msg?: string
}

interface SgEnvelope<T> {
  data?: T
  error?: string
  success?: boolean
}

// 微信小程序全局对象（@dcloudio/types 未声明 wx，这里自行声明以便直接引用）
declare const wx: Record<string, any> | undefined

function getWx(): Record<string, any> | undefined {
  // 老基础库 / 个别运行时没有 globalThis，优先用全局 wx
  if (typeof wx !== 'undefined' && wx) return wx
  if (typeof globalThis !== 'undefined') {
    return (globalThis as any).wx as Record<string, any> | undefined
  }
  return undefined
}

/** 是否走小管家送审（只在微信小程序里为 true） */
export function isSecCheckAvailable(): boolean {
  if (!SECCHECK_ENABLED || !SECCHECK_APP) return false
  const wx = getWx()
  return !!wx && typeof wx.login === 'function'
}

function getToken(): string {
  try {
    const value = uni.getStorageSync(TOKEN_KEY)
    return typeof value === 'string' ? value : ''
  } catch (e) {
    return ''
  }
}

function setToken(token: string): void {
  try {
    if (token) uni.setStorageSync(TOKEN_KEY, token)
    else uni.removeStorageSync(TOKEN_KEY)
  } catch (e) {
    // 忽略存储异常
  }
}

function apiUrl(path: string): string {
  return `${SECCHECK_BASE.replace(/\/+$/, '')}/api/${encodeURIComponent(SECCHECK_APP)}${path}`
}

function httpError(message: string, statusCode: number): Error {
  const err = new Error(message)
  ;(err as Error & { statusCode?: number }).statusCode = statusCode
  return err
}

function sendRaw<T>(
  path: string,
  method: 'GET' | 'POST',
  body?: Record<string, unknown>,
  withToken = true,
): Promise<T> {
  return new Promise((resolve, reject) => {
    const token = withToken ? getToken() : ''
    uni.request({
      url: apiUrl(path),
      method,
      data: body ?? {},
      timeout: 10000,
      header: {
        'content-type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      success: (res) => {
        const status = Number(res.statusCode ?? 0)
        const payload = res.data as SgEnvelope<T> | undefined
        if (status >= 200 && status < 300) {
          if (payload && payload.data !== undefined) resolve(payload.data)
          else resolve(payload as unknown as T)
          return
        }
        reject(httpError(payload?.error || `HTTP ${status}`, status))
      },
      fail: (err) => reject(new Error(err?.errMsg || '网络异常')),
    })
  })
}

/** 管家登录：wx.login 拿一次性 code，换管家 JWT（与业务后端登录互不影响） */
function login(): Promise<string> {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (res) => {
        const code = (res as { code?: string }).code
        if (!code) {
          reject(new Error('微信登录失败'))
          return
        }
        sendRaw<{ token?: string }>('/user/login', 'POST', { code }, false)
          .then((data) => {
            const token = data?.token || ''
            if (!token) {
              reject(new Error('管家登录失败'))
              return
            }
            setToken(token)
            resolve(token)
          })
          .catch(reject)
      },
      fail: (err) => reject(new Error(err?.errMsg || '微信登录失败')),
    })
  })
}

async function ensureToken(): Promise<void> {
  if (!getToken()) await login()
}

/** 带 401 重试的请求：管家 JWT 过期时自动重新登录一次 */
async function request<T>(
  path: string,
  method: 'GET' | 'POST',
  body?: Record<string, unknown>,
): Promise<T> {
  try {
    return await sendRaw<T>(path, method, body, true)
  } catch (err) {
    if ((err as { statusCode?: number }).statusCode === 401) {
      setToken('')
      await login()
      return sendRaw<T>(path, method, body, true)
    }
    throw err
  }
}

/* --------------------------- 本地缩略图 --------------------------- */

function compressFallback(wx: Record<string, any>, filePath: string): Promise<string> {
  return new Promise((resolve) => {
    if (typeof wx.compressImage !== 'function') {
      resolve(filePath)
      return
    }
    wx.compressImage({
      src: filePath,
      quality: 60,
      success: (res: { tempFilePath?: string }) => resolve(res.tempFilePath || filePath),
      fail: () => resolve(filePath),
    })
  })
}

/** 生成送检缩略图：优先离屏 canvas，失败退回 wx.compressImage，再不行用原图 */
function compressForCheck(filePath: string, maxSide = MAX_SIDE): Promise<string> {
  const wx = getWx()
  if (!wx || typeof wx.getImageInfo !== 'function') return Promise.resolve(filePath)

  return new Promise((resolve) => {
    wx.getImageInfo({
      src: filePath,
      success: (info: { width?: number; height?: number }) => {
        const width = Number(info.width ?? 0)
        const height = Number(info.height ?? 0)
        if (!width || !height) {
          compressFallback(wx, filePath).then(resolve)
          return
        }
        const scale = Math.min(1, maxSide / Math.max(width, height))
        const w = Math.max(1, Math.round(width * scale))
        const h = Math.max(1, Math.round(height * scale))

        try {
          const canvas = wx.createOffscreenCanvas({ type: '2d', width: w, height: h })
          const ctx = canvas.getContext('2d')
          const img = canvas.createImage()
          img.onload = () => {
            try {
              ctx.drawImage(img, 0, 0, w, h)
              wx.canvasToTempFilePath({
                canvas,
                fileType: 'jpg',
                quality: 0.7,
                success: (res: { tempFilePath?: string }) => {
                  resolve(res.tempFilePath || filePath)
                },
                fail: () => {
                  compressFallback(wx, filePath).then(resolve)
                },
              })
            } catch (e) {
              compressFallback(wx, filePath).then(resolve)
            }
          }
          img.onerror = () => {
            compressFallback(wx, filePath).then(resolve)
          }
          img.src = filePath
        } catch (e) {
          compressFallback(wx, filePath).then(resolve)
        }
      },
      fail: () => {
        compressFallback(wx, filePath).then(resolve)
      },
    })
  })
}

/* ----------------------------- 送审 ----------------------------- */

function uploadThumb(filePath: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const token = getToken()
    uni.uploadFile({
      url: apiUrl('/upload'),
      filePath,
      name: 'file',
      timeout: 20000,
      header: token ? { Authorization: `Bearer ${token}` } : {},
      success: (res) => {
        let payload: SgEnvelope<{ url?: string }> | undefined
        try {
          payload = JSON.parse(res.data) as SgEnvelope<{ url?: string }>
        } catch (e) {
          reject(new Error('上传响应异常'))
          return
        }
        const status = Number(res.statusCode ?? 0)
        if (status >= 200 && status < 300 && payload?.data?.url) {
          resolve(payload.data.url)
          return
        }
        reject(httpError(payload?.error || `上传失败 HTTP ${status}`, status))
      },
      fail: (err) => reject(new Error(err?.errMsg || '上传失败')),
    })
  })
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * 图片送审全流程。ok=true 才允许展示 / 进入 OCR。
 *
 * @param filePath wx.chooseImage / uni.chooseImage 返回的本地临时路径
 * @param onStep   步骤回调，用于更新“处理中”文案
 */
export async function checkImage(
  filePath: string,
  onStep?: (step: SecCheckStep) => void,
): Promise<SecCheckResult> {
  // 非微信小程序（H5 / App）不送审
  if (!isSecCheckAvailable()) return { ok: true, status: 'pass' }

  const emit = onStep ?? (() => {})

  try {
    await ensureToken()

    emit('compress')
    const thumbPath = await compressForCheck(filePath)

    emit('upload')
    let mediaUrl: string
    try {
      mediaUrl = await uploadThumb(thumbPath)
    } catch (err) {
      if ((err as { statusCode?: number }).statusCode === 401) {
        setToken('')
        await login()
        mediaUrl = await uploadThumb(thumbPath)
      } else {
        throw err
      }
    }

    emit('submit')
    const submitted = await request<SecCheckRecord>('/seccheck/image', 'POST', {
      media_url: mediaUrl,
      scene: SECCHECK_SCENE,
    })
    const submitStatus = submitted?.status
    if (submitStatus === 'failed') {
      return {
        ok: false,
        status: 'failed',
        reason: submitted?.err_msg || '图片送审失败',
      }
    }
    if (submitStatus && submitStatus !== 'pending') {
      return {
        ok: submitStatus === 'pass',
        status: submitStatus,
        label: submitted?.label,
        reason: submitted?.err_msg,
      }
    }

    const traceId = submitted?.trace_id || ''
    if (!traceId) {
      return { ok: false, status: 'failed', reason: '图片送审失败：缺少 trace_id' }
    }

    emit('poll')
    for (let i = 0; i < POLL_MAX; i++) {
      await sleep(POLL_INTERVAL)
      let current: SecCheckRecord
      try {
        current = await request<SecCheckRecord>(
          `/seccheck/image/${encodeURIComponent(traceId)}`,
          'GET',
        )
      } catch (e) {
        continue // 单次轮询失败继续重试
      }
      const status = current?.status
      if (!status || status === 'pending') continue
      return {
        ok: status === 'pass',
        status,
        label: current?.label,
        reason: current?.err_msg,
      }
    }

    return { ok: false, status: 'failed', reason: '检测结果等待超时' }
  } catch (err) {
    return {
      ok: false,
      status: 'failed',
      reason: err instanceof Error ? err.message : '内容安全检测失败',
    }
  }
}
