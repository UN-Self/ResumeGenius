package render

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

// RegisterRoutes registers all render module endpoints.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	versionSvc := NewVersionService(db)

	exporter := NewChromeExporter()
	exportSvc := NewExportService(exporter, store)
	exportSvc.db = db

	h := NewHandler(versionSvc, exportSvc)

	// Version management
	rg.GET("/drafts/:draft_id/versions", h.ListVersions)
	rg.POST("/drafts/:draft_id/versions", h.CreateVersion)
	rg.POST("/drafts/:draft_id/rollback", h.Rollback)

	// PDF export
	rg.POST("/drafts/:draft_id/export", h.CreateExport)
	rg.GET("/tasks/:task_id", h.GetTask)
	rg.GET("/tasks/:task_id/file", h.DownloadFile)
}
