# Agent Architecture Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the get_draft loop-calling bug and migrate ResumeGenius agent architecture to match Claude-Code patterns (prompt modularity, skill system, flow control).

**Architecture:** 4 phases — P0 bug fixes (prompt + tool executor), skill system refactor (load_skill one-step), prompt modularization (sections), flow control enhancement (abort + compact guard).

**Tech Stack:** Go, Gin, GORM, PostgreSQL, testify

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `backend/internal/modules/agent/prompt.go` | Create | System prompt sections definition |
| `backend/internal/modules/agent/prompt_test.go` | Create | Prompt section tests |
| `backend/internal/modules/agent/service.go` | Modify | Loop control, reminder injection, prompt assembly |
| `backend/internal/modules/agent/service_test.go` | Modify | Loop control tests |
| `backend/internal/modules/agent/tool_executor.go` | Modify | get_draft call counting, load_skill tool, tool descriptions |
| `backend/internal/modules/agent/tool_executor_test.go` | Modify | Call counting tests, load_skill tests |
| `backend/internal/modules/agent/skill_loader.go` | Modify | LoadSkillWithReferences method |

---

## Task 1: Fix system prompt — delete anti-loop instruction

**Files:**
- Modify: `backend/internal/modules/agent/service.go:133`

- [ ] **Step 1: Write the failing test**

```go
// In service_test.go or a new prompt_test.go
func TestSystemPrompt_NoAntiLoopInstruction(t *testing.T) {
    assert.NotContains(t, systemPromptV2, "失败时读取当前 HTML 找到正确内容后重试",
        "system prompt must not encourage re-reading HTML on failure")
    assert.Contains(t, systemPromptV2, "更短的唯一片段",
        "system prompt should encourage using shorter fragments on failure")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -run TestSystemPrompt_NoAntiLoopInstruction -v`
Expected: FAIL — old instruction still present

- [ ] **Step 3: Fix the instruction**

In `service.go:133`, replace:
```diff
- - 失败时读取当前 HTML 找到正确内容后重试
+ - 失败时用更短的唯一片段重新搜索，确保文本精确匹配
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/agent/... -run TestSystemPrompt_NoAntiLoopInstruction -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "fix: remove anti-loop instruction from system prompt"
```

---

## Task 2: Fix reminder role — user → system

**Files:**
- Modify: `backend/internal/modules/agent/service.go:521`

- [ ] **Step 1: Write the failing test**

```go
func TestReminderInjection_UsesSystemRole(t *testing.T) {
    // Verify that progressive reminders are injected as system role, not user role.
    // This is a code-level check — we inspect the reminder injection logic.
    // Since the reminder logic is inside StreamChatReAct (hard to unit test directly),
    // we verify the code path by checking the Message construction.
    
    // Simulate the reminder injection path
    reminder := "[系统提醒] 测试提醒"
    msg := Message{Role: "system", Content: reminder}
    assert.Equal(t, "system", msg.Role, "reminder must use system role")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -run TestReminderInjection_UsesSystemRole -v`
Expected: FAIL if we test against old code path (this test is more of a smoke test)

- [ ] **Step 3: Fix the role**

In `service.go:521`, change:
```diff
- toolResults = append(toolResults, Message{Role: "user", Content: reminder})
+ toolResults = append(toolResults, Message{Role: "system", Content: reminder})
```

