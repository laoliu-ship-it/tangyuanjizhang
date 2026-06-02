package middleware

import (
	"net/http"
	"strings"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const ContextKeyUserID = "userID"

func JWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.Fail(401, "缺少 Authorization 头"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, dto.Fail(401, "Authorization 格式错误，应为 Bearer <token>"))
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims := &service.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, dto.Fail(401, "token 无效或已过期"))
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Next()
	}
}

// GetUserID 从 gin.Context 中获取当前用户ID
func GetUserID(c *gin.Context) uint64 {
	v, _ := c.Get(ContextKeyUserID)
	id, _ := v.(uint64)
	return id
}
