package workbench

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func TestGetDraft_SucceedsAndReturnsHtmlContent(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>hello</body></html>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	draftIDStr := strconv.FormatUint(uint64(draft.ID), 10)
	c.Params = gin.Params{{Key: "draft_id", Value: draftIDStr}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/drafts/"+draftIDStr, nil)

	h.GetDraft(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["id"].(float64) != float64(draft.ID) {
		t.Fatalf("expected id %d, got %v", draft.ID, data["id"])
	}
	if data["project_id"].(float64) != float64(project.ID) {
		t.Fatalf("expected project_id %d, got %v", project.ID, data["project_id"])
	}
	if data["html_content"].(string) != "<html><body>hello</body></html>" {
		t.Fatalf("expected html_content '<html><body>hello</body></html>', got %v", data["html_content"])
	}
	if data["updated_at"] == nil {
		t.Fatalf("expected updated_at to be set")
	}
}

func TestGetDraft_Returns400WhenDraftIDInvalid(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "abc"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/drafts/abc", nil)

	h.GetDraft(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 40000 {
		t.Fatalf("expected code 40000, got %v", resp["code"])
	}
}

func TestGetDraft_Returns404WhenDraftNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "999"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/drafts/999", nil)

	h.GetDraft(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 4001 {
		t.Fatalf("expected code 4001, got %v", resp["code"])
	}
	if resp["message"].(string) != "draft not found" {
		t.Fatalf("expected message 'draft not found', got %v", resp["message"])
	}
}

func TestUpdateDraft_SucceedsAndReturnsIdAndUpdatedAt(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>old</body></html>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	draftIDStr := strconv.FormatUint(uint64(draft.ID), 10)
	c.Params = gin.Params{{Key: "draft_id", Value: draftIDStr}}
	body := `{"html_content": "<html><body>new content</body></html>"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/drafts/"+draftIDStr, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["id"].(float64) != float64(draft.ID) {
		t.Fatalf("expected id %d, got %v", draft.ID, data["id"])
	}
	if data["updated_at"] == nil {
		t.Fatalf("expected updated_at to be set")
	}

	// Verify the draft was actually updated
	var updatedDraft models.Draft
	if err := tx.First(&updatedDraft, draft.ID).Error; err != nil {
		t.Fatalf("failed to fetch updated draft: %v", err)
	}
	if updatedDraft.HTMLContent != "<html><body>new content</body></html>" {
		t.Fatalf("expected HTML content to be updated, got %v", updatedDraft.HTMLContent)
	}
}

func TestUpdateDraft_CreateVersion_ReturnsVersionIDAndCreatesVersion(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>old</body></html>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	draftIDStr := strconv.FormatUint(uint64(draft.ID), 10)
	c.Params = gin.Params{{Key: "draft_id", Value: draftIDStr}}
	body := `{"html_content": "<html><body>new content</body></html>", "create_version": true, "version_label": "manual save"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/drafts/"+draftIDStr, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["version_id"] == nil {
		t.Fatalf("expected version_id to be set")
	}
	versionID := uint(data["version_id"].(float64))

	var version models.Version
	if err := tx.First(&version, versionID).Error; err != nil {
		t.Fatalf("failed to fetch created version: %v", err)
	}
	if version.DraftID != draft.ID {
		t.Fatalf("expected version draft_id %d, got %d", draft.ID, version.DraftID)
	}
	if version.HTMLSnapshot != "<html><body>new content</body></html>" {
		t.Fatalf("expected version html_snapshot to match updated html")
	}
	if version.Label == nil || *version.Label != "manual save" {
		t.Fatalf("expected version label 'manual save', got %v", version.Label)
	}
}

func TestUpdateDraft_Returns400WhenHtmlContentIsEmpty(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>old</body></html>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	draftIDStr := strconv.FormatUint(uint64(draft.ID), 10)
	c.Params = gin.Params{{Key: "draft_id", Value: draftIDStr}}
	body := `{"html_content": ""}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/drafts/"+draftIDStr, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 4002 {
		t.Fatalf("expected code 4002, got %v", resp["code"])
	}
	if resp["message"].(string) != "html content empty" {
		t.Fatalf("expected message 'html content empty', got %v", resp["message"])
	}
}

func TestUpdateDraft_Returns404WhenDraftNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "999"}}
	body := `{"html_content": "<html><body>new</body></html>"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/drafts/999", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 4001 {
		t.Fatalf("expected code 4001, got %v", resp["code"])
	}
}

func TestUpdateDraft_Returns400WhenRequestBodyInvalid(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "1"}}
	body := `invalid json`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/drafts/1", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 40000 {
		t.Fatalf("expected code 40000, got %v", resp["code"])
	}
}

