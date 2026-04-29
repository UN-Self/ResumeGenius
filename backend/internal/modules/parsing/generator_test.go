package parsing

import (
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

func TestDraftGeneratorGenerateHTML_DefaultsToMockMode(t *testing.T) {
	t.Setenv("USE_MOCK", "")

	generator := NewDraftGenerator()
	html, err := generator.GenerateHTML("anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(html), "<html") {
		t.Fatalf("expected html content, got %q", html)
	}
}

func TestDraftGeneratorGenerateHTML_ReturnsErrorWhenMockDisabled(t *testing.T) {
	t.Setenv("USE_MOCK", "false")

	generator := NewDraftGenerator()
	_, err := generator.GenerateHTML("anything")
	if err == nil {
		t.Fatal("expected error when mock mode is disabled before B10")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
