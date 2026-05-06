# AI Agent V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current AI agent with a Claude Code-inspired architecture using Search/Replace edits, inline visual diffs, and proper context management.

**Architecture:** 3 core tools (get_draft, apply_edits, search_assets) replace the current 5 CRUD tools. Edits use SearchReplaceOps with atomic validation. Changes are shown as inline diffs in TipTap (Word revision mode). Conversation history is compacted at 80% of the context window. Undo/redo uses a draft_edits table with HTML snapshots.

**Tech Stack:** Go + Gin + GORM (backend), TipTap v3 + React 18 + TypeScript (frontend), PostgreSQL (database)

---

## File Structure

### Backend — modify

| File | Changes |
|------|---------|
| `backend/internal/shared/models/models.go` | Add `DraftEdit` model; add `CurrentEditSequence` to `Draft` |
| `backend/internal/shared/database/database.go` | Add `DraftEdit` to `AutoMigrate` |
| `backend/internal/modules/agent/tool_executor.go` | Rewrite: 3 new tools, remove 5 old tools, remove VersionService/ExportService deps |
| `backend/internal/modules/agent/service.go` | Rewrite: fixed system prompt, compaction, context-aware tools, remove `StreamChat` legacy |
| `backend/internal/modules/agent/provider.go` | Update `Message` struct (add `ToolCallID`, `Name`) |
| `backend/internal/modules/agent/handler.go` | Add undo/redo endpoints, update Handler struct |
| `backend/internal/modules/agent/routes.go` | Add undo/redo routes, add `CONTEXT_WINDOW_SIZE` env, remove unused deps |
| `backend/cmd/server/main.go` | Update agent.RegisterRoutes call |
| `backend/internal/modules/agent/tool_executor_test.go` | Rewrite tests for new tools |
| `backend/internal/modules/agent/service_test.go` | Rewrite tests for new service |
| `backend/internal/modules/agent/handler_test.go` | Add undo/redo tests |

### Backend — add dependency

| Dependency | Purpose |
|------------|---------|
| `github.com/PuerkitoBio/goquery` | CSS selector support in get_draft |

### Frontend — create

| File | Purpose |
|------|---------|
| `frontend/workbench/src/components/editor/extensions/ai-diff.ts` | TipTap marks for `<del>` and `<ins>` |
| `frontend/workbench/src/components/chat/ToolCallLog.tsx` | Collapsible tool call log |
| `frontend/workbench/src/components/chat/AiPlan.tsx` | AI plan/TODO display |
| `frontend/workbench/src/components/editor/UndoRedoBar.tsx` | Undo/Redo buttons |

### Frontend — modify

| File | Changes |
|------|---------|
| `frontend/workbench/src/components/chat/ChatPanel.tsx` | Major rewrite: remove HTML markers, add tool logs, handle edit events |
| `frontend/workbench/src/pages/EditorPage.tsx` | Wire diff extension, undo/redo bar, new ChatPanel props |
| `frontend/workbench/src/lib/api-client.ts` | Add undo/redo API, edit event types |
| `frontend/workbench/src/styles/editor.css` | Add diff CSS (del/ins styling) |

### Frontend — delete

| File | Reason |
|------|--------|
| `frontend/workbench/src/components/chat/HtmlPreview.tsx` | No longer needed (no HTML marker extraction) |

---

## Task 1: DraftEdit Model + Database Migration

**Files:**
- Modify: `backend/internal/shared/models/models.go:69-79,128-140`
- Modify: `backend/internal/shared/database/database.go:43-58`
- Test: `backend/internal/shared/models/models_test.go`

- [ ] **Step 1: Add DraftEdit model**

Add after `AIToolCall` model (after line 140) in `models.go`:

```go
// DraftEdit records each AI edit on a draft for undo/redo.
type DraftEdit struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	DraftID      uint           `gorm:"not null;index" json:"draft_id"`
	Sequence     int            `gorm:"not null" json:"sequence"`
	OpType       string         `gorm:"size:20;not null" json:"op_type"`
	OldString    string         `gorm:"type:text" json:"old_string,omitempty"`
	NewString    string         `gorm:"type:text" json:"new_string,omitempty"`
	Description  string         `gorm:"type:text" json:"description,omitempty"`
	HtmlSnapshot string         `gorm:"type:text;not null" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
```

- [ ] **Step 2: Add CurrentEditSequence to Draft model**

Add field to Draft struct (after `UpdatedAt`):

```go
CurrentEditSequence int `gorm:"default:0" json:"-"`
```

- [ ] **Step 3: Add DraftEdit to AutoMigrate**

In `database.go` Migrate function, add `DraftEdit{}` to the model list (after `AIToolCall{}`).

- [ ] **Step 4: Write test**

```go
func TestDraftEditModel(t *testing.T) {
	db := testutil.SetupTestDB(t)

	draft := models.Draft{ProjectID: 1, HTMLContent: "<html>test</html>"}
	require.NoError(t, db.Create(&draft).Error)

	edit := models.DraftEdit{
		DraftID: draft.ID, Sequence: 1, OpType: "search_replace",
		OldString: "old", NewString: "new", Description: "test",
		HtmlSnapshot: "<html>new</html>",
	}
	require.NoError(t, db.Create(&edit).Error)
	assert.Equal(t, uint(1), edit.ID)
	assert.Equal(t, 1, edit.Sequence)

	var loaded models.Draft
	db.First(&loaded, draft.ID)
	assert.Equal(t, 0, loaded.CurrentEditSequence)
}
```

- [ ] **Step 5: Run test**

Run: `cd backend && go test ./internal/shared/models/... -v -run TestDraftEditModel`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/shared/models/models.go backend/internal/shared/database/database.go backend/internal/shared/models/models_test.go
git commit -m "feat: add DraftEdit model for undo/redo support"
```

---

## Task 2: Add goquery Dependency

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`

- [ ] **Step 1: Install goquery**

Run: `cd backend && go get github.com/PuerkitoBio/goquery`

- [ ] **Step 2: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "deps: add goquery for CSS selector support"
```

---

## Task 3: Rewrite tool_executor.go (3 Tools)

**Files:**
- Rewrite: `backend/internal/modules/agent/tool_executor.go`
- Rewrite test: `backend/internal/modules/agent/tool_executor_test.go`