func TestUpdateDraft_Returns400WhenDraftIDInvalid(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "abc"}}
	body := `{"html_content": "<html><body>new</body></html>"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/drafts/abc", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 40000 {
		t.Fatalf("expected code 40000, got %v", resp["code"])
	}
}

// TestRoutePaths_CorrectlyMounted verifies that the draft routes are mounted
// on the correct resource path (/api/v1/drafts/:draft_id) and not on the
// legacy module-prefixed path (/api/v1/workbench/drafts/:draft_id).
func TestRoutePaths_CorrectlyMounted(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	// Create a test draft so the endpoint doesn't return 404
	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>test</body></html>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterRoutes(v1, tx)

	// Correct resource path should not return 404
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drafts/"+strconv.FormatUint(uint64(draft.ID), 10), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "resource path should be mounted")

	// Legacy module-prefixed path should return 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/workbench/drafts/"+strconv.FormatUint(uint64(draft.ID), 10), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "legacy path should not be mounted")
}

func TestCreateDraft_SucceedsAndReturnsDraft(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"project_id": ` + strconv.FormatUint(uint64(project.ID), 10) + `}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["id"] == nil {
		t.Fatalf("expected id to be set")
	}
	if data["project_id"].(float64) != float64(project.ID) {
		t.Fatalf("expected project_id %d, got %v", project.ID, data["project_id"])
	}
	if data["html_content"].(string) != "" {
		t.Fatalf("expected empty html_content, got %v", data["html_content"])
	}
	if data["updated_at"] == nil {
		t.Fatalf("expected updated_at to be set")
	}

	// Verify the project's current_draft_id was updated
	var updatedProject models.Project
	if err := tx.First(&updatedProject, project.ID).Error; err != nil {
		t.Fatalf("failed to fetch updated project: %v", err)
	}
	if updatedProject.CurrentDraftID == nil {
		t.Fatalf("expected project.current_draft_id to be set")
	}
	draftID := uint(data["id"].(float64))
	if *updatedProject.CurrentDraftID != draftID {
		t.Fatalf("expected project.current_draft_id %d, got %d", draftID, *updatedProject.CurrentDraftID)
	}
}

func TestCreateDraft_Returns404WhenProjectNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"project_id": 99999}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 4003 {
		t.Fatalf("expected code 4003, got %v", resp["code"])
	}
	if resp["message"].(string) != "project not found" {
		t.Fatalf("expected message 'project not found', got %v", resp["message"])
	}
}

func TestCreateDraft_Returns409WhenProjectHasDraft(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>existing</body></html>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Model(&project).Update("current_draft_id", draft.ID).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"project_id": ` + strconv.FormatUint(uint64(project.ID), 10) + `}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 4004 {
		t.Fatalf("expected code 4004, got %v", resp["code"])
	}
	if resp["message"].(string) != "project already has a current draft" {
		t.Fatalf("expected message 'project already has a current draft', got %v", resp["message"])
	}
}

func TestCreateDraft_Returns400WhenRequestBodyInvalid(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `invalid json`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["code"].(float64) != 40000 {
		t.Fatalf("expected code 40000, got %v", resp["code"])
	}
}

func TestCreateDraft_RouteMountedCorrectly(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterRoutes(v1, tx)

	// Create a project so we can make a valid request
	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	body := `{"project_id": ` + strconv.FormatUint(uint64(project.ID), 10) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Fatalf("route not mounted: got 404")
	}
}
