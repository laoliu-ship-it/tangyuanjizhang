# 子任务卡（每张卡交给一个子 agent）

> **全局规则（每个子 agent 必须遵守）**
> 1. 只修改/创建**自己卡内列出的文件**；其它文件一律不动（共用文件冲突 = 返工）。
> 2. 不修改 `pages.json`、`manifest.json`、`package.json`、`vite.config.ts`、`tsconfig.json`、`src/config.ts`、`src/api/types.ts`、`src/store/auth.ts`、`src/store/tenant.ts`、`src/uni.scss`、`src/styles/common.scss`（需要新增全局样式时，用组件内 scoped 样式）。
> 3. 不执行 `npm install` / `npm run build`（会互相踩 dist）；需要自检时执行 `npx vue-tsc --noEmit`，只关注自己文件的报错。
> 4. 编码规范：Vue 3 `<script setup lang="ts">`；样式 `lang="scss"` + `scoped`；只用 rpx；颜色/圆角/字号用 `$c-*` `$radius-*` `$fs-*` 令牌。
> 5. 所有网络请求走 `api/*`，禁止页面内 `uni.request`；所有覆盖层用 `AppModal`；提示用 `utils/ui.ts`。
> 6. 金额用 `utils/format.ts#money`，日期用 `utils/datetime.ts`。
> 7. 每个页面必须处理：加载态、空态、错误态（`utils/ui.ts#toastError`）。
> 8. 完成后在回复里列出：新建/修改的文件清单 + 未实现/简化点。

---

## T01 · 基础设施（阻塞所有任务）

**目标**：一次性交付请求层、API 模块、缓存 store、工具函数、权限守卫。后续所有 agent 都依赖这些签名，**签名不得更改**。

**拥有文件**

```
src/api/request.ts
src/api/auth.ts  tenant.ts  category.ts  merchant.ts  transaction.ts
      statistics.ts  ocr.ts  llm.ts  credit.ts  rbac.ts  excel.ts
src/store/meta.ts
src/utils/format.ts  datetime.ts  ui.ts  image.ts  nav.ts  download.ts
src/composables/useAuthGuard.ts  useRole.ts
```

**必须实现的签名（不可改名/改参）**

