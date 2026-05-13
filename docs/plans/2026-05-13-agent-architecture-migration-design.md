# ResumeGenius Agent 架构迁移设计

## 背景

ResumeGenius 的 AI Agent 存在循环调用问题：模型反复调用 `get_draft`（8+ 次）才调用 `apply_edits`，单次请求耗时 1 分 39 秒、12 轮迭代。根因分析发现 3 个实现 bug + 3 个架构差距，对标 Claude-Code（Anthropic 官方编程 agent）的架构模式进行系统性改进。

## 设计目标

1. 修复当前循环调用 bug（P0）
2. 重构 Skill 系统为模型主动调用模式
3. 重构 Prompt 系统为模块化 sections
4. 增强 Flow Control 防护机制

## Phase 0: P0 Bug 修复

### 0.1 系统提醒角色修正

**文件**: `backend/internal/modules/agent/service.go:521`

将提醒从 `user` 角色改为 `system` 角色：

```go
// Before
toolResults = append(toolResults, Message{Role: "user", Content: reminder})

// After
toolResults = append(toolResults, Message{Role: "system", Content: reminder})
```

### 0.2 删除反循环指令

**文件**: `backend/internal/modules/agent/service.go:133`

删除系统提示中鼓励重读的指令：

```diff
- - 失败时读取当前 HTML 找到正确内容后重试
+ - 失败时用更短的唯一片段重新搜索，确保文本精确匹配
```

### 0.3 提醒阶梯前移

**文件**: `backend/internal/modules/agent/service.go:509-520`

调整提醒触发时机：

| searchOnlyCount | 原提醒 | 新提醒 |
|---|---|---|
| == 2 | 无 | "你已读取了简历结构，现在应该开始编辑了。" |
| == 3 | "可以考虑开始写简历了" | "停止搜索，立即调用 apply_edits 编辑简历。" |
| >= 4 | "信息已经足够了" | "禁止再调用 get_draft。必须立刻调用 apply_edits。" |
| remaining <= 2 | "这是最后一步" | "最后机会。必须立刻调用 apply_edits，否则任务失败。" |

删除 `searchOnlyCount >= 6` 档位（在新阶梯下已无意义）。

### 0.4 get_draft 重复调用防护

**文件**: `backend/internal/modules/agent/tool_executor.go`

在 `AgentToolExecutor` 中新增调用计数：

```go
type AgentToolExecutor struct {
    db               *gorm.DB
    skillLoader      *SkillLoader
    loadedSkills     sync.Map
    getDraftCallCount sync.Map // sessionID → int
}
```

在 `getDraft()` 方法中：
- 每次调用递增计数
- 第 3 次及以后：返回拒绝信息，不返回 HTML 内容

```
"你已经读取了简历 N 次，内容没有变化。请直接使用 apply_edits 编辑简历，不要再调用 get_draft。"
```

在 `ClearSessionState()` 中清理计数。

### 0.5 工具描述优化

**文件**: `backend/internal/modules/agent/tool_executor.go:122`

修改 `get_draft` 工具描述，加入调用限制说明：

```diff
- Description: "获取简历 HTML 内容。支持 4 种模式：...首次调用请使用 structure 模式了解整体结构。",
+ Description: "获取简历 HTML 内容。支持 4 种模式：structure（结构概览）、section（指定区域）、search（关键词搜索）、full（完整内容）。最多调用 2 次（structure + full），之后必须用 apply_edits 编辑。",
```

---

## Phase 1: Skill 系统重构

### 1.1 当前架构问题

- 技能是被动知识文档，模型需要遵循 usage 指令自行调用子工具
- `get_skill_reference` 子工具增加了不必要的复杂度
- 技能加载和引用获取是两步操作，模型容易遗漏

### 1.2 目标架构

将技能从"声明式元数据"改为"模型主动调用的工具"：

```
模型调用 load_skill(skill_name="resume-design")
    → 返回技能描述 + 全部参考内容（一步到位）
    → 模型基于返回内容直接操作
```

### 1.3 改动

**文件**: `backend/internal/modules/agent/tool_executor.go`

1. 新增 `load_skill` 工具定义（替代原 skill 工具 + `get_skill_reference`）：

```go
{
    Name: "load_skill",
    Description: "加载技能参考内容。返回技能描述和全部参考文档。调用后按返回的 usage 指引操作。",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "skill_name": map[string]interface{}{
                "type": "string",
                "description": "技能名称，如 'resume-design'、'resume-interview'",
            },
        },
        "required": []string{"skill_name"},
    },
}
```

2. `load_skill` 执行逻辑：
   - 调用 `skillLoader.LoadSkill(skillName)` 获取描述
   - 遍历描述中的 references，调用 `skillLoader.GetReference()` 获取全部参考内容
   - 将描述 + 全部参考内容合并返回
   - 一步到位，不再需要 `get_skill_reference`

