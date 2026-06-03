#!/bin/bash
# 数据库初始化脚本 - 在远程服务器上创建数据库和用户
# 用法: ./scripts/init-db.sh

set -e

# 配置变量
DB_NAME="tangyuanjizhang"
DB_USER="tangyuanjizhang"
DB_PASSWORD=$(openssl rand -hex 16)  # 生成随机密码
SQL_FILE="/Users/chen/Downloads/fandian.sql"

# 远程服务器配置
REMOTE_HOST="root@ddd.com"

echo "======================================"
echo "数据库初始化脚本"
echo "======================================"
echo ""
echo "数据库名称: $DB_NAME"
echo "数据库用户: $DB_USER"
echo "数据库密码: $DB_PASSWORD"
echo ""
echo "请保存以上数据库密码，后续部署需要使用！"
echo ""

# 询问是否继续
read -p "是否继续？(y/n): " confirm
if [ "$confirm" != "y" ]; then
    echo "已取消"
    exit 0
fi

echo ""
echo "正在连接远程服务器创建数据库..."

# 在远程服务器上执行数据库初始化
ssh "$REMOTE_HOST" bash << REMOTE_SCRIPT
set -e

echo "[1/4] 创建数据库 '$DB_NAME'..."
mysql -u root -e "CREATE DATABASE IF NOT EXISTS \\\`$DB_NAME\\\` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;"

echo "[2/4] 创建数据库用户 '$DB_USER'..."
mysql -u root -e "CREATE USER IF NOT EXISTS '$DB_USER'@'localhost' IDENTIFIED BY '$DB_PASSWORD';"

echo "[3/4] 授权用户..."
mysql -u root -e "GRANT ALL PRIVILEGES ON \\\`$DB_NAME\\\`.* TO '$DB_USER'@'localhost';"
mysql -u root -e "FLUSH PRIVILEGES;"

echo "[4/4] 数据库用户创建完成！"
REMOTE_SCRIPT

echo ""
echo "数据库初始化完成！"
echo ""
echo "======================================"
echo "请保存以下信息："
echo "======================================"
echo "DB_DSN=\"$DB_USER:$DB_PASSWORD@tcp(localhost:3306)/$DB_NAME?charset=utf8mb4&parseTime=True\""
echo "======================================"
echo ""

# 保存配置到本地文件
CONFIG_FILE="$(dirname "$0")/../.env.deploy"
cat > "$CONFIG_FILE" << EOF
# 部署配置文件 - 请勿提交到版本控制
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_DSN="$DB_USER:$DB_PASSWORD@tcp(localhost:3306)/$DB_NAME?charset=utf8mb4&parseTime=True"
EOF

echo "配置已保存到 .env.deploy"
echo ""
echo "下一步：运行 ./scripts/deploy.sh 进行部署"
