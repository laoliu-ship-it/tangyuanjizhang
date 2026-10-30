# 汤圆记账 · 用户端 uni-app 架构设计

> 目标：把现有 `web/`（React + Vite + Tailwind，约 8300 行）中的**用户端 H5** 用 uni-app（Vue 3）重写，
> 保证功能等价、视觉一致，并为后续小程序/App 留出扩展位。
> 平台管理端（`web/src/platform/**`）**不迁移**，继续留在 React 工程里。

---

## 1. 范围

### 1.1 迁移（P0）

| 页面 | 现有文件 | uni-app 页面 |
|---|---|---|
| 登录 | `pages/Auth/Login.tsx` | `pages/auth/login` |
| 注册 | `pages/Auth/Register.tsx` | `pages/auth/register` |
| 首页（今日概览 + 月趋势 + 记一笔） | `pages/Dashboard/index.tsx` | `pages/home/index` |
| 交易记录（筛选/分页/批量删除/导入/导出） | `pages/Transactions/index.tsx` | `pages/transactions/index` |
| 记账表单（多图 OCR + AI 草稿 + 批量提交） | `components/TransactionForm.tsx`（2238 行） | `components/transaction/**` |
| 统计分析（月/年/区间 + 分类占比 + 商户 Top） | `pages/Statistics/index.tsx` | `pages/statistics/index` |
| 商户钻取明细 | `pages/MerchantTransactions/index.tsx` | `pages/merchant/detail` |
| 账本设置（通用/成员/角色/分类/AI/积分） | `pages/TenantSettings/**` | `pages/settings/index` + `components/settings/**` |
| 帮助中心 | `pages/Help/index.tsx` | `pages/help/index` |

### 1.2 不迁移

- 平台管理端全部页面（`/platform/*`）。
- `pages/UploadOCR`（独立拍照页）：现有导航里**没有任何入口**，真实识别链路在 `TransactionForm` 内。
  作为 P2 可选页面保留占位（`pages/upload/index`），不投入实现。

### 1.3 有意简化（PC 专属交互）

| 现有能力 | 处理方式 |
|---|---|
| 拖拽上传图片 | 去掉（移动端用 `uni.chooseImage` 拍照/相册） |
| 粘贴板图片 | 去掉 |
| OCR 文本右键「填入字段」 | 改为**长按 token** 弹出同款菜单 |
| 图片 lightbox 双击缩放 | 改为点击进入 `uni.previewImage` 全屏预览 |
| PC 表格布局 | 全部改为移动端卡片布局（桌面端居中限宽） |
| Excel 导入列映射向导 | 保留（H5 端 `uni.chooseFile`，非 H5 端降级为提示「请用电脑导入」） |

---

## 2. 技术选型

| 领域 | 选择 | 版本/说明 |
|---|---|---|
| 框架 | uni-app + Vue 3 + TypeScript | CLI 模式（vite），非 HbuilderX 可视化工程 |
| 构建 | Vite | **必须锁 5.2.8**（`@dcloudio/vite-plugin-uni` peer 强制） |
| Vue | 3.4.21 | 与 uni-app 内部 `@vue/shared@3.4.21` 对齐，勿升级 |
| 状态 | Pinia 2.3.1 + `uni.setStorageSync` 手动持久化 | 不引入 pinia 持久化插件，存储层自己封装 |
| 请求 | `uni.request` 自研封装（**不用 axios**） | uni 生态下 axios 在非 H5 端不可用 |
| 图表 | `@qiun/ucharts`（纯 JS） + 自写 canvas 包装组件 | 不引入 echarts（体积大、MP 端麻烦） |
| 日期 | dayjs | 与现有实现一致 |
| 样式 | SCSS + rpx + 设计令牌 | 不使用 Tailwind（uni-app 的 class 作用域/编译链不适配） |
| 图标 | emoji（分类图标）+ 内联 SVG（导航/操作） | 不引入图标字体，避免字体加载闪动 |

### 2.1 已知风险与兜底

1. **uCharts 在 H5 的 canvas 兼容性**：`uni.createCanvasContext(canvasId, instance)` 在 H5 可用；
   若某端异常，`ChartBar`/`ChartLine` 降级为纯 CSS 柱条 + 「数据列表」text 展示（组件内 `renderError` 兜底）。
2. **图片压缩**：H5 用 `document.createElement('canvas')`；App/小程序用 `uni.compressImage`。
   **必须条件编译**，否则小程序端编译报错。
