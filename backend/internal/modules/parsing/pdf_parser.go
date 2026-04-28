package parsing

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

type PDFParserImpl struct{}

func NewPDFParser() *PDFParserImpl {
	return &PDFParserImpl{}
}

func (p *PDFParserImpl) Parse(path string) (*ParsedContent, error) {
	text, err := ExtractTextFromPDF(path)
	if err != nil {
		return nil, err
	}

	return &ParsedContent{
		Text:   text,
		Images: nil,
	}, nil
}

func ExtractTextFromPDF(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	plainText, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, plainText); err != nil {
		return "", fmt.Errorf("read pdf text: %w", err)
	}

	return normalizePDFText(buf.String()), nil
}

func normalizePDFText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")
	cleaned := make([]string, 0, len(lines))
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(cleaned) == 0 || blankCount > 0 {
				continue
			}
			cleaned = append(cleaned, "")
			blankCount++
			continue
		}

		cleaned = append(cleaned, trimmed)
		blankCount = 0
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
