package parsing

import (
	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	generator := NewDraftGenerator()
	service := NewParsingServiceWithGeneratorAndStorage(db, NewPDFParser(), NewDocxParser(), NewGitExtractor(), generator, store)
	handler := NewHandler(service)

	rg.POST("/parsing/parse", handler.Parse)
	rg.POST("/parsing/generate", handler.Generate)
}
