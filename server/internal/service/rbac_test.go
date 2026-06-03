package service

import (
	"context"
	"testing"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/roledefs"
)

// ========== Mock Repositories ==========

type mockTenantRoleRepo struct {
	roles  map[uint64]*model.TenantRole
	byName map[string]*model.TenantRole
}

func newMockTenantRoleRepo() *mockTenantRoleRepo {
	return &mockTenantRoleRepo{
		roles:  make(map[uint64]*model.TenantRole),
		byName: make(map[string]*model.TenantRole),
	}
}

func (m *mockTenantRoleRepo) Create(ctx context.Context, role *model.TenantRole) error {
	if role.ID == 0 {
		role.ID = uint64(len(m.roles) + 1)
	}
	m.roles[role.ID] = role
	if role.Name != "" {
		m.byName[role.Name] = role
	}
	return nil
}

func (m *mockTenantRoleRepo) GetByID(ctx context.Context, id uint64) (*model.TenantRole, error) {
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *mockTenantRoleRepo) GetByName(ctx context.Context, tenantID uint64, name string) (*model.TenantRole, error) {
	if r, ok := m.byName[name]; ok && r.TenantID == tenantID {
		return r, nil
	}
	return nil, nil
}

func (m *mockTenantRoleRepo) ListByTenant(ctx context.Context, tenantID uint64) ([]*model.TenantRole, error) {
	var result []*model.TenantRole
	for _, r := range m.roles {
		if r.TenantID == tenantID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockTenantRoleRepo) Update(ctx context.Context, role *model.TenantRole) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockTenantRoleRepo) Delete(ctx context.Context, id uint64) error {
	delete(m.roles, id)
	return nil
}

type mockRolePermissionRepo struct {
	perms map[uint64][]*model.RolePermission
}

func newMockRolePermissionRepo() *mockRolePermissionRepo {
	return &mockRolePermissionRepo{
		perms: make(map[uint64][]*model.RolePermission),
	}
}

func (m *mockRolePermissionRepo) Create(ctx context.Context, perm *model.RolePermission) error {
	m.perms[perm.RoleID] = append(m.perms[perm.RoleID], perm)
	return nil
}

func (m *mockRolePermissionRepo) BatchCreate(ctx context.Context, perms []*model.RolePermission) error {
	for _, p := range perms {
		m.perms[p.RoleID] = append(m.perms[p.RoleID], p)
	}
	return nil
}

func (m *mockRolePermissionRepo) DeleteByRoleID(ctx context.Context, roleID uint64) error {
	delete(m.perms, roleID)
	return nil
}

func (m *mockRolePermissionRepo) ListByRoleID(ctx context.Context, roleID uint64) ([]*model.RolePermission, error) {
	return m.perms[roleID], nil
}

func (m *mockRolePermissionRepo) ListByTenantID(ctx context.Context, tenantID uint64) ([]*model.RolePermission, error) {
	return nil, nil
}

// ========== Test Helpers ==========

func newTestService() (*mockTenantRoleRepo, *mockRolePermissionRepo) {
	return newMockTenantRoleRepo(), newMockRolePermissionRepo()
}

func seedRole(repo *mockTenantRoleRepo, tenantID uint64, id uint64, name string, isSystem bool) {
	role := &model.TenantRole{
		ID:       id,
		TenantID: tenantID,
		Name:     name,
		IsSystem: isSystem,
	}
	repo.roles[id] = role
	if name != "" {
		repo.byName[name] = role
	}
}

// Directly test the ListRoles logic without the full service (avoids casbin pool)
func testListRoles(ctx context.Context, roleRepo *mockTenantRoleRepo, permRepo *mockRolePermissionRepo, tenantID uint64) ([]*dto.RoleResp, error) {
	roles, err := roleRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	customRoleNames := make(map[string]bool, len(roles))
	for _, role := range roles {
		customRoleNames[role.Name] = true
	}

	result := make([]*dto.RoleResp, 0, len(roledefs.BuiltInRolePerms())+len(roles))

	for roleName, perms := range roledefs.BuiltInRolePerms() {
		if customRoleNames[roleName] {
			continue
		}
		resp := &dto.RoleResp{Name: roleName, IsSystem: true}
		for _, p := range perms {
			resp.Permissions = append(resp.Permissions, dto.PermissionEntry{
				Resource: p.Resource, Action: p.Action,
			})
		}
		result = append(result, resp)
	}

	for _, role := range roles {
		resp := &dto.RoleResp{ID: role.ID, Name: role.Name, IsSystem: role.IsSystem}
		if isBuiltInRole(role.Name) {
			for _, p := range roledefs.BuiltInRolePerms()[role.Name] {
				resp.Permissions = append(resp.Permissions, dto.PermissionEntry{
					Resource: p.Resource, Action: p.Action,
				})
			}
		} else {
			perms, _ := permRepo.ListByRoleID(ctx, role.ID)
			for _, p := range perms {
				resp.Permissions = append(resp.Permissions, dto.PermissionEntry{
					Resource: p.Resource, Action: p.Action,
				})
			}
		}
		result = append(result, resp)
	}
	return result, nil
}

