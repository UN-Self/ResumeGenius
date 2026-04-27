package c_agent

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.POST("/ai/sessions", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "c_agent", "status": "stub"})
	})
}
