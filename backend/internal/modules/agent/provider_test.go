package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
