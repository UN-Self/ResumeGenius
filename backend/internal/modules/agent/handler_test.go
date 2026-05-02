package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)
	h := NewHandler(NewSessionService(db), NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ContextUserID, "test-user-1"); c.Next() })
	return h, r, db
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type apiResponse struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func parseDataMap(t *testing.T, raw json.RawMessage) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func parseDataArray(t *testing.T, raw json.RawMessage) []json.RawMessage {
	t.Helper()
	var arr []json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &arr))
	return arr
}

func seedDraft(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	proj := models.Project{Title: "test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body>test</body></html>"}
	require.NoError(t, db.Create(&draft).Error)
	return draft.ID
}

func createSessionViaHandler(t *testing.T, r *gin.Engine, draftID uint) int {
	t.Helper()
	w := doJSON(t, r, "POST", "/ai/sessions", map[string]interface{}{"draft_id": draftID})
	require.Equal(t, http.StatusOK, w.Code)
	data := parseDataMap(t, parseResponse(t, w).Data)
	return int(data["id"].(float64))
}

// --- Tests ---

func TestHandler_CreateSession(t *testing.T) {
	h, r, db := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)

	draftID := seedDraft(t, db)
	w := doJSON(t, r, "POST", "/ai/sessions", map[string]interface{}{"draft_id": draftID})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	data := parseDataMap(t, resp.Data)
	assert.Equal(t, float64(draftID), data["draft_id"])
}

func TestHandler_CreateSession_DraftNotFound(t *testing.T) {
	h, r, _ := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)

	w := doJSON(t, r, "POST", "/ai/sessions", map[string]interface{}{"draft_id": 9999})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeDraftNotFound, resp.Code)
}

func TestHandler_ListSessions(t *testing.T) {
	h, r, db := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)
	r.GET("/ai/sessions", h.ListSessions)

	draftID := seedDraft(t, db)
	doJSON(t, r, "POST", "/ai/sessions", map[string]interface{}{"draft_id": draftID})
	doJSON(t, r, "POST", "/ai/sessions", map[string]interface{}{"draft_id": draftID})

	w := doJSON(t, r, "GET", "/ai/sessions?draft_id="+strconv.Itoa(int(draftID)), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	arr := parseDataArray(t, resp.Data)
	assert.Len(t, arr, 2)
}

