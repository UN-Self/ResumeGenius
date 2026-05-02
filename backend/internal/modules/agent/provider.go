package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

// ToolCallRequest represents a tool call the AI wants to make.
type ToolCallRequest struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

// ProviderAdapter abstracts AI model invocation.
type ProviderAdapter interface {
	// StreamChat is the existing simple streaming method (keep for backward compat).
	StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error

	// StreamChatReAct supports function calling for the ReAct pattern.
	StreamChatReAct(
		ctx context.Context,
		messages []Message,
		tools []ToolDef,
		onReasoning func(chunk string) error,
		onToolCall func(call ToolCallRequest) error,
		onText func(chunk string) error,
	) error
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

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

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
			log.Printf("agent: skipping malformed SSE line: %v", err)
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

// StreamChatReAct streams a chat completion with function calling support.
func (a *OpenAIAdapter) StreamChatReAct(
	ctx context.Context,
	messages []Message,
	tools []ToolDef,
	onReasoning func(chunk string) error,
	onToolCall func(call ToolCallRequest) error,
	onText func(chunk string) error,
) error {
	apiMessages := make([]map[string]string, len(messages))
	for i, m := range messages {
		apiMessages[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	reqBodyMap := map[string]interface{}{
		"model":       a.model,
		"messages":    apiMessages,
		"temperature": 0.7,
		"stream":      true,
	}

	if len(tools) > 0 {
		apiTools := make([]map[string]interface{}, len(tools))
		for i, t := range tools {
			apiTools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		reqBodyMap["tools"] = apiTools
		reqBodyMap["tool_choice"] = "auto"
	}

	reqBody, err := json.Marshal(reqBodyMap)
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

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)

	// accumulator for tool calls received across multiple chunks (keyed by index)
	type accuToolCall struct {
		id        string
		name      string
		arguments string
	}
	toolCallAccums := make(map[int]*accuToolCall)

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
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("agent: skipping malformed SSE line: %v", err)
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		// stream reasoning content
		if choice.Delta.ReasoningContent != "" {
			if err := onReasoning(choice.Delta.ReasoningContent); err != nil {
				return err
			}
		}

		// stream text content
		if choice.Delta.Content != "" {
			if err := onText(choice.Delta.Content); err != nil {
				return err
			}
		}

		// accumulate tool call deltas (a single logical call may span multiple chunks)
		for _, tc := range choice.Delta.ToolCalls {
			acc, exists := toolCallAccums[tc.Index]
			if !exists {
				acc = &accuToolCall{}
				toolCallAccums[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				acc.arguments += tc.Function.Arguments
			}
		}

		// when finish_reason is "tool_calls", dispatch all accumulated tool calls
		if choice.FinishReason != nil && *choice.FinishReason == "tool_calls" {
			for _, acc := range toolCallAccums {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(acc.arguments), &params); err != nil {
					params = map[string]interface{}{}
				}
				call := ToolCallRequest{
					ID:     acc.id,
					Name:   acc.name,
					Params: params,
				}
				if err := onToolCall(call); err != nil {
					return err
				}
			}
			toolCallAccums = make(map[int]*accuToolCall) // reset
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// dispatch any remaining accumulated tool calls (stream ended without explicit finish_reason)
	for _, acc := range toolCallAccums {
		if acc.id == "" || acc.name == "" {
			continue
		}
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(acc.arguments), &params); err != nil {
			params = map[string]interface{}{}
		}
		call := ToolCallRequest{
			ID:     acc.id,
			Name:   acc.name,
			Params: params,
		}
		if err := onToolCall(call); err != nil {
			return err
		}
	}

	return nil
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

// StreamChatReAct simulates a complete ReAct sequence with reasoning, tool calls, and final text.
func (a *MockAdapter) StreamChatReAct(
	ctx context.Context,
	messages []Message,
	tools []ToolDef,
	onReasoning func(chunk string) error,
	onToolCall func(call ToolCallRequest) error,
	onText func(chunk string) error,
) error {
	// Step 1: reasoning
	if err := onReasoning("我需要先获取项目中的资料。"); err != nil {
		return err
	}

	// Step 2: tool call — get_project_assets
	if err := onToolCall(ToolCallRequest{
		Name: "get_project_assets",
		Params: map[string]interface{}{
			"project_id": float64(1),
		},
	}); err != nil {
		return err
	}

	// Step 3: more reasoning
	if err := onReasoning("资料显示用户有3年前端开发经验，我来生成简历。"); err != nil {
		return err
	}

	// Step 4: tool call — save_draft
	if err := onToolCall(ToolCallRequest{
		Name: "save_draft",
		Params: map[string]interface{}{
			"draft_id":     float64(1),
			"html_content": "<!DOCTYPE html><html><body><h1>简历</h1></body></html>",
		},
	}); err != nil {
		return err
	}

	// Step 5: final text
	if err := onText("我已经根据你的资料生成了简历。"); err != nil {
		return err
	}

	return nil
}
