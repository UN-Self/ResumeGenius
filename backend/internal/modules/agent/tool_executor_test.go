package agent

import (
	"context"
	"encoding/json"
	"strings"
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
	executor := NewAgentToolExecutor(nil, nil, nil)
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
	executorWithSkills := NewAgentToolExecutor(nil, loader, nil)
	toolsWithSkills := executorWithSkills.Tools(context.Background())
	require.Len(t, toolsWithSkills, 4, "with skillLoader should have 3 base + 1 load_skill")
}

func TestToolExecutor_Tools_NamesAreCorrect(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)
	tools := executor.Tools(context.Background())

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "get_draft")
	assert.Contains(t, names, "apply_edits")
	assert.Contains(t, names, "search_assets")
	assert.Contains(t, names, "load_skill")
	assert.NotContains(t, names, "resume-design")
	assert.NotContains(t, names, "resume-interview")
	assert.NotContains(t, names, "get_skill_reference")
}

func TestToolExecutor_Tools_ParameterSchemas(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)
	tools := executor.Tools(context.Background())
	toolByName := make(map[string]ToolDef)
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	// get_draft: mode, selector, query are optional, required = []
	{
		tool := toolByName["get_draft"]
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "selector")
		assert.Contains(t, props, "mode")
		assert.Contains(t, props, "query")
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

	// load_skill: required = ["skill_name"]
	{
		tool := toolByName["load_skill"]
		assert.NotEmpty(t, tool.Description)
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Contains(t, props, "skill_name")
		req := tool.Parameters["required"].([]string)
		assert.Equal(t, []string{"skill_name"}, req)
	}
}

// ---------------------------------------------------------------------------
// get_draft
// ---------------------------------------------------------------------------

func TestGetDraft_Full(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Hello</h1><p>World</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, html, data["html"])
	assert.Contains(t, data, "page_count")
}

func TestGetDraft_Full_ReturnsPageCount(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body><p>hello</p></body></html>", PageCount: 2}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(2), data["page_count"], "should return page_count from DB")
	assert.Contains(t, data, "html", "should still contain html")
}

func TestGetDraft_Full_PageCountZero(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body><p>hello</p></body></html>"}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(0), data["page_count"], "default page_count should be 0")
}

func TestGetDraft_Selector(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

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
	executor := NewAgentToolExecutor(db, nil, nil)

	_, err := executor.Execute(context.Background(), "get_draft", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "draft_id not found in context")
}

// ---------------------------------------------------------------------------
// apply_edits
// ---------------------------------------------------------------------------

func TestApplyEdits_Success(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

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
	executor := NewAgentToolExecutor(db, nil, nil)

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
	assert.Contains(t, err.Error(), "匹配失败")
	assert.Contains(t, err.Error(), "NonExistent")
	assert.Contains(t, err.Error(), "未找到匹配内容")

	// Verify DB state unchanged
	var unchanged models.Draft
	require.NoError(t, db.First(&unchanged, draft.ID).Error)
	assert.Equal(t, html, unchanged.HTMLContent)
	assert.Equal(t, 0, unchanged.CurrentEditSequence)

	// No successful edits, but base snapshot may have been created
	var edits []models.DraftEdit
	require.NoError(t, db.Where("draft_id = ?", draft.ID).Find(&edits).Error)
	// Only the base snapshot should exist (if created), no replace edits
	for _, edit := range edits {
		assert.Equal(t, "base_snapshot", edit.OpType, "should only have base_snapshot, not replace")
	}
}


func TestApplyEdits_BaseSnapshot(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

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
	executor := NewAgentToolExecutor(db, nil, nil)

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
	executor := NewAgentToolExecutor(db, nil, nil)

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
// Complete flow integration test
// ---------------------------------------------------------------------------

func TestToolExecutor_CompleteFlow(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)

	ctx := WithSessionID(context.Background(), 100)

	// Step 1: Load skill via load_skill
	loadResult, err := executor.Execute(ctx, "load_skill", map[string]interface{}{
		"skill_name": "resume-interview",
	})
	require.NoError(t, err)

	var loadDesc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(loadResult), &loadDesc))
	assert.Equal(t, "resume-interview", loadDesc["name"])
	assert.NotEmpty(t, loadDesc["description"])
	assert.NotEmpty(t, loadDesc["usage"])

	// Step 2: Verify references are included
	refs, ok := loadDesc["references"].([]interface{})
	require.True(t, ok, "should have references array")
	assert.NotEmpty(t, refs, "references should not be empty")

	// Verify test-engineer reference is in the references
	found := false
	for _, r := range refs {
		ref := r.(map[string]interface{})
		if ref["name"] == "test-engineer" {
			found = true
			assert.NotEmpty(t, ref["content"])
			break
		}
	}
	assert.True(t, found, "test-engineer should be in references")
}

