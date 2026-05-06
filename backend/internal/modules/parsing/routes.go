package parsing

import (
	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	service := NewParsingServiceWithStorage(db, NewPDFParser(), NewDocxParser(), NewGitExtractor(), store)
	handler := NewHandler(service)

	rg.POST("/parsing/parse", handler.Parse)
}
