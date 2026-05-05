package render

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all render module endpoints.
// Accepts pre-created services from main.go (lifecycle managed there).
func RegisterRoutes(rg *gin.RouterGroup, versionSvc *VersionService, exportSvc *ExportService) {
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
