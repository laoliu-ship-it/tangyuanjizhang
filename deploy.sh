#!/bin/bash
set -e

REMOTE="nana@127.0.0.1"
SSH_PORT=3312
REMOTE_DIR="/home/nana/fandianjizhang"
APP_DIR="$REMOTE_DIR/app"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== 1. 构建前端 ==="
cd "$SCRIPT_DIR/web"
npm run build
cd "$SCRIPT_DIR"

echo "=== 2. 将前端产物写入 Go embed 目录 ==="
rm -rf "$SCRIPT_DIR/server/web/static"
mkdir -p "$SCRIPT_DIR/server/web/static"
cp -r "$SCRIPT_DIR/web/dist/." "$SCRIPT_DIR/server/web/static/"

echo "=== 3. 交叉编译后端 (linux/amd64) ==="
cd "$SCRIPT_DIR/server"
GOOS=linux GOARCH=amd64 go build -mod=vendor -o "$SCRIPT_DIR/dist-server" ./cmd/main.go
cd "$SCRIPT_DIR"

echo "=== 4. 上传文件到服务器 ==="
ssh -p $SSH_PORT $REMOTE "mkdir -p $APP_DIR/uploads"

# 上传 Go 二进制（前端已内嵌，无需单独上传静态文件）
rsync -az -e "ssh -p $SSH_PORT" dist-server $REMOTE:$APP_DIR/server
ssh -p $SSH_PORT $REMOTE "chmod +x $APP_DIR/server"

# 上传 OCR 专用 docker-compose
rsync -az -e "ssh -p $SSH_PORT" docker-compose.ocr.yml $REMOTE:$REMOTE_DIR/

# 上传 systemd 服务文件
ssh -p $SSH_PORT $REMOTE "mkdir -p ~/.config/systemd/user"
rsync -az -e "ssh -p $SSH_PORT" fandianjizhang.service $REMOTE:~/.config/systemd/user/

echo "=== 5. 远程配置 ==="
ssh -p $SSH_PORT $REMOTE bash << 'REMOTE_SCRIPT'
set -e

# 确保 .env 存在（含 JWT_SECRET）
if [ ! -f ~/fandianjizhang/.env ]; then
  echo "JWT_SECRET=$(openssl rand -hex 32)" > ~/fandianjizhang/.env
  echo "[!] 创建了新的 .env（JWT_SECRET 已生成，用户需重新登录）"
fi

# 切换 OCR：停旧的 prod compose，启动纯 OCR compose
cd ~/fandianjizhang
docker compose -f docker-compose.prod.yml stop server web 2>/dev/null || true
docker compose -f docker-compose.prod.yml rm -f server web 2>/dev/null || true
docker compose -f docker-compose.ocr.yml up -d
echo "OCR 服务已启动"

# 启动/重启用户级 systemd 服务
systemctl --user daemon-reload
systemctl --user enable fandianjizhang
systemctl --user restart fandianjizhang
sleep 2
systemctl --user status fandianjizhang --no-pager | head -15
REMOTE_SCRIPT

echo "=== 清理本地构建产物 ==="
rm -f dist-server

echo ""
echo "=== 部署完成 ==="
echo "访问地址: http://<服务器IP>:8090"
