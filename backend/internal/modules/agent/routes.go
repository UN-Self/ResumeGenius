package agent

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/env"
)

func normalizeAIURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/chat/completions") {
		return raw
	}
	if strings.HasSuffix(raw, "/v1") || strings.HasSuffix(raw, "/api/paas/v4") {
		return raw + "/chat/completions"
	}
	return raw + "/v1/chat/completions"
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, extractGitTool *ExtractGitRepoTool) {
	skillLoader, err := NewSkillLoader()
	if err != nil {
		log.Printf("agent: failed to load skills (non-fatal): %v", err)
		skillLoader = nil
	}

	sessionSvc := NewSessionService(db)

	var provider ProviderAdapter
	if os.Getenv("USE_MOCK") == "true" {
		provider = &MockAdapter{}
	} else {
		apiURL := strings.TrimSpace(os.Getenv("AI_API_URL"))
		apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY"))
		if apiURL == "" || apiKey == "" {
			log.Println("agent: AI_API_URL or AI_API_KEY is missing; using mock AI provider")
			provider = &MockAdapter{}
		} else {
			provider = NewOpenAIAdapterWithTimeout(
				normalizeAIURL(apiURL),
				apiKey,
				env.DefaultOr("AI_MODEL", "default"),
				envDurationSeconds("AI_HTTP_TIMEOUT_SECONDS", 180),
			)
		}
	}

	toolExecutor := NewAgentToolExecutor(db, skillLoader, extractGitTool)
	maxIterations := 3
	if v := os.Getenv("AGENT_MAX_ITERATIONS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxIterations = parsed
		}
	}
	chatSvc := NewChatService(db, provider, toolExecutor, maxIterations, skillLoader)
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

func envDurationSeconds(key string, fallbackSeconds int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			return time.Duration(parsed) * time.Second
		}
	}
	return time.Duration(fallbackSeconds) * time.Second
}
