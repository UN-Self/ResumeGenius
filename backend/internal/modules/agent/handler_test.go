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
	h := NewHandler(NewSessionService(db), NewChatService(db, &MockAdapter{}, &MockToolExecutor{}, 3, nil), NewEditService(db))
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
	h := NewHandler(NewSessionService(db), NewChatService(db, &ToolCallLoopMock{}, &MockToolExecutor{}, 3, nil), NewEditService(db))
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
// apply_edits changes to the database.
type mockToolExecutor struct {
	results map[string]string // tool name -> result JSON
	errors  map[string]error  // tool name -> error
	db      *gorm.DB          // optional — if set, apply_edits updates the draft row
}

func (m *mockToolExecutor) Tools(_ context.Context) []ToolDef {
	return NewAgentToolExecutor(nil, nil).Tools(context.Background())
}

func (m *mockToolExecutor) Execute(_ context.Context, name string, params map[string]interface{}) (string, error) {
	if err, ok := m.errors[name]; ok {
		return "", err
	}
	// When a db is available and the tool is apply_edits, return canned response.
	if name == "apply_edits" && m.db != nil {
		out, _ := json.Marshal(map[string]interface{}{
			"applied":      1,
			"new_sequence": 1,
		})
		return string(out), nil
	}
	if result, ok := m.results[name]; ok {
		return result, nil
	}
	return "{}", nil
}

func (m *mockToolExecutor) ClearSessionState(_ uint) {}

// ---------------------------------------------------------------------------
// Custom mock adapters for specific test scenarios
// ---------------------------------------------------------------------------

// fullPipelineMockAdapter simulates the same ReAct sequence as MockAdapter but
// uses the provided draft/project IDs so tool execution against the DB works.
type fullPipelineMockAdapter struct {
	callCount int
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
	a.callCount++
	switch a.callCount {
	case 1:
		// First call: reasoning + get_draft
		if err := onReasoning("我需要先获取项目中的资料。"); err != nil {
			return err
		}
		return onToolCall(ToolCallRequest{
			Name:   "get_draft",
			ID:     "call_fp_1",
			Params: map[string]interface{}{"selector": ""},
		})
	case 2:
		// Second call: reasoning + apply_edits
		if err := onReasoning("简历内容已获取，我来应用修改。"); err != nil {
			return err
		}
		return onToolCall(ToolCallRequest{
			ID:   "call_fp_2",
			Name: "apply_edits",
			Params: map[string]interface{}{
				"ops": []interface{}{
					map[string]interface{}{
						"old_string":  "<h1>Mock</h1>",
						"new_string":  "<h1>简历</h1>",
						"description": "set heading",
					},
				},
			},
		})
	default:
		// Third call: final text response
		return onText("我已经完成了简历的修改。")
	}
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

	// Adapter using new tool names (get_draft, apply_edits).
	adapter := &fullPipelineMockAdapter{}

	// Mock executor with canned responses.
	executor := &mockToolExecutor{
		db: db,
		results: map[string]string{
			"get_draft":   `<html><body>test resume</body></html>`,
			"apply_edits": `{"applied":1,"new_sequence":1}`,
		},
	}

	chatSvc := NewChatService(db, adapter, executor, 3, nil)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc, NewEditService(db))

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
	require.GreaterOrEqual(t, len(events), 7, "should have at least 7 SSE events (including edit event)")

	// --- Verify event type sequence ---
	// service.go sends a thinking event at the start of each iteration ("第 X 步：正在请求模型生成下一步操作...")
	// fullPipelineMockAdapter sends: iteration 1: thinking + tool_call (get_draft)
	//                                iteration 2: thinking + tool_call (apply_edits) + edit
	//                                iteration 3: text
	types := make([]string, len(events))
	for i, ev := range events {
		types[i] = ev["type"].(string)
	}

	// Find tool_call events and their indices
	var toolCallIndices []int
	for i, t := range types {
		if t == "tool_call" {
			toolCallIndices = append(toolCallIndices, i)
		}
	}
	require.GreaterOrEqual(t, len(toolCallIndices), 2, "should have at least 2 tool_call events")

	// Verify tool_call names
	assert.Equal(t, "get_draft", events[toolCallIndices[0]]["name"])
	assert.Equal(t, "apply_edits", events[toolCallIndices[1]]["name"])

	// Verify tool_result events follow tool_call events
	assert.Equal(t, "tool_result", types[toolCallIndices[0]+1])

	// For apply_edits, edit event is sent before tool_result
	applyEditsIdx := toolCallIndices[1]
	assert.Equal(t, "edit", types[applyEditsIdx+1], "edit event should follow apply_edits tool_call")
	assert.Equal(t, "tool_result", types[applyEditsIdx+2], "tool_result should follow edit event")

	// Verify tool_result statuses
	assert.Equal(t, "completed", events[toolCallIndices[0]+1]["status"])
	assert.Equal(t, "completed", events[applyEditsIdx+2]["status"])

	// Verify text and done events exist
	var hasText, hasDone, hasEdit bool
	for _, ev := range events {
		switch ev["type"] {
		case "text":
			hasText = true
		case "done":
			hasDone = true
		case "edit":
			hasEdit = true
		}
	}
	assert.True(t, hasText, "should have at least one text event")
	assert.True(t, hasDone, "should have a done event")
	assert.True(t, hasEdit, "should have an edit event for apply_edits")

	// --- Verify draft HTML exists ---
	var draft models.Draft
	err = db.First(&draft, draftID).Error
	require.NoError(t, err)
	assert.Contains(t, draft.HTMLContent, "test", "draft HTML content should exist")

	// --- Verify messages saved to DB ---
	var messages []models.AIMessage
	err = db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	require.NoError(t, err)
	require.Len(t, messages, 2, "should have exactly 2 messages (user + assistant)")
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "帮我生成一份简历", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "我已经完成了简历的修改。", messages[1].Content)
}

