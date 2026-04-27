package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/handy/resume-genius/internal/modules/agent"
	"github.com/handy/resume-genius/internal/modules/intake"
	"github.com/handy/resume-genius/internal/modules/parsing"
	"github.com/handy/resume-genius/internal/modules/render"
	"github.com/handy/resume-genius/internal/modules/workbench"
	"github.com/handy/resume-genius/internal/shared/database"
	"github.com/handy/resume-genius/internal/shared/middleware"
)

var _ *gorm.DB // ensure gorm import is used

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

	v1 := r.Group("/api/v1")
	intake.RegisterRoutes(v1, db)
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
