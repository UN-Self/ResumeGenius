package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

// ---------------------------------------------------------------------------
// Response struct for parsing API responses
// ---------------------------------------------------------------------------

type apiResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupRouter creates a test gin router with render handler routes registered.
// It sets up DB, VersionService, ExportService (MockExporter + LocalStorage temp dir),
// injects DB into ExportService, creates Handler, and registers a middleware that
// sets middleware.ContextUserID = "test-user-1".
// Returns router, handler, and db.
func setupRouter(t *testing.T) (*gin.Engine, *Handler, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	db := SetupTestDB(t)

	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir)
	mock := &MockExporter{
		PDFBytes: samplePDF(t),
	}
	exportSvc := NewExportService(mock, store)
	exportSvc.db = db
	t.Cleanup(func() { exportSvc.Close() })

	versionSvc := NewVersionService(db)
	h := NewHandler(versionSvc, exportSvc)

	// Middleware to set user context
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "test-user-1")
		c.Next()
	})

	// Register routes
	drafts := r.Group("/drafts")
	{
		d := drafts.Group("/:draft_id")
		{
			d.GET("/versions", h.ListVersions)
			d.POST("/versions", h.CreateVersion)
			d.POST("/rollback", h.Rollback)
			d.POST("/export", h.CreateExport)
		}
	}

	tasks := r.Group("/tasks")
	{
		tasks.GET("/:task_id", h.GetTask)
		tasks.GET("/:task_id/file", h.DownloadFile)
	}

	return r, h, db
}

// doJSON performs a JSON request on the test router and returns the response recorder.
func doJSON(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseResp parses the API response from a response recorder.
func parseResp(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body: %s", w.Body.String())
	return resp
}

// seedVersion creates a version via VersionService for a given draft.
func seedVersion(t *testing.T, db *gorm.DB, draftID uint, label string) *models.Version {
	t.Helper()
	svc := NewVersionService(db)
	ver, err := svc.Create(draftID, label)
	require.NoError(t, err)
	return ver
}

// ---------------------------------------------------------------------------
// Version endpoints
// ---------------------------------------------------------------------------

func TestHandler_ListVersions(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	// Create two versions
	seedVersion(t, db, draft.ID, "v1")
	seedVersion(t, db, draft.ID, "v2")

	w := doJSON(t, r, "GET", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/versions", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "data should be a map")
	assert.Equal(t, float64(2), data["total"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 2)

	// v2 should be first (ORDER BY created_at DESC)
	first := items[0].(map[string]interface{})
	assert.Equal(t, "v2", first["label"])
}

func TestHandler_CreateVersion(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/versions", map[string]string{
		"label": "初始版本",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "初始版本", data["label"])
	assert.NotZero(t, data["id"])
	assert.NotZero(t, data["created_at"])
}

func TestHandler_CreateVersion_DefaultLabel(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	// POST with empty label
	w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/versions", map[string]string{
		"label": "",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "手动保存", data["label"])
}

func TestHandler_Rollback(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	// Create initial version
	v1 := seedVersion(t, db, draft.ID, "v1")

	// Modify draft HTML
	UpdateDraftHTML(db, draft.ID, "<html><body><h1>Modified</h1></body></html>")

	// Rollback to v1
	w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/rollback", map[string]interface{}{
		"version_id": v1.ID,
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, fmt.Sprintf("%d", draft.ID), fmt.Sprintf("%v", data["draft_id"]))
	assert.Contains(t, data["new_version_label"], "回退到版本")
	assert.NotZero(t, data["new_version_id"])
}

// ---------------------------------------------------------------------------
// Export endpoints
// ---------------------------------------------------------------------------

func TestHandler_CreateExport(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/export", map[string]string{
		"html_content": "<html><body>Resume</body></html>",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["task_id"] != nil && data["task_id"] != "")
	assert.Equal(t, "pending", data["status"])
}

func TestHandler_CreateExport_DraftNotFound(t *testing.T) {
	r, _, _ := setupRouter(t)

	w := doJSON(t, r, "POST", "/drafts/99999/export", map[string]string{
		"html_content": "<html><body>Resume</body></html>",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, CodeDraftNotFound, resp.Code)
}

func TestHandler_GetTask(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	// Create an export task via the export endpoint
	exportW := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/export", map[string]string{
		"html_content": "<html><body>Resume</body></html>",
	})
	exportResp := parseResp(t, exportW)
	taskData := exportResp.Data.(map[string]interface{})
	taskID := taskData["task_id"].(string)

	// Get task info
	w := doJSON(t, r, "GET", "/tasks/"+taskID, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, taskID, data["task_id"])
	assert.Equal(t, fmt.Sprintf("%d", draft.ID), fmt.Sprintf("%v", data["draft_id"]))
}

func TestHandler_GetTask_NotFound(t *testing.T) {
	r, _, _ := setupRouter(t)

	w := doJSON(t, r, "GET", "/tasks/task_nonexistent", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, CodeTaskNotFound, resp.Code)
}

func TestHandler_DownloadFile(t *testing.T) {
	r, h, db := setupRouter(t)
	draft := seedDraft(t, db)

	// Create export task
	w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/export", map[string]string{
		"html_content": "<html><body>Resume</body></html>",
	})
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w)
	taskID := resp.Data.(map[string]interface{})["task_id"].(string)

	// Wait for task to complete
	task := waitForTask(t, h.exportSvc, taskID, 3*time.Second)
	assert.Equal(t, "completed", task.Status)

	// Download file
	downloadW := doJSON(t, r, "GET", "/tasks/"+taskID+"/file", nil)
	assert.Equal(t, http.StatusOK, downloadW.Code)
	assert.Equal(t, "application/pdf", downloadW.Header().Get("Content-Type"))
	assert.Greater(t, downloadW.Body.Len(), 0, "PDF body should not be empty")

	// Should contain PDF disposition header
	assert.Contains(t, downloadW.Header().Get("Content-Disposition"), "attachment")
}
