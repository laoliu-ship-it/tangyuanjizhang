package middleware

import (
	"net/http"
	"strings"

	"fandianjizhang/server/internal/dto"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gin-gonic/gin"
)

// NewCasbinEnforcer 创建基于内存策略的 Casbin enforcer（无数据库持久化）
func NewCasbinEnforcer() (*casbin.Enforcer, error) {
	// 定义 RBAC 模型文本（不使用角色继承，直接匹配角色名）
	modelText := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (p.sub == "*" || r.sub == p.sub) && (p.obj == "*" || r.obj == p.obj) && (p.act == "*" || r.act == p.act)
`
	m, err := casbinmodel.NewModelFromString(modelText)
	if err != nil {
		return nil, err
	}

	// 固定策略列表
	policies := [][]string{
		// admin：全权限
		{"admin", "*", "*"},
		// editor：记账 + 读取基础数据
		{"editor", "transaction", "write"},
		{"editor", "transaction", "read"},
		{"editor", "category", "read"},
		{"editor", "merchant", "read"},
		{"editor", "ocr", "write"},
		// viewer：只读
		{"viewer", "transaction", "read"},
		{"viewer", "category", "read"},
		{"viewer", "merchant", "read"},
		{"viewer", "statistics", "read"},
		{"viewer", "export", "read"},
		// partner（合伙人）：查看账目、统计、明细，不能编辑
		{"partner", "transaction", "read"},
		{"partner", "category", "read"},
		{"partner", "merchant", "read"},
		{"partner", "statistics", "read"},
		{"partner", "export", "read"},
		// finance（财务）：记账+编辑+删除+统计+人员管理
		{"finance", "transaction", "write"},
		{"finance", "transaction", "read"},
		{"finance", "category", "read"},
		{"finance", "merchant", "read"},
		{"finance", "ocr", "write"},
		{"finance", "statistics", "read"},
		{"finance", "export", "read"},
		{"finance", "tenant", "write"},
	}

	adapter := &memoryAdapter{policies: policies}
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}

	return enforcer, nil
}

// Casbin 中间件，检查当前角色对资源的操作权限
// resource: 资源名称（如 "transaction", "category" 等）
// action: 操作（"read" 或 "write"）
func Casbin(enforcer *casbin.Enforcer, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetUserRole(c)
		if role == "" {
			c.JSON(http.StatusForbidden, dto.Fail(403, "无访问权限"))
			c.Abort()
			return
		}

		allowed, err := enforcer.Enforce(role, resource, action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "权限检查失败"))
			c.Abort()
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, dto.Fail(403, "权限不足，需要 "+strings.ToUpper(action)+" 权限"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// ========== 内存 Adapter ==========
// 启动时加载固定策略，不做持久化

type memoryAdapter struct {
	policies [][]string // 每条策略为 [sub, obj, act]
}

func (a *memoryAdapter) LoadPolicy(model casbinmodel.Model) error {
	for _, rule := range a.policies {
		if len(rule) != 3 {
			continue
		}
		// 策略行格式: "p, sub, obj, act"
		line := "p, " + strings.Join(rule, ", ")
		persist.LoadPolicyLine(line, model)
	}
	return nil
}

func (a *memoryAdapter) SavePolicy(_ casbinmodel.Model) error {
	// 内存模式不持久化
	return nil
}

func (a *memoryAdapter) AddPolicy(_ string, _ string, rule []string) error {
	a.policies = append(a.policies, rule)
	return nil
}

func (a *memoryAdapter) RemovePolicy(_ string, _ string, rule []string) error {
	target := strings.Join(rule, ",")
	result := make([][]string, 0, len(a.policies))
	for _, p := range a.policies {
		if strings.Join(p, ",") != target {
			result = append(result, p)
		}
	}
	a.policies = result
	return nil
}

func (a *memoryAdapter) RemoveFilteredPolicy(_ string, _ string, _ int, _ ...string) error {
	return nil
}
