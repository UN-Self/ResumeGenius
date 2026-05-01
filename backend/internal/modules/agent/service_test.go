package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func createTestDraftDirect(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	proj := models.Project{Title: "test-project", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body>test resume</body></html>"}
	require.NoError(t, db.Create(&draft).Error)
	return draft.ID
}

func TestSessionService_Create(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	session, err := svc.Create(draftID)

	assert.NoError(t, err)
	assert.Greater(t, session.ID, uint(0))
	assert.Equal(t, draftID, session.DraftID)
}

func TestSessionService_Create_DraftNotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSessionService(db)

	_, err := svc.Create(9999)

	assert.ErrorIs(t, err, ErrDraftNotFound)
}

func TestSessionService_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	created, err := svc.Create(draftID)
	require.NoError(t, err)

	session, err := svc.GetByID(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, session.ID)
}

func TestSessionService_GetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSessionService(db)

	_, err := svc.GetByID(9999)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionService_ListByDraftID(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	svc.Create(draftID)
	svc.Create(draftID)

	sessions, err := svc.ListByDraftID(draftID)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSessionService_Delete(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	created, err := svc.Create(draftID)
	require.NoError(t, err)
	svc.SaveMessage(created.ID, "user", "hello")

	err = svc.Delete(created.ID)
	assert.NoError(t, err)

	_, err = svc.GetByID(created.ID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionService_Delete_CascadesMessages(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	created, err := svc.Create(draftID)
	require.NoError(t, err)
	svc.SaveMessage(created.ID, "user", "hello")
	svc.SaveMessage(created.ID, "assistant", "hi")

	err = svc.Delete(created.ID)
	assert.NoError(t, err)

	messages, err := svc.GetHistory(created.ID)
	assert.NoError(t, err)
	assert.Len(t, messages, 0)
}

func TestSessionService_Delete_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSessionService(db)

	err := svc.Delete(9999)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionService_SaveMessageAndGetHistory(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	session, err := svc.Create(draftID)
	require.NoError(t, err)

	svc.SaveMessage(session.ID, "user", "hello")
	svc.SaveMessage(session.ID, "assistant", "hi")

	messages, err := svc.GetHistory(session.ID)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)
}

func TestSessionService_GetHistory_Empty(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	session, err := svc.Create(draftID)
	require.NoError(t, err)

	messages, err := svc.GetHistory(session.ID)
	assert.NoError(t, err)
	assert.Len(t, messages, 0)
}

func TestSessionService_GetDraftContent(t *testing.T) {
	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	svc := NewSessionService(db)

	content, err := svc.GetDraftContent(draftID)
	assert.NoError(t, err)
	assert.Contains(t, content, "test resume")
}

func TestSessionService_GetDraftContent_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSessionService(db)

	_, err := svc.GetDraftContent(9999)
	assert.ErrorIs(t, err, ErrDraftNotFound)
}

// --- ChatService tests ---

func TestChatService_MockStream(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	var events []string
	err = chatSvc.StreamChat(session.ID, "优化简历", func(e string) { events = append(events, e) })

	assert.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Contains(t, events[0], `"type":"text"`)

	var hasDone bool
	for _, e := range events {
		if strings.Contains(e, `"type":"done"`) { hasDone = true }
	}
	assert.True(t, hasDone)
}

func TestChatService_MockStream_SavesMessages(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	chatSvc.StreamChat(session.ID, "优化简历", func(string) {})

	messages, err := sessionSvc.GetHistory(session.ID)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)
}

func TestChatService_SessionNotFound(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	chatSvc := NewChatService(db)

	err := chatSvc.StreamChat(9999, "hello", func(string) {})
	assert.ErrorIs(t, err, ErrSessionNotFound)
}
