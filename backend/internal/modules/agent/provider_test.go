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

	// Call 1: reasoning + tool calls
	var reasoningChunks1 []string
	var toolCalls1 []ToolCallRequest
	var textChunks1 []string

	err := adapter.StreamChatReAct(
		context.Background(),
		nil,
		nil,
		func(chunk string) error {
			reasoningChunks1 = append(reasoningChunks1, chunk)
			return nil
		},
		func(call ToolCallRequest) error {
			toolCalls1 = append(toolCalls1, call)
			return nil
		},
		func(chunk string) error {
			textChunks1 = append(textChunks1, chunk)
			return nil
		},
	)
	require.NoError(t, err)

	require.Len(t, reasoningChunks1, 2)
	assert.Contains(t, reasoningChunks1[0], "查看当前简历内容")
	assert.Contains(t, reasoningChunks1[1], "简历内容已获取")
	require.Len(t, toolCalls1, 2)
	assert.Equal(t, "get_draft", toolCalls1[0].Name)
	assert.Equal(t, "apply_edits", toolCalls1[1].Name)
	assert.Empty(t, textChunks1, "first call should not have text")

	// Call 2: final text
	var reasoningChunks2 []string
	var textChunks2 []string

	err = adapter.StreamChatReAct(
		context.Background(),
		nil,
		nil,
		func(chunk string) error {
			reasoningChunks2 = append(reasoningChunks2, chunk)
			return nil
		},
		func(call ToolCallRequest) error { return nil },
		func(chunk string) error {
			textChunks2 = append(textChunks2, chunk)
			return nil
		},
	)
	require.NoError(t, err)

	assert.Empty(t, reasoningChunks2)
	require.Len(t, textChunks2, 1)
	assert.Equal(t, "我已经完成了简历的修改。", textChunks2[0])
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

func TestMessage_ToolResultFields(t *testing.T) {
	msg := Message{Role: "tool", Content: "result", ToolCallID: "call_123", Name: "get_draft"}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	assert.Equal(t, "tool", parsed["role"])
	assert.Equal(t, "call_123", parsed["tool_call_id"])
	assert.Equal(t, "get_draft", parsed["name"])
}

func TestMessage_ContentOmitEmpty(t *testing.T) {
	msg := Message{Role: "tool", ToolCallID: "call_1", Name: "apply_edits"}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	_, hasContent := parsed["content"]
	assert.False(t, hasContent)
}

func TestOpenAIAdapter_StreamChatReAct_PreservesToolFields(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model")

	messages := []Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant",
			ToolCalls: []ToolCallMessage{
				{ID: "call_1", Type: "function", Function: ToolCallFunction{Name: "get_draft", Arguments: "{}"}},
			},
		},
		{Role: "tool", Content: `{"html":"<p>test</p>"}`, ToolCallID: "call_1", Name: "get_draft"},
	}

	err := adapter.StreamChatReAct(
		context.Background(),
		messages,
		[]ToolDef{{Name: "get_draft", Description: "Get draft"}},
		func(chunk string) error { return nil },
		func(call ToolCallRequest) error { return nil },
		func(chunk string) error { return nil },
	)

	require.NoError(t, err)

	apiMsgs, ok := capturedBody["messages"].([]interface{})
	require.True(t, ok)
	require.Len(t, apiMsgs, 3)

	// Verify user message
	userMsg := apiMsgs[0].(map[string]interface{})
	assert.Equal(t, "user", userMsg["role"])
	assert.Equal(t, "hello", userMsg["content"])

	// Verify assistant message has tool_calls
	asstMsg := apiMsgs[1].(map[string]interface{})
	assert.Equal(t, "assistant", asstMsg["role"])
	asstToolCalls, ok := asstMsg["tool_calls"].([]interface{})
	require.True(t, ok, "assistant message should have tool_calls")
	require.Len(t, asstToolCalls, 1)
	tc := asstToolCalls[0].(map[string]interface{})
	assert.Equal(t, "call_1", tc["id"])
	fn := tc["function"].(map[string]interface{})
	assert.Equal(t, "get_draft", fn["name"])

	// Verify tool message has tool_call_id and name
	toolMsg := apiMsgs[2].(map[string]interface{})
	assert.Equal(t, "tool", toolMsg["role"])
	assert.Equal(t, "call_1", toolMsg["tool_call_id"], "tool message should have tool_call_id")
	assert.Equal(t, "get_draft", toolMsg["name"], "tool message should have name")
	assert.Equal(t, `{"html":"<p>test</p>"}`, toolMsg["content"])
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
