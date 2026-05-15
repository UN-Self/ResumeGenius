package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/modules/agent"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/auth"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/intake"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/parsing"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/render"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/workbench"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/database"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/env"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

func jwtSecret() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is required")
	}
	return secret, nil
}

func jwtTTL() time.Duration {
	hours := 24 * 365
	if v := os.Getenv("JWT_TTL_HOURS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err == nil && parsed > 0 {
			hours = parsed
		}
	}
	return time.Duration(hours) * time.Hour
}

func cookieSecure() bool {
	v := os.Getenv("COOKIE_SECURE")
	if v == "" {
		return false
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return parsed
}

func setupRouter(db *gorm.DB) (*gin.Engine, func()) {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

	r.GET("/health", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(uploadDir, 0755)

	store := storage.NewLocalStorage(uploadDir)

	v1 := r.Group("/api/v1")
	secret, err := jwtSecret()
	if err != nil {
		log.Fatalf("invalid auth config: %v", err)
	}
	ttl := jwtTTL()
	secure := cookieSecure()

	authed := v1.Group("")
	authed.Use(middleware.AuthRequired(secret))
	auth.RegisterRoutes(v1, authed, db, secret, ttl, secure)
	aesKey := os.Getenv("SSH_KEY_AES_KEY")
	if aesKey == "" {
		log.Fatal("SSH_KEY_AES_KEY environment variable is required")
	}
	if len(aesKey) != 16 && len(aesKey) != 24 && len(aesKey) != 32 {
		log.Fatalf("SSH_KEY_AES_KEY must be 16, 24, or 32 bytes, got %d", len(aesKey))
	}
	sshSvc := intake.RegisterRoutes(authed, db, uploadDir, aesKey)
	parsing.RegisterRoutes(authed, db, store)

	// 先创建 render services
	_, _, renderCleanup := render.NewServices(db, store)

	// agent 模块：创建 extract_git_repo 工具并注入
	extractModel := env.DefaultOr("GIT_EXTRACT_MODEL", "haiku")

	sizeLimitMB := 50
	if v := env.DefaultOr("GIT_REPO_SIZE_LIMIT_MB", "50"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			sizeLimitMB = parsed
		}
	}
	extractTool := agent.NewExtractGitRepoTool(db, sshSvc, agent.NewOpenAIProvider(), extractModel, sizeLimitMB)
	agent.RegisterRoutes(authed, db, extractTool)
	workbench.RegisterRoutes(authed, db)

	// render 路由注册（复用已有 services）
	render.RegisterRoutes(authed, db, store)

	return r, renderCleanup
}

func main() {
	// Load env files from either repo root or backend cwd.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	db := database.Connect()
	database.Migrate(db)

	r, cleanup := setupRouter(db)
	defer cleanup()

	log.Println("server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
