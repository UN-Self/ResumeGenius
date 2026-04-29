package parsing

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

const (
	CodeParsePDFFailed   = 2001
	CodeParseDOCXFailed  = 2002
	CodeProjectNotFound  = 2003
	CodeNoUsableAssets   = 2004
	CodeAIGenerateFailed = 2005
	CodeInvalidAssetData = 2006
)

type parseService interface {
	ParseForUser(userID string, projectID uint) ([]ParsedContent, error)
	GenerateForUser(userID string, projectID uint) (*GenerateResult, error)
}

// Handler handles parsing HTTP requests.
type Handler struct {
	service parseService
}

// NewHandler creates a parsing handler.
func NewHandler(service parseService) *Handler {
	return &Handler{service: service}
}

type ParseRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
}

type ParseResponse struct {
	ParsedContents []ParsedContent `json:"parsed_contents"`
}

// Parse handles POST /api/v1/parsing/parse.
func (h *Handler) Parse(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "unauthorized")
		return
	}

	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ProjectID == 0 {
		response.Error(c, 40000, "invalid request body")
		return
	}

	parsedContents, err := h.service.ParseForUser(userID, req.ProjectID)
	if err != nil {
		h.respondParseError(c, err)
		return
	}

	response.Success(c, ParseResponse{
		ParsedContents: parsedContents,
	})
}

// Generate handles POST /api/v1/parsing/generate.
func (h *Handler) Generate(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "unauthorized")
		return
	}

	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ProjectID == 0 {
		response.Error(c, 40000, "invalid request body")
		return
	}

	result, err := h.service.GenerateForUser(userID, req.ProjectID)
	if err != nil {
		h.respondParseError(c, err)
		return
	}

	response.Success(c, result)
}

func (h *Handler) respondParseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProjectNotFound):
		response.ErrorWithStatus(c, http.StatusNotFound, CodeProjectNotFound, "project not found")
	case errors.Is(err, ErrNoUsableAssets):
		response.Error(c, CodeNoUsableAssets, "project has no usable assets")
	case errors.Is(err, ErrAssetURIMissing), errors.Is(err, ErrAssetContentMissing):
		response.Error(c, CodeInvalidAssetData, "project contains invalid asset data")
	case errors.Is(err, ErrNoGeneratableText):
		response.Error(c, CodeInvalidAssetData, "project has no usable text content")
	case errors.Is(err, ErrPDFParseFailed):
		response.Error(c, CodeParsePDFFailed, "pdf parse failed")
	case errors.Is(err, ErrDOCXParseFailed):
		response.Error(c, CodeParseDOCXFailed, "docx parse failed")
	case errors.Is(err, ErrAIGenerateFailed):
		response.ErrorWithStatus(c, http.StatusInternalServerError, CodeAIGenerateFailed, "ai draft generation failed")
	default:
		response.Error(c, 50000, "internal server error")
	}
}
