import { initializeImageMagick, ImageMagick, MagickFormat } from '@imagemagick/magick-wasm'
import { createSHA256 } from 'hash-wasm'

const MAGICK_WASM_URL = '/magick.wasm'

const MAX_WIDTH = 2048
const MAX_HEIGHT = 2048
const MAX_FILE_SIZE = 1 * 1024 * 1024 // 1MB

/** magick-wasm 初始化状态 */
let magickState: 'idle' | 'loading' | 'ready' | 'failed' = 'idle'
let magickInitPromise: Promise<void> | null = null

/** 延迟启动 wasm 加载（首页打开后 2s 开始） */
let lazyInitTimer: ReturnType<typeof setTimeout> | null = null

function scheduleLazyInit() {
  if (magickState !== 'idle') return
  if (lazyInitTimer !== null) return
  lazyInitTimer = setTimeout(() => {
    ensureMagick().catch(() => {})
  }, 2000)
}

/** 初始化 magick-wasm */
async function ensureMagick(): Promise<void> {
  if (magickState === 'ready') return
  if (magickState === 'loading' && magickInitPromise) return magickInitPromise

  magickState = 'loading'
  magickInitPromise = fetch(MAGICK_WASM_URL)
    .then(resp => resp.arrayBuffer())
    .then(buf => initializeImageMagick(new Uint8Array(buf)))
    .then(() => { magickState = 'ready' })
    .catch(() => { magickState = 'failed' })

  return magickInitPromise
}

/** 检测 Canvas 是否支持 WebP 输出 */
let canvasSupportsWebP: boolean | null = null
function checkCanvasWebPSupport(): boolean {
  if (canvasSupportsWebP !== null) return canvasSupportsWebP
  const canvas = document.createElement('canvas')
  canvas.width = 1
  canvas.height = 1
  const dataUrl = canvas.toDataURL('image/webp')
  canvasSupportsWebP = dataUrl.startsWith('data:image/webp')
  return canvasSupportsWebP
}

/** 读取文件头判断真实格式（不依赖浏览器 MIME） */
function detectImageType(buf: ArrayBuffer): 'png' | 'jpeg' | 'webp' | 'other' {
  const bytes = new Uint8Array(buf)
  if (bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47) return 'png'
  if (bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) return 'jpeg'
  if (bytes.length >= 12 && bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46
    && bytes[8] === 0x57 && bytes[9] === 0x45 && bytes[10] === 0x42 && bytes[11] === 0x50) return 'webp'
  return 'other'
}

/** toWebP 转换结果 */
export interface ToWebPResult {
  file?: File
  oversized?: boolean
  size?: number
}

/** 使用 magick-wasm 转换图片 */
async function toWebPMagick(file: File, quality = 0.85): Promise<ToWebPResult> {
  await ensureMagick()
  if (magickState !== 'ready') throw new Error('magick-wasm not ready')

  const bytes = new Uint8Array(await file.arrayBuffer())

  return ImageMagick.read(bytes, (img) => {
    let w = img.width
    let h = img.height
    if (w > MAX_WIDTH) { h = Math.round(h * MAX_WIDTH / w); w = MAX_WIDTH }
    if (h > MAX_HEIGHT) { w = Math.round(w * MAX_HEIGHT / h); h = MAX_HEIGHT }
    if (w !== img.width || h !== img.height) img.resize(w, h)

    img.quality = Math.round(quality * 100)

    return img.write(MagickFormat.WebP, (data: Uint8Array) => {
      const newName = file.name.replace(/\.[^.]+$/, '') + '.webp'
      const resultFile = new File([data.slice()], newName, { type: 'image/webp' })
      if (data.length > MAX_FILE_SIZE) {
        return { oversized: true, size: data.length, file: resultFile }
      }
      return { file: resultFile }
    })
  })
}