Remove `VersionService` and `ExportService` interfaces. Replace 5 old tools with 3 new ones. Tools get draft_id/project_id from context instead of AI parameters.

- [ ] **Step 1: Write failing test for get_draft**

```go
func TestGetDraft_Full(t *testing.T) {
	db := testutil.SetupTestDB(t)
	draft := models.Draft{ProjectID: 1, HTMLContent: "<html><body><h1>Test</h1></body></html>"}
	require.NoError(t, db.Create(&draft).Error)

	ctx := context.WithValue(context.Background(), draftIDKey, draft.ID)
	ctx = context.WithValue(ctx, projectIDKey, uint(1))

	executor := NewAgentToolExecutor(db)
	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{})
	require.NoError(t, err)
	assert.Contains(t, result, "<h1>Test</h1>")
}

func TestGetDraft_Selector(t *testing.T) {
	db := testutil.SetupTestDB(t)
	html := "<html><body><div id='exp'>Experience</div><div id='edu'>Education</div></body></html>"
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := context.WithValue(context.Background(), draftIDKey, draft.ID)
	ctx = context.WithValue(ctx, projectIDKey, uint(1))

	executor := NewAgentToolExecutor(db)
	result, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"selector": "#exp"})
	require.NoError(t, err)
	assert.Contains(t, result, "Experience")
	assert.NotContains(t, result, "Education")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -v -run TestGetDraft`
Expected: FAIL (context keys not defined)

- [ ] **Step 3: Write failing tests for apply_edits**

```go
func TestApplyEdits_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	html := `<html><body><h1>Old Title</h1><p>Old content</p></body></html>`
	draft := models.Draft{ProjectID: 1, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := context.WithValue(context.Background(), draftIDKey, draft.ID)
	executor := NewAgentToolExecutor(db)

	result, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"old_string": "Old Title", "new_string": "New Title", "description": "update title"},
			map[string]interface{}{"old_string": "Old content", "new_string": "New content", "description": "update body"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"applied": 2`)

	var updated models.Draft
	db.First(&updated, draft.ID)
	assert.Contains(t, updated.HTMLContent, "New Title")
	assert.Contains(t, updated.HTMLContent, "New content")
	assert.Equal(t, 2, updated.CurrentEditSequence)

	var edits []models.DraftEdit
	db.Where("draft_id = ?", draft.ID).Order("sequence ASC").Find(&edits)
	assert.Equal(t, 3, len(edits)) // base + 2 ops
	assert.Equal(t, 0, edits[0].Sequence)
	assert.Equal(t, "base", edits[0].OpType)
}

func TestApplyEdits_OldStringNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	draft := models.Draft{ProjectID: 1, HTMLContent: "<html>test</html>"}
	require.NoError(t, db.Create(&draft).Error)

	ctx := context.WithValue(context.Background(), draftIDKey, draft.ID)
	executor := NewAgentToolExecutor(db)

	_, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"old_string": "NOT_FOUND", "new_string": "new"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "old_string not found")

	var unchanged models.Draft
	db.First(&unchanged, draft.ID)
	assert.Equal(t, "<html>test</html>", unchanged.HTMLContent)
	assert.Equal(t, 0, unchanged.CurrentEditSequence)
}
```

- [ ] **Step 4: Write failing test for search_assets**

```go
func TestSearchAssets(t *testing.T) {
	db := testutil.SetupTestDB(t)

	db.Create(&models.Asset{ProjectID: 1, Type: "resume", Label: "Old Resume", Content: "Python developer with 5 years experience"})
	db.Create(&models.Asset{ProjectID: 1, Type: "note", Label: "Notes", Content: "Focus on Go and cloud"})
	db.Create(&models.Asset{ProjectID: 2, Type: "resume", Label: "Other", Content: "Java developer"})

	ctx := context.WithValue(context.Background(), projectIDKey, uint(1))
	executor := NewAgentToolExecutor(db)

	result, err := executor.Execute(ctx, "search_assets", map[string]interface{}{"query": "Python"})
	require.NoError(t, err)
	assert.Contains(t, result, "Python developer")
	assert.NotContains(t, result, "Java developer")

	result, err = executor.Execute(ctx, "search_assets", map[string]interface{}{"type": "note"})
	require.NoError(t, err)
	assert.Contains(t, result, "Focus on Go")
}
```

- [ ] **Step 5: Implement tool_executor.go**

Replace entire file. Key components:

**Context keys:**
```go
type contextKey int

const (
	draftIDKey  contextKey = iota
	projectIDKey
)
```

**Constructor (simplified, no service deps):**
```go
type AgentToolExecutor struct {
	db *gorm.DB
}

