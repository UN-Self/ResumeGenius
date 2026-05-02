package models

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestProjectFields(t *testing.T) {
	p := Project{
		Title:  "测试项目",
		Status: "active",
	}
	if p.Title != "测试项目" {
		t.Errorf("expected title 测试项目, got %s", p.Title)
	}
	if p.Status != "active" {
		t.Errorf("expected status active, got %s", p.Status)
	}
}

func TestAssetType(t *testing.T) {
	a := Asset{Type: "resume_pdf", ProjectID: 1}
	if a.Type != "resume_pdf" {
		t.Errorf("expected type resume_pdf, got %s", a.Type)
	}
}

func TestDraftHTMLContent(t *testing.T) {
	d := Draft{HTMLContent: "<html></html>", ProjectID: 1}
	if d.HTMLContent != "<html></html>" {
		t.Errorf("expected html content")
	}
}

func TestVersionSnapshot(t *testing.T) {
	now := time.Now()
	v := Version{HTMLSnapshot: "<html></html>", DraftID: 1, CreatedAt: now}
	if v.HTMLSnapshot == "" {
		t.Error("expected non-empty snapshot")
	}
}

func TestAIMessageRole(t *testing.T) {
	m := AIMessage{Role: "user", Content: "hello"}
	if m.Role != "user" {
		t.Errorf("expected role user, got %s", m.Role)
	}
}

func TestJSONBValue(t *testing.T) {
	j := JSONB{"key": "value"}
	val, err := j.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}