// ---------------------------------------------------------------------------
// Unknown tool
// ---------------------------------------------------------------------------

func TestExecute_UnknownTool(t *testing.T) {
	executor := NewAgentToolExecutor(nil, nil, nil)
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
// Structured error feedback tests
// ---------------------------------------------------------------------------

func TestApplyEdits_StructuredErrorFeedback(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{
		ProjectID:   1,
		HTMLContent: `<html><body><div class="experience"><h3>Google</h3><p>Senior Engineer</p></div></body></html>`,
	}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Sr. Engineer at Google",
				"new_string": "Staff Engineer at Google",
			},
		},
	}

	_, err := executor.Execute(ctx, "apply_edits", params)
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "未找到匹配")
	assert.Contains(t, errMsg, "搜索内容")
	assert.Contains(t, errMsg, "建议")
}

func TestApplyEdits_ErrorIncludesNearbyHTML(t *testing.T) {
	db := SetupTestDB(t)

	longHTML := `<html><body>` + strings.Repeat("<p>filler content</p>", 50) +
		`<div class="target"><span>Unique Target Text Here</span></div>` +
		strings.Repeat("<p>more filler</p>", 50) + `</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: longHTML}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Unique Target Txt Here",
				"new_string": "New Text",
			},
		},
	}

	_, err := executor.Execute(ctx, "apply_edits", params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "附近")
}

// ---------------------------------------------------------------------------
// get_draft multi-mode tests
// ---------------------------------------------------------------------------

func TestGetDraft_ModeStructure(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html>
	<head><style>body{font-size:14px}.header{color:red}</style></head>
	<body>
		<div class="header"><h1>John Doe</h1><p>Engineer</p></div>
		<div class="experience"><h2>Experience</h2><div class="item"><h3>Google</h3></div></div>
		<div class="education"><h2>Education</h2><div class="item"><h3>MIT</h3></div></div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode": "structure",
	})
	require.NoError(t, err)

	assert.Contains(t, result, "<html>")
	assert.Contains(t, result, "header")
	assert.Contains(t, result, "experience")
	assert.NotContains(t, result, "John Doe")
	assert.NotContains(t, result, "Google")
}

func TestGetDraft_ModeSearch(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<div class="experience">
			<h3>Google</h3>
			<p>Senior Engineer 2020-2024, worked on search infrastructure and ML pipelines</p>
		</div>
		<div class="education">
			<h3>MIT</h3>
			<p>Computer Science, GPA 3.9</p>
		</div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":  "search",
		"query": "Engineer",
	})
	require.NoError(t, err)

	assert.Contains(t, result, "Senior Engineer")
	assert.Contains(t, result, "Google")
}

func TestGetDraft_ModeSearch_MultipleResults(t *testing.T) {
	db := SetupTestDB(t)

	html := "<html><body>" + strings.Repeat("<p>section with keyword</p>\n", 10) + "</body></html>"

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":  "search",
		"query": "keyword",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "keyword")
	assert.Contains(t, result, "找到")
}

func TestGetDraft_ModeSection_BackwardCompatible(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<div class="experience"><h3>Google</h3><p>Engineer</p></div>
		<div class="education"><h3>MIT</h3></div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Old selector param still works
	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"selector": ".experience",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Google")
	assert.NotContains(t, result, "MIT")

	// New mode=section + selector also works
	result2, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":     "section",
		"selector": ".education",
	})
	require.NoError(t, err)
	assert.Contains(t, result2, "MIT")
	assert.NotContains(t, result2, "Google")
}

func TestGetDraft_ModeFull(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body><h1>Test</h1></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode": "full",
	})
	require.NoError(t, err)
	assert.Equal(t, html, result)
}

// ---------------------------------------------------------------------------
// apply_edits step-by-step execution tests
// ---------------------------------------------------------------------------

func TestApplyEdits_StepByStep_PartialSuccess(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<p>Apple</p>
		<p>Banana</p>
		<p>Cherry</p>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Apple",
				"new_string": "Apricot",
			},
			map[string]interface{}{
				"old_string": "NonExistent",
				"new_string": "Something",
			},
		},
	}

	_, err := executor.Execute(ctx, "apply_edits", params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "op #2")
	assert.Contains(t, err.Error(), "部分成功")

	// First op should have been applied
	var updatedDraft models.Draft
	require.NoError(t, db.Where("id = ?", draft.ID).First(&updatedDraft).Error)
	assert.Contains(t, updatedDraft.HTMLContent, "Apricot")
	assert.NotContains(t, updatedDraft.HTMLContent, "Apple")
	assert.Contains(t, updatedDraft.HTMLContent, "Cherry")
}

func TestApplyEdits_StepByStep_ResultDetails(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body><p>Hello World</p></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Hello",
				"new_string": "Hi",
			},
			map[string]interface{}{
				"old_string": "World",
				"new_string": "Earth",
			},
		},
	}

	result, err := executor.Execute(ctx, "apply_edits", params)
	require.NoError(t, err)

	var parsed struct {
		Applied     int `json:"applied"`
		NewSequence int `json:"new_sequence"`
		Failed      int `json:"failed"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, 2, parsed.Applied)
	assert.Equal(t, 0, parsed.Failed)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string {
	return &s
}

