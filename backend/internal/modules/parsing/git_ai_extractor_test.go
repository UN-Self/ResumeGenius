package parsing

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAIGitExtractor_Extract_ReturnsAIAnalysis(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	aiResponse := `# repository_testrepo.md — AI analysis

## 常用命令
- 构建: go build ./...
- 测试: go test ./...
- 单测: go test ./internal/... -v -run TestName

## 高层架构
核心模块使用 embedding 复用 clone/read 能力。`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": aiResponse}},
			},
		})
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	repoDir := t.TempDir()
	writeTestFile(t, filepath.Join(repoDir, "README.md"), "# Test Repo\nA test repository.\n")
	writeTestFile(t, filepath.Join(repoDir, "go.mod"), "module example.com/testrepo\n\ngo 1.25.0\n")
	writeTestFile(t, filepath.Join(repoDir, "Makefile"), ".PHONY: build test\n\nbuild:\n\tgo build ./...\n\ntest:\n\tgo test ./...\n")

	base := NewGitExtractor()
	extractor := NewAIGitExtractor(base)
	parsed, err := extractor.Extract(repoDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed content")
	}
	if !strings.HasPrefix(parsed.Text, "# repository_testrepo.md") {
		t.Fatalf("expected AI analysis header, got %q", parsed.Text)
	}
	if !strings.Contains(parsed.Text, "go build") {
		t.Fatalf("expected build command in analysis, got %q", parsed.Text)
	}
}

func TestAIGitExtractor_Extract_IncludesReadmeInContext(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	var capturedMessages []map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]string `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		capturedMessages = body.Messages

		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "# repository_test.md\n\nAnalysis"}},
			},
		})
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	repoDir := t.TempDir()
	writeTestFile(t, filepath.Join(repoDir, "README.md"), "# UniqueTestRepoName\n")

	base := NewGitExtractor()
	extractor := NewAIGitExtractor(base)
	_, err := extractor.Extract(repoDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedMessages) < 2 {
		t.Fatalf("expected at least 2 messages (system + user), got %d", len(capturedMessages))
	}
	if capturedMessages[0]["role"] != "system" {
		t.Fatalf("expected first message to be system, got %q", capturedMessages[0]["role"])
	}
	if capturedMessages[1]["role"] != "user" {
		t.Fatalf("expected second message to be user, got %q", capturedMessages[1]["role"])
	}
	if !strings.Contains(capturedMessages[1]["content"], "# UniqueTestRepoName") {
		t.Fatalf("expected user message to contain README text, got: %s", capturedMessages[1]["content"])
	}
}

func TestAIGitExtractor_Extract_IncludesCursorRules(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	var capturedUserMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]string `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		for _, m := range body.Messages {
			if m["role"] == "user" {
				capturedUserMessage = m["content"]
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "# repository_test.md\n\nAnalysis"}},
			},
		})
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	repoDir := t.TempDir()
	rulesDir := filepath.Join(repoDir, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	writeTestFile(t, filepath.Join(rulesDir, "typescript.md"), "# TypeScript Rules\nUse strict mode.\n")
	writeTestFile(t, filepath.Join(rulesDir, "testing.md"), "# Testing Rules\nAlways write tests.\n")

	base := NewGitExtractor()
	extractor := NewAIGitExtractor(base)
	_, err := extractor.Extract(repoDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedUserMessage, "# TypeScript Rules") {
		t.Fatalf("expected cursor rules in user message, got: %s", capturedUserMessage)
	}
	if !strings.Contains(capturedUserMessage, "# Testing Rules") {
		t.Fatalf("expected testing rules in user message, got: %s", capturedUserMessage)
	}
}

func TestAIGitExtractor_Extract_ReturnsErrorForBlankURL(t *testing.T) {
	base := NewGitExtractor()
	extractor := NewAIGitExtractor(base)
	_, err := extractor.Extract("   ", "")
	if !errors.Is(err, ErrAssetURIMissing) {
		t.Fatalf("expected ErrAssetURIMissing, got %v", err)
	}
}

func TestAIGitExtractor_Extract_ReturnsErrorOnAIFailure(t *testing.T) {
	os.Setenv("USE_MOCK", "")
	defer os.Unsetenv("USE_MOCK")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	os.Setenv("AI_API_URL", server.URL)
	os.Setenv("AI_API_KEY", "test-key")
	defer os.Unsetenv("AI_API_URL")
	defer os.Unsetenv("AI_API_KEY")

	repoDir := t.TempDir()
	writeTestFile(t, filepath.Join(repoDir, "README.md"), "# Test\n")

	base := NewGitExtractor()
	extractor := NewAIGitExtractor(base)
	_, err := extractor.Extract(repoDir, "")
	if !errors.Is(err, ErrGitAIAnalysisFailed) {
		t.Fatalf("expected ErrGitAIAnalysisFailed, got %v", err)
	}
}