func TestHandler_GetSession(t *testing.T) {
	h, r, db := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)
	r.GET("/ai/sessions/:session_id", h.GetSession)

	draftID := seedDraft(t, db)
	sessionID := createSessionViaHandler(t, r, draftID)

	w := doJSON(t, r, "GET", "/ai/sessions/"+strconv.Itoa(sessionID), nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_DeleteSession(t *testing.T) {
	h, r, db := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)
	r.DELETE("/ai/sessions/:session_id", h.DeleteSession)
	r.GET("/ai/sessions/:session_id", h.GetSession)

	draftID := seedDraft(t, db)
	sessionID := createSessionViaHandler(t, r, draftID)

	w := doJSON(t, r, "DELETE", "/ai/sessions/"+strconv.Itoa(sessionID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := doJSON(t, r, "GET", "/ai/sessions/"+strconv.Itoa(sessionID), nil)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestHandler_Chat_MockMode(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	h, r, db := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)

	draftID := seedDraft(t, db)
	sessionID := createSessionViaHandler(t, r, draftID)

	req, err := http.NewRequest("POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", strings.NewReader(`{"message":"优化简历"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	body := rec.Body.String()
	// Verify all SSE event types produced by ReAct loop
	assert.Contains(t, body, `"type":"thinking"`)
	assert.Contains(t, body, `"type":"tool_call"`)
	assert.Contains(t, body, `"type":"tool_result"`)
	assert.Contains(t, body, `"type":"text"`)
	assert.Contains(t, body, `"type":"done"`)
}

func TestHandler_GetHistory(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	h, r, db := setupTestHandler(t)
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)
	r.GET("/ai/sessions/:session_id/history", h.GetHistory)

	draftID := seedDraft(t, db)
	sessionID := createSessionViaHandler(t, r, draftID)

	doJSON(t, r, "POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", map[string]string{"message": "hello"})

	w := doJSON(t, r, "GET", "/ai/sessions/"+strconv.Itoa(sessionID)+"/history", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	dataMap := parseDataMap(t, resp.Data)
	items := dataMap["items"].([]interface{})
	assert.Len(t, items, 2)
}

func TestHandler_Chat_MaxIterations(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	db := SetupTestDB(t)
	h := NewHandler(NewSessionService(db), NewChatService(db, &ToolCallLoopMock{}, &MockToolExecutor{}, 3))
	r := gin.New()
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)

	draftID := seedDraft(t, db)
	sessionID := createSessionViaHandler(t, r, draftID)

	req, err := http.NewRequest("POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", strings.NewReader(`{"message":"test"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	body := rec.Body.String()
	assert.Contains(t, body, `"type":"error"`)
	assert.Contains(t, body, strconv.Itoa(CodeMaxIterations))
}

// ---------------------------------------------------------------------------
// SSE event parsing helpers
// ---------------------------------------------------------------------------

// sseEvent represents a single parsed SSE data event.
type sseEvent map[string]interface{}

// parseSSEEvents parses an SSE response body into a slice of event maps.
func parseSSEEvents(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	for _, block := range strings.Split(body, "\n\n") {
		block = strings.TrimSpace(block)
		if !strings.HasPrefix(block, "data: ") {
			continue
		}
		data := strings.TrimPrefix(block, "data: ")
		if data == "" {
			continue
		}
		var ev sseEvent
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			events = append(events, ev)
		}
	}
	return events
}

// ---------------------------------------------------------------------------
// Configurable mock tool executor for integration tests
// ---------------------------------------------------------------------------

// mockToolExecutor is a configurable mock implementing ToolExecutor.
// It supports per-tool results and errors, and can optionally persist
// save_draft changes to the database.
type mockToolExecutor struct {
	results map[string]string // tool name -> result JSON
	errors  map[string]error  // tool name -> error
	db      *gorm.DB          // optional — if set, save_draft updates the draft row
}

func (m *mockToolExecutor) Tools() []ToolDef {
	return []ToolDef{
		{Name: "get_project_assets", Description: "Parse project assets", Parameters: map[string]interface{}{"type": "object"}},
		{Name: "save_draft", Description: "Save draft HTML", Parameters: map[string]interface{}{"type": "object"}},
	}
}

func (m *mockToolExecutor) Execute(_ context.Context, name string, params map[string]interface{}) (string, error) {
	if err, ok := m.errors[name]; ok {
		return "", err
	}
	// When a db is available and the tool is save_draft, actually persist the change.
	if name == "save_draft" && m.db != nil {
		draftID, err := getIntParam(params, "draft_id")
		if err == nil {
			htmlContent, _ := getStringParam(params, "html_content")
			if htmlContent != "" {
				_ = m.db.Model(&models.Draft{}).Where("id = ?", draftID).Update("html_content", htmlContent).Error
			}
		}
		out, _ := json.Marshal(map[string]interface{}{
			"draft_id": draftID,
			"status":   "saved",
		})
		return string(out), nil
	}
	if result, ok := m.results[name]; ok {
		return result, nil
	}
	return "{}", nil
}

// ---------------------------------------------------------------------------
// Custom mock adapters for specific test scenarios
// ---------------------------------------------------------------------------

// fullPipelineMockAdapter simulates the same ReAct sequence as MockAdapter but
// uses the provided draft/project IDs so tool execution against the DB works.
type fullPipelineMockAdapter struct {
	draftID   float64
	projectID float64
}

func (a *fullPipelineMockAdapter) StreamChat(_ context.Context, _ []Message, sendChunk func(string) error) error {
	return sendChunk("已经根据你的资料生成了简历。")
}

func (a *fullPipelineMockAdapter) StreamChatReAct(
	_ context.Context,
	_ []Message,
	_ []ToolDef,
	onReasoning func(string) error,
	onToolCall func(ToolCallRequest) error,
	onText func(string) error,
) error {
	if err := onReasoning("我需要先获取项目中的资料。"); err != nil {
		return err
	}
	if err := onToolCall(ToolCallRequest{
		Name:   "get_project_assets",
		Params: map[string]interface{}{"project_id": a.projectID},
	}); err != nil {
		return err
	}
	if err := onReasoning("资料显示用户有3年前端开发经验，我来生成简历。"); err != nil {
		return err
	}
	if err := onToolCall(ToolCallRequest{
		Name: "save_draft",
		Params: map[string]interface{}{
			"draft_id":     a.draftID,
			"html_content": "<!DOCTYPE html><html><body><h1>简历</h1><p>3年前端开发经验</p></body></html>",
		},
	}); err != nil {
		return err
	}
	if err := onText("我已经根据你的资料生成了简历。"); err != nil {
		return err
	}
	return nil
}

// textOnlyMockAdapter produces only a text response, no reasoning or tool calls.
type textOnlyMockAdapter struct {
	response string
}

func (a *textOnlyMockAdapter) StreamChat(_ context.Context, _ []Message, sendChunk func(string) error) error {
	return sendChunk(a.response)
}

func (a *textOnlyMockAdapter) StreamChatReAct(
	_ context.Context,
	_ []Message,
	_ []ToolDef,
	_ func(string) error,
	_ func(ToolCallRequest) error,
	onText func(string) error,
) error {
	if a.response == "" {
		a.response = "你好！我是简历助手，有什么可以帮你的吗？"
	}
	return onText(a.response)
}

// htmlMarkerMockAdapter returns text containing HTML markers.
type htmlMarkerMockAdapter struct{}

func (a *htmlMarkerMockAdapter) StreamChat(_ context.Context, _ []Message, sendChunk func(string) error) error {
	return sendChunk("已生成简历。\n<!--RESUME_HTML_START-->\n<html>test</html>\n<!--RESUME_HTML_END-->")
}

func (a *htmlMarkerMockAdapter) StreamChatReAct(
	_ context.Context,
	_ []Message,
	_ []ToolDef,
	_ func(string) error,
	_ func(ToolCallRequest) error,
	onText func(string) error,
) error {
	return onText("已生成简历。\n<!--RESUME_HTML_START-->\n<html>test</html>\n<!--RESUME_HTML_END-->")
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestHandler_Chat_FullPipeline(t *testing.T) {
	db := SetupTestDB(t)
	draftID := seedDraft(t, db)

	// Adapter configured with the actual draft ID so save_draft updates the right row.
	adapter := &fullPipelineMockAdapter{
		draftID:   float64(draftID),
		projectID: float64(1),
	}

	// Mock executor that persists save_draft to DB.
	executor := &mockToolExecutor{
		db: db,
		results: map[string]string{
			"get_project_assets": `{"assets":[{"id":1,"content":"test resume content"}]}`,
		},
	}

	chatSvc := NewChatService(db, adapter, executor, 3)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ContextUserID, "test-user-1"); c.Next() })
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)

	sessionID := createSessionViaHandler(t, r, draftID)

	// --- Send chat message ---
	body := `{"message":"帮我生成一份简历"}`
	req, err := http.NewRequest("POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	events := parseSSEEvents(t, rec.Body.String())
	require.GreaterOrEqual(t, len(events), 6, "should have at least 6 SSE events")

	// --- Verify event type sequence ---
	types := make([]string, len(events))
	for i, ev := range events {
		types[i] = ev["type"].(string)
	}

	assert.Equal(t, "thinking", types[0], "event 0 should be thinking")
	assert.Equal(t, "tool_call", types[1], "event 1 should be tool_call")
	assert.Equal(t, "tool_result", types[2], "event 2 should be tool_result")
	assert.Equal(t, "thinking", types[3], "event 3 should be thinking")
	assert.Equal(t, "tool_call", types[4], "event 4 should be tool_call")
	assert.Equal(t, "tool_result", types[5], "event 5 should be tool_result")

	// Verify tool_call names
	assert.Equal(t, "get_project_assets", events[1]["name"])
	assert.Equal(t, "save_draft", events[4]["name"])

	// Verify tool_result statuses
	assert.Equal(t, "completed", events[2]["status"])
	assert.Equal(t, "completed", events[5]["status"])

	// Verify text and done events exist
	var hasText, hasDone bool
	for _, ev := range events {
		switch ev["type"] {
		case "text":
			hasText = true
		case "done":
			hasDone = true
		}
	}
	assert.True(t, hasText, "should have at least one text event")
	assert.True(t, hasDone, "should have a done event")

	// --- Verify draft HTML was updated by save_draft ---
	var draft models.Draft
	err = db.First(&draft, draftID).Error
	require.NoError(t, err)
	assert.Contains(t, draft.HTMLContent, "<h1>简历</h1>", "draft HTML should have been updated by save_draft")

	// --- Verify messages saved to DB ---
	var messages []models.AIMessage
	err = db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	require.NoError(t, err)
	require.Len(t, messages, 2, "should have exactly 2 messages (user + assistant)")
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "帮我生成一份简历", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "我已经根据你的资料生成了简历。", messages[1].Content)
}

func TestHandler_Chat_ToolExecutionError(t *testing.T) {
	db := SetupTestDB(t)
	draftID := seedDraft(t, db)

	// ToolCallLoopMock only produces tool calls (never text).
	// Combined with an executor that always errors, the ReAct loop hits maxIterations.
	executor := &mockToolExecutor{
		errors: map[string]error{
			"get_project_assets": fmt.Errorf("database connection failed"),
			"save_draft":          fmt.Errorf("permission denied"),
		},
	}

	chatSvc := NewChatService(db, &ToolCallLoopMock{}, executor, 3)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ContextUserID, "test-user-1"); c.Next() })
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)

	sessionID := createSessionViaHandler(t, r, draftID)

	body := `{"message":"帮我生成简历"}`
	req, err := http.NewRequest("POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	events := parseSSEEvents(t, rec.Body.String())

	// Verify tool_call and tool_result events for each of the 3 iterations
	var toolCallCount, toolResultCount int
	var errorCode float64
	var hasErrorEvent bool

	for _, ev := range events {
		switch ev["type"] {
		case "tool_call":
			toolCallCount++
		case "tool_result":
			toolResultCount++
			assert.Equal(t, "failed", ev["status"], "tool result should have failed status")
		case "error":
			hasErrorEvent = true
			if code, ok := ev["code"]; ok {
				errorCode = code.(float64)
			}
		}
	}

	assert.Equal(t, 3, toolCallCount, "should have 3 tool_call events (one per iteration)")
	assert.Equal(t, 3, toolResultCount, "should have 3 tool_result events (one per iteration)")
	assert.True(t, hasErrorEvent, "should have an error event after max iterations exceeded")
	assert.Equal(t, float64(CodeMaxIterations), errorCode, "error code should be max iterations (3005)")
}

