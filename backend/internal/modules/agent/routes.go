package agent

import (
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func normalizeAIURL(raw string) string {
	raw = strings.TrimSpace(raw)
	// 如果已包含完整 /chat/completions 路径，直接使用（兼容智谱等非标准版本号）
	if strings.Contains(raw, "/chat/completions") {
		return raw
	}
	// 否则追加默认路径（兼容 OpenAI 标准 API）
	return strings.TrimRight(raw, "/") + "/v1/chat/completions"
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	sessionSvc := NewSessionService(db)

	var provider ProviderAdapter
	if os.Getenv("USE_MOCK") == "true" {
		provider = &MockAdapter{}
	} else {
		provider = NewOpenAIAdapter(
			normalizeAIURL(os.Getenv("AI_API_URL")),
			os.Getenv("AI_API_KEY"),
			envOrDefault("AI_MODEL", "default"),
		)
	}

	toolExecutor := NewAgentToolExecutor(db)
	maxIterations := 3
	if v := os.Getenv("AGENT_MAX_ITERATIONS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxIterations = parsed
		}
	}
	chatSvc := NewChatService(db, provider, toolExecutor, maxIterations)
	editSvc := NewEditService(db)
	h := NewHandler(sessionSvc, chatSvc, editSvc)

	rg.POST("/ai/sessions", h.CreateSession)
	rg.GET("/ai/sessions", h.ListSessions)
	rg.GET("/ai/sessions/:session_id", h.GetSession)
	rg.DELETE("/ai/sessions/:session_id", h.DeleteSession)
	rg.POST("/ai/sessions/:session_id/chat", h.Chat)
	rg.GET("/ai/sessions/:session_id/history", h.GetHistory)
	rg.POST("/ai/drafts/:draft_id/undo", h.Undo)
	rg.POST("/ai/drafts/:draft_id/redo", h.Redo)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
