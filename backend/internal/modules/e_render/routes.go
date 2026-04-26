package e_render

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.POST("/export", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "e_render", "status": "stub"})
	})
}
