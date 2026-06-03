package service

import (
	"context"
	"errors"

	"fandianjizhang/server/internal/casbin"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/roledefs"
)

func isBuiltInRole(name string) bool {
	_, ok := roledefs.BuiltInRolePerms()[name]
	return ok
}

// 所有可用的权限对（用于前端展示）
var allPermissions = []dto.PermissionEntry{
	{Resource: "transaction", Action: "read"},
	{Resource: "transaction", Action: "write"},
	{Resource: "category", Action: "read"},
	{Resource: "category", Action: "write"},
	{Resource: "merchant", Action: "read"},
	{Resource: "merchant", Action: "write"},
	{Resource: "statistics", Action: "read"},
	{Resource: "export", Action: "read"},
	{Resource: "tenant", Action: "read"},
	{Resource: "tenant", Action: "write"},
}

var (
	ErrRoleNotFound    = errors.New("角色不存在")
	ErrRoleIsSystem    = errors.New("系统角色不可修改")
	ErrRoleHasMembers  = errors.New("该角色下有成员，不可删除")
	ErrRoleAlreadyName = errors.New("角色名称已存在")
)

type RBACService interface {
	ListRoles(ctx context.Context, tenantID uint64) ([]*dto.RoleResp, error)
	CreateRole(ctx context.Context, tenantID uint64, req dto.CreateRoleReq) (*model.TenantRole, error)
	UpdateRolePermissions(ctx context.Context, tenantID uint64, roleID uint64, permissions []dto.PermissionEntry) error
	DeleteRole(ctx context.Context, tenantID uint64, roleID uint64) error
	AvailablePermissions() []dto.PermissionEntry
	InvalidateCache(tenantID uint64)
}

type rbacService struct {
	roleRepo repo.TenantRoleRepo
	permRepo repo.RolePermissionRepo
	pool     *casbin.EnforcerPool
}

func NewRBACService(roleRepo repo.TenantRoleRepo, permRepo repo.RolePermissionRepo, pool *casbin.EnforcerPool) RBACService {
	return &rbacService{
		roleRepo: roleRepo,
		permRepo: permRepo,
		pool:     pool,
	}
}

func (s *rbacService) ListRoles(ctx context.Context, tenantID uint64) ([]*dto.RoleResp, error) {
	// 先获取租户自定义角色
	roles, err := s.roleRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 收集数据库中已有的角色名（用于去重内置角色）
	customRoleNames := make(map[string]bool, len(roles))
	for _, role := range roles {
		customRoleNames[role.Name] = true
	}

	result := make([]*dto.RoleResp, 0, len(roledefs.BuiltInRolePerms())+len(roles))

	// 添加系统内置角色（仅当数据库中没有同名角色时）
	for roleName, perms := range roledefs.BuiltInRolePerms() {
		if customRoleNames[roleName] {
			continue
		}
		resp := &dto.RoleResp{
			Name:     roleName,
			IsSystem: true,
		}
		for _, p := range perms {
			resp.Permissions = append(resp.Permissions, dto.PermissionEntry{
				Resource: p.Resource,
				Action:   p.Action,
			})
		}
		result = append(result, resp)
	}

	// 添加租户自定义角色
	for _, role := range roles {
		resp := &dto.RoleResp{
			ID:       role.ID,
			Name:     role.Name,
			IsSystem: role.IsSystem,
		}

		// 如果是内置角色名但数据库有记录，使用内置权限
		if isBuiltInRole(role.Name) {
			for _, p := range roledefs.BuiltInRolePerms()[role.Name] {
				resp.Permissions = append(resp.Permissions, dto.PermissionEntry{
					Resource: p.Resource,
					Action:   p.Action,
				})
			}
		} else {
			// 自定义角色，从数据库读取权限
			perms, _ := s.permRepo.ListByRoleID(ctx, role.ID)
			for _, p := range perms {
				resp.Permissions = append(resp.Permissions, dto.PermissionEntry{
					Resource: p.Resource,
					Action:   p.Action,
				})
			}
		}
		result = append(result, resp)
	}
	return result, nil
}

func (s *rbacService) CreateRole(ctx context.Context, tenantID uint64, req dto.CreateRoleReq) (*model.TenantRole, error) {
	// 检查是否与内置角色名冲突
	if isBuiltInRole(req.Name) {
		return nil, ErrRoleAlreadyName
	}

	// 检查名称是否与自定义角色重复
	existing, err := s.roleRepo.GetByName(ctx, tenantID, req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRoleAlreadyName
	}

	role := &model.TenantRole{
		TenantID: tenantID,
		Name:     req.Name,
		IsSystem: false,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	perms := make([]*model.RolePermission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, &model.RolePermission{
			RoleID:   role.ID,
			Resource: p.Resource,
			Action:   p.Action,
		})
	}
	if err := s.permRepo.BatchCreate(ctx, perms); err != nil {
		return nil, err
	}

	s.InvalidateCache(tenantID)
	return role, nil
}

func (s *rbacService) UpdateRolePermissions(ctx context.Context, tenantID uint64, roleID uint64, permissions []dto.PermissionEntry) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}
	if role.TenantID != tenantID {
		return ErrRoleNotFound
	}
	if role.IsSystem {
		return ErrRoleIsSystem
	}

	// 删除旧权限，创建新权限
	if err := s.permRepo.DeleteByRoleID(ctx, roleID); err != nil {
		return err
	}

	perms := make([]*model.RolePermission, 0, len(permissions))
	for _, p := range permissions {
		perms = append(perms, &model.RolePermission{
			RoleID:   roleID,
			Resource: p.Resource,
			Action:   p.Action,
		})
	}
	if err := s.permRepo.BatchCreate(ctx, perms); err != nil {
		return err
	}

	s.InvalidateCache(tenantID)
	return nil
}

func (s *rbacService) DeleteRole(ctx context.Context, tenantID uint64, roleID uint64) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}
	if role.TenantID != tenantID {
		return ErrRoleNotFound
	}
	if role.IsSystem {
		return ErrRoleIsSystem
	}

	// 删除角色关联的权限
	if err := s.permRepo.DeleteByRoleID(ctx, roleID); err != nil {
		return err
	}

	s.InvalidateCache(tenantID)
	return s.roleRepo.Delete(ctx, roleID)
}

func (s *rbacService) AvailablePermissions() []dto.PermissionEntry {
	return allPermissions
}

func (s *rbacService) InvalidateCache(tenantID uint64) {
	s.pool.Invalidate(tenantID)
}
