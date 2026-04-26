package models

import (
	"testing"
	"time"
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
