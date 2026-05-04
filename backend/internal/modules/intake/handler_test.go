package intake

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	sharedstorage "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

// --- Test helpers ---

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	return setupTestHandlerWithStorage(t, nil)
}

func setupTestHandlerWithStorage(t *testing.T, store sharedstorage.FileStorage) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)
	if store == nil {
		store = NewLocalStorage(t.TempDir())
	}
	h := NewHandler(NewProjectService(db), NewAssetService(db, store))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ContextUserID, "test-user-1"); c.Next() })
	return h, r
}

func createMultipartForm(t *testing.T, fieldName, fileName string, content []byte, extraFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	for k, v := range extraFields {
		err := writer.WriteField(k, v)
		require.NoError(t, err)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

// Helper to create a project via the handler's service directly (avoids HTTP overhead for setup)
func createTestProject(t *testing.T, h *Handler, title string) uint {
	t.Helper()
	proj, err := h.projectSvc.Create("test-user-1", title)
	require.NoError(t, err)
	return proj.ID
}

// Helper to make JSON request and return response recorder
func doJSON(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Helper to parse APIResponse
type apiResponse struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

func parseDataArray(t *testing.T, raw json.RawMessage) []json.RawMessage {
	t.Helper()
	var arr []json.RawMessage
	err := json.Unmarshal(raw, &arr)
	require.NoError(t, err)
	return arr
}

func parseDataMap(t *testing.T, raw json.RawMessage) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	err := json.Unmarshal(raw, &m)
	require.NoError(t, err)
	return m
}

// --- Tests ---

func TestHandler_CreateProject(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/projects", h.CreateProject)

	w := doJSON(t, r, "POST", "/projects", map[string]string{"title": "My Resume"})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "My Resume", data["title"])
	assert.NotEmpty(t, data["id"])
}

func TestHandler_CreateProject_EmptyTitle(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/projects", h.CreateProject)

	w := doJSON(t, r, "POST", "/projects", map[string]string{"title": ""})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeParamInvalid, resp.Code)
}

func TestHandler_ListProjects(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/projects", h.CreateProject)
	r.GET("/projects", h.ListProjects)

	// Create two projects
	doJSON(t, r, "POST", "/projects", map[string]string{"title": "Project A"})
	doJSON(t, r, "POST", "/projects", map[string]string{"title": "Project B"})

	w := doJSON(t, r, "GET", "/projects", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	arr := parseDataArray(t, resp.Data)
	assert.Len(t, arr, 2)
}

func TestHandler_GetProject(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/projects", h.CreateProject)
	r.GET("/projects/:project_id", h.GetProject)

	projID := createTestProject(t, h, "Target Project")

	w := doJSON(t, r, "GET", "/projects/"+strconv.Itoa(int(projID)), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "Target Project", data["title"])
}

func TestHandler_DeleteProject(t *testing.T) {
	h, r := setupTestHandler(t)
	r.DELETE("/projects/:project_id", h.DeleteProject)
	r.GET("/projects/:project_id", h.GetProject)

	projID := createTestProject(t, h, "To Be Deleted")

	w := doJSON(t, r, "DELETE", "/projects/"+strconv.Itoa(int(projID)), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, parseResponse(t, w).Code)

	// Verify deleted: GetProject returns CodeProjectNotFound (1004) with HTTP 400
	// because error code 1004 maps to HTTP 400 per response.httpStatusFromCode
	w2 := doJSON(t, r, "GET", "/projects/"+strconv.Itoa(int(projID)), nil)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	resp := parseResponse(t, w2)
	assert.Equal(t, CodeProjectNotFound, resp.Code)
}

func TestHandler_DeleteProject_ReturnsErrorWhenDeleteProjectAssetsFails(t *testing.T) {
	h, r := setupTestHandlerWithStorage(t, newFailingDeleteStorage(errors.New("disk busy")))
	r.DELETE("/projects/:project_id", h.DeleteProject)

	projID := createTestProject(t, h, "Delete Fails")
	_, err := h.assetSvc.UploadFile("test-user-1", projID, "resume.pdf", []byte("%PDF-1.4 fake content"), 10)
	require.NoError(t, err)

	w := doJSON(t, r, "DELETE", "/projects/"+strconv.Itoa(int(projID)), nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeInternalError, resp.Code)
	assert.Equal(t, "failed to delete project assets", resp.Message)

	proj, err := h.projectSvc.GetByID("test-user-1", projID)
	require.NoError(t, err)
	assert.Equal(t, projID, proj.ID)

	assets, err := h.assetSvc.ListByProject("test-user-1", projID)
	require.NoError(t, err)
	assert.Len(t, assets, 1)
}

func TestHandler_UploadFile(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/assets/upload", h.UploadFile)

	projID := createTestProject(t, h, "Upload Project")

	body, ct := createMultipartForm(t, "file", "resume.pdf", []byte("%PDF-1.4 fake content"), map[string]string{
		"project_id": strconv.Itoa(int(projID)),
	})

	req, err := http.NewRequest("POST", "/assets/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", ct)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "resume_pdf", data["type"])
}

