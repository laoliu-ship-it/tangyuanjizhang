# 虚拟支付/积分购买配置模板（无密钥）

> 真实值由管理员/运维在安全渠道配置；不要提交或回显 AppKey/Secret/Token/AESKey/HMAC secret。

## 小管家（通用 virtualpay，默认关闭）

每个小程序一行配置，管理员掩码显示，提交 `*` 保持原值：

| 配置 | 说明 |
|---|---|
| enabled | 默认 false |
| env | 0 正式 / 1 沙箱 |
| offer_id | 虚拟支付 OfferID |
| app_key | 支付签名 AppKey |
| push_token / push_aes_key / push_encrypt_mode | 消息推送验签/解密，与微信后台一致 |
| iap_enabled | iOS IAP 能力 |
| delivery_target_url | 接入方固定 HTTPS 回调 |
| delivery_app_id / delivery_secret | 服务间 HMAC |
| allowed_host_suffixes | 交付主机白名单 |

商品 `delivery_payload` 是不透明 JSON 对象，示例（非积分）：
```json
{"skin_id":"aurora","license":"personal","valid_days":30}
```
小管家不解释字段，只保存/投递。

## 汤圆记账（fan server）

```text
BILLING_ENABLED=true
BILLING_XGJ_APP_SLUG=hehuojizhangzhushou
BILLING_XGJ_BASE_URL=https://<小管家>
BILLING_XGJ_SECRET=<与小管家 delivery_secret 一致>
BILLING_CLOCK_SKEW_SECONDS=300
BILLING_DEMO_LOGIN_ENABLED=false
BILLING_DEMO_TTL_MINUTES=30
```

汤圆商品 payload schema：
```json
{"credit_amount":600,"validity_days":365}
```
- 默认 100 积分/元；`credit_amount` 可含赠送，但只能发普通积分。
- 无效/缺失/<=0/超限由汤圆拒绝。
- 购买意图本地保存 tenant/user/credit_amount/validity_days；小管家只知道 `business_reference`。

## 积分扣费配置（沿用现有 platform_configs）

| key | 默认 |
|---|---:|
| credits_per_upload | 5 |
| credits_per_ocr | 10 |
| credits_per_llm（即 orc2bill） | 20 |
| credits_per_gb_per_day | 按现网 |
| credits_new_user_gift | 1000 |
| credits_validity_days | 365 |

估算使用当前配置；0/缺失/非法不虚构默认单价。

## 上线前

- 微信虚拟支付开通、OfferID/AppKey、商品发布/productId/价格、消息推送 URL/Token/AESKey、事件订阅。
- 小管家商品配置 delivery_payload + 上架；汤圆同步/刷新。
- iOS IAP 开通或保持禁用。
- 生产迁移须备份并在独立预发验证；不自动迁移共享库。
