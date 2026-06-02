package middleware

import (
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/repo"

	"github.com/gin-gonic/gin"
)

const (
	ContextKeyTenantID = "tenantID"
	ContextKeyUserRole = "userRole"
)

func Tenant(tenantMemberRepo repo.TenantMemberRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		if tenantIDStr == "" {
			c.JSON(http.StatusBadRequest, dto.Fail(400, "缺少 X-Tenant-ID 请求头"))
			c.Abort()
			return
		}

		tenantID, err := strconv.ParseUint(tenantIDStr, 10, 64)
		if err != nil || tenantID == 0 {
			c.JSON(http.StatusBadRequest, dto.Fail(400, "X-Tenant-ID 格式错误"))
			c.Abort()
			return
		}

		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, dto.Fail(401, "未认证"))
			c.Abort()
			return
		}

		role, err := tenantMemberRepo.GetRole(c.Request.Context(), tenantID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "服务器内部错误"))
			c.Abort()
			return
		}
		if role == "" {
			c.JSON(http.StatusForbidden, dto.Fail(403, "您不是该租户的成员"))
			c.Abort()
			return
		}

		c.Set(ContextKeyTenantID, tenantID)
		c.Set(ContextKeyUserRole, role)
		c.Next()
	}
}

// GetTenantID 从 gin.Context 中获取当前租户ID
func GetTenantID(c *gin.Context) uint64 {
	v, _ := c.Get(ContextKeyTenantID)
	id, _ := v.(uint64)
	return id
}

// GetUserRole 从 gin.Context 中获取当前用户在租户中的角色
func GetUserRole(c *gin.Context) string {
	v, _ := c.Get(ContextKeyUserRole)
	role, _ := v.(string)
	return role
}
