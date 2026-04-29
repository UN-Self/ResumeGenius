package parsing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DraftGenerator struct {
	readFile func(path string) ([]byte, error)
}

func NewDraftGenerator() *DraftGenerator {
	return &DraftGenerator{
		readFile: os.ReadFile,
	}
}

// GenerateHTML returns a complete HTML draft. B7 stays in mock mode only.
func (g *DraftGenerator) GenerateHTML(parsedText string) (string, error) {
	_ = parsedText

	if os.Getenv("USE_MOCK") == "false" {
		return "", fmt.Errorf("real ai generation is not implemented yet")
	}

	fixturePath, err := resolveFixturePath("sample_draft.html")
	if err != nil {
		return "", err
	}

	content, err := g.readFile(fixturePath)
	if err != nil {
		return "", fmt.Errorf("read mock draft fixture: %w", err)
	}

	html := strings.TrimSpace(string(content))
	if html == "" {
		return "", fmt.Errorf("mock draft fixture is empty")
	}

	return html, nil
}

func resolveFixturePath(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	candidates := []string{
		filepath.Join(wd, "..", "..", "..", "..", "fixtures", name),
		filepath.Join(wd, "..", "fixtures", name),
		filepath.Join(wd, "fixtures", name),
		filepath.Join("..", "fixtures", name),
		filepath.Join("fixtures", name),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("fixture %s not found from %s", name, wd)
}
