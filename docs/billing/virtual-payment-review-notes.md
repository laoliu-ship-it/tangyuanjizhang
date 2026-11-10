# 开发中检查项（需 opencode 逐项修复并测试）

> 2026-11-10：本文件记录初版的历史检查。独立次数账户、usage/mode、UsageGuard 等要求已作废；当前以 [积分与解耦纠正说明](credits-payment-correction.md) 和 [最新验收报告](virtual-payment-acceptance.md) 为准。小管家只提供通用虚拟道具支付与不透明交付，积分兑换、发放和扣费均由汤圆处理。下文与当前规则冲突的条目不再作为开发要求。

这是当前实现阶段的检查结果，不是完成状态。检查时间：2026-11-09。

1. `xiaoguanjia/internal/modules/virtualpay/payment.go:newOutTradeNo` 当前是 `vp`(2) + Unix 秒(10) + 12 字节 hex(24) = 36 字符，超过计划和官方 API 的 8–32 字符。修正并测试实际生成值的长度、字符集与随机失败处理。
2. `PrepareOrderRequest` 中 target_tenant_id、target_user_id、business_reference 当前直接来自小程序请求。必须绑定记账后端签名/注册的购买意图，不能信任客户端选择到账账本；补伪造/过期/跨用户意图测试。
3. `MarkOrderPaidByNotify` 当前先 SaveOrder 再 CreateOutbox 且忽略后者错误，两步不在一个事务，可能永久漏发货；并发也没有锁。应在同一事务内锁订单、验证状态/身份/商品/数量/环境/金额、更新支付状态并写 outbox；唯一约束防止不同入口重复发放。
4. `MarkOrderPaidByNotify` 当前允许 openID 为空、amountFen<=0 绕过校验。缺字段必须查单核实或拒绝/人工核查，不能据此确认支付。退款/关闭订单也不能被乱序支付事件直接复活。
5. `seccheck/router.go:PushReceive` 原实现只在 signature 非空时才验签，现在新增的 xpay dispatcher 若复用此分支，会把无签名 POST 送到支付处理。修复所有消息入口：缺签名/缺配置拒绝业务处理，安全模式校验 msg_signature 与密文/appid，必须有伪造 xpay 经旧 seccheck URL 不发货的回归测试。不要只在新 /virtualpay/push 上验签。
6. 当前纯 SKU 预查询防重没有数据库唯一约束，支付配置按 app 也需唯一行。补 MySQL 安全的 app+SKU、app+配置与订单/平台单号约束，考虑软删除键语义；所有写入保持 app 隔离。
7. 本地可交付不能以 `push_stub.go`、只记录响应而不查单、空的发货/退款/次数预占逻辑收尾。外部协议无法验证要记录并默认关闭真实收款，但先从腾讯官方源码核对和实现适配器、测试与独立本地流程。
8. 构建/测试使用管道 `... | tail` 时要保留实际退出码（例如 pipefail），不能把 tail 的 0 当作测试通过。最终测试报告列出实际结果与未验证项。
9. 新增 `s2s.go:AppConfirmOrder` 当前无条件调用 NotifyProvideGoods，即使订单尚未支付或 outbox 未被业务侧确认。只能在业务权益实际到账、订单持久化为 delivered 后通知微信发货完成；否则可能让微信先标已发货而本地没到账。补“未付款确认”“业务侧交付失败”“重试确认”测试。
10. `query.go:QueryOrder` 当前通过 `SaveOrder(o)` 保存查单原文，o 可能是旧快照，会覆盖同时到达的支付/退款状态；必须使用受限字段更新或事务锁，不能 Save 整个陈旧实体。查询响应应核验返回的订单标识，不要用本地快照填回“上游已验证的商品/金额”来冒充校验。只使用官方实际字段，`order_fee/total_fee/pay_fee` 等别名需有官方来源，不可猜测。
11. `ServicePrepareOrder` 也需要服务间 nonce 防重放；所有幂等标记必须与关联业务变更同事务，不能仅预查询后忽略插入失败。退款与订单终态的乱序重放测试必须覆盖。
12. `billingService.BeginUsage` 当前忽略 ReserveUsage 返回状态，repo 遇相同 request_id 直接返回原记录；这会让 committed/released 请求继续调用上游而不再扣次数，也能把 OCR 的 request_id 用于 AI。必须绑定 benefit/count/业务标识，重复 in-flight/closed 请求拒绝执行或复用已保存结果，绝不能再次访问付费上游；加跨权益和已完成/已释放重放的服务/handler 测试，不仅测 repo“不重复扣”。
13. 记账端 `ProcessBillingDelivery` 当前仅跳过 delivered，允许 refunded/cancelled 意图重新发放；退款先到、发货 outbox 后重试会重新到账。补乱序退款后发货测试，终态不能复活。回调要核验 user/app/订单绑定，未知 event_type 不应默认作为发货。
14. `releaseUsageTx` 当前不核对原批次状态/有效期直接增加 remaining；退款/到期与预占释放并发须保持权益不复活、账实一致。GetUsageBalance 也应按当前时间排除到期批次，不能等 cron 才改变可用余额。补 MySQL 并发和到期测试。
15. `demoService.Ensure` 当前直接按 username="demo" 查找并复用用户；若现有正式用户已叫 demo，一键体验会登录到其真实身份并暴露其账本。必须用不可由普通注册伪造的系统演示标记/专用关联记录识别演示用户；遇名称/邮箱冲突不能复用。演示 JWT 绑定唯一 DemoTenantID，登录响应只包含这个演示账本，Tenant 中间件必须对演示 token 强制匹配，不能只依赖该用户当前 memberships。补“已有普通 demo 用户及私人账本”“演示用户额外加入真实账本”“首次并发初始化”测试。演示身份不能访问或升级为任何正式身份。
16. 套餐/定价的唯一管理入口应是小管家。当前记账 Summary/CreatePurchase 只读本地 BillingPackage，若仅有平台手工 seed，会要求管理员维护两套价格，上下架/改价不能传到购买页面。请补从小管家服务端读取/同步商品的明确链路（含鉴权/错误处理），下单前按权威商品重新校验与固化快照；已下架/改价应明确提示刷新，不能展示旧价后静默按新价签名。补跨服务商品同步、改价、下架测试。种子仍默认草稿，不能自动改线上商品。
17. 当前 `handleRefundNotify` 仅凭 RetCode+退款号就调用全量回收，没有读取/核对退款金额；部分退款不能被自动当作全额退款，缺金额也不能全量回收。核对微信官方退款通知中原支付单字段（重点核查 `MchOrderId` / `WxOrderId`，当前仅取 OutTradeNo），校验原订单、openid、env、RefundFee、退款单号，采用官方格式 fixture 测试全额/部分/失败/缺字段与重放。未定义部分退款权益映射时进入人工核查，不回收全包。
18. 当前 getAmountFen 候选未包含嵌套 GoodsInfo 的实际支付金额字段，请对照官方发货通知/腾讯官方示例确认金额字段（重点核查 GoodsInfo.ActualPrice），用真实协议形状的 JSON/XML fixture 测试直接发货。不要只用自定义 TotalFee/OutTradeNo 的假通知测通，再依赖查单兜底掩盖协议解析错误。以上字段核对以微信/腾讯官方来源为准。
19. `pages/billing/index.vue` 当前仅展示 mode，没有调用 setMode 的入口，账本默认 credits，购买后无法通过 UI 启用次数。补管理员明确确认的模式切换（保留旧余额），说明当前使用哪种计费；购买按钮按 useRole、demo、enabled、终端能力控制。小管家 iap_enabled 目前只保存未用于能力判断，需把支付能力传到记账端并按 iOS/IAP/版本判断，H5 不拉起。保留到期与退款规则展示。
20. 用户购买页面不应暴露“小管家”“H5/App 不发起虚拟支付”“请在小管家上架”之类实现说明；换成“套餐暂不可用”“请在微信小程序内购买”“刷新套餐”等面向记账用户的文案。流水 action 用中文动作名，不直接显示 reserve/commit 等内部枚举。