3. **SHA-256 去重**：仅 H5 用 `crypto.subtle`；非 H5 直接跳过 `original_hash`（后端该字段可选）。
4. **`@qiun/ucharts` 版本**：`2.5.0-20230101`，API 与 2.x 文档一致。

---

## 3. 目录结构

```
uniapp/
├── package.json                 # 脚本：dev:h5 / build:h5 / type-check
├── vite.config.ts               # /api /uploads 代理，@ -> src 别名
├── tsconfig.json
├── index.html
├── ARCHITECTURE.md              # 本文档
├── TASKS.md                     # 子 agent 任务卡（文件所有权 + 验收标准）
└── src/
    ├── main.ts                  # createSSRApp + pinia
    ├── App.vue                  # onLaunch：恢复登录态 / 全局错误捕获
    ├── pages.json               # 全部页面一次性注册（子 agent 禁止修改）
    ├── manifest.json
    ├── uni.scss                 # 设计令牌（颜色/字号/圆角/间距），全局注入
    ├── env.d.ts
    ├── config.ts                # API_BASE / UPLOAD_BASE / 常量
    ├── styles/
    │   ├── common.scss          # 原子类：card / row / btn / tag / safe-area
    │   └── icon.scss            # 内联 SVG 背景类（可选）
    ├── api/
    │   ├── request.ts           # uni.request 封装（token/租户/401/业务错误）
    │   ├── types.ts             # 全部 DTO 类型（对齐后端 dto）
    │   ├── auth.ts  tenant.ts  category.ts  merchant.ts  transaction.ts
    │   ├── statistics.ts  llm.ts  ocr.ts  credit.ts  rbac.ts
    │   └── excel.ts             # 导入/导出（H5 下载 + 文件选择）
    ├── store/
    │   ├── auth.ts              # token/userId/username + 登录态持久化
    │   ├── tenant.ts            # 租户列表 / currentTenantId
    │   └── meta.ts              # 分类 / 商户 / 当前角色（跨页缓存）
    ├── utils/
    │   ├── storage.ts           # uni storage 读写 + JSON 安全解析
    │   ├── image.ts             # 压缩（条件编译）+ SHA256（H5）
    │   ├── datetime.ts          # "YYYY-MM-DDTHH:mm" <-> "YYYY-MM-DD HH:mm:00"
    │   ├── ui.ts                # toast / confirm / loading 封装（统一 Promise 风格）
    │   └── download.ts          # 二进制下载（H5）/ 保存提示（非 H5）
    ├── composables/
    │   ├── useAuthGuard.ts      # 页面级登录守卫
    │   ├── useRole.ts           # owner/admin/member 权限判断
    │   └── useImageSlots.ts     # OCR 队列 + 槽位状态机（记账表单核心）
    ├── components/
    │   ├── AppPage.vue          # 页面壳：自定义导航 + 安全区 + 滚动区
    │   ├── AppHeader.vue        # 顶部栏（标题 / 返回 / 积分 / 租户切换）
    │   ├── AppTabBar.vue        # 自定义底部 5 tab
    │   ├── AppModal.vue         # 底部抽屉/居中弹窗（统一动画）
    │   ├── AppToast.vue         # 轻提示
    │   ├── EmptyState.vue       # 空态
    │   ├── CategoryIcon.vue     # 分类图标（emoji 圆形底）
    │   ├── AmountText.vue       # 金额格式化（+¥ / -¥ / 颜色）
    │   ├── TenantSwitcher.vue   # 账本切换（actionSheet）
    │   ├── CreditsSheet.vue     # 积分明细抽屉
    │   ├── charts/ChartLine.vue  ChartBar.vue  ChartPie.vue
    │   ├── transaction/
    │   │   ├── TransactionForm.vue     # 外壳 + 模式（create/edit/detail）
    │   │   ├── ImageSlotStrip.vue      # 多图缩略图条 + 状态角标
    │   │   ├── SlotEditor.vue          # 单图表单（类型/金额/日期/分类/商户/备注）
    │   │   ├── LlmSuggestionList.vue   # AI 建议卡片 + 「填入」
    │   │   ├── OcrTextPanel.vue        # 识别原文 + 长按填入
    │   │   ├── DraftList.vue           # 草稿列表（编辑/删除）
    │   │   └── ReviewSheet.vue         # 提交前确认
    │   ├── transactions/
    │   │   ├── FilterPanel.vue   TransactionCard.vue   Pagination.vue
    │   │   └── ImportSheet.vue   # Excel 导入三步（映射/预检/结果）
    │   └── settings/
    │       ├── GeneralTab.vue  MembersTab.vue  RolesTab.vue  CategoriesTab.vue
    │       ├── LlmTab.vue      PromptSheet.vue CreditsTab.vue
    └── pages/
        ├── auth/login.vue  auth/register.vue
        ├── home/index.vue
        ├── transactions/index.vue
        ├── statistics/index.vue
        ├── merchant/detail.vue
        ├── settings/index.vue
        ├── help/index.vue
        └── upload/index.vue        # P2 占位
```