func TestHandler_UploadFile_UnsupportedFormat(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/assets/upload", h.UploadFile)

	projID := createTestProject(t, h, "Upload Project")

	body, ct := createMultipartForm(t, "file", "malware.exe", []byte("exe content"), map[string]string{
		"project_id": strconv.Itoa(int(projID)),
	})

	req, err := http.NewRequest("POST", "/assets/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", ct)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeUnsupportedFormat, resp.Code)
}

func TestHandler_CreateGitRepo(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/assets/git", h.CreateGitRepo)

	projID := createTestProject(t, h, "Git Project")

	w := doJSON(t, r, "POST", "/assets/git", map[string]interface{}{
		"project_id": projID,
		"repo_url":   "https://github.com/example/resume.git",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "git_repo", data["type"])
}

func TestHandler_ListAssets(t *testing.T) {
	h, r := setupTestHandler(t)
	r.GET("/assets", h.ListAssets)

	projID := createTestProject(t, h, "Asset Project")

	// Create some assets via service
	_, err := h.assetSvc.CreateNote("test-user-1", projID, "Note 1", "Label 1")
	require.NoError(t, err)
	_, err = h.assetSvc.CreateNote("test-user-1", projID, "Note 2", "Label 2")
	require.NoError(t, err)

	w := doJSON(t, r, "GET", "/assets?project_id="+strconv.Itoa(int(projID)), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	arr := parseDataArray(t, resp.Data)
	assert.Len(t, arr, 2)
}

func TestHandler_DeleteAsset(t *testing.T) {
	h, r := setupTestHandler(t)
	r.DELETE("/assets/:asset_id", h.DeleteAsset)

	projID := createTestProject(t, h, "Delete Asset Project")

	asset, err := h.assetSvc.CreateNote("test-user-1", projID, "To delete", "Delete me")
	require.NoError(t, err)

	w := doJSON(t, r, "DELETE", "/assets/"+strconv.Itoa(int(asset.ID)), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, parseResponse(t, w).Code)

	// Verify the asset is gone by trying to delete again
	w2 := doJSON(t, r, "DELETE", "/assets/"+strconv.Itoa(int(asset.ID)), nil)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Equal(t, CodeAssetNotFound, parseResponse(t, w2).Code)
}

func TestHandler_UpdateAsset(t *testing.T) {
	h, r := setupTestHandler(t)
	r.PATCH("/assets/:asset_id", h.UpdateAsset)

	projID := createTestProject(t, h, "Update Asset Project")

	content := "Original parsed text"
	label := "Original label"
	asset := models.Asset{
		ProjectID: projID,
		Type:      "resume_pdf",
		Content:   &content,
		Label:     &label,
	}
	require.NoError(t, h.assetSvc.db.Create(&asset).Error)

	w := doJSON(t, r, "PATCH", "/assets/"+strconv.Itoa(int(asset.ID)), map[string]interface{}{
		"content": "Updated parsed text",
		"label":   "Updated label",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "Updated parsed text", data["content"])
	assert.Equal(t, "Updated label", data["label"])
	assert.Equal(t, "resume_pdf", data["type"])
}

func TestHandler_UpdateAsset_RequiresEditableFields(t *testing.T) {
	h, r := setupTestHandler(t)
	r.PATCH("/assets/:asset_id", h.UpdateAsset)

	w := doJSON(t, r, "PATCH", "/assets/1", map[string]interface{}{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, CodeParamInvalid, parseResponse(t, w).Code)
}

func TestHandler_CreateNote(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/assets/notes", h.CreateNote)

	projID := createTestProject(t, h, "Note Project")

	w := doJSON(t, r, "POST", "/assets/notes", map[string]interface{}{
		"project_id": projID,
		"content":    "Candidate has 5 years of Go experience",
		"label":      "Background",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "note", data["type"])
	assert.Equal(t, "Candidate has 5 years of Go experience", data["content"])
	assert.Equal(t, "Background", data["label"])
}

func TestHandler_UpdateNote(t *testing.T) {
	h, r := setupTestHandler(t)
	r.PUT("/assets/notes/:note_id", h.UpdateNote)

	projID := createTestProject(t, h, "Update Note Project")

	asset, err := h.assetSvc.CreateNote("test-user-1", projID, "Original", "Label1")
	require.NoError(t, err)

	w := doJSON(t, r, "PUT", "/assets/notes/"+strconv.Itoa(int(asset.ID)), map[string]interface{}{
		"content": "Updated content",
		"label":   "Label2",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := parseDataMap(t, resp.Data)
	assert.Equal(t, "Updated content", data["content"])
	assert.Equal(t, "Label2", data["label"])
}