// Ensure DraftEdit model is properly used in tests
var _ time.Time

// ---------------------------------------------------------------------------
// get_draft call counting tests
// ---------------------------------------------------------------------------

func TestGetDraft_CallCounting_RejectsAfterLimit(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Test</h1></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	sessionID := uint(999)
	ctx := WithDraftID(WithSessionID(context.Background(), sessionID), draft.ID)

	// Call 1: should succeed
	result1, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "structure"})
	require.NoError(t, err)
	assert.NotEmpty(t, result1)

	// Call 2: should succeed
	result2, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "full"})
	require.NoError(t, err)
	assert.Equal(t, html, result2)

	// Call 3: should be rejected
	result3, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "full"})
	require.NoError(t, err, "should not return error, just rejection message")
	assert.Contains(t, result3, "已经读取")
	assert.Contains(t, result3, "apply_edits")
	assert.NotContains(t, result3, "<html>", "should not return HTML content after limit")
}

func TestGetDraft_CallCounting_ResetsAfterClearSessionState(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Test</h1></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	sessionID := uint(998)
	ctx := WithDraftID(WithSessionID(context.Background(), sessionID), draft.ID)

	// Exhaust the limit
	executor.Execute(ctx, "get_draft", nil)
	executor.Execute(ctx, "get_draft", nil)
	result3, _ := executor.Execute(ctx, "get_draft", nil)
	assert.Contains(t, result3, "已经读取")

	// Clear session state
	executor.ClearSessionState(sessionID)

	// Should work again after clear
	result4, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "structure"})
	require.NoError(t, err)
	assert.NotContains(t, result4, "已经读取")
	assert.NotEmpty(t, result4)
}

func TestGetDraft_CallCounting_IndependentPerSession(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Test</h1></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx1 := WithDraftID(WithSessionID(context.Background(), 501), draft.ID)
	ctx2 := WithDraftID(WithSessionID(context.Background(), 502), draft.ID)

	// Session 1 exhausts limit
	executor.Execute(ctx1, "get_draft", nil)
	executor.Execute(ctx1, "get_draft", nil)
	executor.Execute(ctx1, "get_draft", nil)

	// Session 2 should still work
	result, err := executor.Execute(ctx2, "get_draft", map[string]interface{}{"mode": "structure"})
	require.NoError(t, err)
	assert.NotContains(t, result, "已经读取")
}

func TestGetDraft_ToolDescription_IncludesCallLimit(t *testing.T) {
	executor := NewAgentToolExecutor(nil, nil, nil)
	tools := executor.Tools(context.Background())

	var getDraft ToolDef
	for _, tool := range tools {
		if tool.Name == "get_draft" {
			getDraft = tool
			break
		}
	}

	assert.Contains(t, getDraft.Description, "最多调用 2 次",
		"description should mention call limit")
	assert.NotContains(t, getDraft.Description, "首次调用请使用 structure",
		"description should not encourage specific first-call behavior")
}