可用的一手参考：

- https://github.com/TencentCloudBase/skills/blob/main/skills/cloudbase-wechat-integration/references/virtual-payment.md
- https://github.com/TencentCloudBase/CloudBase-AI-Toolkit/blob/main/config/source/skills/cloudbase-wechat-integration/references/virtual-payment.md
- https://github.com/TencentCloudBase/awesome-cloudbase-examples/tree/master/integration/cloudbase-virtualpay
- https://docs.cloudbase.net/integration/wechat-virtual-pay

腾讯官方 skills 参考已经列出 query_order 的请求参数、数字 status、退款 RetCode 等，优先据其引用的微信原始页面/官方示例确认，不能把协议字段改成管理员任意配置“成功映射”。尤其：通知的应答成功只表示接收；退款通知本身并不代表退款成功。

## 处理结果（2026-11-09）

第 1–20 条已在 `feature-20261109-virtual-payment` 中处理，逐项结论与测试见 `virtual-payment-acceptance.md` 第 3 节。摘要：

1. outTradeNo 改为 23 位并通过官方 8–32/字符集/开头约束测试。
2. 客户端不再提交到账账本；购买意图由服务间注册，prepare 只传 intent_ref；伪造/过期/跨用户测试通过。
3. 支付状态与 outbox 单事务 + 行锁 + 唯一键；并发 10 次通知只到账一次。
4. 缺 openid/金额不确认支付；查单失败进入人工核查；退款/关闭订单不复活。
5. seccheck 旧入口 fail-closed；无签名 xpay 不进入分发器。
6. app+SKU、app+配置、订单号、平台单号、outbox/event/refund 唯一约束补齐。
7. query_order、notify_provide_goods、outbox 重试、退款、次数预占/回收均已实现，不是 stub。
8. 所有测试/构建使用 `set -o pipefail` 并保留真实退出码。
9. `CanProvideGoods` 仅在已支付+已交付+outbox acked 后通知发货。
10. 查单只更新审计字段并校验上游订单标识，不覆盖并发状态。
11. intent 与 prepare-s2s 均 nonce 防重放，失败释放、成功标记。
12. 预占重复按状态拒绝（reserved 并发重复、committed/released/expired 复用均拒绝）。
13. 退款后乱序发货不重发；余额排除到期批次；释放不复活退款/到期批次。
14. Webhook 未知 event_type 返回 400，不默认发货；缺/错 HMAC 401。
15. 演示账号使用 `demo_accounts` 唯一关联 + is_demo 标记 + JWT demoTenantID；同名普通用户/额外 membership/并发初始化隔离测试通过。
16. 权威商品从服务间同步；下单/支付前校验价格/权益/上下架，改价需刷新；跨服务契约/下架/改价测试通过。
17. 退款读取 `RefundFee`，仅全额匹配自动回收；部分/缺金额人工核查；支持 `MchOrderId/WxOrderId` 找回原单。
18. 发货金额优先 `GoodsInfo.ActualPrice`；JSON/XML 官方字段 fixture 测试通过。
19. 计费模式管理员确认切换；购买按钮受角色/demo/enabled/支付能力/iOS IAP 控制；H5/App 不拉起。
20. 用户文案移除内部系统说明；流水动作中文化。

