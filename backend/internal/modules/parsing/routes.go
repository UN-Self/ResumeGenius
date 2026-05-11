package parsing

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	base := NewGitExtractor()
	gitExtractor := chooseGitExtractor(base)
	service := NewParsingServiceWithStorage(db, NewPDFParser(), NewDocxParser(), gitExtractor, store)
	handler := NewHandler(service)

	rg.POST("/parsing/parse", handler.Parse)
	rg.POST("/parsing/assets/:asset_id/parse", handler.ParseAsset)
}

func chooseGitExtractor(base *GitRepositoryExtractor) GitExtractor {
	if strings.TrimSpace(os.Getenv("USE_MOCK")) == "true" {
		return base
	}
	apiURL := strings.TrimSpace(os.Getenv("AI_API_URL"))
	apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	if apiURL == "" || apiKey == "" {
		return base
	}
	return NewAIGitExtractor(base)
}
