# 虚拟道具支付协议与状态机（2026-11-10 纠正版）

> 小管家通用支付协议；汤圆积分解释在汤圆侧。切勿把小管家当成记账充值系统。

## 1. 客户端支付

- `wx.requestVirtualPayment({signData, mode:'short_series_goods', paySig, signature})`。
- `signData` 为原始 JSON 字符串，字段：`offerId/buyQuantity/env/currencyType/productId/goodsPrice/outTradeNo/attach`。
- `paySig = hex(hmac_sha256(appKey, "requestVirtualPayment" + "&" + signData))`。
- `signature = hex(hmac_sha256(sessionKey, signData))`。
- `session_key` 只能服务端 `code2session` 获取；前端 success 不作为发货依据。

## 2. 小管家通用契约

- 商品：`sku_code/name/wx_product_id/price_fen/delivery_payload/status/sort`；`delivery_payload` 为不透明 JSON 对象（≤16KB），仅校验结构，不解释业务字段。
- 购买意图请求：`{business_reference, sku_code, quantity, amount_fen, payer_open_id?}`；不含账本/积分/用户业务归属。
- 订单固化：`app_id/openid/business_reference/sku/product_id/quantity/amount_fen/delivery_payload`。
- 发货/退款 outbox：`event_key/event_type/app_id/out_trade_no/wx_order_id/open_id/business_reference/sku_code/wx_product_id/quantity/amount_fen/delivery_payload`；退款额外 `mch_refund_id/wx_refund_id/refund_fee_fen/order_amount_fen/full_refund`。
- `delivery_payload` 原样透传，接入方按自己的 schema 解释。

## 3. 服务间 HMAC

- `hex(hmac_sha256(secret, timestamp+"\n"+nonce+"\n"+rawBody))`，头 `X-VP-Timestamp/X-VP-Nonce/X-VP-Signature`。
- 5 分钟时间窗；intent/prepare nonce 幂等；失败关闭流程。
- 交付目标由管理员固定配置，HTTPS + 主机后缀白名单；不接受客户端回调地址。

## 4. 微信回调

- 明文：`signature = SHA1(sort(token,timestamp,nonce))`。
- 安全模式：`msg_signature = SHA1(sort(token,timestamp,nonce,Encrypt))`，AES-256-CBC 解密并核对 appid。
- 缺签名/缺配置/解密失败拒绝，不分发 xpay。
- 发货字段：`OutTradeNo/OpenId/WeChatPayInfo.MchOrderNo/GoodsInfo.ProductId/Quantity/ActualPrice`。
- 退款：`RetCode=0` 才是成功；支持 `OutTradeNo/MchOrderId/WxOrderId` 定位原单；`RefundFee` 记录金额。部分/全额均作为通用支付事实通知接入方，是否回收业务权益由接入方决定。
- 应答跟随格式：JSON `{"ErrCode":0,"ErrMsg":"success"}` 或 XML；业务错误非 0 触发重试。

## 5. 查单/发货确认

- `POST https://api.weixin.qq.com/xpay/query_order?access_token=...&pay_sig=...`，body `{openid,env,order_id}`；`status=2` 才补发货。
- `notify_provide_goods` 只在业务 outbox 已确认后调用；无 session_key 仅带 pay_sig（需真机确认）。

## 6. 汤圆积分解释

- `delivery_payload` schema：`{"credit_amount":600,"validity_days":365}`。
- 汤圆本地意图固化 tenant/user/credit_amount/validity_days；按 `business_reference` 映射事件。
- 发货事务：幂等事件 → 锁 `tenant_credits` → 惰性清理到期 → 创建 `credit_lots(source=purchase, order_ref=business_ref)` → 更新余额/写 `credit_logs` → delivered。
- 退款：仅 `refund_fee_fen == 本地订单金额` 自动回收原订单批次剩余；部分/缺金额人工核查；`credit_log_details.Amount` 保持正数，退款日志负金额、按 action 过滤统计已消费。
- OCR/AI 继续旧 `CheckAndDeduct`；没有独立次数余额/mode/UsageGuard。

## 7. 状态机

`created/pending_payment → paid_confirmed → delivering → delivered | delivery_failed_retryable`；`payment_closed/cancelled/timeout_unknown`；退款 `partially_refunded/refunded`；异常 `manual_review`。前端 success、通知到达、退款任务受理均不能单独作为最终业务事实。

## 8. 未验证

真实微信开通/OfferID/AppKey/商品发布/推送配置、Android/iOS 真机、真实 `query_order`/`notify_provide_goods` 响应、无 session_key 服务端签名、iOS IAP、部分退款自动映射、部署与生产迁移。
