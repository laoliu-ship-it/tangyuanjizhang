# 汤圆记账：虚拟商品购买积分接入说明

> 2026-11-10 纠正：**积分是唯一实际计费权益**。虚拟道具购买成功后只发普通积分；OCR/AI 继续走原有 `credits_per_upload` / `credits_per_ocr` / `credits_per_llm` 扣费。不存在独立 OCR/AI 次数余额、usage 模式或 UsageGuard。
>
> 小管家是通用虚拟道具支付模块；汤圆是接入方。二者解耦，见小管家 `docs/virtualpay-module.md`。

## 1. 商品与换算

- 小管家商品只保存通用售价、微信 productId 与不透明 `delivery_payload`。
- 汤圆在自己的 payload schema 中解释积分：

```json
{"credit_amount":600,"validity_days":365}
```

- 默认换算：**100 积分 = 1 元**，即 `credit_amount = price_fen`；后台可配置赠送/优惠，但只增加积分，不改变价格与微信商品价格一致性。
- 商品快照在购买意图/订单中固化；之后改价、改积分额、改扣费配置都不影响历史订单。
- 无效 payload（缺 `credit_amount`、<=0、超上限、缺/非法 `validity_days`）由汤圆拒绝，不能上架/下单，小管家不解释这些字段。

## 2. 购买流程

1. 小程序请求汤圆后端创建购买意图：`POST /api/billing/purchase-intents`，body `{sku_code, expected_price_fen, platform}`。
2. 汤圆：
   - 通过本地 `PaymentGateway` 读取小管家权威商品；
   - 校验价格/上下架/微信 productId/`delivery_payload`；
   - 本地 `billing_purchase_intents` 固化 `tenant_id/user_id/credit_amount/validity_days/business_ref`；
   - 调用小管家服务间 `POST /api/:app/virtualpay/intents`，只传 `business_reference/sku_code/quantity/amount_fen`。
3. 小程序拿 `business_ref` 请求汤圆 `prepare`，汤圆校验本地意图与权威商品后，代理小管家 `prepare-s2s` 返回 `signData/paySig/signature`。
4. 客户端 `wx.requestVirtualPayment`；前端 success 不作为到账依据。
5. 小管家收到微信发货通知 → 通用 outbox → 汤圆 `/api/internal/billing/delivery`。
6. 汤圆验 HMAC、按 `business_reference` 映射本地意图、解析 `delivery_payload`，在**同一数据库事务**内：
   - 写 `billing_processed_events` 唯一事件；
   - 校验本地意图的 `tenant/sku/amount/credit_amount/validity_days`；
   - 先锁 `tenant_credits`、惰性清理到期批次；
   - 创建 `credit_lots(source=purchase, order_ref=business_ref, expires_at=now+validity_days)`；
   - 更新 `tenant_credits.balance`；
   - 写 `credit_logs(action=purchase)`；
   - 标记意图 delivered。
7. 重复/并发通知只发一次积分。

## 3. 扣费（保持不变）

- OCR：上传成功后扣 `upload` + `ocr`；上传/存储仍按原配置。
- AI：LLM 成功后才扣 `orc2bill`（`credits_per_llm`）。
- 缓存读取不重复扣；用户主动重新解析成功按一次计。
- `tenant_credits.unlimited` 白名单保留。
- 所有扣费仍走原 `TenantCreditsRepo.Deduct`，最近到期优先，写 `credit_log_details`。

## 4. 退款

- 小管家把已验真的成功退款（含部分）作为通用支付事实 outbox 通知汤圆，不判断是否回收积分包。
- 汤圆：
  - `refund_fee_fen == 本地意图金额` 才自动回收**原订单 `credit_lots` 剩余**；
  - 部分退款/缺金额只记录 `billing_refunds(status=human_review)`，不扣任何积分；
  - 已消费部分由 `credit_log_details` + `credit_logs`（排除 refund/expire）统计，进入人工核查；
  - 不扣其他赠送/购买批次，不制造负余额；退款后晚到发货不再次发放。

## 5. 余额与估算

- 余额来自现有 `tenant_credits.balance`，真相源为 `credit_lots.remaining`。
- 估算使用当前平台配置：
  - 上传+OCR：`balance / (credits_per_upload + credits_per_ocr)`
  - AI：`balance / credits_per_llm`
  - 全链路：`balance / (upload + ocr + llm)`
- 这些是**互斥用途估算**，不是可同时兑现的独立配额。
- 任一单价 0/未配置/非法：该估算为空，不虚构默认单价；只有合计为 0 时提示“当前不扣积分/无法给有限次数”。
- 配置解析与真实扣费共用 `getPlatformActionCost`：缺失/读取失败/非法/负数一律 0。

## 6. 配置

```text
BILLING_ENABLED=true
BILLING_XGJ_APP_SLUG=<小程序 slug>
BILLING_XGJ_BASE_URL=https://<小管家>
BILLING_XGJ_SECRET=<与小管家 delivery_secret 一致>
BILLING_CLOCK_SKEW_SECONDS=300
BILLING_DEMO_LOGIN_ENABLED=false
```

数据库新增（AutoMigrate）：`billing_packages`、`billing_purchase_intents`、`billing_processed_events`、`billing_refunds`、`billing_capabilities`、`demo_accounts`，以及 `credit_lots.order_ref`、`users.is_demo`、`tenants.is_demo`。
不新增 usage 表/模式/独立次数 cron；购买积分到期由现有 `credits_expire` 清理。

## 7. 代码边界

- `service.BillingService` 只依赖本地 `PaymentGateway` 窄接口与本地域类型。
- `internal/xgjclient` 只是 HTTP/HMAC 适配器；`internal/paymentadapter` 做本地类型 ↔ wire DTO 转换。
- 核心测试用 fake `PaymentGateway`；适配器契约测试验证字段/签名/幂等。

## 8. 验收

- 购买只增加现有积分；重复/并发通知只发一次；发放+幂等同事务。
- 下单后改积分额/价格不影响历史快照。
- OCR/AI 仍走旧 `CheckAndDeduct`，无额外次数扣减。
- 退款只回收原订单批次剩余；部分退款人工核查；明细 Amount 保持正数，`abs(log.amount)=明细合计`。
- 估算的 0/非法/缺配置/溢出用例通过。
- 演示账号不能支付/充值/消耗真实积分。
- 真机支付、微信后台配置、真实收款仍未验证。
