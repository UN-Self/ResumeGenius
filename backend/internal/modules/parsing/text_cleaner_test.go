package parsing

import (
	"testing"
	"unicode/utf8"
)

func TestCleanParsedText_NormalizesWhitespaceAndParagraphs(t *testing.T) {
	raw := "\n张三\r\n\r\n  工作经历  \n\n\nABC   科技\t前端工程师\r\n"

	got := cleanParsedText(raw)
	want := "张三\n\n工作经历\n\nABC 科技 前端工程师"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCleanParsedText_RemovesSeparatorsAndPageNumbers(t *testing.T) {
	raw := "Page 2 of 3\n----\n张三\n第 2 页 / 共 3 页\n1 / 3\n工作经历"

	got := cleanParsedText(raw)
	want := "张三\n工作经历"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCleanParsedText_NormalizesBulletPrefixes(t *testing.T) {
	raw := "• React 工程化\n· TypeScript 组件设计\n* 性能优化\n- 测试建设"

	got := cleanParsedText(raw)
	want := "- React 工程化\n- TypeScript 组件设计\n- 性能优化\n- 测试建设"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestTruncateCleanedText_PreservesUTF8Boundaries(t *testing.T) {
	got := truncateCleanedText("你好世界Go", 3)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8 string, got %q", got)
	}
	if got != "你好世..." {
		t.Fatalf("expected rune-safe truncation, got %q", got)
	}
}

func TestCleanParsedContentText_PreservesImages(t *testing.T) {
	parsed := &ParsedContent{
		Text: "Page 1 of 2\n• React   工程化",
		Images: []ParsedImage{
			{Description: "头像", DataBase64: "abc"},
		},
	}

	cleaned := cleanParsedContentText(parsed)
	if cleaned == nil {
		t.Fatal("expected cleaned content")
	}
	if cleaned.Text != "- React 工程化" {
		t.Fatalf("expected cleaned text, got %q", cleaned.Text)
	}
	if len(cleaned.Images) != 1 || cleaned.Images[0].Description != "头像" {
		t.Fatalf("expected image metadata to be preserved, got %+v", cleaned.Images)
	}
}
