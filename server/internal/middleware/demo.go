package middleware

import (
	"net/http"
	"strings"

	"fandianjizhang/server/internal/dto"
	"github.com/gin-gonic/gin"
)

// IsDemo 判断当前 JWT 是否为演示身份。
func IsDemo(c *gin.Context) bool {
	v, _ := c.Get(ContextKeyDemo)
	demo, _ := v.(bool)
	return demo
}

// GetDemoTenantID 返回演示 token 绑定的唯一账本。
func GetDemoTenantID(c *gin.Context) uint64 {
	v, _ := c.Get(ContextKeyDemoTenantID)
	id, _ := v.(uint64)
	return id
}

// DemoReadOnly 阻止演示身份执行真实写操作；OCR/AI 示例接口放行（由 handler 返回示例结果）。
func DemoReadOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsDemo(c) {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		path := c.Request.URL.Path
		allowed := strings.HasSuffix(path, "/upload/ocr") ||
			strings.HasSuffix(path, "/upload/ocr/analyze") ||
			strings.HasSuffix(path, "/llm/analyze")
		if allowed {
			c.Next()
			return
		}
		c.JSON(http.StatusOK, dto.Fail(403, "演示账号仅可浏览，不可修改数据或购买"))
		c.Abort()
	}
}
