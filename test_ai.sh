#!/bin/bash

# 测试AI识别功能
# 用法: ./test_ai_recognition.sh [图片路径]

IMAGE_PATH="${1:-/Users/chen/Downloads/jizhang/微信图片_20260529150356_5338_118.jpg}"

if [ ! -f "$IMAGE_PATH" ]; then
    echo "❌ 图片文件不存在: $IMAGE_PATH"
    exit 1
fi

echo "📸 测试图片: $IMAGE_PATH"
echo "🚀 正在上传并识别..."

# 替换成你的服务器地址
API_URL="http://localhost:8080/api/upload/ocr/analyze"

# 上传图片并获取结果
RESPONSE=$(curl -s -X POST "$API_URL" \
    -H "Authorization: Bearer YOUR_TOKEN" \
    -F "file=@$IMAGE_PATH" \
    -w "\nHTTP_STATUS:%{http_code}")

echo ""
echo "📋 识别结果:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 提取HTTP状态码
HTTP_STATUS=$(echo "$RESPONSE" | grep -o 'HTTP_STATUS:[0-9]*' | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS:/d')

if [ "$HTTP_STATUS" = "200" ]; then
    echo "$BODY" | python3 -m json.tool
else
    echo "❌ HTTP状态码: $HTTP_STATUS"
    echo "$BODY"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
