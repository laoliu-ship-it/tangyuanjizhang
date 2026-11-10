/** 全局运行时配置（微信小程序优先，兼容 H5 / App） */

/** 读取本机存储里的调试覆盖值（仅小程序 / App 使用，方便真机连本地或测试环境） */
function readOverride(key: string): string {
  try {
    const value = uni.getStorageSync(key)
    return typeof value === 'string' ? value.trim() : ''
  } catch (e) {
    return ''
  }
}

/** 开发与生产统一使用正式接口域名（如需临时切换，使用 storage ty_api_base / ty_upload_base） */
const API_ORIGIN = 'https://tangyuanjizhang.zaixianapp.cn'

/**
 * 默认接口前缀。
 * - H5：同源部署走 /api，由 Go 后端统一提供（dev 由 vite 代理到 API_TARGET）
 * - 小程序 / App：必须使用完整域名（微信后台需配置 request 合法域名）；
 *   开发与生产统一使用正式域名，可通过 storage 键 ty_api_base 覆盖
 */
const DEFAULT_API_BASE = (() => {
  // #ifdef H5
  return '/api'
  // #endif
  // #ifndef H5
  const override = readOverride('ty_api_base')
  if (override) return override
  return `${API_ORIGIN}/api`
  // #endif
})()

/**
 * 默认上传资源前缀。
 * 后端 image_path 形如 /uploads/xxx.webp，由 Go 根路径静态服务。
 * H5 同源时为空串；小程序需要完整域名（downloadFile 合法域名）。
 */
const DEFAULT_UPLOAD_BASE = (() => {
  // #ifdef H5
  return ''
  // #endif
  // #ifndef H5
  const override = readOverride('ty_upload_base')
  if (override) return override
  return API_ORIGIN
  // #endif
})()

/** 接口前缀，可通过 .env / 构建环境变量覆盖 */
export const API_BASE = import.meta.env.VITE_API_BASE || DEFAULT_API_BASE

/** 上传资源前缀，可通过 .env / 构建环境变量覆盖 */
export const UPLOAD_BASE = import.meta.env.VITE_UPLOAD_BASE || DEFAULT_UPLOAD_BASE

/** 普通请求超时（ms） */
export const REQUEST_TIMEOUT = 20000

/** OCR 上传超时（ms），略短于服务端 30s */
export const OCR_TIMEOUT = 25000

/** 单次最多选图数量 */
export const MAX_IMAGE_COUNT = 9

/** 图片压缩目标上限（字节） */
export const MAX_FILE_SIZE = 1024 * 1024

/**
 * 小管家内容安全检测（仅微信小程序生效；H5 / App 自动跳过）。
 * 后台注册的小程序 slug，接口路径形如 /api/{SECCHECK_APP}/seccheck/image。
 */
export const SECCHECK_APP = import.meta.env.VITE_SECCHECK_APP || 'hehuojizhangzhushou'

/** 小管家服务地址（只传 origin，不要带 /api） */
export const SECCHECK_BASE =
  import.meta.env.VITE_SECCHECK_BASE || 'https://xiaoguanjia.zaixianapp.cn'

/** 微信内容安全场景值：1 资料 / 2 评论 / 3 论坛 / 4 社交日志 */
export const SECCHECK_SCENE = Number(import.meta.env.VITE_SECCHECK_SCENE || 2)

/**
 * 是否启用图片送审。正式环境必须开启（审核要求 fail-closed）；
 * 仅本地调试需要临时放行时设置 VITE_SECCHECK_ENABLED=false。
 */
export const SECCHECK_ENABLED = import.meta.env.VITE_SECCHECK_ENABLED !== 'false'

/** 列表默认每页条数 */
export const DEFAULT_PAGE_SIZE = 20

/** 租户名称固定后缀 */
export const TENANT_NAME_SUFFIX = '的记账本'

/** 本地缓存 key */
export const STORAGE_KEYS = {
  auth: 'ty_auth',
  tenant: 'ty_tenant',
  remember: 'ty_remember',
  wechatBind: 'ty_wechat_bind',
} as const

/** 后端存储的 image_path 形如 /uploads/xxx.webp，补全为可直接渲染的地址 */
export function resolveUpload(path?: string | null): string {
  if (!path) return ''
  if (/^(https?:|blob:|data:|wxfile:|file:|temp:)/i.test(path)) return path
  return `${UPLOAD_BASE}${path.startsWith('/') ? path : `/${path}`}`
}

/** 去掉租户名称固定后缀，返回展示用前缀 */
export function stripTenantSuffix(name?: string | null): string {
  if (!name) return ''
  return name.endsWith(TENANT_NAME_SUFFIX) ? name.slice(0, -TENANT_NAME_SUFFIX.length) : name
}