func TestHandler_Chat_NoToolsNeeded(t *testing.T) {
	db := SetupTestDB(t)
	draftID := seedDraft(t, db)

	adapter := &textOnlyMockAdapter{response: "你好！我是简历助手，有什么可以帮你的吗？"}
	executor := &mockToolExecutor{results: map[string]string{}}

	chatSvc := NewChatService(db, adapter, executor, 3)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ContextUserID, "test-user-1"); c.Next() })
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)

	sessionID := createSessionViaHandler(t, r, draftID)

	body := `{"message":"你好"}`
	req, err := http.NewRequest("POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	events := parseSSEEvents(t, rec.Body.String())

	// Should only have text + done — no thinking, tool_call, tool_result, or error.
	var hasText, hasDone bool
	var hasThinking, hasToolCall, hasToolResult, hasError bool
	for _, ev := range events {
		switch ev["type"] {
		case "text":
			hasText = true
		case "done":
			hasDone = true
		case "thinking":
			hasThinking = true
		case "tool_call":
			hasToolCall = true
		case "tool_result":
			hasToolResult = true
		case "error":
			hasError = true
		}
	}

	assert.True(t, hasText, "should have text event")
	assert.True(t, hasDone, "should have done event")
	assert.False(t, hasThinking, "should NOT have thinking events when no tool calls are needed")
	assert.False(t, hasToolCall, "should NOT have tool_call events")
	assert.False(t, hasToolResult, "should NOT have tool_result events")
	assert.False(t, hasError, "should NOT have error events")

	// Verify user + assistant messages saved to DB
	var messages []models.AIMessage
	err = db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	require.NoError(t, err)
	require.Len(t, messages, 2, "should have exactly 2 messages (user + assistant)")
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "你好", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "你好！我是简历助手，有什么可以帮你的吗？", messages[1].Content)
}