func NewAgentToolExecutor(db *gorm.DB) *AgentToolExecutor {
	return &AgentToolExecutor{db: db}
}
```

**Tool definitions — 3 tools:**
```go
func (e *AgentToolExecutor) Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "get_draft",
			Description: "读取当前简历 HTML。不带参数返回完整 HTML，带 selector 参数只返回匹配的片段（CSS 选择器）。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS 选择器，例如 '#experience'、'.skill-item'。不传则返回完整 HTML。",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "apply_edits",
			Description: "对简历 HTML 应用搜索替换编辑。提交一组操作，全部验证通过后原子执行。old_string 必须精确匹配当前 HTML。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ops": map[string]interface{}{
						"type":        "array",
						"description": "搜索替换操作数组",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"old_string":  map[string]interface{}{"type": "string", "description": "必须在当前 HTML 中精确匹配的文本"},
								"new_string":  map[string]interface{}{"type": "string", "description": "替换后的文本"},
								"description": map[string]interface{}{"type": "string", "description": "修改说明（可选）"},
							},
							"required":             []string{"old_string", "new_string"},
						},
					},
				},
				"required": []string{"ops"},
			},
		},
		{
			Name:        "search_assets",
			Description: "搜索用户资料（旧简历、Git 摘要、笔记等）。可按关键词和类型过滤。长内容返回前 2000 字符。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "搜索关键词"},
					"type":  map[string]interface{}{"type": "string", "description": "资料类型：resume | git_summary | note"},
					"limit": map[string]interface{}{"type": "integer", "description": "返回数量上限，默认 5"},
				},
				"required": []string{},
			},
		},
	}
}
```

**Execute dispatch:**
```go
func (e *AgentToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	switch toolName {
	case "get_draft":
		return e.getDraft(ctx, params)
	case "apply_edits":
		return e.applyEdits(ctx, params)
	case "search_assets":
		return e.searchAssets(ctx, params)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}
```

**get_draft implementation:**
```go
func (e *AgentToolExecutor) getDraft(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok {
		return "", errors.New("draft_id not found in context")
	}

	var draft models.Draft
	if err := e.db.Select("html_content").First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}

	selector, _ := params["selector"].(string)
	if selector == "" {
		return draft.HTMLContent, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(draft.HTMLContent))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}
	html, err := doc.Find(selector).Html()
	if err != nil {
		return "", fmt.Errorf("extract selector: %w", err)
	}
	return html, nil
}
```

**apply_edits implementation (transactional):**
```go
func (e *AgentToolExecutor) applyEdits(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok {
		return "", errors.New("draft_id not found in context")
	}

	opsRaw, ok := params["ops"].([]interface{})
	if !ok {
		return "", errors.New("ops must be an array")
	}

	return e.db.Transaction(func(tx *gorm.DB) error {
		var draft models.Draft
		if err := tx.First(&draft, draftID).Error; err != nil {
			return fmt.Errorf("get draft: %w", err)
		}

		currentHTML := draft.HTMLContent

		// Ensure base snapshot exists
		if draft.CurrentEditSequence == 0 {
			var count int64
			tx.Model(&models.DraftEdit{}).Where("draft_id = ? AND sequence = 0", draftID).Count(&count)
			if count == 0 {
				tx.Create(&models.DraftEdit{
					DraftID: draftID, Sequence: 0, OpType: "base",
					HtmlSnapshot: currentHTML,
				})
			}
		}

		// Validate all ops first (dry run)
		testHTML := currentHTML
		for i, opRaw := range opsRaw {
			opMap, ok := opRaw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("op[%d]: must be an object", i)
			}
			oldStr, _ := opMap["old_string"].(string)
			if oldStr == "" {
				return fmt.Errorf("op[%d]: old_string is required", i)
			}
			if !strings.Contains(testHTML, oldStr) {
				return fmt.Errorf("op[%d]: old_string not found in current HTML", i)
			}
			newStr, _ := opMap["new_string"].(string)
			testHTML = strings.Replace(testHTML, oldStr, newStr, 1)
		}

		// All validated — apply for real
		currentHTML = draft.HTMLContent // reset
		for i, opRaw := range opsRaw {
			opMap := opRaw.(map[string]interface{})
			oldStr, _ := opMap["old_string"].(string)
			newStr, _ := opMap["new_string"].(string)
			desc, _ := opMap["description"].(string)

			currentHTML = strings.Replace(currentHTML, oldStr, newStr, 1)
			seq := draft.CurrentEditSequence + 1 + i
			tx.Create(&models.DraftEdit{
				DraftID: draftID, Sequence: seq, OpType: "search_replace",
				OldString: oldStr, NewString: newStr, Description: desc,
				HtmlSnapshot: currentHTML,
			})
		}

		newSeq := draft.CurrentEditSequence + len(opsRaw)
		tx.Model(&models.Draft{}).Where("id = ?", draftID).Updates(map[string]interface{}{
			"html_content": currentHTML, "current_edit_sequence": newSeq,
		})

		// Return result (outside transaction, so use a variable)
		// Actually this is inside the Transaction func, so we return nil on success
		return nil
	})
	// After transaction, read the final state
	// Hmm, this pattern doesn't let us return the result. Let me restructure.

	// Actually, let me use a different pattern:
	var result string
	err := e.db.Transaction(func(tx *gorm.DB) error {
		// ... same logic ...
		result = fmt.Sprintf(`{"applied": %d, "sequence": %d}`, len(opsRaw), newSeq)
		return nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}
```

Note: use a closure variable `result` to capture the return value from within the transaction.

**search_assets implementation:**
```go
func (e *AgentToolExecutor) searchAssets(ctx context.Context, params map[string]interface{}) (string, error) {
	projectID, ok := ctx.Value(projectIDKey).(uint)
	if !ok {
		return "", errors.New("project_id not found in context")
	}

	query, _ := params["query"].(string)
	assetType, _ := params["type"].(string)
	limit := 5
	if l, ok := getIntParam(params, "limit"); ok && l > 0 && l <= 20 {
		limit = l
	}

	db := e.db.Where("project_id = ?", projectID)
	if assetType != "" {
		db = db.Where("type = ?", assetType)
	}
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("label ILIKE ? OR content ILIKE ?", like, like)
	}

	var assets []models.Asset
	if err := db.Select("id, type, label, content").
		Order("created_at DESC").Limit(limit).Find(&assets).Error; err != nil {
		return "", fmt.Errorf("search assets: %w", err)
	}

	if len(assets) == 0 {
		return `{"results":[],"message":"no matching assets found"}`, nil
	}

	type assetResult struct {
		ID      uint   `json:"id"`
		Type    string `json:"type"`
		Label   string `json:"label"`
		Content string `json:"content"`
	}
	results := make([]assetResult, len(assets))
	for i, a := range assets {
		content := a.Content
		if len(content) > 2000 {
			content = content[:2000] + "...(truncated)"
		}
		results[i] = assetResult{ID: a.ID, Type: a.Type, Label: a.Label, Content: content}
	}

	data, _ := json.Marshal(map[string]interface{}{"results": results, "count": len(results)})
	return string(data), nil
}
```

Remove `VersionService`, `ExportService` interfaces and `getIntParam`, `getStringParam` helpers if still used by search_assets (keep `getIntParam` if used). Remove all 5 old tool implementations.

- [ ] **Step 6: Run all tool tests**

Run: `cd backend && go test ./internal/modules/agent/... -v -run "TestGetDraft|TestApplyEdits|TestSearchAssets"`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: rewrite tool_executor with 3 tools (get_draft, apply_edits, search_assets)"
```

---

## Task 4: Update Provider Adapter

**Files:**
- Modify: `backend/internal/modules/agent/provider.go:18-21`
- Modify test: `backend/internal/modules/agent/provider_test.go`

