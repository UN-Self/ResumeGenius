# 技能即工具重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `load_skill`/`get_skill_reference` 从 AI 工具列表移除，让每个技能本身作为工具出现，子工具按需动态注入。

**Architecture:** `Tools(ctx)` 根据 session 已加载的技能动态返回工具列表。技能工具调用时系统自动加载描述文档并标记已加载，下一轮 `Tools()` 检测到已加载技能后注入 `get_skill_reference` 子工具。

**Tech Stack:** Go, Gin, GORM, OpenAI function calling, React, TypeScript

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `backend/internal/modules/agent/skill_loader.go` | Modify | 新增 `HasSkill()` 方法 |
| `backend/internal/modules/agent/tool_executor.go` | Modify | 核心重构：动态 `Tools(ctx)`、技能工具执行、删除 `load_skill` |
| `backend/internal/modules/agent/tool_executor_test.go` | Modify | 重写技能相关测试 |
| `backend/internal/modules/agent/service.go` | Modify | System Prompt 更新、`Tools()` 调用加 `ctx` |
| `backend/internal/modules/agent/service_test.go` | Modify | `MockToolExecutor.Tools()` 签名适配 |
| `backend/internal/modules/agent/provider.go` | Modify | `MockAdapter` 更新 |
| `frontend/workbench/src/components/chat/ToolCallLog.tsx` | Modify | 更新 `TOOL_META` |
| `frontend/workbench/src/components/chat/ChatPanel.tsx` | Modify | 更新 `ThinkingBubble` |

---

### Task 1: SkillLoader 新增 HasSkill 方法

**Files:**
- Modify: `backend/internal/modules/agent/skill_loader.go`
- Modify: `backend/internal/modules/agent/skill_loader_test.go`

- [ ] **Step 1: 写失败测试**

`skill_loader_test.go` 追加：

```go
func TestSkillLoader_HasSkill_Existing(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	assert.True(t, loader.HasSkill("resume-interview"))
	assert.True(t, loader.HasSkill("resume-design"))
}

func TestSkillLoader_HasSkill_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	assert.False(t, loader.HasSkill("nonexistent"))
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/agent/... -run TestSkillLoader_HasSkill -v
```

Expected: `undefined: loader.HasSkill`

- [ ] **Step 3: 实现 HasSkill**

`skill_loader.go` 在 `Skills()` 方法后追加：

```go
// HasSkill reports whether a skill with the given name exists.
func (l *SkillLoader) HasSkill(name string) bool {
	_, ok := l.skills[name]
	return ok
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -run TestSkillLoader_HasSkill -v
```

Expected: PASS

- [ ] **Step 5: 运行全部 SkillLoader 测试确认无回归**

```bash
cd backend && go test ./internal/modules/agent/... -run TestSkillLoader -v
```

Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add backend/internal/modules/agent/skill_loader.go backend/internal/modules/agent/skill_loader_test.go
git commit -m "feat: SkillLoader 新增 HasSkill 方法"
```

---

### Task 2: ToolExecutor 接口 Tools() 加 ctx 参数

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`
- Modify: `backend/internal/modules/agent/service.go`
- Modify: `backend/internal/modules/agent/service_test.go`

- [ ] **Step 1: 修改 ToolExecutor 接口**

`tool_executor.go` 第 57-62 行，改接口签名：

```go
// ToolExecutor defines the interface for executing AI tool calls.
type ToolExecutor interface {
	// Tools returns the list of tool definitions.
	Tools(ctx context.Context) []ToolDef
	// Execute runs a tool by name with the given parameters and returns the result as a JSON string.
	Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error)
}
```

- [ ] **Step 2: 修改 AgentToolExecutor.Tools() 签名**

`tool_executor.go` 第 77 行：

```go
// Tools returns the AI-callable tool definitions.
func (e *AgentToolExecutor) Tools(ctx context.Context) []ToolDef {
```

- [ ] **Step 3: 修改 service.go 调用处**

`service.go` 第 326-327 行：

```go
log.Printf("agent: iteration %d calling model with %d messages and %d tools", totalIter, len(apiMessages), len(s.toolExecutor.Tools(ctx)))
err := s.provider.StreamChatReAct(
    ctx,
    apiMessages,
    s.toolExecutor.Tools(ctx),
```

- [ ] **Step 4: 修改 MockToolExecutor 签名**

