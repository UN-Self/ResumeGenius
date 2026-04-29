package parsing

import (
	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	rg.POST("/parsing/parse", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "parsing", "status": "stub"})
	})
}