/** 使用 Canvas 转换 */
function toWebPCanvas(file: File, quality = 0.85): Promise<ToWebPResult> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      URL.revokeObjectURL(url)
      let w = img.naturalWidth
      let h = img.naturalHeight
      if (w > MAX_WIDTH) { h = Math.round(h * MAX_WIDTH / w); w = MAX_WIDTH }
      if (h > MAX_HEIGHT) { w = Math.round(w * MAX_HEIGHT / h); h = MAX_HEIGHT }

      const canvas = document.createElement('canvas')
      canvas.width = w
      canvas.height = h
      const ctx = canvas.getContext('2d')!
      ctx.drawImage(img, 0, 0, w, h)
      canvas.toBlob(blob => {
        if (!blob) { reject(new Error('WebP conversion failed')); return }
        const newName = file.name.replace(/\.[^.]+$/, '') + '.webp'
        const resultFile = new File([blob], newName, { type: 'image/webp' })
        if (blob.size > MAX_FILE_SIZE) {
          resolve({ oversized: true, size: blob.size, file: resultFile })
        } else {
          resolve({ file: resultFile })
        }
      }, 'image/webp', quality)
    }
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('Image load failed')) }
    img.src = url
  })
}

/**
 * 转换图片为 WebP。
 * 策略：
 * 1. Canvas 支持 WebP → 优先用 Canvas（快速，无需 wasm）
 * 2. Canvas 不支持 WebP（如 Safari）→ 尝试 magick-wasm
 * 3. magick-wasm 未加载好 → 触发加载并等待
 * 4. 两者都失败 → reject
 *
 * 首次调用 toWebP 时会触发 wasm 延迟加载（如果还未开始）。
 */
export async function toWebP(file: File, quality = 0.85): Promise<ToWebPResult> {
  const header = await file.slice(0, 12).arrayBuffer()
  const realType = detectImageType(header)

  if (realType === 'webp') {
    if (file.size <= MAX_FILE_SIZE) return { file }
    return { oversized: true, size: file.size, file }
  }

  // 触发 wasm 延迟加载
  scheduleLazyInit()

  // Canvas 支持 WebP → 直接用
  if (checkCanvasWebPSupport()) {
    return toWebPCanvas(file, quality)
  }

  // Canvas 不支持 WebP → 必须用 magick-wasm
  if (magickState === 'failed') {
    throw new Error('图片转换不可用：WebAssembly 加载失败')
  }

  // 如果 wasm 还没开始加载或还在加载，先触发/等待
  if (magickState === 'idle') {
    ensureMagick().catch(() => {})
  }

  // 等待 wasm 就绪
  await ensureMagick()

  if (magickState === 'ready') {
    return toWebPMagick(file, quality)
  }

  throw new Error('图片转换不可用：WebAssembly 加载失败')
}

/** 返回当前图片引擎状态，供 UI 显示 loading */
export function getImageEngineStatus(): 'canvas' | 'wasm-loading' | 'wasm-ready' | 'unavailable' {
  if (checkCanvasWebPSupport()) return 'canvas'
  if (magickState === 'ready') return 'wasm-ready'
  if (magickState === 'loading' || magickState === 'idle') return 'wasm-loading'
  return 'unavailable'
}

/** 对原始文件计算 SHA-256 hash，用于服务端去重 */
export async function fileSHA256(file: File): Promise<string> {
  const buf = await file.arrayBuffer()

  // 优先使用原生 crypto.subtle（更快）
  if (typeof crypto !== 'undefined' && crypto.subtle) {
    try {
      const hashBuf = await crypto.subtle.digest('SHA-256', buf)
      return Array.from(new Uint8Array(hashBuf))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('')
    } catch {
      // 降级到 hash-wasm
    }
  }

  // 使用 hash-wasm 纯 JS 实现作为降级方案
  const hasher = await createSHA256()
  hasher.init()
  hasher.update(new Uint8Array(buf))
  return hasher.digest('hex')
}
