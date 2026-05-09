# Agent 调试日志实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Agent 模块添加全景调试日志，让 AI 对话全流程可观测

**Architecture:** 新建 `debug.go` 提供带开关的 `debugLog` 函数，各模块统一调用。环境变量 `AGENT_DEBUG_LOG=false` 关闭，默认开启。日志使用中文描述。

**Tech Stack:** Go 标准库 `log` + `fmt`，零外部依赖

---

## 改动文件清单

| 文件 | 操作 |
|---|---|
| `backend/internal/modules/agent/debug.go` | 新建 |
| `backend/internal/modules/agent/debug_test.go` | 新建 |
| `backend/internal/modules/agent/provider.go` | 修改 |
| `backend/internal/modules/agent/service.go` | 修改 |
| `backend/internal/modules/agent/tool_executor.go` | 修改 |
| `backend/internal/modules/agent/skill_loader.go` | 修改 |

---

### Task 1: 新建 debug.go — 调试日志基础设施

**Files:**
- Create: `backend/internal/modules/agent/debug.go`
- Create: `backend/internal/modules/agent/debug_test.go`

- [ ] **Step 1: 写失败的测试**

```go
// debug_test.go
package agent

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugLog_Enabled(t *testing.T) {
	// 默认开启（不设 AGENT_DEBUG_LOG）
	os.Unsetenv("AGENT_DEBUG_LOG")
	// 重新加载 debugEnabled（需要重置 sync.OnceValue 的缓存，用子进程方式测试太复杂，
	// 这里直接测试 debugLog 的输出行为）
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// 因为 sync.OnceValue 只执行一次，这里测试的是当前状态下的行为
	debugLog("test", "你好世界 %d", 42)

	output := buf.String()
	// 如果当前 debugEnabled=true，应该有输出
	if debugEnabled() {
		assert.Contains(t, output, "[agent:test]")
		assert.Contains(t, output, "你好世界 42")
	}
}

func TestDebugLog_TruncateString(t *testing.T) {
	long := strings.Repeat("a", 500)
	truncated := truncateForLog(long, 200)
	assert.Len(t, truncated, 203) // 200 + len("...")
	assert.True(t, strings.HasSuffix(truncated, "..."))
}

func TestDebugLog_TruncateString_Short(t *testing.T) {
	short := "hello"
	truncated := truncateForLog(short, 200)
	assert.Equal(t, "hello", truncated)
}

func TestTruncateParams(t *testing.T) {
	params := map[string]interface{}{
		"selector": strings.Repeat("x", 500),
		"limit":    5,
	}
	result := truncateParams(params, 300)
	assert.LessOrEqual(t, len(result), 310) // 截断 + "..."
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/agent/... -run TestDebugLog -v
```

预期：FAIL — `debugLog`、`truncateForLog`、`truncateParams` 未定义

- [ ] **Step 3: 写实现**

```go
// debug.go
package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

var debugEnabled = sync.OnceValue(func() bool {
	return os.Getenv("AGENT_DEBUG_LOG") != "false"
})

func debugLog(component string, format string, args ...interface{}) {
	if debugEnabled() {
		log.Printf("[agent:%s] %s", component, fmt.Sprintf(format, args...))
	}
}

// truncateForLog 截断长字符串，超过 limit 时保留前 limit 个字符 + "..."
func truncateForLog(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

// truncateParams 将参数 map 序列化为 JSON 并截断
func truncateParams(params map[string]interface{}, limit int) string {
	b, err := json.Marshal(params)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	return truncateForLog(string(b), limit)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -run "TestDebugLog|TestTruncate" -v
```

预期：PASS

- [ ] **Step 5: 提交**

```bash
git add backend/internal/modules/agent/debug.go backend/internal/modules/agent/debug_test.go
git commit -m "feat: 添加 agent 调试日志基础设施 debug.go"
```

---

