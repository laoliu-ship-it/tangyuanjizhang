package service

import (
	"context"
	"errors"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

var ErrTransactionNotFound = errors.New("交易记录不存在")

type TransactionService interface {
	Create(ctx context.Context, tenantID, userID uint64, req dto.CreateTransactionReq) (*model.Transaction, error)
	BatchCreate(ctx context.Context, tenantID, userID uint64, req dto.BatchCreateTransactionReq) ([]*model.Transaction, error)
	Update(ctx context.Context, id, tenantID uint64, req dto.UpdateTransactionReq) (*model.Transaction, error)
	Delete(ctx context.Context, id, tenantID uint64) error
	GetByID(ctx context.Context, id, tenantID uint64) (*model.Transaction, error)
	List(ctx context.Context, tenantID uint64, filter dto.TransactionFilter) (*dto.TransactionListResp, error)
}

type transactionService struct {
	transactionRepo repo.TransactionRepo
	categoryRepo    repo.CategoryRepo
	merchantRepo    repo.MerchantRepo
}

func NewTransactionService(
	transactionRepo repo.TransactionRepo,
	categoryRepo repo.CategoryRepo,
	merchantRepo repo.MerchantRepo,
) TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
		merchantRepo:    merchantRepo,
	}
}

func (s *transactionService) Create(ctx context.Context, tenantID, userID uint64, req dto.CreateTransactionReq) (*model.Transaction, error) {
	// 验证分类属于该租户
	cat, err := s.categoryRepo.GetByID(ctx, req.CategoryID, tenantID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrCategoryNotFound
	}

	// 自动创建商户：如果没指定 merchant_id 但有 merchant_name
	if req.MerchantID == 0 && req.MerchantName != "" {
		m, err := s.merchantRepo.GetByName(ctx, req.MerchantName, tenantID)
		if err != nil {
			return nil, err
		}
		if m != nil {
			req.MerchantID = m.ID
		} else {
			newM := &model.Merchant{TenantID: tenantID, Name: req.MerchantName}
			if err := s.merchantRepo.Create(ctx, newM); err != nil {
				return nil, err
			}
			req.MerchantID = newM.ID
		}
	}

	tx := &model.Transaction{
		TenantID:        tenantID,
		UserID:          userID,
		Type:            req.Type,
		Amount:          req.Amount,
		CategoryID:      req.CategoryID,
		MerchantID:      req.MerchantID,
		TransactionDate: req.TransactionDate,
		Note:            req.Note,
	}
	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return nil, err
	}
	if req.ImagePath != "" {
		if err := s.transactionRepo.SaveImage(ctx, tx.ID, req.ImagePath, req.OCRAmount, req.OCRDate, req.OCRMerchant, req.OCRRawTexts); err != nil {
			return nil, err
		}
		tx.Images = []*model.TransactionImage{{TransactionID: tx.ID, ImagePath: req.ImagePath, OCRAmount: req.OCRAmount, OCRDate: req.OCRDate, OCRMerchant: req.OCRMerchant, OCRRawTexts: req.OCRRawTexts}}
	}
	tx.Category = cat
	return tx, nil
}

func (s *transactionService) BatchCreate(ctx context.Context, tenantID, userID uint64, req dto.BatchCreateTransactionReq) ([]*model.Transaction, error) {
	txs := make([]*model.Transaction, 0, len(req.Transactions))
	for _, item := range req.Transactions {
		// 验证分类
		cat, err := s.categoryRepo.GetByID(ctx, item.CategoryID, tenantID)
		if err != nil {
			return nil, err
		}
		if cat == nil {
			return nil, ErrCategoryNotFound
		}
		// 自动创建商户
		if item.MerchantID == 0 && item.MerchantName != "" {
			m, err := s.merchantRepo.GetByName(ctx, item.MerchantName, tenantID)
			if err != nil {
				return nil, err
			}
			if m != nil {
				item.MerchantID = m.ID
			} else {
				newM := &model.Merchant{TenantID: tenantID, Name: item.MerchantName}
				if err := s.merchantRepo.Create(ctx, newM); err != nil {
					return nil, err
				}
				item.MerchantID = newM.ID
			}
		}
		txs = append(txs, &model.Transaction{
			TenantID:        tenantID,
			UserID:          userID,
			Type:            item.Type,
			Amount:          item.Amount,
			CategoryID:      item.CategoryID,
			MerchantID:      item.MerchantID,
			TransactionDate: item.TransactionDate,
			Note:            item.Note,
		})
	}
	if err := s.transactionRepo.BatchCreate(ctx, txs); err != nil {
		return nil, err
	}
	return txs, nil
}

