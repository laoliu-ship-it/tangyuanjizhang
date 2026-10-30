/** 全局运行时配置 */

/** 接口前缀。H5 同源部署走 /api，由 Go 后端统一提供 */
export const API_BASE = import.meta.env.VITE_API_BASE || '/api'

/** 上传资源前缀。后端 /uploads 由 Go 静态服务，H5 同源时为空串 */
export const UPLOAD_BASE = import.meta.env.VITE_UPLOAD_BASE || ''

/** 普通请求超时（ms） */
export const REQUEST_TIMEOUT = 20000

/** OCR 上传超时（ms），略短于服务端 30s */
export const OCR_TIMEOUT = 25000

/** 单次最多选图数量 */
export const MAX_IMAGE_COUNT = 9

/** 图片压缩目标上限（字节） */
export const MAX_FILE_SIZE = 1024 * 1024

/** 列表默认每页条数 */
export const DEFAULT_PAGE_SIZE = 20

/** 租户名称固定后缀 */
export const TENANT_NAME_SUFFIX = '的记账本'

/** 本地缓存 key */
export const STORAGE_KEYS = {
  auth: 'ty_auth',
  tenant: 'ty_tenant',
  remember: 'ty_remember',
} as const

/** 后端存储的 image_path 形如 /uploads/xxx.webp，补全为可直接渲染的地址 */
export function resolveUpload(path?: string | null): string {
  if (!path) return ''
  if (/^https?:\/\//.test(path) || path.startsWith('data:')) return path
  return `${UPLOAD_BASE}${path.startsWith('/') ? path : `/${path}`}`
}