func TestHandler_Chat_ToolExecutionError(t *testing.T) {
	db := SetupTestDB(t)
	draftID := seedDraft(t, db)

	// ToolCallLoopMock only produces tool calls (never text).
	// Combined with an executor that always errors, the ReAct loop hits maxIterations.
	executor := &mockToolExecutor{
		errors: map[string]error{
				"get_draft":   fmt.Errorf("database connection failed"),
				"apply_edits": fmt.Errorf("permission denied"),
		},
	}

	chatSvc := NewChatService(db, &ToolCallLoopMock{}, executor, 3, nil)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc, NewEditService(db))

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

	// Verify tool_call and tool_result events
	// ToolCallLoopMock returns get_draft on every call, so we get tool calls on each iteration
	// maxIterations=3, so max iterations = 3*2+1 = 7
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

	assert.Greater(t, toolCallCount, 0, "should have at least 1 tool_call event")
	assert.Equal(t, toolCallCount, toolResultCount, "should have equal tool_call and tool_result events")
	assert.True(t, hasErrorEvent, "should have an error event after max iterations exceeded")
	assert.Equal(t, float64(CodeMaxIterations), errorCode, "error code should be max iterations (3005)")
}

func TestHandler_Chat_NoToolsNeeded(t *testing.T) {
	db := SetupTestDB(t)
	draftID := seedDraft(t, db)

	adapter := &textOnlyMockAdapter{response: "你好！我是简历助手，有什么可以帮你的吗？"}
	executor := &mockToolExecutor{results: map[string]string{}}

	chatSvc := NewChatService(db, adapter, executor, 3, nil)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc, NewEditService(db))

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

	// Should only have thinking + text + done — no tool_call, tool_result, or error.
	// service.go sends a thinking event at the start of each iteration.
	var hasText, hasDone bool
	var hasToolCall, hasToolResult, hasError bool
	for _, ev := range events {
		switch ev["type"] {
		case "text":
			hasText = true
		case "done":
			hasDone = true
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

	chatSvc := NewChatService(db, adapter, executor, 3, nil)
	sessionSvc := NewSessionService(db)
	h := NewHandler(sessionSvc, chatSvc, NewEditService(db))

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

// ---------------------------------------------------------------------------
// Undo / Redo tests
// ---------------------------------------------------------------------------

func TestHandler_Undo(t *testing.T) {
	db := SetupTestDB(t)

	project := models.Project{Title: "test"}
	db.Create(&project)
	draft := models.Draft{ProjectID: project.ID, HTMLContent: `<html><body><h1>V0</h1></body></html>`}
	db.Create(&draft)

	db.Create(&models.DraftEdit{DraftID: draft.ID, Sequence: 0, OpType: "base", HtmlSnapshot: draft.HTMLContent})
	html1 := `<html><body><h1>V1</h1></body></html>`
	db.Create(&models.DraftEdit{DraftID: draft.ID, Sequence: 1, OpType: "search_replace", OldString: "V0", NewString: "V1", HtmlSnapshot: html1})
	html2 := `<html><body><h1>V2</h1></body></html>`
	db.Create(&models.DraftEdit{DraftID: draft.ID, Sequence: 2, OpType: "search_replace", OldString: "V1", NewString: "V2", HtmlSnapshot: html2})
	db.Model(&draft).Update("current_edit_sequence", 2)

	editSvc := NewEditService(db)
	sessionSvc := NewSessionService(db)
	provider := &MockAdapter{}
	toolExecutor := NewAgentToolExecutor(db, nil)
	chatSvc := NewChatService(db, provider, toolExecutor, 10, nil)
	h := NewHandler(sessionSvc, chatSvc, editSvc)

	r := gin.New()
	r.POST("/ai/drafts/:draft_id/undo", h.Undo)
	r.POST("/ai/drafts/:draft_id/redo", h.Redo)

	// Test Undo
	w := doJSON(t, r, "POST", "/ai/drafts/"+strconv.Itoa(int(draft.ID))+"/undo", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	data := parseDataMap(t, resp.Data)
	assert.Equal(t, html1, data["html_content"])

	// Verify DB state
	var d models.Draft
	db.First(&d, draft.ID)
	assert.Equal(t, html1, d.HTMLContent)
	assert.Equal(t, 1, d.CurrentEditSequence)

	// Test Redo
	w = doJSON(t, r, "POST", "/ai/drafts/"+strconv.Itoa(int(draft.ID))+"/redo", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp = parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	data = parseDataMap(t, resp.Data)
	assert.Equal(t, html2, data["html_content"])

	db.First(&d, draft.ID)
	assert.Equal(t, html2, d.HTMLContent)
	assert.Equal(t, 2, d.CurrentEditSequence)
}

func TestHandler_Undo_NoMoreEdits(t *testing.T) {
	db := SetupTestDB(t)

	project := models.Project{Title: "test"}
	db.Create(&project)
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "test"}
	db.Create(&draft)

	editSvc := NewEditService(db)
	sessionSvc := NewSessionService(db)
	provider := &MockAdapter{}
	toolExecutor := NewAgentToolExecutor(db, nil)
	chatSvc := NewChatService(db, provider, toolExecutor, 10, nil)
	h := NewHandler(sessionSvc, chatSvc, editSvc)

	r := gin.New()
	r.POST("/ai/drafts/:draft_id/undo", h.Undo)

	w := doJSON(t, r, "POST", "/ai/drafts/"+strconv.Itoa(int(draft.ID))+"/undo", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
