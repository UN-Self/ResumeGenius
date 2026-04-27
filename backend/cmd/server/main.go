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

func main() {
	db := database.Connect()
	database.Migrate(db)

	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

	v1 := r.Group("/api/v1")
	a_intake.RegisterRoutes(v1.Group("/intake"), db)
	b_parsing.RegisterRoutes(v1.Group("/parsing"), db)
	c_agent.RegisterRoutes(v1.Group("/ai"), db)
	d_workbench.RegisterRoutes(v1.Group("/workbench"), db)
	e_render.RegisterRoutes(v1.Group("/render"), db)

	log.Println("server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
