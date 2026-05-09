# 技能即工具：渐进式披露重构设计

更新时间：2026-05-09
状态：设计已完成，待实施

## 1. 设计目标

将 `load_skill` 和 `get_skill_reference` 从 AI 工具列表中移除，让每个技能本身作为一个工具出现。AI 调用技能工具后，系统自动加载描述文档并注入子工具到下一轮工具列表。

### 核心理念

技能 = 工具。AI 不需要知道"加载技能"这个概念，它只知道：有些工具调用后会解锁更多工具。

## 2. 架构

### 2.1 AI 视角的交互流程

```
Round 1:
  工具列表: [get_draft, apply_edits, search_assets, resume-design, resume-interview]
  用户: "我想应聘测试工程师，帮我改简历"
  AI: tool_call(resume-design)

Round 2:
  工具列表: [get_draft, apply_edits, search_assets, resume-design, resume-interview, get_skill_reference]
  系统返回: skill.yaml 内容（描述文档 + 可用引用列表）
  AI: tool_call(get_skill_reference, {skill_name: "resume-design", reference_name: "a4-guidelines"})

Round 3:
  系统返回: A4 设计规范全文
  AI: tool_call(apply_edits, ...)
```

### 2.2 组件职责

- **SkillLoader**：不变，继续负责加载和解析 YAML
- **AgentToolExecutor**：`Tools()` 变动态，新增技能工具执行逻辑
- **ChatService**：不变，每轮调 `Tools(ctx)` 获取最新工具列表

### 2.3 数据流

```
SkillLoader ──▶ AgentToolExecutor.Tools(ctx) ──▶ AI Provider (OpenAI API)
                     ▲                                    │
                     │                              tool_call
                     ▼                                    │
              AgentToolExecutor.Execute() ◀───────────────┘
                     │
         ┌───────────┼───────────┐
         ▼           ▼           ▼
   基础工具      技能工具     子工具
   (固定)    (Skills生成)  (动态注入)
```

## 3. ToolExecutor 改动

### 3.1 接口变更

```go
// 改前
type ToolExecutor interface {
    Tools() []ToolDef
    Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error)
}

// 改后
type ToolExecutor interface {
    Tools(ctx context.Context) []ToolDef
    Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error)
}
```

### 3.2 Tools() 动态生成逻辑

```go
func (e *AgentToolExecutor) Tools(ctx context.Context) []ToolDef {
    // 1. 基础工具（固定）
    tools := []ToolDef{getDraftTool, applyEditsTool, searchAssetsTool}

    // 2. 技能工具（从 SkillLoader 自动生成，无参数）
    for _, skill := range e.skillLoader.Skills() {
        tools = append(tools, ToolDef{
            Name:        skill.Name,
            Description: skill.Description,
            Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
        })
    }

    // 3. 子工具（根据已加载的技能动态注入）
    sessionID, _ := ctx.Value(sessionIDKey).(uint)
    if e.hasLoadedSkills(sessionID) {
        tools = append(tools, getSkillReferenceTool)  // 共享子工具
        // 技能专属工具：遍历已加载技能的 skill.yaml tools 字段，
        // 将非 get_skill_reference 的工具定义转为 ToolDef 追加
    }

    return tools
}
```

### 3.3 技能工具执行

```go
func (e *AgentToolExecutor) executeSkillTool(ctx context.Context, skillName string) (string, error) {
    desc, err := e.skillLoader.LoadSkill(skillName)
    if err != nil {
        return errorJSON(err), nil
    }

    sessionID, _ := ctx.Value(sessionIDKey).(uint)
    e.markSkillLoaded(sessionID, skillName)

    // 返回 SkillDescriptor JSON（含 name, description, usage, tools, references）
    // AI 从中得知可用引用列表和使用方法
    b, _ := json.Marshal(desc)
    return string(b), nil
}
```

### 3.4 Execute() 路由变更

```go
func (e *AgentToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
    switch toolName {
    case "get_draft", "apply_edits", "search_assets":
        // 原有逻辑不变
    case "get_skill_reference":
        // 原有逻辑不变（仍检查父技能是否已加载）
    default:
        if e.skillLoader.HasSkill(toolName) {
            return e.executeSkillTool(ctx, toolName)
        }
        return "", fmt.Errorf("unknown tool: %s", toolName)
    }
}
```

## 4. SkillLoader 变更

新增一个方法：

```go
func (l *SkillLoader) HasSkill(name string) bool {
    _, ok := l.skills[name]
    return ok
}
```

其余不变。

### skill.yaml 结构

