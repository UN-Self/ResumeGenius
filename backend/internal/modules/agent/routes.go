package agent

import (
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/render"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, versionSvc *render.VersionService, exportSvc *render.ExportService) {
	sessionSvc := NewSessionService(db)

	var provider ProviderAdapter
	if os.Getenv("USE_MOCK") == "true" {
		provider = &MockAdapter{}
	} else {
		provider = NewOpenAIAdapter(
			os.Getenv("AI_API_URL"),
			os.Getenv("AI_API_KEY"),
			envOrDefault("AI_MODEL", "default"),
		)
	}

	toolExecutor := NewAgentToolExecutor(db, versionSvc, exportSvc)
	maxIterations := 3
	if v := os.Getenv("AGENT_MAX_ITERATIONS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxIterations = parsed
		}
	}
	chatSvc := NewChatService(db, provider, toolExecutor, maxIterations)
	h := NewHandler(sessionSvc, chatSvc)

	rg.POST("/ai/sessions", h.CreateSession)
	rg.GET("/ai/sessions", h.ListSessions)
	rg.GET("/ai/sessions/:session_id", h.GetSession)
	rg.DELETE("/ai/sessions/:session_id", h.DeleteSession)
	rg.POST("/ai/sessions/:session_id/chat", h.Chat)
	rg.GET("/ai/sessions/:session_id/history", h.GetHistory)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
