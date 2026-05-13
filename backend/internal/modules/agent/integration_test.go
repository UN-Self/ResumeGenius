//go:build integration

package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// TestFullFlow_Integration exercises the entire agent v2 tool + edit pipeline
// against a real PostgreSQL instance via Docker Compose.
func TestFullFlow_Integration(t *testing.T) {
	db := SetupTestDB(t)

	project := models.Project{Title: "Integration Test", Status: "active"}
	require.NoError(t, db.Create(&project).Error)

	draft := models.Draft{
		ProjectID:   project.ID,
		HTMLContent: "<html><body><h1>Old</h1><p>Content</p></body></html>",
	}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	ctx = WithProjectID(ctx, project.ID)
	executor := NewAgentToolExecutor(db, nil)

	// 1. get_draft — full HTML
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)
	assert.Contains(t, result, "Old")

	// 2. get_draft — CSS selector
	result, err = executor.Execute(ctx, "get_draft", map[string]interface{}{"selector": "h1"})
	require.NoError(t, err)
	assert.Contains(t, result, "Old")

	// 3. apply_edits — single replacement
	result, err = executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"old_string": "Old", "new_string": "New", "description": "title"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"applied":1`)

	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Contains(t, updated.HTMLContent, "New")
	assert.Equal(t, 1, updated.CurrentEditSequence)

	// 4. search_assets — empty result
	result, err = executor.Execute(ctx, "search_assets", map[string]interface{}{"query": "nothing"})
	require.NoError(t, err)
	assert.Contains(t, result, `"results":[]`)

	// 5. undo
	editSvc := NewEditService(db)
	html, err := editSvc.Undo(draft.ID)
	require.NoError(t, err)
	assert.Contains(t, html, "Old")

	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Equal(t, 0, updated.CurrentEditSequence)

	// 6. redo
	html, err = editSvc.Redo(draft.ID)
	require.NoError(t, err)
	assert.Contains(t, html, "New")
}

func TestIntegration_StructuredWorkflow(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html>
	<head><style>body{font-size:14px}</style></head>
	<body>
		<div class="header"><h1>John Doe</h1><p>Engineer</p></div>
		<div class="experience">
			<h2>Experience</h2>
			<div class="item"><h3>Google</h3><p>Senior Engineer</p></div>
		</div>
		<div class="education">
			<h2>Education</h2>
			<div class="item"><h3>MIT</h3><p>Computer Science</p></div>
		</div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Step 1: Get structure overview
	structure, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode": "structure",
	})
	require.NoError(t, err)
	assert.Contains(t, structure, "experience")
	assert.NotContains(t, structure, "Google")

	// Step 2: Get specific section
	section, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":     "section",
		"selector": ".experience",
	})
	require.NoError(t, err)
	assert.Contains(t, section, "Google")

	// Step 3: Search by keyword
	searchResult, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":  "search",
		"query": "MIT",
	})
	require.NoError(t, err)
	assert.Contains(t, searchResult, "MIT")

	// Step 4: Execute edit
	editResult, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Google",
				"new_string": "Alphabet",
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, editResult, `"applied":1`)

	// Step 5: Verify edit applied
	newSection, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":     "section",
		"selector": ".experience",
	})
	require.NoError(t, err)
	assert.Contains(t, newSection, "Alphabet")
	assert.NotContains(t, newSection, "Google")
}

func TestIntegration_StepByStepWithFailure(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{
		ProjectID:   1,
		HTMLContent: `<html><body><p>Apple</p><p>Banana</p></body></html>`,
	}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// First succeeds, second fails
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"old_string": "Apple", "new_string": "Apricot"},
			map[string]interface{}{"old_string": "NotFound", "new_string": "Something"},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "op #2")
	assert.Contains(t, err.Error(), "部分成功")

	// First edit should be preserved
	var updated models.Draft
	db.First(&updated, draft.ID)
	assert.Contains(t, updated.HTMLContent, "Apricot")
}

func TestIntegration_StructuredErrorFeedback(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{
		ProjectID:   1,
		HTMLContent: `<html><body><div class="job"><h3>Google</h3><p>Senior Engineer</p></div></body></html>`,
	}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Sr. Engineer at Google Inc",
				"new_string": "Staff Engineer",
			},
		},
	})
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "未找到匹配")
	assert.Contains(t, errMsg, "搜索内容")
	assert.Contains(t, errMsg, "建议")
}
