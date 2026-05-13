package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateToolResult_GetDraftFull(t *testing.T) {
	longHTML := "<html><body>" + string(make([]byte, 6000)) + "</body></html>"
	result := TruncateToolResult("get_draft", longHTML, "full")
	assert.LessOrEqual(t, len([]rune(result)), 5200)
	assert.Contains(t, result, "已截断")
}

func TestTruncateToolResult_GetDraftStructure(t *testing.T) {
	longStructure := string(make([]byte, 3000))
	result := TruncateToolResult("get_draft", longStructure, "structure")
	assert.Equal(t, longStructure, result)
}

func TestTruncateToolResult_SearchAssets(t *testing.T) {
	longResult := string(make([]byte, 5000))
	result := TruncateToolResult("search_assets", longResult, "")
	assert.LessOrEqual(t, len([]rune(result)), 3200)
}

func TestTruncateToolResult_ApplyEdits(t *testing.T) {
	longResult := string(make([]byte, 2000))
	result := TruncateToolResult("apply_edits", longResult, "")
	assert.LessOrEqual(t, len([]rune(result)), 1200)
}

func TestTruncateToolResult_Error(t *testing.T) {
	longError := string(make([]byte, 3000))
	result := TruncateToolResult("error", longError, "")
	assert.LessOrEqual(t, len([]rune(result)), 2200)
}

func TestTruncateWithNotice(t *testing.T) {
	assert.Equal(t, "hello", truncateWithNotice("hello", 10, "..."))
	assert.Equal(t, "hel...", truncateWithNotice("hello world", 3, "..."))
	assert.Equal(t, "hello world", truncateWithNotice("hello world", 0, "..."))
	assert.Equal(t, "hello world", truncateWithNotice("hello world", -1, "..."))
}
