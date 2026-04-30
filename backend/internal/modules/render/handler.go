package render

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

// Error code constants for the render module (5xxxx series).
const (
	CodeExportFailed    = 5001
	CodeDraftNotFound   = 5002
	CodeVersionNotFound = 5004
	CodeTaskNotFound    = 5005
)

// ---------------------------------------------------------------------------
// Request/response DTOs
// ---------------------------------------------------------------------------

// createVersionReq is the request body for creating a new version snapshot.
type createVersionReq struct {
	Label string `json:"label"`
}

// versionItem is a single version entry in the list response.
type versionItem struct {
	ID        uint   `json:"id"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

// versionListResp is the response for listing versions of a draft.
type versionListResp struct {
	Items []versionItem `json:"items"`
	Total int           `json:"total"`
}

// rollbackReq is the request body for rolling back to a specific version.
type rollbackReq struct {
	VersionID uint `json:"version_id"`
}

// createExportReq is the request body for creating a PDF export task.
type createExportReq struct {
	HTMLContent string `json:"html_content"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler wires version and export HTTP endpoints to their respective services.
type Handler struct {
	versionSvc *VersionService
	exportSvc  *ExportService
}

// NewHandler creates a new Handler.
func NewHandler(versionSvc *VersionService, exportSvc *ExportService) *Handler {
	return &Handler{
		versionSvc: versionSvc,
		exportSvc:  exportSvc,
	}
}

// ---------------------------------------------------------------------------
// Version endpoints
// ---------------------------------------------------------------------------

// ListVersions handles GET /drafts/:draft_id/versions.
func (h *Handler) ListVersions(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, CodeDraftNotFound, "invalid draft_id")
		return
	}

	versions, err := h.versionSvc.ListByDraft(draftID)
	if err != nil {
		response.Error(c, CodeExportFailed, "failed to list versions")
		return
	}

	items := make([]versionItem, 0, len(versions))
	for _, v := range versions {
		label := ""
		if v.Label != nil {
			label = *v.Label
		}
		items = append(items, versionItem{
			ID:        v.ID,
			Label:     label,
			CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	response.Success(c, versionListResp{
		Items: items,
		Total: len(items),
	})
}

// CreateVersion handles POST /drafts/:draft_id/versions.
func (h *Handler) CreateVersion(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, CodeDraftNotFound, "invalid draft_id")
		return
	}

	var req createVersionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeExportFailed, "invalid request body")
		return
	}

	ver, err := h.versionSvc.Create(draftID, req.Label)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		response.Error(c, CodeExportFailed, "failed to create version")
		return
	}

	label := ""
	if ver.Label != nil {
		label = *ver.Label
	}

	response.Success(c, versionItem{
		ID:        ver.ID,
		Label:     label,
		CreatedAt: ver.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Rollback handles POST /drafts/:draft_id/rollback.
func (h *Handler) Rollback(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, CodeDraftNotFound, "invalid draft_id")
		return
	}

	var req rollbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeExportFailed, "invalid request body")
		return
	}

	result, err := h.versionSvc.Rollback(draftID, req.VersionID)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		if errors.Is(err, ErrVersionNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeVersionNotFound, "version not found")
			return
		}
		response.Error(c, CodeExportFailed, "failed to rollback")
		return
	}

	response.Success(c, result)
}

// ---------------------------------------------------------------------------
// Export endpoints
// ---------------------------------------------------------------------------

// CreateExport handles POST /drafts/:draft_id/export.
func (h *Handler) CreateExport(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, CodeDraftNotFound, "invalid draft_id")
		return
	}

	var req createExportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeExportFailed, "invalid request body")
		return
	}

	taskID, err := h.exportSvc.CreateTask(draftID, req.HTMLContent)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		response.Error(c, CodeExportFailed, "failed to create export task")
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
	})
}

// GetTask handles GET /tasks/:task_id.
func (h *Handler) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.exportSvc.GetTask(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeTaskNotFound, "task not found")
			return
		}
		response.Error(c, CodeExportFailed, "failed to get task")
		return
	}

	response.Success(c, task)
}

// DownloadFile handles GET /tasks/:task_id/file.
func (h *Handler) DownloadFile(c *gin.Context) {
	taskID := c.Param("task_id")

	data, err := h.exportSvc.GetFile(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeTaskNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrTaskNotCompleted) {
			response.ErrorWithStatus(c, http.StatusBadRequest, CodeExportFailed, "task not completed yet")
			return
		}
		response.Error(c, CodeExportFailed, "failed to get file")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, taskID))
	c.Data(http.StatusOK, "application/pdf", data)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseUintParam extracts and parses a uint path parameter from the gin context.
func parseUintParam(c *gin.Context, name string) (uint, error) {
	str := c.Param(name)
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}
