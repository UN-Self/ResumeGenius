# AI Agent 系统化改进 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 apply_edits 反复失败问题，引入结构化错误反馈、多模式 get_draft、分步执行、可观测性、上下文管理和 Agent 循环策略改进

**Architecture:** 在现有 agent 模块基础上增强，不改变外部 API 接口。核心改动集中在 `tool_executor.go`（工具执行层）和 `service.go`（ReAct 循环），新增 `htmlutil.go`（HTML 解析工具）和 `context_budget.go`（token 预算管理）。遵循 TDD，每个 Task 先写测试再实现。

**Tech Stack:** Go, Gin, GORM, goquery, log/slog, PostgreSQL

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `backend/internal/modules/agent/htmlutil.go` | HTML 结构解析、文本节点搜索、相似度匹配、上下文片段截取 |
| `backend/internal/modules/agent/context_budget.go` | Token 预算实时监控、差异化截断策略 |
| `backend/internal/shared/middleware/requestid.go` | X-Request-ID 中间件，注入 requestID 到 context |

### Modified Files
| File | Changes |
|------|---------|
| `backend/internal/modules/agent/tool_executor.go` | apply_edits 结构化错误反馈 + 分步执行、get_draft 多模式、search_assets 截断 |
| `backend/internal/modules/agent/service.go` | slog 替换 debugLog、token 预算集成、压缩改进、步骤漏调检测、API 重试、降级放行 |
| `backend/internal/modules/agent/provider.go` | LLMProvider 接口提取、指数退避重试 |
| `backend/internal/modules/agent/handler.go` | requestID 注入、slog 替换、错误码映射更新 |
| `backend/internal/modules/agent/routes.go` | 接线新的中间件和依赖 |
| `backend/internal/modules/agent/debug.go` | 标记废弃（保留兼容，逐步迁移） |
| `backend/internal/shared/middleware/middleware.go` | 注册 RequestID 中间件 |

### Test Files
| File | Covers |
|------|--------|
| `backend/internal/modules/agent/tool_executor_test.go` | 结构化错误反馈、分步执行、get_draft 多模式 |
| `backend/internal/modules/agent/service_test.go` | token 预算、压缩改进、步骤漏调、降级放行 |
| `backend/internal/modules/agent/provider_test.go` | 指数退避重试 |
| `backend/internal/modules/agent/htmlutil_test.go` | HTML 解析工具函数 |
| `backend/internal/modules/agent/context_budget_test.go` | token 预算和截断策略 |
| `backend/internal/shared/middleware/requestid_test.go` | RequestID 中间件 |

---

## Task 1: HTML 解析工具函数（htmlutil.go）

**Files:**
- Create: `backend/internal/modules/agent/htmlutil.go`
- Create: `backend/internal/modules/agent/htmlutil_test.go`

- [ ] **Step 1: 写失败的测试 — findBestMatchNode**

```go
// htmlutil_test.go
package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBestMatchNode_Found(t *testing.T) {
	html := `<html><body>
		<div class="experience">
			<h3>Google</h3>
			<p>Senior Engineer 2020-2024</p>
		</div>
		<div class="education">
			<h3>MIT</h3>
			<p>Computer Science</p>
		</div>
	</body></html>`

	node, score := FindBestMatchNode(html, "Senior Engineer")
	require.NotNil(t, node)
	assert.Greater(t, score, 0.0)
	assert.Contains(t, node.OuterHTML, "Senior Engineer")
}

func TestFindBestMatchNode_NotFound(t *testing.T) {
	html := `<html><body><p>Hello World</p></body></html>`
	node, score := FindBestMatchNode(html, "xyznonexistent")
	assert.Nil(t, node)
	assert.Equal(t, 0.0, score)
}

func TestFindBestMatchNode_LongestCommonSubstring(t *testing.T) {
	html := `<html><body><p>Senior Software Engineer at Google</p></body></html>`
	// 搜索 "Sr. Software Engineer" — 不完全匹配但有公共子串
	node, score := FindBestMatchNode(html, "Sr. Software Engineer")
	require.NotNil(t, node)
	assert.Greater(t, score, 0.0)
	assert.Contains(t, node.OuterHTML, "Senior Software Engineer")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestFindBestMatchNode -v 2>&1 | head -20`
Expected: 编译失败，`FindBestMatchNode` 未定义

- [ ] **Step 3: 实现 FindBestMatchNode**

```go
// htmlutil.go
package agent

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// MatchResult represents a matched HTML node with context.
type MatchResult struct {
	OuterHTML  string  // 匹配节点的 OuterHTML
	Score      float64 // 相似度分数 (0-1)
	ParentTag  string  // 父元素标签名
	TagName    string  // 匹配节点标签名
}

// FindBestMatchNode searches all text nodes in html for the one most similar to query.
// Returns nil if no node contains any common substring.
func FindBestMatchNode(html, query string) (*MatchResult, float64) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, 0
	}

	var bestResult *MatchResult
	bestScore := 0.0

	doc.Find("*").Each(func(_ int, sel *goquery.Selection) {
		text := sel.Text()
		if len(strings.TrimSpace(text)) == 0 {
			return
		}

		lcs := longestCommonSubstring(strings.ToLower(text), strings.ToLower(query))
		if len(lcs) == 0 {
			return
		}

		// Score = length of LCS / length of query
		score := float64(len(lcs)) / float64(len(query))
		if score <= bestScore {
			return
		}

		outerHTML, _ := sel.Html()
		tagName := goquery.NodeName(sel)
		parentTag := ""
		if parent := sel.Parent(); parent != nil {
			parentTag = goquery.NodeName(parent)
		}

		bestScore = score
		bestResult = &MatchResult{
			OuterHTML: outerHTML,
			Score:     score,
			ParentTag: parentTag,
			TagName:   tagName,
		}
	})

	return bestResult, bestScore
}

// longestCommonSubstring returns the longest common substring of a and b.
func longestCommonSubstring(a, b string) string {
	if len(a) == 0 || len(b) == 0 {
		return ""
	}

	// Use a single-row DP to save memory
	prev := make([]int, len(b)+1)
	maxLen := 0
	endIdx := 0

	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
				if curr[j] > maxLen {
					maxLen = curr[j]
					endIdx = i
				}
			}
		}
		prev = curr
	}

	if maxLen == 0 {
		return ""
	}
	return a[endIdx-maxLen : endIdx]
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestFindBestMatchNode -v`
Expected: PASS — 3 个测试全部通过

- [ ] **Step 5: 写失败的测试 — TruncateAroundMatch**

```go
func TestTruncateAroundMatch(t *testing.T) {
	html := `<html><body>
		<div class="header"><h1>Resume</h1></div>
		<div class="experience">
			<div class="job">
				<h3>Google</h3>
				<p>Senior Engineer from 2020 to 2024, worked on search infrastructure</p>
			</div>
		</div>
		<div class="education"><h3>MIT</h3></div>
	</body></html>`

	snippet := TruncateAroundMatch(html, "Senior Engineer", 100)
	assert.Contains(t, snippet, "Senior Engineer")
	assert.LessOrEqual(t, len(snippet), 300) // 上下文 ±100 + 匹配内容
}

func TestTruncateAroundMatch_MultipleOccurrences(t *testing.T) {
	html := `<html><body>
		<p>First occurrence of test</p>
		<p>Second occurrence of test here</p>
	</body></html>`

	snippet := TruncateAroundMatch(html, "test", 50)
	assert.Contains(t, snippet, "test")
}
```

