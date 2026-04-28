package middleware

import "github.com/gin-gonic/gin"

const ContextUserID = "user_id"

func UserIDFromContext(c *gin.Context) string {
	v, ok := c.Get(ContextUserID)
	if !ok {
		return ""
	}
	id, _ := v.(string)
	return id
}
