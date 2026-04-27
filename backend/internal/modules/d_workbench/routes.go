package d_workbench

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/drafts/:draft_id", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "d_workbench", "status": "stub"})
	})

	rg.PUT("/drafts/:draft_id", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "d_workbench", "status": "stub"})
	})
}