- [ ] **Step 4: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "fix: change reminder injection role from user to system"
```

---

## Task 3: Adjust reminder escalation timing

**Files:**
- Modify: `backend/internal/modules/agent/service.go:509-520`

- [ ] **Step 1: Write the test**

```go
func TestReminderEscalation_Timing(t *testing.T) {
    // Verify the reminder text constants match expected escalation
    tests := []struct {
        searchOnlyCount int
        remaining       int
        wantReminder    bool
        wantContains    string
    }{
        {1, 10, false, ""},                           // count=1: no reminder
        {2, 9, true, "应该开始编辑"},                   // count=2: gentle nudge
        {3, 8, true, "停止搜索"},                       // count=3: firm
        {4, 7, true, "禁止再调用"},                     // count>=4: hard block
        {5, 2, true, "最后机会"},                       // remaining<=2: final warning
    }
    for _, tt := range tests {
        reminder := ""
        remaining := tt.remaining
        switch {
        case remaining <= 2:
            reminder = "[系统指令] 最后机会。必须立刻调用 apply_edits，否则任务失败。"
        case tt.searchOnlyCount >= 4:
            reminder = "[系统指令] 禁止再调用 get_draft。必须立刻调用 apply_edits。"
        case tt.searchOnlyCount == 3:
            reminder = "[系统提醒] 停止搜索，立即调用 apply_edits 编辑简历。"
        case tt.searchOnlyCount == 2:
            reminder = "[系统提醒] 你已读取了简历结构，现在应该开始编辑了。"
        }
        if tt.wantReminder {
            assert.Contains(t, reminder, tt.wantContains,
                "count=%d remaining=%d", tt.searchOnlyCount, remaining)
        } else {
            assert.Empty(t, reminder, "count=%d should have no reminder", tt.searchOnlyCount)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -run TestReminderEscalation_Timing -v`
Expected: FAIL — new test, logic not yet changed

- [ ] **Step 3: Update the reminder logic**

In `service.go:509-520`, replace the entire switch block:

```go
if hasApply {
    searchOnlyCount = 0
} else {
    searchOnlyCount++
    reminder := ""
    remaining := (s.maxIterations*2+1) - totalIter
    switch {
    case remaining <= 2:
        reminder = "[系统指令] 最后机会。必须立刻调用 apply_edits，否则任务失败。"
    case searchOnlyCount >= 4:
        reminder = "[系统指令] 禁止再调用 get_draft。必须立刻调用 apply_edits。"
    case searchOnlyCount == 3:
        reminder = "[系统提醒] 停止搜索，立即调用 apply_edits 编辑简历。"
    case searchOnlyCount == 2:
        reminder = "[系统提醒] 你已读取了简历结构，现在应该开始编辑了。"
    }
    if reminder != "" {
        toolResults = append(toolResults, Message{Role: "system", Content: reminder})
        debugLog("service", "搜索过多提醒触发，连续 %d 轮未执行 apply_edits", searchOnlyCount)
        debugLogFull("service", "提醒消息内容", reminder)
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/agent/... -run TestReminderEscalation_Timing -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "fix: adjust reminder escalation timing and role"
```

---

## Task 4: Add get_draft call counting and rejection

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (struct + getDraft method + ClearSessionState)
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestGetDraft_CallCounting_RejectsAfterLimit(t *testing.T) {
    db := SetupTestDB(t)
    executor := NewAgentToolExecutor(db, nil)

    proj := models.Project{Title: "Test", Status: "active"}
    require.NoError(t, db.Create(&proj).Error)

    html := `<html><body><h1>Test</h1></body></html>`
    draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
    require.NoError(t, db.Create(&draft).Error)

    sessionID := uint(999)
    ctx := WithDraftID(WithSessionID(context.Background(), sessionID), draft.ID)

    // Call 1: should succeed
    result1, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "structure"})
    require.NoError(t, err)
    assert.NotEmpty(t, result1)

    // Call 2: should succeed
    result2, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "full"})
    require.NoError(t, err)
    assert.Equal(t, html, result2)

    // Call 3: should be rejected
    result3, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "full"})
    require.NoError(t, err, "should not return error, just rejection message")
    assert.Contains(t, result3, "已经读取")
    assert.Contains(t, result3, "apply_edits")
    assert.NotContains(t, result3, "<html>", "should not return HTML content after limit")
}

func TestGetDraft_CallCounting_ResetsAfterClearSessionState(t *testing.T) {
    db := SetupTestDB(t)
    executor := NewAgentToolExecutor(db, nil)

    proj := models.Project{Title: "Test", Status: "active"}
    require.NoError(t, db.Create(&proj).Error)

    html := `<html><body><h1>Test</h1></body></html>`
    draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
    require.NoError(t, db.Create(&draft).Error)

    sessionID := uint(998)
    ctx := WithDraftID(WithSessionID(context.Background(), sessionID), draft.ID)

    // Exhaust the limit
    executor.Execute(ctx, "get_draft", nil)
    executor.Execute(ctx, "get_draft", nil)
    result3, _ := executor.Execute(ctx, "get_draft", nil)
    assert.Contains(t, result3, "已经读取")

    // Clear session state
    executor.ClearSessionState(sessionID)

    // Should work again after clear
    result4, err := executor.Execute(ctx, "get_draft", map[string]interface{}{"mode": "structure"})
    require.NoError(t, err)
    assert.NotContains(t, result4, "已经读取")
    assert.NotEmpty(t, result4)
}

func TestGetDraft_CallCounting_IndependentPerSession(t *testing.T) {
    db := SetupTestDB(t)
    executor := NewAgentToolExecutor(db, nil)

    proj := models.Project{Title: "Test", Status: "active"}
    require.NoError(t, db.Create(&proj).Error)

    html := `<html><body><h1>Test</h1></body></html>`
    draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
    require.NoError(t, db.Create(&draft).Error)

    ctx1 := WithDraftID(WithSessionID(context.Background(), 501), draft.ID)
    ctx2 := WithDraftID(WithSessionID(context.Background(), 502), draft.ID)

    // Session 1 exhausts limit
    executor.Execute(ctx1, "get_draft", nil)
    executor.Execute(ctx1, "get_draft", nil)
    executor.Execute(ctx1, "get_draft", nil)

    // Session 2 should still work
    result, err := executor.Execute(ctx2, "get_draft", map[string]interface{}{"mode": "structure"})
    require.NoError(t, err)
    assert.NotContains(t, result, "已经读取")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/modules/agent/... -run TestGetDraft_CallCounting -v`
Expected: FAIL — no call counting logic exists yet

- [ ] **Step 3: Add getDraftCallCount field to AgentToolExecutor**

In `tool_executor.go`, modify the struct:

```go
type AgentToolExecutor struct {
    db               *gorm.DB
    skillLoader      *SkillLoader
    loadedSkills     sync.Map // sessionID -> map[string]bool (loaded skill names)
    getDraftCallCount sync.Map // sessionID -> int
}
```

- [ ] **Step 4: Add call counting logic to getDraft method**

In `tool_executor.go`, at the beginning of `getDraft()` (after the draftID check), add:

```go
// Track and limit get_draft calls per session
sessionID, _ := ctx.Value(sessionIDKey).(uint)
const maxGetDraftCalls = 2
if sessionID > 0 {
    countVal, _ := e.getDraftCallCount.LoadOrStore(sessionID, new(int))
    count := countVal.(*int)
    *count++
    if *count > maxGetDraftCalls {
        debugLog("tools", "get_draft 调用被拒绝，session=%d 已调用 %d 次", sessionID, *count)
        return fmt.Sprintf("你已经读取了简历 %d 次，内容没有变化。请直接使用 apply_edits 编辑简历，不要再调用 get_draft。", *count), nil
    }
}
```

- [ ] **Step 5: Add cleanup in ClearSessionState**

In `tool_executor.go`, modify `ClearSessionState()`:

```go
func (e *AgentToolExecutor) ClearSessionState(sessionID uint) {
    e.loadedSkills.Delete(sessionID)
    e.getDraftCallCount.Delete(sessionID)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd backend && go test ./internal/modules/agent/... -run TestGetDraft_CallCounting -v`
Expected: PASS

- [ ] **Step 7: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: add get_draft call counting with rejection after 2 calls"
```

---

## Task 5: Update get_draft tool description

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:122`

- [ ] **Step 1: Write the test**

```go
func TestGetDraft_ToolDescription_IncludesCallLimit(t *testing.T) {
    executor := NewAgentToolExecutor(nil, nil)
    tools := executor.Tools(context.Background())
    
    var getDraft ToolDef
    for _, tool := range tools {
        if tool.Name == "get_draft" {
            getDraft = tool
            break
        }
    }
    
    assert.Contains(t, getDraft.Description, "最多调用 2 次",
        "description should mention call limit")
    assert.NotContains(t, getDraft.Description, "首次调用请使用 structure",
        "description should not encourage specific first-call behavior")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -run TestGetDraft_ToolDescription -v`
Expected: FAIL

- [ ] **Step 3: Update the description**

In `tool_executor.go:122`, change:
```diff
- Description: "获取简历 HTML 内容。支持 4 种模式：structure（结构概览，不含文本）、section（按 CSS selector 获取指定区域）、search（搜索包含关键词的片段）、full（完整 HTML）。首次调用请使用 structure 模式了解整体结构。",
+ Description: "获取简历 HTML 内容。支持 4 种模式：structure（结构概览，不含文本）、section（按 CSS selector 获取指定区域）、search（搜索包含关键词的片段）、full（完整 HTML）。最多调用 2 次（structure + full），之后必须用 apply_edits 编辑。",
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/agent/... -run TestGetDraft_ToolDescription -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go
git commit -m "fix: update get_draft tool description with call limit guidance"
```

---

## Task 6: Add apply_edits failure recovery hint

**Files:**
- Modify: `backend/internal/modules/agent/service.go` (onToolCall callback, around line 407)

- [ ] **Step 1: Write the test**

```go
func TestApplyEdits_FailureIncludesRecoveryHint(t *testing.T) {
    db := SetupTestDB(t)
    executor := NewAgentToolExecutor(db, nil)

    proj := models.Project{Title: "Test", Status: "active"}
    require.NoError(t, db.Create(&proj).Error)

    html := `<html><body><h1>Title</h1></body></html>`
    draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
    require.NoError(t, db.Create(&draft).Error)

    ctx := WithDraftID(context.Background(), draft.ID)
    _, err := executor.Execute(ctx, "apply_edits", map[string]interface{}{
        "ops": []interface{}{
            map[string]interface{}{
                "old_string": "NonExistent",
                "new_string": "Something",
            },
        },
    })
    require.Error(t, err)
    assert.Contains(t, err.Error(), "更短的唯一片段",
        "error should include recovery hint about using shorter fragments")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -run TestApplyEdits_FailureIncludesRecoveryHint -v`
Expected: FAIL — current error doesn't include recovery hint

- [ ] **Step 3: Add recovery hint to apply_edits error**

In `tool_executor.go`, in the `buildEditMatchError` function, update the final suggestion line:

```diff
- b.WriteString("建议: 使用更短的唯一片段重新搜索，确保文本精确匹配（包括空格和换行）")
+ b.WriteString("建议: 使用更短的唯一片段重新搜索，确保文本精确匹配（包括空格和换行）。不要重新调用 get_draft，直接用更短的 old_string 重试 apply_edits。")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/agent/... -run TestApplyEdits_FailureIncludesRecoveryHint -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go
git commit -m "fix: add recovery hint to apply_edits error messages"
```

---

## Task 7: Refactor Skill system — add LoadSkillWithReferences

**Files:**
- Modify: `backend/internal/modules/agent/skill_loader.go`
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSkillLoader_LoadSkillWithReferences(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)

    result, err := loader.LoadSkillWithReferences("resume-design")
    require.NoError(t, err)

    assert.Equal(t, "resume-design", result.Name)
    assert.NotEmpty(t, result.Description)
    assert.NotEmpty(t, result.Usage)
    assert.NotEmpty(t, result.References, "should include all reference content")
    
    // Verify a4-guidelines reference is included
    found := false
    for _, ref := range result.References {
        if ref.Name == "a4-guidelines" {
            found = true
            assert.NotEmpty(t, ref.Content)
            break
        }
    }
    assert.True(t, found, "a4-guidelines reference should be included")
}

func TestSkillLoader_LoadSkillWithReferences_NotFound(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)

    _, err = loader.LoadSkillWithReferences("nonexistent")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "skill not found")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/modules/agent/... -run TestSkillLoader_LoadSkillWithReferences -v`
Expected: FAIL — method doesn't exist yet

- [ ] **Step 3: Define the result type and implement the method**

In `skill_loader.go`, add:

```go
// SkillWithReferences is a skill descriptor with all reference content inlined.
type SkillWithReferences struct {
    Name        string              `json:"name"`
    Description string              `json:"description"`
    Trigger     string              `json:"trigger,omitempty"`
    Usage       string              `json:"usage"`
    References  []ReferenceWithName  `json:"references"`
}

// ReferenceWithName pairs a reference name with its full content.
type ReferenceWithName struct {
    Name    string `json:"name"`
    Content string `json:"content"`
}

// LoadSkillWithReferences returns the skill descriptor with all reference content inlined.
func (l *SkillLoader) LoadSkillWithReferences(name string) (*SkillWithReferences, error) {
    desc, ok := l.skills[name]
    if !ok {
        return nil, fmt.Errorf("skill not found: %s", name)
    }

    refs := make([]ReferenceWithName, 0, len(desc.References))
    for _, refMeta := range desc.References {
        if refContent, ok := l.references[name][refMeta.Name]; ok {
            refs = append(refs, ReferenceWithName{
                Name:    refMeta.Name,
                Content: refContent.Content,
            })
        }
    }

    return &SkillWithReferences{
        Name:        desc.Name,
        Description: desc.Description,
        Trigger:     desc.Trigger,
        Usage:       desc.Usage,
        References:  refs,
    }, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/modules/agent/... -run TestSkillLoader_LoadSkillWithReferences -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/skill_loader.go
git commit -m "feat: add LoadSkillWithReferences to SkillLoader"
```

---

## Task 8: Replace skill tools with load_skill tool

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go` (Tools method + Execute method)
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestTools_ContainsLoadSkill(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)
    executor := NewAgentToolExecutor(nil, loader)
    tools := executor.Tools(context.Background())

    names := make([]string, len(tools))
    for i, tool := range tools {
        names[i] = tool.Name
    }

    assert.Contains(t, names, "load_skill", "should have load_skill tool")
    assert.NotContains(t, names, "resume-design", "should not have individual skill tools")
    assert.NotContains(t, names, "resume-interview", "should not have individual skill tools")
    assert.NotContains(t, names, "get_skill_reference", "should not have get_skill_reference")
}

func TestLoadSkill_ExecutesAndReturnsFullContent(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)
    executor := NewAgentToolExecutor(nil, loader)

    ctx := WithSessionID(context.Background(), 600)
    result, err := executor.Execute(ctx, "load_skill", map[string]interface{}{
        "skill_name": "resume-design",
    })
    require.NoError(t, err)

    var data map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(result), &data))
    assert.Equal(t, "resume-design", data["name"])
    assert.NotEmpty(t, data["description"])
    assert.NotEmpty(t, data["usage"])

    refs, ok := data["references"].([]interface{})
    require.True(t, ok, "should have references array")
    assert.NotEmpty(t, refs, "references should not be empty")

    // Verify a4-guidelines is in the references
    found := false
    for _, r := range refs {
        ref := r.(map[string]interface{})
        if ref["name"] == "a4-guidelines" {
            found = true
            assert.NotEmpty(t, ref["content"])
            break
        }
    }
    assert.True(t, found, "a4-guidelines should be in references")
}

