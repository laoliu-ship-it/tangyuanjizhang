package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

type StatisticsService interface {
	Daily(ctx context.Context, tenantID uint64, date string) (*dto.DailyStatResp, error)
	Monthly(ctx context.Context, tenantID uint64, year, month int) (*dto.MonthlyStatResp, error)
	Yearly(ctx context.Context, tenantID uint64, year int) (*dto.YearlyStatResp, error)
	Range(ctx context.Context, tenantID uint64, start, end string) (*dto.RangeStatResp, error)
	RefreshMerchants(ctx context.Context, tenantID uint64, view string, year, month int, start, end string) (*dto.MerchantStatsResp, error)
	RunDailyJob(ctx context.Context) error
}

type statisticsService struct {
	transactionRepo repo.TransactionRepo
	statsCacheRepo  repo.StatsCacheRepo
	tenantRepo      repo.TenantRepo
}

func NewStatisticsService(transactionRepo repo.TransactionRepo, statsCacheRepo repo.StatsCacheRepo, tenantRepo repo.TenantRepo) StatisticsService {
	return &statisticsService{
		transactionRepo: transactionRepo,
		statsCacheRepo:  statsCacheRepo,
		tenantRepo:      tenantRepo,
	}
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

	topMerchants, _ := s.getMerchantStatsFromCache(ctx, tenantID, "monthly", fmt.Sprintf("%d-%02d", year, month), func() ([]*dto.MerchantStat, error) {
		return s.transactionRepo.MonthlyMerchantStats(ctx, tenantID, year, month)
	})

	return &dto.MonthlyStatResp{
		Year:  year,
		Month: month,
		Total: dto.StatisticsResp{
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
			NetAmount:    totalIncome - totalExpense,
		},
		Daily:        days,
		Categories:   categories,
		TopMerchants: topMerchants,
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

	topMerchants, _ := s.getMerchantStatsFromCache(ctx, tenantID, "yearly", fmt.Sprintf("%d", year), func() ([]*dto.MerchantStat, error) {
		return s.transactionRepo.YearlyMerchantStats(ctx, tenantID, year)
	})

	return &dto.YearlyStatResp{
		Year: year,
		Total: dto.StatisticsResp{
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
			NetAmount:    totalIncome - totalExpense,
		},
		Monthly:      monthly,
		Categories:   categories,
		TopMerchants: topMerchants,
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

	topMerchants, err := s.transactionRepo.RangeMerchantStats(ctx, tenantID, start, end)
	if err != nil {
		topMerchants = []*dto.MerchantStat{}
	}

	return &dto.RangeStatResp{
		Total:        *total,
		Daily:        daily,
		Categories:   categories,
		TopMerchants: topMerchants,
	}, nil
}

// RefreshMerchants 实时计算商户统计并更新缓存（范围视图不缓存）
func (s *statisticsService) RefreshMerchants(ctx context.Context, tenantID uint64, view string, year, month int, start, end string) (*dto.MerchantStatsResp, error) {
	var merchants []*dto.MerchantStat
	var err error

	switch view {
	case "monthly":
		merchants, err = s.transactionRepo.MonthlyMerchantStats(ctx, tenantID, year, month)
		if err != nil {
			return nil, err
		}
		_ = s.saveMerchantCache(ctx, tenantID, "monthly", fmt.Sprintf("%d-%02d", year, month), merchants)
	case "yearly":
		merchants, err = s.transactionRepo.YearlyMerchantStats(ctx, tenantID, year)
		if err != nil {
			return nil, err
		}
		_ = s.saveMerchantCache(ctx, tenantID, "yearly", fmt.Sprintf("%d", year), merchants)
	default: // range — 实时，不缓存
		merchants, err = s.transactionRepo.RangeMerchantStats(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
	}

	if merchants == nil {
		merchants = []*dto.MerchantStat{}
	}
	return &dto.MerchantStatsResp{
		TopMerchants: merchants,
		ComputedAt:   time.Now().Format(time.RFC3339),
		FromCache:    false,
	}, nil
}

// RunDailyJob 凌晨3点计划任务：为所有租户预计算当月+当年商户统计
func (s *statisticsService) RunDailyJob(ctx context.Context) error {
	now := time.Now()
	year, month := now.Year(), int(now.Month())
	monthKey := fmt.Sprintf("%d-%02d", year, month)
	yearKey := fmt.Sprintf("%d", year)

	tenants, err := s.tenantRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list tenants: %w", err)
	}

	log.Printf("[stats job] 开始预计算，租户数: %d, 周期: %s / %s", len(tenants), monthKey, yearKey)

	for _, t := range tenants {
		tid := t.ID

		if ms, err := s.transactionRepo.MonthlyMerchantStats(ctx, tid, year, month); err == nil {
			_ = s.saveMerchantCache(ctx, tid, "monthly", monthKey, ms)
		} else {
			log.Printf("[stats job] tenant %d monthly error: %v", tid, err)
		}

		if ys, err := s.transactionRepo.YearlyMerchantStats(ctx, tid, year); err == nil {
			_ = s.saveMerchantCache(ctx, tid, "yearly", yearKey, ys)
		} else {
			log.Printf("[stats job] tenant %d yearly error: %v", tid, err)
		}
	}

	log.Printf("[stats job] 预计算完成")
	return nil
}

// getMerchantStatsFromCache 从缓存读取，不存在则实时计算并写缓存
func (s *statisticsService) getMerchantStatsFromCache(ctx context.Context, tenantID uint64, cacheType, periodKey string, live func() ([]*dto.MerchantStat, error)) ([]*dto.MerchantStat, error) {
	cached, err := s.statsCacheRepo.Get(ctx, tenantID, cacheType, periodKey)
	if err == nil && cached != nil {
		var merchants []*dto.MerchantStat
		if jsonErr := json.Unmarshal([]byte(cached.Data), &merchants); jsonErr == nil {
			return merchants, nil
		}
	}

	merchants, err := live()
	if err != nil {
		return []*dto.MerchantStat{}, nil
	}
	if merchants == nil {
		merchants = []*dto.MerchantStat{}
	}
	_ = s.saveMerchantCache(ctx, tenantID, cacheType, periodKey, merchants)
	return merchants, nil
}

func (s *statisticsService) saveMerchantCache(ctx context.Context, tenantID uint64, cacheType, periodKey string, merchants []*dto.MerchantStat) error {
	data, err := json.Marshal(merchants)
	if err != nil {
		return err
	}
	return s.statsCacheRepo.Upsert(ctx, &model.StatsCache{
		TenantID:   tenantID,
		CacheType:  cacheType,
		PeriodKey:  periodKey,
		Data:       string(data),
		ComputedAt: time.Now(),
	})
}
