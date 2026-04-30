package intake

import (
	"errors"
	"strconv"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// Error codes for intake module (01xxx)
const (
	CodeParamInvalid      = 10001 // request parameter invalid
	CodeInternalError     = 50001 // internal server error
	CodeUnsupportedFormat = 1001  // unsupported file format
	CodeFileTooLarge      = 1002  // file size exceeds limit
	CodeInvalidGitURL     = 1003  // invalid git URL
	CodeProjectNotFound   = 1004  // project not found
	CodeAssetNotFound     = 1006  // asset not found
)

type Handler struct {
	projectSvc *ProjectService
	assetSvc   *AssetService
}

func NewHandler(projectSvc *ProjectService, assetSvc *AssetService) *Handler {
	return &Handler{projectSvc: projectSvc, assetSvc: assetSvc}
}

func userID(c *gin.Context) string {
	return middleware.UserIDFromContext(c)
}

// --- Request structs ---

type createProjectReq struct {
	Title string `json:"title" binding:"required"`
}

type createGitReq struct {
	ProjectID uint   `json:"project_id"`
	RepoURL   string `json:"repo_url"`
}

type createNoteReq struct {
	ProjectID uint   `json:"project_id"`
	Content   string `json:"content"`
	Label     string `json:"label"`
}

type updateNoteReq struct {
	Content string `json:"content"`
	Label   string `json:"label"`
}

// --- Handlers ---

// CreateProject handles POST /projects
func (h *Handler) CreateProject(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "title is required")
		return
	}

	proj, err := h.projectSvc.Create(userID(c), req.Title)
	if err != nil {
		response.Error(c, CodeInternalError, "failed to create project")
		return
	}

	response.Success(c, proj)
}

// ListProjects handles GET /projects
func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.projectSvc.List(userID(c))
	if err != nil {
		response.Error(c, CodeInternalError, "failed to list projects")
		return
	}

	response.Success(c, projects)
}

// GetProject handles GET /projects/:project_id
func (h *Handler) GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("project_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid project_id")
		return
	}

	proj, err := h.projectSvc.GetByID(userID(c), uint(id))
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			response.Error(c, CodeProjectNotFound, "project not found")
			return
		}
		response.Error(c, CodeInternalError, "failed to get project")
		return
	}

	response.Success(c, proj)
}

// DeleteProject handles DELETE /projects/:project_id
func (h *Handler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("project_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid project_id")
		return
	}

	uid := userID(c)
	projectID := uint(id)

	// DeleteProject handles full cascade (assets, drafts, versions, sessions, messages)
	if err := h.projectSvc.Delete(uid, projectID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			response.Error(c, CodeProjectNotFound, "project not found")
		} else {
			response.Error(c, CodeInternalError, "delete failed")
		}
		return
	}

	response.Success(c, nil)
}

// UploadFile handles POST /assets/upload
func (h *Handler) UploadFile(c *gin.Context) {
	projectIDStr := c.PostForm("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid project_id")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, CodeParamInvalid, "file is required")
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Error(c, CodeInternalError, "failed to read file")
		return
	}
	defer src.Close()

	data := make([]byte, file.Size)
	if _, err := src.Read(data); err != nil {
		response.Error(c, CodeInternalError, "failed to read file")
		return
	}

	asset, err := h.assetSvc.UploadFile(userID(c), uint(projectID), file.Filename, data, file.Size)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedFormat):
			response.Error(c, CodeUnsupportedFormat, "unsupported file format")
		case errors.Is(err, ErrFileTooLarge):
			response.Error(c, CodeFileTooLarge, "file size exceeds 20MB limit")
		case errors.Is(err, ErrProjectNotFound):
			response.Error(c, CodeProjectNotFound, "project not found")
		default:
			response.Error(c, CodeInternalError, "failed to upload file")
		}
		return
	}

	response.Success(c, asset)
}

// CreateGitRepo handles POST /assets/git
func (h *Handler) CreateGitRepo(c *gin.Context) {
	var req createGitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "invalid request body")
		return
	}

	asset, err := h.assetSvc.CreateGitRepo(userID(c), req.ProjectID, req.RepoURL)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidGitURL):
			response.Error(c, CodeInvalidGitURL, "invalid git repository URL")
		case errors.Is(err, ErrProjectNotFound):
			response.Error(c, CodeProjectNotFound, "project not found")
		default:
			response.Error(c, CodeInternalError, "failed to create git repo")
		}
		return
	}

	response.Success(c, asset)
}

// ListAssets handles GET /assets?project_id=X
func (h *Handler) ListAssets(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid project_id")
		return
	}

	assets, err := h.assetSvc.ListByProject(userID(c), uint(projectID))
	if err != nil {
		switch {
		case errors.Is(err, ErrProjectNotFound):
			response.Error(c, CodeProjectNotFound, "project not found")
		default:
			response.Error(c, CodeInternalError, "failed to list assets")
		}
		return
	}

	response.Success(c, assets)
}

// DeleteAsset handles DELETE /assets/:asset_id
func (h *Handler) DeleteAsset(c *gin.Context) {
	assetID, err := strconv.ParseUint(c.Param("asset_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid asset_id")
		return
	}

	if err := h.assetSvc.DeleteAsset(userID(c), uint(assetID)); err != nil {
		switch {
		case errors.Is(err, ErrAssetNotFound):
			response.Error(c, CodeAssetNotFound, "asset not found")
		case errors.Is(err, ErrProjectNotFound):
			response.Error(c, CodeProjectNotFound, "project not found")
		default:
			response.Error(c, CodeInternalError, "failed to delete asset")
		}
		return
	}

	response.Success(c, nil)
}

// CreateNote handles POST /assets/notes
func (h *Handler) CreateNote(c *gin.Context) {
	var req createNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "invalid request body")
		return
	}

	asset, err := h.assetSvc.CreateNote(userID(c), req.ProjectID, req.Content, req.Label)
	if err != nil {
		switch {
		case errors.Is(err, ErrProjectNotFound):
			response.Error(c, CodeProjectNotFound, "project not found")
		default:
			response.Error(c, CodeInternalError, "failed to create note")
		}
		return
	}

	response.Success(c, asset)
}

// UpdateNote handles PUT /assets/notes/:note_id
func (h *Handler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("note_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid note_id")
		return
	}

	var req updateNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "invalid request body")
		return
	}

	asset, err := h.assetSvc.UpdateNote(userID(c), uint(noteID), req.Content, req.Label)
	if err != nil {
		switch {
		case errors.Is(err, ErrAssetNotFound):
			response.Error(c, CodeAssetNotFound, "asset not found")
		case errors.Is(err, ErrProjectNotFound):
			response.Error(c, CodeProjectNotFound, "project not found")
		default:
			response.Error(c, CodeInternalError, "failed to update note")
		}
		return
	}

	response.Success(c, asset)
}