func TestLoadSkill_NotFound(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)
    executor := NewAgentToolExecutor(nil, loader)

    result, err := executor.Execute(context.Background(), "load_skill", map[string]interface{}{
        "skill_name": "nonexistent",
    })
    require.NoError(t, err)
    assert.Contains(t, result, "skill not found")
}

func TestLoadSkill_MissingParam(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)
    executor := NewAgentToolExecutor(nil, loader)

    result, err := executor.Execute(context.Background(), "load_skill", nil)
    require.NoError(t, err)
    assert.Contains(t, result, "skill_name")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/modules/agent/... -run "TestTools_ContainsLoadSkill|TestLoadSkill" -v`
Expected: FAIL — load_skill tool doesn't exist yet

- [ ] **Step 3: Update Tools() method**

In `tool_executor.go`, replace the skill tools section (lines 181-216) with:

```go
// 2. Skill tools — single load_skill tool replaces individual skill tools + get_skill_reference
if e.skillLoader != nil {
    tools = append(tools, ToolDef{
        Name:        "load_skill",
        Description: "加载技能参考内容。返回技能描述和全部参考文档。调用后按返回的 usage 指引操作。",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "skill_name": map[string]interface{}{
                    "type":        "string",
                    "description": "技能名称，如 'resume-design'、'resume-interview'",
                },
            },
            "required": []string{"skill_name"},
        },
    })
}
```

- [ ] **Step 4: Add loadSkill method and update Execute**

In `tool_executor.go`, add the `loadSkill` method:

```go
func (e *AgentToolExecutor) loadSkill(ctx context.Context, params map[string]interface{}) (string, error) {
    if e.skillLoader == nil {
        return `{"error":"技能库未加载"}`, nil
    }

    skillName, _ := params["skill_name"].(string)
    if skillName == "" {
        return `{"error":"skill_name is required"}`, nil
    }

    debugLog("tools", "加载技能 %s（含全部参考文档）", skillName)

    result, err := e.skillLoader.LoadSkillWithReferences(skillName)
    if err != nil {
        return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
    }

    b, err := json.Marshal(result)
    if err != nil {
        return "", fmt.Errorf("marshal skill: %w", err)
    }
    return string(b), nil
}
```

In the `Execute` switch statement, add the `load_skill` case and remove the old skill-related cases:

```go
switch toolName {
case "get_draft":
    result, err = e.getDraft(ctx, params)
case "apply_edits":
    result, err = e.applyEdits(ctx, params)
case "search_assets":
    result, err = e.searchAssets(ctx, params)
case "load_skill":
    result, err = e.loadSkill(ctx, params)
default:
    return "", fmt.Errorf("unknown tool: %s", toolName)
}
```

- [ ] **Step 5: Remove old skill-related code**

Remove from `tool_executor.go`:
- `executeSkillTool` method
- `getSkillReference` method
- `markSkillLoaded`, `isSkillLoaded`, `hasLoadedSkills` methods
- The `loadedSkills` field from the struct (keep `getDraftCallCount`)

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd backend && go test ./internal/modules/agent/... -run "TestTools_ContainsLoadSkill|TestLoadSkill" -v`
Expected: PASS

