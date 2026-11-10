package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fandianjizhang/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type billingRepo struct {
	db *gorm.DB
}

func NewBillingRepo(db *gorm.DB) BillingRepo { return &billingRepo{db: db} }

func (r *billingRepo) ListBillingPackages(ctx context.Context, appSlug string, listedOnly bool) ([]*model.BillingPackage, error) {
	q := r.db.WithContext(ctx).Where("app_slug = ?", appSlug)
	if listedOnly {
		q = q.Where("status = ?", "listed")
	}
	var items []*model.BillingPackage
	err := q.Order("sort ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *billingRepo) UpsertBillingPackages(ctx context.Context, packages []*model.BillingPackage) error {
	if len(packages) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "app_slug"}, {Name: "sku_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "wx_product_id", "price_fen", "credit_amount", "validity_days",
			"status", "sort", "synced_at", "updated_at",
		}),
	}).Create(&packages).Error
}

func (r *billingRepo) GetBillingCapability(ctx context.Context, appSlug string) (*model.BillingCapability, error) {
	var c model.BillingCapability
	err := r.db.WithContext(ctx).Where("app_slug = ?", appSlug).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.BillingCapability{AppSlug: appSlug}, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *billingRepo) SaveBillingCapability(ctx context.Context, cap *model.BillingCapability) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "app_slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"iap_enabled", "env", "synced_at", "updated_at"}),
	}).Create(cap).Error
}

func (r *billingRepo) CreateBillingIntent(ctx context.Context, intent *model.BillingPurchaseIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *billingRepo) GetBillingIntentByRef(ctx context.Context, tenantID uint64, ref string) (*model.BillingPurchaseIntent, error) {
	var i model.BillingPurchaseIntent
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND business_ref = ?", tenantID, ref).First(&i).Error
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *billingRepo) ListBillingIntents(ctx context.Context, tenantID uint64, page, size int) ([]*model.BillingPurchaseIntent, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.BillingPurchaseIntent{}).Where("tenant_id = ?", tenantID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*model.BillingPurchaseIntent
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (r *billingRepo) UpdateBillingIntent(ctx context.Context, intent *model.BillingPurchaseIntent) error {
	return r.db.WithContext(ctx).Save(intent).Error
}

// ProcessBillingDelivery 在单事务内：事件幂等 → 校验本地意图快照 → 创建现有积分批次 →
// 更新 tenant_credits 缓存与 credit_logs → 确认交付。重复/退款后晚到发货不会再次发放。
func (r *billingRepo) ProcessBillingDelivery(ctx context.Context, event *model.BillingDeliveryEvent) (bool, error) {
	if event == nil || event.EventKey == "" {
		return false, fmt.Errorf("缺少事件键")
	}
	already := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.BillingProcessedEvent{
			EventKey: event.EventKey, EventType: model.BillingProcessedDeliver,
			OrderRef: event.BusinessReference, Result: "done",
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			already = true
			return nil
		}

		var intent model.BillingPurchaseIntent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("business_ref = ?", event.BusinessReference).First(&intent).Error; err != nil {
			return fmt.Errorf("购买意图不存在: %w", err)
		}
		if intent.SKUCode != event.SKUCode || intent.AmountFen != event.AmountFen ||
			intent.CreditAmount != event.CreditAmount {
			return fmt.Errorf("发货事件与本地购买意图/积分快照不匹配")
		}
		if intent.Status == model.BillingIntentRefunded {
			return nil
		}
		var refundCount int64
		if err := tx.Model(&model.BillingRefund{}).Where("order_ref = ?", event.BusinessReference).Count(&refundCount).Error; err != nil {
			return err
		}
		if refundCount > 0 || intent.Status == model.BillingIntentDelivered {
			return nil
		}
		// 有效期使用本地意图快照；事件只允许与快照一致，不能用事件覆盖。
		validity := intent.ValidityDays
		if validity <= 0 {
			return fmt.Errorf("本地购买意图缺少积分有效期")
		}
		if event.ValidityDays > 0 && event.ValidityDays != validity {
			return fmt.Errorf("发货事件有效期与本地意图快照不一致")
		}

		credits, err := lockCreditsRowTx(tx, intent.TenantID)
		if err != nil {
			return err
		}
		// 与 Deduct 相同顺序：先锁 tenant_credits，再惰性清理到期批次，最后汇总余额。
		now := time.Now()
		if _, err := expireDueLotsTx(tx, intent.TenantID, now); err != nil {
			return err
		}
		lot := &model.CreditLot{
			TenantID: intent.TenantID, Amount: intent.CreditAmount, Remaining: intent.CreditAmount,
			Source: "purchase", OrderRef: event.BusinessReference,
			ExpiresAt: now.AddDate(0, 0, validity),
		}
		if err := tx.Create(lot).Error; err != nil {
			return err
		}
		balance, err := sumLotsRemainingTx(tx, intent.TenantID)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.TenantCredits{}).Where("id = ?", credits.ID).
			Update("balance", balance).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.CreditLog{
			TenantID: intent.TenantID, Amount: intent.CreditAmount, Action: "purchase",
			BalanceAfter: balance, RefType: "credit_lot", RefID: lot.ID,
			Note: "购买积分到账 " + event.BusinessReference,
		}).Error; err != nil {
			return err
		}
		intent.Status = model.BillingIntentDelivered
		return tx.Save(&intent).Error
	})
	return already, err
}

