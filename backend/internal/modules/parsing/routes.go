package parsing

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	generator := NewDraftGenerator()
	service := NewParsingServiceWithGenerator(db, NewPDFParser(), NewDocxParser(), NewGitExtractor(), generator)
	handler := NewHandler(service)

	rg.POST("/parsing/parse", handler.Parse)
	rg.POST("/parsing/generate", handler.Generate)
}