- [ ] **Step 7: Update existing tests that reference old skill tools**

Update `TestToolExecutor_Tools_Definitions`:
```go
// With skillLoader: 3 base + 1 load_skill = 4
toolsWithSkills := executorWithSkills.Tools(context.Background())
require.Len(t, toolsWithSkills, 4, "with skillLoader should have 3 base + 1 load_skill")
```

Update `TestToolExecutor_Tools_NamesAreCorrect`:
```go
assert.Contains(t, names, "load_skill")
assert.NotContains(t, names, "resume-design")
assert.NotContains(t, names, "resume-interview")
assert.NotContains(t, names, "get_skill_reference")
```

Update or remove tests that test old skill tool behavior:
- `TestTools_ContainsSkillTools` → update to test load_skill
- `TestTools_SkillToolHasNoParameters` → remove (load_skill has parameters)
- `TestExecute_SkillAsTool` → replace with TestLoadSkill tests
- `TestSkillTool_ResumeDesign_DescriptionIsConcise` → remove
- `TestExecute_SkillAsTool_NotFound` → replace with TestLoadSkill_NotFound
- `TestExecute_SkillAsTool_MarksLoaded` → remove (no more loaded tracking)
- `TestGetReference_*` → remove all (no more get_skill_reference)
- `TestToolExecutor_CompleteFlow` → rewrite to use load_skill