- [ ] **Step 1: Write failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -v -run TestMessage_ToolResultFields`
Expected: FAIL

- [ ] **Step 3: Update Message struct**

Replace in `provider.go`:

```go
type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Name       string `json:"name,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/agent/... -v -run TestMessage_ToolResultFields`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/provider.go backend/internal/modules/agent/provider_test.go
git commit -m "feat: add tool_call_id and name to Message struct for proper tool results"
```

---

## Task 5: Service Rewrite — System Prompt + Compaction + ReAct

**Files:**
- Rewrite: `backend/internal/modules/agent/service.go`
- Rewrite test: `backend/internal/modules/agent/service_test.go`

Remove `buildSystemPrompt`, `StreamChat` (legacy). Rewrite `StreamChatReAct`. Add compaction.

- [ ] **Step 1: Write fixed system prompt**

```go
const systemPromptV2 = `你是简历编辑专家。你可以像编辑代码一样精确编辑简历 HTML。

## 核心工具
- get_draft: 读取当前简历 HTML（可选 CSS selector 指定范围）
- apply_edits: 提交搜索替换操作修改简历（old_string 必须精确匹配）
- search_assets: 搜索用户资料库（旧简历、Git 摘要、笔记等）

## 工作流程
1. 先 get_draft 了解当前简历状态
2. 如需用户资料，用 search_assets 搜索
3. 用 apply_edits 提交精确修改
4. 修改后可用 get_draft 验证结果
5. 完成后用自然语言总结修改内容

## 编辑原则
- 每次只修改需要变化的部分，不要重写整个简历
- old_string 必须精确匹配，不匹配则修改会失败
- 失败时读取当前 HTML 找到正确内容后重试
- 保持 HTML 结构完整，确保渲染正确
- 注意 A4 页面尺寸限制
- 内容简洁专业，突出关键信息
`
```

- [ ] **Step 2: Update ChatService struct**

```go
type ChatService struct {
	db                *gorm.DB
	provider          ProviderAdapter
	toolExecutor      ToolExecutor
	recorder          *ThinkingRecorder
	maxIterations     int
	contextWindowSize int
}
```

- [ ] **Step 3: Implement compaction**

```go
func (s *ChatService) estimateTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) + len(m.Name) + len(m.ToolCallID)
	}
	return total / 2
}

func (s *ChatService) needsCompaction(messages []Message) bool {
	threshold := int(float64(s.contextWindowSize) * 0.8)
	return s.estimateTokens(messages) > threshold
}