- [ ] **Step 6: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestTruncateAroundMatch -v 2>&1 | head -20`
Expected: 编译失败，`TruncateAroundMatch` 未定义

- [ ] **Step 7: 实现 TruncateAroundMatch**

在 `htmlutil.go` 中追加：

```go
// TruncateAroundMatch finds the first occurrence of query in html and returns
// a snippet with contextRadius characters of context around it.
func TruncateAroundMatch(html, query string, contextRadius int) string {
	idx := strings.Index(strings.ToLower(html), strings.ToLower(query))
	if idx < 0 {
		// 找不到精确匹配，尝试 LCS 方式
		result, _ := FindBestMatchNode(html, query)
		if result != nil {
			return truncateString(result.OuterHTML, contextRadius*2+len(query))
		}
		return truncateString(html, contextRadius*2)
	}

	start := idx - contextRadius
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + contextRadius
	if end > len(html) {
		end = len(html)
	}

	snippet := html[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(html) {
		snippet = snippet + "..."
	}
	return snippet
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
```

- [ ] **Step 8: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestTruncateAroundMatch -v`
Expected: PASS

- [ ] **Step 9: 写失败的测试 — BuildStructureOverview**

```go
func TestBuildStructureOverview(t *testing.T) {
	html := `<html>
	<head><style>body{font-size:14px} .header{color:red}</style></head>
	<body>
		<div class="header"><h1>John Doe</h1><p>Engineer</p></div>
		<div class="section experience">
			<h2>Experience</h2>
			<div class="item"><h3>Google</h3><p>Senior Engineer</p></div>
		</div>
		<div class="section education">
			<h2>Education</h2>
			<div class="item"><h3>MIT</h3></div>
		</div>
	</body></html>`

	overview := BuildStructureOverview(html)
	assert.Contains(t, overview, "<html>")
	assert.Contains(t, overview, "<body>")
	assert.Contains(t, overview, "header")
	assert.Contains(t, overview, "experience")
	assert.NotContains(t, overview, "John Doe")   // 不包含文本内容
	assert.NotContains(t, overview, "Senior Engineer")
}

func TestBuildStructureOverview_EmptyHTML(t *testing.T) {
	overview := BuildStructureOverview("")
	assert.Equal(t, "", overview)
}

func TestBuildStructureOverview_StyleTruncated(t *testing.T) {
	longCSS := "<style>" + strings.Repeat("body{color:red}", 100) + "</style>"
	html := "<html><head>" + longCSS + "</head><body><div>content</div></body></html>"

	overview := BuildStructureOverview(html)
	assert.Contains(t, overview, "<style>")
	assert.NotContains(t, overview, strings.Repeat("body{color:red}", 100))
}
```

- [ ] **Step 10: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestBuildStructureOverview -v 2>&1 | head -20`
Expected: 编译失败

- [ ] **Step 11: 实现 BuildStructureOverview**

在 `htmlutil.go` 中追加：

```go
// BuildStructureOverview returns a compact HTML tag tree showing structure
// without text content. Style tags are truncated to 200 chars.
func BuildStructureOverview(html string) string {
	if len(html) == 0 {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return truncateString(html, 500)
	}

	var b strings.Builder
	buildNodeOverview(doc.Find("html").First(), &b, 0)
	return b.String()
}

func buildNodeOverview(sel *goquery.Selection, b *strings.Builder, depth int) {
	if depth > 6 {
		return
	}

	sel.Contents().Each(func(_ int, child *goquery.Selection) {
		if child.Is("text") || child.Is("comment") {
			return // 跳过文本节点和注释
		}

		tag := goquery.NodeName(child)

		// 获取 class 属性
		class, _ := child.Attr("class")
		id, _ := child.Attr("id")

		// 构建标签描述
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteString("<" + tag)
		if id != "" {
			b.WriteString(` id="` + id + `"`)
		}
		if class != "" {
			b.WriteString(` class="` + class + `"`)
		}
		b.WriteString(">")

		// style 标签内容截断
		if tag == "style" {
			styleText := child.Text()
			if len(styleText) > 200 {
				b.WriteString(truncateString(styleText, 200))
			} else {
				b.WriteString(styleText)
			}
			b.WriteString("</style>")
			return
		}

		b.WriteString("\n")

		// 递归处理子元素
		buildNodeOverview(child, b, depth+1)
	})
}
```

- [ ] **Step 12: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestBuildStructureOverview -v`
Expected: PASS

- [ ] **Step 13: 提交**

```bash
git add backend/internal/modules/agent/htmlutil.go backend/internal/modules/agent/htmlutil_test.go
git commit -m "feat: add HTML parsing utilities for structured error feedback and multi-mode get_draft"
```

---

## Task 2: apply_edits 结构化错误反馈

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (lines 354-492)
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

依赖: Task 1

- [ ] **Step 1: 写失败的测试 — 结构化错误反馈**

在 `tool_executor_test.go` 中追加：

```go
func TestApplyEdits_StructuredErrorFeedback(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{
		ProjectID:   1,
		HTMLContent: `<html><body><div class="experience"><h3>Google</h3><p>Senior Engineer</p></div></body></html>`,
	}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)

	ctx := WithDraftID(context.Background(), draft.ID)

	// 故意搜索不存在但有相似内容的字符串
	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Sr. Engineer at Google",
				"new_string": "Staff Engineer at Google",
			},
		},
	}

	_, err := executor.Execute(ctx, "apply_edits", params)
	require.Error(t, err)

	errMsg := err.Error()
	// 错误消息应包含结构化反馈
	assert.Contains(t, errMsg, "未找到匹配")
	assert.Contains(t, errMsg, "搜索内容")
	assert.Contains(t, errMsg, "建议")
}

func TestApplyEdits_ErrorIncludesNearbyHTML(t *testing.T) {
	db := SetupTestDB(t)

	longHTML := `<html><body>` + strings.Repeat("<p>filler content</p>", 50) +
		`<div class="target"><span>Unique Target Text Here</span></div>` +
		strings.Repeat("<p>more filler</p>", 50) + `</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: longHTML}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Unique Target Txt Here", // 故意拼错
				"new_string": "New Text",
			},
		},
	}

	_, err := executor.Execute(ctx, "apply_edits", params)
	require.Error(t, err)

	// 错误消息应包含附近 HTML 片段
	assert.Contains(t, err.Error(), "附近")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_StructuredErrorFeedback -v 2>&1 | tail -20`
Expected: 当前错误消息只有简单的 "未找到匹配"，不包含结构化信息，测试 FAIL

- [ ] **Step 3: 实现结构化错误反馈**

修改 `tool_executor.go` 中的 `applyEdits` 方法。将 dry-run 验证部分（约 lines 429-436）替换为结构化错误反馈：

```go
// 在 tool_executor.go 中新增辅助函数：

// buildEditMatchError constructs a structured error when old_string is not found in HTML.
func buildEditMatchError(oldString, html string) string {
	var b strings.Builder
	b.WriteString("未找到匹配内容\n\n")

	b.WriteString("搜索内容: ")
	b.WriteString(truncateString(oldString, 100))
	b.WriteString("\n\n")

	// 尝试找到最相似的文本节点
	matchResult, score := FindBestMatchNode(html, oldString)

	// 判断失败原因
	b.WriteString("可能原因: ")
	if matchResult != nil && score > 0.3 {
		b.WriteString("缩进/换行不一致或片段跨越标签边界")
	} else if matchResult != nil && score > 0.1 {
		b.WriteString("HTML 已被修改，内容部分匹配")
	} else {
		b.WriteString("HTML 中不存在相似内容，可能已被完全替换")
	}
	b.WriteString("\n\n")

	// 提供附近 HTML 片段
	b.WriteString("附近 HTML 片段:\n")
	if matchResult != nil {
		snippet := TruncateAroundMatch(html, oldString, 200)
		b.WriteString(snippet)
	} else {
		// 返回 draft 的结构概览
		overview := BuildStructureOverview(html)
		if len(overview) > 0 {
			b.WriteString("当前 HTML 结构:\n")
			b.WriteString(truncateString(overview, 500))
		} else {
			b.WriteString(truncateString(html, 500))
		}
	}
	b.WriteString("\n\n")

	b.WriteString("建议: 使用更短的唯一片段重新搜索，确保文本精确匹配（包括空格和换行）")

	return b.String()
}
```

然后修改 `applyEdits` 中的 dry-run 验证逻辑：

```go
// 原来的 dry-run 验证（约 line 429-436）：
// for _, op := range ops {
//     if !strings.Contains(html, op.OldString) {
//         return "", fmt.Errorf("未找到匹配...")
//     }
// }

