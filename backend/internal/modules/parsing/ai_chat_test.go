package parsing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAIChat_ReturnsMockContentWhenMockEnvSet(t *testing.T) {
	os.Setenv("USE_MOCK", "true")
	defer os.Unsetenv("USE_MOCK")

	result, err := aiChat(t.Context(), "system", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# repository_mock.md") {
		t.Fatalf("expected mock analysis header, got %q", result)
	}
}

func TestAIChat_ReturnsErrorWhenNoCredentials(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	os.Unsetenv("AI_API_URL")
	os.Unsetenv("AI_API_KEY")

	_, err := aiChat(t.Context(), "system", "user")
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestAIChat_ParsesValidResponse(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}

		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if len(body.Messages) < 2 {
			t.Errorf("expected at least 2 messages, got %d", len(body.Messages))
		}
		if body.Stream {
			t.Error("expected stream=false")
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "# repository_test.md\n\nAI generated analysis",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	result, err := aiChat(t.Context(), "You are an expert", "Analyze this repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# repository_test.md") {
		t.Fatalf("expected AI response, got %q", result)
	}
}

func TestAIChat_HandlesHTTPError(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal"}`))
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	_, err := aiChat(t.Context(), "system", "user")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestAIChat_HandlesEmptyChoices(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	_, err := aiChat(t.Context(), "system", "user")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestNormalizeChatURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		{"https://api.unself.cn/api/paas/v4", "https://api.unself.cn/api/paas/v4/chat/completions"},
	}
	for _, tt := range tests {
		got := normalizeChatURL(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeChatURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
