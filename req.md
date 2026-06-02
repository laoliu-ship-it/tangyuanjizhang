多租户个人/家庭记账应用（OCR + RBAC + 多端自适应）
本提示词完整描述了开发一个多租户、支持 OCR 识别收据/发票、带 RBAC 权限控制、手机与 PC 自动适配的记账应用的全部技术细节与业务需求。可直接作为 AI 编程提示词或开发文档使用。

1. 多租户模型
系统采用 共享数据库、独立数据行 的多租户架构（租户通过 tenant_id 隔离）。
每个注册用户默认创建一个个人租户（workspace），也可以创建新的租户（例如“家庭账本”、“工作室账本”）。
租户内部支持成员邀请与角色分配（管理员 / 编辑者 / 查看者），角色权限由 Casbin 控制。
所有交易记录、分类、截图等资源均属于某个租户，同一用户在不同租户中拥有不同的角色。
用户登录后需选择进入的租户空间（或直接进入默认租户），租户切换功能存在于顶部导航栏。
API 请求头中携带当前选中的租户 ID（如 X-Tenant-ID），所有资源查询自动带上该租户过滤条件。
2. 技术栈
层次	技术选型
前端	React 18 + TypeScript + Vite + Tailwind CSS + Zustand + React Router v6 + Recharts + dayjs + Axios
响应式	移动端优先，Tailwind 断点，移动端表格转卡片，关键操作触摸优化
后端	Go 1.21+ (Gin) + GORM + MySQL 8.0
权限	Casbin v2 (RBAC) + GORM Adapter，策略持久化，租户内角色管理
认证	JWT (access + refresh token)
OCR	PaddleOCR 本地推理引擎，通过 CGO 编译进 Go 二进制，可插拔
存储	本地磁盘 ./uploads（可扩展 S3），按租户/日期组织路径
Excel	excelize
数据库	MySQL InnoDB, utf8mb4
3. 核心功能
3.1 多租户与权限
用户注册后自动创建个人租户，同时成为该租户的管理员。
用户可创建新租户并邀请其他用户加入，分配角色：admin（管理成员与分类）、editor（添加/修改交易）、viewer（仅查看）。
Casbin 策略格式：p, <tenant_id>:<user_id>, <resource>, <action>，同时支持角色继承 g。
切换租户后，全局状态（当前租户 ID）更新，所有 API 请求自动携带租户头。
3.2 记账与 OCR
每笔交易独立记录，关联租户、用户、截图路径。
批量上传图片 → PaddleOCR 本地识别 → 提取金额、日期、商户 → 前端编辑确认 → 批量保存。
OCR 模块接口化，通过配置文件指定引擎，轻松切换（Tesseract/云 OCR）。
图片按 uploads/{tenant_id}/{date}/ 存储，确保隔离。
3.3 统计与导出
仪表板：今日收支、本月趋势（租户内）。
按日/按月/时间段统计，支持分类筛选。
Excel 导出：根据当前筛选条件，导出租户数据（仅当前成员有权查看的部分）。
3.4 多端适配
PC：侧边栏导航 + 表格视图。
移动端：底部 Tab 导航，列表卡片化，上传调用相机/相册，全屏模态表单。
4. 数据库设计（核心表，多租户版）
CREATE TABLE users (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(50) UNIQUE NOT NULL,
    email         VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tenants (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    owner_id      BIGINT NOT NULL,          -- 创建者
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE tenant_members (
    tenant_id     BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    role          VARCHAR(20) DEFAULT 'viewer', -- admin/editor/viewer
    joined_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, user_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (user_id)   REFERENCES users(id)
);

CREATE TABLE categories (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id     BIGINT NOT NULL,
    name          VARCHAR(50) NOT NULL,
    type          ENUM('income','expense') NOT NULL,
    icon          VARCHAR(50) DEFAULT '',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE transactions (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id         BIGINT NOT NULL,
    user_id           BIGINT NOT NULL,       -- 创建者
    type              ENUM('income','expense') NOT NULL,
    amount            DECIMAL(12,2) NOT NULL,
    category_id       BIGINT NOT NULL,
    transaction_date  DATE NOT NULL,
    note              VARCHAR(255) DEFAULT '',
    image_path        VARCHAR(500) DEFAULT '',
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id)   REFERENCES tenants(id),
    FOREIGN KEY (user_id)     REFERENCES users(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    INDEX idx_tenant_date (tenant_id, transaction_date)
);

-- Casbin 策略表由 GORM Adapter 自动创建，无需手动编写。
-- 策略示例：g, tenant:1:user:2, editor   (表示用户2在租户1中拥有editor角色)
-- p, editor, transaction, read           (editor角色对交易资源有读权限)
5. API 设计（增加租户相关）
方法	路径	说明	权限
POST	/api/auth/register	注册，自动创建默认租户	公开
POST	/api/auth/login	登录，返回 JWT 及用户关联的租户列表	公开
GET	/api/tenants	获取当前用户所属的所有租户	登录
POST	/api/tenants	创建新租户（当前用户成为管理员）	登录
PUT	/api/tenants/:id	修改租户名称（管理员）	租户管理员
POST	/api/tenants/:id/members	邀请成员（指定用户名和角色）	租户管理员
DELETE	/api/tenants/:id/members/:userId	移除成员	租户管理员
GET	/api/tenants/:id/members	查看成员列表	租户成员
POST	/api/upload/ocr	上传图片并 OCR（租户隔离）	租户 editor+
POST	/api/transactions/batch	确认并批量创建交易	租户 editor+
GET	/api/transactions	查询交易（租户+筛选）	租户 viewer+
GET	/api/statistics/daily	日统计	租户 viewer+
GET	/api/statistics/monthly	月统计	租户 viewer+
GET	/api/statistics/range	时间段统计	租户 viewer+
GET	/api/export/excel	导出 Excel	租户 viewer+
GET/POST/PUT/DELETE	/api/categories*	分类 CRUD（管理：admin，查看：全部成员）	租户管理员
所有业务 API 需在 Header 中带 Authorization: Bearer <jwt> 和 X-Tenant-ID: <tenant_id>。中间件依次验证 JWT → 租户成员身份 → Casbin 权限。

6. 后端分层与 OCR 可插拔设计
server/
├── cmd/
├── internal/
│   ├── model/              # User, Tenant, TenantMember, Category, Transaction
│   ├── repo/               # 接口定义，所有方法包含 tenantID 参数
│   ├── service/            # 业务：租户服务、交易服务、OCR 服务、Excel 服务
│   ├── handler/            # 路由绑定、请求校验
│   ├── middleware/         # JWT 中间件、租户解析中间件、Casbin 鉴权中间件
│   └── dto/                # 请求/响应结构体
├── pkg/
│   ├── ocr/
│   │   ├── engine.go       # Engine 接口：Recognize(ctx, imagePath) (*Result, error)
│   │   ├── paddle.go       # PaddleOCR 实现（CGO）
│   │   └── factory.go      # 工厂 + 注册机制，根据配置创建引擎
│   ├── excel/
│   └── storage/
├── config/                 # 配置文件（数据库、OCR 引擎类型、模型路径）
└── models/                 # PaddleOCR 模型文件（det/rec/cls）
OCR 引擎初始化示例：ocr.NewEngine(config.OCR.Engine, config.OCR.Options)，只需改配置即可替换为 Tesseract 或云 API，无需改动业务代码。

7. 前端结构（多租户 + 响应式）
src/
├── components/          # 公共组件（ResponsiveTable, MobileCardList, TenantSwitcher...）
├── pages/
│   ├── Auth/            # 登录/注册
│   ├── Dashboard/       # 首页（选择/切换租户后展示）
│   ├── UploadOCR/
│   ├── Transactions/
│   ├── Statistics/
│   └── TenantSettings/  # 成员管理、分类管理
├── hooks/               # useAuth, useTenant, useResponsive
├── services/            # Axios 实例（自动附加 JWT 和 X-Tenant-ID）
├── store/               # Zustand（auth, tenant, transactions）
└── layouts/             # PC侧边栏 / 移动端标签栏
登录后调用 /api/tenants 获取租户列表，存入 store，默认进入第一个（或上次选择的）。
所有页面左上角或顶部显示租户切换组件（TenantSwitcher），切换时重置相关状态并重新请求数据。
8. 交付要点总结
多租户隔离：所有数据操作通过 tenant_id 严格过滤，Casbin 规则包含租户前缀，成员管理 API 必须由租户管理员调用。
PaddleOCR 集成：Docker 或二进制部署均支持，模型可嵌入，引擎可替换，保证本地离线运行。
RBAC 权限：租户内角色精细化控制（管理员可管理分类和成员，编辑者可记账，查看者只读）。
手机/PC 自适应：不区分两套代码，全部通过 Tailwind 响应式实现。
Excel 导出：基于租户和用户权限，导出数据仅包括该用户在租户内有权查看的记录。
代码质量：Go 后端正向分层，杜绝 interface{}，前端 Zustand 管理全局状态，类型安全贯穿始终。
