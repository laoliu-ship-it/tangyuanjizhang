package service

import (
	"context"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/repo"
)

type StatisticsService interface {
	Daily(ctx context.Context, tenantID uint64, date string) (*dto.DailyStatResp, error)
	Monthly(ctx context.Context, tenantID uint64, year, month int) (*dto.MonthlyStatResp, error)
	Yearly(ctx context.Context, tenantID uint64, year int) (*dto.YearlyStatResp, error)
	Range(ctx context.Context, tenantID uint64, start, end string) (*dto.RangeStatResp, error)
}

type statisticsService struct {
	transactionRepo repo.TransactionRepo
}

func NewStatisticsService(transactionRepo repo.TransactionRepo) StatisticsService {
	return &statisticsService{transactionRepo: transactionRepo}
}

func (s *statisticsService) Daily(ctx context.Context, tenantID uint64, date string) (*dto.DailyStatResp, error) {
	return s.transactionRepo.DailyStats(ctx, tenantID, date)
}

func (s *statisticsService) Monthly(ctx context.Context, tenantID uint64, year, month int) (*dto.MonthlyStatResp, error) {
	days, err := s.transactionRepo.MonthlyStats(ctx, tenantID, year, month)
	if err != nil {
		return nil, err
	}

	categories, err := s.transactionRepo.MonthlyCategoryStats(ctx, tenantID, year, month)
	if err != nil {
		return nil, err
	}

	var totalIncome, totalExpense float64
	for _, d := range days {
		totalIncome += d.TotalIncome
		totalExpense += d.TotalExpense
	}

	return &dto.MonthlyStatResp{
		Year:  year,
		Month: month,
		Total: dto.StatisticsResp{
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
			NetAmount:    totalIncome - totalExpense,
		},
		Daily:      days,
		Categories: categories,
	}, nil
}

func (s *statisticsService) Yearly(ctx context.Context, tenantID uint64, year int) (*dto.YearlyStatResp, error) {
	monthly, err := s.transactionRepo.YearlyStats(ctx, tenantID, year)
	if err != nil {
		return nil, err
	}

	categories, err := s.transactionRepo.YearlyCategoryStats(ctx, tenantID, year)
	if err != nil {
		return nil, err
	}

	var totalIncome, totalExpense float64
	for _, m := range monthly {
		totalIncome += m.TotalIncome
		totalExpense += m.TotalExpense
	}

	return &dto.YearlyStatResp{
		Year:  year,
		Total: dto.StatisticsResp{
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
			NetAmount:    totalIncome - totalExpense,
		},
		Monthly:    monthly,
		Categories: categories,
	}, nil
}

func (s *statisticsService) Range(ctx context.Context, tenantID uint64, start, end string) (*dto.RangeStatResp, error) {
	total, err := s.transactionRepo.RangeStats(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	daily, err := s.transactionRepo.RangeDailyStats(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	categories, err := s.transactionRepo.RangeCategoryStats(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	return &dto.RangeStatResp{
		Total:      *total,
		Daily:      daily,
		Categories: categories,
	}, nil
}