`tools` 字段含义变为：该技能加载后会解锁的专属工具。`get_skill_reference` 不在 skill.yaml 里声明，它是系统自动注入的共享工具。如果 `tools` 中有非 `get_skill_reference` 的工具定义，系统会将其作为该技能专属的子工具注入到工具列表中（仅在该技能已加载时可用）。

```yaml
name: resume-design
description: 提供 A4 单页简历设计规范和保守风格参考
trigger: 用户要求调整视觉、排版、配色时

usage: |
  1. 调用 get_skill_reference 获取 A4 设计规范
  2. 基于规范指导简历样式修改

tools: []   # 当前无专属工具

references:
  - name: a4-guidelines
    description: A4 单页简历设计规范
```

## 5. Service 层与 System Prompt

### 5.1 Service 层

```go
// 改前
s.provider.StreamChatReAct(ctx, apiMessages, s.toolExecutor.Tools(), ...)

// 改后
s.provider.StreamChatReAct(ctx, apiMessages, s.toolExecutor.Tools(ctx), ...)
```

### 5.2 System Prompt

删除 `load_skill` 和 `get_skill_reference` 的核心工具说明，工作流程里的"用 load_skill 加载"改为"调用技能工具"。技能库索引不变。

## 6. 前端适配

### 6.1 ToolCallLog.tsx

```go
const TOOL_META = {
  get_draft: { label: '读取简历', icon: FileText },
  apply_edits: { label: '应用修改', icon: PencilLine },
  search_assets: { label: '搜索资料', icon: Search },
  'resume-design': { label: '加载设计技能', icon: Palette },
  'resume-interview': { label: '加载面试技能', icon: Search },
  get_skill_reference: { label: '获取参考内容', icon: FileText },
}
```

### 6.2 ThinkingBubble

```go
if (name === 'resume-design') return '正在加载设计规范...'
if (name === 'resume-interview') return '正在加载面试指南...'
if (name === 'get_skill_reference') return '正在获取参考内容...'
```

## 7. 错误处理

| 场景 | 行为 |
|---|---|
| AI 调用不存在的技能工具 | `HasSkill()` 返回 false → `"unknown tool: xxx"` |
| AI 直接调 `get_skill_reference` 但未加载技能 | 工具列表里根本没有这个工具，AI 调不到 |
| `get_skill_reference` 的 skill_name 未加载 | `"skill 'X' not loaded: call 'X' tool first"` |
| reference 不存在 | `"reference 'Y' not found in skill 'X'. Available: [list]"` |

## 8. 测试策略

### 删除

- `TestToolExecutor_LoadSkill_*` 系列

### 新增

- `TestToolExecutor_SkillAsTool` — 调用技能工具返回描述文档
- `TestToolExecutor_SkillAsTool_NotFound` — 调用不存在的技能名
- `TestTools_DynamicInjection` — 加载技能后 Tools(ctx) 包含 get_skill_reference
- `TestTools_NoSkillsLoaded` — 未加载技能时 Tools(ctx) 不含 get_skill_reference

### 保留

- `TestToolExecutor_GetReference_*`（逻辑不变）
- `TestSkillLoader_*`（完全不变）

### MockAdapter

`provider.go` 里的 MockAdapter 需改为 mock `resume-design` 技能工具调用。

## 9. Prompt Cache 影响

无实质影响。系统提示词完全不变（cache 大头），工具列表变化仅在技能加载那一轮导致 cache miss，之后稳定。当前实现每轮消息内容已不同（多了 tool result），cache 本来就每轮在变。

## 10. 实施检查清单

- [ ] `tool_executor.go`：接口 `Tools()` 加 `ctx` 参数
- [ ] `tool_executor.go`：`Tools()` 动态生成（基础 + 技能 + 子工具）
- [ ] `tool_executor.go`：新增 `executeSkillTool()` 方法
- [ ] `tool_executor.go`：`Execute()` 新增技能工具路由
- [ ] `tool_executor.go`：删除 `load_skill` 工具定义和执行方法
- [ ] `tool_executor.go`：`get_skill_reference` 错误文案更新
- [ ] `tool_executor.go`：新增 `hasLoadedSkills()` 辅助方法
- [ ] `skill_loader.go`：新增 `HasSkill()` 方法
- [ ] `service.go`：`Tools()` 调用改为 `Tools(ctx)`
- [ ] `service.go`：System Prompt 删除 load_skill/get_skill_reference 说明
- [ ] `provider.go`：MockAdapter 更新
- [ ] `ToolCallLog.tsx`：更新 TOOL_META
- [ ] `ChatPanel.tsx`：更新 ThinkingBubble
- [ ] 测试文件适配