```ts
// api/request.ts
export interface RequestOptions {
  url: string
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: Record<string, any> | any[] | null
  timeout?: number
  header?: Record<string, string>
}
export function request<T>(opt: RequestOptions): Promise<T>
/** 上传（H5 用 XHR+FormData 以支持 File/Blob；非 H5 用 uni.uploadFile） */
export function uploadFile<T>(opt: {
  url: string
  filePath?: string
  file?: File          // #ifdef H5 编译后可传入
  name?: string        // 默认 'file'
  formData?: Record<string, any>
  timeout?: number
}): Promise<T>

// api/auth.ts
export const authApi = {
  register(data: { username: string; email: string; password: string; book_type: string }): Promise<AuthResult>
  login(data: { email: string; password: string }): Promise<AuthResult>
}

// api/tenant.ts
export const tenantApi = {
  list(): Promise<Tenant[]>
  create(data: { name: string }): Promise<Tenant>
  update(id: number, data: { name: string }): Promise<Tenant>
  getMembers(id: number): Promise<TenantMember[]>
  addMember(id: number, data: { username: string; role: string }): Promise<TenantMember>
  removeMember(id: number, userId: number): Promise<null>
  updateMemberRole(id: number, userId: number, role: string): Promise<null>
  getSettings(id: number): Promise<TenantSettings>
  updateSettings(id: number, data: TenantSettings): Promise<TenantSettings>
}

// api/category.ts / api/merchant.ts
export const categoryApi = {
  list(): Promise<Category[]>
  create(data: { name: string; type: TxType; icon: string }): Promise<Category>
  update(id: number, data: { name: string; type: TxType; icon: string }): Promise<Category>
  remove(id: number): Promise<null>
}
export const merchantApi = {
  list(): Promise<Merchant[]>
  create(name: string): Promise<Merchant>
  update(id: number, name: string): Promise<Merchant>
  remove(id: number): Promise<null>
}

// api/transaction.ts
export const transactionApi = {
  list(params: TransactionListParams): Promise<TransactionListResponse>
  create(data: TransactionCreatePayload): Promise<Transaction>
  update(id: number, data: TransactionCreatePayload): Promise<Transaction>
  remove(id: number): Promise<null>
  batchRemove(ids: number[]): Promise<number>
}

// api/statistics.ts
export const statisticsApi = {
  daily(date: string): Promise<DailyStatistics>
  monthly(year: number, month: number): Promise<MonthlyStatistics>
  yearly(year: number): Promise<YearlyStatistics>
  range(start: string, end: string): Promise<RangeStatistics>
  refreshMerchants(params:
    | { view: 'monthly'; year: number; month: number }
    | { view: 'yearly'; year: number }
    | { view: 'range'; start: string; end: string }
  ): Promise<MerchantStatsResp>
}

// api/ocr.ts
export const ocrApi = {
  /** 上传并识别，originalHash 仅 H5 有值 */
  upload(image: { filePath: string; file?: File; hash?: string }, timeout?: number): Promise<OcrUploadResp>
}

// api/llm.ts
export const llmApi = {
  getConfig(): Promise<TenantLLMConfig>
  saveConfig(data: Partial<Pick<TenantLLMConfig,'enabled'|'use_platform'|'provider'|'base_url'|'model'|'mode'> & { api_key?: string }>): Promise<TenantLLMConfig>
  getPrompt(): Promise<TenantPrompt>
  savePrompt(data: { custom_prompt: string }): Promise<TenantPrompt>
  analyzeByOcrId(data: { ocr_id: number }): Promise<LLMAnalyzeResponse>
}

// api/credit.ts
export const creditApi = {
  getBalance(): Promise<CreditsResp>
  getLogs(page: number, size: number, type?: 'income' | 'expense'): Promise<CreditLogsPageResp>
}

// api/rbac.ts
export const rbacApi = {
  listPermissions(): Promise<PermissionEntry[]>
  listRoles(tenantId: number): Promise<TenantRole[]>
  createRole(tenantId: number, data: { name: string; permissions: PermissionEntry[] }): Promise<TenantRole>
  updateRole(tenantId: number, roleId: number, data: { permissions: PermissionEntry[] }): Promise<TenantRole>
  deleteRole(tenantId: number, roleId: number): Promise<null>
}

// api/excel.ts
export const excelApi = {
  /** H5: 触发浏览器下载；非 H5: toast 提示用电脑导出 */
  exportExcel(params: { start_date?: string; end_date?: string; type?: string }): Promise<void>
  downloadTemplate(): Promise<void>
  parseHeaders(file: { path: string; raw?: File }, sheetIndex?: number): Promise<ParseHeadersResult>
  importExcel(file: { path: string; raw?: File }, mapping?: ColumnMapping, sheetIndex?: number, dryRun?: boolean): Promise<ImportResult>
  /** H5 动态创建 input 选 .xlsx/.xls；非 H5 抛出友好错误 */
  chooseExcelFile(): Promise<{ path: string; name: string; raw?: File }>
}

// store/meta.ts
export const useMetaStore = defineStore('meta', {
  state: () => ({ categories: [] as Category[], merchants: [] as Merchant[], role: 'member' as 'owner'|'admin'|'member', loadedAt: 0 }),
  actions: {
    ensureCategories(force?: boolean): Promise<Category[]>
    ensureMerchants(force?: boolean): Promise<Merchant[]>
    ensureRole(): Promise<'owner'|'admin'|'member'>   // 用 tenantApi.getMembers + authStore.userId
    setRole(role: 'owner'|'admin'|'member'): void
    reset(): void
  }
})

// utils/format.ts
export function money(n?: number | null): string            // 1,234.56（无 ¥）
export function moneySigned(type: TxType, n?: number | null): string  // +¥1,234.56 / -¥1,234.56
export function compactMoney(n?: number | null): string     // 1.2万 / 3,456
export function percent(v: number, total: number): string   // 12.3%

// utils/datetime.ts
export function todayStr(): string                          // YYYY-MM-DD
export function toApiDateTime(local: string): string        // 'YYYY-MM-DDTHH:mm' -> 'YYYY-MM-DD HH:mm:00'
export function fromApiDateTime(api: string): string        // 'YYYY-MM-DD HH:mm:ss' -> 'YYYY-MM-DDTHH:mm'
export function humanDateTime(s: string): string            // MM-DD HH:mm（带容错）
export function humanDate(s: string): string                // YYYY-MM-DD
export function extractTimeFromOcrDate(ocrDate?: string, fallback?: string): string

// utils/ui.ts
export function toast(title: string, icon?: 'none'|'success'|'error'): void
export function toastError(err: unknown, fallback?: string): void   // 统一 err instanceof Error -> message
export function confirm(content: string, title?: string): Promise<boolean>
export function showLoading(title?: string): void
export function hideLoading(): void

// utils/image.ts
export interface PickedImage { path: string; file?: File; size: number; name: string }
export function pickImages(count?: number): Promise<PickedImage[]>
/** 压缩到 1MB 内（H5 canvas→webp/jpeg；非 H5 uni.compressImage）；失败返回原图 */
export function compressImage(img: PickedImage): Promise<PickedImage>
export function sha256(img: PickedImage): Promise<string>          // 非 H5 返回 ''
export function revokePreview(path: string): void                  // H5 释放 blob URL

// utils/nav.ts
export const routes = { home: '/pages/home/index', login: '/pages/auth/login', register: '/pages/auth/register',
  transactions: '/pages/transactions/index', statistics: '/pages/statistics/index', settings: '/pages/settings/index',
  help: '/pages/help/index', merchantDetail: '/pages/merchant/detail' } as const
export function goHome(): void            // reLaunch
export function goLogin(replace?: boolean): void
export function goTab(key: 'home'|'transactions'|'statistics'|'settings'|'help'): void  // reLaunch
export function goBack(fallback?: string): void

// composables/useAuthGuard.ts
export function useAuthGuard(): void   // 内部用 onShow：无 token 或 无租户 -> reLaunch 登录
// composables/useRole.ts
export function useRole(): { role: ComputedRef<'owner'|'admin'|'member'>; canEdit: ComputedRef<boolean>; load(): Promise<void> }
```

