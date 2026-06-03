package repo

import (
	"context"
	"errors"

	"fandianjizhang/server/internal/model"
	"gorm.io/gorm"
)

// ========== TenantRoleRepo ==========

type tenantRoleRepo struct {
	db *gorm.DB
}

func NewTenantRoleRepo(db *gorm.DB) TenantRoleRepo {
	return &tenantRoleRepo{db: db}
}

func (r *tenantRoleRepo) Create(ctx context.Context, role *model.TenantRole) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *tenantRoleRepo) GetByID(ctx context.Context, id uint64) (*model.TenantRole, error) {
	var role model.TenantRole
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *tenantRoleRepo) GetByName(ctx context.Context, tenantID uint64, name string) (*model.TenantRole, error) {
	var role model.TenantRole
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND deleted_at IS NULL", tenantID, name).
		First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *tenantRoleRepo) ListByTenant(ctx context.Context, tenantID uint64) ([]*model.TenantRole, error) {
	var roles []*model.TenantRole
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("is_system DESC, name ASC").
		Find(&roles).Error
	return roles, err
}

func (r *tenantRoleRepo) Update(ctx context.Context, role *model.TenantRole) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *tenantRoleRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.TenantRole{}, id).Error
}

// ========== RolePermissionRepo ==========

type rolePermissionRepo struct {
	db *gorm.DB
}

func NewRolePermissionRepo(db *gorm.DB) RolePermissionRepo {
	return &rolePermissionRepo{db: db}
}

func (r *rolePermissionRepo) Create(ctx context.Context, perm *model.RolePermission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *rolePermissionRepo) BatchCreate(ctx context.Context, perms []*model.RolePermission) error {
	if len(perms) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(perms, 50).Error
}

func (r *rolePermissionRepo) DeleteByRoleID(ctx context.Context, roleID uint64) error {
	return r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error
}

func (r *rolePermissionRepo) ListByRoleID(ctx context.Context, roleID uint64) ([]*model.RolePermission, error) {
	var perms []*model.RolePermission
	err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&perms).Error
	return perms, err
}

func (r *rolePermissionRepo) ListByTenantID(ctx context.Context, tenantID uint64) ([]*model.RolePermission, error) {
	var perms []*model.RolePermission
	err := r.db.WithContext(ctx).
		Table("role_permissions rp").
		Joins("JOIN tenant_roles tr ON rp.role_id = tr.id AND tr.deleted_at IS NULL").
		Where("tr.tenant_id = ?", tenantID).
		Select("rp.*").
		Find(&perms).Error
	return perms, err
}
