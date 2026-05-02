package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// ---------------------------------------------------------------------------
// Tool definition tests
// ---------------------------------------------------------------------------

func TestToolExecutor_Tools_FiveDefinitions(t *testing.T) {
	executor := NewAgentToolExecutor(nil, "")
	tools := executor.Tools()
	require.Len(t, tools, 5)

	// Verify each tool has the required fields.
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name, "tool name must not be empty")
		assert.NotEmpty(t, tool.Description, "tool %s description must not be empty", tool.Name)

		params := tool.Parameters
		require.NotNil(t, params, "tool %s parameters must not be nil", tool.Name)
		assert.Equal(t, "object", params["type"], "tool %s parameters type must be 'object'", tool.Name)

		props, ok := params["properties"].(map[string]interface{})
		require.True(t, ok, "tool %s must have properties", tool.Name)
		require.NotEmpty(t, props, "tool %s must have at least one property", tool.Name)

		required, hasRequired := params["required"]
		assert.True(t, hasRequired, "tool %s must have a required field", tool.Name)
		requiredList, ok := required.([]interface{})
		require.True(t, ok, "tool %s required must be an array", tool.Name)
		assert.NotEmpty(t, requiredList, "tool %s required must not be empty", tool.Name)
	}
}

func TestToolExecutor_Tools_NamesAreCorrect(t *testing.T) {
	executor := NewAgentToolExecutor(nil, "")
	tools := executor.Tools()

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	expected := []string{
		"get_project_assets",
		"get_draft",
		"save_draft",
		"create_version",
		"export_pdf",
	}
	assert.Equal(t, expected, names)
}

func TestToolExecutor_Tools_ParameterSchemas(t *testing.T) {
	executor := NewAgentToolExecutor(nil, "")
	tools := executor.Tools()
	toolByName := make(map[string]ToolDef)
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	// get_project_assets: required = ["project_id"]
	{
		tool := toolByName["get_project_assets"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "project_id")
		p := props["project_id"].(map[string]interface{})
		assert.Equal(t, "integer", p["type"])
		req := tool.Parameters["required"].([]interface{})
		assert.Contains(t, req, "project_id")
	}

	// get_draft: required = ["draft_id"]
	{
		tool := toolByName["get_draft"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "draft_id")
		p := props["draft_id"].(map[string]interface{})
		assert.Equal(t, "integer", p["type"])
		req := tool.Parameters["required"].([]interface{})
		assert.Contains(t, req, "draft_id")
	}

	// save_draft: required = ["draft_id", "html_content"]
	{
		tool := toolByName["save_draft"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "draft_id")
		assert.Contains(t, props, "html_content")
		p := props["html_content"].(map[string]interface{})
		assert.Equal(t, "string", p["type"])
		req := tool.Parameters["required"].([]interface{})
		assert.Contains(t, req, "draft_id")
		assert.Contains(t, req, "html_content")
	}

	// create_version: required = ["draft_id", "label"]
	{
		tool := toolByName["create_version"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "draft_id")
		assert.Contains(t, props, "label")
		p := props["label"].(map[string]interface{})
		assert.Equal(t, "string", p["type"])
		req := tool.Parameters["required"].([]interface{})
		assert.Contains(t, req, "draft_id")
		assert.Contains(t, req, "label")
	}

	// export_pdf: required = ["draft_id", "html_content"]
	{
		tool := toolByName["export_pdf"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "draft_id")
		assert.Contains(t, props, "html_content")
		p := props["html_content"].(map[string]interface{})
		assert.Equal(t, "string", p["type"])
		req := tool.Parameters["required"].([]interface{})
		assert.Contains(t, req, "draft_id")
		assert.Contains(t, req, "html_content")
	}
}

// ---------------------------------------------------------------------------
// Unknown tool
// ---------------------------------------------------------------------------