// 替换为：
for i, op := range ops {
    if !strings.Contains(html, op.OldString) {
        errMsg := buildEditMatchError(op.OldString, html)
        debugLogFull("apply_edits", "match_failed_ops", string(opsJSON))
        return "", fmt.Errorf("op #%d 匹配失败:\n%s", i+1, errMsg)
    }
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestApplyEdits_StructuredErrorFeedback|TestApplyEdits_ErrorIncludesNearbyHTML" -v`
Expected: PASS

- [ ] **Step 5: 运行全部 tool_executor 测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits -v`
Expected: 所有 apply_edits 相关测试通过

- [ ] **Step 6: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: structured error feedback for apply_edits with nearby HTML context"
```

---

## Task 3: get_draft 多模式增强

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (lines 321-348, 82-133)
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

依赖: Task 1

- [ ] **Step 1: 写失败的测试 — mode=structure**

在 `tool_executor_test.go` 中追加：

```go
func TestGetDraft_ModeStructure(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html>
	<head><style>body{font-size:14px}.header{color:red}</style></head>
	<body>
		<div class="header"><h1>John Doe</h1><p>Engineer</p></div>
		<div class="experience"><h2>Experience</h2><div class="item"><h3>Google</h3></div></div>
		<div class="education"><h2>Education</h2><div class="item"><h3>MIT</h3></div></div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode": "structure",
	})
	require.NoError(t, err)

	assert.Contains(t, result, "<html>")
	assert.Contains(t, result, "header")
	assert.Contains(t, result, "experience")
	assert.NotContains(t, result, "John Doe")
	assert.NotContains(t, result, "Google")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestGetDraft_ModeStructure -v 2>&1 | tail -10`
Expected: 当前 get_draft 不支持 mode 参数，测试 FAIL

- [ ] **Step 3: 写失败的测试 — mode=search**

```go
func TestGetDraft_ModeSearch(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<div class="experience">
			<h3>Google</h3>
			<p>Senior Engineer 2020-2024, worked on search infrastructure and ML pipelines</p>
		</div>
		<div class="education">
			<h3>MIT</h3>
			<p>Computer Science, GPA 3.9</p>
		</div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":  "search",
		"query": "Engineer",
	})
	require.NoError(t, err)

	assert.Contains(t, result, "Senior Engineer")
	assert.Contains(t, result, "Google")
}

func TestGetDraft_ModeSearch_MultipleResults(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<p>First section with keyword</p>
		<p>Second section with keyword</p>
		<p>Third section with keyword</p>
		<p>Fourth section with keyword</p>
		<p>Fifth section with keyword</p>
		<p>Sixth section with keyword</p>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":  "search",
		"query": "keyword",
	})
	require.NoError(t, err)

	// 最多返回 5 条匹配
	assert.Contains(t, result, "keyword")
}
```

- [ ] **Step 4: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestGetDraft_ModeSearch" -v 2>&1 | tail -10`
Expected: FAIL — 不支持 mode=search

- [ ] **Step 5: 写失败的测试 — mode=section（已有 selector 支持，测试向后兼容）**

```go
func TestGetDraft_ModeSection_BackwardCompatible(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<div class="experience"><h3>Google</h3><p>Engineer</p></div>
		<div class="education"><h3>MIT</h3></div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// 旧的 selector 参数仍然工作
	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"selector": ".experience",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Google")
	assert.NotContains(t, result, "MIT")

	// 新的 mode=section + selector 也工作
	result2, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":     "section",
		"selector": ".education",
	})
	require.NoError(t, err)
	assert.Contains(t, result2, "MIT")
	assert.NotContains(t, result2, "Google")
}

func TestGetDraft_ModeFull(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body><h1>Test</h1></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode": "full",
	})
	require.NoError(t, err)
	assert.Equal(t, html, result)
}
```

- [ ] **Step 6: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestGetDraft_Mode" -v 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 7: 更新 get_draft 工具定义（增加 mode 和 query 参数）**

修改 `tool_executor.go` 中 `Tools()` 方法里的 `get_draft` 定义：

```go
// 原来的 get_draft 定义（约 lines 85-105）替换为：
{
    Name:        "get_draft",
    Description: "获取简历 HTML 内容。支持 4 种模式：structure（结构概览，不含文本）、section（按 CSS selector 获取指定区域）、search（搜索包含关键词的片段）、full（完整 HTML）。首次调用请使用 structure 模式了解整体结构。",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "mode": map[string]interface{}{
                "type":        "string",
                "description": "查询模式：structure（结构概览）、section（指定区域）、search（关键词搜索）、full（完整内容）。默认 full。",
                "enum":        []string{"structure", "section", "search", "full"},
            },
            "selector": map[string]interface{}{
                "type":        "string",
                "description": "CSS 选择器，mode=section 时使用。例如 .experience、#education",
            },
            "query": map[string]interface{}{
                "type":        "string",
                "description": "搜索关键词，mode=search 时使用。返回包含关键词的 HTML 片段（±200 字符上下文）。",
            },
        },
    },
},
```

- [ ] **Step 8: 实现 getDraft 多模式分发**

修改 `getDraft` 方法：

```go
func (e *AgentToolExecutor) getDraft(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok || draftID == 0 {
		return "", fmt.Errorf("missing draft_id in context")
	}

	var draft models.Draft
	if err := e.db.WithContext(ctx).Select("id", "html_content").First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("failed to get draft: %w", err)
	}

	// 解析 mode 参数，默认 "full"（向后兼容）
	mode := "full"
	if m, ok := params["mode"].(string); ok && m != "" {
		mode = m
	}

	// 向后兼容：有 selector 但没有 mode 时，当作 section
	selector, _ := params["selector"].(string)
	if selector != "" && mode == "full" {
		mode = "section"
	}

	switch mode {
	case "structure":
		return BuildStructureOverview(draft.HTMLContent), nil

	case "section":
		if selector == "" {
			return "", fmt.Errorf("mode=section 需要 selector 参数")
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(draft.HTMLContent))
		if err != nil {
			return "", fmt.Errorf("failed to parse HTML: %w", err)
		}
		selection := doc.Find(selector)
		if selection.Length() == 0 {
			return "", fmt.Errorf("selector %q 未匹配到任何元素", selector)
		}
		html, err := selection.Html()
		if err != nil {
			return "", fmt.Errorf("failed to extract HTML: %w", err)
		}
		return html, nil

	case "search":
		query, _ := params["query"].(string)
		if query == "" {
			return "", fmt.Errorf("mode=search 需要 query 参数")
		}
		return searchInDraft(draft.HTMLContent, query), nil

	case "full":
		// 原有逻辑：截断到 5000 字符
		if len([]rune(draft.HTMLContent)) > 5000 {
			return string([]rune(draft.HTMLContent)[:5000]) + "\n...(已截断，请使用 mode=section 或 mode=search 获取具体内容)", nil
		}
		return draft.HTMLContent, nil

	default:
		return "", fmt.Errorf("未知 mode: %s，支持 structure/section/search/full", mode)
	}
}
```

- [ ] **Step 9: 实现 searchInDraft 辅助函数**

在 `htmlutil.go` 中追加：

```go
// searchInDraft finds up to 5 snippets in html that contain query,
// each with ±200 characters of context.
func searchInDraft(html, query string) string {
	lowerHTML := strings.ToLower(html)
	lowerQuery := strings.ToLower(query)

	var results []string
	searchFrom := 0
	maxResults := 5
	radius := 200

	for len(results) < maxResults {
		idx := strings.Index(lowerHTML[searchFrom:], lowerQuery)
		if idx < 0 {
			break
		}
		absIdx := searchFrom + idx

		start := absIdx - radius
		if start < 0 {
			start = 0
		}
		end := absIdx + len(query) + radius
		if end > len(html) {
			end = len(html)
		}

		snippet := html[start:end]
		if start > 0 {
			snippet = "..." + snippet
		}
		if end < len(html) {
			snippet = snippet + "..."
		}

		results = append(results, snippet)
		searchFrom = absIdx + len(query)
	}

	if len(results) == 0 {
		return fmt.Sprintf("未找到包含 %q 的内容", query)
	}

	return fmt.Sprintf("找到 %d 处匹配:\n\n%s", len(results), strings.Join(results, "\n\n---\n\n"))
}
```

- [ ] **Step 10: 运行全部 get_draft 测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestGetDraft" -v`
Expected: PASS — 所有多模式测试通过

- [ ] **Step 11: 运行全部 tool_executor 测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -v 2>&1 | tail -30`
Expected: 所有测试通过

- [ ] **Step 12: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go backend/internal/modules/agent/htmlutil.go
git commit -m "feat: get_draft multi-mode support (structure/section/search/full)"
```

---

## Task 4: apply_edits 分步执行

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (applyEdits 方法)
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

依赖: Task 2

- [ ] **Step 1: 写失败的测试 — 分步执行，成功 ops 保留**

```go
func TestApplyEdits_StepByStep_PartialSuccess(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body>
		<p>Apple</p>
		<p>Banana</p>
		<p>Cherry</p>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// 第一个 op 成功，第二个 op 失败
	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Apple",
				"new_string": "Apricot",
			},
			map[string]interface{}{
				"old_string": "NonExistent",
				"new_string": "Something",
			},
		},
	}

	_, err := executor.Execute(ctx, "apply_edits", params)
	require.Error(t, err)

	// 错误消息应告知哪个 op 失败
	assert.Contains(t, err.Error(), "op #2")

	// 第一个 op 应该已经成功应用
	var updatedDraft models.Draft
	require.NoError(t, db.Where("id = ?", draft.ID).First(&updatedDraft).Error)
	assert.Contains(t, updatedDraft.HTMLContent, "Apricot")
	assert.NotContains(t, updatedDraft.HTMLContent, "Apple")

	// 第二个 op 未应用
	assert.Contains(t, updatedDraft.HTMLContent, "Cherry")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_StepByStep_PartialSuccess -v 2>&1 | tail -10`
Expected: 当前实现是原子的（全部成功或全部回滚），测试 FAIL

- [ ] **Step 3: 写失败的测试 — 分步执行结果包含详细信息**

```go
func TestApplyEdits_StepByStep_ResultDetails(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html><body><p>Hello World</p></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	params := map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Hello",
				"new_string": "Hi",
			},
			map[string]interface{}{
				"old_string": "World",
				"new_string": "Earth",
			},
		},
	}

	result, err := executor.Execute(ctx, "apply_edits", params)
	require.NoError(t, err)

	var parsed struct {
		Applied     int `json:"applied"`
		NewSequence int `json:"new_sequence"`
		Failed      int `json:"failed"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, 2, parsed.Applied)
	assert.Equal(t, 0, parsed.Failed)
}
```

- [ ] **Step 4: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_StepByStep_ResultDetails -v 2>&1 | tail -10`
Expected: FAIL — 当前结果没有 `failed` 字段