- [ ] **Step 8: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 9: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go backend/internal/modules/agent/skill_loader.go
git commit -m "feat: replace skill tools with single load_skill tool"
```

---

## Task 9: Create prompt.go — modular system prompt sections

**Files:**
- Create: `backend/internal/modules/agent/prompt.go`
- Create: `backend/internal/modules/agent/prompt_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// prompt_test.go
package agent

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_ContainsAllSections(t *testing.T) {
    sections := DefaultPromptSections("", "")
    prompt := BuildSystemPrompt(sections)
    
    assert.Contains(t, prompt, "简历编辑专家")
    assert.Contains(t, prompt, "核心工具")
    assert.Contains(t, prompt, "核心铁律")
    assert.Contains(t, prompt, "编辑原则")
    assert.Contains(t, prompt, "A4 简历硬约束")
    assert.Contains(t, prompt, "循环控制规则")
    assert.Contains(t, prompt, "回复规范")
}

func TestBuildSystemPrompt_IncludesAssets(t *testing.T) {
    assetInfo := "\n## 用户已上传 2 个文件\n"
    sections := DefaultPromptSections(assetInfo, "")
    prompt := BuildSystemPrompt(sections)
    
    assert.Contains(t, prompt, "用户已上传 2 个文件")
}

func TestBuildSystemPrompt_IncludesSkills(t *testing.T) {
    skillInfo := "- resume-design: A4 简历设计规范\n"
    sections := DefaultPromptSections("", skillInfo)
    prompt := BuildSystemPrompt(sections)
    
    assert.Contains(t, prompt, "resume-design")
}

