# 部署指南

## 概述

本文档描述如何将记账应用部署到生产服务器。

## 服务器架构

```
用户浏览器
    ↓ HTTPS
Caddy (自动 HTTPS)
    ↓ HTTP
记账应用 (localhost:8080)
    ↓
MySQL 数据库 (tangyuanjizhang)
    ↓
OCR 服务 (远程服务器:8001)
```

## 前置条件

1. **远程服务器访问**
   - 主服务器: `ssh root@ddd.com`
   - OCR 服务器: `root@iZ2ze05w6p8x6jerbpdui8Z` (端口 8001)
   - Nana 服务器: `ssh -p 3312 nana@127.0.0.1`

2. **软件要求**
   - 本地: Node.js, Go
   - 远程: MySQL, Caddy, systemd

## 部署步骤

### 第一步：初始化数据库

```bash
./scripts/init-db.sh
```

这会：
- 在远程 MySQL 上创建数据库 `tangyuanjizhang`
- 创建专用数据库用户并生成随机密码
- 保存配置到 `.env.deploy` 文件

### 第二步：部署应用

```bash
./scripts/deploy.sh
```

这会：
1. 构建前端 (React/Vite)
2. 编译后端 (Go, linux/amd64)
3. 上传到远程服务器 `/www/wwwroot/tangyuanjizhang.zaixianapp.cn`
4. 导入数据库结构和初始数据
5. 配置 systemd 服务和 Caddy 反向代理
6. 启动应用

### 第三步：同步图片（可选）

```bash
./scripts/sync-images.sh
```

从 Nana 服务器同步 `uploads/` 目录的图片到部署服务器。

## 手动操作指南

### 手动导入数据库

```bash
# 上传 SQL 文件
scp /Users/chen/Downloads/fandian.sql root@ddd.com:/tmp/fandian.sql

# SSH 到服务器导入
ssh root@ddd.com
mysql -u tangyuanjizhang -p tangyuanjizhang < /tmp/fandian.sql
rm /tmp/fandian.sql
```

### 手动配置 Caddy

编辑 `/etc/caddy/Caddyfile`:

```
tangyuanjizhang.zaixianapp.cn {
    reverse_proxy localhost:8080
}
```

然后重载 Caddy:

```bash
systemctl reload caddy
```

### 手动管理服务

```bash
# 查看状态
systemctl status tangyuanjizhang

# 查看日志
journalctl -u tangyuanjizhang -f

# 重启服务
systemctl restart tangyuanjizhang

# 停止服务
systemctl stop tangyuanjizhang
```

## 配置文件

### .env.deploy (自动生成)

```
DB_NAME=tangyuanjizhang
DB_USER=tangyuanjizhang
DB_PASSWORD=<随机生成的密码>
DB_DSN=tangyuanjizhang:<密码>@tcp(localhost:3306)/tangyuanjizhang?charset=utf8mb4&parseTime=True
```

### 远程 .env 文件

部署时自动创建在 `/www/wwwroot/tangyuanjizhang.zaixianapp.cn/.env`

```
DB_DSN=tangyuanjizhang:<密码>@tcp(localhost:3306)/tangyuanjizhang?charset=utf8mb4&parseTime=True
JWT_SECRET=<随机生成>
OCR_ENGINE=rapidocr
OCR_RAPIDOCR_URL=http://iZ2ze05w6p8x6jerbpdui8Z:8001/ocr
OCR_AI_MODE=false
OCR_LLM_CACHE_TTL=3600
UPLOAD_PATH=/www/wwwroot/tangyuanjizhang.zaixianapp.cn/uploads
SERVER_PORT=8080
```

## 目录结构

```
/www/wwwroot/tangyuanjizhang.zaixianapp.cn/
├── server          # Go 二进制文件
├── .env            # 配置文件
└── uploads/        # 上传的图片
```

## 安全注意事项

1. **不要提交 `.env.deploy` 到版本控制** - 包含数据库密码
2. **定期更换 JWT_SECRET** - 在远程服务器的 `.env` 文件中
3. **限制数据库用户权限** - 只给必要的权限
4. **使用 HTTPS** - Caddy 自动配置 Let's Encrypt 证书

## 故障排查

### 服务无法启动

```bash
# 查看详细日志
journalctl -u tangyuanjizhang -n 100 --no-pager

# 检查配置文件
cat /www/wwwroot/tangyuanjizhang.zaixianapp.cn/.env

# 测试数据库连接
ssh root@ddd.com "mysql -u tangyuanjizhang -p tangyuanjizhang -e 'SELECT 1'"
```

### OCR 无法使用

```bash
# 测试 OCR 服务连通性
ssh root@ddd.com "curl -s http://iZ2ze05w6p8x6jerbpdui8Z:8001/ocr"

# 检查 OCR 容器状态
ssh root@iZ2ze05w6p8x6jerbpdui8Z "podman ps"
```

### Caddy 证书问题

```bash
# 查看 Caddy 日志
journalctl -u caddy -f

# 强制重新获取证书
systemctl restart caddy
```
