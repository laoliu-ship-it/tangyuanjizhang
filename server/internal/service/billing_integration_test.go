package service_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("FANDIAN_TEST_DSN_SERVICE")
	if dsn == "" {
		dsn = "root:allgood@tcp(127.0.0.1:3306)/fandian_vp_service_it?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Skipf("test MySQL 不可用（设置 FANDIAN_TEST_DSN）: %v", err)
	}
	models := []any{
		&model.User{}, &model.Tenant{}, &model.TenantMember{}, &model.Category{}, &model.Transaction{},
		&model.DemoAccount{}, &model.BillingPackage{}, &model.BillingPurchaseIntent{},
		&model.BillingProcessedEvent{}, &model.BillingRefund{}, &model.BillingCapability{},
		&model.CreditLot{}, &model.CreditLog{}, &model.CreditLogDetail{}, &model.TenantCredits{},
	}
	_ = db.Exec("SET FOREIGN_KEY_CHECKS=0").Error
	for _, m := range models {
		_ = db.Migrator().DropTable(m)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestDemoEnsureIsolation(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	userRepo := repo.NewUserRepo(db)
	tenantRepo := repo.NewTenantRepo(db)
	memberRepo := repo.NewTenantMemberRepo(db)
	categoryRepo := repo.NewCategoryRepo(db)
	txRepo := repo.NewTransactionRepo(db)
	demoRepo := repo.NewDemoRepo(db)

	// 已存在的普通用户占用保留名 sys_demo，且拥有真实账本。
	realUser := &model.User{Username: "sys_demo", Email: "real@example.com", PasswordHash: "x"}
	if err := db.Create(realUser).Error; err != nil {
		t.Fatal(err)
	}
	realTenant := &model.Tenant{Name: "真实账本", OwnerID: realUser.ID, BookType: "restaurant"}
	if err := db.Create(realTenant).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.TenantMember{TenantID: realTenant.ID, UserID: realUser.ID, Role: "admin", JoinedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}

	cfg := config.BillingConfig{Enabled: true, DemoLoginEnabled: true, DemoTTLMinutes: 30, AppSlug: "hehuojizhangzhushou"}
	demoSvc := service.NewDemoService(demoRepo, userRepo, tenantRepo, memberRepo, categoryRepo, txRepo, cfg)
	user, err := demoSvc.Ensure(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == realUser.ID {
		t.Fatal("demo must not reuse real user with reserved username")
	}
	if !user.IsDemo {
		t.Fatal("demo user must be marked IsDemo")
	}
	account, err := demoRepo.GetAccount(ctx)
	if err != nil || account == nil {
		t.Fatalf("demo account: %v", err)
	}
	if account.UserID != user.ID {
		t.Fatal("demo account user mismatch")
	}
	demoTenant, err := tenantRepo.GetByID(ctx, account.TenantID)
	if err != nil || demoTenant == nil || !demoTenant.IsDemo {
		t.Fatalf("demo tenant invalid: %+v err=%v", demoTenant, err)
	}

	// 即使演示用户被错误加入真实账本，体验登录也只能返回演示账本。
	if err := db.Create(&model.TenantMember{TenantID: realTenant.ID, UserID: user.ID, Role: "admin", JoinedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	appCfg := &config.Config{
		JWT:     config.JWTConfig{Secret: "test-secret"},
		Billing: cfg,
	}
	authSvc := service.NewAuthService(userRepo, tenantRepo, memberRepo, categoryRepo, nil, nil, demoRepo, appCfg)
	resp, err := authSvc.DemoLogin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Demo || len(resp.Tenants) != 1 || resp.Tenants[0].ID != demoTenant.ID {
		t.Fatalf("demo login leaks extra tenants: %+v", resp)
	}
	claims := &service.Claims{}
	tok, err := jwt.ParseWithClaims(resp.Token, claims, func(token *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	if err != nil || !tok.Valid {
		t.Fatalf("parse demo token: %v", err)
	}
	if !claims.Demo || claims.DemoTenantID != demoTenant.ID {
		t.Fatalf("demo claims not bound to demo tenant: %+v", claims)
	}

	// 普通用户仍不能被体验登录复用。
	normal, err := authSvc.Login(ctx, dto.LoginReq{Email: "real@example.com", Password: "x"})
	if err == nil && normal != nil && normal.Demo {
		t.Fatal("normal user login must not be demo")
	}
}

func TestDemoEnsureConcurrentSingleAccount(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	userRepo := repo.NewUserRepo(db)
	tenantRepo := repo.NewTenantRepo(db)
	memberRepo := repo.NewTenantMemberRepo(db)
	categoryRepo := repo.NewCategoryRepo(db)
	txRepo := repo.NewTransactionRepo(db)
	demoRepo := repo.NewDemoRepo(db)
	cfg := config.BillingConfig{Enabled: true, DemoLoginEnabled: true, AppSlug: "hehuojizhangzhushou"}
	demoSvc := service.NewDemoService(demoRepo, userRepo, tenantRepo, memberRepo, categoryRepo, txRepo, cfg)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := demoSvc.Ensure(ctx); err != nil {
				t.Errorf("ensure: %v", err)
			}
		}()
	}
	wg.Wait()
	var accounts int64
	db.Model(&model.DemoAccount{}).Count(&accounts)
	if accounts != 1 {
		t.Fatalf("demo accounts = %d, want 1", accounts)
	}
	var demos int64
	db.Model(&model.User{}).Where("is_demo = ?", true).Count(&demos)
	if demos != 1 {
		t.Fatalf("demo users = %d, want 1", demos)
	}
}
