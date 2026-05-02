package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockAdapter_StreamChat(t *testing.T) {
	adapter := &MockAdapter{}

	var chunks []string
	err := adapter.StreamChat(context.Background(), nil, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, chunks)

	full := strings.Join(chunks, "")
	assert.Contains(t, full, "Mock优化简历")
	assert.Contains(t, full, "RESUME_HTML_START")
	assert.Contains(t, full, "RESUME_HTML_END")
}

func TestMockAdapter_StreamChat_ContextCancelled(t *testing.T) {
	adapter := &MockAdapter{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := adapter.StreamChat(ctx, nil, func(chunk string) error {
		return nil
	})

	assert.NoError(t, err, "MockAdapter does not check context — it completes instantly")
}

func TestOpenAIAdapter_MessageFormat(t *testing.T) {
	adapter := NewOpenAIAdapter("http://localhost", "key", "test-model")

	assert.NotNil(t, adapter.client)
	assert.Equal(t, "http://localhost", adapter.apiURL)
	assert.Equal(t, "key", adapter.apiKey)
	assert.Equal(t, "test-model", adapter.model)
}

func TestMockAdapter_StreamChatReAct(t *testing.T) {
	adapter := &MockAdapter{}

	var reasoningChunks []string
	var toolCalls []ToolCallRequest
	var textChunks []string

	err := adapter.StreamChatReAct(
		context.Background(),
		nil,
		nil,
		func(chunk string) error {
			reasoningChunks = append(reasoningChunks, chunk)
			return nil
		},
		func(call ToolCallRequest) error {
			toolCalls = append(toolCalls, call)
			return nil
		},
		func(chunk string) error {
			textChunks = append(textChunks, chunk)
			return nil
		},
	)

	assert.NoError(t, err)

	// Verify reasoning
	require.Len(t, reasoningChunks, 2)
	assert.Contains(t, reasoningChunks[0], "需要先获取项目中的资料")
	assert.Contains(t, reasoningChunks[1], "3年前端开发经验")

	// Verify tool calls
	require.Len(t, toolCalls, 2)
	assert.Equal(t, "get_project_assets", toolCalls[0].Name)
	assert.Equal(t, float64(1), toolCalls[0].Params["project_id"])
	assert.Equal(t, "save_draft", toolCalls[1].Name)
	assert.Equal(t, float64(1), toolCalls[1].Params["draft_id"])

	// Verify final text
	require.Len(t, textChunks, 1)
	assert.Equal(t, "我已经根据你的资料生成了简历。", textChunks[0])
}

func TestMockAdapter_StreamChatReAct_ContextCancelled(t *testing.T) {
	adapter := &MockAdapter{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := adapter.StreamChatReAct(
		ctx,
		nil,
		nil,
		func(chunk string) error { return nil },
		func(call ToolCallRequest) error { return nil },
		func(chunk string) error { return nil },
	)

	assert.NoError(t, err, "MockAdapter does not check context — it completes instantly")
}

func TestOpenAIAdapter_NewOpenAIAdapter_DefaultTimeout(t *testing.T) {
	adapter := NewOpenAIAdapter("http://localhost", "key", "test-model")
	assert.NotNil(t, adapter.client)
	assert.Equal(t, 120*time.Second, adapter.client.Timeout)
}

func TestOpenAIAdapter_StreamChatReAct_IncludesToolsInRequest(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model")

	var textChunks []string
	err := adapter.StreamChatReAct(
		context.Background(),
		[]Message{{Role: "user", Content: "hello"}},
		[]ToolDef{{Name: "test_tool", Description: "A test tool"}},
		func(chunk string) error { return nil },
		func(call ToolCallRequest) error { return nil },
		func(chunk string) error { textChunks = append(textChunks, chunk); return nil },
	)

	assert.NoError(t, err)
	assert.Contains(t, capturedBody, "tools")
	assert.Contains(t, capturedBody, "tool_choice")
	assert.Equal(t, "auto", capturedBody["tool_choice"])

	// Verify tool definition structure
	tools, ok := capturedBody["tools"].([]interface{})
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool := tools[0].(map[string]interface{})
	assert.Equal(t, "function", tool["type"])
	fn, ok := tool["function"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test_tool", fn["name"])
	assert.Equal(t, "A test tool", fn["description"])

	// Verify text was streamed
	assert.Equal(t, "Hello", strings.Join(textChunks, ""))
}

func TestOpenAIAdapter_StreamChatReAct_NoTools(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model")

	err := adapter.StreamChatReAct(
		context.Background(),
		[]Message{{Role: "user", Content: "hello"}},
		nil, // no tools
		func(chunk string) error { return nil },
		func(call ToolCallRequest) error { return nil },
		func(chunk string) error { return nil },
	)

	assert.NoError(t, err)
	assert.NotContains(t, capturedBody, "tools")
	assert.NotContains(t, capturedBody, "tool_choice")
}

func TestOpenAIAdapter_StreamChatReAct_ParseToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Reasoning
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"I need to think...\"},\"finish_reason\":null}]}\n\n")
		// Tool call first chunk (with name and id)
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_abc\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}\n\n")
		// Tool call argument chunks
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"location\\\":\\\"Beijing\\\"}\"}}]},\"finish_reason\":null}]}\n\n")
		// Finish
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model")

	var reasoning []string
	var toolCalls []ToolCallRequest
	var text []string

	err := adapter.StreamChatReAct(
		context.Background(),
		[]Message{{Role: "user", Content: "weather"}},
		[]ToolDef{{Name: "get_weather", Description: "Get weather"}},
		func(chunk string) error { reasoning = append(reasoning, chunk); return nil },
		func(call ToolCallRequest) error { toolCalls = append(toolCalls, call); return nil },
		func(chunk string) error { text = append(text, chunk); return nil },
	)

	assert.NoError(t, err)

	// Verify reasoning was captured
	require.Len(t, reasoning, 1)
	assert.Contains(t, reasoning[0], "I need to think")

	// Verify tool calls were parsed
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_abc", toolCalls[0].ID)
	assert.Equal(t, "get_weather", toolCalls[0].Name)
	assert.Equal(t, "Beijing", toolCalls[0].Params["location"])

	// No text content since finish_reason was "tool_calls"
	assert.Empty(t, text)
}

func TestOpenAIAdapter_StreamChatReAct_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `{"error": {"message": "invalid model"}}`)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model")

	err := adapter.StreamChatReAct(
		context.Background(),
		[]Message{{Role: "user", Content: "hello"}},
		[]ToolDef{{Name: "test", Description: "test"}},
		func(chunk string) error { return nil },
		func(call ToolCallRequest) error { return nil },
		func(chunk string) error { return nil },
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API returned status 400")
	assert.Contains(t, err.Error(), "invalid model")
}
