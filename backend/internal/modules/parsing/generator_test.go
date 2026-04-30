package parsing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDraftGeneratorGenerateHTML_UsesFixtureInMockMode(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	generator := NewDraftGenerator()
	html, err := generator.GenerateHTML("张三\n前端工程师\nReact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fixturePath, err := resolveFixturePath("sample_draft.html")
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	expectedBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	expected := strings.TrimSpace(string(expectedBytes))
	if html != expected {
		t.Fatalf("expected generated html to match fixture content")
	}
	if !strings.Contains(strings.ToLower(html), "<html") {
		t.Fatalf("expected full html document, got %q", html[:min(60, len(html))])
	}
}

func TestDraftGeneratorGenerateHTML_DefaultsToRealModeWhenEnvUnset(t *testing.T) {
	t.Setenv("USE_MOCK", "")
	t.Setenv("AI_API_URL", "")
	t.Setenv("AI_API_KEY", "")

	generator := NewDraftGenerator()
	_, err := generator.GenerateHTML("anything")
	if err == nil {
		t.Fatal("expected error when defaulting to real mode without AI config")
	}
	if !strings.Contains(err.Error(), "AI_API_URL") {
		t.Fatalf("expected missing AI_API_URL error, got %v", err)
	}
}

func TestDraftGeneratorGenerateHTML_ReturnsErrorWhenParsedTextEmpty(t *testing.T) {
	t.Setenv("USE_MOCK", "true")

	generator := NewDraftGenerator()
	_, err := generator.GenerateHTML("   ")
	if err == nil {
		t.Fatal("expected error for empty parsed text")
	}
	if !strings.Contains(err.Error(), "parsed text is empty") {
		t.Fatalf("expected empty parsed text error, got %v", err)
	}
}

func TestDraftGeneratorGenerateHTML_UsesOpenAICompatibleAPIWhenMockDisabled(t *testing.T) {
	t.Setenv("USE_MOCK", "false")
	t.Setenv("AI_API_KEY", "test-key")
	t.Setenv("AI_MODEL", "resume-model")

	fixturePath, err := resolveFixturePath("sample_draft.html")
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	expectedBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	expectedHTML := strings.TrimSpace(string(expectedBytes))

	var capturedAuth string
	var capturedContentType string
	var capturedRequest chatCompletionRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		capturedAuth = r.Header.Get("Authorization")
		capturedContentType = r.Header.Get("Content-Type")

		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		resp := chatCompletionResponse{
			Choices: []chatCompletionChoice{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "```html\n" + expectedHTML + "\n```",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	t.Setenv("AI_API_URL", server.URL)
	generator := NewDraftGenerator()
	generator.httpClient = server.Client()

	html, err := generator.GenerateHTML("姓名：张三\n技能：React, TypeScript")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if html != expectedHTML {
		t.Fatalf("expected html to match server response fixture")
	}
	if capturedAuth != "Bearer test-key" {
		t.Fatalf("expected bearer auth header, got %q", capturedAuth)
	}
	if capturedContentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", capturedContentType)
	}
	if capturedRequest.Model != "resume-model" {
		t.Fatalf("expected model resume-model, got %q", capturedRequest.Model)
	}
	if len(capturedRequest.Messages) != 2 {
		t.Fatalf("expected 2 chat messages, got %d", len(capturedRequest.Messages))
	}
	if capturedRequest.Messages[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %q", capturedRequest.Messages[0].Role)
	}
	if !strings.Contains(capturedRequest.Messages[0].Content, "只输出 HTML 代码") {
		t.Fatalf("expected system prompt to include output constraint")
	}
	if capturedRequest.Messages[1].Role != "user" {
		t.Fatalf("expected second message to be user, got %q", capturedRequest.Messages[1].Role)
	}
	if !strings.Contains(capturedRequest.Messages[1].Content, "姓名：张三\n技能：React, TypeScript") {
		t.Fatalf("expected user prompt to include parsed text")
	}
	if !strings.Contains(capturedRequest.Messages[1].Content, "<!DOCTYPE html>") {
		t.Fatalf("expected user prompt to include html template")
	}
}

func TestDraftGeneratorGenerateHTML_ReturnsErrorWhenAIConfigMissing(t *testing.T) {
	t.Setenv("USE_MOCK", "false")
	t.Setenv("AI_API_URL", "")
	t.Setenv("AI_API_KEY", "")

	generator := NewDraftGenerator()
	_, err := generator.GenerateHTML("anything")
	if err == nil {
		t.Fatal("expected error when AI config is missing")
	}
	if !strings.Contains(err.Error(), "AI_API_URL") {
		t.Fatalf("expected missing url error, got %v", err)
	}
}

func TestDraftGeneratorGenerateHTML_ReturnsErrorWhenAIServerFails(t *testing.T) {
	t.Setenv("USE_MOCK", "false")
	t.Setenv("AI_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	t.Setenv("AI_API_URL", server.URL)
	generator := NewDraftGenerator()
	generator.httpClient = server.Client()

	_, err := generator.GenerateHTML("anything")
	if err == nil {
		t.Fatal("expected error when AI server returns non-2xx")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}

func TestDraftGeneratorGenerateHTML_ReturnsErrorWhenAIResponseTooLarge(t *testing.T) {
	t.Setenv("USE_MOCK", "false")
	t.Setenv("AI_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat("a", maxAIResponseBodyBytes+16)))
	}))
	defer server.Close()

	t.Setenv("AI_API_URL", server.URL)
	generator := NewDraftGenerator()
	generator.httpClient = server.Client()

	_, err := generator.GenerateHTML("anything")
	if err == nil {
		t.Fatal("expected error when AI response body is too large")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected response size limit error, got %v", err)
	}
}

func TestParseAIResponse_StripsMarkdownCodeFence(t *testing.T) {
	body := []byte("{\"choices\":[{\"message\":{\"content\":\"```html\\n<!DOCTYPE html><html><body>mock</body></html>\\n```\"}}]}")

	html, err := parseAIResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "<!DOCTYPE html><html><body>mock</body></html>" {
		t.Fatalf("unexpected html: %q", html)
	}
}

func TestParseAIResponse_ReturnsErrorWhenChoicesEmpty(t *testing.T) {
	_, err := parseAIResponse([]byte(`{"choices":[]}`))
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