### 3.1 分层契约（单向依赖，禁止反向）

```
pages/*            —— 只做「页面编排 + 状态」，禁止直接 uni.request
  ↓
components/*       —— 纯展示 + 局部交互，通过 props/emit，可直接用 api/store
  ↓
composables/*      —— 可复用业务逻辑（无视图）
  ↓
api/*.ts           —— 类型化接口函数
  ↓
api/request.ts     —— 唯一网络出口
```

硬性约束：

- **所有网络请求必须经过 `api/request.ts`**，页面里出现 `uni.request` 视为不合规。
- **所有金额展示必须走 `AmountText` 或 `utils/format`**，禁止页面 `/100`、`toFixed` 各写一份。
- **页面禁止直接 `uni.setStorageSync`**，一律走 `store/*`。
- **不同模块不允许互相 import 对方的页面/组件**（跨模块复用必须下沉到公共组件）。

---

## 4. 路由与页面表

`pages.json` 由架构阶段一次性写定，子 agent 不修改；新增页面需走架构变更。

| # | path | 说明 | 导航 |
|---|---|---|---|
| 1 | `pages/home/index` | 首页（启动页） | Tab 1 |
| 2 | `pages/auth/login` | 登录 | `reLaunch` |
| 3 | `pages/auth/register` | 注册 | `navigateTo` |
| 4 | `pages/transactions/index` | 交易记录 | Tab 2 |
| 5 | `pages/statistics/index` | 统计 | Tab 3 |
| 6 | `pages/merchant/detail` | 商户明细（参数：`id,name,start,end`） | `navigateTo` |
| 7 | `pages/settings/index` | 账本设置（参数：`tab`） | Tab 4 |
| 8 | `pages/help/index` | 帮助 | Tab 5 |
| 9 | `pages/upload/index` | 拍照识别（P2 占位） | 无入口 |

全局样式：

```jsonc
"globalStyle": {
  "navigationStyle": "custom",       // 统一用 AppHeader，避免双导航栏
  "backgroundColor": "#f9fafb",
  "rpxCalcMaxDeviceWidth": 960,      // 桌面端 rpx 不再无限放大
  "rpxCalcBaseDeviceWidth": 375,
  "rpxCalcIncludeWidth": 750
}
```

H5 路由：`history` 模式，`base: "/"`；Go 侧 `NoRoute` 已回退 `index.html`，无需改后端。
页面具体路径形如 `/pages/home/index`，**对外跳转一律用封装**（见 `utils/nav.ts` 约定：`goHome()` 等），
禁止在业务代码里拼 `/pages/xxx` 字符串（后续若改 hash 模式可一处切换）。

### 4.1 登录守卫

- `store/auth` 持久化 token；`App.vue` `onLaunch` 读取缓存（同步）。
- 每个受保护页面在 `onShow` 调用 `useAuthGuard()`：无 token → `uni.reLaunch('/pages/auth/login')`。
- 有 token 但无 `currentTenantId`（历史脏数据）→ 拉取租户列表，为空则回到登录页。
- 401（HTTP 层）→ 清空 auth + tenant，`reLaunch` 登录页（在 `request.ts` 统一处理，页面不用管）。

---

## 5. 状态管理

### 5.1 `store/auth.ts`

```
state: token: string | null; userId: number | null; username: string | null
actions: setAuth(token, userId, username) / logout()
持久化 key: 'ty_auth'（uni storage，JSON）
```

### 5.2 `store/tenant.ts`

```
state: tenants: Tenant[]; currentTenantId: number | null
actions: setTenants(list)   // 默认选中第一个（与现有一致）
         switchTenant(id)   // 切换后必须 meta.reset()
         reset()
持久化 key: 'ty_tenant'
```

