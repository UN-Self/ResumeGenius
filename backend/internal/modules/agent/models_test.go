package agent

import (
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDraftEditModel(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{ProjectID: 1, HTMLContent: "test"}
	require.NoError(t, db.Create(&draft).Error)

	edit := models.DraftEdit{
		DraftID:      draft.ID,
		Sequence:     1,
		OpType:       "search_replace",
		OldString:    "old",
		NewString:    "new",
		Description:  "test",
		HtmlSnapshot: "new",
	}
	require.NoError(t, db.Create(&edit).Error)
	assert.Equal(t, uint(1), edit.ID)
	assert.Equal(t, 1, edit.Sequence)

	var loaded models.Draft
	db.First(&loaded, draft.ID)
	assert.Equal(t, 0, loaded.CurrentEditSequence)
}
