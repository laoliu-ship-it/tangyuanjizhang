/// <reference types="@dcloudio/types" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string
  readonly VITE_UPLOAD_BASE?: string
  readonly VITE_API_TARGET?: string
  readonly VITE_SECCHECK_BASE?: string
  readonly VITE_SECCHECK_APP?: string
  readonly VITE_SECCHECK_SCENE?: string
  readonly VITE_SECCHECK_ENABLED?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '@qiun/ucharts' {
  export default class uCharts {
    constructor(options: Record<string, unknown>)
    updateData(options?: Record<string, unknown>): void
  }
}
