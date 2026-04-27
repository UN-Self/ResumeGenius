package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/handy/resume-genius/internal/modules/a_intake"
	"github.com/handy/resume-genius/internal/modules/b_parsing"
	"github.com/handy/resume-genius/internal/modules/c_agent"
	"github.com/handy/resume-genius/internal/modules/d_workbench"
	"github.com/handy/resume-genius/internal/modules/e_render"
	"github.com/handy/resume-genius/internal/shared/database"
	"github.com/handy/resume-genius/internal/shared/middleware"
)

var _ *gorm.DB // ensure gorm import is used

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

	v1 := r.Group("/api/v1")
	a_intake.RegisterRoutes(v1, db)
	b_parsing.RegisterRoutes(v1, db)
	c_agent.RegisterRoutes(v1, db)
	d_workbench.RegisterRoutes(v1, db)
	e_render.RegisterRoutes(v1, db)

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