`service_test.go` 第 207-209 行：

```go
func (e *MockToolExecutor) Tools(_ context.Context) []ToolDef {
	return NewAgentToolExecutor(nil, nil).Tools(context.Background())
}
```

- [ ] **Step 5: 运行编译确认无错**

```bash
cd backend && go build ./...
```

Expected: 无错误

- [ ] **Step 6: 运行全部测试确认无回归**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS（注意：当前 `TestToolExecutor_Tools_NamesAreCorrect` 等测试会因 `Tools()` 签名变化而编译失败，需要在 Task 3 中修复）

- [ ] **Step 7: 修复因签名变化导致的测试编译错误**

`tool_executor_test.go` 中所有 `executor.Tools()` 改为 `executor.Tools(context.Background())`：

```go
// 第 21 行
tools := executor.Tools(context.Background())

// 第 39-40 行
executor := NewAgentToolExecutor(nil, nil)
tools := executor.Tools(context.Background())

// 第 58-59 行
executor := NewAgentToolExecutor(nil, nil)
tools := executor.Tools(context.Background())
```

- [ ] **Step 8: 运行全部测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS

- [ ] **Step 9: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/service.go backend/internal/modules/agent/service_test.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "refactor: Tools() 接口加 ctx 参数"
```

---

### Task 3: Tools() 动态生成 - 基础工具 + 技能工具

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 写失败测试 - 技能工具出现在工具列表中**

`tool_executor_test.go` 追加：

```go
func TestTools_ContainsSkillTools(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	tools := executor.Tools(context.Background())
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "resume-design")
	assert.Contains(t, names, "resume-interview")
}

func TestTools_SkillToolHasNoParameters(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	tools := executor.Tools(context.Background())
	toolByName := make(map[string]ToolDef)
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	designTool := toolByName["resume-design"]
	assert.NotEmpty(t, designTool.Description)
	props := designTool.Parameters["properties"].(map[string]interface{})
	assert.Empty(t, props, "skill tool should have no parameters")
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/agent/... -run TestTools_ContainsSkillTools -v
```

Expected: FAIL（当前 Tools() 返回固定 5 个工具，不含技能工具）

- [ ] **Step 3: 重构 Tools() 方法**

`tool_executor.go` 重写 `Tools()` 方法：

```go
// Tools returns the AI-callable tool definitions.
func (e *AgentToolExecutor) Tools(ctx context.Context) []ToolDef {
	// 1. 基础工具（固定）
	tools := []ToolDef{
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
							"required": []string{"old_string", "new_string"},
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

	// 2. 技能工具（从 SkillLoader 自动生成，无参数）
	if e.skillLoader != nil {
		for _, skill := range e.skillLoader.Skills() {
			tools = append(tools, ToolDef{
				Name:        skill.Name,
				Description: skill.Description,
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			})
		}
	}

	// 3. 子工具（根据已加载的技能动态注入）
	sessionID, _ := ctx.Value(sessionIDKey).(uint)
	if e.hasLoadedSkills(sessionID) {
		tools = append(tools, ToolDef{
			Name:        "get_skill_reference",
			Description: "获取技能库中指定岗位的面经内容或设计规范。必须先调用对应技能工具。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"skill_name": map[string]interface{}{
						"type":        "string",
						"description": "技能名称，如 'resume-interview'、'resume-design'",
					},
					"reference_name": map[string]interface{}{
						"type":        "string",
						"description": "reference 名称，如 'test-engineer'、'a4-guidelines'",
					},
				},
				"required": []string{"skill_name", "reference_name"},
			},
		})
	}

	return tools
}
```

- [ ] **Step 4: 新增 hasLoadedSkills 辅助方法**

`tool_executor.go` 在 `clearSessionSkills` 方法后追加：

```go
func (e *AgentToolExecutor) hasLoadedSkills(sessionID uint) bool {
	val, ok := e.loadedSkills.Load(sessionID)
	if !ok {
		return false
	}
	m := val.(*sync.Map)
	has := false
	m.Range(func(_, _ interface{}) bool {
		has = true
		return false
	})
	return has
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -run "TestTools_" -v
```

Expected: 全部 PASS

- [ ] **Step 6: 修复因工具数量变化导致的旧测试**

`tool_executor_test.go` 中 `TestToolExecutor_Tools_Definitions` 需要更新工具数量断言。因为现在基础工具 3 个 + 技能工具 2 个 = 5 个（无加载技能时不含 get_skill_reference）：

```go
func TestToolExecutor_Tools_Definitions(t *testing.T) {
	executor := NewAgentToolExecutor(nil, nil)
	tools := executor.Tools(context.Background())
	require.Len(t, tools, 3, "without skillLoader should have 3 base tools")

	// With skillLoader
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executorWithSkills := NewAgentToolExecutor(nil, loader)
	toolsWithSkills := executorWithSkills.Tools(context.Background())
	require.Len(t, toolsWithSkills, 5, "with skillLoader should have 3 base + 2 skill tools")
}
```

`TestToolExecutor_Tools_NamesAreCorrect` 更新：

```go
func TestToolExecutor_Tools_NamesAreCorrect(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)
	tools := executor.Tools(context.Background())

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "get_draft")
	assert.Contains(t, names, "apply_edits")
	assert.Contains(t, names, "search_assets")
	assert.Contains(t, names, "resume-design")
	assert.Contains(t, names, "resume-interview")
	assert.NotContains(t, names, "load_skill", "load_skill should be removed")
	assert.NotContains(t, names, "get_skill_reference", "get_skill_reference should only appear after skill loaded")
}
```

`TestToolExecutor_Tools_ParameterSchemas` 删除 `load_skill` 和 `get_skill_reference` 的断言块，替换为技能工具断言：

```go
	// resume-design: skill tool, no parameters
	{
		tool := toolByName["resume-design"]
		assert.NotEmpty(t, tool.Description)
		props := tool.Parameters["properties"].(map[string]interface{})
		assert.Empty(t, props)
	}
