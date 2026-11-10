# 虚拟支付与积分购买验收报告（2026-11-10 纠正版）

日期：2026-11-10
分支：`feature-20261109-virtual-payment`（两仓保留原有未提交改动，未提交旧改动）

## 1. 当前架构

- **小管家**：通用虚拟道具支付模块，只处理 App 隔离、商品/售价/微信 productId、购买意图、支付、验签、查单、通用发货/退款 outbox 与重试。
- **汤圆记账**：接入方，本地保存 `tenant/user/credit_amount/validity_days/业务引用`，解释 `delivery_payload`，发放现有积分批次，保留原 OCR/AI/上传/存储扣费。
- 小管家不保存 `target_tenant_id/target_user_id`、OCRCount/AICount 或积分字段；商品/订单/事件只带不透明 `delivery_payload`。
- 汤圆核心 `BillingService` 依赖本地窄 `PaymentGateway` 接口；`xgjclient` 仅 HTTP/HMAC 适配器，`paymentadapter` 做类型转换。

## 2. 本地验证结果

| 命令 | 结果 |
|---|---|
| `fandianjizhang/server: go test -mod=vendor ./...` | **PASS**（独立临时 MySQL） |
| `xiaoguanjia: XIAOGUANJIA_TEST_DSN=<临时库> go test ./...` | **PASS** |
| `xiaoguanjia/web/admin: npm run build` | **PASS** |
| `uniapp: npm run type-check` | **PASS** |
| `uniapp: npm run test:billing-capability` | **PASS**（5/5） |
| `uniapp: npm run build:mp-weixin` | **PASS** |
| `uniapp: npm run build:h5` | **PASS** |
| gofmt / diff --check | 本次改动 Go 文件已格式化 |

临时库：`fandian_vp_repo_it`、`fandian_vp_service_it`、`xiaoguanjia_vp_it`；测试后删除，未连接共享/生产库。

## 3. review-notes 处理结果（含纠正）

1. outTradeNo：23 位、8–32、字符集/开头校验；测试通过。
2. 客户端不能指定到账对象：小管家意图只收 `business_reference/sku/quantity/amount`；汤圆本地映射 tenant/user。
3. 支付事实与 outbox：小管家 `ApplyPaidNotify` 单事务锁订单、校验身份/金额/商品/数量/环境，写状态+outbox；并发通知只到账一次。
4. 缺字段：缺 openid/金额人工核查，不确认支付；退款/关闭不复活。
5. seccheck 入口 fail-closed：缺签名/错 appid/解密失败拒绝，不分发 xpay。
6. 唯一约束：app+config、app+SKU、订单号、平台单号、event/outbox/refund 幂等键。
7. 非 stub：query_order、notify_provide_goods、outbox 重试、退款事实、积分发放/回收均已实现。
8. pipefail：测试/构建保留真实退出码。
9. 发货确认：仅已支付+已交付+outbox acked 后调用 `notify_provide_goods`。
10. 查单：只更新审计字段并校验上游订单标识，不覆盖并发状态。
11. nonce：intent/prepare-s2s 防重放。
12. **已被纠正覆盖**：不使用 usage 预占/UsageGuard；OCR/AI 仍走原积分 `CheckAndDeduct`。
13. **已被纠正覆盖**：不建独立次数余额；积分发放/退款按现有 `credit_lots/tenant_credits/credit_logs`。
14. Webhook 未知 event_type：400，不默认发货；HMAC 缺失/错误 401。
15. 演示账号隔离：`demo_accounts` + is_demo + JWT demoTenantID；同名普通用户/额外 membership/并发初始化测试通过。
16. 权威商品：小管家 S2S 商品（含 `delivery_payload`/IAP）；汤圆同步并解析自己的积分 schema；改价/改权益需刷新；无效 payload 拒绝上架/下单。
17. **纠正后**：小管家把已验真的成功退款（含部分）记录为通用支付事实并通知；汤圆仅在全额时回收原订单积分批次剩余，部分/缺金额人工核查。`MchOrderId` 兼容先查本地 outTradeNo 再查平台单号。
18. 发货金额优先 `GoodsInfo.ActualPrice`；JSON/XML 官方字段 fixture 测试通过。
19. 计费/购买能力：无 mode 切换；购买按钮受角色、demo、enabled、微信支付能力、iOS IAP/微信版本控制；Android 不依赖 IAP；H5/App 不拉起。能力纯函数回归测试通过。商品卡由后端 `estimateUsage` 按每包 `credit_amount` + 当前扣费单价返回 OCR/AI/全链路参考次数，前端标明“共享同一积分、不可叠加”；余额为 0 时按单价是否为 0 分别提示，不一律说 OCR/AI 不可用。
20. 文案：积分充值/余额/原积分流水；移除独立 OCR/AI 次数卡与 mode；流水动作中文化。

## 4. 积分正确性（MySQL 实测）

- 发货只增加现有积分；`tenant_credits.balance`、`credit_lots`、`credit_logs` 一致。
- 重复/并发同 event_key 只发一次；发放+幂等事件同事务。
- 退款只回收原订单 `source=purchase` 批次剩余；赠送积分不动；已消费从 `credit_log_details`（排除 refund/expire）统计；不制造负余额。
- 购买/退款与 `Deduct` 都先锁 `tenant_credits` 再处理批次，避免死锁；发货/退款先惰性清理到期批次再汇总余额。
- 退款 `credit_log_details.Amount` 保持正数；`abs(credit_logs.amount) == 明细合计` 测试通过。
- 部分退款不扣积分，记录人工核查。
- 估算：上传 0 + OCR 10 仍可估算；AI/全 0 不虚构；非法/缺配置/溢出有测试；与 `getPlatformActionCost` 共用解析。

## 5. 未完成/未验证

- 微信虚拟支付开通、OfferID/AppKey、商品发布/productId/价格。
- 微信后台发货推送 URL/Token/EncodingAESKey/加密方式/事件订阅。
- Android/iOS 真机、iOS IAP、微信 8.0.68+。
- 真实 `query_order`/`notify_provide_goods` 响应、无 session_key 服务端签名。
- 真实退款通知、部分退款自动映射、生产迁移、部署、真实收款/退款/对账。

## 6. 文档

- 计划/协议/进度/配置/上线：同目录。
- 小管家通用接入：`xiaoguanjia/docs/virtualpay-module.md`。
- 汤圆积分接入：`docs/billing/credits-billing-integration.md`。
- 纠正来源：`credits-payment-correction.md`。
- 旧 review-notes 中 12/13/19/20 的独立次数/usage/mode 设计已被纠正覆盖，不再作为当前需求。
