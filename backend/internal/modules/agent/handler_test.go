package agent

import (
	"encoding/json"
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
	h := NewHandler(NewSessionService(db), NewChatService(db, &MockAdapter{}, nil, 3))
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