**实现要点**

- `request.ts`：业务错误 `code !== 0` 一律 `throw new Error(message)`；HTTP 401 清 auth/tenant 并 `reLaunch` 登录；超时/网络错误统一文案；GET 参数直接放 `data`。
- Header 注入 `Authorization` + `X-Tenant-ID`（当前租户存在时）。
- `uploadFile` H5 分支用 `XMLHttpRequest` + `FormData`（带 token/tenant 头，支持 AbortController 超时），
  响应是字符串 → `JSON.parse` → `code` 判定。非 H5 用 `uni.uploadFile`。
- `excel.ts` 里响应为二进制：H5 用 `uni.downloadFile` 或 `fetch(API_BASE+...)` 取 blob 后 `a.download` 触发下载。
- `pickImages`：`uni.chooseImage({ count, sourceType: ['camera','album'], sizeType: ['original'] })`，
  H5 下从 `res.tempFiles[i]` 取 `File`（`uni.chooseImage` H5 返回的 tempFiles 含 File 实例）。

**验收**

- `npx vue-tsc --noEmit` 中本任务文件零报错。
- 每个 api 函数都是强类型（无 `any` 返回值）。
- 不出现 `window`/`document`（`#ifdef H5` 块内除外）。

---

## T02 · 认证页

**拥有文件**：`src/pages/auth/login.vue`、`src/pages/auth/register.vue`
**依赖**：T01

**要求**

- 视觉对齐现有：渐变背景 `linear-gradient(135deg,#eff6ff,#e0e7ff)`、白色卡片、`汤圆记账` 蓝色标题、`专业记账管理工具` 副标题。
- 登录：邮箱 + 密码 + 「记住密码」开关（`STORAGE_KEYS.remember`，明文存储，与现有一致）+ 提交 loading + 「还没有账号？立即注册」。
- 注册：用户名 / 邮箱 / 密码 / 确认密码 / 账本类型三选一卡片（🍽️ 饭店·餐厅经营记账 / 🏠 家庭·家庭收支管理 / 🚀 创业·初创企业财务）；校验两次密码一致、长度 ≥ 6。
- 成功后：`authStore.setAuth` → 重新 `tenantApi.list()`（失败回退登录响应里的 tenants）→ `tenantStore.setTenants` → `nav.goHome()`（`reLaunch`）。
- 错误：`toastError(err)`，文案与现有 `alert` 对齐。
- 已登录进入登录页时直接 `goHome()`（避免回退死循环）。

