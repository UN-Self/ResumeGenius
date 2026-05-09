package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// ---------------------------------------------------------------------------
// Tool definition tests
// ---------------------------------------------------------------------------

func TestToolExecutor_Tools_Definitions(t *testing.T) {
	executor := NewAgentToolExecutor(nil, nil)
	tools := executor.Tools(context.Background())
	require.Len(t, tools, 3, "without skillLoader should have 3 base tools")

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name, "tool name must not be empty")
		assert.NotEmpty(t, tool.Description, "tool %s description must not be empty", tool.Name)

		params := tool.Parameters
		require.NotNil(t, params, "tool %s parameters must not be nil", tool.Name)
		assert.Equal(t, "object", params["type"], "tool %s parameters type must be 'object'", tool.Name)

		props, ok := params["properties"].(map[string]interface{})
		require.True(t, ok, "tool %s must have properties", tool.Name)
		require.NotEmpty(t, props, "tool %s must have at least one property", tool.Name)
	}

	// With skillLoader
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executorWithSkills := NewAgentToolExecutor(nil, loader)
	toolsWithSkills := executorWithSkills.Tools(context.Background())
	require.Len(t, toolsWithSkills, 5, "with skillLoader should have 3 base + 2 skill tools")
}

func TestToolExecutor_Tools_NamesAreCorrect(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)
	tools := executor.Tools(context.Background())

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "get_draft")
	assert.Contains(t, names, "apply_edits")
	assert.Contains(t, names, "search_assets")
	assert.Contains(t, names, "resume-design")
	assert.Contains(t, names, "resume-interview")
	assert.NotContains(t, names, "load_skill", "load_skill should be removed")
	assert.NotContains(t, names, "get_skill_reference", "get_skill_reference should only appear after skill loaded")
}

func TestToolExecutor_Tools_ParameterSchemas(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)
	tools := executor.Tools(context.Background())
	toolByName := make(map[string]ToolDef)
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	// get_draft: selector is optional, required = []
	{
		tool := toolByName["get_draft"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "selector")
		p := props["selector"].(map[string]interface{})
		assert.Equal(t, "string", p["type"])
		req := tool.Parameters["required"].([]string)
		assert.Empty(t, req)
	}

	// apply_edits: required = ["ops"]
	{
		tool := toolByName["apply_edits"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "ops")
		p := props["ops"].(map[string]interface{})
		assert.Equal(t, "array", p["type"])
		items := p["items"].(map[string]interface{})
		itemProps := items["properties"].(map[string]interface{})
		assert.Contains(t, itemProps, "old_string")
		assert.Contains(t, itemProps, "new_string")
		assert.Contains(t, itemProps, "description")
		req := tool.Parameters["required"].([]string)
		assert.Equal(t, []string{"ops"}, req)
	}

	// search_assets: all optional, required = []
	{
		tool := toolByName["search_assets"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "query")
		assert.Contains(t, props, "type")
		assert.Contains(t, props, "limit")
		req := tool.Parameters["required"].([]string)
		assert.Empty(t, req)
	}

	// resume-design: skill tool, no parameters
	{
		tool := toolByName["resume-design"]
		assert.NotEmpty(t, tool.Description)
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Empty(t, props)
	}
}

// ---------------------------------------------------------------------------
// Skill tools in Tools()
// ---------------------------------------------------------------------------

func TestTools_ContainsSkillTools(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	tools := executor.Tools(context.Background())
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "resume-design")
	assert.Contains(t, names, "resume-interview")
}

func TestTools_SkillToolHasNoParameters(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	tools := executor.Tools(context.Background())
	toolByName := make(map[string]ToolDef)
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	designTool := toolByName["resume-design"]
	assert.NotEmpty(t, designTool.Description)
	props := designTool.Parameters["properties"].(map[string]interface{})
	assert.Empty(t, props, "skill tool should have no parameters")
}

// ---------------------------------------------------------------------------
// Skill tool execution
// ---------------------------------------------------------------------------

func TestExecute_SkillAsTool(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 200)
	result, err := executor.Execute(ctx, "resume-design", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "resume-design", data["name"])
	assert.NotEmpty(t, data["description"])
	assert.NotEmpty(t, data["usage"])
}

