#!/bin/bash
# 记账应用部署脚本
# 用途：构建前端、编译后端、上传到远程服务器、配置 Caddy、导入数据库
# 用法: ./scripts/deploy.sh

set -e

# 配置变量
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 远程服务器配置
REMOTE_HOST="root@ddd.com"
REMOTE_DIR="/www/wwwroot/tangyuanjizhang.zaixianapp.cn"
REMOTE_OCR_HOST="root@iZ2ze05w6p8x6jerbpdui8Z"

# Nana 服务器配置（用于同步图片）
NANA_REMOTE="nana@127.0.0.1"
NANA_SSH_PORT=3312
NANA_UPLOAD_DIR="/home/nana/fandianjizhang/app/uploads"

# OCR 服务配置
OCR_SERVICE_URL="http://iZ2ze05w6p8x6jerbpdui8Z:8001"

# 加载部署配置（如果存在）
DEPLOY_ENV="$PROJECT_DIR/.env.deploy"
if [ -f "$DEPLOY_ENV" ]; then
    echo "加载部署配置: $DEPLOY_ENV"
    source "$DEPLOY_ENV"
else
    echo "警告: 未找到 .env.deploy 文件"
    echo "请先运行 ./scripts/init-db.sh 初始化数据库"
    echo "或手动创建 $DEPLOY_ENV 文件"
    exit 1
fi

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查必要文件
check_prerequisites() {
    info "检查前置条件..."
    
    if [ ! -f "$PROJECT_DIR/web/package.json" ]; then
        error "未找到 web/package.json，请在项目根目录运行此脚本"
        exit 1
    fi
    
    if [ ! -f "$PROJECT_DIR/server/cmd/main.go" ]; then
        error "未找到 server/cmd/main.go"
        exit 1
    fi
    
    if [ ! -f "/Users/chen/Downloads/fandian.sql" ]; then
        warn "未找到 SQL 文件: /Users/chen/Downloads/fandian.sql"
        read -p "是否继续部署（不导入数据库）? (y/n): " confirm
        if [ "$confirm" != "y" ]; then
            exit 1
        fi
        SKIP_DB_IMPORT=true
    else
        SKIP_DB_IMPORT=false
    fi
    
    info "前置条件检查通过"
}

# 构建前端
build_frontend() {
    info "步骤 1/6: 构建前端..."
    
    cd "$PROJECT_DIR/web"
    npm install
    npm run build
    
    info "前端构建完成"
}

# 编译后端
build_backend() {
    info "步骤 2/6: 编译后端..."
    
    # 复制前端构建产物到 Go embed 目录
    rm -rf "$PROJECT_DIR/server/web/static"
    mkdir -p "$PROJECT_DIR/server/web/static"
    cp -r "$PROJECT_DIR/web/dist/." "$PROJECT_DIR/server/web/static/"
    
    # 交叉编译
    cd "$PROJECT_DIR/server"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -mod=vendor -o "$PROJECT_DIR/dist/server" ./cmd/main.go
    
    info "后端编译完成"
}

# 上传到远程服务器
upload_to_remote() {
    info "步骤 3/6: 上传文件到远程服务器..."
    
    # 创建远程目录
    ssh "$REMOTE_HOST" "mkdir -p $REMOTE_DIR"
    
    # 上传服务器二进制
    rsync -az -e ssh "$PROJECT_DIR/dist/server" "$REMOTE_HOST:$REMOTE_DIR/server"
    ssh "$REMOTE_HOST" "chmod +x $REMOTE_DIR/server"
    
    # 上传 .env 配置文件
    cat > "$PROJECT_DIR/dist/.env" << EOF
# 记账应用配置文件
DB_DSN=$DB_DSN
JWT_SECRET=$(openssl rand -hex 32)
OCR_ENGINE=rapidocr
OCR_RAPIDOCR_URL=$OCR_SERVICE_URL/ocr
OCR_AI_MODE=false
OCR_LLM_CACHE_TTL=3600
UPLOAD_PATH=$REMOTE_DIR/uploads
SERVER_PORT=8080
EOF
    
    rsync -az -e ssh "$PROJECT_DIR/dist/.env" "$REMOTE_HOST:$REMOTE_DIR/.env"
    
    # 创建上传目录
    ssh "$REMOTE_HOST" "mkdir -p $REMOTE_DIR/uploads"
    
    info "文件上传完成"
}