---

## T03 · 主框架组件

**拥有文件**

```
src/components/AppPage.vue  AppHeader.vue  AppTabBar.vue  AppModal.vue
src/components/AppToast.vue  EmptyState.vue  CategoryIcon.vue  AmountText.vue
src/components/TenantSwitcher.vue  CreditsSheet.vue
```

**契约**

```ts
// AppPage.vue：页面壳（自定义导航 + 可滚动内容 + 可选 TabBar）
props: { title?: string; back?: boolean; tabbar?: boolean; showCredits?: boolean; showTenant?: boolean; padded?: boolean }
slots: default / right（头部右侧）
// 内部 <scroll-view scroll-y> 高度 = 100vh - header - tabbar，内容底部预留 32rpx

// AppHeader.vue
props: { title: string; back?: boolean; showCredits?: boolean; showTenant?: boolean }
slots: right
// 高度 96rpx + 状态栏安全区；标题居中；back 用「‹」返回 goBack()

// AppTabBar.vue
props: { active: 'home'|'transactions'|'statistics'|'settings'|'help' }
// 5 个 tab：首页🏠 / 记账📒 / 统计📊 / 设置⚙️ / 帮助❓，激活态 $c-primary，fixed 底部 + 安全区
// 点击 goTab(key)（reLaunch）

// AppModal.vue
props: { visible: boolean; title?: string; position?: 'bottom'|'center'; closeOnMask?: boolean; showClose?: boolean }
emits: ['update:visible', 'close']
slots: default / footer
// bottom：圆角上沿、最大高度 85vh、内部滚动；打开时 body 不滚动（@touchmove.stop.prevent）

// AppToast.vue
props: { text: string; error?: boolean }   // 居中深色胶囊，error 时红色

// EmptyState.vue
props: { icon?: string; text?: string; hint?: string }  slots: default（操作按钮）

// CategoryIcon.vue
props: { icon?: string; name?: string; size?: 'sm'|'md'|'lg' }   // 圆形浅蓝底 + emoji

// AmountText.vue
props: { amount: number; type?: 'income'|'expense'; size?: 'sm'|'md'|'lg'; signed?: boolean }

// TenantSwitcher.vue
props: { compact?: boolean }
// 显示当前账本名（去掉「的记账本」后缀）+ ⌄，点击 uni.showActionSheet 切换；切换后 metaStore.reset()

// CreditsSheet.vue
props: { visible: boolean }  emits: ['update:visible','close']
// 顶部余额（∞ 表示不限，显示「无限制」+ 白名单提示）+ 明细列表（分页 10 条，收入/支出筛选）
// 数据来源 creditApi；颜色：+绿 -红；动作文案映射 upload=上传 ocr=OCR识别 llm=AI分析 gift=赠送 admin_adjust=管理员调整
```

**验收**：组件在 `pagess` 中可直接使用；无业务耦合（CreditsSheet 除外）；TabBar 在 5 个 tab 页复用同一组件。

---

## T04-1 · 记账核心状态机

**拥有文件**：`src/composables/useImageSlots.ts`
**依赖**：T01

**导出（不得改名）**

```ts
export type SlotStatus = 'pending' | 'compressing' | 'ocr_recognizing' | 'llm_recognizing' | 'done' | 'error'
export interface SlotForm { type: TxType; amount: string; categoryId: number; merchantId: number; merchantName: string; date: string; note: string }
export interface ImageSlot {
  id: string; previewUrl: string; status: SlotStatus; error?: string
  ocrId: number; imagePath: string; rawTexts: string[]; ocrAmount: number; ocrDate: string; ocrMerchant: string
  saved: boolean; form: SlotForm; llm: LLMSuggestion[]; llmError: string
}
export interface SlotDraft { id: string; type: TxType; amount: number; categoryId: number; categoryName: string; merchantId: number; merchantName: string; date: string; note: string; imagePath: string; llmIndex?: number }

export function defaultSlotForm(): SlotForm
export function useImageSlots(): {
  slots: Ref<ImageSlot[]>
  activeIndex: Ref<number>
  activeSlot: ComputedRef<ImageSlot | undefined>
  drafts: ComputedRef<SlotDraft[]>           // 当前激活槽位的草稿
  isRecognizing: ComputedRef<boolean>
  llmEnabled: Ref<boolean>
  totalDrafts: ComputedRef<number>           // 所有槽位草稿总数
  loadLlmConfig(): Promise<void>
  chooseAndAdd(): Promise<void>             // 选图 -> 压缩 -> sha256 -> 入队
  addImages(images: PickedImage[]): Promise<void>
  retry(slotId: string): void
  retryLlm(slotId: string): void
  removeSlot(slotId: string): void
  select(index: number): void
  patchForm(patch: Partial<SlotForm>): void
  applySuggestion(sug: LLMSuggestion, index: number): void
  addDraft(draft: SlotDraft): void
  removeDraft(index: number): void
  updateDraft(index: number, draft: SlotDraft): void
  setDrafts(list: SlotDraft[]): void
  markSaved(slotId: string): void
  reset(): void
}
```

