#!/bin/bash
# 图片同步脚本
# 用途：从 Nana 服务器同步上传的图片到远程部署服务器
# 用法: ./scripts/sync-images.sh

set -e

# 配置变量
NANA_REMOTE="nana@127.0.0.1"
NANA_SSH_PORT=3312
NANA_UPLOAD_DIR="/home/nana/fandianjizhang/app/uploads"

REMOTE_HOST="root@ddd.com"
REMOTE_UPLOAD_DIR="/www/wwwroot/tangyuanjizhang.zaixianapp.cn/uploads"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo ""
echo "======================================"
echo "  图片同步脚本"
echo "======================================"
echo ""
echo "源: Nana 服务器 ($NANA_REMOTE:$NANA_UPLOAD_DIR)"
echo "目标: 部署服务器 ($REMOTE_HOST:$REMOTE_UPLOAD_DIR)"
echo ""

# 确保远程上传目录存在
info "确保远程上传目录存在..."
ssh "$REMOTE_HOST" "mkdir -p $REMOTE_UPLOAD_DIR"

# 同步图片
info "开始同步图片..."
info "这将通过 Nana 服务器中转..."

# 方案 1: 直接从 Nana 服务器推送到远程服务器
# 这需要 Nana 服务器能够访问远程服务器
ssh -p "$NANA_SSH_PORT" "$NANA_REMOTE" bash << REMOTE_SCRIPT
set -e

if [ ! -d "$NANA_UPLOAD_DIR" ]; then
    echo "错误: Nana 服务器上未找到目录 $NANA_UPLOAD_DIR"
    exit 1
fi

echo "找到 $(find "$NANA_UPLOAD_DIR" -type f | wc -l) 个文件"
echo "开始同步到 $REMOTE_HOST:$REMOTE_UPLOAD_DIR..."

# 使用 rsync 通过 SSH 同步
rsync -az -e "ssh" "$NANA_UPLOAD_DIR/" "$REMOTE_HOST:$REMOTE_UPLOAD_DIR/"

echo "同步完成！"
REMOTE_SCRIPT

echo ""
echo "======================================"
echo -e "${GREEN}同步完成！${NC}"
echo "======================================"
echo ""

# 显示远程服务器上的文件数量
REMOTE_FILE_COUNT=$(ssh "$REMOTE_HOST" "find $REMOTE_UPLOAD_DIR -type f | wc -l")
info "远程服务器现在有 $REMOTE_FILE_COUNT 个文件"
echo ""
