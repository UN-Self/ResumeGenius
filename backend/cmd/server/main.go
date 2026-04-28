package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/modules/agent"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/intake"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/parsing"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/render"
	"github.com/UN-Self/ResumeGenius/backend/internal/modules/workbench"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/database"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
)

var _ *gorm.DB // ensure gorm import is used

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.UserIdentify(), middleware.Logger())

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(uploadDir, 0755)

	v1 := r.Group("/api/v1")
	intake.RegisterRoutes(v1, db, uploadDir)
	parsing.RegisterRoutes(v1, db)
	agent.RegisterRoutes(v1, db)
	workbench.RegisterRoutes(v1, db)
	render.RegisterRoutes(v1, db)

	return r
}

func main() {
	db := database.Connect()
	database.Migrate(db)

	r := setupRouter(db)

	log.Println("server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