func TestApplyEdits_FailureIncludesRecoveryHint(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil, nil)

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
				"new_string": "Something",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "更短的唯一片段",
		"error should include recovery hint about using shorter fragments")
}

func TestSkillLoader_LoadSkillWithReferences(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	result, err := loader.LoadSkillWithReferences("resume-design")
	require.NoError(t, err)

	assert.Equal(t, "resume-design", result.Name)
	assert.NotEmpty(t, result.Description)
	assert.NotEmpty(t, result.Usage)
	assert.NotEmpty(t, result.References, "should include all reference content")

	// Verify a4-guidelines reference is included
	found := false
	for _, ref := range result.References {
		if ref.Name == "a4-guidelines" {
			found = true
			assert.NotEmpty(t, ref.Content)
			break
		}
	}
	assert.True(t, found, "a4-guidelines reference should be included")
}

func TestSkillLoader_LoadSkillWithReferences_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	_, err = loader.LoadSkillWithReferences("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skill not found")
}

func TestTools_ContainsLoadSkill(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)
	tools := executor.Tools(context.Background())

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "load_skill", "should have load_skill tool")
	assert.NotContains(t, names, "resume-design", "should not have individual skill tools")
	assert.NotContains(t, names, "resume-interview", "should not have individual skill tools")
	assert.NotContains(t, names, "get_skill_reference", "should not have get_skill_reference")
}

func TestLoadSkill_ExecutesAndReturnsFullContent(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)

	ctx := WithSessionID(context.Background(), 600)
	result, err := executor.Execute(ctx, "load_skill", map[string]interface{}{
		"skill_name": "resume-design",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "resume-design", data["name"])
	assert.NotEmpty(t, data["description"])
	assert.NotEmpty(t, data["usage"])

	refs, ok := data["references"].([]interface{})
	require.True(t, ok, "should have references array")
	assert.NotEmpty(t, refs, "references should not be empty")

	// Verify a4-guidelines is in the references
	found := false
	for _, r := range refs {
		ref := r.(map[string]interface{})
		if ref["name"] == "a4-guidelines" {
			found = true
			assert.NotEmpty(t, ref["content"])
			break
		}
	}
	assert.True(t, found, "a4-guidelines should be in references")
}

func TestLoadSkill_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)

	result, err := executor.Execute(context.Background(), "load_skill", map[string]interface{}{
		"skill_name": "nonexistent",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "skill not found")
}

func TestLoadSkill_MissingParam(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader, nil)

	result, err := executor.Execute(context.Background(), "load_skill", nil)
	require.NoError(t, err)
	assert.Contains(t, result, "skill_name")
}

// ---------------------------------------------------------------------------
// CSS whitespace normalization tests
// ---------------------------------------------------------------------------

func TestNormalizeCSSWhitespace_MultiLineToSingleLine(t *testing.T) {
	input := ".resume-document .header {\n  color: rgb(255, 255, 255);\n  position: relative;\n}"
	expected := ".resume-document .header { color: rgb(255, 255, 255); position: relative; }"
	assert.Equal(t, expected, normalizeCSSWhitespace(input))
}

func TestNormalizeCSSWhitespace_AlreadySingleLine(t *testing.T) {
	input := ".name { font-size: 16px; }"
	expected := ".name { font-size: 16px; }"
	assert.Equal(t, expected, normalizeCSSWhitespace(input))
}

func TestNormalizeCSSWhitespace_TabsAndSpaces(t *testing.T) {
	input := ".name {\tfont-size: 16px;\t\tcolor: red;\n}"
	expected := ".name { font-size: 16px; color: red; }"
	assert.Equal(t, expected, normalizeCSSWhitespace(input))
}

func TestNormalizeCSSWhitespace_EmptyString(t *testing.T) {
	assert.Equal(t, "", normalizeCSSWhitespace(""))
}

func TestNormalizeCSSWhitespace_OnlyWhitespace(t *testing.T) {
	assert.Equal(t, "", normalizeCSSWhitespace("  \n\t  "))
}

// ---------------------------------------------------------------------------
// findWithCSSNormalization tests
// ---------------------------------------------------------------------------