func TestExecute_SkillAsTool_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	_, err = executor.Execute(context.Background(), "nonexistent-skill", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestExecute_SkillAsTool_MarksLoaded(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	sessionID := uint(201)
	ctx := WithSessionID(context.Background(), sessionID)

	// Before calling skill tool, get_skill_reference should not be in tools
	toolsBefore := executor.Tools(ctx)
	for _, tool := range toolsBefore {
		assert.NotEqual(t, "get_skill_reference", tool.Name)
	}

	// Call skill tool
	_, err = executor.Execute(ctx, "resume-design", nil)
	require.NoError(t, err)

	// After calling skill tool, get_skill_reference should appear
	toolsAfter := executor.Tools(ctx)
	found := false
	for _, tool := range toolsAfter {
		if tool.Name == "get_skill_reference" {
			found = true
			break
		}
	}
	assert.True(t, found, "get_skill_reference should appear after loading a skill")
}

// ---------------------------------------------------------------------------
// get_draft
// ---------------------------------------------------------------------------

func TestGetDraft_Full(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Hello</h1><p>World</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)
	assert.Equal(t, html, result)
}

func TestGetDraft_Selector(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Hello</h1><p>World</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"selector": "h1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello", result)
}

func TestGetDraft_ContextMissing(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	_, err := executor.Execute(context.Background(), "get_draft", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "draft_id not found in context")
}

// ---------------------------------------------------------------------------
// apply_edits
// ---------------------------------------------------------------------------

func TestApplyEdits_Success(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Old Title</h1><p>Old paragraph</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string":  "Old Title",
				"new_string":  "New Title",
				"description": "update heading",
			},
			map[string]interface{}{
				"old_string":  "Old paragraph",
				"new_string":  "New paragraph",
				"description": "update text",
			},
		},
	})
	require.NoError(t, err)

	// Verify result JSON
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(2), data["applied"])
	assert.NotNil(t, data["new_sequence"])

	// Verify DB state
	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Contains(t, updated.HTMLContent, "New Title")
	assert.Contains(t, updated.HTMLContent, "New paragraph")
	assert.NotContains(t, updated.HTMLContent, "Old Title")
	assert.NotContains(t, updated.HTMLContent, "Old paragraph")
	assert.Equal(t, 2, updated.CurrentEditSequence)

	// Verify DraftEdit records (1 base snapshot + 2 edits)
	var edits []models.DraftEdit
	require.NoError(t, db.Where("draft_id = ?", draft.ID).Order("sequence ASC").Find(&edits).Error)
	require.Len(t, edits, 3)

	// Check base snapshot (sequence 0)
	assert.Equal(t, 0, edits[0].Sequence)
	assert.Equal(t, "base_snapshot", edits[0].OpType)
	assert.Contains(t, edits[0].HtmlSnapshot, "Old Title")

	// Check edit 1 (sequence 1)
	assert.Equal(t, 1, edits[1].Sequence)
	assert.Equal(t, "replace", edits[1].OpType)
	assert.Equal(t, "Old Title", edits[1].OldString)
	assert.Equal(t, "New Title", edits[1].NewString)
	assert.Equal(t, "update heading", edits[1].Description)
	assert.Contains(t, edits[1].HtmlSnapshot, "New Title")
	assert.Contains(t, edits[1].HtmlSnapshot, "Old paragraph")

	// Check edit 2 (sequence 2)
	assert.Equal(t, 2, edits[2].Sequence)
	assert.Equal(t, "replace", edits[2].OpType)
	assert.Equal(t, "Old paragraph", edits[2].OldString)
	assert.Equal(t, "New paragraph", edits[2].NewString)
	assert.Equal(t, "update text", edits[2].Description)
	assert.Contains(t, edits[2].HtmlSnapshot, "New Title")
	assert.Contains(t, edits[2].HtmlSnapshot, "New paragraph")
}

func TestApplyEdits_OldStringNotFound(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Title</h1></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "NonExistent",
				"new_string": "Replacement",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "old_string not found")

	// Verify DB state unchanged
	var unchanged models.Draft
	require.NoError(t, db.First(&unchanged, draft.ID).Error)
	assert.Equal(t, html, unchanged.HTMLContent)
	assert.Equal(t, 0, unchanged.CurrentEditSequence)

	// No DraftEdit records created
	var edits []models.DraftEdit
	require.NoError(t, db.Where("draft_id = ?", draft.ID).Find(&edits).Error)
	assert.Empty(t, edits)
}

func TestApplyEdits_RejectsOverdesignedResumeStyle(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><p>Original</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Original",
				"new_string": `<style>.resume{background:linear-gradient(135deg,#667eea,#764ba2);}</style><p>Modified</p>`,
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "violates resume design constraints")

	var unchanged models.Draft
	require.NoError(t, db.First(&unchanged, draft.ID).Error)
	assert.Equal(t, html, unchanged.HTMLContent)
}

