package middleware

import "github.com/gin-gonic/gin"

const ContextUserID = "user_id"

// UserIdentify 从 X-User-ID header 提取匿名用户 ID，注入 context
func UserIdentify() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}