- [ ] **Step 5: 重构 applyEdits 为分步执行**

修改 `applyEdits` 中的事务逻辑。将 dry-run + apply 两阶段改为单次遍历逐 op 执行：

```go
func (e *AgentToolExecutor) applyEdits(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok || draftID == 0 {
		return "", fmt.Errorf("missing draft_id in context")
	}

	// 解析 ops（保持不变）
	opsRaw, ok := params["ops"].([]interface{})
	if !ok || len(opsRaw) == 0 {
		return "", fmt.Errorf("ops 参数缺失或为空")
	}

	type editOp struct {
		OldString   string `json:"old_string"`
		NewString   string `json:"new_string"`
		Description string `json:"description"`
	}

	ops := make([]editOp, 0, len(opsRaw))
	for i, raw := range opsRaw {
		opMap, ok := raw.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("ops[%d] 格式错误", i)
		}
		oldStr, _ := opMap["old_string"].(string)
		newStr, _ := opMap["new_string"].(string)
		if oldStr == "" {
			return "", fmt.Errorf("ops[%d].old_string 不能为空", i)
		}
		desc, _ := opMap["description"].(string)
		ops = append(ops, editOp{OldString: oldStr, NewString: newStr, Description: desc})
	}

	opsJSON, _ := json.Marshal(ops)
	debugLog("apply_edits", "draftID=%d ops=%s", draftID, truncateParams(string(opsJSON)))

	// 分步执行（非原子）
	var result struct {
		Applied     int `json:"applied"`
		Failed      int `json:"failed"`
		NewSequence int `json:"new_sequence"`
	}
	var lastErr error

	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var draft models.Draft
		if err := tx.Select("id", "html_content", "current_edit_sequence").First(&draft, draftID).Error; err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		// 确保 base snapshot 存在
		var baseCount int64
		tx.Model(&models.DraftEdit{}).Where("draft_id = ? AND sequence = 0", draftID).Count(&baseCount)
		if baseCount == 0 {
			if err := tx.Create(&models.DraftEdit{
				DraftID:      draftID,
				Sequence:     0,
				OpType:       "base_snapshot",
				HtmlSnapshot: draft.HTMLContent,
			}).Error; err != nil {
				return fmt.Errorf("failed to create base snapshot: %w", err)
			}
		}

		html := draft.HTMLContent
		seq := draft.CurrentEditSequence

		for i, op := range ops {
			if !strings.Contains(html, op.OldString) {
				// 当前 op 失败，记录错误，继续处理剩余 op
				lastErr = fmt.Errorf("op #%d 匹配失败:\n%s", i+1, buildEditMatchError(op.OldString, html))
				result.Failed++
				continue
			}

			// 应用 op
			html = strings.ReplaceAll(html, op.OldString, op.NewString)
			seq++

			// 记录 edit
			if err := tx.Create(&models.DraftEdit{
				DraftID:      draftID,
				Sequence:     seq,
				OpType:       "replace",
				OldString:    op.OldString,
				NewString:    op.NewString,
				Description:  op.Description,
				HtmlSnapshot: html,
			}).Error; err != nil {
				return fmt.Errorf("failed to record edit #%d: %w", i+1, err)
			}

			result.Applied++
		}

		// 更新 draft（即使有失败的 op，成功的也要保留）
		if result.Applied > 0 {
			if err := tx.Model(&draft).Updates(map[string]interface{}{
				"html_content":          html,
				"current_edit_sequence": seq,
			}).Error; err != nil {
				return fmt.Errorf("failed to update draft: %w", err)
			}
		}

		result.NewSequence = seq
		return nil
	})

	if err != nil {
		debugLogFull("apply_edits", "failed_ops", string(opsJSON))
		return "", err
	}

	if lastErr != nil {
		// 部分成功：返回结果 + 最后一个错误
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), fmt.Errorf("部分成功 (%d/%d applied, %d failed): %w",
			result.Applied, len(ops), result.Failed, lastErr)
	}

	debugLog("apply_edits", "applied=%d new_sequence=%d", result.Applied, result.NewSequence)
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestApplyEdits_StepByStep" -v`
Expected: PASS

- [ ] **Step 7: 运行全部 apply_edits 测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestApplyEdits" -v`
Expected: 所有测试通过

