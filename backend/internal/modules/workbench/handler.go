package workbench

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

// Handler handles HTTP requests for draft operations.
type Handler struct {
	service *DraftService
}

// NewHandler creates a new Handler instance.
func NewHandler(service *DraftService) *Handler {
	return &Handler{service: service}
}

// GetDraftResponse represents the response for GET /drafts/:draft_id
type GetDraftResponse struct {
	ID          uint   `json:"id"`
	ProjectID   uint   `json:"project_id"`
	HTMLContent string `json:"html_content"`
	UpdatedAt   string `json:"updated_at"`
}

// UpdateDraftRequest represents the request body for PUT /drafts/:draft_id
type UpdateDraftRequest struct {
	HTMLContent   string `json:"html_content"`
	CreateVersion bool   `json:"create_version"`
	VersionLabel  string `json:"version_label"`
}

// UpdateDraftResponse represents the response for PUT /drafts/:draft_id
type UpdateDraftResponse struct {
	ID        uint   `json:"id"`
	UpdatedAt string `json:"updated_at"`
	VersionID *uint  `json:"version_id,omitempty"`
}

// GetDraft handles GET /drafts/:draft_id
func (h *Handler) GetDraft(c *gin.Context) {
	draftIDStr := c.Param("draft_id")
	draftID, err := strconv.ParseUint(draftIDStr, 10, 32)
	if err != nil {
		response.Error(c, 40000, "invalid draft id")
		return
	}

	draft, err := h.service.GetByID(uint(draftID))
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, 4001, "draft not found")
			return
		}
		response.Error(c, 50000, "internal server error")
		return
	}

	response.Success(c, GetDraftResponse{
		ID:          draft.ID,
		ProjectID:   draft.ProjectID,
		HTMLContent: draft.HTMLContent,
		UpdatedAt:   draft.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

// UpdateDraft handles PUT /drafts/:draft_id
func (h *Handler) UpdateDraft(c *gin.Context) {
	draftIDStr := c.Param("draft_id")
	draftID, err := strconv.ParseUint(draftIDStr, 10, 32)
	if err != nil {
		response.Error(c, 40000, "invalid draft id")
		return
	}

	var req UpdateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "invalid request body")
		return
	}

	draft, versionID, err := h.service.Update(uint(draftID), req.HTMLContent, req.CreateVersion, req.VersionLabel)
	if err != nil {
		if errors.Is(err, ErrHTMLContentEmpty) {
			response.ErrorWithStatus(c, http.StatusBadRequest, 4002, "html content empty")
			return
		}
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, 4001, "draft not found")
			return
		}
		response.Error(c, 50000, "internal server error")
		return
	}

	response.Success(c, UpdateDraftResponse{
		ID:        draft.ID,
		UpdatedAt: draft.UpdatedAt.UTC().Format(time.RFC3339),
		VersionID: versionID,
	})
}
