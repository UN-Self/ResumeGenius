package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// ---------------------------------------------------------------------------
// ListByDraft
// ---------------------------------------------------------------------------

func TestListByDraft_Empty(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)

	versions, err := svc.ListByDraft(draft.ID)
	require.NoError(t, err)
	assert.Empty(t, versions)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_Version(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)
	label := "初始版本"
	ver, err := svc.Create(draft.ID, label)

	require.NoError(t, err)
	assert.Equal(t, draft.ID, ver.DraftID)
	require.NotNil(t, ver.Label)
	assert.Equal(t, label, *ver.Label)
	assert.Equal(t, draft.HTMLContent, ver.HTMLSnapshot)
	assert.NotZero(t, ver.ID)
	assert.NotZero(t, ver.CreatedAt)
}

func TestCreate_DefaultLabel(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)
	ver, err := svc.Create(draft.ID, "")

	require.NoError(t, err)
	require.NotNil(t, ver.Label)
	assert.Equal(t, "手动保存", *ver.Label)
}

func TestCreate_DraftNotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewVersionService(db)

	_, err := svc.Create(99999, "label")
	require.ErrorIs(t, err, ErrDraftNotFound)
}

// ---------------------------------------------------------------------------
// ListByDraft after Create
// ---------------------------------------------------------------------------

func TestListByDraft_AfterCreate(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)

	_, err := svc.Create(draft.ID, "v1")
	require.NoError(t, err)

	_, err = svc.Create(draft.ID, "v2")
	require.NoError(t, err)

	versions, err := svc.ListByDraft(draft.ID)
	require.NoError(t, err)
	require.Len(t, versions, 2)

	// v2 should be first (ORDER BY created_at DESC)
	require.NotNil(t, versions[0].Label)
	assert.Equal(t, "v2", *versions[0].Label)
	require.NotNil(t, versions[1].Label)
	assert.Equal(t, "v1", *versions[1].Label)
}

// ---------------------------------------------------------------------------
// Rollback
// ---------------------------------------------------------------------------

func TestRollback_Success(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)

	// 1. Create version v1 with the original HTML
	originalHTML := draft.HTMLContent
	v1, err := svc.Create(draft.ID, "v1")
	require.NoError(t, err)
	assert.Equal(t, originalHTML, v1.HTMLSnapshot)

	// 2. Modify the draft's HTML via test helper
	newHTML := "<html><body><h1>Modified Resume</h1></body></html>"
	UpdateDraftHTML(db, draft.ID, newHTML)

	// 3. Create version v2 with the modified HTML
	v2, err := svc.Create(draft.ID, "v2")
	require.NoError(t, err)
	assert.Equal(t, newHTML, v2.HTMLSnapshot)

	// 4. Modify the draft's HTML again
	newerHTML := "<html><body><h1>Even More Modified</h1></body></html>"
	UpdateDraftHTML(db, draft.ID, newerHTML)

	// 5. Rollback to v1
	result, err := svc.Rollback(draft.ID, v1.ID)
	require.NoError(t, err)

	// Verify the result
	assert.Equal(t, draft.ID, result.DraftID)
	assert.Equal(t, originalHTML, result.HTMLContent)
	assert.NotZero(t, result.NewVersionID)
	assert.NotZero(t, result.UpdatedAt)

	// Verify the new auto-snapshot label starts with "回退到版本"
	assert.Contains(t, result.NewVersionLabel, "回退到版本")

	// Verify the draft's HTML in DB was actually updated
	var updatedDraft models.Draft
	require.NoError(t, db.First(&updatedDraft, draft.ID).Error)
	assert.Equal(t, originalHTML, updatedDraft.HTMLContent)
}

func TestRollback_VersionNotFound(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)

	_, err := svc.Rollback(draft.ID, 99999)
	require.ErrorIs(t, err, ErrVersionNotFound)
}

func TestRollback_VersionBelongsToOtherDraft(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	otherDraft := seedDraft(t, db)

	svc := NewVersionService(db)

	// Create a version on otherDraft
	ver, err := svc.Create(otherDraft.ID, "other version")
	require.NoError(t, err)

	// Try to rollback draft using otherDraft's version
	_, err = svc.Rollback(draft.ID, ver.ID)
	require.ErrorIs(t, err, ErrVersionNotFound)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestGetByID_Success(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)

	created, err := svc.Create(draft.ID, "测试版本")
	require.NoError(t, err)

	found, err := svc.GetByID(draft.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.HTMLSnapshot, found.HTMLSnapshot)
	require.NotNil(t, found.Label)
	assert.Equal(t, "测试版本", *found.Label)
}

func TestGetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	_, err := svc.GetByID(draft.ID, 99999)
	require.ErrorIs(t, err, ErrVersionNotFound)
}

func TestGetByID_WrongDraft(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	otherDraft := seedDraft(t, db)

	svc := NewVersionService(db)

	ver, err := svc.Create(otherDraft.ID, "other")
	require.NoError(t, err)

	_, err = svc.GetByID(draft.ID, ver.ID)
	require.ErrorIs(t, err, ErrVersionNotFound)
}