3. 移除 `get_skill_reference` 工具定义和相关代码

4. 移除 session 级别的 skill loaded tracking（不再需要）

**文件**: `backend/internal/modules/agent/skill_loader.go`

- `LoadSkill()` 返回值中直接包含全部 reference 内容（而非仅描述）

---

## Phase 2: Prompt 系统重构

### 2.1 当前架构问题

- 单体 `systemPromptV2` 常量（~2000 字符），难以维护
- 资源信息直接拼接到末尾，污染 system prompt
- 工具描述是静态字符串，无法注入运行时状态

### 2.2 目标架构

将系统提示拆分为模块化 sections：

```go
type PromptSection struct {
    ID        string
    Content   string
    Cacheable bool // true = 跨请求可缓存
}

func buildSystemPrompt(sections []PromptSection) string {
    var sb strings.Builder
    for _, s := range sections {
        sb.WriteString(s.Content)
        sb.WriteString("\n")
    }
    return sb.String()
}
```

### 2.3 Sections 定义

| Section ID | 内容 | Cacheable | 来源 |
|---|---|---|---|
| `identity` | 角色定义、核心铁律 | true | 原 systemPromptV2 第一段 |
| `tools` | 工具使用说明 | true | 原 "核心工具" + "工作流程" |
| `edit_rules` | 编辑原则 | true | 原 "编辑原则" |
| `a4_constraints` | A4 简历硬约束 | true | 原 "A4 简历硬约束" |
| `flow_rules` | 循环控制规则（调用限制、提醒机制） | true | **新增** |
| `reply_rules` | 回复规范 | true | 原 "回复规范" |
| `assets` | 用户资源信息 | false | 运行时注入 |
| `skills` | 可用技能列表 | false | 运行时注入 |

### 2.4 改动

**新建文件**: `backend/internal/modules/agent/prompt.go`

- 定义 `PromptSection` 结构体
- 定义各 section 常量
- 实现 `buildSystemPrompt()` 函数

**修改文件**: `backend/internal/modules/agent/service.go`

- `systemPromptV2` 常量拆分到 `prompt.go`
- `preloadAssets()` 返回值改为 `PromptSection`
- `StreamChatReAct()` 中使用 `buildSystemPrompt()` 构建完整提示

### 2.5 flow_rules section 内容

```
## 循环控制规则
- get_draft 最多调用 2 次（structure + full），之后必须直接用 apply_edits 编辑
- 重复读取不会获得新信息，只会浪费步骤
- apply_edits 失败时，用更短的唯一片段重试，不要重新读取整个简历
- 如果步骤即将耗尽，优先输出当前最佳结果，不要继续搜索
```

---

## Phase 3: Flow Control 增强

### 3.1 工具失败恢复指令

**文件**: `backend/internal/modules/agent/service.go`

在 `apply_edits` 失败时，注入恢复指令作为 tool result 的一部分：

```go
if call.Name == "apply_edits" && execErr != nil {
    // 在错误信息后附加恢复指令
    recoveryHint := "\n\n[提示] 编辑失败。请用更短的唯一片段重试 old_string，确保精确匹配当前 HTML 中的文本。"
    iterToolResults = append(iterToolResults, Message{
        Role:       "tool",
        Content:    errMsg + recoveryHint,
        ToolCallID: call.ID,
        Name:       call.Name,
    })
}
```

### 3.2 abort signal 支持

**文件**: `backend/internal/modules/agent/service.go`

在 ReAct 循环中检查 context 取消信号：

```go
for totalIter := 0; totalIter < s.maxIterations*2+1; totalIter++ {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // ... existing loop body
}
```

需要 handler 层传入可取消的 context（当前使用 `context.Background()`）。

### 3.3 reactive compact guard

**文件**: `backend/internal/modules/agent/service.go`

在压缩逻辑中添加保护：

```go
hasAttemptedCompact := false

if s.needsCompaction(allMsgs) {
    if hasAttemptedCompact {
        debugLog("service", "压缩已尝试过，跳过以避免无限循环")
    } else {
        hasAttemptedCompact = true
        // ... existing compaction logic
    }
}
```

---

## 实施顺序

1. **Phase 0** — 立即修复 bug，预计 1-2 小时
2. **Phase 1** — Skill 重构，预计 2-3 小时
3. **Phase 2** — Prompt 重构，预计 2-3 小时
4. **Phase 3** — Flow Control 增强，预计 1-2 小时

## 验证方式

- Phase 0: 手动测试 AI 对话，确认 get_draft 调用不超过 2-3 次，总迭代从 12 降到 3-5
- Phase 1: 测试 `load_skill` 工具调用，确认一步返回完整技能内容
- Phase 2: 对比新旧 system prompt 输出，确认内容一致
- Phase 3: 测试 apply_edits 失败后的恢复行为
