package workbench

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func TestCreate_Succeeds(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	// Create a project without a draft
	project := models.Project{Title: "test project", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewDraftService(tx)
	draft, err := svc.Create(project.ID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if draft == nil {
		t.Fatal("expected draft to be created, got nil")
	}

	if draft.ProjectID != project.ID {
		t.Fatalf("expected draft project_id %d, got %d", project.ID, draft.ProjectID)
	}

	if draft.HTMLContent != "" {
		t.Fatalf("expected empty HTML content, got %q", draft.HTMLContent)
	}

	if draft.ID == 0 {
		t.Fatal("expected draft ID to be set")
	}

	// Verify the draft was actually created in the database
	var fetchedDraft models.Draft
	if err := tx.First(&fetchedDraft, draft.ID).Error; err != nil {
		t.Fatalf("failed to fetch created draft from database: %v", err)
	}

	if fetchedDraft.ProjectID != project.ID {
		t.Fatalf("expected fetched draft project_id %d, got %d", project.ID, fetchedDraft.ProjectID)
	}

	if fetchedDraft.HTMLContent != "" {
		t.Fatalf("expected fetched draft to have empty HTML content, got %q", fetchedDraft.HTMLContent)
	}

	// Verify project's current_draft_id was updated
	var updatedProject models.Project
	if err := tx.First(&updatedProject, project.ID).Error; err != nil {
		t.Fatalf("failed to fetch updated project: %v", err)
	}

	if updatedProject.CurrentDraftID == nil {
		t.Fatal("expected project current_draft_id to be set, got nil")
	}

	if *updatedProject.CurrentDraftID != draft.ID {
		t.Fatalf("expected project current_draft_id %d, got %d", draft.ID, *updatedProject.CurrentDraftID)
	}
}

func TestCreate_ReturnsErrProjectNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	svc := NewDraftService(tx)
	draft, err := svc.Create(99999)

	if err != ErrProjectNotFound {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	if draft != nil {
		t.Fatalf("expected draft to be nil, got %v", draft)
	}
}

func TestCreate_ReturnsErrProjectHasDraft(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	// Create a project with an existing draft
	project := models.Project{Title: "test project", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	existingDraft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>existing</body></html>"}
	if err := tx.Create(&existingDraft).Error; err != nil {
		t.Fatal(err)
	}

	// Set the project's current_draft_id
	project.CurrentDraftID = &existingDraft.ID
	if err := tx.Save(&project).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewDraftService(tx)
	draft, err := svc.Create(project.ID)

	if err != ErrProjectHasDraft {
		t.Fatalf("expected ErrProjectHasDraft, got %v", err)
	}

	if draft != nil {
		t.Fatalf("expected draft to be nil, got %v", draft)
	}
}

func TestCreate_ErrorsAreDistinct(t *testing.T) {
	// Verify that the error variables are distinct and not equal to each other
	assert.NotEqual(t, ErrProjectNotFound, ErrProjectHasDraft, "ErrProjectNotFound and ErrProjectHasDraft should be distinct errors")
	assert.NotEqual(t, ErrProjectNotFound, ErrDraftNotFound, "ErrProjectNotFound and ErrDraftNotFound should be distinct errors")
	assert.NotEqual(t, ErrProjectHasDraft, ErrDraftNotFound, "ErrProjectHasDraft and ErrDraftNotFound should be distinct errors")
}