func TestJSONBScan(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`{"key":"value"}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if j["key"] != "value" {
		t.Errorf("expected key=value, got %v", j["key"])
	}
}

func TestJSONBScanNil(t *testing.T) {
	var j JSONB
	err := j.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if j != nil {
		t.Errorf("expected nil, got %v", j)
	}
}

// --- New tests for ReAct agent data model extensions ---

func TestAISessionWithProjectID(t *testing.T) {
	projID := uint(42)
	s := AISession{
		DraftID:   1,
		ProjectID: &projID,
		Status:    "active",
	}
	if s.ProjectID == nil {
		t.Error("expected ProjectID to be non-nil")
	} else if *s.ProjectID != 42 {
		t.Errorf("expected ProjectID=42, got %d", *s.ProjectID)
	}
}

func TestAISessionStatusDefault(t *testing.T) {
	s := AISession{
		DraftID: 1,
	}
	// The default tag 'active' is enforced at DB level via GORM.
	// In Go struct zero value, Status will be empty string.
	// This test verifies that setting explicitly works and checks the GORM tag default.
	if s.Status != "" {
		t.Errorf("expected empty status in zero-value struct, got %s", s.Status)
	}
	// Verify explicit assignment
	s.Status = "active"
	if s.Status != "active" {
		t.Errorf("expected status active, got %s", s.Status)
	}
}

func TestAISessionUpdatedAt(t *testing.T) {
	now := time.Now()
	s := AISession{
		DraftID:   1,
		UpdatedAt: now,
	}
	if s.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestAIMessageWithThinking(t *testing.T) {
	thinking := "I need to analyze the resume structure..."
	m := AIMessage{
		SessionID: 1,
		Role:      "assistant",
		Content:   "Here is the updated HTML.",
		Thinking:  &thinking,
	}
	if m.Thinking == nil {
		t.Error("expected Thinking to be non-nil")
	} else if *m.Thinking != "I need to analyze the resume structure..." {
		t.Errorf("unexpected thinking content: %s", *m.Thinking)
	}
}

func TestAIMessageWithToolCallID(t *testing.T) {
	toolCallID := uint(99)
	m := AIMessage{
		SessionID:  1,
		Role:       "tool",
		Content:    `{"result": "ok"}`,
		ToolCallID: &toolCallID,
	}
	if m.ToolCallID == nil {
		t.Error("expected ToolCallID to be non-nil")
	} else if *m.ToolCallID != 99 {
		t.Errorf("expected ToolCallID=99, got %d", *m.ToolCallID)
	}
}

func TestAIToolCallFields(t *testing.T) {
	params := JSONB{"query": "update skills section"}
	result := JSONB{"success": true}
	started := time.Now()
	completed := time.Now().Add(time.Second)
	errMsg := "execution failed"

	tc := AIToolCall{
		SessionID:   1,
		ToolName:    "update_html",
		Params:      params,
		Result:      &result,
		Status:      "completed",
		Error:       &errMsg,
		StartedAt:   &started,
		CompletedAt: &completed,
	}

	if tc.ToolName != "update_html" {
		t.Errorf("expected ToolName=update_html, got %s", tc.ToolName)
	}
	if tc.Params["query"] != "update skills section" {
		t.Errorf("expected params query, got %v", tc.Params["query"])
	}
	if tc.Result == nil {
		t.Error("expected Result to be non-nil")
	} else if (*tc.Result)["success"] != true {
		t.Errorf("expected result success=true, got %v", (*tc.Result)["success"])
	}
}

func TestAIToolCallStatusDefault(t *testing.T) {
	tc := AIToolCall{
		SessionID: 1,
		ToolName:  "read_html",
		Params:    JSONB{},
	}
	// Verify the zero value
	if tc.Status != "" {
		t.Errorf("expected empty status in zero-value struct, got %s", tc.Status)
	}
	// Verify setting explicitly
	tc.Status = "pending"
	if tc.Status != "pending" {
		t.Errorf("expected status pending, got %s", tc.Status)
	}
}

func TestAIToolCallJSONBParamsValue(t *testing.T) {
	params := JSONB{"key": "value", "count": float64(3)}
	tc := AIToolCall{
		SessionID: 1,
		ToolName:  "test_tool",
		Params:    params,
	}

	val, err := tc.Params.Value()
	if err != nil {
		t.Errorf("unexpected error from Value(): %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil value from Params.Value()")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(val.([]byte), &parsed); err != nil {
		t.Errorf("unexpected error unmarshaling: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %v", parsed["key"])
	}
	if parsed["count"] != float64(3) {
		t.Errorf("expected count=3, got %v", parsed["count"])
	}
}

func TestAIToolCallJSONBResultScan(t *testing.T) {
	var result *JSONB
	// Simulate scanning a JSON value from the database
	result = &JSONB{}
	err := result.Scan([]byte(`{"success":true,"message":"done"}`))
	if err != nil {
		t.Errorf("unexpected error scanning: %v", err)
	}
	if (*result)["success"] != true {
		t.Errorf("expected success=true, got %v", (*result)["success"])
	}
	if (*result)["message"] != "done" {
		t.Errorf("expected message=done, got %v", (*result)["message"])
	}
}

func TestAIToolCallJSONBResultScanNil(t *testing.T) {
	// Simulate scanning a NULL JSON value
	var result JSONB
	err := result.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error scanning nil: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result after scanning nil, got %v", result)
	}
}

func TestAIToolCallJSONBDefaults(t *testing.T) {
	// Empty JSONB{} should marshal to {}
	tc := AIToolCall{
		SessionID: 1,
		ToolName:  "test",
		Params:    JSONB{},
	}
	val, err := tc.Params.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value for empty JSONB")
	}
	parsed := string(val.([]byte))
	if parsed != "{}" {
		t.Errorf("expected '{}', got '%s'", parsed)
	}
}

func TestAIToolCallPointerFields(t *testing.T) {
	// Verify that Error, StartedAt, CompletedAt are all nullable
	tc := AIToolCall{
		SessionID: 1,
		ToolName:  "test",
		Params:    JSONB{},
		Status:    "pending",
	}
	if tc.Error != nil {
		t.Error("expected Error to be nil for fresh tool call")
	}
	if tc.StartedAt != nil {
		t.Error("expected StartedAt to be nil for fresh tool call")
	}
	if tc.CompletedAt != nil {
		t.Error("expected CompletedAt to be nil for fresh tool call")
	}
	if tc.Result != nil {
		t.Error("expected Result to be nil for fresh tool call")
	}
}

// --- DB-dependent tests ---

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "resume_genius"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Skipf("skipping DB-dependent test: cannot connect to PostgreSQL: %v", err)
	}

	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	if err := tx.AutoMigrate(&AISession{}, &AIMessage{}, &AIToolCall{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	return tx
}

func TestAIToolCallCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create session first (required for tool call foreign key)
	session := AISession{
		DraftID: 1,
		Status:  "active",
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// CREATE: insert a tool call
	params := JSONB{"action": "update", "section": "skills"}
	tc := AIToolCall{
		SessionID: session.ID,
		ToolName:  "update_html",
		Params:    params,
		Status:    "pending",
	}
	if err := db.Create(&tc).Error; err != nil {
		t.Fatalf("failed to create tool call: %v", err)
	}
	if tc.ID == 0 {
		t.Fatal("expected tool call ID to be set after create")
	}

	// READ: fetch by ID
	var fetched AIToolCall
	if err := db.First(&fetched, tc.ID).Error; err != nil {
		t.Fatalf("failed to read tool call: %v", err)
	}
	if fetched.ToolName != "update_html" {
		t.Errorf("expected ToolName=update_html, got %s", fetched.ToolName)
	}
	if fetched.Params["action"] != "update" {
		t.Errorf("expected params action=update, got %v", fetched.Params["action"])
	}
	if fetched.Status != "pending" {
		t.Errorf("expected status pending, got %s", fetched.Status)
	}

	// UPDATE: change status to running
	if err := db.Model(&fetched).Update("status", "running").Error; err != nil {
		t.Fatalf("failed to update tool call status: %v", err)
	}
	var updated AIToolCall
	if err := db.First(&updated, tc.ID).Error; err != nil {
		t.Fatalf("failed to read updated tool call: %v", err)
	}
	if updated.Status != "running" {
		t.Errorf("expected status=running, got %s", updated.Status)
	}

	// UPDATE with result
	result := JSONB{"success": true, "html_changed": true}
	if err := db.Model(&updated).Updates(map[string]interface{}{
		"status": "completed",
		"result": result,
	}).Error; err != nil {
		t.Fatalf("failed to update tool call with result: %v", err)
	}
	var completed AIToolCall
	if err := db.First(&completed, tc.ID).Error; err != nil {
		t.Fatalf("failed to read completed tool call: %v", err)
	}
	if completed.Status != "completed" {
		t.Errorf("expected status=completed, got %s", completed.Status)
	}
	if completed.Result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if (*completed.Result)["success"] != true {
		t.Errorf("expected result success=true, got %v", (*completed.Result)["success"])
	}

	// DELETE
	if err := db.Delete(&AIToolCall{}, tc.ID).Error; err != nil {
		t.Fatalf("failed to delete tool call: %v", err)
	}
	var deleted AIToolCall
	if err := db.First(&deleted, tc.ID).Error; err == nil {
		t.Error("expected error reading deleted tool call")
	}
}

func TestAISessionWithProjectIDAndStatusDB(t *testing.T) {
	db := setupTestDB(t)

	projID := uint(100)
	session := AISession{
		DraftID:   1,
		ProjectID: &projID,
		Status:    "active",
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session with ProjectID: %v", err)
	}

	var fetched AISession
	if err := db.First(&fetched, session.ID).Error; err != nil {
		t.Fatalf("failed to read session: %v", err)
	}
	if fetched.ProjectID == nil {
		t.Fatal("expected ProjectID to be non-nil")
	}
	if *fetched.ProjectID != 100 {
		t.Errorf("expected ProjectID=100, got %d", *fetched.ProjectID)
	}
	if fetched.Status != "active" {
		t.Errorf("expected status=active, got %s", fetched.Status)
	}
}

func TestAIMessageWithThinkingDB(t *testing.T) {
	db := setupTestDB(t)

	session := AISession{
		DraftID: 1,
		Status:  "active",
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	thinking := "Analyzing the user's resume for improvement opportunities..."
	msg := AIMessage{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "I have updated the skills section.",
		Thinking:  &thinking,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatalf("failed to create message with Thinking: %v", err)
	}

	var fetched AIMessage
	if err := db.First(&fetched, msg.ID).Error; err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	if fetched.Thinking == nil {
		t.Fatal("expected Thinking to be non-nil")
	}
	if *fetched.Thinking != "Analyzing the user's resume for improvement opportunities..." {
		t.Errorf("unexpected thinking content: %s", *fetched.Thinking)
	}
}

func TestAIToolCallCRUDWithAllFields(t *testing.T) {
	db := setupTestDB(t)

	session := AISession{DraftID: 1, Status: "active"}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond) // GORM strips nanos
	params := JSONB{"tool": "html_editor", "args": []interface{}{"update", "header"}}
	result := JSONB{"changed": true, "lines": float64(5)}
	errMsg := "warning: partial update"

	tc := AIToolCall{
		SessionID:   session.ID,
		ToolName:    "html_editor",
		Params:      params,
		Result:      &result,
		Status:      "completed",
		Error:       &errMsg,
		StartedAt:   &now,
		CompletedAt: &now,
	}
	if err := db.Create(&tc).Error; err != nil {
		t.Fatalf("failed to create tool call with all fields: %v", err)
	}

	var fetched AIToolCall
	if err := db.First(&fetched, tc.ID).Error; err != nil {
		t.Fatalf("failed to read tool call: %v", err)
	}

	if fetched.ToolName != "html_editor" {
		t.Errorf("expected ToolName=html_editor, got %s", fetched.ToolName)
	}
	if fetched.Status != "completed" {
		t.Errorf("expected status=completed, got %s", fetched.Status)
	}
	if fetched.Error == nil || *fetched.Error != "warning: partial update" {
		t.Errorf("unexpected Error: %v", fetched.Error)
	}
	if !fetched.StartedAt.Equal(now) {
		t.Errorf("expected StartedAt=%v, got %v", now, fetched.StartedAt)
	}
	if !fetched.CompletedAt.Equal(now) {
		t.Errorf("expected CompletedAt=%v, got %v", now, fetched.CompletedAt)
	}
}

func TestToolCallSessionRelationship(t *testing.T) {
	db := setupTestDB(t)

	session := AISession{DraftID: 1, Status: "active"}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	tc1 := AIToolCall{SessionID: session.ID, ToolName: "read_html", Params: JSONB{"action": "read"}, Status: "completed"}
	tc2 := AIToolCall{SessionID: session.ID, ToolName: "update_html", Params: JSONB{"action": "update"}, Status: "pending"}
	if err := db.Create(&tc1).Error; err != nil {
		t.Fatalf("failed to create tc1: %v", err)
	}
	if err := db.Create(&tc2).Error; err != nil {
		t.Fatalf("failed to create tc2: %v", err)
	}

	// Query tool calls by session
	var calls []AIToolCall
	if err := db.Where("session_id = ?", session.ID).Order("id ASC").Find(&calls).Error; err != nil {
		t.Fatalf("failed to query tool calls: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(calls))
	}
	if calls[0].ToolName != "read_html" {
		t.Errorf("expected first tool call name=read_html, got %s", calls[0].ToolName)
	}
	if calls[1].ToolName != "update_html" {
		t.Errorf("expected second tool call name=update_html, got %s", calls[1].ToolName)
	}
}

func TestToolCallCascadeDelete(t *testing.T) {
	db := setupTestDB(t)

	session := AISession{DraftID: 1, Status: "active"}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	tc1 := AIToolCall{SessionID: session.ID, ToolName: "tool1", Params: JSONB{"a": 1}, Status: "pending"}
	tc2 := AIToolCall{SessionID: session.ID, ToolName: "tool2", Params: JSONB{"b": 2}, Status: "running"}
	if err := db.Create(&tc1).Error; err != nil {
		t.Fatalf("failed to create tc1: %v", err)
	}
	if err := db.Create(&tc2).Error; err != nil {
		t.Fatalf("failed to create tc2: %v", err)
	}

	// Delete session - this should trigger BeforeDelete hook
	if err := db.Delete(&session).Error; err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify tool calls were cascade-deleted
	var count int64
	db.Model(&AIToolCall{}).Where("session_id = ?", session.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 tool calls after session delete, got %d", count)
	}
}
