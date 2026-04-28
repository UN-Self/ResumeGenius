package intake

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, uploadDir string) {
	storage := NewLocalStorage(uploadDir)
	projectSvc := NewProjectService(db)
	assetSvc := NewAssetService(db, storage)
	h := NewHandler(projectSvc, assetSvc)

	// Project CRUD
	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects", h.ListProjects)
	rg.GET("/projects/:project_id", h.GetProject)
	rg.DELETE("/projects/:project_id", h.DeleteProject)

	// Asset management
	rg.POST("/assets/upload", h.UploadFile)
	rg.POST("/assets/git", h.CreateGitRepo)
	rg.GET("/assets", h.ListAssets)
	rg.DELETE("/assets/:asset_id", h.DeleteAsset)

	// Notes
	rg.POST("/assets/notes", h.CreateNote)
	rg.PUT("/assets/notes/:note_id", h.UpdateNote)
}