- [ ] **Step 8: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: apply_edits step-by-step execution, successful ops preserved on partial failure"
```

---

## Task 5: slog 结构化日志 + requestID 链路追踪

**Files:**
- Create: `backend/internal/shared/middleware/requestid.go`
- Create: `backend/internal/shared/middleware/requestid_test.go`
- Modify: `backend/internal/modules/agent/debug.go` (保留兼容，新增 slog 函数)
- Modify: `backend/internal/modules/agent/service.go` (替换部分 debugLog 为 slog)
- Modify: `backend/internal/modules/agent/handler.go` (注入 requestID)
- Modify: `backend/internal/modules/agent/tool_executor.go` (替换部分 debugLog 为 slog)
- Modify: `backend/internal/shared/middleware/middleware.go` (注册 RequestID)

- [ ] **Step 1: 写失败的测试 — RequestID 中间件**

```go
// requestid_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestID_Generated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		rid := RequestIDFromContext(c)
		c.JSON(200, gin.H{"request_id": rid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestID_PassedThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		rid := RequestIDFromContext(c)
		c.JSON(200, gin.H{"request_id": rid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "existing-id-123", w.Header().Get("X-Request-ID"))
}

func TestRequestID_LoggerInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		logger := LoggerFromContext(c)
		assert.NotNil(t, logger)
		c.JSON(200, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/shared/middleware/ -run TestRequestID -v 2>&1 | tail -10`
Expected: 编译失败，`RequestID` 未定义

- [ ] **Step 3: 实现 RequestID 中间件**

```go
// requestid.go
package middleware

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ContextRequestID = "request_id"
	ContextLogger    = "logger"
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// RequestID middleware generates or passes through X-Request-ID,
// injects it into context, and attaches a structured logger with requestID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = generateRequestID()
		}

		c.Header("X-Request-ID", rid)
		c.Set(ContextRequestID, rid)

		// 创建带 requestID 的 slog logger
		logger := slog.With("requestID", rid)
		c.Set(ContextLogger, logger)

		// 注入到 context.Context（供下游 service 层使用）
		ctx := context.WithValue(c.Request.Context(), requestIDKey{}, rid)
		ctx = context.WithValue(ctx, loggerKey{}, logger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

type requestIDKey struct{}
type loggerKey struct{}

// RequestIDFromContext extracts requestID from gin.Context.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(ContextRequestID); ok {
		if rid, ok := v.(string); ok {
			return rid
		}
	}
	return ""
}

// LoggerFromContext extracts the slog.Logger from gin.Context.
func LoggerFromContext(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(ContextLogger); ok {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

// LoggerFromStdContext extracts the slog.Logger from context.Context.
func LoggerFromStdContext(ctx context.Context) *slog.Logger {
	if v := ctx.Value(loggerKey{}); v != nil {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

// RequestIDFromStdContext extracts requestID from context.Context.
func RequestIDFromStdContext(ctx context.Context) string {
	if v := ctx.Value(requestIDKey{}); v != nil {
		if rid, ok := v.(string); ok {
			return rid
		}
	}
	return ""
}

func generateRequestID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/shared/middleware/ -run TestRequestID -v`
Expected: PASS

- [ ] **Step 5: 在 middleware.go 中注册 RequestID**

在 `middleware.go` 顶部添加 RequestID 的导出（如果 middleware.go 没有统一导出，可以跳过此步，直接在 routes.go 中使用）。

- [ ] **Step 6: 在 handler.go 的 Chat handler 中注入 logger**

修改 `handler.go` 中的 `Chat` 方法，使用带 requestID 的 logger：

```go
func (h *Handler) Chat(c *gin.Context) {
    // ... 现有的参数解析代码 ...

    // 获取带 requestID 的 logger
    logger := middleware.LoggerFromContext(c)

    // 将 logger 注入到 context 中，供 service 层使用
    ctx := c.Request.Context()
    logger.InfoContext(ctx, "chat started",
        "sessionID", sessionID,
        "messageLen", len(message),
    )

    // ... 现有的 SSE 代码 ...
}
```

- [ ] **Step 7: 在 service.go 中替换关键 debugLog 为 slog**

修改 `StreamChatReAct` 中的关键日志点：

```go
// 在 StreamChatReAct 方法开始处获取 logger
logger := middleware.LoggerFromStdContext(ctx)

// 替换 debugLog("chat", "sessionID=%d ...") 为：
logger.Info("react_loop_start",
    "sessionID", sessionID,
    "messageLen", len(userMessage),
)

// 替换工具执行日志：
logger.Info("tool_executed",
    "tool", call.Name,
    "duration", elapsed,
    "resultLen", len(result),
    "success", err == nil,
)

// 替换循环结束日志：
logger.Info("react_loop_done",
    "iterations", iteration,
    "hasText", len(text) > 0,
    "toolCallCount", totalToolCalls,
)
```

- [ ] **Step 8: 在 tool_executor.go 中替换 debugLog 为 slog**

```go
// 在 applyEdits 中：
logger := middleware.LoggerFromStdContext(ctx)
logger.Info("apply_edits_start", "draftID", draftID, "opsCount", len(ops))
// ... 执行后 ...
logger.Info("apply_edits_done", "applied", result.Applied, "failed", result.Failed, "duration", elapsed)

// 在 getDraft 中：
logger.Info("get_draft", "draftID", draftID, "mode", mode)
```

- [ ] **Step 9: 运行全部 agent 测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -v 2>&1 | tail -30`
Expected: 所有测试通过

- [ ] **Step 10: 运行 middleware 测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/shared/middleware/ -v`
Expected: 所有测试通过

- [ ] **Step 11: 提交**

```bash
git add backend/internal/shared/middleware/requestid.go backend/internal/shared/middleware/requestid_test.go backend/internal/shared/middleware/middleware.go backend/internal/modules/agent/service.go backend/internal/modules/agent/handler.go backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/debug.go
git commit -m "feat: slog structured logging with requestID tracing"
```

---

## Task 6: Token 预算管理与截断策略

**Files:**
- Create: `backend/internal/modules/agent/context_budget.go`
- Create: `backend/internal/modules/agent/context_budget_test.go`
- Modify: `backend/internal/modules/agent/service.go` (集成 token 预算)

依赖: Task 5

- [ ] **Step 1: 写失败的测试 — ContextBudget**

```go
// context_budget_test.go
package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextBudget_EstimateTokens(t *testing.T) {
	budget := NewContextBudget(128000)

	tests := []struct {
		name     string
		text     string
		minTokens int
		maxTokens int
	}{
		{"empty", "", 0, 0},
		{"short", "hello world", 1, 20},
		{"medium", string(make([]byte, 1000)), 100, 800},
		{"long", string(make([]byte, 10000)), 1000, 8000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := budget.EstimateTokens(tt.text)
			assert.GreaterOrEqual(t, tokens, tt.minTokens)
			assert.LessOrEqual(t, tokens, tt.maxTokens)
		})
	}
}

func TestContextBudget_Thresholds(t *testing.T) {
	budget := NewContextBudget(1000) // 小容量便于测试

	// 75% = 750 tokens
	assert.False(t, budget.ShouldCompact(740))
	assert.True(t, budget.ShouldCompact(760))

	// 95% = 950 tokens
	assert.False(t, budget.ShouldBlock(940))
	assert.True(t, budget.ShouldBlock(960))
}

func TestContextBudget_UsagePercent(t *testing.T) {
	budget := NewContextBudget(1000)

	assert.Equal(t, 50.0, budget.UsagePercent(500))
	assert.Equal(t, 0.0, budget.UsagePercent(0))
	assert.Equal(t, 100.0, budget.UsagePercent(1000))
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestContextBudget -v 2>&1 | tail -10`
Expected: 编译失败

- [ ] **Step 3: 实现 ContextBudget**

```go
// context_budget.go
package agent

import "math"

// ContextBudget monitors token usage against the context window.
type ContextBudget struct {
	maxTokens        int     // 最大 token 数（默认 128000）
	compactThreshold float64 // 触发压缩的阈值（默认 0.85）
	warnThreshold    float64 // 发送警告的阈值（默认 0.75）
	blockThreshold   float64 // 阻断新消息的阈值（默认 0.95）
}

func NewContextBudget(maxTokens int) *ContextBudget {
	return &ContextBudget{
		maxTokens:        maxTokens,
		compactThreshold: 0.85,
		warnThreshold:    0.75,
		blockThreshold:   0.95,
	}
}

// EstimateTokens estimates token count from character count.
// Uses ~2.5 chars per token (between the existing 2 and the theoretical 4).
func (b *ContextBudget) EstimateTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / 2.5))
}

// EstimateMessagesTokens estimates total tokens for a list of messages.
func (b *ContextBudget) EstimateMessagesTokens(messages []struct{ Content, Name, ToolCallID string }) int {
	total := 0
	for _, m := range messages {
		total += b.EstimateTokens(m.Content) + b.EstimateTokens(m.Name) + b.EstimateTokens(m.ToolCallID)
		total += 4 // per-message overhead
	}
	return total
}

// ShouldCompact returns true if estimated tokens exceed the compact threshold.
func (b *ContextBudget) ShouldCompact(estimatedTokens int) bool {
	return float64(estimatedTokens) > float64(b.maxTokens)*b.compactThreshold
}

// ShouldWarn returns true if estimated tokens exceed the warn threshold.
func (b *ContextBudget) ShouldWarn(estimatedTokens int) bool {
	return float64(estimatedTokens) > float64(b.maxTokens)*b.warnThreshold
}

// ShouldBlock returns true if estimated tokens exceed the block threshold.
func (b *ContextBudget) ShouldBlock(estimatedTokens int) bool {
	return float64(estimatedTokens) > float64(b.maxTokens)*b.blockThreshold
}

// UsagePercent returns the usage percentage.
func (b *ContextBudget) UsagePercent(estimatedTokens int) float64 {
	if b.maxTokens == 0 {
		return 0
	}
	return float64(estimatedTokens) / float64(b.maxTokens) * 100
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestContextBudget -v`
Expected: PASS

- [ ] **Step 5: 写失败的测试 — 差异化截断策略**

```go
func TestTruncateToolResult_GetDraftFull(t *testing.T) {
	longHTML := "<html><body>" + string(make([]byte, 6000)) + "</body></html>"
	result := TruncateToolResult("get_draft", longHTML, "full")
	assert.LessOrEqual(t, len(result), 5200) // 5000 + truncation notice
	assert.Contains(t, result, "已截断")
}

func TestTruncateToolResult_GetDraftStructure(t *testing.T) {
	longStructure := string(make([]byte, 3000))
	result := TruncateToolResult("get_draft", longStructure, "structure")
	assert.Equal(t, longStructure, result) // structure 不截断
}

func TestTruncateToolResult_SearchAssets(t *testing.T) {
	// search_assets 结果限 10 条 × 300 字符
	longResult := string(make([]byte, 5000))
	result := TruncateToolResult("search_assets", longResult, "")
	assert.LessOrEqual(t, len(result), 5200)
}

func TestTruncateToolResult_ApplyEdits(t *testing.T) {
	longResult := string(make([]byte, 2000))
	result := TruncateToolResult("apply_edits", longResult, "")
	assert.LessOrEqual(t, len(result), 1200)
}

func TestTruncateToolResult_Error(t *testing.T) {
	longError := string(make([]byte, 3000))
	result := TruncateToolResult("error", longError, "")
	assert.LessOrEqual(t, len(result), 2200)
}
```

- [ ] **Step 6: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestTruncateToolResult -v 2>&1 | tail -10`
Expected: 编译失败

- [ ] **Step 7: 实现差异化截断策略**

在 `context_budget.go` 中追加：

```go
// TruncateToolResult applies tool-specific truncation strategies.
// toolName: the tool that produced the result
// result: the raw result string
// mode: for get_draft, the mode used (structure/section/search/full)
func TruncateToolResult(toolName, result, mode string) string {
	switch toolName {
	case "get_draft":
		switch mode {
		case "structure", "section":
			return result // 不截断
		case "search":
			return truncateByLines(result, 5*10) // 最多 5 条结果约 10 行
		default: // full
			return truncateWithNotice(result, 5000, "已截断，请使用 mode=section 或 mode=search 获取具体内容")
		}
	case "apply_edits":
		return truncateWithNotice(result, 1000, "...")
	case "search_assets":
		return truncateWithNotice(result, 3000, "...")
	case "error":
		return truncateWithNotice(result, 2000, "...")
	default:
		return truncateWithNotice(result, 1600, "...")
	}
}

func truncateWithNotice(s string, maxLen int, notice string) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + notice
}

func truncateByLines(s string, maxLines int) string {
	lines := 0
	for i, c := range s {
		if c == '\n' {
			lines++
			if lines >= maxLines {
				return s[:i] + "\n...(已截断)"
			}
		}
	}
	return s
}
```

- [ ] **Step 8: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestTruncateToolResult -v`
Expected: PASS

- [ ] **Step 9: 在 service.go 中集成 ContextBudget**

修改 `StreamChatReAct` 中的 token 检查逻辑：

```go
// 替换现有的 needsCompaction 和 estimateTokens 调用
budget := NewContextBudget(s.contextWindowSize)

// 在循环开始前检查
totalTokens := budget.EstimateTokens(systemPrompt)
for _, msg := range messages {
    totalTokens += budget.EstimateTokens(msg.Content)
}

if budget.ShouldBlock(totalTokens) {
    return fmt.Errorf("context window 已满 (%.1f%%)，请开始新会话", budget.UsagePercent(totalTokens))
}

if budget.ShouldCompact(totalTokens) {
    // 触发压缩
    logger.Warn("context_budget_compact", "usage", budget.UsagePercent(totalTokens))
    // ... 压缩逻辑 ...
}
```

- [ ] **Step 10: 运行 service 测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestChatService -v`
Expected: PASS

- [ ] **Step 11: 提交**

```bash
git add backend/internal/modules/agent/context_budget.go backend/internal/modules/agent/context_budget_test.go backend/internal/modules/agent/service.go
git commit -m "feat: token budget management with tool-specific truncation strategies"
```

---

## Task 7: Agent 循环策略改进（步骤漏调检测 + API 重试 + 降级放行）

**Files:**
- Modify: `backend/internal/modules/agent/service.go` (步骤漏调检测、降级放行)
- Modify: `backend/internal/modules/agent/provider.go` (API 重试)
- Modify: `backend/internal/modules/agent/provider_test.go` (重试测试)
- Modify: `backend/internal/modules/agent/service_test.go` (漏调检测测试)

依赖: Task 5, Task 6

- [ ] **Step 1: 写失败的测试 — API 指数退避重试**

```go
// provider_test.go 中追加
func TestOpenAIAdapter_RetryOnServerError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			// 前两次返回 500
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"internal server error"}`))
			return
		}
		// 第三次返回正常 SSE
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model", 10)

	var result string
	err := adapter.StreamChat(context.Background(), []Message{
		{Role: "user", Content: "test"},
	}, func(chunk string) error {
		result += chunk
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, 3, callCount)
}