### 5.3 `store/meta.ts`（新增，替代现有每页各自拉取）

```
state: categories: Category[]; merchants: Merchant[]; role: 'owner'|'admin'|'member'
       loadedAt: number
actions: ensureCategories(force?) / ensureMerchants(force?) / setRole / reset
```
缓存 TTL 5 分钟，`currentTenantId` 变化时 `reset()`。交易表单、筛选、设置页统一从这里取，避免重复请求。

---

## 6. 请求层规范（`api/request.ts`）

```ts
type ApiResponse<T> = { code: number; data: T; message?: string }

request<T>(options): Promise<T>   // 直接返回 data，业务/网络错误一律 throw Error(message)
```

- Header：`Authorization: Bearer <token>`、`X-Tenant-ID: <currentTenantId>`（有则带）。
- `API_BASE = import.meta.env.VITE_API_BASE || '/api'`；H5 dev 由 vite 代理到 `API_TARGET`。
- **业务错误**：后端统一 HTTP 200 + `code !== 0` → `throw new Error(res.data.message || '请求失败')`。
- **HTTP 401**：清登录态 → `reLaunch` 登录页 → `throw new Error('登录已过期')`。
- **HTTP 5xx / 网络错误**：`throw new Error('网络异常，请稍后重试')`，并把原始错误打到 `console.error`。
- **超时**：默认 20s；OCR 上传单独 25s（对齐现有 `OCR_TIMEOUT`）。
- 上传：`uni.uploadFile({ url, filePath, name: 'file', formData: { original_hash } })`，
  返回体是字符串，必须 `JSON.parse` 后再按 `code` 判定。
- 下载（导出 Excel / 模板）：H5 用 `uni.downloadFile` 会走 blob，统一在 `api/excel.ts` 内做条件编译，
  非 H5 端 `uni.showToast('请在电脑端导出')`。

### 6.1 图片地址

后端 `image_path` 形如 `/uploads/xxx.webp`，由 Go 根路径静态服务。
`config.ts` 导出 `resolveUpload(path)`：H5 下为空前缀（同源），App/MP 下拼 `VITE_UPLOAD_BASE`。

---

## 7. 跨端适配清单

| 能力 | H5 | 非 H5 | 实现位置 |
|---|---|---|---|
| 选图 | `uni.chooseImage({count:9, sourceType:['camera','album']})` | 同 | `utils/image.ts#pickImages` |
| 压缩 | canvas + WebP(降级 JPEG) | `uni.compressImage` | `utils/image.ts#compress` |
| 去重 hash | `crypto.subtle.digest('SHA-256')` | 跳过 | `utils/image.ts#sha256` |
| 上传 | `uni.uploadFile` | 同 | `api/ocr.ts` |
| 下载文件 | `downloadFile` + `a.download` | 提示用电脑 | `utils/download.ts` |
| 文件选择（Excel） | `<input type=file>` 动态创建 | 提示用电脑 | `api/excel.ts` |
| 全屏看图 | `uni.previewImage` | 同 | 组件内 |
| 剪贴板 | 不需要 | — | — |
| Canvas 图表 | `uni.createCanvasContext` | 同 | `components/charts/*` |

**条件编译**统一用：

```ts
// #ifdef H5
...
// #endif
```
禁止在公共逻辑里裸用 `window` / `document`（除 `#ifdef H5` 块内）。

---

## 8. UI 规范（设计令牌）

`uni.scss` 全局注入：

```scss
$c-primary: #2563eb;      $c-primary-bg: #eff6ff;
$c-bg: #f9fafb;           $c-card: #ffffff;
$c-text: #1f2937;         $c-text-2: #6b7280;   $c-text-3: #9ca3af;
$c-income: #16a34a;       $c-income-bg: #f0fdf4;
$c-expense: #dc2626;      $c-expense-bg: #fef2f2;
$c-border: #f3f4f6;       $c-border-2: #e5e7eb;
$radius-sm: 12rpx; $radius: 24rpx; $radius-lg: 32rpx;
$fs-xs: 22rpx; $fs-sm: 26rpx; $fs: 28rpx; $fs-lg: 32rpx; $fs-xl: 40rpx;
```

通用类（`styles/common.scss`，全局引入，**不需要 scoped**）：

