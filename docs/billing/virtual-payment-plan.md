# 虚拟道具支付与积分购买开发计划（2026-11-10 纠正版）

> 原始计划中的“独立 OCR/AI 次数账户、usage 模式、UsageGuard”已被 `credits-payment-correction.md` 覆盖。
> 当前规则：**积分是唯一实际权益**；小管家是通用虚拟道具支付模块，汤圆是接入方；二者解耦。

## 目标

- 汤圆记账（小程序 slug：`hehuojizhangzhushou`）通过通用虚拟道具支付购买**普通积分**。
- 小管家提供可复用的商品/售价/支付/验签/查单/通用发货退款通知；不理解汤圆积分、账本、OCR/AI。
- 汤圆负责购买意图（tenant/user）、积分换算、`credit_lots` 发放、原积分扣费、退款回收、前端展示。
- 保留体验账号、支付安全、消息验签、outbox/重试、权威商品同步、终端能力控制。

## 当前业务规则

- 商品：售价（整数分）、到账积分由汤圆在 `delivery_payload` 中定义；默认 100 积分/元，后台可配置赠送。
- 商品在小管家只保存不透明 JSON；汤圆解析 `{"credit_amount":600,"validity_days":365}`。
- 积分是唯一余额；OCR/AI/上传/存储继续按现有 `credits_per_*` 扣减，不新建独立次数余额，没有 mode 切换。
- 购买成功后只写现有 `tenant_credits` / `credit_lots` / `credit_logs`；发放与幂等事件同事务。
- 退款只回收原订单购买批次剩余；已消费部分人工核查；不扣其他批次、不制造负余额。
- OCR/AI 次数仅作为互斥估算展示；单价来自当前配置，0/非法/缺配置不虚构。
- 体验账号只读，不可支付/充值/消耗真实积分。

## 系统边界

- 小管家：`virtualpay` 专有模块；商品/意图/订单/支付配置/验签/outbox/通用退款事实；不含业务字段。
- 汤圆：`billing_*` 本地意图/事件/退款/能力缓存 + 现有积分表；`PaymentGateway` 窄接口；`xgjclient` 仅 HTTP 适配器。
- 服务间仍使用 HTTP + HMAC + 时间窗 + nonce；不引入中间件/插件平台/共享数据库。

## 阶段

| 阶段 | 交付 | 状态 |
|---|---|---|
| P0 | 两仓基线、分支、协议/配置/状态机 | 完成 |
| P1 | 小管家通用商品/配置/管理 UI/草稿种子 | 完成 |
| P2 | 双签名、意图、支付、消息验签、查单、outbox、通用退款事件 | 完成（本地） |
| P3 | 汤圆本地意图、积分批次发放/退款、原扣费保留、购买/订单/积分 UI | 完成（本地） |
| P4 | 演示账号隔离、示例 OCR/AI、构建/验收/上线说明 | 完成（本地） |

## 验收命令

```text
fandianjizhang/server: go test -mod=vendor ./...
xiaoguanjia: XIAOGUANJIA_TEST_DSN=<独立临时库> go test ./...
xiaoguanjia/web/admin: npm run build
uniapp: npm run type-check; npm run test:billing-capability; npm run build:mp-weixin; npm run build:h5
```

不部署、不操作共享库、不真实扣款。真实微信开通/真机支付仍为未验证项，详见验收报告。
