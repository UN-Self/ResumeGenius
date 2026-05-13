# apply_edits CSS 规范化匹配 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `apply_edits` 中增加 CSS 空白规范化回退匹配，解决 AI 模型生成的 CSS 格式与存储格式不一致导致的匹配失败问题。

**Architecture:** 在 `tool_executor.go` 中新增两个纯函数 `normalizeCSSWhitespace` 和 `findWithCSSNormalization`，修改 `applyEdits` 方法的匹配逻辑，精确匹配优先，失败后走规范化回退。

**Tech Stack:** Go, strings/regexp 标准库, testify

---

### Task 1: 写 normalizeCSSWhitespace 的失败测试

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 在 tool_executor_test.go 末尾追加测试**

```go
// ---------------------------------------------------------------------------
// CSS whitespace normalization tests
// ---------------------------------------------------------------------------

func TestNormalizeCSSWhitespace_MultiLineToSingleLine(t *testing.T) {
	input := ".resume-document .header {\n  color: rgb(255, 255, 255);\n  position: relative;\n}"
	expected := ".resume-document .header { color: rgb(255, 255, 255); position: relative; }"
	assert.Equal(t, expected, normalizeCSSWhitespace(input))
}

func TestNormalizeCSSWhitespace_AlreadySingleLine(t *testing.T) {
	input := ".name { font-size: 16px; }"
	expected := ".name { font-size: 16px; }"
	assert.Equal(t, expected, normalizeCSSWhitespace(input))
}

func TestNormalizeCSSWhitespace_TabsAndSpaces(t *testing.T) {
	input := ".name {\tfont-size: 16px;\t\tcolor: red;\n}"
	expected := ".name { font-size: 16px; color: red; }"
	assert.Equal(t, expected, normalizeCSSWhitespace(input))
}

func TestNormalizeCSSWhitespace_EmptyString(t *testing.T) {
	assert.Equal(t, "", normalizeCSSWhitespace(""))
}

func TestNormalizeCSSWhitespace_OnlyWhitespace(t *testing.T) {
	assert.Equal(t, "", normalizeCSSWhitespace("  \n\t  "))
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestNormalizeCSSWhitespace -v`
Expected: FAIL — `normalizeCSSWhitespace` 未定义

---

### Task 2: 实现 normalizeCSSWhitespace

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:1-17` (imports) 和文件末尾

- [ ] **Step 1: 在 tool_executor.go 的 import 中追加 "regexp" 和 "unicode"**

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	...
)
```

- [ ] **Step 2: 在文件末尾追加实现**

```go
var multiWhitespace = regexp.MustCompile(`\s+`)

func normalizeCSSWhitespace(s string) string {
	return strings.TrimSpace(multiWhitespace.ReplaceAllString(s, " "))
}
```

- [ ] **Step 3: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestNormalizeCSSWhitespace -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: add normalizeCSSWhitespace function with tests"
```

---

### Task 3: 写 findWithCSSNormalization 的失败测试

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 追加测试**

```go
func TestFindWithCSSNormalization_MultiLineVsSingleLine(t *testing.T) {
	html := `<style>
.resume-document .header {
  color: rgb(255, 255, 255);
  position: relative;
}
</style>`
	oldString := ".resume-document .header { color: rgb(255, 255, 255); position: relative; }"

	start, end := findWithCSSNormalization(html, oldString)
	assert.True(t, start >= 0, "should find match")
	assert.True(t, end > start, "end should be after start")
	// Verify the matched region in original HTML
	assert.Equal(t, "\n.resume-document .header {\n  color: rgb(255, 255, 255);\n  position: relative;\n}", html[start:end])
}

func TestFindWithCSSNormalization_ExactMatchStillWorks(t *testing.T) {
	html := `<style>.name { font-size: 16px; }</style>`
	oldString := ".name { font-size: 16px; }"

	start, end := findWithCSSNormalization(html, oldString)
	assert.True(t, start >= 0)
	assert.Equal(t, ".name { font-size: 16px; }", html[start:end])
}

func TestFindWithCSSNormalization_NoMatch(t *testing.T) {
	html := `<style>.name { font-size: 16px; }</style>`
	oldString := ".nonexistent { color: red; }"

	start, end := findWithCSSNormalization(html, oldString)
	assert.Equal(t, -1, start)
	assert.Equal(t, -1, end)
}

func TestFindWithCSSNormalization_EmptyOldString(t *testing.T) {
	html := `<style>.name { font-size: 16px; }</style>`
	start, end := findWithCSSNormalization(html, "")
	assert.Equal(t, -1, start)
	assert.Equal(t, -1, end)
}

