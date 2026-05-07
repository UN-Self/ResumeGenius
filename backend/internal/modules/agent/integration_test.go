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
//
// Prerequisites:
//
//	docker compose up -d postgres
//	DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=resume_genius
//
// Run:
//
//	go test ./internal/modules/agent/... -v -tags=integration -run TestFullFlow_Integration
func TestFullFlow_Integration(t *testing.T) {
	db := SetupTestDB(t)

	// --- Seed data ---
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

	// 4. search_assets — empty result (no assets in project)
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
