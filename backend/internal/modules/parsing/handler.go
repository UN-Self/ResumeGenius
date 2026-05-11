package parsing

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

const (
	CodeParsePDFFailed   = 2001
	CodeParseDOCXFailed  = 2002
	CodeProjectNotFound  = 2003
	CodeNoUsableAssets   = 2004
	CodeGitExtractFailed     = 2007
	CodeGitAIAnalysisFailed  = 2008
	CodeAssetNotFound        = 2009
)

type parseService interface {
	ParseForUser(userID string, projectID uint) ([]ParsedContent, error)
	ParseAssetForUser(userID string, assetID uint, userContext string) (*ParsedContent, error)
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

// ParseAsset handles POST /api/v1/parsing/assets/:asset_id/parse.
func (h *Handler) ParseAsset(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "unauthorized")
		return
	}

	assetID, err := strconv.ParseUint(c.Param("asset_id"), 10, 64)
	if err != nil || assetID == 0 {
		response.Error(c, 40000, "invalid asset_id")
		return
	}

	var req struct {
		UserContext string `json:"user_context"`
	}
	_ = c.ShouldBindJSON(&req) // body is optional

	parsed, err := h.service.ParseAssetForUser(userID, uint(assetID), req.UserContext)
	if err != nil {
		h.respondParseError(c, err)
		return
	}

	response.Success(c, parsed)
}

func (h *Handler) respondParseError(c *gin.Context, err error) {
	log.Printf("[parsing] error: %v", err)
	switch {
	case errors.Is(err, ErrAssetNotFound):
		response.ErrorWithStatus(c, http.StatusNotFound, CodeAssetNotFound, "asset not found")
	case errors.Is(err, ErrProjectNotFound):
		response.ErrorWithStatus(c, http.StatusNotFound, CodeProjectNotFound, "project not found")
	case errors.Is(err, ErrNoUsableAssets):
		response.Error(c, CodeNoUsableAssets, "project has no usable assets")
	case errors.Is(err, ErrAssetURIMissing), errors.Is(err, ErrAssetContentMissing):
		response.Error(c, 2006, "project contains invalid asset data")
	case errors.Is(err, ErrPDFParseFailed):
		response.Error(c, CodeParsePDFFailed, "pdf parse failed")
	case errors.Is(err, ErrDOCXParseFailed):
		response.Error(c, CodeParseDOCXFailed, "docx parse failed")
	case errors.Is(err, ErrGitExtractFailed):
		response.Error(c, CodeGitExtractFailed, "git repository extract failed")
	case errors.Is(err, ErrGitAIAnalysisFailed):
		response.Error(c, CodeGitAIAnalysisFailed, "git repository AI analysis failed")
	case errors.Is(err, ErrAssetTypeSkipped), errors.Is(err, ErrUnsupportedAssetType):
		response.Error(c, 2006, "asset type is not supported for parsing")
	default:
		response.Error(c, 50000, "internal server error")
	}
}