func (s *ChatService) compactMessages(ctx context.Context, messages []models.AIMessage) ([]models.AIMessage, error) {
	if len(messages) <= 4 {
		return messages, nil
	}

	splitIdx := len(messages) - 4
	oldMessages := messages[:splitIdx]
	retained := messages[splitIdx:]

	var sb strings.Builder
	sb.WriteString("请将以下对话历史压缩为简洁摘要，保留：讨论了什么需求、做了哪些修改、当前简历状态。\n\n")
	for _, m := range oldMessages {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}

	var summary strings.Builder
	err := s.provider.StreamChat(ctx, []Message{
		{Role: "system", Content: "你是对话历史摘要助手。压缩为简洁摘要，保留关键信息。"},
		{Role: "user", Content: sb.String()},
	}, func(chunk string) error {
		summary.WriteString(chunk)
		return nil
	})
	if err != nil {
		return messages, fmt.Errorf("compaction failed: %w", err)
	}

	result := []models.AIMessage{{
		Role:    "system",
		Content: "[对话摘要] " + summary.String(),
		CreatedAt: oldMessages[0].CreatedAt,
	}}
	return append(result, retained...), nil
}
```

- [ ] **Step 4: Rewrite StreamChatReAct**

Key changes from current implementation:
1. Use `systemPromptV2` (no dynamic suffix)
2. Inject `draftIDKey`/`projectIDKey` into context (not system prompt)
3. Check compaction before loop
4. Pass `ToolCallID` and `Name` in tool result messages
5. Emit `edit` SSE event for `apply_edits` calls

```go
func (s *ChatService) StreamChatReAct(sessionID uint, userMessage string, sendEvent func(string)) error {
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return ErrSessionNotFound
	}

	s.db.Create(&models.AIMessage{SessionID: sessionID, Role: "user", Content: userMessage})

	history, err := s.loadMessages(sessionID)
	if err != nil {
		return err
	}

	// Compaction check
	apiHistory := make([]Message, len(history))
	for i, m := range history {
		apiHistory[i] = Message{Role: m.Role, Content: m.Content}
	}
	allMsgs := append([]Message{{Role: "system", Content: systemPromptV2}}, apiHistory...)
	if s.needsCompaction(allMsgs) {
		compacted, compactErr := s.compactMessages(context.Background(), history)
		if compactErr == nil {
			history = compacted
			s.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{})
			for _, m := range history {
				m.ID = 0
				m.SessionID = sessionID
				s.db.Create(&m)
			}
		}
	}

	ctx := context.WithValue(context.Background(), draftIDKey, session.DraftID)
	if session.ProjectID != nil {
		ctx = context.WithValue(ctx, projectIDKey, *session.ProjectID)
	}
	toolResults := make([]Message, 0)

	for stallCount := 0; stallCount < s.maxIterations; stallCount++ {
		apiMessages := []Message{{Role: "system", Content: systemPromptV2}}
		for _, m := range history {
			apiMessages = append(apiMessages, Message{Role: m.Role, Content: m.Content})
		}
		apiMessages = append(apiMessages, toolResults...)

		var fullText, thinkingAccum strings.Builder
		hadText, hadToolCalls := false, false

		err := s.provider.StreamChatReAct(ctx, apiMessages, s.toolExecutor.Tools(),
			func(chunk string) error {
				thinkingAccum.WriteString(chunk)
				data, _ := json.Marshal(map[string]string{"type": "thinking", "content": chunk})
				sendEvent(string(data))
				return nil
			},
			func(call ToolCallRequest) error {
				now := time.Now()
				toolCall := models.AIToolCall{
					SessionID: sessionID, ToolName: call.Name,
					Params: models.JSONB(call.Params), Status: "running", StartedAt: &now,
				}
				s.db.Create(&toolCall)

				callData, _ := json.Marshal(map[string]interface{}{
					"type": "tool_call", "name": call.Name, "params": call.Params,
				})
				sendEvent(string(callData))
				hadToolCalls = true

				result, execErr := s.toolExecutor.Execute(ctx, call.Name, call.Params)
				completedAt := time.Now()

				if execErr != nil {
					errMsg := execErr.Error()
					s.db.Model(&toolCall).Updates(map[string]interface{}{
						"status": "failed", "error": errMsg, "completed_at": completedAt,
					})
					failData, _ := json.Marshal(map[string]string{
						"type": "tool_result", "name": call.Name, "status": "failed",
					})
					sendEvent(string(failData))
					toolResults = append(toolResults, Message{
						Role: "tool", Content: fmt.Sprintf(`{"error":"%s"}`, errMsg),
						ToolCallID: call.ID, Name: call.Name,
					})
				} else {
					if call.Name == "apply_edits" {
						editData, _ := json.Marshal(map[string]interface{}{
							"type": "edit", "name": call.Name,
							"params": call.Params, "result": result,
						})
						sendEvent(string(editData))
					}

					var parsed map[string]interface{}
					var resultJSON *models.JSONB
					if json.Unmarshal([]byte(result), &parsed) == nil {
						j := models.JSONB(parsed)
						resultJSON = &j
					}
					updates := map[string]interface{}{"status": "completed", "completed_at": completedAt}
					if resultJSON != nil {
						updates["result"] = resultJSON
					}
					s.db.Model(&toolCall).Updates(updates)
					okData, _ := json.Marshal(map[string]string{
						"type": "tool_result", "name": call.Name, "status": "completed",
					})
					sendEvent(string(okData))
					toolResults = append(toolResults, Message{
						Role: "tool", Content: result,
						ToolCallID: call.ID, Name: call.Name,
					})
				}
				return nil
			},
			func(chunk string) error {
				hadText = true
				fullText.WriteString(chunk)
				data, _ := json.Marshal(map[string]string{"type": "text", "content": chunk})
				sendEvent(string(data))
				return nil
			},
		)
		if err != nil {
			return err
		}

		if hadText {
			thinkingStr := thinkingAccum.String()
			var thinkingPtr *string
			if thinkingStr != "" {
				thinkingPtr = &thinkingStr
			}
			s.db.Create(&models.AIMessage{
				SessionID: sessionID, Role: "assistant",
				Content: fullText.String(), Thinking: thinkingPtr,
			})
			sendEvent(`{"type":"done"}`)
			return nil
		}

		if hadToolCalls {
			stallCount--
		}
	}

	return ErrMaxIterations
}
```

- [ ] **Step 5: Remove legacy code**

Delete `buildSystemPrompt` function and `StreamChat` method entirely.

- [ ] **Step 6: Write compaction test**

```go
func TestNeedsCompaction(t *testing.T) {
	svc := &ChatService{contextWindowSize: 1000}

	short := append([]Message{{Role: "system", Content: "sys"}},
		Message{Role: "user", Content: strings.Repeat("a", 100)})
	assert.False(t, svc.needsCompaction(short))

	long := append([]Message{{Role: "system", Content: "sys"}},
		Message{Role: "user", Content: strings.Repeat("测", 1000)})
	assert.True(t, svc.needsCompaction(long))
}
```

- [ ] **Step 7: Run all agent tests**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: PASS (fix any compilation errors from changed interfaces)

- [ ] **Step 8: Commit**

```bash
git add backend/internal/modules/agent/service.go backend/internal/modules/agent/service_test.go
git commit -m "feat: rewrite service with fixed system prompt, compaction, and context-aware tools"
```

---

## Task 6: Undo/Redo Backend

**Files:**
- Modify: `backend/internal/modules/agent/service.go` (add EditService)
- Modify: `backend/internal/modules/agent/handler.go` (add endpoints)
- Modify: `backend/internal/modules/agent/routes.go` (add routes)
- Modify test: `backend/internal/modules/agent/handler_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestUndoRedo(t *testing.T) {
	db := testutil.SetupTestDB(t)

	draft := models.Draft{ProjectID: 1, HTMLContent: `<html><body><h1>V0</h1></body></html>`}
	require.NoError(t, db.Create(&draft).Error)

	db.Create(&models.DraftEdit{DraftID: draft.ID, Sequence: 0, OpType: "base", HtmlSnapshot: draft.HTMLContent})
	html1 := `<html><body><h1>V1</h1></body></html>`
	db.Create(&models.DraftEdit{DraftID: draft.ID, Sequence: 1, OpType: "search_replace", OldString: "V0", NewString: "V1", HtmlSnapshot: html1})
	html2 := `<html><body><h1>V2</h1></body></html>`
	db.Create(&models.DraftEdit{DraftID: draft.ID, Sequence: 2, OpType: "search_replace", OldString: "V1", NewString: "V2", HtmlSnapshot: html2})
	db.Model(&draft).Update("current_edit_sequence", 2)

	svc := NewEditService(db)

	result, err := svc.Undo(draft.ID)
	require.NoError(t, err)
	assert.Equal(t, html1, result)
	var d models.Draft
	db.First(&d, draft.ID)
	assert.Equal(t, html1, d.HTMLContent)
	assert.Equal(t, 1, d.CurrentEditSequence)

	result, err = svc.Redo(draft.ID)
	require.NoError(t, err)
	assert.Equal(t, html2, result)
	db.First(&d, draft.ID)
	assert.Equal(t, html2, d.HTMLContent)
	assert.Equal(t, 2, d.CurrentEditSequence)
}

