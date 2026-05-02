package agent

import (
	"context"
	"fmt"
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
	chatSvc := NewChatService(db, &MockAdapter{}, nil, 3)

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
	chatSvc := NewChatService(db, &MockAdapter{}, nil, 3)

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
	chatSvc := NewChatService(db, &MockAdapter{}, nil, 3)

	err := chatSvc.StreamChat(9999, "hello", func(string) {})
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// --- MockToolExecutor for ReAct tests ---

// MockToolExecutor returns canned results for tool executions in tests.
type MockToolExecutor struct{}

func (e *MockToolExecutor) Tools() []ToolDef {
	return NewAgentToolExecutor(nil, "").Tools()
}

func (e *MockToolExecutor) Execute(_ context.Context, toolName string, _ map[string]interface{}) (string, error) {
	return fmt.Sprintf(`{"result":"%s executed"}`, toolName), nil
}

// ToolCallLoopMock only produces tool calls (no text), simulating an infinite loop.
type ToolCallLoopMock struct{}

func (m *ToolCallLoopMock) StreamChat(_ context.Context, _ []Message, _ func(string) error) error {
	return nil
}

func (m *ToolCallLoopMock) StreamChatReAct(
	_ context.Context,
	_ []Message,
	_ []ToolDef,
	_ func(string) error,
	onToolCall func(ToolCallRequest) error,
	_ func(string) error,
) error {
	return onToolCall(ToolCallRequest{
		Name:   "parse_project_assets",
		Params: map[string]interface{}{"project_id": float64(1)},
	})
}

// --- StreamChatReAct tests ---

func TestChatService_StreamChatReAct_FullSequence(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	var events []string
	err = chatSvc.StreamChatReAct(session.ID, "帮我生成简历", func(e string) { events = append(events, e) })
	require.NoError(t, err)

	// Verify all event types are present
	var hasThinking, hasToolCall, hasToolResult, hasText, hasDone bool
	var thinkingCount, toolCallCount, textCount int
	for _, e := range events {
		if strings.Contains(e, `"type":"thinking"`) {
			hasThinking = true
			thinkingCount++
		}
		if strings.Contains(e, `"type":"tool_call"`) {
			hasToolCall = true
			toolCallCount++
		}
		if strings.Contains(e, `"type":"tool_result"`) {
			hasToolResult = true
		}
		if strings.Contains(e, `"type":"text"`) {
			hasText = true
			textCount++
		}
		if strings.Contains(e, `"type":"done"`) {
			hasDone = true
		}
	}

	assert.True(t, hasThinking, "should have thinking events")
	assert.True(t, hasToolCall, "should have tool_call events")
	assert.True(t, hasToolResult, "should have tool_result events")
	assert.True(t, hasText, "should have text events")
	assert.True(t, hasDone, "should have done event")
	assert.Equal(t, 2, thinkingCount, "MockAdapter produces 2 reasoning chunks")
	assert.Equal(t, 2, toolCallCount, "MockAdapter produces 2 tool calls")
	assert.Equal(t, 1, textCount, "MockAdapter produces 1 text chunk")
}

func TestChatService_StreamChatReAct_SavesUserMessage(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	err = chatSvc.StreamChatReAct(session.ID, "帮我生成简历", func(string) {})
	require.NoError(t, err)

	// Verify user message was saved
	messages, err := sessionSvc.GetHistory(session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "帮我生成简历", messages[0].Content)
}

func TestChatService_StreamChatReAct_SavesAssistantWithThinking(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	err = chatSvc.StreamChatReAct(session.ID, "帮我生成简历", func(string) {})
	require.NoError(t, err)

	// Verify assistant message was saved with thinking
	messages, err := sessionSvc.GetHistory(session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assistantMsg := messages[1]
	assert.Equal(t, "assistant", assistantMsg.Role)
	assert.Contains(t, assistantMsg.Content, "根据你的资料生成了简历")
	require.NotNil(t, assistantMsg.Thinking, "assistant message should have thinking content")
	assert.Contains(t, *assistantMsg.Thinking, "需要先获取项目中的资料")
}

func TestChatService_StreamChatReAct_SavesToolCalls(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	err = chatSvc.StreamChatReAct(session.ID, "帮我生成简历", func(string) {})
	require.NoError(t, err)

	// Verify tool calls were saved
	var toolCalls []models.AIToolCall
	err = db.Where("session_id = ?", session.ID).Find(&toolCalls).Error
	require.NoError(t, err)
	require.Len(t, toolCalls, 2, "MockAdapter produces 2 tool calls")

	assert.Equal(t, "parse_project_assets", toolCalls[0].ToolName)
	assert.Equal(t, "completed", toolCalls[0].Status)

	assert.Equal(t, "save_draft", toolCalls[1].ToolName)
	assert.Equal(t, "completed", toolCalls[1].Status)
}

func TestChatService_StreamChatReAct_SessionNotFound(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	chatSvc := NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3)

	err := chatSvc.StreamChatReAct(9999, "hello", func(string) {})
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestChatService_StreamChatReAct_MaxIterationsExceeded(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	draftID := createTestDraftDirect(t, db)
	sessionSvc := NewSessionService(db)

	// Use a mock that only produces tool calls (no text), causing the loop to hit maxIterations
	chatSvc := NewChatService(db, &ToolCallLoopMock{}, &MockToolExecutor{}, 3)

	session, err := sessionSvc.Create(draftID)
	require.NoError(t, err)

	err = chatSvc.StreamChatReAct(session.ID, "帮我生成简历", func(string) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max tool-calling iterations exceeded")
}