// ProcessBillingRefund 只回收原订单 purchase 批次剩余；已消耗部分记录人工核查。
func (r *billingRepo) ProcessBillingRefund(ctx context.Context, event *model.BillingDeliveryEvent) (bool, error) {
	if event == nil || event.EventKey == "" {
		return false, fmt.Errorf("缺少事件键")
	}
	already := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.BillingProcessedEvent{
			EventKey: event.EventKey, EventType: model.BillingProcessedRefund,
			OrderRef: event.BusinessReference, Result: "done",
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			already = true
			return nil
		}
		var intent model.BillingPurchaseIntent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("business_ref = ?", event.BusinessReference).First(&intent).Error; err != nil {
			return fmt.Errorf("购买意图不存在: %w", err)
		}
		// 只有退款金额明确等于本地订单金额才自动回收原批次；部分/缺金额只记录人工核查。
		fullRefund := event.RefundFeeFen > 0 && event.RefundFeeFen == intent.AmountFen
		if !fullRefund {
			refund := &model.BillingRefund{
				TenantID: intent.TenantID, RefundKey: event.EventKey, OrderRef: event.BusinessReference,
				MchRefundID: event.MchRefundID, Status: "human_review", RefundFeeFen: event.RefundFeeFen,
				ReclaimedCredits: 0, ConsumedCredits: 0,
				Reason: "部分退款或退款金额缺失，人工核查",
			}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "refund_key"}},
				DoUpdates: clause.AssignmentColumns([]string{"status", "reclaimed_credits", "consumed_credits", "refund_fee_fen", "reason", "updated_at"}),
			}).Create(refund).Error
		}

		// 与 Deduct 相同顺序：先锁 tenant_credits，再惰性清理到期批次，最后锁批次。
		if _, err := lockCreditsRowTx(tx, intent.TenantID); err != nil {
			return err
		}
		now := time.Now()
		if _, err := expireDueLotsTx(tx, intent.TenantID, now); err != nil {
			return err
		}
		var lots []model.CreditLot
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND source = ? AND order_ref = ?", intent.TenantID, "purchase", event.BusinessReference).
			Find(&lots).Error; err != nil {
			return err
		}
		var reclaimed, consumed int64
		for _, lot := range lots {
			reclaimed += lot.Remaining
			// 已消费从既有扣费明细统计；排除退款/到期冲正，自然到期不能算作已消费。
			var used int64
			if err := tx.Model(&model.CreditLogDetail{}).
				Joins("JOIN credit_logs ON credit_logs.id = credit_log_details.log_id").
				Where("credit_log_details.lot_id = ? AND credit_logs.amount < 0 AND credit_logs.action NOT IN ?",
					lot.ID, []string{"refund", "expire"}).
				Select("COALESCE(SUM(credit_log_details.amount),0)").Scan(&used).Error; err != nil {
				return err
			}
			consumed += used
			if err := tx.Model(&model.CreditLot{}).Where("id = ?", lot.ID).Update("remaining", 0).Error; err != nil {
				return err
			}
		}
		balance, err := sumLotsRemainingTx(tx, intent.TenantID)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.TenantCredits{}).Where("tenant_id = ?", intent.TenantID).
			Update("balance", balance).Error; err != nil {
			return err
		}
		if reclaimed > 0 {
			refundLog := &model.CreditLog{
				TenantID: intent.TenantID, Amount: -reclaimed, Action: "refund",
				BalanceAfter: balance, RefType: "billing_refund",
				Note: "退款回收 " + event.BusinessReference,
			}
			if err := tx.Create(refundLog).Error; err != nil {
				return err
			}
			// 退款流水关联回收批次，便于对账；沿用模型约定：明细 Amount 始终为正数扣减。
			for _, lot := range lots {
				if lot.Remaining <= 0 {
					continue
				}
				if err := tx.Create(&model.CreditLogDetail{
					LogID: refundLog.ID, LotID: lot.ID, Amount: lot.Remaining,
				}).Error; err != nil {
					return err
				}
			}
		}
		refund := &model.BillingRefund{
			TenantID: intent.TenantID, RefundKey: event.EventKey, OrderRef: event.BusinessReference,
			MchRefundID: event.MchRefundID, Status: "refunded", RefundFeeFen: event.RefundFeeFen,
			ReclaimedCredits: reclaimed, ConsumedCredits: consumed, Reason: "退款成功通知",
		}
		if consumed > 0 {
			refund.Status = "human_review"
			refund.Reason = "部分积分已消耗，需人工核查是否追偿"
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "refund_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "reclaimed_credits", "consumed_credits", "refund_fee_fen", "reason", "updated_at"}),
		}).Create(refund).Error; err != nil {
			return err
		}
		intent.Status = model.BillingIntentRefunded
		return tx.Save(&intent).Error
	})
	return already, err
}
