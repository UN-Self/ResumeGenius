package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/nguyenthenguyen/docx"
)

type DocxParserImpl struct{}

func NewDocxParser() *DocxParserImpl {
	return &DocxParserImpl{}
}

func (p *DocxParserImpl) Parse(path string) (*ParsedContent, error) {
	text, err := ExtractTextFromDOCX(path)
	if err != nil {
		return nil, err
	}

	return &ParsedContent{
		Text:   text,
		Images: nil,
	}, nil
}

func ExtractTextFromDOCX(path string) (string, error) {
	reader, err := docx.ReadDocxFile(path)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer reader.Close()

	content := reader.Editable().GetContent()
	text, err := extractTextFromDOCXML(content)
	if err != nil {
		return "", fmt.Errorf("extract docx text: %w", err)
	}

	return normalizeDOCXText(text), nil
}

func extractTextFromDOCXML(content string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(content))
	var builder strings.Builder

	writeParagraphBreak := func() {
		current := builder.String()
		if current == "" || strings.HasSuffix(current, "\n") {
			return
		}
		builder.WriteString("\n")
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		switch node := token.(type) {
		case xml.StartElement:
			switch node.Name.Local {
			case "t":
				var text string
				if err := decoder.DecodeElement(&text, &node); err != nil {
					return "", err
				}
				builder.WriteString(text)
			case "tab":
				builder.WriteString("\t")
			case "br", "cr":
				builder.WriteString("\n")
			}
		case xml.EndElement:
			if node.Name.Local == "p" {
				writeParagraphBreak()
			}
		}
	}

	return builder.String(), nil
}

func normalizeDOCXText(raw string) string {
	return cleanParsedText(raw)
}
