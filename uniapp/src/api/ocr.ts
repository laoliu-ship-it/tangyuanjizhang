import { OCR_TIMEOUT } from '@/config'
import { uploadFile } from './request'
import type { OcrUploadResp } from './types'

export const ocrApi = {
  /**
   * 上传并识别票据。
   * H5 传 file（File 实例）；小程序 / App 传 filePath（临时文件路径）。
   * originalHash 仅 H5 有值。
   */
  upload(
    image: { filePath: string; file?: File; hash?: string },
    timeout: number = OCR_TIMEOUT,
  ): Promise<OcrUploadResp> {
    const formData: Record<string, string> = {}
    if (image.hash) formData.original_hash = image.hash
    return uploadFile<OcrUploadResp>({
      url: '/upload/ocr',
      filePath: image.filePath,
      file: image.file,
      name: 'file',
      formData,
      timeout,
    })
  },
}