func TestHandler_Chat_HTMLPreviewInText(t *testing.T) {
	db := SetupTestDB(t)
	draftID := seedDraft(t, db)

	adapter := &htmlMarkerMockAdapter{}
	executor := &mockToolExecutor{results: map[string]string{}}

	chatSvc := NewChatService(db, adapter, executor, 3)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ContextUserID, "test-user-1"); c.Next() })
	r.POST("/ai/sessions", h.CreateSession)
	r.POST("/ai/sessions/:session_id/chat", h.Chat)

	sessionID := createSessionViaHandler(t, r, draftID)

	body := `{"message":"生成简历"}`
	req, err := http.NewRequest("POST", "/ai/sessions/"+strconv.Itoa(sessionID)+"/chat", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	events := parseSSEEvents(t, rec.Body.String())

	// Collate text event content
	var textContent string
	var hasDone bool
	for _, ev := range events {
		switch ev["type"] {
		case "text":
			content, ok := ev["content"].(string)
			if ok {
				textContent += content
			}
		case "done":
			hasDone = true
		}
	}

	assert.Contains(t, textContent, "<!--RESUME_HTML_START-->", "text should contain HTML start marker")
	assert.Contains(t, textContent, "<!--RESUME_HTML_END-->", "text should contain HTML end marker")
	assert.Contains(t, textContent, "<html>test</html>", "text should contain HTML content")
	assert.True(t, hasDone, "should have done event")
}