**行为（必须与现有一致）**

1. 选图后**串行**处理（队列），避免并发打爆 OCR/LLM：`compressImage` → `sha256` → `ocrApi.upload`。
2. OCR 成功且 `ai_mode` 时用 `amount/date/merchant` 预填表单；日期转 `YYYY-MM-DDTHH:mm`（时间取 `extractTimeFromOcrDate`）。
3. LLM 已启用且 `ocr_id > 0` → `llmApi.analyzeByOcrId`；`suggestions.length > 1` → 自动生成草稿（`categoryId` 为 0 的分类名记「未匹配」，日期如无时间用 OCR 时间）。
4. `applySuggestion`：类型/金额/日期/商户/备注/分类；分类优先 `category_id`，否则 `category_hint` 在同类型分类里 `includes` 匹配；同时追加草稿（同 `llmIndex` 不重复）。
5. `removeSlot` 释放 preview（`revokePreview`）。
6. 错误：槽位 `status='error'` + `error` 文案（超时文案「识别超时，请重试」）；不打断队列。
7. 分类/商户通过 `useMetaStore().ensureCategories()/ensureMerchants()` 获取。
8. 组件卸载（`onUnmounted`）释放全部 preview URL。

---

## T04-2 · 记账展示组件

**拥有文件**

```
src/components/transaction/ImageSlotStrip.vue
src/components/transaction/SlotEditor.vue
src/components/transaction/LlmSuggestionList.vue
src/components/transaction/OcrTextPanel.vue
```
**依赖**：T03（AppToast/EmptyState 可选）、T04-1（类型）

**契约**

```ts
// ImageSlotStrip.vue —— 横向缩略图条 + 状态角标 + 删除 + 重试 + 末尾「+」
props: { slots: ImageSlot[]; activeIndex: number; max?: number }
emits: ['select', 'remove', 'retry', 'add']
// 角标：ocr_recognizing/compressing 白色转圈；llm_recognizing 紫色；done ✓；error 红色 + 点击重试；saved 打「已保存」标签

// SlotEditor.vue —— 单笔表单
props: { form: SlotForm; categories: Category[]; disabled?: boolean; readonly?: boolean }
emits: ['update:form']  // 或 patch 事件：['patch', Partial<SlotForm>]
// 字段：支出/收入 二段切换、金额（数字键盘，大号）、日期时间（picker mode=date + time）、分类九宫格（4 列，emoji + 名称）、商户（可输入 + 下拉建议）、备注

// LlmSuggestionList.vue
props: { suggestions: LLMSuggestion[]; error?: string; loading?: boolean }
emits: ['apply', LLMSuggestion, number]

// OcrTextPanel.vue
props: { rawTexts: string[]; usedLines?: number[] }
emits: ['fill', { action: 'amount'|'merchant'|'date'|'note'; text: string }]
// 长按 token 弹 actionSheet（uni.showActionSheet），选「填入金额/商户/日期/备注」
// 如果 token 全是数字则默认「填入金额」
```

**验收**：无网络请求；纯 props/emit；移动端 375 宽不溢出；分类网格 4 列；长按有震动反馈（`uni.vibrateShort`）。

---

## T04-3 · 记账表单装配

**拥有文件**

```
src/components/transaction/TransactionForm.vue
src/components/transaction/DraftList.vue
src/components/transaction/ReviewSheet.vue
```
**依赖**：T01、T03、T04-1、T04-2

**契约**