### Task 2: provider.go — AI 模型调用日志

**Files:**
- Modify: `backend/internal/modules/agent/provider.go`

- [ ] **Step 1: 在 StreamChat 方法中添加日志**

在 `StreamChat` 方法中（约第 87 行），在 `reqBody` marshal 之后、`http.NewRequestWithContext` 之前，添加：

```go
debugLog("provider", "StreamChat 正在调用模型 %s，消息数 %d", a.model, len(messages))
```

在 `resp.Body.Close()` 之前（约第 121 行），添加：

```go
debugLog("provider", "StreamChat 模型响应，状态码 %d，耗时 %v", resp.StatusCode, time.Since(start))
```

需要在方法开头加 `start := time.Now()`。

在超时错误返回处（约第 113-118 行），添加：

```go
debugLog("provider", "StreamChat 模型调用超时，耗时 %v", time.Since(start))
```

- [ ] **Step 2: 在 StreamChatReAct 方法中添加日志**

在方法开头添加：

```go
start := time.Now()
toolNames := make([]string, len(tools))
for i, t := range tools {
    toolNames[i] = t.Name
}
debugLog("provider", "StreamChatReAct 正在调用模型 %s，消息数 %d，工具数 %d，工具列表 %v", a.model, len(messages), len(tools), toolNames)
```

在响应状态码检查之后（约第 253 行），添加：

```go
debugLog("provider", "StreamChatReAct 模型响应，状态码 %d，耗时 %v", resp.StatusCode, time.Since(start))
```

在 tool call accumulation 完成后（`finish_reason == "tool_calls"` 分支，约第 342 行），添加：

```go
debugLog("provider", "StreamChatReAct 收到 %d 个工具调用请求", len(toolCallAccums))
```

在工具参数解析失败处（约第 345-346 行 `json.Unmarshal` 失败时），添加：

```go
debugLog("provider", "StreamChatReAct 工具 %s 参数解析失败: %v，使用空参数", acc.name, err)
```

在流结束后剩余 tool call 分发处（约第 366 行），添加：

```go
debugLog("provider", "StreamChatReAct 流结束，分发剩余 %d 个工具调用", len(toolCallAccums))
```

在超时错误处，添加：

```go
debugLog("provider", "StreamChatReAct 模型调用超时，耗时 %v", time.Since(start))
```

- [ ] **Step 3: 运行编译确认无语法错误**

```bash
cd backend && go build ./cmd/server/...
```

- [ ] **Step 4: 运行现有测试确认无破坏**

```bash
cd backend && go test ./internal/modules/agent/... -v -count=1 2>&1 | tail -30
```

- [ ] **Step 5: 提交**

```bash
git add backend/internal/modules/agent/provider.go
git commit -m "feat: provider.go 添加 AI 模型调用调试日志"
```

---

### Task 3: service.go — ReAct 循环日志

**Files:**
- Modify: `backend/internal/modules/agent/service.go`

- [ ] **Step 1: 在 StreamChatReAct 入口添加日志**

在方法开头（约第 246 行后），session 加载成功后：

```go
debugLog("service", "收到请求，session=%d，draft=%d，project=%v", sessionID, session.DraftID, session.ProjectID)
```

用户消息保存后：

```go
debugLog("service", "用户消息已保存，长度 %d 字符", len(userMessage))
```

- [ ] **Step 2: 历史加载和压缩日志**

历史加载后（约第 265 行）：

```go
debugLog("service", "历史消息加载完成，共 %d 条", len(history))
```

压缩检查处（约第 273 行），在 `needsCompaction` 判断内部：

```go
debugLog("service", "压缩触发，token 估算 %d，压缩前 %d 条消息", s.estimateTokens(allMsgs), len(history))
```

压缩成功后：

```go
debugLog("service", "压缩完成，压缩后 %d 条消息", len(compacted))
```

压缩失败时（`compactErr != nil`，目前是静默忽略）：

