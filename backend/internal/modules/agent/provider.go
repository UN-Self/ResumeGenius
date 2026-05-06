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

// ToolCallMessage represents a tool call in an assistant message (for OpenAI API format).
type ToolCallMessage struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction holds the function name and arguments for a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Message represents a single chat message for the AI provider.
type Message struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"`
	ToolCalls  []ToolCallMessage `json:"tool_calls,omitempty"`
}

// ToolCallRequest represents a tool call the AI wants to make.
type ToolCallRequest struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

// ProviderAdapter abstracts AI model invocation.
type ProviderAdapter interface {
	// StreamChat is used for compaction (summarizing conversation history).
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
	apiMessages := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		msg := map[string]interface{}{"role": m.Role}
		if m.Content != "" {
			msg["content"] = m.Content
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if m.Name != "" {
			msg["name"] = m.Name
		}
		if len(m.ToolCalls) > 0 {
			msg["tool_calls"] = m.ToolCalls
		}
		apiMessages[i] = msg
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
// It is stateful: first StreamChatReAct call returns reasoning + tool calls,
// second call returns final text. This matches real API behavior where
// tool_calls and final text come in separate responses.
type MockAdapter struct{ callCount int }

func (a *MockAdapter) StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error {
	chunks := []string{"好的，我来帮你处理。"}
	for _, c := range chunks {
		if err := sendChunk(c); err != nil {
			return err
		}
	}
	return nil
}

// StreamChatReAct simulates a complete ReAct sequence across multiple calls.
func (a *MockAdapter) StreamChatReAct(
	ctx context.Context,
	messages []Message,
	tools []ToolDef,
	onReasoning func(chunk string) error,
	onToolCall func(call ToolCallRequest) error,
	onText func(string) error,
) error {
	a.callCount++
	if a.callCount == 1 {
		// First call: reasoning + tool calls (no text)
		_ = onReasoning("让我先查看当前简历内容。")
		_ = onToolCall(ToolCallRequest{
			ID:   "call_mock_1",
			Name: "get_draft",
			Params: map[string]interface{}{
				"selector": "",
			},
		})
		_ = onReasoning("简历内容已获取，我来应用修改。")
		_ = onToolCall(ToolCallRequest{
			ID:   "call_mock_2",
			Name: "apply_edits",
			Params: map[string]interface{}{
				"ops": []interface{}{
					map[string]interface{}{
						"old_string":  "<h1>Mock</h1>",
						"new_string":  "<h1>Updated</h1>",
						"description": "update heading",
					},
				},
			},
		})
		return nil
	}
	// Subsequent calls: final text
	return onText("我已经完成了简历的修改。")

	return nil
}