```

- [ ] **Step 7: 运行全部测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS

- [ ] **Step 8: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: Tools() 动态生成技能工具"
```

---

### Task 4: 技能工具执行 + 路由

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 写失败测试 - 调用技能工具返回描述文档**

`tool_executor_test.go` 追加：

```go
func TestExecute_SkillAsTool(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 200)
	result, err := executor.Execute(ctx, "resume-design", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "resume-design", data["name"])
	assert.NotEmpty(t, data["description"])
	assert.NotEmpty(t, data["usage"])
}

func TestExecute_SkillAsTool_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	_, err = executor.Execute(context.Background(), "nonexistent-skill", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestExecute_SkillAsTool_MarksLoaded(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	sessionID := uint(201)
	ctx := WithSessionID(context.Background(), sessionID)

	// Before calling skill tool, get_skill_reference should not be in tools
	toolsBefore := executor.Tools(ctx)
	for _, tool := range toolsBefore {
		assert.NotEqual(t, "get_skill_reference", tool.Name)
	}

	// Call skill tool
	_, err = executor.Execute(ctx, "resume-design", nil)
	require.NoError(t, err)

	// After calling skill tool, get_skill_reference should appear
	toolsAfter := executor.Tools(ctx)
	found := false
	for _, tool := range toolsAfter {
		if tool.Name == "get_skill_reference" {
			found = true
			break
		}
	}
	assert.True(t, found, "get_skill_reference should appear after loading a skill")
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/agent/... -run "TestExecute_SkillAsTool" -v
```

Expected: FAIL（Execute 不认识 "resume-design" 工具名）

- [ ] **Step 3: 实现 executeSkillTool 方法**

`tool_executor.go` 在 `getSkillReference` 方法后追加：

```go
// ---------------------------------------------------------------------------
// skill tool execution
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) executeSkillTool(ctx context.Context, skillName string) (string, error) {
	if e.skillLoader == nil {
		return `{"error":"技能库未加载"}`, nil
	}

	desc, err := e.skillLoader.LoadSkill(skillName)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}

	// Mark skill as loaded for this session.
	if sessionID, ok := ctx.Value(sessionIDKey).(uint); ok {
		e.markSkillLoaded(sessionID, skillName)
	}

	b, err := json.Marshal(desc)
	if err != nil {
		return "", fmt.Errorf("marshal skill descriptor: %w", err)
	}
	return string(b), nil
}
```

- [ ] **Step 4: 修改 Execute() 路由**

`tool_executor.go` 的 `Execute` 方法，删除 `case "load_skill":` 行，在 `default` 分支前加入技能路由：

