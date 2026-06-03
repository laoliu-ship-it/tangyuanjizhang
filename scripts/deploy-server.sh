#!/bin/bash
# 快速发布脚本 - 仅更新 Go 二进制
# 用法: ./scripts/deploy-server.sh

set -e

REMOTE_HOST="root@ddd.com"
REMOTE_DIR="/www/wwwroot/tangyuanjizhang.zaixianapp.cn"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }

echo ""
echo "=== 快速发布 Go 二进制 ==="
echo ""

# 1. 构建前端
info "1. 构建前端..."
cd "$PROJECT_DIR/web"
npm install --silent
npm run build

# 2. 复制前端产物到 Go embed 目录
info "2. 复制前端产物到 Go embed 目录..."
export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22 --silent > /dev/null 2>&1
mkdir -p "$PROJECT_DIR/server/web/static"
cp -r "$PROJECT_DIR/web/dist/." "$PROJECT_DIR/server/web/static/"

# 3. 交叉编译
info "3. 交叉编译 Go (linux/amd64)..."
cd "$PROJECT_DIR/server"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -o "$PROJECT_DIR/dist/server" ./cmd/main.go

BINARY_SIZE=$(du -h "$PROJECT_DIR/dist/server" | cut -f1)
info "编译完成，二进制大小: $BINARY_SIZE"

# 4. 上传
info "4. 上传到远程服务器..."
rsync -az -e ssh "$PROJECT_DIR/dist/server" "$REMOTE_HOST:$REMOTE_DIR/server"

# 5. 重启服务
info "5. 重启服务..."
ssh "$REMOTE_HOST" "systemctl restart tangyuanjizhang"
sleep 2

# 6. 检查状态
info "6. 检查服务状态..."
STATUS=$(ssh "$REMOTE_HOST" "systemctl is-active tangyuanjizhang")
if [ "$STATUS" = "active" ]; then
    echo ""
    echo -e "${GREEN}=== 发布成功 ===${NC}"
    echo "访问: https://tangyuanjizhang.zaixianapp.cn"
else
    echo ""
    warn "服务状态: $STATUS"
    ssh "$REMOTE_HOST" "journalctl -u tangyuanjizhang -n 5 --no-pager"
fi
