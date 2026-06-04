import { createSHA256 } from 'hash-wasm'

const MAX_WIDTH = 2048
const MAX_HEIGHT = 2048
const MAX_FILE_SIZE = 1 * 1024 * 1024 // 1MB

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

/** toWebP 转换结果 */
export interface ToWebPResult {
  file?: File
  oversized?: boolean
  size?: number
}

/** 使用 Canvas 压缩图片 */
function compressWithCanvas(file: File, quality = 0.85): Promise<ToWebPResult> {
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

      // 优先 WebP，不支持则用 JPEG
      const format = checkCanvasWebPSupport() ? 'image/webp' : 'image/jpeg'
      const ext = checkCanvasWebPSupport() ? 'webp' : 'jpg'

      canvas.toBlob(blob => {
        if (!blob) { reject(new Error('图片压缩失败')); return }
        const newName = file.name.replace(/\.[^.]+$/, '') + '.' + ext
        const resultFile = new File([blob], newName, { type: format })
        if (blob.size > MAX_FILE_SIZE) {
          resolve({ oversized: true, size: blob.size, file: resultFile })
        } else {
          resolve({ file: resultFile })
        }
      }, format, quality)
    }
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('图片加载失败')) }
    img.src = url
  })
}

/**
 * 压缩图片（质量 85%）。
 * 所有图片都会压缩，失败则返回原图。
 */
export async function toWebP(file: File, quality = 0.85): Promise<ToWebPResult> {
  try {
    return await compressWithCanvas(file, quality)
  } catch (err) {
    console.warn('[图片压缩] 压缩失败，返回原图:', err)
    if (file.size > MAX_FILE_SIZE) {
      return { oversized: true, size: file.size, file }
    }
    return { file }
  }
}

/** 返回当前图片引擎状态 */
export function getImageEngineStatus(): 'canvas' {
  return 'canvas'
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