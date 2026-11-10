/**
 * 图片选择 / 压缩 / SHA-256 / 预览 URL 释放
 * 跨端说明：
 *  - H5：canvas 压缩 + crypto.subtle 计算 hash + URL.createObjectURL
 *  - 小程序 / App：uni.compressImage，跳过 hash（后端该字段可选）
 */
import { MAX_FILE_SIZE } from '@/config'

export interface PickedImage {
  path: string
  file?: File
  size: number
  name: string
}

interface ChooseImageTempFile {
  path?: string
  size?: number
  name?: string
  type?: string
}

/** 从路径中取文件名 */
function baseName(path: string): string {
  const clean = (path || '').split('?')[0].split('#')[0]
  const idx = clean.lastIndexOf('/')
  return idx >= 0 ? clean.slice(idx + 1) : clean
}

/** 选择图片（相机 / 相册），返回统一结构 */
export function pickImages(count = 9): Promise<PickedImage[]> {
  return new Promise((resolve, reject) => {
    uni.chooseImage({
      count: Math.max(1, Math.min(9, count)),
      sizeType: ['original'],
      sourceType: ['camera', 'album'],
      success: (res) => {
        const files = (res.tempFiles ?? []) as unknown as ChooseImageTempFile[]
        const paths = res.tempFilePaths ?? []
        const list: PickedImage[] = []
        const total = Math.max(files.length, paths.length)
        for (let i = 0; i < total; i++) {
          const temp = files[i] ?? {}
          const path = temp.path || paths[i] || ''
          if (!path) continue
          let file: File | undefined
          // #ifdef H5
          if (typeof File !== 'undefined' && temp instanceof File) {
            file = temp as unknown as File
          }
          // #endif
          list.push({
            path,
            file,
            size: Number(temp.size ?? file?.size ?? 0),
            name: String(temp.name ?? file?.name ?? baseName(path) ?? `image_${i + 1}.jpg`),
          })
        }
        resolve(list)
      },
      fail: (err) => {
        const msg = String((err as { errMsg?: string })?.errMsg ?? '')
        reject(new Error(msg.includes('cancel') ? '已取消选择' : '选择图片失败'))
      },
    })
  })
}

/* ----------------------------- H5 压缩 ----------------------------- */

// #ifdef H5
function loadImageElement(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('图片加载失败'))
    img.src = src
  })
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('图片压缩失败'))),
      type,
      quality,
    )
  })
}

let webpSupported: boolean | null = null
function supportsWebp(): boolean {
  if (webpSupported !== null) return webpSupported
  try {
    const canvas = document.createElement('canvas')
    canvas.width = 1
    canvas.height = 1
    webpSupported = canvas.toDataURL('image/webp').startsWith('data:image/webp')
  } catch (e) {
    webpSupported = false
  }
  return webpSupported
}

async function compressH5(img: PickedImage): Promise<PickedImage> {
  if (img.size > 0 && img.size <= MAX_FILE_SIZE) return img
  let sourceUrl = img.path
  let objectUrl = ''
  try {
    let blob: Blob | null = img.file ?? null
    if (!blob) {
      const res = await fetch(img.path)
      blob = await res.blob()
    }
    objectUrl = URL.createObjectURL(blob)
    sourceUrl = objectUrl
    const element = await loadImageElement(sourceUrl)

    const MAX_SIDE = 2048
    let width = element.naturalWidth || element.width
    let height = element.naturalHeight || element.height
    if (width > MAX_SIDE) {
      height = Math.round((height * MAX_SIDE) / width)
      width = MAX_SIDE
    }
    if (height > MAX_SIDE) {
      width = Math.round((width * MAX_SIDE) / height)
      height = MAX_SIDE
    }

    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return img
    ctx.drawImage(element, 0, 0, width, height)

    const useWebp = supportsWebp()
    const mime = useWebp ? 'image/webp' : 'image/jpeg'
    const ext = useWebp ? 'webp' : 'jpg'
    const rawName = img.name.replace(/\.[^.]+$/, '') || `image_${Date.now()}`

    let quality = 0.85
    let result: Blob | null = null
    for (let i = 0; i < 7; i++) {
      result = await canvasToBlob(canvas, mime, quality)
      if (result.size <= MAX_FILE_SIZE || quality <= 0.3) break
      quality = Math.max(0.3, Number((quality - 0.1).toFixed(2)))
    }
    if (!result) return img

    const file = new File([result], `${rawName}.${ext}`, { type: mime })
    const newPath = URL.createObjectURL(file)
    // 原图是 chooseImage 生成的 blob URL，压缩完成后可释放
    if (img.path.startsWith('blob:') && img.path !== newPath) {
      URL.revokeObjectURL(img.path)
    }
    return { path: newPath, file, size: file.size, name: file.name }
  } catch (e) {
    console.warn('[image] H5 压缩失败，使用原图', e)
    return img
  } finally {
    if (objectUrl) URL.revokeObjectURL(objectUrl)
  }
}
// #endif