func TestUndo_NoMoreEdits(t *testing.T) {
	db := testutil.SetupTestDB(t)
	draft := models.Draft{ProjectID: 1, HTMLContent: "test"}
	require.NoError(t, db.Create(&draft).Error)

	svc := NewEditService(db)
	_, err := svc.Undo(draft.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no more edits to undo")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -v -run TestUndoRedo`
Expected: FAIL (EditService not defined)

- [ ] **Step 3: Implement EditService**

Add to `service.go`:

```go
type EditService struct {
	db *gorm.DB
}

func NewEditService(db *gorm.DB) *EditService {
	return &EditService{db: db}
}

func (s *EditService) Undo(draftID uint) (string, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}
	if draft.CurrentEditSequence <= 0 {
		return "", errors.New("no more edits to undo")
	}

	targetSeq := draft.CurrentEditSequence - 1
	var edit models.DraftEdit
	if err := s.db.Where("draft_id = ? AND sequence = ?", draftID, targetSeq).First(&edit).Error; err != nil {
		return "", fmt.Errorf("get snapshot: %w", err)
	}

	s.db.Model(&draft).Updates(map[string]interface{}{
		"html_content": edit.HtmlSnapshot, "current_edit_sequence": targetSeq,
	})
	return edit.HtmlSnapshot, nil
}

func (s *EditService) Redo(draftID uint) (string, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}

	nextSeq := draft.CurrentEditSequence + 1
	var edit models.DraftEdit
	if err := s.db.Where("draft_id = ? AND sequence = ?", draftID, nextSeq).First(&edit).Error; err != nil {
		return "", errors.New("no more edits to redo")
	}

	s.db.Model(&draft).Updates(map[string]interface{}{
		"html_content": edit.HtmlSnapshot, "current_edit_sequence": nextSeq,
	})
	return edit.HtmlSnapshot, nil
}
```

- [ ] **Step 4: Add handler methods**

Add to `handler.go`:

```go
func (h *Handler) Undo(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("draft_id"), 10, 64)
	if err != nil {
		response.Error(c, 40001, "invalid draft_id")
		return
	}
	html, err := h.editSvc.Undo(uint(draftID))
	if err != nil {
		response.Error(c, 40401, err.Error())
		return
	}
	response.Success(c, gin.H{"html_content": html})
}

func (h *Handler) Redo(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("draft_id"), 10, 64)
	if err != nil {
		response.Error(c, 40001, "invalid draft_id")
		return
	}
	html, err := h.editSvc.Redo(uint(draftID))
	if err != nil {
		response.Error(c, 40401, err.Error())
		return
	}
	response.Success(c, gin.H{"html_content": html})
}
```

Update Handler struct:

```go
type Handler struct {
	sessionSvc *SessionService
	chatSvc    *ChatService
	editSvc    *EditService
}

func NewHandler(sessionSvc *SessionService, chatSvc *ChatService, editSvc *EditService) *Handler {
	return &Handler{sessionSvc: sessionSvc, chatSvc: chatSvc, editSvc: editSvc}
}
```

- [ ] **Step 5: Add routes**

In `routes.go`, add after existing routes:

```go
rg.POST("/drafts/:draft_id/undo", h.Undo)
rg.POST("/drafts/:draft_id/redo", h.Redo)
```

- [ ] **Step 6: Run tests**

Run: `cd backend && go test ./internal/modules/agent/... -v -run "TestUndoRedo|TestUndo_"`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/modules/agent/service.go backend/internal/modules/agent/handler.go backend/internal/modules/agent/routes.go backend/internal/modules/agent/handler_test.go
git commit -m "feat: add undo/redo API with HTML snapshot restoration"
```

---

## Task 7: Routes + main.go + Env Config

**Files:**
- Modify: `backend/internal/modules/agent/routes.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Update RegisterRoutes**

Remove `versionSvc` and `exportSvc` parameters. Add `CONTEXT_WINDOW_SIZE` env:

```go
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	sessionSvc := NewSessionService(db)

	var provider ProviderAdapter
	if os.Getenv("USE_MOCK") == "true" {
		provider = NewMockAdapter()
	} else {
		provider = NewOpenAIAdapter(
			os.Getenv("AI_API_URL"),
			os.Getenv("AI_API_KEY"),
			os.Getenv("AI_MODEL"),
		)
	}

	maxIterations := 10
	if v := os.Getenv("AGENT_MAX_ITERATIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxIterations = n
		}
	}

	contextWindowSize := 128000
	if v := os.Getenv("CONTEXT_WINDOW_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			contextWindowSize = n
		}
	}

	toolExecutor := NewAgentToolExecutor(db)
	editSvc := NewEditService(db)
	chatSvc := NewChatService(db, provider, toolExecutor, maxIterations)
	chatSvc.contextWindowSize = contextWindowSize

	h := NewHandler(sessionSvc, chatSvc, editSvc)

	rg.POST("/sessions", h.CreateSession)
	rg.GET("/sessions", h.ListSessions)
	rg.GET("/sessions/:session_id", h.GetSession)
	rg.DELETE("/sessions/:session_id", h.DeleteSession)
	rg.POST("/sessions/:session_id/chat", h.Chat)
	rg.GET("/sessions/:session_id/history", h.GetHistory)
	rg.POST("/drafts/:draft_id/undo", h.Undo)
	rg.POST("/drafts/:draft_id/redo", h.Redo)
}
```

- [ ] **Step 2: Update main.go call**

Change the agent registration in `main.go` from:

```go
agent.RegisterRoutes(authed, db, versionSvc, exportSvc)
```

To:

```go
agent.RegisterRoutes(authed, db)
```

- [ ] **Step 3: Run all backend tests**

Run: `cd backend && go test ./... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/modules/agent/routes.go backend/cmd/server/main.go
git commit -m "feat: update routes and env config for agent v2"
```

---

## Task 8: Frontend — TipTap Diff Extension

**Files:**
- Create: `frontend/workbench/src/components/editor/extensions/ai-diff.ts`
- Create test: `frontend/workbench/tests/AiDiffExtension.test.ts`
- Modify: `frontend/workbench/src/styles/editor.css`

- [ ] **Step 1: Write failing test**

```typescript
import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Deletion, Insertion } from '@/components/editor/extensions/ai-diff'

function createEditor(content: string) {
  return new Editor({ extensions: [StarterKit, Deletion, Insertion], content })
}

describe('ai-diff extension', () => {
  it('parses <del> as deletion mark', () => {
    const editor = createEditor('<p>hello <del>world</del> there</p>')
    const json = editor.getJSON()
    const para = json.content[0] as any
    const hasDel = para.content?.some(
      (n: any) => n.marks?.some((m: any) => m.type === 'deletion')
    )
    expect(hasDel).toBe(true)
  })

  it('parses <ins> as insertion mark', () => {
    const editor = createEditor('<p>hello <ins>world</ins> there</p>')
    const json = editor.getJSON()
    const para = json.content[0] as any
    const hasIns = para.content?.some(
      (n: any) => n.marks?.some((m: any) => m.type === 'insertion')
    )
    expect(hasIns).toBe(true)
  })

  it('serializes deletion back to <del>', () => {
    const editor = createEditor('<p><del>removed</del></p>')
    expect(editor.getHTML()).toContain('<del>')
  })

  it('serializes insertion back to <ins>', () => {
    const editor = createEditor('<p><ins>added</ins></p>')
    expect(editor.getHTML()).toContain('<ins>')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/AiDiffExtension.test.ts`
Expected: FAIL (module not found)

- [ ] **Step 3: Implement ai-diff.ts**

```typescript
import { Mark, markInputRule, markPasteRule } from '@tiptap/core'

export const Deletion = Mark.create({
  name: 'deletion',
  parseHTML() { return [{ tag: 'del' }] },
  renderHTML({ HTMLAttributes }) { return ['del', HTMLAttributes, 0] },
  addInputRules() {
    return [markInputRule({ find: /(?:^|\s)~~((?:[^~]+))~~$/, type: this.type })]
  },
  addPasteRules() {
    return [markPasteRule({ find: /(?:^|\s)~~((?:[^~]+))~~$/g, type: this.type })]
  },
})

export const Insertion = Mark.create({
  name: 'insertion',
  parseHTML() { return [{ tag: 'ins' }] },
  renderHTML({ HTMLAttributes }) { return ['ins', HTMLAttributes, 0] },
  addInputRules() {
    return [markInputRule({ find: /\+\+((?:[^+]+))\+\+$/, type: this.type })]
  },
  addPasteRules() {
    return [markPasteRule({ find: /\+\+((?:[^+]+))\+\+$/g, type: this.type })]
  },
})
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/AiDiffExtension.test.ts`
Expected: PASS

- [ ] **Step 5: Add diff CSS**

Add to `src/styles/editor.css`:

```css
.ProseMirror del {
  text-decoration: line-through;
  color: #dc2626;
  background-color: #fef2f2;
  border-radius: 2px;
}

.ProseMirror ins {
  text-decoration: none;
  color: #16a34a;
  background-color: #f0fdf4;
  border-radius: 2px;
}
```

- [ ] **Step 6: Commit**

```bash
git add frontend/workbench/src/components/editor/extensions/ai-diff.ts \
       frontend/workbench/tests/AiDiffExtension.test.ts \
       frontend/workbench/src/styles/editor.css
git commit -m "feat: add TipTap diff marks for inline edit visualization"
```

---

## Task 9: Frontend — API Client Updates

**Files:**
- Modify: `frontend/workbench/src/lib/api-client.ts`

- [ ] **Step 1: Add undo/redo API and edit types**

Add to `agentApi` object:

```typescript
export async function undoDraft(draftId: number) {
  return request<{ html_content: string }>(`/ai/drafts/${draftId}/undo`, { method: 'POST' })
}

export async function redoDraft(draftId: number) {
  return request<{ html_content: string }>(`/ai/drafts/${draftId}/redo`, { method: 'POST' })
}
```

Add type for edit events (near existing AI types):

```typescript
export interface PendingEdit {
  old_string: string
  new_string: string
  description?: string
}

export interface ToolCallEntry {
  name: string
  status: 'running' | 'completed' | 'failed'
  params?: Record<string, unknown>
  result?: string
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/workbench/src/lib/api-client.ts
git commit -m "feat: add undo/redo API and edit event types"
```

---

## Task 10: Frontend — Enhanced ChatPanel

**Files:**
- Create: `frontend/workbench/src/components/chat/ToolCallLog.tsx`
- Create: `frontend/workbench/src/components/chat/AiPlan.tsx`
- Rewrite: `frontend/workbench/src/components/chat/ChatPanel.tsx`
- Delete: `frontend/workbench/src/components/chat/HtmlPreview.tsx`
- Update test: `frontend/workbench/tests/ChatPanel.test.tsx`

- [ ] **Step 1: Create ToolCallLog**

```tsx
import type { ToolCallEntry } from '@/lib/api-client'

const TOOL_LABELS: Record<string, string> = {
  get_draft: '读取简历',
  apply_edits: '应用修改',
  search_assets: '搜索资料',
}

interface Props {
  calls: ToolCallEntry[]
}

export function ToolCallLog({ calls }: Props) {
  if (calls.length === 0) return null
  return (
    <div className="space-y-1 text-xs text-gray-500 mt-1">
      {calls.map((call, i) => (
        <details key={i} className="group">
          <summary className="cursor-pointer hover:text-gray-700 flex items-center gap-1">
            <span>{call.status === 'running' ? '...' : call.status === 'completed' ? 'OK' : 'FAIL'}</span>
            <span>{TOOL_LABELS[call.name] || call.name}</span>
          </summary>
          <pre className="mt-1 p-2 bg-gray-50 rounded text-[10px] overflow-auto max-h-32">
            {JSON.stringify({ params: call.params, result: call.result }, null, 2)}
          </pre>
        </details>
      ))}
    </div>
  )
}
```

- [ ] **Step 2: Create AiPlan**

```tsx
interface Props {
  content: string
}

export function AiPlan({ content }: Props) {
  if (!content) return null
  return (
    <details className="text-xs mt-1">
      <summary className="cursor-pointer text-gray-500 hover:text-gray-700 font-medium">
        AI 规划
      </summary>
      <div className="mt-1 p-2 bg-amber-50 border border-amber-200 rounded text-gray-700 whitespace-pre-wrap">
        {content}
      </div>
    </details>
  )
}
```

- [ ] **Step 3: Rewrite ChatPanel**

Key changes:
1. Remove `HTML_START`/`HTML_END` constants and marker extraction logic
2. Remove `HtmlPreview` usage
3. New props: `onApplyDiffHTML?: (diffHTML: string) => void`
4. Add state: `toolCalls`, `pendingEdits`
5. Handle new `edit` SSE event type
6. Render `ToolCallLog` and `AiPlan` components
7. On `done` event with pending edits: generate diff HTML with `<del>`/`<ins>` tags

New SSE handling logic (in handleSend):

```typescript
const [toolCalls, setToolCalls] = useState<ToolCallEntry[]>([])
const [pendingEdits, setPendingEdits] = useState<PendingEdit[]>([])
const editorHTMLRef = useRef<string>('')

// In the SSE data parsing switch:
case 'tool_call':
  setToolCalls(prev => [...prev, { name: data.name, status: 'running', params: data.params }])
  break
case 'tool_result':
  setToolCalls(prev => {
    const updated = [...prev]
    const last = updated[updated.length - 1]
    if (last) updated[updated.length - 1] = { ...last, status: data.status }
    return updated
  })
  break
case 'edit':
  setPendingEdits(prev => [...prev, ...(data.params?.ops || [])])
  break
case 'done':
  if (pendingEdits.length > 0 && onApplyDiffHTML) {
    const currentHTML = editorHTMLRef.current
    let diffHTML = currentHTML
    for (const edit of pendingEdits) {
      diffHTML = diffHTML.replace(
        edit.old_string,
        `<del>${edit.old_string}</del><ins>${edit.new_string}</ins>`
      )
    }
    onApplyDiffHTML(diffHTML)
    setPendingEdits([])
  }
  break
```

Props interface:

```typescript
interface Props {
  draftId: number
  onApplyDiffHTML?: (diffHTML: string) => void
}
```

Message rendering: remove `getDisplayText()` calls (no more HTML markers). Render text content directly.

Remove the "Apply" button and HTML preview section entirely.

- [ ] **Step 4: Update ChatPanel tests**

Remove tests for HTML marker extraction and HtmlPreview. Add tests for:
- Tool call log renders on tool_call/tool_result events
- edit events accumulate pending edits
- onApplyDiffHTML called on done with correct diff HTML

- [ ] **Step 5: Run tests**

Run: `cd frontend/workbench && bunx vitest run tests/ChatPanel.test.tsx`
Expected: PASS

- [ ] **Step 6: Delete HtmlPreview.tsx**

```bash
rm frontend/workbench/src/components/chat/HtmlPreview.tsx
```

- [ ] **Step 7: Commit**

```bash
git add frontend/workbench/src/components/chat/ frontend/workbench/tests/ChatPanel.test.tsx
git commit -m "feat: rewrite ChatPanel with tool logs and inline diff support"
```

---

## Task 11: Frontend — Undo/Redo + EditorPage Integration

**Files:**
- Create: `frontend/workbench/src/components/editor/UndoRedoBar.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`

- [ ] **Step 1: Create UndoRedoBar**

```tsx
import { useState } from 'react'
import { undoDraft, redoDraft } from '@/lib/api-client'

interface Props {
  draftId: number
  onRestore: (html: string) => void
}

export function UndoRedoBar({ draftId, onRestore }: Props) {
  const [loading, setLoading] = useState(false)

  const handleUndo = async () => {
    setLoading(true)
    try {
      const { data } = await undoDraft(draftId)
      onRestore(data.html_content)
    } finally { setLoading(false) }
  }

  const handleRedo = async () => {
    setLoading(true)
    try {
      const { data } = await redoDraft(draftId)
      onRestore(data.html_content)
    } finally { setLoading(false) }
  }

  return (
    <div className="flex gap-1">
      <button onClick={handleUndo} disabled={loading}
        className="px-2 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded disabled:opacity-50">
        Undo
      </button>
      <button onClick={handleRedo} disabled={loading}
        className="px-2 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded disabled:opacity-50">
        Redo
      </button>
    </div>
  )
}
```

- [ ] **Step 2: Wire into EditorPage**

In EditorPage.tsx:
1. Add `Deletion` and `Insertion` to editor extensions
2. Replace `onApplyHTML` prop on ChatPanel with `onApplyDiffHTML`
3. Add `UndoRedoBar` to the ActionBar area
4. Remove old `HtmlPreview` import if any

```typescript
import { Deletion, Insertion } from '@/components/editor/extensions/ai-diff'
import { UndoRedoBar } from '@/components/editor/UndoRedoBar'

// In useEditor extensions:
extensions: [
  StarterKit,
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
  Deletion,
  Insertion,
],

// In render, replace ChatPanel props:
<ChatPanel
  draftId={Number(draftId)}
  onApplyDiffHTML={(diffHTML) => editor?.commands.setContent(diffHTML)}
/>

// Add UndoRedoBar (e.g., inside ActionBar):
<UndoRedoBar
  draftId={Number(draftId)}
  onRestore={(html) => editor?.commands.setContent(html)}
/>
```

- [ ] **Step 3: Run all frontend tests**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/workbench/src/components/editor/UndoRedoBar.tsx frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat: add undo/redo bar and wire inline diff into editor"
```

---

## Task 12: Integration Test with Docker Compose

**Files:**
- Create: `backend/internal/modules/agent/integration_test.go`

- [ ] **Step 1: Start environment**

Run: `docker compose up -d postgres`

- [ ] **Step 2: Write integration test**

```go
//go:build integration

package agent

import (
	"context"
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/database"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullFlow_Integration(t *testing.T) {
	db := database.Connect()
	database.Migrate(db)

	project := models.Project{Title: "Integration Test"}
	require.NoError(t, db.Create(&project).Error)

	draft := models.Draft{
		ProjectID:   project.ID,
		HTMLContent: "<html><body><h1>Old</h1><p>Content</p></body></html>",
	}
	require.NoError(t, db.Create(&draft).Error)

	ctx := context.WithValue(context.Background(), draftIDKey, draft.ID)
	ctx = context.WithValue(ctx, projectIDKey, project.ID)
	executor := NewAgentToolExecutor(db)

	// 1. get_draft
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)
	assert.Contains(t, result, "Old")

	// 2. get_draft with selector
	result, err = executor.Execute(ctx, "get_draft", map[string]interface{}{"selector": "h1"})
	require.NoError(t, err)
	assert.Contains(t, result, "Old")

	// 3. apply_edits
	result, err = executor.Execute(ctx, "apply_edits", map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"old_string": "Old", "new_string": "New", "description": "title"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"applied": 1`)

	var updated models.Draft
	db.First(&updated, draft.ID)
	assert.Contains(t, updated.HTMLContent, "New")
	assert.Equal(t, 1, updated.CurrentEditSequence)

	// 4. search_assets (empty)
	result, err = executor.Execute(ctx, "search_assets", map[string]interface{}{"query": "nothing"})
	require.NoError(t, err)

	// 5. undo
	editSvc := NewEditService(db)
	html, err := editSvc.Undo(draft.ID)
	require.NoError(t, err)
	assert.Contains(t, html, "Old")

	db.First(&updated, draft.ID)
	assert.Equal(t, 0, updated.CurrentEditSequence)

	// 6. redo
	html, err = editSvc.Redo(draft.ID)
	require.NoError(t, err)
	assert.Contains(t, html, "New")
}
```

- [ ] **Step 3: Run integration test**

Run: `cd backend && go test ./internal/modules/agent/... -v -tags=integration -run TestFullFlow`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/modules/agent/integration_test.go
git commit -m "test: add integration test for agent v2 full flow"
```
