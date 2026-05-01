package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Message represents a single chat message for the AI provider.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ProviderAdapter abstracts AI model invocation.
type ProviderAdapter interface {
	StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error
}

// OpenAIAdapter implements ProviderAdapter for OpenAI-compatible APIs.
type OpenAIAdapter struct {
	apiURL string
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIAdapter(apiURL, apiKey, model string) *OpenAIAdapter {
	return &OpenAIAdapter{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *OpenAIAdapter) StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error {
	apiMessages := make([]map[string]string, len(messages))
	for i, m := range messages {
		apiMessages[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"model":       a.model,
		"messages":    apiMessages,
		"temperature": 0.7,
		"stream":      true,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.apiURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		var netErr net.Error
		if ok := errors.Is(err, context.DeadlineExceeded); ok {
			return fmt.Errorf("%w: %w", ErrModelTimeout, err)
		}
		if errors.As(err, &netErr) && netErr.Timeout() {
			return fmt.Errorf("%w: %w", ErrModelTimeout, err)
		}
		return fmt.Errorf("model call failed: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}

		if err := sendChunk(content); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// MockAdapter returns a preset mock response for testing.
type MockAdapter struct{}

func (a *MockAdapter) StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error {
	chunks := []string{
		"好的，我来帮你优化简历。",
		"\n<!--RESUME_HTML_START-->\n<html><body><h1>Mock优化简历</h1><p>这是AI生成的优化版本</p></body></html>\n<!--RESUME_HTML_END-->\n",
	}
	for _, c := range chunks {
		if err := sendChunk(c); err != nil {
			return err
		}
	}
	return nil
}