```go
func (e *AgentToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	switch toolName {
	case "get_draft":
		return e.getDraft(ctx, params)
	case "apply_edits":
		return e.applyEdits(ctx, params)
	case "search_assets":
		return e.searchAssets(ctx, params)
	case "get_skill_reference":
		return e.getSkillReference(ctx, params)
	default:
		// Check if it's a skill tool
		if e.skillLoader != nil && e.skillLoader.HasSkill(toolName) {
			return e.executeSkillTool(ctx, toolName)
		}
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -run "TestExecute_SkillAsTool" -v
```

Expected: 全部 PASS

- [ ] **Step 6: 运行全部测试确认无回归**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: 技能工具执行和路由"
```

---

### Task 5: 删除 load_skill 工具定义和执行方法

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`
- Modify: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 删除 loadSkill 方法**

`tool_executor.go` 删除第 421-450 行的 `load_skill` 注释和 `loadSkill` 方法。

- [ ] **Step 2: 更新 get_skill_reference 错误文案**

`tool_executor.go` 第 465 行：

```go
// 改前
return `{"error":"skill_name is required: you must call load_skill first"}`, nil

// 改后
return `{"error":"skill_name is required"}`, nil
```

第 474 行：

```go
// 改前
return fmt.Sprintf(`{"error":"skill '%s' not loaded: call load_skill('%s') first"}`, skillName, skillName), nil

// 改后
return fmt.Sprintf(`{"error":"skill '%s' not loaded: call '%s' tool first"}`, skillName, skillName), nil
```

- [ ] **Step 3: 删除旧的 load_skill 测试**

`tool_executor_test.go` 删除以下测试函数：
- `TestLoadSkill_Valid`
- `TestLoadSkill_InvalidName`
- `TestLoadSkill_MissingParam`
- `TestLoadSkill_NilLoader`

- [ ] **Step 4: 更新 get_skill_reference 测试中的触发方式**

`tool_executor_test.go` 中 `TestGetReference_Valid` 改为通过技能工具加载：

```go
func TestGetReference_Valid(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 2)
	// First load the skill via skill tool (not load_skill)
	_, err = executor.Execute(ctx, "resume-interview", nil)
	require.NoError(t, err)

	// Then get the reference
	result, err := executor.Execute(ctx, "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, "test-engineer", data["name"])
	assert.NotEmpty(t, data["content"])
}
```

`TestGetReference_SkillNotLoaded` 更新错误文案断言：

```go
	assert.Contains(t, result, "not loaded")
	assert.Contains(t, result, "call 'resume-interview' tool first")
```

`TestGetReference_ReferenceNotFound` 更新触发方式：

```go
	// Load the skill via skill tool
	_, err = executor.Execute(ctx, "resume-interview", nil)
	require.NoError(t, err)
```

`TestGetReference_MissingParams` 保持不变（不需要加载技能就能测参数校验）。

- [ ] **Step 5: 更新 CompleteFlow 测试**

```go
func TestToolExecutor_CompleteFlow(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 100)

	// Step 1: Call skill tool (not load_skill)
	loadResult, err := executor.Execute(ctx, "resume-interview", nil)
	require.NoError(t, err)

	var loadDesc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(loadResult), &loadDesc))
	assert.Equal(t, "resume-interview", loadDesc["name"])

	// Step 2: Get reference
	refResult, err := executor.Execute(ctx, "get_skill_reference", map[string]interface{}{
		"skill_name":     "resume-interview",
		"reference_name": "test-engineer",
	})
	require.NoError(t, err)

	var refData map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(refResult), &refData))
	assert.Equal(t, "test-engineer", refData["name"])
	assert.NotEmpty(t, refData["content"])
}
```

