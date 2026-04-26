package a_intake

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/projects", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "a_intake", "status": "stub"})
	})
}
