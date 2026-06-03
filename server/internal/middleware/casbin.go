package middleware

import (
	"net/http"

	"fandianjizhang/server/internal/casbin"
	"fandianjizhang/server/internal/dto"

	"github.com/gin-gonic/gin"
)

// Casbin 中间件，检查当前角色对资源的操作权限
func Casbin(pool *casbin.EnforcerPool, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetUserRole(c)
		if role == "" {
			c.JSON(http.StatusForbidden, dto.Fail(403, "无访问权限"))
			c.Abort()
			return
		}

		tenantID := GetTenantID(c)
		enforcer, err := pool.GetEnforcer(tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "权限检查失败"))
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
			c.JSON(http.StatusForbidden, dto.Fail(403, "权限不足"))
			c.Abort()
			return
		}
		c.Next()
	}
}