```
.page        页面容器（min-height:100vh; background:$c-bg; padding:24rpx 24rpx calc(160rpx + env(safe-area-inset-bottom))）
.card        白底 + $radius + 细边框 + 轻阴影
.row         横向居中对齐，space-between 用 .row-between
.btn / .btn-primary / .btn-ghost / .btn-danger / .btn-block
.tag / .tag-income / .tag-expense
.input        输入框统一样式（36rpx 高 72rpx，圆角 $radius-sm）
.divider
.text-2 / .text-3 / .num（等宽数字 font-variant-numeric: tabular-nums）
```

规则：

- 所有可点区域 ≥ 88rpx × 88rpx。
- 底部固定元素必须加 `padding-bottom: env(safe-area-inset-bottom)`。
- 金额统一 `¥1,234.56`（`utils/format.ts#money`，千分位 + 两位小数，`null` 兜底 `0.00`）。
- 空态统一 `<EmptyState icon text />`。
- 弹层统一 `<AppModal>`：移动端从底部升起（`position: fixed` + `transition`），`@touchmove.stop.prevent` 防穿透。

---

## 9. 关键业务流程

### 9.1 记账表单（最高复杂度，拆 3 个任务）

状态机（单张图片槽位）：

```
pending → compressing(可选) → ocr_recognizing → llm_recognizing(可选) → done
                                          ↘ error（可重试，重试回到 ocr_recognizing）
```

数据流：

1. 选图 → `pickImages` → 并行 `sha256`（H5）→ 顺序 `compress`（超 1MB 降质量重试，质量下限 0.3，用户确认后继续）。
2. 每张图创建 `ImageSlot { id, file, previewUrl, status, ocrResult, ocrId, imagePath, saved, form, originalHash }`。
3. **串行队列** `processQueue`（避免并发打爆 OCR/LLM 限流）：
   - `POST /upload/ocr`（multipart + `original_hash`）→ 拿 `ocr_id / image_path / amount / date / merchant_* / raw_texts`。
   - 若 LLM 已启用且 `ocr_id > 0`：`POST /llm/analyze { ocr_id }` → `suggestions[]`。
   - `suggestions.length > 1` → 自动生成草稿（每笔一个 `Draft`），提示「AI 识别出 N 笔」。
4. 表单编辑绑定「当前激活槽位」；无图片时用 `noSlotForm`（纯手输记账）。
5. 提交：
   - `edit` 模式：单笔 `PUT /transactions/:id`。
   - `create` 模式：先攒草稿（本地），点「完成」进确认弹窗 → 逐笔 `POST /transactions`（串行，不用 batch 接口，与现有行为一致）。
   - 提交前校验：金额 > 0、分类必选、`requireExpenseImage && type==='expense' && !imagePath` → 拦截。
   - 防重复提交：`submittingRef` 等价的 `isSubmitting` 标志。
6. 图片预览 URL 释放：H5 下槽位移除/页面卸载时 `URL.revokeObjectURL`（封装在 `utils/image.ts#revokePreview`）。

> 迁移时必须保留的细节（容易漏）：
> - OCR 日期 `"2024-05-20 14:30"` 提取 `HH:mm` 作为交易时间；无时间则用当前时间。
> - LLM 日期含时间（`/\\d{2}:\\d{2}$/`）时直接拆用，否则 `LLM 日期 + OCR 时间`。
> - 分类匹配：优先 `llm.category_id`，否则按 `category_hint` 在**同类型**分类里 `name.includes` 模糊匹配。
> - 草稿缺有效分类时禁止提交并高亮该行。
> - 已保存的图片槽位打「已保存」标记，自动跳到下一张未保存的。

### 9.2 权限

- `role` 由 `GET /tenants/:id/members` 里 `user_id === 自己` 得到，缓存进 `store/meta`。
- `canEdit = role === 'owner' || role === 'admin'`；`member` 只读（隐藏新增/编辑/删除/批量操作、导入导出）。
- 设置页各 tab 的可见性：`members/roles/llm/categories` 需 admin+；`credits` 所有人可见。以前端隐藏为主，后端为准。

### 9.3 统计

- 三个视图（月/年/区间）共用一套「总额卡片 + 柱状图 + 分类饼图 + 商户 Top10」。
- `top_merchants` 支持手动刷新：`POST /statistics/merchants/refresh`。
- 点击商户：`navigateTo('/pages/merchant/detail?id=&name=&start=&end=')`（区间随当前视图计算）。
- 图表数据组装与原实现一致（月 → 按日；年 → 按月；区间 → 按日）。

### 9.4 交易列表