```go
debugLog("service", "压缩失败，使用原始消息: %v", compactErr)
```

- [ ] **Step 3: 资源预加载日志**

资源预加载后（约第 297 行）：

```go
debugLog("service", "资源预加载完成，增强 prompt 长度 %d 字符", len(augmentedPrompt)-len(systemPromptV2))
```

- [ ] **Step 4: 迭代循环日志**

将已有的 `log.Printf` 改为 `debugLog`：

```go
// 原: log.Printf("agent: iteration %d calling model with %d messages and %d tools", ...)
debugLog("service", "第 %d 轮迭代，消息数 %d，工具数 %d", totalIter, len(apiMessages), len(s.toolExecutor.Tools()))
```

- [ ] **Step 5: 工具执行日志**

在 `onToolCall` 回调中，工具开始执行前（约第 363 行）：

```go
debugLog("service", "开始执行工具 %s，参数: %s", call.Name, truncateParams(call.Params, 300))
```

工具执行完成后（成功和失败两个分支都加）：

成功分支（约第 416 行后）：

```go
debugLog("service", "工具 %s 执行成功，耗时 %v，结果长度 %d 字符", call.Name, time.Since(now), len(result))
```

失败分支（约第 382 行后）：

```go
debugLog("service", "工具 %s 执行失败，耗时 %v，错误: %s", call.Name, time.Since(now), errMsg)
```

- [ ] **Step 6: stall 保护和循环结束日志**

stall 保护触发处（约第 523 行）：

```go
debugLog("service", "stall 保护触发，连续 %d 轮无输出", stallCount)
```

搜索过多提醒注入处（约第 497 行，`if reminder != ""`）：

```go
debugLog("service", "搜索过多提醒触发，连续 %d 轮未执行 apply_edits", searchOnlyCount)
```

循环正常结束处（保存助手消息后，约第 517 行）：

```go
debugLog("service", "循环结束，助手回复长度 %d 字符", len(fullText.String()))
```

将已有的错误日志改为 debugLog：

```go
// 原: log.Printf("agent: iteration %d model call failed: %v", totalIter, err)
debugLog("service", "第 %d 轮模型调用失败: %v", totalIter, err)
```

- [ ] **Step 7: 运行编译和测试**

```bash
cd backend && go build ./cmd/server/... && go test ./internal/modules/agent/... -v -count=1 2>&1 | tail -30
```

- [ ] **Step 8: 提交**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "feat: service.go 添加 ReAct 循环调试日志"
```

---

### Task 4: tool_executor.go — 工具执行详情日志

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`

- [ ] **Step 1: 在 Execute 入口添加日志**

在 `Execute` 方法的 switch 之前（约第 166 行），添加：

```go
debugLog("tools", "调用工具 %s，参数: %s", toolName, truncateParams(params, 300))
```

- [ ] **Step 2: get_draft 日志**

在 `getDraft` 方法中，selector 解析后（约第 199 行）：

```go
if selector == "" {
    debugLog("tools", "get_draft 读取完整 HTML，长度 %d 字符", len(draft.HTMLContent))
    return draft.HTMLContent, nil
}
debugLog("tools", "get_draft 使用 selector=%q", selector)
```

selector 结果返回前：

```go
debugLog("tools", "get_draft selector=%q 结果长度 %d 字符", selector, len(html))
```

- [ ] **Step 3: apply_edits 日志**

在 ops 解析完成后（约第 262 行），添加：

```go
debugLog("tools", "apply_edits 开始，共 %d 个操作", len(ops))
```

在每个操作的验证失败处（约第 255-257 行），添加：

```go
debugLog("tools", "apply_edits 操作 %d 验证失败: %v", i, err)
```

在循环中每条操作应用时（约第 302-303 行），添加：

```go
debugLog("tools", "apply_edits 操作 %d/%d: old=%s → new=%s", i+1, len(ops), truncateForLog(op.OldString, 100), truncateForLog(op.NewString, 100))
```