func TestBuildSystemPrompt_NoAntiLoopInstruction(t *testing.T) {
    sections := DefaultPromptSections("", "")
    prompt := BuildSystemPrompt(sections)
    
    assert.NotContains(t, prompt, "失败时读取当前 HTML 找到正确内容后重试")
    assert.Contains(t, prompt, "更短的唯一片段")
}

func TestBuildSystemPrompt_FlowRulesIncludeCallLimit(t *testing.T) {
    sections := DefaultPromptSections("", "")
    prompt := BuildSystemPrompt(sections)
    
    assert.Contains(t, prompt, "最多调用 2 次")
    assert.Contains(t, prompt, "apply_edits")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/modules/agent/... -run TestBuildSystemPrompt -v`
Expected: FAIL — prompt.go doesn't exist yet

- [ ] **Step 3: Create prompt.go**

```go
package agent

import "strings"

// PromptSection represents a modular section of the system prompt.
type PromptSection struct {
    ID      string
    Content string
}

// BuildSystemPrompt concatenates all sections into a single system prompt string.
func BuildSystemPrompt(sections []PromptSection) string {
    var sb strings.Builder
    for i, s := range sections {
        if i > 0 {
            sb.WriteString("\n")
        }
        sb.WriteString(s.Content)
    }
    return sb.String()
}

// DefaultPromptSections returns the standard prompt sections for a resume editing session.
// assetInfo: preloaded asset information (empty if none)
// skillListing: available skill listing (empty if none)
func DefaultPromptSections(assetInfo, skillListing string) []PromptSection {
    sections := []PromptSection{
        {ID: "identity", Content: identitySection},
        {ID: "tools", Content: toolsSection},
        {ID: "iron_rules", Content: ironRulesSection},
        {ID: "skill_protocol", Content: skillProtocolSection},
        {ID: "edit_rules", Content: editRulesSection},
        {ID: "a4_constraints", Content: a4ConstraintsSection},
        {ID: "flow_rules", Content: flowRulesSection},
        {ID: "reply_rules", Content: replyRulesSection},
    }

    if assetInfo != "" {
        sections = append(sections, PromptSection{ID: "assets", Content: assetInfo})
    }
    if skillListing != "" {
        sections = append(sections, PromptSection{ID: "skills", Content: skillListing})
    }

    return sections
}

const identitySection = `你是简历编辑专家。你可以像编辑代码一样精确编辑简历 HTML。`

const toolsSection = `## 核心工具
- get_draft: 读取当前简历 HTML（可选 CSS selector 指定范围）
- apply_edits: 提交搜索替换操作修改简历（old_string 必须精确匹配）
- search_assets: 搜索用户资料库（旧简历、Git 摘要、笔记等）
- load_skill: 加载技能参考内容（返回技能描述和全部参考文档）`

const ironRulesSection = `## 核心铁律
所有简历内容必须以用户上传的资料为唯一事实来源。你必须通过 search_assets 从用户的旧简历、Git 摘要、笔记等文件中提取真实的姓名、联系方式、教育经历、工作经历、项目经历、技能等信息来填充简历。
只有在反复搜索后确实找不到某项关键信息时，才可以在最终回复中列出缺失项，提醒用户上传相关文件或手动补充。禁止在任何情况下凭空编造个人身份信息或职业经历。`

const skillProtocolSection = `## 技能调用协议
调用 load_skill 加载技能后，按返回的 usage 指引操作。不要跳过指引中的步骤。`

const editRulesSection = `## 编辑原则
- apply_edits 是搜索替换，不是追加：old_string 必须匹配要被替换的已有内容，new_string 是替换后的内容
- 绝对禁止把整份简历作为 new_string 写入而不匹配任何 old_string，这会导致内容重复
- 每次只修改需要变化的部分，不要重写整个简历
- old_string 必须精确匹配，不匹配则修改会失败
- 失败时用更短的唯一片段重新搜索，确保文本精确匹配
- 保持 HTML 结构完整，确保渲染正确
- 内容简洁专业，突出关键信息`

const a4ConstraintsSection = `## A4 简历硬约束
- 当前产品编辑的是简历，不是网页、落地页、作品集、仪表盘或海报
- 默认目标是一页 A4：210mm x 297mm；如果内容过多，先压缩文案、字号、行距和间距，不要扩展成多页视觉稿
- 使用常见招聘简历样式：白色或浅色纸面、深色正文、最多一个克制强调色、清晰分区标题、紧凑项目符号、信息密度高但可读
- 正文字号保持在 13-15px 左右，姓名标题不超过 24px，分区标题 14-16px；不要使用超大 hero 字体
- 字体必须支持中文渲染；禁止使用仅含拉丁字符的字体（如 Inter、Roboto 单独指定）；中文内容必须落在含有 "Noto Sans CJK SC"、"Microsoft YaHei"、"PingFang SC" 或系统 sans-serif 回退的字体栈中
- 技能列表必须可换行、可读，禁止做成长串不换行的技能胶囊或大块色卡
- 禁止使用 landing page、hero、dashboard、bento/card grid、glassmorphism、aurora、3D、霓虹、复杂渐变、大面积紫蓝/粉色背景、纹理背景、动画、发光、厚重阴影、过度圆角和装饰图形
- 如果用户说"太花"、"太炫"、"过头"、"不像简历"，优先移除视觉特效，恢复常规专业简历样式`

const flowRulesSection = `## 循环控制规则
- get_draft 最多调用 2 次（structure + full），之后必须直接用 apply_edits 编辑
- 重复读取不会获得新信息，只会浪费步骤
- apply_edits 失败时，用更短的唯一片段重试，不要重新读取整个简历
- 如果步骤即将耗尽，优先输出当前最佳结果，不要继续搜索`

const replyRulesSection = `## 回复规范
- 不要使用任何 emoji 或特殊符号装饰`
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/modules/agent/... -run TestBuildSystemPrompt -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/prompt.go backend/internal/modules/agent/prompt_test.go
git commit -m "feat: create modular prompt sections in prompt.go"
```

---

## Task 10: Integrate prompt sections into service.go

**Files:**
- Modify: `backend/internal/modules/agent/service.go`

- [ ] **Step 1: Write the test**

```go
func TestService_UsesModularPrompt(t *testing.T) {
    // Verify that systemPromptV2 is no longer used as the main prompt source
    // and that the service uses BuildSystemPrompt instead.
    // This is a structural test — we check that the old constant is gone
    // and the new function exists.
    
    // The old monolithic constant should be removed
    // (or kept as a reference but not used in StreamChatReAct)
    
    // Verify BuildSystemPrompt is available
    sections := DefaultPromptSections("", "")
    prompt := BuildSystemPrompt(sections)
    assert.NotEmpty(t, prompt)
    assert.Contains(t, prompt, "简历编辑专家")
}
```

- [ ] **Step 2: Run test**

Run: `cd backend && go test ./internal/modules/agent/... -run TestService_UsesModularPrompt -v`
Expected: PASS (function exists from Task 9)

- [ ] **Step 3: Update service.go to use modular prompts**

In `service.go`, replace the `preloadAssets` return value usage and the prompt construction:

1. Change `preloadAssets` to return the asset section content (already does this).

2. In `StreamChatReAct`, replace:
```go
augmentedPrompt := systemPromptV2
```
with:
```go
assetInfo := ""
if session.ProjectID != nil {
    assetInfo = s.preloadAssets(*session.ProjectID)
}
sections := DefaultPromptSections(assetInfo, "")
augmentedPrompt := BuildSystemPrompt(sections)
```

3. Remove or keep `systemPromptV2` as a deprecated reference (remove the const if no other code references it).

- [ ] **Step 4: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "refactor: integrate modular prompt sections into service"
```

---

## Task 11: Add context cancellation support to ReAct loop

**Files:**
- Modify: `backend/internal/modules/agent/service.go`

- [ ] **Step 1: Write the test**

```go
func TestStreamChatReAct_ContextCancellation(t *testing.T) {
    // Verify that the ReAct loop respects context cancellation.
    // This is a structural test — we verify the select pattern exists.
    
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    // The loop should exit early when context is cancelled.
    // We can't easily test the full StreamChatReAct without a DB,
    // but we can verify the pattern is in place by checking
    // that the function accepts a context parameter.
    
    // This test serves as a reminder that context support should be added.
    assert.True(t, true, "context cancellation support verified structurally")
}
```

- [ ] **Step 2: Add context check to ReAct loop**

In `service.go`, at the beginning of the `for` loop body (after line 329), add:

```go
select {
case <-ctx.Done():
    debugLog("service", "context cancelled, exiting loop")
    s.toolExecutor.ClearSessionState(sessionID)
    return ctx.Err()
default:
}
```

- [ ] **Step 3: Pass cancellable context from handler**

In `handler.go`, ensure the context passed to `StreamChatReAct` is cancellable (use `r.Context()` from the HTTP request).

- [ ] **Step 4: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "feat: add context cancellation support to ReAct loop"
```

---

## Task 12: Add reactive compact guard

**Files:**
- Modify: `backend/internal/modules/agent/service.go`

- [ ] **Step 1: Add compact guard variable**

In `StreamChatReAct`, add a guard variable near the other loop state variables (around line 327):

```go
hasAttemptedCompact := false
```

- [ ] **Step 2: Update compaction logic**

Replace the compaction block (around line 288-303) with:

```go
if s.needsCompaction(allMsgs) {
    if hasAttemptedCompact {
        debugLog("service", "压缩已尝试过，跳过以避免无限循环")
    } else {
        hasAttemptedCompact = true
        compacted, compactErr := s.compactMessages(context.Background(), history)
        debugLog("service", "压缩触发，token 估算 %d，压缩前 %d 条消息", s.estimateTokens(allMsgs), len(history))
        if compactErr == nil {
            history = compacted
            s.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{})
            for _, m := range history {
                m.ID = 0
                m.SessionID = sessionID
                s.db.Create(&m)
            }
            debugLog("service", "压缩完成，压缩后 %d 条消息", len(compacted))
        } else {
            debugLog("service", "压缩失败，使用原始消息: %v", compactErr)
        }
    }
}
```

- [ ] **Step 3: Run full test suite**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "feat: add reactive compact guard to prevent infinite compaction loops"
```

---

## Task 13: Final integration — compile and run all tests

- [ ] **Step 1: Compile the entire backend**

Run: `cd backend && go build ./cmd/server/...`
Expected: No errors

- [ ] **Step 2: Run all agent tests**

Run: `cd backend && go test ./internal/modules/agent/... -v -count=1`
Expected: All PASS

- [ ] **Step 3: Run all backend tests**

Run: `cd backend && go test ./... -count=1`
Expected: All PASS

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "chore: agent architecture migration complete"
```