func TestFindWithCSSNormalization_MultiLineVsSingleLine(t *testing.T) {
	html := `<style>
.resume-document .header {
  color: rgb(255, 255, 255);
  position: relative;
}
</style>`
	oldString := ".resume-document .header { color: rgb(255, 255, 255); position: relative; }"

	start, end := findWithCSSNormalization(html, oldString)
	assert.True(t, start >= 0, "should find match")
	assert.True(t, end > start, "end should be after start")
	// Verify the matched region in original HTML
	assert.Equal(t, ".resume-document .header {\n  color: rgb(255, 255, 255);\n  position: relative;\n}", html[start:end])
}

func TestFindWithCSSNormalization_ExactMatchStillWorks(t *testing.T) {
	html := `<style>.name { font-size: 16px; }</style>`
	oldString := ".name { font-size: 16px; }"

	start, end := findWithCSSNormalization(html, oldString)
	assert.True(t, start >= 0)
	assert.Equal(t, ".name { font-size: 16px; }", html[start:end])
}

func TestFindWithCSSNormalization_NoMatch(t *testing.T) {
	html := `<style>.name { font-size: 16px; }</style>`
	oldString := ".nonexistent { color: red; }"

	start, end := findWithCSSNormalization(html, oldString)
	assert.Equal(t, -1, start)
	assert.Equal(t, -1, end)
}

func TestFindWithCSSNormalization_EmptyOldString(t *testing.T) {
	html := `<style>.name { font-size: 16px; }</style>`
	start, end := findWithCSSNormalization(html, "")
	assert.Equal(t, -1, start)
	assert.Equal(t, -1, end)
}

func TestFindWithCSSNormalization_PreservesOriginalPosition(t *testing.T) {
	// Ensure the returned positions point to the correct region in original HTML
	html := `<div>prefix</div><style>
.class-a {
  margin: 0;
}
</style><div>suffix</div>`
	oldString := ".class-a { margin: 0; }"

	start, end := findWithCSSNormalization(html, oldString)
	require.True(t, start >= 0)
	matched := html[start:end]
	assert.Contains(t, matched, ".class-a")
	assert.Contains(t, matched, "margin: 0")
}

// ---------------------------------------------------------------------------
// applyEdits CSS normalization integration tests
// ---------------------------------------------------------------------------

func TestApplyEdits_CSSNormalization_MultiLineCSS(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><head><style>
.resume-document .header {
  color: rgb(255, 255, 255);
  position: relative;
}
.resume-document .name {
  font-size: 18px;
  font-weight: 700;
}
</style></head><body><div class="header"><p class="name">John</p></div></body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Model generates single-line CSS as old_string
	result, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string":  ".resume-document .name { font-size: 18px; font-weight: 700; }",
				"new_string":  ".resume-document .name { font-size: 16px; font-weight: 700; }",
				"description": "reduce name font size",
			},
		},
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(1), data["applied"])
	assert.Equal(t, float64(0), data["failed"])

	// Verify the replacement was applied correctly
	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Contains(t, updated.HTMLContent, "font-size: 16px")
	assert.NotContains(t, updated.HTMLContent, "font-size: 18px")
	// Other CSS should be untouched
	assert.Contains(t, updated.HTMLContent, "color: rgb(255, 255, 255)")
}

func TestApplyEdits_CSSNormalization_FallbackBehavior(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body><h1>Hello</h1></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Both exact and normalized should fail for truly nonexistent content
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "nonexistent content",
				"new_string": "replacement",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "匹配失败")
}

func TestApplyEdits_CSSNormalization_ChineseContent(t *testing.T) {
	db := SetupTestDB(t)

	// Test with Chinese text before CSS to verify byte/rune offset handling
	html := `<html><head><style>
.resume-document .header {
  color: rgb(255, 255, 255);
}
</style></head><body><p>你好世界</p></body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": ".resume-document .header { color: rgb(255, 255, 255); }",
				"new_string": ".resume-document .header { color: rgb(200, 200, 200); }",
			},
		},
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(1), data["applied"])

	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Contains(t, updated.HTMLContent, "color: rgb(200, 200, 200)")
	assert.Contains(t, updated.HTMLContent, "你好世界", "Chinese content should be preserved")
}
