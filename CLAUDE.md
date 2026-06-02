# 饭店记账 — 开发规范

## 项目运行

```bash
# 启动后端 + OCR 服务（MySQL 需本地运行，密码 allgood）
cd /Users/chen/IdeaProjects/fandianjizhang
docker compose up -d

# 启动前端（需 Node 22，通过 nvm use 22 切换）
cd web && npm run dev -- --port 5274
```

> 注意：5173 端口被 aiblog 项目占用，固定使用 5274。

## 编码规范

### Go 后端

- **无 interface{}**：所有函数参数和返回值必须强类型，DTO 和 model 严格分离
- **repo 接口隔离**：所有数据库操作通过 `internal/repo/interface.go` 中的接口定义，handler 层不得直接访问 DB
- **中间件顺序不可变**：JWT → Tenant → Casbin，三层依赖前一层注入 context 的值
- **日期查询必须包裹 DATE()**：`transaction_date` 是 DATETIME 类型，所有日期范围/精确匹配查询必须用 `DATE(transaction_date)`，否则时间部分导致匹配失败
- **JoinedAt 必须显式赋值**：创建 `TenantMember` 时必须设置 `JoinedAt: time.Now()`，MySQL 5.7 严格模式拒绝零值 datetime
- **默认分类在 default_categories.go 维护**：注册和创建租户均调用 `createDefaultCategories()`，不在各自 service 中各自维护列表

### React 前端

- **防御性编程**：所有可能为 null/undefined 的属性访问均使用 `?.` 和 `?? 默认值`，杜绝 `.toFixed()` 等直接调用
- **Object URL 必须清理**：凡是用 `URL.createObjectURL()` 创建的预览 URL，必须在 `useEffect` 返回值或下次创建前调用 `URL.revokeObjectURL()` 释放内存
- **datetime-local 格式转换**：前端使用 `YYYY-MM-DDTHH:mm` 格式，提交时转为 `YYYY-MM-DD HH:mm:00`（replace T + 拼接秒）
- **图片 URL 构造**：后端存储的 `image_path` 以 `/uploads/` 开头，前端访问时加 `/api` 前缀（vite proxy 会自动剥离转发）
- **租户名后缀固定**：租户重命名时前端强制追加 `的记账本`，保存时拼接，显示时可剥离后缀展示前缀

### 部署

- **Go 镜像构建用 vendor 模式**：`go build -mod=vendor`，修改依赖后需在宿主机重新 `go mod vendor`
- **OCR 服务不传代理 build-args**：利用 OrbStack 的 `proxy.orb.internal` 自动代理，宿主机的 `http_proxy` 环境变量不得透传进 Docker build

## 待办（下次会话）

- [x] 多图上传：左右切换条、排队识别、已保存图片标记
- [x] 交易列表查看关联图片（缩略图 + lightbox）
- [ ] 现有账号（test/testuser）补充默认分类的 SQL 脚本
