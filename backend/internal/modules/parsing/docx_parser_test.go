package parsing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDocxParser(t *testing.T) {
	parser := NewDocxParser()
	if parser == nil {
		t.Fatal("expected parser instance")
	}
}

func TestExtractTextFromDOCXFixture(t *testing.T) {
	path := fixturePath(t, "sample_resume.docx")

	text, err := ExtractTextFromDOCX(path)
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

func TestDocxParserParseReturnsParsedContent(t *testing.T) {
	path := fixturePath(t, "sample_resume.docx")
	parser := NewDocxParser()

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
		t.Fatalf("expected no images in B3, got %d", len(parsed.Images))
	}
}

func TestExtractTextFromDOCXMissingFile(t *testing.T) {
	_, err := ExtractTextFromDOCX(filepath.Join(t.TempDir(), "missing.docx"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExtractTextFromDOCXBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.docx")
	if err := os.WriteFile(path, []byte("not a valid docx"), 0644); err != nil {
		t.Fatalf("write broken docx: %v", err)
	}

	_, err := ExtractTextFromDOCX(path)
	if err == nil {
		t.Fatal("expected error for broken docx")
	}
}

func TestExtractTextFromDOCXML(t *testing.T) {
	xmlContent := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>张三</w:t></w:r></w:p><w:p><w:r><w:t>工作经历</w:t></w:r><w:r><w:tab/></w:r><w:r><w:t>ABC 科技</w:t></w:r></w:p></w:body></w:document>`

	text, err := extractTextFromDOCXML(xmlContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "张三\n工作经历\tABC 科技") {
		t.Fatalf("unexpected extracted text: %q", text)
	}
}

func TestNormalizeDOCXText(t *testing.T) {
	raw := "\n张三\r\n\r\n  工作经历  \n\n\nABC 科技\r\n"

	normalized := normalizeDOCXText(raw)
	expected := "张三\n\n工作经历\n\nABC 科技"
	if normalized != expected {
		t.Fatalf("expected %q, got %q", expected, normalized)
	}
}
