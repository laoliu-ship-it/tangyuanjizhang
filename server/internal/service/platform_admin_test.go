package service_test

import (
	"context"
	"testing"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/service"
)

// fakeStatsRepo 内存版平台统计仓库，仅覆盖 ListUsers 用到的方法
type fakeStatsRepo struct {
	users       []*model.User
	lastTxTimes map[uint64]time.Time
}

func (r *fakeStatsRepo) CountUsers(ctx context.Context) (int64, error)        { return 0, nil }
func (r *fakeStatsRepo) CountTenants(ctx context.Context) (int64, error)      { return 0, nil }
func (r *fakeStatsRepo) CountTransactions(ctx context.Context) (int64, error) { return 0, nil }

func (r *fakeStatsRepo) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
	return r.users, int64(len(r.users)), nil
}

func (r *fakeStatsRepo) LastTransactionTimesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64]time.Time, error) {
	result := make(map[uint64]time.Time)
	for _, id := range userIDs {
		if t, ok := r.lastTxTimes[id]; ok {
			result[id] = t
		}
	}
	return result, nil
}

func (r *fakeStatsRepo) CountTransactionsByUserID(ctx context.Context, userID uint64) (int64, error) {
	return 0, nil
}
func (r *fakeStatsRepo) CountMediaByUserID(ctx context.Context, userID uint64) (int64, error) {
	return 0, nil
}
func (r *fakeStatsRepo) CountTenantsByUserID(ctx context.Context, userID uint64) (int64, error) {
	return 0, nil
}

func TestListUsers_活跃时间字段(t *testing.T) {
	loginAt := time.Date(2026, 7, 1, 10, 30, 0, 0, time.Local)
	txAt := time.Date(2026, 7, 3, 18, 5, 0, 0, time.Local)

	statsRepo := &fakeStatsRepo{
		users: []*model.User{
			{ID: 1, Username: "active", Email: "a@x.com", LastLoginAt: &loginAt, CreatedAt: time.Now()},
			{ID: 2, Username: "silent", Email: "b@x.com", LastLoginAt: nil, CreatedAt: time.Now()},
		},
		lastTxTimes: map[uint64]time.Time{1: txAt},
	}
	svc := service.NewPlatformAdminService(nil, statsRepo, nil, nil, nil, nil, nil)

	resp, err := svc.ListUsers(context.Background(), dto.PlatformUserFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListUsers 出错: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("期望 2 条记录，实际 %d", len(resp.Items))
	}

	active := resp.Items[0]
	if active.LastLoginAt != "2026-07-01 10:30:00" {
		t.Errorf("活跃用户最近登录时间错误: %q", active.LastLoginAt)
	}
	if active.LastTransactionAt != "2026-07-03 18:05:00" {
		t.Errorf("活跃用户最近记账时间错误: %q", active.LastTransactionAt)
	}

	silent := resp.Items[1]
	if silent.LastLoginAt != "" {
		t.Errorf("从未登录用户应返回空串，实际: %q", silent.LastLoginAt)
	}
	if silent.LastTransactionAt != "" {
		t.Errorf("从未记账用户应返回空串，实际: %q", silent.LastTransactionAt)
	}
}
