import { request } from './request'
import type { LLMAnalyzeResponse, TenantLLMConfig, TenantPrompt } from './types'

export const llmApi = {
  /** GET /llm/config */
  getConfig(): Promise<TenantLLMConfig> {
    return request<TenantLLMConfig>({ url: '/llm/config' })
  },

  /** PUT /llm/config */
  saveConfig(
    data: Partial<
      Pick<TenantLLMConfig, 'enabled' | 'use_platform' | 'provider' | 'base_url' | 'model' | 'mode'>
    > & { api_key?: string },
  ): Promise<TenantLLMConfig> {
    return request<TenantLLMConfig>({ url: '/llm/config', method: 'PUT', data })
  },

  /** GET /llm/prompt */
  getPrompt(): Promise<TenantPrompt> {
    return request<TenantPrompt>({ url: '/llm/prompt' })
  },

  /** PUT /llm/prompt（custom_prompt 为空 = 恢复平台默认） */
  savePrompt(data: { custom_prompt: string }): Promise<TenantPrompt> {
    return request<TenantPrompt>({ url: '/llm/prompt', method: 'PUT', data })
  },

  /** POST /llm/analyze（新流程只传 ocr_id） */
  analyzeByOcrId(data: { ocr_id: number }): Promise<LLMAnalyzeResponse> {
    return request<LLMAnalyzeResponse>({ url: '/llm/analyze', method: 'POST', data })
  },
}