- [ ] **Step 6: 运行全部测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "refactor: 删除 load_skill 工具，更新错误文案和测试"
```

---

### Task 6: System Prompt 更新

**Files:**
- Modify: `backend/internal/modules/agent/service.go`

- [ ] **Step 1: 更新 systemPromptV2**

`service.go` 第 106-156 行，修改核心工具和工作流程部分：

```go
const systemPromptV2 = `你是简历编辑专家。你可以像编辑代码一样精确编辑简历 HTML。

## 核心工具
- get_draft: 读取当前简历 HTML（可选 CSS selector 指定范围）
- apply_edits: 提交搜索替换操作修改简历（old_string 必须精确匹配）
- search_assets: 搜索用户资料库（旧简历、Git 摘要、笔记等）

## 核心铁律
所有简历内容必须以用户上传的资料为唯一事实来源。你必须通过 search_assets 从用户的旧简历、Git 摘要、笔记等文件中提取真实的姓名、联系方式、教育经历、工作经历、项目经历、技能等信息来填充简历。
只有在反复搜索后确实找不到某项关键信息时，才可以在最终回复中列出缺失项，提醒用户上传相关文件或手动补充。禁止在任何情况下凭空编造个人身份信息或职业经历。

## 工作流程
1. 先 get_draft 查看当前简历 HTML
2. 无论如何（新建或修改），都必须先用 search_assets 搜索用户上传的资料，提取真实信息作为简历内容依据
3. 如果 search_assets 返回了资料，仔细阅读其中内容，将真实的个人信息、教育背景、工作经历、项目、技能提取出来用于构建或修改简历
4. 只有当 search_assets 反复搜索后确实找不到某项关键信息（如没有旧简历可参考、缺少联系方式等），才在最终回复中明确告知用户缺少什么，建议上传文件或手动补充
5. 当用户明确目标岗位时，调用 resume-interview 技能工具，再用 get_skill_reference 获取该岗位的面经和简历建议
6. 当用户要求调整视觉、排版、配色、模板、样式，调用 resume-design 技能工具，再用 get_skill_reference 获取 A4 设计规范
7. 基于面经或设计规范中的建议修改简历
8. 用 apply_edits 提交精确修改
9. 修改后可用 get_draft 验证结果
10. 完成后用自然语言总结修改内容

## 编辑原则
- apply_edits 是搜索替换，不是追加：old_string 必须匹配要被替换的已有内容，new_string 是替换后的内容
- 绝对禁止把整份简历作为 new_string 写入而不匹配任何 old_string，这会导致内容重复
- 每次只修改需要变化的部分，不要重写整个简历
- old_string 必须精确匹配，不匹配则修改会失败
- 失败时读取当前 HTML 找到正确内容后重试
- 保持 HTML 结构完整，确保渲染正确
- 内容简洁专业，突出关键信息

## A4 简历硬约束
- 当前产品编辑的是简历，不是网页、落地页、作品集、仪表盘或海报
- 默认目标是一页 A4：210mm x 297mm；如果内容过多，先压缩文案、字号、行距和间距，不要扩展成多页视觉稿
- 使用常见招聘简历样式：白色或浅色纸面、深色正文、最多一个克制强调色、清晰分区标题、紧凑项目符号、信息密度高但可读
- 正文字号保持在 13-15px 左右，姓名标题不超过 24px，分区标题 14-16px；不要使用超大 hero 字体
- 字体必须支持中文渲染；禁止使用仅含拉丁字符的字体（如 Inter、Roboto 单独指定）；中文内容必须落在含有 "Noto Sans CJK SC"、"Microsoft YaHei"、"PingFang SC" 或系统 sans-serif 回退的字体栈中
- 技能列表必须可换行、可读，禁止做成长串不换行的技能胶囊或大块色卡
- 禁止使用 landing page、hero、dashboard、bento/card grid、glassmorphism、aurora、3D、霓虹、复杂渐变、大面积紫蓝/粉色背景、纹理背景、动画、发光、厚重阴影、过度圆角和装饰图形
- 如果用户说"太花"、"太炫"、"过头"、"不像简历"，优先移除视觉特效，恢复常规专业简历样式

## 回复规范
- 不要使用任何 emoji 或特殊符号装饰

## 技能库（Skills）
- resume-interview: 根据目标岗位的面试官视角优化简历，提供岗位面经、面试官关注点和简历针对性修改建议。当用户明确了目标岗位（如"测试工程师"、"前端开发"）时使用。
- resume-design: 提供 A4 单页简历设计规范和保守风格参考，帮助用户调整视觉、排版、配色。当用户要求调整简历样式或需要设计参考时使用。
`
```

- [ ] **Step 2: 运行编译确认无错**

```bash
cd backend && go build ./...
```

Expected: 无错误

- [ ] **Step 3: 运行全部测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "refactor: System Prompt 删除 load_skill/get_skill_reference 说明"
```

---

### Task 7: MockAdapter 更新