func TestFindWithCSSNormalization_PreservesOriginalPosition(t *testing.T) {
	// Ensure the returned positions point to the correct region in original HTML
	html := `<div>prefix</div><style>
.class-a {
  margin: 0;
}
</style><div>suffix</div>`
	oldString := ".class-a { margin: 0; }"

	start, end := findWithCSSNormalization(html, oldString)
	require.True(t, start >= 0)
	matched := html[start:end]
	assert.Contains(t, matched, ".class-a")
	assert.Contains(t, matched, "margin: 0")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestFindWithCSSNormalization -v`
Expected: FAIL — `findWithCSSNormalization` 未定义

---

### Task 4: 实现 findWithCSSNormalization

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (在 normalizeCSSWhitespace 后追加)

- [ ] **Step 1: 追加实现**

```go
func findWithCSSNormalization(html, oldString string) (start, end int) {
	normalizedOld := normalizeCSSWhitespace(oldString)
	if normalizedOld == "" {
		return -1, -1
	}

	// Build normalized HTML with position mapping
	var normalized strings.Builder
	positionMap := make([]int, 0, len(html))
	lastWasSpace := false

	for i, r := range html {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				normalized.WriteRune(' ')
				positionMap = append(positionMap, i)
				lastWasSpace = true
			}
		} else {
			normalized.WriteRune(r)
			positionMap = append(positionMap, i)
			lastWasSpace = false
		}
	}

	normalizedHTML := normalized.String()
	idx := strings.Index(normalizedHTML, normalizedOld)
	if idx == -1 {
		return -1, -1
	}

	start = positionMap[idx]
	endPos := idx + len(normalizedOld) - 1
	if endPos < len(positionMap) {
		end = positionMap[endPos] + 1
	} else {
		end = len(html)
	}
	return start, end
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestFindWithCSSNormalization -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: add findWithCSSNormalization with position mapping"
```

---

### Task 5: 写 applyEdits CSS 规范化集成测试

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 追加集成测试**

```go
func TestApplyEdits_CSSNormalization_MultiLineCSS(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><head><style>
.resume-document .header {
  color: rgb(255, 255, 255);
  position: relative;
}
.resume-document .name {
  font-size: 18px;
  font-weight: 700;
}
</style></head><body><div class="header"><p class="name">John</p></div></body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Model generates single-line CSS as old_string
	result, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": ".resume-document .name { font-size: 18px; font-weight: 700; }",
				"new_string": ".resume-document .name { font-size: 16px; font-weight: 700; }",
				"description": "reduce name font size",
			},
		},
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(1), data["applied"])
	assert.Equal(t, float64(0), data["failed"])

	// Verify the replacement was applied correctly
	var updated models.Draft
	require.NoError(t, db.First(&updated, draft.ID).Error)
	assert.Contains(t, updated.HTMLContent, "font-size: 16px")
	assert.NotContains(t, updated.HTMLContent, "font-size: 18px")
	// Other CSS should be untouched
	assert.Contains(t, updated.HTMLContent, "color: rgb(255, 255, 255)")
}

func TestApplyEdits_CSSNormalization_FallbackBehavior(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body><h1>Hello</h1></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Both exact and normalized should fail for truly nonexistent content
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "nonexistent content",
				"new_string": "replacement",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "匹配失败")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_CSSNormalization -v`
Expected: FAIL — applyEdits 尚未使用 findWithCSSNormalization

---

### Task 6: 修改 applyEdits 匹配逻辑

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:480-489`

- [ ] **Step 1: 替换匹配逻辑**

将:
```go
			if !strings.Contains(html, op.OldString) {
				lastErr = fmt.Errorf("op #%d 匹配失败:\n%s", i+1, buildEditMatchError(op.OldString, html))
				resultData.Failed++
				continue
			}

			debugLog("tools", "操作 %d/%d: old=%s → new=%s", i+1, len(ops), truncateHTML(op.OldString), truncateHTML(op.NewString))
			html = strings.ReplaceAll(html, op.OldString, op.NewString)
```

替换为:
```go
			if strings.Contains(html, op.OldString) {
				// Exact match path
				debugLog("tools", "操作 %d/%d: old=%s → new=%s", i+1, len(ops), truncateHTML(op.OldString), truncateHTML(op.NewString))
				html = strings.ReplaceAll(html, op.OldString, op.NewString)
			} else if start, end := findWithCSSNormalization(html, op.OldString); start >= 0 {
				// CSS normalization fallback
				debugLog("tools", "操作 %d/%d: 通过 CSS 规范化匹配成功", i+1, len(ops))
				html = html[:start] + op.NewString + html[end:]
			} else {
				lastErr = fmt.Errorf("op #%d 匹配失败:\n%s", i+1, buildEditMatchError(op.OldString, html))
				resultData.Failed++
				continue
			}
```

- [ ] **Step 2: 运行新增测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_CSSNormalization -v`
Expected: PASS

- [ ] **Step 3: 运行全部测试确认无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: apply_edits CSS normalization fallback matching"
```