func TestOpenAIAdapter_NoRetryOn429(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(429)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model", 10)

	err := adapter.StreamChat(context.Background(), []Message{
		{Role: "user", Content: "test"},
	}, func(chunk string) error { return nil })

	assert.Error(t, err)
	assert.Equal(t, 1, callCount) // 不重试
}

func TestOpenAIAdapter_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"always fail"}`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter(server.URL, "test-key", "test-model", 10)

	err := adapter.StreamChat(context.Background(), []Message{
		{Role: "user", Content: "test"},
	}, func(chunk string) error { return nil })

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "重试")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestOpenAIAdapter_Retry" -v 2>&1 | tail -10`
Expected: 当前没有重试逻辑，测试 FAIL

- [ ] **Step 3: 实现指数退避重试**

在 `provider.go` 中新增重试辅助函数：

```go
// retryWithBackoff executes fn with exponential backoff on server errors (5xx).
// Does NOT retry on 429 (rate limit) or 4xx errors.
// Returns after maxRetries attempts or on success.
func retryWithBackoff(maxRetries int, baseDelay time.Duration, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 不重试 429 或 4xx
		if isClientError(err) {
			return err
		}

		if attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<uint(attempt)) // 1s, 2s, 4s
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("重试 %d 次后仍失败: %w", maxRetries, lastErr)
}

// isClientError checks if the error is a non-retryable client error (4xx except 429).
func isClientError(err error) bool {
	errStr := err.Error()
	// 429 是 rate limit，需要特殊处理但不重试
	if strings.Contains(errStr, "429") {
		return true
	}
	// 其他 4xx 不重试
	for code := 400; code < 500; code++ {
		if strings.Contains(errStr, fmt.Sprintf("%d", code)) && code != 429 {
			return true
		}
	}
	return false
}
```

