/**
 * Excel 导入 / 导出。
 * - 导出：H5 浏览器下载；小程序 downloadFile + openDocument
 * - 导入：H5 选择本地文件；微信小程序 chooseMessageFile（从聊天记录选择）
 */
import dayjs from 'dayjs'
import { buildAuthHeaders, fullUrl, uploadFile } from './request'
import { downloadBinary } from '@/utils/download'
import { useTenantStore } from '@/store/tenant'
import type { ColumnMapping, ImportResult, ParseHeadersResult } from './types'

export interface ExcelFile {
  path: string
  name: string
  /** 仅 H5 有 File 实例 */
  raw?: File
}

function toQuery(params: Record<string, string | undefined>): string {
  const parts: string[] = []
  Object.keys(params).forEach((key) => {
    const value = params[key]
    if (value === undefined || value === null || value === '') return
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
  })
  return parts.length ? `?${parts.join('&')}` : ''
}

function exportFileName(): string {
  const tenant = useTenantStore()
  const name = tenant.currentTenant?.name || '账本'
  return `${name}-${dayjs().format('YYYY-MM-DD-HH-mm')}.xlsx`
}

export const excelApi = {
  /** 导出交易 Excel（参数取当前筛选条件） */
  async exportExcel(params: {
    start_date?: string
    end_date?: string
    type?: string
  }): Promise<void> {
    const query = toQuery({
      start_date: params.start_date,
      end_date: params.end_date,
      type: params.type,
    })
    await downloadBinary({
      url: fullUrl(`/export/excel${query}`),
      filename: exportFileName(),
      header: buildAuthHeaders(),
    })
  },

  /** 下载导入模板 */
  async downloadTemplate(): Promise<void> {
    await downloadBinary({
      url: fullUrl('/import/template'),
      filename: '导入模板.xlsx',
      header: buildAuthHeaders(),
    })
  },

  /** 解析 Excel 表头（返回 sheet 列表、表头、样例数据、推荐映射） */
  parseHeaders(file: ExcelFile, sheetIndex?: number): Promise<ParseHeadersResult> {
    const query = sheetIndex !== undefined ? `?sheet=${sheetIndex}` : ''
    return uploadFile<ParseHeadersResult>({
      url: `/import/parse-headers${query}`,
      filePath: file.path,
      file: file.raw,
      name: 'file',
    })
  },

  /** 导入 Excel；dryRun=true 只预检不落库 */
  importExcel(
    file: ExcelFile,
    mapping?: ColumnMapping,
    sheetIndex?: number,
    dryRun?: boolean,
  ): Promise<ImportResult> {
    const formData: Record<string, string> = {}
    if (mapping) formData.mapping = JSON.stringify(mapping)
    if (sheetIndex !== undefined) formData.sheet = String(sheetIndex)
    if (dryRun) formData.dry_run = 'true'
    return uploadFile<ImportResult>({
      url: '/import/excel',
      filePath: file.path,
      file: file.raw,
      name: 'file',
      formData,
      timeout: 60000,
    })
  },

  /** 选择本地 Excel 文件 */
  async chooseExcelFile(): Promise<ExcelFile> {
    // #ifdef H5
    return chooseH5()
    // #endif
    // #ifdef MP-WEIXIN
    return chooseMp()
    // #endif
    // #ifndef H5
    // #ifndef MP-WEIXIN
    throw new Error('当前环境不支持选择文件，请使用电脑端导入')
    // #endif
    // #endif
  },
}

/* ------------------------------- H5 选择 ------------------------------- */

// #ifdef H5
function chooseH5(): Promise<ExcelFile> {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.xlsx,.xls'
    input.style.position = 'fixed'
    input.style.left = '-9999px'
    document.body.appendChild(input)

    let settled = false
    const cleanup = () => {
      window.removeEventListener('focus', onFocus)
      if (input.parentNode) input.parentNode.removeChild(input)
    }
    const finish = (fn: () => void) => {
      if (settled) return
      settled = true
      cleanup()
      fn()
    }
    const onFocus = () => {
      setTimeout(() => {
        finish(() => reject(new Error('已取消选择')))
      }, 400)
    }

    input.addEventListener('change', () => {
      const file = input.files?.[0]
      finish(() => {
        if (!file) {
          reject(new Error('未选择文件'))
          return
        }
        resolve({ path: URL.createObjectURL(file), name: file.name, raw: file })
      })
    })
    input.addEventListener('cancel', () => {
      finish(() => reject(new Error('已取消选择')))
    })
    window.addEventListener('focus', onFocus)
    input.click()
  })
}
// #endif

/* --------------------------- 微信小程序选择 --------------------------- */

// #ifdef MP-WEIXIN

interface MessageFile {
  path: string
  name: string
  size?: number
}

function chooseMp(): Promise<ExcelFile> {
  return new Promise((resolve, reject) => {
    if (typeof uni.chooseMessageFile !== 'function') {
      reject(new Error('当前小程序版本不支持选择文件，请升级微信后重试'))
      return
    }
    uni.chooseMessageFile({
      count: 1,
      type: 'file',
      extension: ['xlsx', 'xls'],
      success: (res) => {
        const files = (res.tempFiles ?? []) as unknown as MessageFile[]
        const file = files[0]
        if (!file?.path) {
          reject(new Error('未选择文件'))
          return
        }
        resolve({ path: file.path, name: file.name || '导入文件.xlsx' })
      },
      fail: (err) => {
        const msg = String((err as { errMsg?: string })?.errMsg ?? '')
        reject(new Error(msg.includes('cancel') ? '已取消选择' : '选择文件失败，请从聊天记录中选择 Excel 文件'))
      },
    })
  })
}
// #endif