**Files:**
- Modify: `backend/internal/modules/agent/provider.go`
- Modify: `backend/internal/modules/agent/service_test.go`

- [ ] **Step 1: 更新 MockAdapter 的 StreamChatReAct**

`provider.go` 第 411-438 行，将 `load_skill` 调用改为 `resume-design` 技能工具调用：

```go
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
			Name:   "resume-design",
			Params: map[string]interface{}{},
		})
		_ = onToolCall(ToolCallRequest{
			ID:   "call_mock_3",
			Name: "get_skill_reference",
			Params: map[string]interface{}{
				"skill_name":     "resume-design",
				"reference_name": "a4-guidelines",
			},
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
```

- [ ] **Step 2: 更新 service_test.go 中的工具调用断言**

`service_test.go` 第 430-441 行：

```go
	assert.Equal(t, "get_draft", toolCalls[0].ToolName)
	assert.Equal(t, "completed", toolCalls[0].Status)

	assert.Equal(t, "resume-design", toolCalls[1].ToolName)
	assert.Equal(t, "completed", toolCalls[1].Status)

	assert.Equal(t, "get_skill_reference", toolCalls[2].ToolName)
	assert.Equal(t, "completed", toolCalls[2].Status)

	assert.Equal(t, "apply_edits", toolCalls[3].ToolName)
	assert.Equal(t, "completed", toolCalls[3].Status)
```

- [ ] **Step 3: 运行全部测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -v
```

Expected: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add backend/internal/modules/agent/provider.go backend/internal/modules/agent/service_test.go
git commit -m "refactor: MockAdapter 适配技能即工具"
```

---

### Task 8: 前端适配

**Files:**
- Modify: `frontend/workbench/src/components/chat/ToolCallLog.tsx`
- Modify: `frontend/workbench/src/components/chat/ChatPanel.tsx`

- [ ] **Step 1: 更新 ToolCallLog.tsx TOOL_META**

`ToolCallLog.tsx` 第 14-20 行：

```typescript
const TOOL_META: Record<string, { label: string; icon: ComponentType<{ className?: string }> }> = {
  get_draft: { label: '读取简历', icon: FileText },
  apply_edits: { label: '应用修改', icon: PencilLine },
  search_assets: { label: '搜索资料', icon: Search },
  'resume-design': { label: '加载设计技能', icon: Palette },
  'resume-interview': { label: '加载面试技能', icon: Search },
  get_skill_reference: { label: '获取参考内容', icon: FileText },
}
```

- [ ] **Step 2: 更新 ChatPanel.tsx ThinkingBubble**

`ChatPanel.tsx` 第 97-102 行：

```typescript
  const label = runningTool
    ? runningTool.name === 'apply_edits'
      ? '正在同步到画布'
      : runningTool.name === 'resume-design'
        ? '正在加载设计规范'
        : runningTool.name === 'resume-interview'
          ? '正在加载面试指南'
          : runningTool.name === 'get_skill_reference'
            ? '正在获取参考内容'
            : '正在处理资料'
    : '正在构思简历方案'
```

- [ ] **Step 3: 前端类型检查**

```bash
cd frontend/workbench && bun run build
```

Expected: 无错误

- [ ] **Step 4: 运行前端测试**

```bash
cd frontend/workbench && bunx vitest run
```

Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/workbench/src/components/chat/ToolCallLog.tsx frontend/workbench/src/components/chat/ChatPanel.tsx
git commit -m "feat: 前端适配技能即工具"
```

---

### Task 9: 全量验证

- [ ] **Step 1: 后端全部测试**

```bash
cd backend && go test ./... -v
```

Expected: 全部 PASS

- [ ] **Step 2: 后端编译检查**

```bash
cd backend && go build ./cmd/server/...
```

Expected: 无错误

- [ ] **Step 3: 前端构建**

```bash
cd frontend/workbench && bun run build
```

Expected: 无错误

- [ ] **Step 4: 确认 load_skill 已完全移除**

```bash
grep -rn "load_skill" backend/internal/modules/agent/ frontend/workbench/src/
```

Expected: 无匹配（除了可能的注释）

- [ ] **Step 5: 确认 search_skills/search_design_skill 已完全移除**

```bash
grep -rn "search_skills\|search_design_skill" frontend/workbench/src/
```

Expected: 无匹配
