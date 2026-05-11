package parsing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAIModel   = "default"
	aiChatTimeout    = 120 * time.Second
	aiChatTemperature = 0.3
)

func aiChat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if strings.TrimSpace(os.Getenv("USE_MOCK")) == "true" {
		return `# repository_mock.md — AI mock analysis

## 常用命令
- 构建: make build
- 测试: make test

## 高层架构
Mock analysis — AI provider not configured.
`, nil
	}

	apiURL := strings.TrimSpace(os.Getenv("AI_API_URL"))
	apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	if apiURL == "" || apiKey == "" {
		return "", fmt.Errorf("AI_API_URL or AI_API_KEY not configured")
	}

	model := strings.TrimSpace(os.Getenv("AI_MODEL"))
	if model == "" {
		model = defaultAIModel
	}

	endpoint := normalizeChatURL(apiURL)

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": aiChatTemperature,
		"stream":      false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal AI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create AI request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: aiChatTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read AI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse AI response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI response has no choices")
	}

	return result.Choices[0].Message.Content, nil
}

func normalizeChatURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/chat/completions") {
		return raw
	}
	if strings.HasSuffix(raw, "/v1") || strings.HasSuffix(raw, "/api/paas/v4") {
		return raw + "/chat/completions"
	}
	return raw + "/v1/chat/completions"
}
