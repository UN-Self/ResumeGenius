package intake

import (
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, uploadDir string) {
	store := storage.NewLocalStorage(uploadDir)
	projectSvc := NewProjectService(db)
	assetSvc := NewAssetService(db, store)
	h := NewHandler(projectSvc, assetSvc)

	// Project CRUD
	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects", h.ListProjects)
	rg.GET("/projects/:project_id", h.GetProject)
	rg.DELETE("/projects/:project_id", h.DeleteProject)

	// Asset management
	rg.POST("/assets/upload", h.UploadFile)
	rg.POST("/assets/folders", h.CreateFolder)
	rg.POST("/assets/git", h.CreateGitRepo)
	rg.GET("/assets", h.ListAssets)
	rg.GET("/assets/:asset_id/file", h.GetAssetFile)
	rg.DELETE("/assets/:asset_id", h.DeleteAsset)
	rg.PATCH("/assets/:asset_id", h.UpdateAsset)
	rg.PATCH("/assets/:asset_id/move", h.MoveAsset)

	// Notes
	rg.POST("/assets/notes", h.CreateNote)
	rg.PUT("/assets/notes/:note_id", h.UpdateNote)
}