# 导入数据库
import_database() {
    if [ "$SKIP_DB_IMPORT" = true ]; then
        warn "跳过数据库导入"
        return
    fi
    
    info "步骤 4/6: 导入数据库..."
    
    # 上传 SQL 文件到远程服务器
    rsync -az -e ssh /Users/chen/Downloads/fandian.sql "$REMOTE_HOST:/tmp/fandian_import.sql"
    
    # 在远程服务器上导入
    ssh "$REMOTE_HOST" bash << REMOTE_SCRIPT
set -e
echo "导入数据库到 $DB_NAME..."
mysql -u $DB_USER -p'$DB_PASSWORD' $DB_NAME < /tmp/fandian_import.sql
rm -f /tmp/fandian_import.sql
echo "数据库导入完成！"
REMOTE_SCRIPT
    
    info "数据库导入完成"
}

# 配置 Caddy
configure_caddy() {
    info "步骤 5/6: 配置 Caddy..."
    
    # 创建 systemd 服务文件
    ssh "$REMOTE_HOST" bash << 'REMOTE_SCRIPT'
set -e

# 创建 systemd 服务
cat > /etc/systemd/system/tangyuanjizhang.service << 'EOF'
[Unit]
Description=Tangyuan Accounting Application
After=network.target mysql.service

[Service]
Type=simple
WorkingDirectory=/www/wwwroot/tangyuanjizhang.zaixianapp.cn
EnvironmentFile=/www/wwwroot/tangyuanjizhang.zaixianapp.cn/.env
ExecStart=/www/wwwroot/tangyuanjizhang.zaixianapp.cn/server
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tangyuanjizhang
systemctl restart tangyuanjizhang

echo "systemd 服务已配置并启动"
REMOTE_SCRIPT
    
    # 配置 Caddy（如果 Caddyfile 中还没有配置）
    ssh "$REMOTE_HOST" bash << 'REMOTE_SCRIPT'
set -e

CADDYFILE="/etc/caddy/Caddyfile"

# 检查是否已有配置
if ! grep -q "tangyuanjizhang.zaixianapp.cn" "$CADDYFILE" 2>/dev/null; then
    echo "添加 Caddy 配置..."
    cat >> "$CADDYFILE" << 'EOF'

tangyuanjizhang.zaixianapp.cn {
    reverse_proxy localhost:8080
}
EOF
    
    # 重新加载 Caddy
    systemctl reload caddy
    echo "Caddy 配置已更新"
else
    echo "Caddy 配置已存在，跳过"
fi
REMOTE_SCRIPT
    
    # 检查服务状态
    ssh "$REMOTE_HOST" "systemctl status tangyuanjizhang --no-pager | head -20" || true
    
    info "Caddy 配置完成"
}

# 同步图片
sync_images() {
    info "步骤 6/6: 同步图片到远程服务器..."
    
    if [ -d "$NANA_UPLOAD_DIR" ]; then
        warn "无法直接访问 Nana 服务器的本地目录，请手动同步："
        echo ""
        echo "rsync -az -e 'ssh -p $NANA_SSH_PORT' $NANA_REMOTE:$NANA_UPLOAD_DIR/ $REMOTE_HOST:$REMOTE_DIR/uploads/"
        echo ""
        echo "或者在 Nana 服务器上执行："
        echo "scp -P $NANA_SSH_PORT -r $NANA_UPLOAD_DIR/* $REMOTE_HOST:$REMOTE_DIR/uploads/"
    else
        warn "本地未找到 Nana 上传目录，跳过图片同步"
    fi
    
    info "部署流程完成"
}

# 清理
cleanup() {
    info "清理临时文件..."
    rm -f "$PROJECT_DIR/dist/.env"
}

# 主流程
main() {
    echo ""
    echo "======================================"
    echo "  记账应用部署脚本"
    echo "======================================"
    echo ""
    echo "目标服务器: $REMOTE_HOST"
    echo "部署目录: $REMOTE_DIR"
    echo "数据库: $DB_NAME"
    echo ""
    
    read -p "是否继续部署？(y/n): " confirm
    if [ "$confirm" != "y" ]; then
        echo "部署已取消"
        exit 0
    fi
    
    echo ""
    
    check_prerequisites
    build_frontend
    build_backend
    upload_to_remote
    import_database
    configure_caddy
    sync_images
    cleanup
    
    echo ""
    echo "======================================"
    echo -e "${GREEN}部署完成！${NC}"
    echo "======================================"
    echo ""
    echo "访问地址: https://tangyuanjizhang.zaixianapp.cn"
    echo ""
    echo "常用命令："
    echo "  查看服务状态: ssh $REMOTE_HOST 'systemctl status tangyuanjizhang'"
    echo "  查看日志: ssh $REMOTE_HOST 'journalctl -u tangyuanjizhang -f'"
    echo "  重启服务: ssh $REMOTE_HOST 'systemctl restart tangyuanjizhang'"
    echo ""
}

main "$@"
