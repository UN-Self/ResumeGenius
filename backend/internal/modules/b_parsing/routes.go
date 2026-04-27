package b_parsing

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.POST("/parsing/parse", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "b_parsing", "status": "stub"})
	})
}