- 筛选：类型 / 分类 / 商户（本地模糊搜索下拉）/ 起止日期 / 排序口径（交易时间 vs 创建时间，影响筛选字段语义）。
- 分页：上拉加载更多（`onReachBottom`）替代页码，**同时保留「每页 N 条」不可见**；总数展示 `共 N 条`。
- 批量选择：长按进入多选模式（移动端惯例），底部出现操作条。
- 导入/导出：仅在 `canEdit` 时出现「更多」菜单。

### 9.5 积分 / 租户切换

- 顶部 `AppHeader` 右侧：积分徽标（点击开 `CreditsSheet`）+ 租户名（点击 `actionSheet` 切换）。
- 切换租户后：`meta.reset()` → 当前 tab 页 `onShow` 重新拉数据。

---

## 10. 接口清单（与 `web/src/services/api.ts` 一一对应）

| 模块 | 方法 & 路径 |
|---|---|
| 认证 | `POST /auth/register` `{username,email,password,book_type}` · `POST /auth/login` `{email,password}` |
| 租户 | `GET/POST /tenants` · `PUT /tenants/:id` · `GET/POST /tenants/:id/members` · `PUT/DELETE /tenants/:id/members/:uid` · `GET/PUT /tenants/:id/settings` |
| 分类 | `GET/POST /categories` · `PUT/DELETE /categories/:id` |
| 商户 | `GET/POST /merchants` · `PUT/DELETE /merchants/:id` |
| 交易 | `GET /transactions`（page,page_size,type,category_id,merchant_id,start_date,end_date,sort_by）· `POST /transactions` · `PUT/DELETE /transactions/:id` · `DELETE /transactions/batch` `{ids}` |
| 统计 | `GET /statistics/daily?date=` · `/monthly?year=&month=` · `/yearly?year=` · `/range?start=&end=` · `POST /statistics/merchants/refresh` |
| OCR | `POST /upload/ocr`（multipart: `file`, `original_hash`） |
| LLM | `GET/PUT /llm/config` · `GET/PUT /llm/prompt` · `POST /llm/analyze` `{ocr_id}` |
| 积分 | `GET /credits` · `GET /credits/logs?page=&size=&type=` |
| RBAC | `GET /permissions` · `GET/POST /tenants/:id/roles` · `PUT/DELETE /tenants/:id/roles/:rid` |
| Excel | `GET /export/excel` · `GET /import/template` · `POST /import/parse-headers` · `POST /import/excel` |

DTO 全部集中在 `api/types.ts`，字段名与后端 **snake_case** 保持一致（不做驼峰转换，减少映射出错）。

---

## 11. 任务拆分（每张卡 = 一个子 agent）

依赖图：

```
T01 基础设施 ──┬── T02 认证页
               ├── T03 框架组件（Header/TabBar/Modal/Toast/CreditsSheet/TenantSwitcher）
               ├── T04 记账表单（拆 T04a/T04b/T04c）
               ├── T05 统计 + 图表
               ├── T06 设置页（拆 T06a/T06b）
               └── T07 帮助
T03 + T04 ──── T08 首页
T03 + T04 ──── T09 交易列表
T09 ────────── T10 商户明细
全部 ───────── T11 构建接入 + 联调验收
```

详细文件所有权与验收标准见 `TASKS.md`。

---

## 12. 联调与部署

| 项 | 说明 |
|---|---|
| 开发 | `cd uniapp && npm run dev:h5`（默认 5275，代理 `/api` `/uploads`） |
| 环境切换 | `API_TARGET=http://192.168.0.25:8090 npm run dev:h5`；生产同源无需配置 |
| 构建 | `npm run build:h5` → `uniapp/dist/build/h5` |
| 切换发布 | 产物替换 `web/dist` 再走现有 `make prod`（T11 提供 `make uniapp` 目标，默认**不**改动现网流程） |
| 回归基线 | 登录→首页今日数据→多图记账→AI 草稿提交→列表筛选→统计三视图→商户钻取→设置改分类→积分明细 |

---

## 13. 质量门禁

- `npm run type-check`（vue-tsc）零错误。
- `npm run build:h5` 成功，产物首屏 JS < 800KB（gzip 后 < 300KB）。
- 页面内无 `uni.request` / `localStorage` / `window`（除 `#ifdef H5`）。
- 每个页面至少覆盖：加载态、空态、错误态。
- 金额/日期格式化必须复用 `utils/format.ts`、`utils/datetime.ts`。
