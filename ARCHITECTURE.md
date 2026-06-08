# 饭店记账 — 架构文档

## 1. 系统概览

多租户餐厅记账应用，支持 OCR 识别凭证、RBAC 权限控制、PC/移动端自适应。

```
┌─────────────────┐    HTTP     ┌──────────────┐   MySQL   ┌──────────────┐
│  React 前端      │ ──────────► │  Go 后端      │ ────────► │  MySQL 5.7   │
│  (Vite + TS)    │             │  (Gin/GORM)  │           │  (本地运行)   │
└─────────────────┘             └──────┬───────┘           └──────────────┘
                                       │ HTTP
                                       ▼
                                ┌──────────────┐
                                │ RapidOCR 服务 │
                                │ (Python/ONNX) │
                                └──────────────┘
```

**部署方式**：
- **本地/NAS**：Docker Compose 管理两个容器（Go 后端 + OCR 服务），MySQL 运行在宿主机，后端通过 `host.docker.internal` 访问（Linux 需配置 `extra_hosts: host-gateway`）
- **生产服务器（ddd.com）**：原生 systemd 部署（`make prod` → `scripts/deploy-server.sh`），Go 二进制直接运行，环境变量通过 `/www/wwwroot/.../.env` 文件管理，OCR 服务另行部署；deploy 脚本负责同步必要的 `.env` 配置

---

## 2. 多租户模型

采用**共享数据库、独立数据行**模式，所有业务表含 `tenant_id` 列。

- 用户注册时自动创建默认租户，同时成为该租户的 `admin`
- 创建租户时自动生成 16 个餐厅场景默认分类（在 `default_categories.go` 中维护）
- 租户名格式固定后缀：`{用户输入}的记账本`（前端强制，后缀 4 字不可修改）
- API 请求通过 `X-Tenant-ID` Header 传递当前租户，`Tenant` 中间件验证成员身份并注入 context

---

## 3. 后端分层

```
server/
├── cmd/main.go               # 依赖注入入口，路由注册
├── config/config.go          # 从环境变量加载，含 OCR.AIMode 开关
├── internal/
│   ├── model/                # GORM 结构体（5 张表 + transaction_images）
│   ├── dto/                  # 请求/响应结构体，与 model 分离
│   ├── repo/                 # interface.go（接口定义）+ impl.go（实现）
│   ├── service/              # 业务逻辑，不含 HTTP 细节
│   ├── handler/              # 绑定路由，调用 service，统一返回格式
│   └── middleware/           # JWT → Tenant 验证 → Casbin 鉴权（顺序不可变）
└── pkg/
    ├── ocr/                  # Engine 接口 + RapidOCR 实现 + Cloud 桩 + factory
    ├── excel/                # 导出 + 导入 + 模板生成
    └── storage/              # 本地磁盘存储，路径格式 uploads/{tenantID}/{date}/
```

**响应格式统一**：`{"code": 0, "message": "ok", "data": ...}`，错误时 code 非 0。前端 `api.ts` 响应拦截器检测 `code !== 0` 并自动 `Promise.reject`，所有业务错误（包括分析超时等）均通过此路径统一弹 Toast，handler 层不得用 `dto.OK` 包裹错误信息绕过此机制。

---

## 4. OCR 架构

### 微服务化原因
PaddleOCR 完整运行需 1.5GB+，2GB 服务器无法与 Go 后端共存，改用 RapidOCR（ONNX 轻量版），独立 Python 微服务，内存约 280MB。

### 可插拔引擎
```go
type Engine interface {
    Recognize(ctx context.Context, imagePath string) (*Result, error)
}
// factory.go 根据 config.OCR.Engine 创建：rapidocr / cloud（桩）
```

### AI 模式开关（`OCR_AI_MODE` 环境变量）
- `false`（默认）：只返回 `raw_texts`，供用户手动填写
- `true`：额外运行正则提取 amount/date/merchant，前端"一键填入"

### 坐标感知行分组
Python 服务返回每个文字块的 `x`/`y` 坐标（bounding box 中心），Go 端 `groupTextsByLine()` 按动态阈值（中位行高 × 0.6）将同行文本合并，保留原始排版，便于人工对比凭证。

---

## 5. 权限控制

使用 Casbin 内存模式，启动时加载固定策略（不存数据库）：

```
admin   → 所有资源所有操作
editor  → transaction/ocr（读写），category（读）
viewer  → transaction/statistics/export/category（只读）
```

中间件链：`JWT()` → `Tenant()` → `Casbin(resource, action)`，三层均可单独拦截。

---

## 6. 前端架构

### 状态管理
两个 Zustand store，均配置 `persist`（localStorage）：
- `useAuthStore`：token、userId、username
- `useTenantStore`：currentTenantId、tenants 列表

切换租户后相关页面数据自动重新请求（各页面在 `useEffect` 中依赖 `currentTenantId`）。

### 响应式布局策略
- `useResponsive()` hook 判断 `window.innerWidth < 768`
- PC：侧边栏导航 + 内容区；移动端：底部 Tab 导航
- 同一套组件，通过条件渲染切换布局，不维护两套代码

### TransactionForm 三栏设计（PC）
```
┌────────────┬──────────────┬───────────────┐
│ 左：凭证图片 │ 中：OCR 识别结果 │ 右：记账表单    │
│ (208px 固定)│ (256px 固定)   │ (flex-1 弹性) │
│ 点击放大   │ 右键→填入字段  │               │
└────────────┴──────────────┴───────────────┘
```
移动端改为全屏弹层竖向堆叠：图片 → OCR 结果 → 表单。

### 右键填入交互
OCR 原文区每行文本支持右键（PC）/点击（移动端）唤起上下文菜单，选择填入：金额、商户、日期、备注。日期解析支持 `YYYY-MM-DD` / `YYYY年M月D日` 格式，填入时自动补上当前时分。

---

## 7. 数据库关键设计

- `transaction_date` 类型为 **DATETIME**（非 DATE），支持记录时分
- 所有日期筛选/统计查询使用 `DATE(transaction_date)` 包裹，避免时间部分干扰比较
- `categories` 表含 `UNIQUE(tenant_id, name, type)` 约束，防止租户内重名
- 所有业务表含 `deleted_at` 软删除字段
- `transaction_images` 表通过 `ON DELETE CASCADE` 随交易删除

---

## 8. Docker 构建策略

- **Go 后端**：使用 `vendor` 模式（`go mod vendor` 预下载），Docker build 中 `go build -mod=vendor`，完全离线无需网络
- **OCR 服务**：依赖 OrbStack 内置 `proxy.orb.internal:8305` 透传 macOS 系统代理拉取镜像和安装 pip 包；不可将 `http_proxy=127.0.0.1:7890` 直接传入 build args（容器内 127.0.0.1 是容器自身）