```ts
// TransactionForm.vue
props: { visible: boolean; mode?: 'create'|'edit'|'detail'; initialData?: Transaction }
emits: ['update:visible', 'success']
// create：多图槽位 + AI 建议 + 草稿列表 + 「完成」→ ReviewSheet 确认 → 逐笔 POST /transactions
// edit：单笔（initialData 填表）→ PUT /transactions/:id
// detail：只读展示（金额/类型/分类/时间/商户/备注/凭证图，点击 uni.previewImage）
```

**要求**

- `create` 模式：无图片时允许纯手输（`noSlotForm`）；有图片时表单绑定当前槽位。
- 生成 payload：`transaction_date = toApiDateTime(form.date)`；`image_path`、`ocr_amount/date/merchant/raw_texts` 从槽位 OCR 结果带出（`raw_texts` 用 `JSON.stringify`）。
- 校验：金额 > 0；分类必选；`require_expense_image && 支出 && !imagePath` → 提示「支出记录需要上传截图」。
- 提交前校验每条草稿的 `categoryId > 0`，否则提示「第 N 笔未匹配有效分类，请编辑」。
- 防重复提交；成功后 `emit('success')` 并关闭；`edit` 模式保存后关闭。
- 保存成功后把对应槽位标记 `saved`，自动切到下一张未保存槽位。
- 详情模式：只读，底部只有一个「关闭」。
- 提交状态用 `AppToast` 内联提示（错误 3.5s 自动消失）。

---

## T05 · 统计页 + 图表组件

**拥有文件**

```
src/components/charts/ChartLine.vue  ChartBar.vue  ChartPie.vue
src/pages/statistics/index.vue
```
**依赖**：T01、T03

**契约**

```ts
props: {
  categories: string[]
  series: { name: string; data: number[]; color?: string }[]
  height?: number     // px，默认 220
}
// ChartLine / ChartBar

props: { items: { name: string; value: number; color?: string }[]; height?: number }
// ChartPie：环形图 + 中心总额 + 外部图例
```

**要求**

- 用 `@qiun/ucharts`（`new uCharts({ type: 'line'|'column'|'ring', context: uni.createCanvasContext(id, instance), width, height, categories, series, ... })`），
  高 DPI 下按 `uni.getSystemInfoSync().pixelRatio` 缩放；数据变化 `watch` 重新渲染。
- 渲染失败兜底：捕获异常 → 显示「图表暂不可用」+ 简单数据列表（不允许白屏）。
- 统计页：三视图切换（月/年/区间）、月份/年份左右切换（不允许超过当前）、区间选择 `picker mode="date"`。
- 结构顺序：导航 → 视图切换 → 期间选择 → 三总额卡（收入/支出/净结余）→ 柱状图（收入/支出）→ 分类占比（环形图 + 列表）→ 商户 Top10（含「刷新」按钮，点击 `refreshMerchants`）。
- 点击商户 → `uni.navigateTo('/pages/merchant/detail?id=&name=&start=&end=')`（区间按当前视图计算）。
- 加载/空/错误态齐全；切换视图时保留上次数据避免闪烁。

---

## T06 · 设置页（6 个 tab）

**拥有文件**

```
src/pages/settings/index.vue
src/components/settings/GeneralTab.vue  CategoriesTab.vue  MembersTab.vue
src/components/settings/RolesTab.vue    LlmTab.vue  PromptSheet.vue  CreditsTab.vue
```
**依赖**：T01、T03

**要求**

- 顶部：账本名称编辑（输入前缀 + 固定后缀「的记账本」实时预览，保存 `tenantApi.update` + `tenantStore.updateTenantName`）。
- Tab 切换用横向滚动 tab 条（通用/成员/角色/分类/AI/积分），支持 `?tab=` 参数定位（`onLoad` 读取）。
- 通用：`require_expense_image` 开关（`tenantApi.getSettings/updateSettings`）。
- 成员：列表（用户名/邮箱/角色标签/「我」/「所有者」）、邀请（用户名 + 角色下拉，`addMember`）、改角色、移除（`confirm`）。
- 角色：权限矩阵（资源 × 动作 checkbox，来自 `rbacApi.listPermissions`）、系统角色不可删、创建/编辑/删除。
- 分类：新增（emoji 选择器 + 名称 + 收支类型）、列表分组（支出/收入）、编辑、删除（`confirm`）。
- AI：启用开关；来源（平台提供 / 自定义 Key 二选一，平台未配置时禁用并提示）；自定义时 provider/base_url/api_key/model/mode；保存后 toast；「提示词设置」打开 `PromptSheet`（`getPrompt` 展示占位符说明 + 编辑自定义提示词 + 重置为默认）。
- 积分：余额卡（无限制提示）、收入/支出筛选、日志分页（10 条，上拉加载更多）。
- 权限不足（admin 以下）时非积分 tab 显示「权限不足」空态。

