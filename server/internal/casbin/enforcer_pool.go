package casbin

import (
	"sync"

	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/roledefs"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

// RBAC 模型定义
const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (p.obj == "*" || r.obj == p.obj) && (p.act == "*" || r.act == p.act)
`

// EnforcerPool 按租户缓存 Casbin Enforcer
type EnforcerPool struct {
	mu    sync.RWMutex
	cache map[uint64]*casbin.Enforcer
	db    *gorm.DB
	model casbinmodel.Model
}

func NewEnforcerPool(db *gorm.DB) (*EnforcerPool, error) {
	m, err := casbinmodel.NewModelFromString(modelText)
	if err != nil {
		return nil, err
	}
	return &EnforcerPool{
		cache: make(map[uint64]*casbin.Enforcer),
		db:    db,
		model: m,
	}, nil
}

// GetEnforcer 获取指定租户的 enforcer（带缓存）
func (p *EnforcerPool) GetEnforcer(tenantID uint64) (*casbin.Enforcer, error) {
	p.mu.RLock()
	enforcer, ok := p.cache[tenantID]
	p.mu.RUnlock()
	if ok {
		return enforcer, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// double check
	if enforcer, ok = p.cache[tenantID]; ok {
		return enforcer, nil
	}

	adapter, err := p.loadTenantAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	enforcer, err = casbin.NewEnforcer(p.model, adapter)
	if err != nil {
		return nil, err
	}

	p.cache[tenantID] = enforcer
	return enforcer, nil
}

// Invalidate 清除指定租户的 enforcer 缓存
func (p *EnforcerPool) Invalidate(tenantID uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cache, tenantID)
}

// loadTenantAdapter 从数据库加载指定租户的角色权限策略
func (p *EnforcerPool) loadTenantAdapter(tenantID uint64) (*tenantAdapter, error) {
	// 查询该租户的所有角色
	var roles []model.TenantRole
	if err := p.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Find(&roles).Error; err != nil {
		return nil, err
	}

	// 查询所有角色的权限
	var perms []model.RolePermission
	roleIDs := make([]uint64, 0, len(roles))
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}
	if len(roleIDs) > 0 {
		if err := p.db.Where("role_id IN ?", roleIDs).Find(&perms).Error; err != nil {
			return nil, err
		}
	}

	// 构建策略
	policies := make([][]string, 0, len(perms))
	permMap := make(map[uint64][][2]string)
	for _, perm := range perms {
		permMap[perm.RoleID] = append(permMap[perm.RoleID], [2]string{perm.Resource, perm.Action})
	}

	roleMap := make(map[uint64]model.TenantRole)
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	for roleID, rps := range permMap {
		role, ok := roleMap[roleID]
		if !ok {
			continue
		}
		for _, rp := range rps {
			policies = append(policies, []string{role.Name, rp[0], rp[1]})
		}
	}

	// 追加系统内置角色策略（所有租户共享）
	for roleName, perms := range roledefs.BuiltInRolePerms() {
		for _, rp := range perms {
			policies = append(policies, []string{roleName, rp.Resource, rp.Action})
		}
	}

	return &tenantAdapter{policies: policies}, nil
}

// ========== 租户级内存 Adapter ==========

type tenantAdapter struct {
	policies [][]string
}

func (a *tenantAdapter) LoadPolicy(model casbinmodel.Model) error {
	for _, rule := range a.policies {
		if len(rule) != 3 {
			continue
		}
		line := "p, " + rule[0] + ", " + rule[1] + ", " + rule[2]
		persist.LoadPolicyLine(line, model)
	}
	return nil
}

func (a *tenantAdapter) SavePolicy(_ casbinmodel.Model) error { return nil }
func (a *tenantAdapter) AddPolicy(_ string, _ string, _ []string) error {
	return nil // 不支持运行时添加
}
func (a *tenantAdapter) RemovePolicy(_ string, _ string, _ []string) error {
	return nil // 不支持运行时删除
}
func (a *tenantAdapter) RemoveFilteredPolicy(_ string, _ string, _ int, _ ...string) error {
	return nil
}
