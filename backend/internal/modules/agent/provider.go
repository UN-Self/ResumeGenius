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
	"regexp"
	"strings"
	"time"
)

// retryWithBackoff executes fn with exponential backoff on server errors (5xx).
// Does NOT retry on 429 (rate limit) or 4xx errors.
func retryWithBackoff(maxRetries int, baseDelay time.Duration, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if isNonRetryable(err) {
			return err
		}
		if attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("重试 %d 次后仍失败: %w", maxRetries, lastErr)
}

var httpStatusRe = regexp.MustCompile(`status["':\s]+(\d{3})`)

func isNonRetryable(err error) bool {
	errStr := err.Error()
	matches := httpStatusRe.FindAllStringSubmatch(errStr, -1)
	for _, m := range matches {
		if len(m) > 1 && strings.HasPrefix(m[1], "4") {
			return true
		}
	}
	return false
}

func wrapTimeoutError(err error, start time.Time) error {
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		debugLog("provider", "模型调用超时，耗时 %v: %v", time.Since(start), err)
		return fmt.Errorf("%w: %w", ErrModelTimeout, err)
	}
	return nil
}

func checkHTTPResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

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
	Role             string            `json:"role"`
	Content          string            `json:"content,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	ToolCallID       string            `json:"tool_call_id,omitempty"`
	Name             string            `json:"name,omitempty"`
	ToolCalls        []ToolCallMessage `json:"tool_calls,omitempty"`
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
	return NewOpenAIAdapterWithTimeout(apiURL, apiKey, model, 180*time.Second)
}

func NewOpenAIAdapterWithTimeout(apiURL, apiKey, model string, timeout time.Duration) *OpenAIAdapter {
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	return &OpenAIAdapter{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: timeout},
	}
}

func (a *OpenAIAdapter) StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error {
	return retryWithBackoff(3, 1*time.Second, func() error {
		return a.streamChatOnce(ctx, messages, sendChunk)
	})
}

func (a *OpenAIAdapter) streamChatOnce(ctx context.Context, messages []Message, sendChunk func(string) error) error {
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
	debugLog("provider", "正在调用模型 %s（StreamChat），消息数 %d", a.model, len(messages))
	start := time.Now()

	resp, err := a.client.Do(req)
	if err != nil {
		if timeoutErr := wrapTimeoutError(err, start); timeoutErr != nil {
			return timeoutErr
		}
		return fmt.Errorf("model call failed: %w", err)
	}
	defer resp.Body.Close()

	debugLog("provider", "模型响应，状态码 %d，耗时 %v", resp.StatusCode, time.Since(start))
	if err := checkHTTPResponse(resp); err != nil {
		return err
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

	if err := scanner.Err(); err != nil {
		debugLog("provider", "SSE 流读取异常: %v", err)
		return err
	}
	return nil
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
	return retryWithBackoff(3, 1*time.Second, func() error {
		return a.streamChatReActOnce(ctx, messages, tools, onReasoning, onToolCall, onText)
	})
}

func (a *OpenAIAdapter) streamChatReActOnce(
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

		// tool-role: always carry tool_call_id and name
		if m.Role == "tool" {
			msg["content"] = m.Content
			msg["tool_call_id"] = m.ToolCallID
			if m.Name != "" {
				msg["name"] = m.Name
			}
			apiMessages[i] = msg
			continue
		}

		// assistant with tool_calls: serialize tool_calls array, content can be empty
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			if m.Content != "" {
				msg["content"] = m.Content
			}
			if m.ReasoningContent != "" {
				msg["reasoning_content"] = m.ReasoningContent
			}
			tcs := make([]map[string]interface{}, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				tcs[j] = map[string]interface{}{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			msg["tool_calls"] = tcs
			apiMessages[i] = msg
			continue
		}

		// user, system, or assistant without tool_calls
		content := m.Content
		if content == "" {
			content = " "
		}
		msg["content"] = content
		if m.ReasoningContent != "" {
			msg["reasoning_content"] = m.ReasoningContent
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
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

	// Debug: log the last 2 messages (assistant tool_calls + tool result) for debugging 400 errors
	if len(tools) > 0 && len(apiMessages) > 2 {
		for i := len(apiMessages) - 2; i < len(apiMessages); i++ {
			if b, err := json.Marshal(apiMessages[i]); err == nil {
				preview := string(b)
				if len(preview) > 1000 {
					preview = preview[:1000] + "...(truncated)"
				}
				debugLog("provider", "消息[%d] 完整结构: %s", i, preview)
			}
		}
	}

	// Debug: verify total content size being sent
	totalChars := 0
	for _, m := range apiMessages {
		if c, ok := m["content"].(string); ok {
			totalChars += len(c)
		}
	}
	debugLog("provider", "发送消息总数 %d，总内容字符数 %d", len(apiMessages), totalChars)

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
	debugLog("provider", "正在调用模型 %s（ReAct），消息数 %d，工具数 %d", a.model, len(messages), len(tools))
	start := time.Now()

	resp, err := a.client.Do(req)
	if err != nil {
		if timeoutErr := wrapTimeoutError(err, start); timeoutErr != nil {
			return timeoutErr
		}
		return fmt.Errorf("model call failed: %w", err)
	}
	defer resp.Body.Close()

	debugLog("provider", "模型响应，状态码 %d，耗时 %v", resp.StatusCode, time.Since(start))
	if err := checkHTTPResponse(resp); err != nil {
		return err
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
			toolNames := make([]string, 0, len(toolCallAccums))
			for _, acc := range toolCallAccums {
				toolNames = append(toolNames, acc.name)
			}
			debugLog("provider", "模型请求调用 %d 个工具: %v", len(toolCallAccums), toolNames)
			for _, acc := range toolCallAccums {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(acc.arguments), &params); err != nil {
					params = map[string]interface{}{}
					debugLog("provider", "工具 %s 参数解析失败: %v", acc.name, err)
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
		debugLog("provider", "SSE 流读取异常: %v", err)
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
			debugLog("provider", "工具 %s 参数解析失败: %v", acc.name, err)
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
	chunks := []string{"Mock AI is ready."}
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
		_ = onReasoning("Reading the current draft and design guidance.")
		_ = onToolCall(ToolCallRequest{
			ID:   "call_mock_1",
			Name: "get_draft",
			Params: map[string]interface{}{
				"selector": "",
			},
		})
		_ = onToolCall(ToolCallRequest{
			ID:     "call_mock_2",
			Name:   "load_skill",
			Params: map[string]interface{}{"skill_name": "resume-design"},
		})
		return nil
	}
	if a.callCount == 2 {
		oldString, newString := mockBodyMarkerEdit(messages)
		if oldString != "" {
			_ = onReasoning("Applying a safe mock edit.")
			return onToolCall(ToolCallRequest{
				ID:   "call_mock_3",
				Name: "apply_edits",
				Params: map[string]interface{}{
					"ops": []interface{}{
						map[string]interface{}{
							"old_string":  oldString,
							"new_string":  newString,
							"description": "mark draft as processed by mock AI",
						},
					},
				},
			})
		}
	}
	return onText("Mock AI response completed. Configure AI_API_URL and AI_API_KEY to use a real model.")
}

func mockBodyMarkerEdit(messages []Message) (string, string) {
	draftHTML := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" && messages[i].Name == "get_draft" {
			draftHTML = messages[i].Content
			break
		}
	}
	if draftHTML == "" {
		return "", ""
	}
	lower := strings.ToLower(draftHTML)
	start := strings.Index(lower, "<body")
	if start < 0 {
		return "", ""
	}
	end := strings.Index(draftHTML[start:], ">")
	if end < 0 {
		return "", ""
	}
	oldString := draftHTML[start : start+end+1]
	if strings.Contains(oldString, "data-ai-mock=") {
		return "", ""
	}
	newString := strings.TrimSuffix(oldString, ">") + ` data-ai-mock="polished">`
	return oldString, newString
}