---

## T07 · 帮助页

**拥有文件**：`src/pages/help/index.vue`
**依赖**：T03

**要求**：与现有 `pages/Help/index.tsx` 内容一致 —— iOS Safari「添加到主屏幕」4 步图文示意、Android Chrome 3 步示意、「提示」区块、常见问题 FAQ（可折叠）。加入 `AppTabBar active="help"`。

---

## T08 · 首页

**拥有文件**：`src/pages/home/index.vue`
**依赖**：T01、T03、T04-3

**要求**

- 顶部 `AppHeader`（汤圆记账 + 积分 + 账本切换）+ 底部 TabBar（active=home）。
- 今日概览：今日收入 / 今日支出 / 净结余三张卡（loading 骨架），数据 `statisticsApi.daily(today)`。
- 月趋势：`ChartLine`，数据 `statisticsApi.monthly(now)`，右上角显示本月收入/支出合计；无数据空态。
- 「记一笔」按钮打开 `TransactionForm`（create 模式），成功后刷新今日 + 本月。
- `onShow` 重新拉数据（从其它 tab 返回时保持新鲜）。

---

## T09 · 交易列表

**拥有文件**

```
src/components/transactions/TransactionCard.vue
src/components/transactions/FilterPanel.vue
src/components/transactions/Pagination.vue      // 上拉加载 + 共 N 条
src/components/transactions/ImportSheet.vue
src/pages/transactions/index.vue
```
**依赖**：T01、T03、T04-3

**要求**

- 顶部标题 + 「更多」菜单（下载模板 / 导入 / 导出，仅 canEdit）。
- 筛选：交易时间/创建时间 二段切换（影响日期筛选字段）、类型、分类、商户（本地模糊搜索下拉，最多 20 条）、起止日期；「查询」「重置」。
- 列表：`TransactionCard`（分类 emoji、分类名、金额、交易时间、备注、创建时间）；`onReachBottom` 加载下一页；`共 N 条`。
- 长按进入多选模式（顶部「全选本页」、底部固定操作条「已选 N 项 / 批量删除」，`confirm` 后 `batchRemove`）。
- 点击卡片 → 详情（`TransactionForm` detail）；编辑按钮 → edit 模式；删除按钮 → `confirm`。
- 浮动「+」按钮 → 新增（create 模式），成功后刷新第一页。
- 导入：`excelApi.chooseExcelFile` → `ImportSheet` 三步（列映射 / 预检 / 结果），映射字段：日期/类型/金额/分类（必填），商户/备注（可选），支持多 sheet 切换。
- 导出：`excelApi.exportExcel({ start_date, end_date, type })`（取当前筛选），H5 下载文件名 `${账本名}-YYYY-MM-DD-HH-mm.xlsx`。

---

## T10 · 商户明细页

**拥有文件**：`src/pages/merchant/detail.vue`
**依赖**：T01、T03、T09（列表卡片样式）

**要求**：`onLoad` 读取 `id/name/start/end`；标题「{商户名}」+ 区间副标题；列表 `transactionApi.list({ merchant_id, start_date, end_date, page, page_size:20 })`；上拉加载更多；金额合计（前端汇总当前已加载数据并标注「已加载 N 笔」）；点击项打开只读详情；空态/加载态。

---

## T11 · 构建接入与验收（架构负责人执行）

- `npm run type-check` 零错误、`npm run build:h5` 成功。
- 用 Playwright/浏览器跑通回归基线（见 `ARCHITECTURE.md §12`）。
- 增加 Makefile 目标 `make uniapp:dev` / `make uniapp:build`（**不改动现网发布流程**）。
- 输出 `uniapp/README.md`：开发/构建/发布切换步骤。
