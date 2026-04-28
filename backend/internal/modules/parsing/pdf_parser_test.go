package parsing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPDFParser(t *testing.T) {
	parser := NewPDFParser()
	if parser == nil {
		t.Fatal("expected parser instance")
	}
}

func TestExtractTextFromPDFFixture(t *testing.T) {
	path := fixturePath(t, "sample_resume.pdf")

	text, err := ExtractTextFromPDF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty text")
	}
	if !strings.Contains(text, "张三") {
		t.Fatalf("expected extracted text to contain 张三, got: %s", text)
	}
	if !strings.Contains(text, "工作经历") {
		t.Fatalf("expected extracted text to contain 工作经历, got: %s", text)
	}
}

func TestPDFParserParseReturnsParsedContent(t *testing.T) {
	path := fixturePath(t, "sample_resume.pdf")
	parser := NewPDFParser()

	parsed, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed content")
	}
	if parsed.Text == "" {
		t.Fatal("expected parsed text")
	}
	if len(parsed.Images) != 0 {
		t.Fatalf("expected no images in B2, got %d", len(parsed.Images))
	}
}

func TestExtractTextFromPDFMissingFile(t *testing.T) {
	_, err := ExtractTextFromPDF(filepath.Join(t.TempDir(), "missing.pdf"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExtractTextFromPDFBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.pdf")
	if err := os.WriteFile(path, []byte("not a valid pdf"), 0644); err != nil {
		t.Fatalf("write broken pdf: %v", err)
	}

	_, err := ExtractTextFromPDF(path)
	if err == nil {
		t.Fatal("expected error for broken pdf")
	}
}

func TestNormalizePDFText(t *testing.T) {
	raw := "\n张三\r\n\r\n  工作经历  \n\n\nABC 科技\r\n"

	normalized := normalizePDFText(raw)
	expected := "张三\n\n工作经历\n\nABC 科技"
	if normalized != expected {
		t.Fatalf("expected %q, got %q", expected, normalized)
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}

	candidates := []string{
		filepath.Join(wd, "..", "..", "..", "..", "fixtures", name),
		filepath.Join(wd, "fixtures", name),
		filepath.Join("fixtures", name),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	t.Fatalf("fixture %s not found from %s", name, wd)
	return ""
}
