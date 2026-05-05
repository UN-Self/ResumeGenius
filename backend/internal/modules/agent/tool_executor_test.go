package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// mockVersionSvc mocks the VersionService for testing
type mockVersionSvc struct {
	mock.Mock
}

func (m *mockVersionSvc) Create(draftID uint, label string) (*models.Version, error) {
	args := m.Called(draftID, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Version), args.Error(1)
}

// mockExportSvc mocks the ExportService for testing
type mockExportSvc struct {
	mock.Mock
}

func (m *mockExportSvc) CreateTask(draftID uint, htmlContent string) (string, error) {
	args := m.Called(draftID, htmlContent)
	return args.String(0), args.Error(1)
}

// Helper to create an executor with mocks for testing
func newMockExecutor() (*AgentToolExecutor, *mockVersionSvc, *mockExportSvc) {
	vSvc := new(mockVersionSvc)
	eSvc := new(mockExportSvc)
	executor := NewAgentToolExecutor(nil, vSvc, eSvc)
	return executor, vSvc, eSvc
}

// ---------------------------------------------------------------------------
// Tool definition tests
// ---------------------------------------------------------------------------

func TestToolExecutor_Tools_FiveDefinitions(t *testing.T) {
	executor, _, _ := newMockExecutor()
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
	executor, _, _ := newMockExecutor()
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
	executor, _, _ := newMockExecutor()
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
	executor, _, _ := newMockExecutor()
	_, err := executor.Execute(context.Background(), "nonexistent_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

// ---------------------------------------------------------------------------
// Missing / invalid parameters
// ---------------------------------------------------------------------------

func TestToolExecutor_Execute_MissingRequiredParams(t *testing.T) {
	executor, _, _ := newMockExecutor()

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

	_, err = executor.Execute(context.Background(), "export_pdf", map[string]interface{}{"draft_id": float64(1)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "html_content")
}

func TestToolExecutor_Execute_InvalidParamType(t *testing.T) {
	executor, _, _ := newMockExecutor()

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
	executor, _, _ := newMockExecutor()
	executor.db = db

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
	executor, _, _ := newMockExecutor()
	executor.db = db

	proj := models.Project{Title: "Empty Project", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	result, err := executor.Execute(context.Background(), "get_project_assets", map[string]interface{}{"project_id": float64(proj.ID)})
	require.NoError(t, err)
	assert.Equal(t, "该项目暂无资产。", result)
}

func TestToolExecutor_GetProjectAssets_HasMultipleAssets(t *testing.T) {
	db := SetupTestDB(t)
	db.AutoMigrate(&models.Asset{})
	executor, _, _ := newMockExecutor()
	executor.db = db

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
	executor, _, _ := newMockExecutor()
	executor.db = db

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
	executor, _, _ := newMockExecutor()
	executor.db = db

	_, err := executor.Execute(context.Background(), "get_draft", map[string]interface{}{"draft_id": float64(99999)})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestToolExecutor_SaveDraft(t *testing.T) {
	db := SetupTestDB(t)
	executor, _, _ := newMockExecutor()
	executor.db = db

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
	executor, _, _ := newMockExecutor()
	executor.db = db

	_, err := executor.Execute(context.Background(), "save_draft", map[string]interface{}{
		"draft_id":     float64(99999),
		"html_content": "<html>nope</html>",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestToolExecutor_SaveDraft_EmptyContent(t *testing.T) {
	db := SetupTestDB(t)
	executor, _, _ := newMockExecutor()
	executor.db = db

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
// Service-based tool tests (use mocks)
// ---------------------------------------------------------------------------

func TestToolExecutor_CreateVersion(t *testing.T) {
	executor, vSvc, _ := newMockExecutor()

	now := time.Now()
	label := "v1.0"
	vSvc.On("Create", uint(5), "v1.0").Return(&models.Version{
		ID:        10,
		DraftID:   5,
		Label:     &label,
		CreatedAt: now,
	}, nil)

	result, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
		"draft_id": float64(5),
		"label":    "v1.0",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(10), data["version_id"])
	assert.Equal(t, "v1.0", data["label"])
	vSvc.AssertCalled(t, "Create", uint(5), "v1.0")
}

func TestToolExecutor_ExportPDF(t *testing.T) {
	executor, _, eSvc := newMockExecutor()

	eSvc.On("CreateTask", uint(3), "<html>resume</html>").Return("task_abc", nil)

	result, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
		"draft_id":     float64(3),
		"html_content": "<html>resume</html>",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "task_abc", data["task_id"])
	eSvc.AssertCalled(t, "CreateTask", uint(3), "<html>resume</html>")
}

func TestToolExecutor_ExportPDF_EmptyHTML(t *testing.T) {
	executor, _, eSvc := newMockExecutor()

	eSvc.On("CreateTask", uint(3), "").Return("task_xyz", nil)

	result, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
		"draft_id":     float64(3),
		"html_content": "",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "task_xyz")
}

// ---------------------------------------------------------------------------
// Service error handling
// ---------------------------------------------------------------------------

func TestToolExecutor_CreateVersion_ServiceError(t *testing.T) {
	executor, vSvc, _ := newMockExecutor()

	vSvc.On("Create", uint(1), "test").Return(nil, errors.New("draft not found"))

	_, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
		"draft_id": float64(1),
		"label":    "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "draft not found")
}

func TestToolExecutor_ExportPDF_ServiceError(t *testing.T) {
	executor, _, eSvc := newMockExecutor()

	eSvc.On("CreateTask", uint(1), "<html></html>").Return("", errors.New("export failed"))

	_, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
		"draft_id":     float64(1),
		"html_content": "<html></html>",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export failed")
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