在事务成功后（约第 339 行），添加：

```go
debugLog("tools", "apply_edits 完成，成功 %d 个编辑，新序列号 %d", applied, nextSeq-1)
```

在方法最后（事务返回 error 时），也需要在 err != nil 时记录：

```go
if err != nil {
    debugLog("tools", "apply_edits 失败: %v", err)
}
```

- [ ] **Step 4: search_assets 日志**

在查询构建完成后、执行前（约第 378 行前），添加：

```go
queryStr, _ := params["query"].(string)
typeStr, _ := params["type"].(string)
debugLog("tools", "search_assets 查询=%q，类型=%q，limit=%d", queryStr, typeStr, limit)
```

在结果返回前（约第 418 行），添加：

```go
debugLog("tools", "search_assets 返回 %d 条结果", len(results))
```

- [ ] **Step 5: load_skill 和 get_skill_reference 日志**

在 `loadSkill` 方法中，技能加载成功后（约第 438 行），添加：

```go
debugLog("tools", "加载技能 %s 成功", skillName)
```

在 `getSkillReference` 方法中，参考文档获取成功后（约第 484 行），添加：

```go
debugLog("tools", "获取技能参考文档 %s/%s 成功", skillName, refName)
```

- [ ] **Step 6: 运行编译和测试**

```bash
cd backend && go build ./cmd/server/... && go test ./internal/modules/agent/... -v -count=1 2>&1 | tail -30
```

- [ ] **Step 7: 提交**

```bash
git add backend/internal/modules/agent/tool_executor.go
git commit -m "feat: tool_executor.go 添加工具执行调试日志"
```

---

### Task 5: skill_loader.go — 技能加载日志

**Files:**
- Modify: `backend/internal/modules/agent/skill_loader.go`

- [ ] **Step 1: 在 NewSkillLoader 完成后添加日志**

在 `NewSkillLoader` 返回前（约第 106 行），添加：

```go
skillNames := make([]string, 0, len(loader.skills))
for name := range loader.skills {
    skillNames = append(skillNames, name)
}
refCount := 0
for _, refs := range loader.references {
    refCount += len(refs)
}
debugLog("skills", "技能加载完成，共 %d 个技能 %v，%d 个参考文档", len(loader.skills), skillNames, refCount)
```

需要在文件顶部添加 `"fmt"` 到 import（如果还没有的话）。实际上不需要 fmt，debugLog 内部处理格式化。但需要确保 import 中没有遗漏。

- [ ] **Step 2: 运行编译和测试**

```bash
cd backend && go build ./cmd/server/... && go test ./internal/modules/agent/... -v -count=1 2>&1 | tail -30
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/modules/agent/skill_loader.go
git commit -m "feat: skill_loader.go 添加技能加载调试日志"
```

---

### Task 6: 端到端验证

- [ ] **Step 1: 启动服务，发送一条 AI 对话请求，观察日志输出**

```bash
cd backend && AGENT_DEBUG_LOG=true go run cmd/server/main.go
```

在另一个终端发送请求（或通过前端操作），确认日志中能看到：
- `[agent:provider]` 开头的模型调用日志
- `[agent:service]` 开头的循环迭代日志
- `[agent:tools]` 开头的工具执行日志

- [ ] **Step 2: 验证关闭开关**

```bash
cd backend && AGENT_DEBUG_LOG=false go run cmd/server/main.go
```

确认 AI 相关调试日志不再输出，原有的 5 条日志仍然保留。

- [ ] **Step 3: 运行全量测试**

```bash
cd backend && go test ./... -count=1 2>&1 | tail -20
```

- [ ] **Step 4: 最终提交**

```bash
git add -A
git commit -m "feat: agent 模块全景调试日志，支持 AGENT_DEBUG_LOG 开关"
```
