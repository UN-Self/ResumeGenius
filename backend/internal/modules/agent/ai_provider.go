package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AIProvider is a simple non-streaming AI call interface used by tools
// that need a standalone AI response (e.g. git repo analysis).
type AIProvider interface {
	Call(ctx context.Context, model, systemPrompt, userMessage string) (string, error)
}

// OpenAIProvider implements AIProvider using a standard OpenAI-compatible API.
type OpenAIProvider struct {
	apiURL string
	apiKey string
	client *http.Client
}

// NewOpenAIProvider creates a provider reading AI_API_URL and AI_API_KEY from env.
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		apiURL: os.Getenv("AI_API_URL"),
		apiKey: os.Getenv("AI_API_KEY"),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Call(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	if p.apiURL == "" || p.apiKey == "" {
		return "", fmt.Errorf("AI_API_URL or AI_API_KEY not configured")
	}

	endpoint := normalizeAIURL(p.apiURL)

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.3,
		"stream":      false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("response has no choices")
	}

	return result.Choices[0].Message.Content, nil
}
