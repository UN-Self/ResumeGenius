package agent

import (
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
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

	chatSvc := NewChatService(db, provider)
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
