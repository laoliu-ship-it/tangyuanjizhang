package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// DemoService 只创建一个固定演示账号/账本，并通过 DemoAccount 关联记录识别。
// 绝不按 username 复用任何现有用户，避免匿名体验进入真实账本。
type DemoService interface {
	Enabled() bool
	Ensure(ctx context.Context) (*model.User, error)
}

type demoService struct {
	demoRepo     repo.DemoRepo
	userRepo     repo.UserRepo
	tenantRepo   repo.TenantRepo
	memberRepo   repo.TenantMemberRepo
	categoryRepo repo.CategoryRepo
	txRepo       repo.TransactionRepo
	cfg          config.BillingConfig
}

func NewDemoService(demoRepo repo.DemoRepo, userRepo repo.UserRepo, tenantRepo repo.TenantRepo, memberRepo repo.TenantMemberRepo, categoryRepo repo.CategoryRepo, txRepo repo.TransactionRepo, cfg config.BillingConfig) DemoService {
	return &demoService{demoRepo: demoRepo, userRepo: userRepo, tenantRepo: tenantRepo, memberRepo: memberRepo, categoryRepo: categoryRepo, txRepo: txRepo, cfg: cfg}
}

func (s *demoService) Enabled() bool { return s.cfg.DemoLoginEnabled }

func (s *demoService) Ensure(ctx context.Context) (*model.User, error) {
	var user *model.User
	err := s.demoRepo.WithInitLock(ctx, func() error {
		u, e := s.ensureLocked(ctx)
		user = u
		return e
	})
	return user, err
}

func (s *demoService) ensureLocked(ctx context.Context) (*model.User, error) {
	account, err := s.demoRepo.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	if account != nil {
		user, err := s.userRepo.GetByID(ctx, account.UserID)
		if err != nil {
			return nil, err
		}
		tenant, err := s.tenantRepo.GetByID(ctx, account.TenantID)
		if err != nil {
			return nil, err
		}
		if user == nil || !user.IsDemo || tenant == nil || !tenant.IsDemo || tenant.OwnerID != user.ID {
			return nil, fmt.Errorf("演示关联记录指向非演示账号/账本，拒绝复用")
		}
		return user, nil
	}

	// 预留名冲突时改用带随机后缀的保留名，绝不复用已存在的普通用户。
	username, user, err := s.reserveDemoUsername(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		hash, herr := bcrypt.GenerateFromPassword([]byte(randomDemoPassword()), bcrypt.DefaultCost)
		if herr != nil {
			return nil, herr
		}
		user = &model.User{
			Username: username, Email: "demo+" + username + "@demo.local",
			PasswordHash: string(hash), IsDemo: true,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("create demo user: %w", err)
		}
	}

	tenant := &model.Tenant{Name: "演示账本（只读）", OwnerID: user.ID, BookType: "restaurant", IsDemo: true}
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}
	if err := s.memberRepo.Add(ctx, &model.TenantMember{TenantID: tenant.ID, UserID: user.ID, Role: "admin", JoinedAt: time.Now()}); err != nil {
		return nil, err
	}
	createDefaultCategories(ctx, s.categoryRepo, tenant.ID, "restaurant")
	s.seedTransactions(ctx, tenant.ID, user.ID)

	account = &model.DemoAccount{UserID: user.ID, TenantID: tenant.ID}
	if err := s.demoRepo.CreateAccount(ctx, account); err != nil {
		// 并发初始化：唯一索引冲突后读取已存在的关联记录。
		if existing, gerr := s.demoRepo.GetAccount(ctx); gerr == nil && existing != nil {
			return s.userRepo.GetByID(ctx, existing.UserID)
		}
		return nil, err
	}
	return user, nil
}

// reserveDemoUsername 返回一个未被占用的保留名；若已存在系统演示用户则返回它。
func (s *demoService) reserveDemoUsername(ctx context.Context) (string, *model.User, error) {
	for i := 0; i < 5; i++ {
		name := "sys_demo"
		if i > 0 {
			name = "sys_demo_" + randomSuffix(4)
		}
		u, err := s.userRepo.GetByUsername(ctx, name)
		if err != nil {
			return "", nil, err
		}
		if u == nil {
			return name, nil, nil
		}
		if u.IsDemo {
			return name, u, nil
		}
		// 普通用户占用保留名：不重用，换随机后缀。
	}
	return "sys_demo_" + randomSuffix(8), nil, nil
}

func (s *demoService) seedTransactions(ctx context.Context, tenantID, userID uint64) {
	_, total, err := s.txRepo.List(ctx, tenantID, dto.TransactionFilter{Page: 1, PageSize: 1})
	if err != nil || total > 0 {
		return
	}
	cats, err := s.categoryRepo.List(ctx, tenantID)
	if err != nil || len(cats) == 0 {
		return
	}
	var expenseCat, incomeCat uint64
	for _, c := range cats {
		if c.Type == "expense" && expenseCat == 0 {
			expenseCat = c.ID
		}
		if c.Type == "income" && incomeCat == 0 {
			incomeCat = c.ID
		}
	}
	if expenseCat == 0 {
		expenseCat = cats[0].ID
	}
	now := time.Now()
	rows := []struct {
		typ    string
		amount float64
		cat    uint64
		note   string
		days   int
	}{
		{"expense", 38.50, expenseCat, "演示：午餐", 0},
		{"expense", 126.00, expenseCat, "演示：晚餐", 1},
		{"expense", 12.00, expenseCat, "演示：咖啡", 2},
		{"income", 8600.00, incomeCat, "演示：工资", 3},
		{"expense", 268.00, expenseCat, "演示：采购", 4},
		{"expense", 45.00, expenseCat, "演示：打车", 5},
	}
	txs := make([]*model.Transaction, 0, len(rows))
	for _, r := range rows {
		txs = append(txs, &model.Transaction{
			TenantID: tenantID, UserID: userID, Type: r.typ, Amount: r.amount, CategoryID: r.cat,
			TransactionDate: now.AddDate(0, 0, -r.days).Format("2006-01-02 15:04:05"),
			Note:            r.note,
		})
	}
	if err := s.txRepo.BatchCreate(ctx, txs); err != nil {
		log.Printf("[demo] 初始化演示交易失败: %v", err)
	}
}

func randomDemoPassword() string {
	var b [24]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func randomSuffix(n int) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])[:n]
}