真机支付、微信后台配置、真实收款/退款仍为未验证项，详见验收报告。

## 2026-11-10 纠正覆盖说明

用户已确认：**积分是唯一实际权益**，小管家是通用虚拟道具支付模块，汤圆按自身 `delivery_payload` 解释积分；不使用独立 OCR/AI 次数账户、usage 模式、UsageGuard、mode 切换。因此本文件第 12、13、19、20 条中涉及“独立次数/usage/mode”的表述不再作为当前需求，以 `credits-payment-correction.md`、`virtual-payment-acceptance.md` 和 `credits-billing-integration.md` 为准。

当前实现：
- 小管家不保存 target_tenant_id/target_user_id、OCRCount/AICount 或业务积分字段；商品/订单/事件使用不透明 `delivery_payload`。
- 汤圆购买发货在单事务内写现有 `credit_lots/tenant_credits/credit_logs`；重复通知只发一次。
- OCR/AI 正式路径仍走原 `CheckAndDeduct`，无额外次数扣减。
- 退款仅全额回收原订单 `source=purchase` 批次剩余；部分退款人工核查；明细 Amount 保持正数。
- 每包估算由后端按当前扣费单价计算，前端标明共享同一积分、不可叠加。
- 解耦、非积分 payload、并发/死锁/到期/估算测试通过；真机与真实收款仍未验证。
