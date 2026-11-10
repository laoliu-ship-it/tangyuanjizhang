/**
 * 二进制下载（Excel 导出 / 模板）。
 * H5：fetch -> blob -> 浏览器下载
 * 小程序 / App：uni.downloadFile -> uni.openDocument
 */
import { handleUnauthorized } from '@/api/request'

interface DownloadOptions {
  url: string
  filename: string
  header?: Record<string, string>
}

function fileTypeOf(filename: string): 'xlsx' | 'xls' | 'doc' | 'docx' | 'pdf' | 'ppt' | 'pptx' {
  const ext = filename.split('.').pop()?.toLowerCase() ?? 'xlsx'
  if (ext === 'xls') return 'xls'
  if (ext === 'doc') return 'doc'
  if (ext === 'docx') return 'docx'
  if (ext === 'pdf') return 'pdf'
  if (ext === 'ppt') return 'ppt'
  if (ext === 'pptx') return 'pptx'
  return 'xlsx'
}

export function downloadBinary(opt: DownloadOptions): Promise<void> {
  // #ifdef H5
  return downloadH5(opt)
  // #endif
  // #ifndef H5
  return downloadMini(opt)
  // #endif
}

// #ifdef H5
async function downloadH5(opt: DownloadOptions): Promise<void> {
  const res = await fetch(opt.url, { headers: opt.header })
  if (res.status === 401) {
    handleUnauthorized()
    throw new Error('登录已过期')
  }
  if (!res.ok) {
    throw new Error('下载失败，请稍后重试')
  }
  const contentType = res.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    const text = await res.text()
    let message = '导出失败'
    try {
      const body = JSON.parse(text) as { message?: string }
      if (body?.message) message = body.message
    } catch (e) {
      // 忽略解析失败
    }
    throw new Error(message)
  }
  const blob = await res.blob()
  if (!blob.size) throw new Error('导出失败：返回的文件为空')
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = opt.filename
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
// #endif

// #ifndef H5
function downloadMini(opt: DownloadOptions): Promise<void> {
  return new Promise((resolve, reject) => {
    uni.downloadFile({
      url: opt.url,
      header: opt.header,
      timeout: 60000,
      success: (res) => {
        if (res.statusCode === 401) {
          handleUnauthorized()
          reject(new Error('登录已过期'))
          return
        }
        if (res.statusCode !== 200) {
          reject(new Error('下载失败，请稍后重试'))
          return
        }
        uni.openDocument({
          filePath: res.tempFilePath,
          fileType: fileTypeOf(opt.filename),
          showMenu: true,
          success: () => resolve(),
          fail: () => reject(new Error('文件已下载，但当前环境无法打开，请使用电脑端导出')),
        })
      },
      fail: () => reject(new Error('下载失败，请检查网络或域名配置')),
    })
  })
}
// #endif
