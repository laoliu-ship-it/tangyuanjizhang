# 虚拟支付开发启动记录

启动日期：2026-11-09（Asia/Shanghai）。

- 开发计划：[virtual-payment-plan.md](virtual-payment-plan.md)
- 执行工具：opencode，使用现有后台服务，默认 build 代理。
- 会话标题：小管家虚拟支付与汤圆积分购买。
- 会话 ID：`ses_e3f3b2611ffeDHFVkHN2cwT4pG`。
- 两仓功能分支：`feature-20261109-virtual-payment`，已确认创建成功。
- 仓库：`/Users/chen/IdeaProjects/fandianjizhang`、`/Users/chen/IdeaProjects/xiaoguanjia`。
- 原有改动清单：`/tmp/fdj_git_baseline.txt`、`/tmp/xgj_git_baseline.txt`（本机临时文件）。
- 已观察到执行：读取两仓规范、读取计划、创建功能分支、保存状态基线。后续阶段由执行会话更新进度文件，尚不代表功能开发或测试完成。

在主仓库目录接入同一会话：

```bash
opencode --session ses_e3f3b2611ffeDHFVkHN2cwT4pG
```

不要另建同一功能的第二个编码会话，以免并发编辑相同文件。

Herdr 接续：用户已在 Herdr 内重新打开本任务。已核实实际 Codex 终端进程位于 `w5:p5`，其 `HERDR_ENV=1`（工具子进程未继承该环境，控制命令复用从实际终端核实的上下文）。在同一 tab 下方新建 `w5:p6`，接入上述同一 opencode 会话；已观察到 Herdr 将其识别为 `opencode / working`，正在执行小管家模块编码与构建。保持用户焦点不变，未关闭其他终端。

原开发任务曾因模型接口返回无效 JSON 报错退出；本次已恢复同一会话，要求继续完成 P1–P4、本地测试、构建与验收报告，不重复建立开发会话。

任务明确要求保留两仓未提交改动、不部署、不运行共享库迁移、不真实扣款、不读取或输出真实密钥。真实收款需微信侧开通、发布道具和真机验收。


## 上线配置清单（2026-11-10 纠正版）

### A. 微信后台（外部开通，未完成）

1. 小程序主体/类目满足虚拟支付要求，开通虚拟支付。
2. 取得 OfferID、正式 AppKey；沙箱另配，生产只用正式。
3. 在「虚拟支付 → 道具管理」创建并发布道具，取得 `productId` 与价格（分）。
4. 配置「虚拟支付 → 基本配置 → 发货推送配置」：URL、Token、EncodingAESKey、加密方式、JSON/XML 格式，订阅 `xpay_goods_deliver_notify`、`xpay_refund_notify` 等。
5. iOS：配置小程序简称并开通 Apple IAP；真机微信需 8.0.68+。
6. 配置合法域名（小管家/汤圆服务地址、推送域名）。

### B. 小管家（通用 virtualpay）

1. 给小程序勾选 `virtualpay` 模块。
2. 管理员「虚拟支付」页配置：启用、`offer_id/app_key`、推送 Token/AESKey/加密方式、交付回调 URL、`delivery_app_id/delivery_secret`、IAP 能力。
3. 商品维护：价格 + 微信 productId + **不透明 `delivery_payload`**（合法 JSON 对象，≤16KB）。小管家不解释积分/账本/OCR/AI。
4. 草稿种子只按显式命令初始化；上架前核对 productId/价格/交付 JSON。
5. 推送入口：`/api/<app>/virtualpay/push`，或复用 `/api/<app>/seccheck/push`（验签成功后按 Event 分发）。
6. 交付 outbox 每分钟重试，管理端可手动重试；退款全额/部分均作为通用支付事实通知接入方。

### C. 汤圆记账

环境变量：

```text
BILLING_ENABLED=true
BILLING_XGJ_APP_SLUG=hehuojizhangzhushou
BILLING_XGJ_BASE_URL=https://<小管家>
BILLING_XGJ_SECRET=<与小管家 delivery_secret 一致>
BILLING_CLOCK_SKEW_SECONDS=300
BILLING_DEMO_LOGIN_ENABLED=false
```

商品 `delivery_payload` 由汤圆解释：

```json
{"credit_amount":600,"validity_days":365}
```

默认 100 积分/元；`credit_amount` 可含赠送，但仍只发普通积分。无效/缺失/<=0 由汤圆拒绝。

数据库新增：`billing_packages`、`billing_purchase_intents`、`billing_processed_events`、`billing_refunds`、`billing_capabilities`、`demo_accounts`，以及 `credit_lots.order_ref`、`users/tenants.is_demo` 列。**不新增 usage 表/模式/独立次数 cron**；购买积分到期复用现有 `credits_expire`。

初始化：平台管理员 `POST /api/platform/billing/packages/seed` 仅生成草稿示例；真实上架以汤圆从小管家同步的权威商品 + 汤圆解析的 `delivery_payload` 为准。价格/积分/上下架变化后前端刷新；下单/支付前服务端再次校验。

### D. 发布顺序与回滚

发布：微信后台道具/推送 → 小管家（关闭开关部署）→ 汤圆（`BILLING_ENABLED=false` 部署）→ 配置密钥 → 小管家启用 → 汤圆启用 → 真机小额验证。

回滚：

1. 小管家关闭 virtualpay 开关；汤圆设 `BILLING_ENABLED=false` 并重启。
2. 回滚代码；**不删除新增表**，保留订单/outbox/退款/积分批次用于对账。
3. 待投递 outbox 先查看/重试或人工补偿，不丢弃事件。
4. 已发积分不回滚；不删除 `credit_lots/credit_logs`，不制造负余额。
5. 退款只按原订单批次回收剩余；已消费人工核查；不扣其他批次。
6. 密钥泄露轮换 AppKey、delivery secret、推送 Token/AESKey，并重启。

### E. 上线前真机验证

- 安卓：下单 → 收银台 → 支付 → 推送/查单 → 汤圆积分到账 → 原积分扣费 → 订单状态。
- iOS：IAP 未开通不得拉起；开通后按 Apple 流程；微信 8.0.68+。
- 推送：GET echostr、明文/安全模式、错签名/错 appid 拒绝、重复只到账一次。
- 退款：全额回收原订单批次剩余；部分退款人工核查；退款后乱序发货不重发。
- 安全：无签名 xpay 不发货；HMAC 错误/过期拒绝；跨账本/跨用户/演示身份不可越权。

真机、微信后台、真实收款/退款、部署均未执行。
