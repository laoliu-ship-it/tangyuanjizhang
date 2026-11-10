package repo

import (
	"context"
	"errors"

	"fandianjizhang/server/internal/model"
	"gorm.io/gorm"
)

type demoRepo struct {
	db *gorm.DB
}

func NewDemoRepo(db *gorm.DB) DemoRepo { return &demoRepo{db: db} }

func (r *demoRepo) GetAccount(ctx context.Context) (*model.DemoAccount, error) {
	var a model.DemoAccount
	err := r.db.WithContext(ctx).Order("id ASC").First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *demoRepo) CreateAccount(ctx context.Context, account *model.DemoAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *demoRepo) WithInitLock(ctx context.Context, fn func() error) error {
	return r.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		var got int
		if err := conn.Raw("SELECT GET_LOCK(?, 10)", "fandian_demo_init").Scan(&got).Error; err != nil {
			return err
		}
		if got != 1 {
			return errors.New("demo init lock timeout")
		}
		defer conn.Exec("SELECT RELEASE_LOCK(?)", "fandian_demo_init")
		return fn()
	})
}