func (s *transactionService) Update(ctx context.Context, id, tenantID uint64, req dto.UpdateTransactionReq) (*model.Transaction, error) {
	tx, err := s.transactionRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}

	// 验证分类
	cat, err := s.categoryRepo.GetByID(ctx, req.CategoryID, tenantID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrCategoryNotFound
	}

	tx.Type = req.Type
	tx.Amount = req.Amount
	tx.CategoryID = req.CategoryID
	tx.MerchantID = req.MerchantID
	tx.TransactionDate = req.TransactionDate
	tx.Note = req.Note

	if err := s.transactionRepo.Update(ctx, tx); err != nil {
		return nil, err
	}
	if err := s.transactionRepo.SaveImage(ctx, tx.ID, req.ImagePath, req.OCRAmount, req.OCRDate, req.OCRMerchant, req.OCRRawTexts); err != nil {
		return nil, err
	}
	if req.ImagePath != "" {
		tx.Images = []*model.TransactionImage{{TransactionID: tx.ID, ImagePath: req.ImagePath, OCRAmount: req.OCRAmount, OCRDate: req.OCRDate, OCRMerchant: req.OCRMerchant, OCRRawTexts: req.OCRRawTexts}}
	} else {
		tx.Images = nil
	}
	tx.Category = cat
	return tx, nil
}

func (s *transactionService) Delete(ctx context.Context, id, tenantID uint64) error {
	tx, err := s.transactionRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if tx == nil {
		return ErrTransactionNotFound
	}
	return s.transactionRepo.Delete(ctx, id, tenantID)
}

func (s *transactionService) GetByID(ctx context.Context, id, tenantID uint64) (*model.Transaction, error) {
	tx, err := s.transactionRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}
	return tx, nil
}

func (s *transactionService) List(ctx context.Context, tenantID uint64, filter dto.TransactionFilter) (*dto.TransactionListResp, error) {
	filter.Normalize()
	txs, total, err := s.transactionRepo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.TransactionResp, 0, len(txs))
	for _, tx := range txs {
		item := &dto.TransactionResp{
			ID:              tx.ID,
			TenantID:        tx.TenantID,
			UserID:          tx.UserID,
			Type:            tx.Type,
			Amount:          tx.Amount,
			CategoryID:      tx.CategoryID,
			MerchantID:      tx.MerchantID,
			TransactionDate: tx.TransactionDate,
			Note:            tx.Note,
			CreatedAt:       tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if tx.Category != nil {
			item.CategoryName = tx.Category.Name
		}
		if tx.Merchant != nil {
			item.MerchantName = tx.Merchant.Name
		}
		if len(tx.Images) > 0 {
			imgs := make([]*dto.TransactionImageResp, 0, len(tx.Images))
			for _, img := range tx.Images {
				imgs = append(imgs, &dto.TransactionImageResp{
					ImagePath:   img.ImagePath,
					OCRAmount:   img.OCRAmount,
					OCRDate:     img.OCRDate,
					OCRMerchant: img.OCRMerchant,
					OCRRawTexts: img.OCRRawTexts,
				})
			}
			item.Images = imgs
		}
		items = append(items, item)
	}

	return &dto.TransactionListResp{
		Total: total,
		Page:  filter.Page,
		Size:  filter.PageSize,
		Items: items,
	}, nil
}