/* --------------------------- 非 H5 压缩 --------------------------- */

// #ifndef H5
function getFileSize(path: string): Promise<number> {
  return new Promise((resolve) => {
    uni.getFileInfo({
      filePath: path,
      success: (res) => resolve(Number(res.size ?? 0)),
      fail: () => resolve(0),
    })
  })
}

function compressOnce(path: string, quality: number): Promise<string> {
  return new Promise((resolve) => {
    uni.compressImage({
      src: path,
      quality,
      success: (res) => resolve(res.tempFilePath || path),
      fail: () => resolve(''),
    })
  })
}

async function compressUni(img: PickedImage): Promise<PickedImage> {
  if (img.size > 0 && img.size <= MAX_FILE_SIZE) return img
  let current = img
  const qualities = [80, 60, 40, 20]
  for (const quality of qualities) {
    const path = await compressOnce(current.path, quality)
    if (!path || path === current.path) break
    const size = await getFileSize(path)
    const next: PickedImage = {
      path,
      size: size || current.size,
      name: current.name.replace(/\.[^.]+$/, '') + '.jpg',
    }
    current = next
    if (current.size > 0 && current.size <= MAX_FILE_SIZE) break
  }
  return current
}
// #endif

/** 压缩到 1MB 内；失败返回原图。H5 会把原 blob URL 释放并返回新 URL。 */
export async function compressImage(img: PickedImage): Promise<PickedImage> {
  // #ifdef H5
  return compressH5(img)
  // #endif
  // #ifndef H5
  return compressUni(img)
  // #endif
}

/** SHA-256（仅 H5；其它平台返回空串，后端 original_hash 可选） */
export async function sha256(img: PickedImage): Promise<string> {
  // #ifdef H5
  try {
    let buffer: ArrayBuffer
    if (img.file) {
      buffer = await img.file.arrayBuffer()
    } else {
      const res = await fetch(img.path)
      buffer = await res.arrayBuffer()
    }
    if (typeof crypto !== 'undefined' && crypto.subtle) {
      const digest = await crypto.subtle.digest('SHA-256', buffer)
      return Array.from(new Uint8Array(digest))
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('')
    }
  } catch (e) {
    console.warn('[image] SHA-256 计算失败，跳过去重', e)
  }
  // #endif
  return ''
}

/** 释放 H5 blob 预览 URL（其它平台无需处理） */
export function revokePreview(path?: string | null): void {
  if (!path) return
  // #ifdef H5
  if (path.startsWith('blob:')) {
    try {
      URL.revokeObjectURL(path)
    } catch (e) {
      // 忽略重复释放
    }
  }
  // #endif
}

/** 全屏预览图片（本地临时路径 / 服务端路径均可） */
export function previewImages(urls: string[], current?: string): void {
  const list = urls.filter(Boolean)
  if (!list.length) return
  uni.previewImage({
    urls: list,
    current: current && list.includes(current) ? current : list[0],
  })
}