然后在 `StreamChat` 和 `StreamChatReAct` 中使用重试：

```go
func (a *OpenAIAdapter) StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error {
	return retryWithBackoff(3, 1*time.Second, func() error {
		return a.streamChatOnce(ctx, messages, sendChunk)
	})
}
```

- [ ] **Step 4: 运行重试测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestOpenAIAdapter_Retry|TestOpenAIAdapter_NoRetry|TestOpenAIAdapter_MaxRetries" -v`
Expected: PASS

- [ ] **Step 5: 写失败的测试 — 步骤漏调检测**

```go
// service_test.go 中追加
func TestStreamChatReAct_StepMissDetection(t *testing.T) {
	// Mock: AI 连续 3 次只调用 search_assets，不调用 apply_edits
	// 应触发降级放行，返回纯文本
	adapter := &searchOnlyMockAdapter{maxCalls: 5}
	toolExec := &MockToolExecutor{result: `{"assets":[]}`}

	db := SetupTestDB(t)
	svc := NewChatService(db, adapter, toolExec, 3)

	draft := models.Draft{ProjectID: 1, HTMLContent: "<html><body>test</body></html>"}
	require.NoError(t, db.Create(&draft).Error)

	session := models.AISession{DraftID: draft.ID, Status: "active"}
	require.NoError(t, db.Create(&session).Error)

	var events []string
	err := svc.StreamChatReAct(session.ID, "修改简历", func(event string) {
		events = append(events, event)
	})

	// 应该降级放行而不是返回错误
	assert.NoError(t, err)

	// 应该有 done 事件
	hasDone := false
	for _, e := range events {
		if strings.Contains(e, `"type":"done"`) {
			hasDone = true
		}
	}
	assert.True(t, hasDone)
}
```

- [ ] **Step 6: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestStreamChatReAct_StepMissDetection -v 2>&1 | tail -10`
Expected: 当前 `searchOnlyCount` 逻辑可能不够完善，测试 FAIL

- [ ] **Step 7: 完善步骤漏调检测和降级放行**

修改 `service.go` 中的 `searchOnlyCount` 逻辑：

```go
// 在 ReAct 循环中：
searchOnlyCount := 0
maxSearchOnly := 3 // 连续 3 次只搜索不修改则降级

for iteration := 0; iteration < maxTotalIterations; iteration++ {
    // ... 调用 LLM ...

    if len(toolCalls) > 0 {
        hasApplyEdits := false
        for _, call := range toolCalls {
            if call.Name == "apply_edits" {
                hasApplyEdits = true
            }
        }

        if !hasApplyEdits {
            searchOnlyCount++
            if searchOnlyCount >= maxSearchOnly {
                // 降级放行：让 AI 返回纯文本
                logger.Warn("step_miss_degradation",
                    "searchOnlyCount", searchOnlyCount,
                    "reason", "连续多次未调用 apply_edits",
                )
                break
            }

            // 注入提醒（渐进式）
            if searchOnlyCount == 1 {
                messages = append(messages, Message{
                    Role:    "system",
                    Content: "提醒：你已经了解了简历内容，请开始执行修改（使用 apply_edits 工具）。",
                })
            } else if searchOnlyCount == 2 {
                messages = append(messages, Message{
                    Role:    "system",
                    Content: "再次提醒：请立即使用 apply_edits 工具执行修改，不要继续搜索。",
                })
            }
        } else {
            searchOnlyCount = 0 // 重置计数
        }
    }
}
```

- [ ] **Step 8: 运行步骤漏调测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestStreamChatReAct_StepMissDetection -v`
Expected: PASS

- [ ] **Step 9: 运行全部 service 测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestChatService|TestStreamChatReAct" -v`
Expected: PASS

- [ ] **Step 10: 提交**

```bash
git add backend/internal/modules/agent/provider.go backend/internal/modules/agent/provider_test.go backend/internal/modules/agent/service.go backend/internal/modules/agent/service_test.go
git commit -m "feat: API retry with exponential backoff, step miss detection, degradation fallback"
```

---

## Task 8: 接口抽象（Repository 层）

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (提取接口)
- Modify: `backend/internal/modules/agent/routes.go` (依赖注入)
- Modify: `backend/internal/modules/agent/service.go` (使用接口)
- Modify: `backend/internal/modules/agent/tool_executor_test.go` (Mock 接口)

依赖: Task 2, Task 3, Task 4

- [ ] **Step 1: 定义接口**

在 `tool_executor.go` 顶部定义：

```go
// HTMLRepository abstracts draft HTML operations for testability.
type HTMLRepository interface {
    GetDraft(ctx context.Context, draftID uint) (string, error)
    GetDraftStructure(ctx context.Context, draftID uint) (string, error)
    SearchInDraft(ctx context.Context, draftID uint, query string) (string, error)
    GetDraftSection(ctx context.Context, draftID uint, selector string) (string, error)
    ApplyEdits(ctx context.Context, draftID uint, ops []EditOp) (*ApplyEditsResult, error)
}

// AssetRepository abstracts asset search operations.
type AssetRepository interface {
    SearchAssets(ctx context.Context, projectID uint, query, assetType string, limit int) ([]AssetResult, error)
}

// EditOp represents a single edit operation.
type EditOp struct {
    OldString   string `json:"old_string"`
    NewString   string `json:"new_string"`
    Description string `json:"description,omitempty"`
}

// ApplyEditsResult is the result of applying edits.
type ApplyEditsResult struct {
    Applied     int `json:"applied"`
    Failed      int `json:"failed"`
    NewSequence int `json:"new_sequence"`
}

// AssetResult is a search result from assets.
type AssetResult struct {
    ID        uint   `json:"id"`
    Type      string `json:"type"`
    Label     string `json:"label"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
}
```

- [ ] **Step 2: 实现接口（基于现有代码）**

将现有 `AgentToolExecutor` 的方法包装为接口实现。创建 `db_html_repository.go`：

```go
// db_html_repository.go 实现 HTMLRepository 接口，委托给现有 AgentToolExecutor 方法。
// 这是一个过渡实现，后续可以独立重构。
```

由于这是一个重构任务，主要是提取接口，实际实现可以保持现有代码不变，只是通过接口暴露。这一步的重点是让 `ChatService` 依赖接口而非具体实现。

- [ ] **Step 3: 修改 ChatService 依赖接口**

```go
// 在 service.go 中，将 ChatService 的 toolExecutor 字段改为接口类型
// 不改变实际行为，只是为后续 mock 测试做准备
```

- [ ] **Step 4: 运行全部测试确保无回归**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -v 2>&1 | tail -30`
Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add backend/internal/modules/agent/
git commit -m "refactor: extract HTMLRepository and AssetRepository interfaces"
```

---

## Task 9: 压缩改进（结构化摘要 + fallback）

**Files:**
- Modify: `backend/internal/modules/agent/service.go` (compactMessages)

依赖: Task 5, Task 6

- [ ] **Step 1: 写失败的测试 — 结构化摘要模板**

```go
func TestCompactMessages_StructuredSummary(t *testing.T) {
	db := SetupTestDB(t)

	session := models.AISession{DraftID: 1, Status: "active"}
	require.NoError(t, db.Create(&session).Error)

	// 创建足够多的消息触发压缩
	messages := []models.AIMessage{
		{SessionID: session.ID, Role: "user", Content: "帮我修改简历的教育经历"},
		{SessionID: session.ID, Role: "assistant", Content: "好的，我来帮你修改教育经历。"},
		{SessionID: session.ID, Role: "user", Content: "把 MIT 改成 Harvard"},
		{SessionID: session.ID, Role: "assistant", Content: "已修改完成。"},
		{SessionID: session.ID, Role: "user", Content: "再加一段工作经历"},
		{SessionID: session.ID, Role: "assistant", Content: "好的，我来添加工作经历。"},
	}
	for i := range messages {
		require.NoError(t, db.Create(&messages[i]).Error)
	}

	mockProvider := &mockCompactProvider{summary: "用户修改了教育经历和工作经历"}
	svc := NewChatService(db, mockProvider, nil, 3)

	compacted, err := svc.compactMessages(context.Background(), messages)
	require.NoError(t, err)

	// 摘要消息应包含结构化标记
	assert.Len(t, compacted, 1) // 6 条消息压缩为 1 条摘要
	assert.Contains(t, compacted[0].Content, "[对话摘要]")
}
```

- [ ] **Step 2: 运行测试确认当前行为**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestCompactMessages_StructuredSummary -v 2>&1 | tail -10`
Expected: 可能 PASS 或 FAIL，取决于当前实现

