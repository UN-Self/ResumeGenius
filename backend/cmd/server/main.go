package main

import (
	"fmt"
	"log"
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
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

var _ *gorm.DB // ensure gorm import is used

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

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

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
	intake.RegisterRoutes(authed, db, uploadDir)
	parsing.RegisterRoutes(authed, db, store)
	agent.RegisterRoutes(authed, db)
	workbench.RegisterRoutes(authed, db)
	render.RegisterRoutes(authed, db, store)

	return r
}

func main() {
	// Load .env — search parent dirs so go run works from any subdirectory
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(p); err == nil {
			log.Printf("loaded env from %s", p)
			break
		}
	}

	db := database.Connect()
	database.Migrate(db)

	r := setupRouter(db)

	log.Println("server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
