# 虚拟支付与积分购买进度（2026-11-10 纠正版）

> 分支：`feature-20261109-virtual-payment`；两仓原有未提交改动保留，未提交旧改动。
> 规则：积分是唯一权益；小管家通用支付、汤圆积分解释；不使用 usage 模式/独立 OCR/AI 次数。

## 状态

| 阶段 | 状态 | 说明 |
|---|---|---|
| P0 契约/基线 | ✅ | 分支、协议、配置、测试基线 |
| P1 小管家通用模块 | ✅ | 商品/售价/`delivery_payload`/上下架/草稿/管理 UI/registry/purge |
| P2 支付与交付 | ✅（本地） | 双签名、HMAC intent、支付、验签、查单、outbox/重试、通用退款事实（全额/部分均通知） |
| P3 汤圆积分 | ✅（本地） | 本地意图、`credit_lots/tenant_credits/credit_logs` 发放/退款、原 OCR/AI 扣费、充值/订单/流水 UI |
| P4 体验与验收 | ✅（本地） | 演示隔离、示例 OCR/AI、能力测试、构建、文档 |

## 最终验证

| 命令 | 结果 |
|---|---|
| `fandianjizhang/server: go test -mod=vendor ./...` | PASS（独立临时 MySQL） |
| `xiaoguanjia: XIAOGUANJIA_TEST_DSN=<临时库> go test ./...` | PASS |
| `xiaoguanjia/web/admin: npm run build` | PASS |
| `uniapp: npm run type-check` | PASS |
| `uniapp: npm run test:billing-capability` | PASS（5/5） |
| `uniapp: npm run build:mp-weixin` | PASS |
| `uniapp: npm run build:h5` | PASS |
| gofmt / diff --check | 本次改动 Go 文件已格式化 |

临时库测试后已删除；未部署、未真实扣款、未读取真实密钥。

## 关键纠正

- 小管家移除 `target_tenant_id/target_user_id/credit_amount/OCRCount/AICount`；商品/订单/事件使用不透明 `delivery_payload`。
- 汤圆 `delivery_payload`：`{"credit_amount":600,"validity_days":365}`；本地意图固化 tenant/user/积分快照。
- 购买成功后写现有积分表；退款仅全额回收原订单批次剩余，部分退款人工核查。
- OCR/AI/上传/存储恢复原积分扣费；无 UsageGuard/mode/独立次数 cron。
- 退款明细 Amount 为正；已消费按 `credit_logs` action 过滤；锁顺序统一先 `tenant_credits`。
- 估算与真实扣费共用配置解析；0/非法/缺配置/溢出有测试。
- 商品卡按每包到账积分返回 OCR/AI/全链路参考次数（共享同一积分、不可叠加）；余额 0 时区分“积分不足”与“单价为 0/未配置”。

## 文档

- `virtual-payment-acceptance.md`：验收与 review-notes 纠正说明
- `virtual-payment-plan.md`：当前计划
- `virtual-payment-protocol.md`：通用支付 + 汤圆积分解释
- `virtual-payment-config.example.md`：配置模板
- `virtual-payment-launch.md`：上线/回滚
- `credits-billing-integration.md`：汤圆积分接入
- 小管家 `docs/virtualpay-module.md`：通用模块接入

## 未验证

微信开通/商品发布/推送配置、Android/iOS 真机、iOS IAP、真实查单/发货确认/退款、无 session_key 签名、部署/生产迁移/真实收款。