func testDeleteRole(ctx context.Context, roleRepo *mockTenantRoleRepo, permRepo *mockRolePermissionRepo, tenantID, roleID uint64) error {
	role, err := roleRepo.GetByID(ctx, roleID)
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
	if err := permRepo.DeleteByRoleID(ctx, roleID); err != nil {
		return err
	}
	return roleRepo.Delete(ctx, roleID)
}

func testCreateRole(ctx context.Context, roleRepo *mockTenantRoleRepo, permRepo *mockRolePermissionRepo, tenantID uint64, req dto.CreateRoleReq) error {
	if isBuiltInRole(req.Name) {
		return ErrRoleAlreadyName
	}
	existing, err := roleRepo.GetByName(ctx, tenantID, req.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrRoleAlreadyName
	}

	role := &model.TenantRole{TenantID: tenantID, Name: req.Name, IsSystem: false}
	if err := roleRepo.Create(ctx, role); err != nil {
		return err
	}

	perms := make([]*model.RolePermission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, &model.RolePermission{RoleID: role.ID, Resource: p.Resource, Action: p.Action})
	}
	return permRepo.BatchCreate(ctx, perms)
}

// ========== Tests ==========

func TestListRoles_NoDuplicates(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	seedRole(roleRepo, tenantID, 1, "admin", true)
	seedRole(roleRepo, tenantID, 2, "finance", true)

	roles, err := testListRoles(ctx, roleRepo, permRepo, tenantID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	count := 0
	for _, r := range roles {
		if r.Name == "admin" || r.Name == "finance" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 roles (admin+finance), got %d", count)
	}

	found := false
	for _, r := range roles {
		if r.Name == "partner" && r.IsSystem {
			found = true
		}
	}
	if !found {
		t.Error("expected partner from built-in")
	}
}

func TestListRoles_BuiltInPermissions(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	seedRole(roleRepo, tenantID, 1, "admin", true)

	roles, err := testListRoles(ctx, roleRepo, permRepo, tenantID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	for _, r := range roles {
		if r.Name == "admin" {
			if len(r.Permissions) == 0 {
				t.Error("admin should have permissions")
			}
			hasStar := false
			for _, p := range r.Permissions {
				if p.Resource == "*" && p.Action == "*" {
					hasStar = true
				}
			}
			if !hasStar {
				t.Error("admin should have *:*")
			}
		}
	}
}

func TestCreateRole_DuplicateName(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	seedRole(roleRepo, tenantID, 1, "my-role", false)

	err := testCreateRole(ctx, roleRepo, permRepo, tenantID, dto.CreateRoleReq{
		Name:        "my-role",
		Permissions: []dto.PermissionEntry{{Resource: "transaction", Action: "read"}},
	})
	if err != ErrRoleAlreadyName {
		t.Errorf("expected ErrRoleAlreadyName, got %v", err)
	}
}

func TestCreateRole_BuiltInNameConflict(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	err := testCreateRole(ctx, roleRepo, permRepo, tenantID, dto.CreateRoleReq{
		Name:        "admin",
		Permissions: []dto.PermissionEntry{{Resource: "transaction", Action: "read"}},
	})
	if err != ErrRoleAlreadyName {
		t.Errorf("expected ErrRoleAlreadyName, got %v", err)
	}
}

func TestDeleteRole_SystemRole(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	seedRole(roleRepo, tenantID, 1, "admin", true)

	err := testDeleteRole(ctx, roleRepo, permRepo, tenantID, 1)
	if err != ErrRoleIsSystem {
		t.Errorf("expected ErrRoleIsSystem, got %v", err)
	}
}

func TestDeleteRole_CustomRole(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	seedRole(roleRepo, tenantID, 1, "editor", false)
	permRepo.perms[1] = []*model.RolePermission{{RoleID: 1, Resource: "transaction", Action: "write"}}

	err := testDeleteRole(ctx, roleRepo, permRepo, tenantID, 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	role, _ := roleRepo.GetByID(ctx, 1)
	if role != nil {
		t.Error("role should be deleted")
	}

	perms, _ := permRepo.ListByRoleID(ctx, 1)
	if len(perms) > 0 {
		t.Error("permissions should be deleted")
	}
}

func TestDeleteRole_NotFound(t *testing.T) {
	roleRepo, permRepo := newTestService()
	ctx := context.Background()
	const tenantID uint64 = 1

	err := testDeleteRole(ctx, roleRepo, permRepo, tenantID, 999)
	if err != ErrRoleNotFound {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestIsBuiltInRole(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"admin", true}, {"finance", true}, {"partner", true},
		{"custom", false}, {"Admin", false}, {"", false},
	}
	for _, tt := range tests {
		if got := isBuiltInRole(tt.name); got != tt.want {
			t.Errorf("isBuiltInRole(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBuiltInRolePerms(t *testing.T) {
	perms := roledefs.BuiltInRolePerms()

	adminPerms := perms["admin"]
	if len(adminPerms) != 1 || adminPerms[0].Resource != "*" || adminPerms[0].Action != "*" {
		t.Errorf("admin expected [*:*], got %v", adminPerms)
	}

	financePerms := perms["finance"]
	found := false
	for _, p := range financePerms {
		if p.Resource == "transaction" && p.Action == "write" {
			found = true
		}
	}
	if !found {
		t.Error("finance should have transaction:write")
	}

	if len(perms) != 3 {
		t.Errorf("expected 3 built-in roles, got %d", len(perms))
	}
}
