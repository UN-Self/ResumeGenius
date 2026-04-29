package parsing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultAIModel                 = "default"
	defaultAIGenerationTemperature = 0.7
)

const resumeHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <style>
    @page { size: A4; margin: 0; }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Noto Sans SC', sans-serif; font-size: 10.5pt; line-height: 1.4; color: #333; }
    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
    .profile { display: flex; align-items: center; gap: 16px; margin-bottom: 12pt; }
    .avatar { width: 48pt; height: 48pt; border-radius: 50%; object-fit: cover; }
    .profile h1 { font-size: 18pt; font-weight: 700; }
    .profile p { font-size: 10pt; color: #666; margin-top: 2pt; }
    .section { margin-bottom: 10pt; }
    .section h2 { font-size: 12pt; font-weight: 600; border-bottom: 1pt solid #ddd; padding-bottom: 3pt; margin-bottom: 6pt; }
    .item { margin-bottom: 6pt; }
    .item-header { display: flex; justify-content: space-between; }
    .item h3 { font-size: 10.5pt; font-weight: 600; }
    .item .date { font-size: 9pt; color: #888; }
    .item .subtitle { font-size: 9.5pt; color: #555; }
    .item ul { padding-left: 14pt; }
    .item li { margin-bottom: 2pt; }
    .tags { display: flex; flex-wrap: wrap; gap: 6pt; }
    .tag { background: #f0f0f0; padding: 2pt 8pt; border-radius: 3pt; font-size: 9pt; }
  </style>
</head>
<body>
  <div class="resume">
    <header class="profile">
      <!-- AI 填充：头像、姓名、职位、联系方式 -->
    </header>
    <!-- AI 自由生成 section -->
  </div>
</body>
</html>`

type DraftGenerator struct {
	readFile   func(path string) ([]byte, error)
	httpClient *http.Client
	apiURL     string
	apiKey     string
	model      string
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float64                 `json:"temperature"`
}

type chatCompletionChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Text string `json:"text"`
}

type chatCompletionResponse struct {
	Choices []chatCompletionChoice `json:"choices"`
}

func NewDraftGenerator() *DraftGenerator {
	return &DraftGenerator{
		readFile: os.ReadFile,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiURL: strings.TrimSpace(os.Getenv("AI_API_URL")),
		apiKey: strings.TrimSpace(os.Getenv("AI_API_KEY")),
		model:  aiModelFromEnv(),
	}
}

// GenerateHTML returns a complete HTML draft in either mock or real AI mode.
func (g *DraftGenerator) GenerateHTML(parsedText string) (string, error) {
	parsedText = strings.TrimSpace(parsedText)
	if parsedText == "" {
		return "", fmt.Errorf("parsed text is empty")
	}

	if os.Getenv("USE_MOCK") != "false" {
		return g.generateMockHTML()
	}

	return g.generateWithRealAI(parsedText)
}

func (g *DraftGenerator) generateMockHTML() (string, error) {
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

func (g *DraftGenerator) generateWithRealAI(parsedText string) (string, error) {
	parsedText = strings.TrimSpace(parsedText)
	if parsedText == "" {
		return "", fmt.Errorf("parsed text is empty")
	}
	if g.apiURL == "" {
		return "", fmt.Errorf("AI_API_URL is required when USE_MOCK=false")
	}
	if g.apiKey == "" {
		return "", fmt.Errorf("AI_API_KEY is required when USE_MOCK=false")
	}

	requestBody := chatCompletionRequest{
		Model: g.model,
		Messages: []chatCompletionMessage{
			{
				Role:    "system",
				Content: buildGenerationSystemPrompt(),
			},
			{
				Role:    "user",
				Content: buildGenerationUserPrompt(parsedText),
			},
		},
		Temperature: defaultAIGenerationTemperature,
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal ai request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, g.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create ai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	client := g.httpClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call ai api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ai response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("ai api returned status %d: %s", resp.StatusCode, truncateResponseBody(body))
	}

	html, err := parseAIResponse(body)
	if err != nil {
		return "", err
	}
	if !strings.Contains(strings.ToLower(html), "<html") {
		return "", fmt.Errorf("ai response does not contain a complete html document")
	}

	return html, nil
}

func buildGenerationSystemPrompt() string {
	return `你是一个简历优化助手。请根据用户提供的原始简历资料，生成一份完整、专业、适合中文求职场景的 HTML 简历。

要求：
1. 输出必须是完整 HTML 文档，从 <!DOCTYPE html> 开始，包含 <html>、<head>、<body>。
2. 严格参考提供的 HTML 模板骨架，保留可打印 A4 的页面结构和 CSS 风格。
3. 只输出 HTML 代码，不要解释，不要 Markdown 代码块，不要额外说明。
4. 内容必须忠于用户资料，不要编造未提供的经历；无法确认的信息宁可留空或省略。`
}

func buildGenerationUserPrompt(parsedText string) string {
	return fmt.Sprintf(`请根据下面的资料生成简历 HTML。

HTML 模板骨架：
%s

用户资料：
%s

输出要求：
- 保持中文输出
- 可以根据资料自由组织 section，但应保持模板的整体风格
- 直接返回完整 HTML`, resumeHTMLTemplate, parsedText)
}

func parseAIResponse(body []byte) (string, error) {
	var resp chatCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse ai response json: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("ai response choices is empty")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		content = strings.TrimSpace(resp.Choices[0].Text)
	}
	if content == "" {
		return "", fmt.Errorf("ai response content is empty")
	}

	return unwrapMarkdownCodeFence(content), nil
}

func unwrapMarkdownCodeFence(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 {
		return trimmed
	}
	if strings.TrimSpace(lines[len(lines)-1]) != "```" {
		return trimmed
	}

	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
}

func truncateResponseBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "empty response body"
	}
	if len(trimmed) > 200 {
		return trimmed[:200] + "..."
	}
	return trimmed
}

func aiModelFromEnv() string {
	if model := strings.TrimSpace(os.Getenv("AI_MODEL")); model != "" {
		return model
	}
	return defaultAIModel
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
