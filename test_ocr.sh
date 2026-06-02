#!/bin/bash

# 测试AI识别功能
echo "🧪 开始测试AI识别功能..."
echo ""

# 登录获取token
echo "📝 正在登录..."
LOGIN_RESPONSE=$(curl -s -X POST http://192.168.0.25:8090/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"cece@qq.com","password":"mm123456"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败"
    echo "$LOGIN_RESPONSE"
    exit 1
fi

echo "✅ 登录成功，获取到token"
echo ""

# 获取租户ID
echo " 正在获取租户信息..."
TENANT_RESPONSE=$(curl -s http://192.168.0.25:8090/api/tenants \
  -H "Authorization: Bearer $TOKEN")

TENANT_ID=$(echo "$TENANT_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data'][0]['id'])" 2>/dev/null)

if [ -z "$TENANT_ID" ]; then
    echo "❌ 获取租户失败"
    echo "$TENANT_RESPONSE"
    exit 1
fi

echo "✅ 租户ID: $TENANT_ID"
echo ""

# 测试第一张图片（微信单个付款）
IMAGE1="/Users/chen/Downloads/jizhang/微信图片_20260529150356_5338_118.jpg"
if [ -f "$IMAGE1" ]; then
    echo " 测试图片1: 微信付款截图"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    UPLOAD_RESPONSE=$(curl -s -X POST http://192.168.0.25:8090/api/upload/ocr/analyze \
      -H "Authorization: Bearer $TOKEN" \
      -H "X-Tenant-ID: $TENANT_ID" \
      -F "file=@$IMAGE1")
    
    echo "$UPLOAD_RESPONSE" | python3 -m json.tool
    
    echo ""
    echo "✅ 预期结果："
    echo "  - 商户: 杭州连杭食品有限公司"
    echo "  - 金额: 32.00"
    echo "  - 时间: 2026-05-27"
    echo "  - 备注: 大桶水"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
fi

# 测试第四张图片（支付宝账单详情）
IMAGE4="/Users/chen/Downloads/jizhang/62dc207a-09e0-44d5-9c20-7a09e014d501 (2).jpg"
if [ -f "$IMAGE4" ]; then
    echo "📸 测试图片4: 支付宝账单详情"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    UPLOAD_RESPONSE=$(curl -s -X POST http://192.168.0.25:8090/api/upload/ocr/analyze \
      -H "Authorization: Bearer $TOKEN" \
      -H "X-Tenant-ID: $TENANT_ID" \
      -F "file=@$IMAGE4")
    
    echo "$UPLOAD_RESPONSE" | python3 -m json.tool
    
    echo ""
    echo "✅ 预期结果："
    echo "  - 商户: 心人"
    echo "  - 金额: 20.00"
    echo "  - 时间: 2026-06-01"
    echo "  - 特征: 余额宝、支付奖励（支付宝标识）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
fi

echo "🎉 测试完成！"