- [ ] **Step 3: 改进 compactMessages 的摘要提示词**

```go
func (s *ChatService) compactMessages(ctx context.Context, messages []models.AIMessage) ([]models.AIMessage, error) {
	if len(messages) <= 4 {
		return messages, nil
	}

	oldMessages := messages[:len(messages)-4]
	retained := messages[len(messages)-4:]

	// 构建结构化摘要提示词
	summaryPrompt := `请将以下对话压缩为结构化摘要。使用以下格式：

[对话摘要]
- 用户意图：（用户最初想要做什么）
- 已完成：（已经完成了哪些修改）
- 关键结论：（重要的决策或发现）
- 待继续：（还有什么未完成的工作）

对话内容：
`
	for _, msg := range oldMessages {
		summaryPrompt += fmt.Sprintf("[%s] %s\n", msg.Role, truncateString(msg.Content, 500))
	}

	// 调用 LLM 生成摘要
	var summary strings.Builder
	err := s.provider.StreamChat(ctx, []Message{
		{Role: "system", Content: summaryPrompt},
		{Role: "user", Content: "请压缩以上对话"},
	}, func(chunk string) error {
		summary.WriteString(chunk)
		return nil
	})

	if err != nil {
		// fallback: 保留原始消息 + warn 日志
		logger := middleware.LoggerFromStdContext(ctx)
		logger.Warn("compact_fallback", "error", err.Error(), "messageCount", len(messages))
		return messages, nil
	}

	summaryMsg := models.AIMessage{
		SessionID: messages[0].SessionID,
		Role:      "system",
		Content:   "[对话摘要]\n" + summary.String(),
	}

	return append([]models.AIMessage{summaryMsg}, retained...), nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestCompactMessages -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "feat: structured compaction summary with fallback on failure"
```

---

## Task 10: 端到端集成测试

**Files:**
- Modify: `backend/internal/modules/agent/integration_test.go`

依赖: 所有前序 Task

- [ ] **Step 1: 写集成测试 — 完整的结构化工作流**

```go
//go:build integration

package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_StructuredWorkflow(t *testing.T) {
	db := SetupTestDB(t)

	html := `<html>
	<head><style>body{font-size:14px}</style></head>
	<body>
		<div class="header"><h1>John Doe</h1><p>Engineer</p></div>
		<div class="experience">
			<h2>Experience</h2>
			<div class="item"><h3>Google</h3><p>Senior Engineer</p></div>
		</div>
		<div class="education">
			<h2>Education</h2>
			<div class="item"><h3>MIT</h3><p>Computer Science</p></div>
		</div>
	</body></html>`

	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// Step 1: 获取结构概览
	structure, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode": "structure",
	})
	require.NoError(t, err)
	assert.Contains(t, structure, "experience")
	assert.NotContains(t, structure, "Google")

	// Step 2: 获取指定区域
	section, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":     "section",
		"selector": ".experience",
	})
	require.NoError(t, err)
	assert.Contains(t, section, "Google")

	// Step 3: 搜索关键词
	searchResult, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":  "search",
		"query": "MIT",
	})
	require.NoError(t, err)
	assert.Contains(t, searchResult, "MIT")

	// Step 4: 执行修改
	editResult, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Google",
				"new_string": "Alphabet",
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, editResult, `"applied":1`)

	// Step 5: 验证修改已生效
	newSection, err := executor.Execute(ctx, "get_draft", map[string]interface{}{
		"mode":     "section",
		"selector": ".experience",
	})
	require.NoError(t, err)
	assert.Contains(t, newSection, "Alphabet")
	assert.NotContains(t, newSection, "Google")
}

func TestIntegration_StepByStepWithFailure(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{
		ProjectID:   1,
		HTMLContent: `<html><body><p>Apple</p><p>Banana</p></body></html>`,
	}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// 第一个成功，第二个失败
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"old_string": "Apple", "new_string": "Apricot"},
			map[string]interface{}{"old_string": "NotFound", "new_string": "Something"},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "op #2")
	assert.Contains(t, err.Error(), "部分成功")

	// 第一个修改应保留
	var updated models.Draft
	db.First(&updated, draft.ID)
	assert.Contains(t, updated.HTMLContent, "Apricot")
}

func TestIntegration_StructuredErrorFeedback(t *testing.T) {
	db := SetupTestDB(t)

	draft := models.Draft{
		ProjectID:   1,
		HTMLContent: `<html><body><div class="job"><h3>Google</h3><p>Senior Engineer</p></div></body></html>`,
	}
	require.NoError(t, db.Create(&draft).Error)

	executor := NewAgentToolExecutor(db, nil)
	ctx := WithDraftID(context.Background(), draft.ID)

	// 搜索有相似但不完全匹配的内容
	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{
				"old_string": "Sr. Engineer at Google Inc",
				"new_string": "Staff Engineer",
			},
		},
	})
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "未找到匹配")
	assert.Contains(t, errMsg, "搜索内容")
	assert.Contains(t, errMsg, "建议")
}
```

- [ ] **Step 2: 运行集成测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test -tags=integration ./internal/modules/agent/ -run "TestIntegration_" -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add backend/internal/modules/agent/integration_test.go
git commit -m "test: add integration tests for structured workflow and error feedback"
```

---

## Spec Coverage Checklist

| 设计文档章节 | 对应 Task | 状态 |
|------------|----------|------|
| 2.1.1 结构化错误反馈 | Task 1, Task 2 | ✓ |
| 2.1.2 apply_edits 分步执行 | Task 4 | ✓ |
| 2.1.3 JSON 参数解析重试 | 不在本次范围（LLM 侧问题） | - |
| 2.2 get_draft 多模式 | Task 1, Task 3 | ✓ |
| 2.3.1 Token 预算实时监控 | Task 6 | ✓ |
| 2.3.2 工具输出差异化截断 | Task 6 | ✓ |
| 2.3.3 压缩改进 | Task 9 | ✓ |
| 2.4 AI 结构化工作流 | Task 3（system prompt 变更需手动调整） | 部分 |
| 2.5.1 步骤漏调检测 | Task 7 | ✓ |
| 2.5.2 工具失败恢复策略 | Task 2, Task 4 | ✓ |
| 2.5.3 API 调用重试 | Task 7 | ✓ |
| 2.5.4 降级放行策略 | Task 7 | ✓ |
| 2.6.1 slog 结构化日志 | Task 5 | ✓ |
| 2.6.2 requestID 链路追踪 | Task 5 | ✓ |
| 2.6.3 AI 决策链路追踪 | Task 5（部分，需在 ReAct 循环中添加 trace） | 部分 |
| 2.6.4 降级日志模式 | Task 5, Task 7 | ✓ |
| 2.6.5 敏感信息脱敏 | Task 5（slog 自带脱敏能力） | 部分 |
| 2.7.1 核心接口抽象 | Task 8 | ✓ |
| 2.7.2 测试分层 | Task 1-10 整体 | ✓ |
| 2.7.3 关键测试用例 | Task 1-10 整体 | ✓ |