func TestApplyEdits_BaseSnapshot(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><p>Original</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Original",
				"new_string": "Modified",
			},
		},
	})
	require.NoError(t, err)

	// Base snapshot should exist at sequence 0
	var baseEdit models.DraftEdit
	require.NoError(t, db.Where("draft_id = ? AND sequence = 0", draft.ID).First(&baseEdit).Error)
	assert.Equal(t, "base_snapshot", baseEdit.OpType)
	assert.Equal(t, html, baseEdit.HtmlSnapshot)
}

// ---------------------------------------------------------------------------
// search_assets
// ---------------------------------------------------------------------------

func TestSearchAssets(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	c1 := "Resume content for John"
	require.NoError(t, db.Create(&models.Asset{
		ProjectID: proj.ID,
		Type:      "resume",
		Content:   &c1,
		Label:     strPtr("john.pdf"),
	}).Error)

	c2 := "Git summary for frontend"
	require.NoError(t, db.Create(&models.Asset{
		ProjectID: proj.ID,
		Type:      "git_summary",
		Content:   &c2,
		Label:     strPtr("repo"),
	}).Error)

	c3 := "Some note content"
	require.NoError(t, db.Create(&models.Asset{
		ProjectID: proj.ID,
		Type:      "note",
		Content:   &c3,
		Label:     strPtr("note1"),
	}).Error)

	ctx := WithProjectID(context.Background(), proj.ID)
	result, err := executor.Execute(ctx, "search_assets", map[string]interface{}{
		"query": "John",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	results, ok := data["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 1)
}

func TestSearchAssets_Empty(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	ctx := WithProjectID(context.Background(), proj.ID)
	result, err := executor.Execute(ctx, "search_assets", map[string]interface{}{
		"query": "nonexistent",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	results, ok := data["results"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, results)
}

// ---------------------------------------------------------------------------
// get_skill_reference
// ---------------------------------------------------------------------------

func TestGetReference_Valid(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 2)
	// First load the skill via skill tool (not load_skill)
	_, err = executor.Execute(ctx, "resume-interview", nil)
	require.NoError(t, err)

	// Then get the reference
	result, err := executor.Execute(ctx, "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "test-engineer", data["name"])
	assert.NotEmpty(t, data["content"])
}

func TestGetReference_SkillNotLoaded(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 3)
	// Try to get reference without loading skill first
	result, err := executor.Execute(ctx, "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "not loaded")
	assert.Contains(t, result, "call 'resume-interview' tool first")
}

func TestGetReference_ReferenceNotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 4)
	// Load the skill via skill tool
	_, err = executor.Execute(ctx, "resume-interview", nil)
	require.NoError(t, err)

	// Try to get a nonexistent reference
	result, err := executor.Execute(ctx, "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "nonexistent-ref",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
	assert.Contains(t, result, "test-engineer") // should list available
}

func TestGetReference_MissingParams(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	// Missing skill_name
	result, err := executor.Execute(context.Background(), "get_skill_reference", map[string]interface{}{
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "skill_name is required")

	// Missing reference_name
	result, err = executor.Execute(context.Background(), "get_skill_reference", map[string]interface{}{
		"skill_name": "resume-interview",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "reference_name is required")
}

func TestGetReference_NilLoader(t *testing.T) {
	executor := NewAgentToolExecutor(nil, nil)

	result, err := executor.Execute(context.Background(), "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "技能库未加载")
}

// ---------------------------------------------------------------------------
// Complete flow integration test
// ---------------------------------------------------------------------------

func TestToolExecutor_CompleteFlow(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 100)

	// Step 1: Call skill tool (not load_skill)
	loadResult, err := executor.Execute(ctx, "resume-interview", nil)
	require.NoError(t, err)

	var loadDesc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(loadResult), &loadDesc))
	assert.Equal(t, "resume-interview", loadDesc["name"])

	// Step 2: Get reference
	refResult, err := executor.Execute(ctx, "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)

	var refData map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(refResult), &refData))
	assert.Equal(t, "test-engineer", refData["name"])
	assert.NotEmpty(t, refData["content"])
}

// ---------------------------------------------------------------------------
// Unknown tool
// ---------------------------------------------------------------------------

func TestExecute_UnknownTool(t *testing.T) {
	executor := NewAgentToolExecutor(nil, nil)
	_, err := executor.Execute(context.Background(), "nonexistent_tool", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

// ---------------------------------------------------------------------------
// getIntParam helper
// ---------------------------------------------------------------------------

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

// Ensure DraftEdit model is properly used in tests
var _ time.Time