func TestToolExecutor_Execute_UnknownTool_ReturnsError(t *testing.T) {
	executor := NewAgentToolExecutor(nil, "")
	_, err := executor.Execute(context.Background(), "nonexistent_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

// ---------------------------------------------------------------------------
// Missing / invalid parameters
// ---------------------------------------------------------------------------

func TestToolExecutor_Execute_MissingRequiredParams(t *testing.T) {
	executor := NewAgentToolExecutor(nil, "")

	_, err := executor.Execute(context.Background(), "get_project_assets", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project_id")

	_, err = executor.Execute(context.Background(), "save_draft", map[string]interface{}{"draft_id": float64(1)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "html_content")

	_, err = executor.Execute(context.Background(), "save_draft", map[string]interface{}{"html_content": "hi"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "draft_id")

	_, err = executor.Execute(context.Background(), "get_draft", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "draft_id")

	_, err = executor.Execute(context.Background(), "create_version", map[string]interface{}{"draft_id": float64(1)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "label")

	_, err = executor.Execute(context.Background(), "export_pdf", map[string]interface{}{"draft_id": float64(1)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "html_content")
}

func TestToolExecutor_Execute_InvalidParamType(t *testing.T) {
	executor := NewAgentToolExecutor(nil, "")

	_, err := executor.Execute(context.Background(), "get_project_assets", map[string]interface{}{"project_id": "not-a-number"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project_id")

	_, err = executor.Execute(context.Background(), "save_draft", map[string]interface{}{
		"draft_id":     float64(1),
		"html_content": float64(42),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "html_content")
}

// ---------------------------------------------------------------------------
// DB-backed tool tests (require PostgreSQL, skip if unavailable)
// ---------------------------------------------------------------------------

func TestToolExecutor_GetProjectAssets(t *testing.T) {
	db := SetupTestDB(t)
	db.AutoMigrate(&models.Asset{})
	executor := NewAgentToolExecutor(db, "")

	proj := models.Project{Title: "Test Project", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	content := "张三 | 前端工程师\n工作经历\nABC科技 2022-至今"
	asset := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_pdf",
		Content:   &content,
		Label:     strPtr("resume.pdf"),
	}
	require.NoError(t, db.Create(&asset).Error)

	result, err := executor.Execute(context.Background(), "get_project_assets", map[string]interface{}{"project_id": float64(proj.ID)})
	require.NoError(t, err)

	assert.Contains(t, result, "项目资产列表")
	assert.Contains(t, result, "[resume_pdf]")
	assert.Contains(t, result, "resume.pdf")
	assert.Contains(t, result, fmt.Sprintf("ID:%d", asset.ID))
}

func TestToolExecutor_GetProjectAssets_EmptyList(t *testing.T) {
	db := SetupTestDB(t)
	db.AutoMigrate(&models.Asset{})
	executor := NewAgentToolExecutor(db, "")

	proj := models.Project{Title: "Empty Project", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	result, err := executor.Execute(context.Background(), "get_project_assets", map[string]interface{}{"project_id": float64(proj.ID)})
	require.NoError(t, err)
	assert.Equal(t, "该项目暂无资产。", result)
}

func TestToolExecutor_GetProjectAssets_HasMultipleAssets(t *testing.T) {
	db := SetupTestDB(t)
	db.AutoMigrate(&models.Asset{})
	executor := NewAgentToolExecutor(db, "")

	proj := models.Project{Title: "Multi Assets", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	c1 := "content1"
	c2 := "content2"
	require.NoError(t, db.Create(&models.Asset{ProjectID: proj.ID, Type: "pdf", Content: &c1}).Error)
	require.NoError(t, db.Create(&models.Asset{ProjectID: proj.ID, Type: "docx", Content: &c2}).Error)

	result, err := executor.Execute(context.Background(), "get_project_assets", map[string]interface{}{"project_id": float64(proj.ID)})
	require.NoError(t, err)

	assert.Contains(t, result, "pdf")
	assert.Contains(t, result, "docx")
	assert.Contains(t, result, "content1")
	assert.Contains(t, result, "content2")
}

func TestToolExecutor_GetDraft(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, "")

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body>Hello</body></html>"}
	require.NoError(t, db.Create(&draft).Error)

	result, err := executor.Execute(context.Background(), "get_draft", map[string]interface{}{"draft_id": float64(draft.ID)})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(draft.ID), data["draft_id"])
	assert.Equal(t, "<html><body>Hello</body></html>", data["html_content"])
	assert.NotEmpty(t, data["updated_at"])
}

func TestToolExecutor_GetDraft_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, "")

	_, err := executor.Execute(context.Background(), "get_draft", map[string]interface{}{"draft_id": float64(99999)})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestToolExecutor_SaveDraft(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, "")

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html>original</html>"}
	require.NoError(t, db.Create(&draft).Error)

	result, err := executor.Execute(context.Background(), "save_draft", map[string]interface{}{
		"draft_id":     float64(draft.ID),
		"html_content": "<html>updated</html>",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(draft.ID), data["draft_id"])
	assert.Equal(t, "saved", data["status"])

	// Verify the database was actually updated.
	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Equal(t, "<html>updated</html>", updated.HTMLContent)
}

func TestToolExecutor_SaveDraft_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, "")

	_, err := executor.Execute(context.Background(), "save_draft", map[string]interface{}{
		"draft_id":     float64(99999),
		"html_content": "<html>nope</html>",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestToolExecutor_SaveDraft_EmptyContent(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, "")

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html>original</html>"}
	require.NoError(t, db.Create(&draft).Error)

	result, err := executor.Execute(context.Background(), "save_draft", map[string]interface{}{
		"draft_id":     float64(draft.ID),
		"html_content": "",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(draft.ID), data["draft_id"])
	assert.Equal(t, "saved", data["status"])

	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Equal(t, "", updated.HTMLContent)
}

// ---------------------------------------------------------------------------
// HTTP tool tests (use mock HTTP server)
// ---------------------------------------------------------------------------

func TestToolExecutor_CreateVersion(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/drafts/5/versions", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "v1.0", body["label"])

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0,"data":{"id":10,"label":"v1.0","created_at":"2026-05-02T12:00:00Z"}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	executor := NewAgentToolExecutor(nil, srv.URL)
	result, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
		"draft_id": float64(5),
		"label":    "v1.0",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(10), data["id"])
	assert.Equal(t, "v1.0", data["label"])
}

func TestToolExecutor_ExportPDF(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/drafts/3/export", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "<html>resume</html>", body["html_content"])

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0,"data":{"task_id":"task_abc","status":"pending"}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	executor := NewAgentToolExecutor(nil, srv.URL)
	result, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
		"draft_id":     float64(3),
		"html_content": "<html>resume</html>",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "task_abc", data["task_id"])
	assert.Equal(t, "pending", data["status"])
}

func TestToolExecutor_ExportPDF_EmptyHTML(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/drafts/3/export", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "", body["html_content"])

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0,"data":{"task_id":"task_xyz","status":"pending"}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	executor := NewAgentToolExecutor(nil, srv.URL)
	result, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
		"draft_id":     float64(3),
		"html_content": "",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "task_xyz")
}

// ---------------------------------------------------------------------------
// HTTP error handling
// ---------------------------------------------------------------------------

func TestToolExecutor_HTTP_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/drafts/1/export", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"code":5001,"message":"PDF export failed"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	executor := NewAgentToolExecutor(nil, srv.URL)
	_, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
		"draft_id":     float64(1),
		"html_content": "<html></html>",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestToolExecutor_HTTP_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/drafts/999/versions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":5002,"data":null,"message":"draft not found"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	executor := NewAgentToolExecutor(nil, srv.URL)
	_, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
		"draft_id": float64(999),
		"label":    "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestToolExecutor_GetIntParam_Float64(t *testing.T) {
	n, err := getIntParam(map[string]interface{}{"x": float64(42)}, "x")
	require.NoError(t, err)
	assert.Equal(t, 42, n)
}

func TestToolExecutor_GetIntParam_Int(t *testing.T) {
	n, err := getIntParam(map[string]interface{}{"x": 99}, "x")
	require.NoError(t, err)
	assert.Equal(t, 99, n)
}

func TestToolExecutor_GetIntParam_String(t *testing.T) {
	n, err := getIntParam(map[string]interface{}{"x": "123"}, "x")
	require.NoError(t, err)
	assert.Equal(t, 123, n)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string {
	return &s
}

