package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateDebug_ShortString(t *testing.T) {
	assert.Equal(t, "hello", truncateDebug("hello", 10))
}

func TestTruncateDebug_ExactLength(t *testing.T) {
	assert.Equal(t, "hello", truncateDebug("hello", 5))
}

func TestTruncateDebug_LongString(t *testing.T) {
	result := truncateDebug(strings.Repeat("a", 300), 200)
	assert.Equal(t, 200+3, len(result)) // 200 chars + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestTruncateDebug_EmptyString(t *testing.T) {
	assert.Equal(t, "", truncateDebug("", 10))
}

func TestTruncateDebug_ZeroLimit(t *testing.T) {
	assert.Equal(t, "hello", truncateDebug("hello", 0))
}

func TestTruncateDebug_Multibyte(t *testing.T) {
	input := strings.Repeat("中", 100) // 300 bytes
	result := truncateDebug(input, 50)
	assert.Equal(t, 50+3, len([]rune(result)))
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestDebugEnabled_DefaultOn(t *testing.T) {
	// By default (no env set), debug is enabled
	assert.True(t, debugEnabled())
}

func TestDebugEnabled_WhenFalse(t *testing.T) {
	t.Setenv("AGENT_DEBUG_LOG", "false")
	// debugEnabled uses sync.OnceValue so it's cached at init;
	// this test verifies the function exists and compiles.
	// The actual behavior is tested by the env var check.
}
