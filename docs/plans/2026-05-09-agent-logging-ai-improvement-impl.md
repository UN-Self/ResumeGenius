# Agent 日志与 AI 行为改进实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 AI Agent 死循环问题：移除后端 CSS 正则拦截，把设计规范前置到技能描述，改进日志以便诊断。

**Architecture:** 三管齐下——(1) 删除 `validateResumeEditFragment` 及相关常量，(2) 在 `resume-design` 技能描述中嵌入 CSS 规范摘要，(3) 在失败路径和关键节点增加完整日志。

**Tech Stack:** Go, GORM, testify

---

## File Structure

| 文件 | 改动类型 | 职责 |
|---|---|---|
| `backend/internal/modules/agent/tool_executor.go` | 修改 | 删除 CSS 验证、改进错误消息、改进失败日志 |
| `backend/internal/modules/agent/tool_executor_test.go` | 修改 | 删除 CSS 验证测试、添加新错误消息测试 |
| `backend/internal/modules/agent/service.go` | 修改 | 添加模型文本输出日志、提醒消息日志、迭代汇总日志 |
| `backend/internal/modules/agent/debug.go` | 修改 | 添加 `truncateFull` 辅助函数 |
| `backend/internal/modules/agent/skills/resume-design/skill.yaml` | 修改 | description 加入 CSS 规范摘要 |

---

### Task 1: 移除 CSS 正则拦截

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:573-606`
- Modify: `backend/internal/modules/agent/tool_executor.go:389-392`
- Modify: `backend/internal/modules/agent/tool_executor_test.go:378-404`

- [ ] **Step 1: 删除 CSS 验证测试**

在 `tool_executor_test.go` 中删除 `TestApplyEdits_RejectsOverdesignedResumeStyle` 函数（lines 378-404）。

```go
// 删除这个函数：
func TestApplyEdits_RejectsOverdesignedResumeStyle(t *testing.T) { ... }
```

- [ ] **Step 2: 运行测试确认删除后无影响**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits -v`
Expected: 剩余的 apply_edits 测试全部 PASS

- [ ] **Step 3: 删除 validateResumeEditFragment 函数和常量**

在 `tool_executor.go` 中删除以下内容：

删除 lines 573-606（`resumeEditRejectPatterns` 常量和 `validateResumeEditFragment` 函数）：
```go
// 删除这段：
var resumeEditRejectPatterns = []struct { ... }
func validateResumeEditFragment(fragment string) error { ... }
```

删除 lines 389-392（调用处）：
```go
// 删除这段：
if err := validateResumeEditFragment(newStr); err != nil {
    debugLog("tools", "操作 %d 验证失败: %v", i, err)
    return "", fmt.Errorf("ops[%d].new_string violates resume design constraints: %w", i, err)
}
```

- [ ] **Step 4: 运行测试确认编译通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go build ./...`
Expected: 编译成功，无未引用的变量/函数错误

- [ ] **Step 5: 运行全部 agent 测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -v`
Expected: 全部 PASS

- [ ] **Step 6: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "refactor: remove CSS regex validation from apply_edits"
```

---

### Task 2: 改进 apply_edits 错误消息

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:431-435`
- Modify: `backend/internal/modules/agent/tool_executor_test.go:343-376`

- [ ] **Step 1: 更新 old_string_not_found 测试的断言**

在 `tool_executor_test.go` 的 `TestApplyEdits_OldStringNotFound` 中，更新断言以匹配新的错误消息格式：

```go
func TestApplyEdits_OldStringNotFound(t *testing.T) {
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
                "new_string": "Replacement",
            },
        },
    })
    require.Error(t, err)
    assert.Contains(t, err.Error(), "old_string not found")
    assert.Contains(t, err.Error(), "NonExistent")           // 包含 old_string 前 100 字符
    assert.Contains(t, err.Error(), "<html><body><h1>Title") // 包含当前 HTML 前 200 字符

    var unchanged models.Draft
    require.NoError(t, db.First(&unchanged, draft.ID).Error)
    assert.Equal(t, html, unchanged.HTMLContent)
    assert.Equal(t, 0, unchanged.CurrentEditSequence)

    var edits []models.DraftEdit
    require.NoError(t, db.Where("draft_id = ?", draft.ID).Find(&edits).Error)
    assert.Empty(t, edits)
}
```

- [ ] **Step 2: 运行测试确认新断言失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_OldStringNotFound -v`
Expected: FAIL（因为错误消息还没有改）

- [ ] **Step 3: 改进错误消息**

在 `tool_executor.go` 的 `applyEdits` 函数中，修改 old_string 验证失败的错误消息（lines 431-435）：

```go
// 改前：
if !strings.Contains(html, op.OldString) {
    debugLog("tools", "操作 %d 验证失败: old_string 未找到: %s", i, truncateHTML(op.OldString))
    return fmt.Errorf("old_string not found: %q", op.OldString)
}

// 改后：
if !strings.Contains(html, op.OldString) {
    oldPreview := truncateDebug(op.OldString, 100)
    htmlPreview := truncateDebug(html, 200)
    debugLog("tools", "操作 %d 验证失败: old_string 未找到: %s", i, truncateHTML(op.OldString))
    return fmt.Errorf("old_string not found in current draft. old_string前100字符: %q | 当前HTML前200字符: %q", oldPreview, htmlPreview)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestApplyEdits_OldStringNotFound -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: improve apply_edits error message with old_string and HTML preview"
```

---

### Task 3: CSS 规范写入技能描述

**Files:**
- Modify: `backend/internal/modules/agent/skills/resume-design/skill.yaml`

- [ ] **Step 1: 更新 skill.yaml 的 description 字段**

在 `skill.yaml` 的 `description` 中嵌入 CSS 规范摘要，使模型调用一次就能看到规则：

```yaml
name: resume-design
description: |
  提供 A4 单页简历设计规范和保守风格参考，帮助用户调整视觉、排版、配色。
  调用后请用 get_skill_reference 获取完整 A4 设计规范（reference_name: a4-guidelines）。

  ## CSS 设计规范摘要

  推荐样式：
  - 背景：白色或接近白色纸面（background: #ffffff 或 #f7fafc）
  - 正文：深灰或黑色，高对比（color: #1a202c 或 #333）
  - 强调色：最多一个克制颜色（如 #1a365d 深蓝），只用于姓名、分区标题、细线
  - 布局：单栏或常规双栏，顶部信息紧凑，分区标题清晰
  - 字号：正文 13-15px，姓名 ≤24px，分区标题 14-16px
  - 字体：必须包含中文支持，如 'Noto Sans CJK SC', 'Microsoft YaHei', 'PingFang SC', sans-serif
  - 间距：行距紧凑但可读，段间距少量，避免大块留白

  禁止样式（会导致打印问题或视觉不专业）：
  - 渐变背景：linear-gradient、radial-gradient、conic-gradient
  - 模糊效果：backdrop-filter、filter:blur
  - 动画：animation、@keyframes、transition
  - 阴影：text-shadow、复杂 box-shadow
  - 定位：position:fixed、position:absolute
  - 视口单位：100vh、100vw
  - 装饰风格：glassmorphism、aurora、3D、霓虹、发光
  - 大面积彩色背景、超大标题、营销式布局

trigger: 用户要求调整简历样式或需要设计参考时

usage: |
  1. 使用 get_skill_reference 获取 A4 设计规范
  2. 基于规范指导简历样式修改

tools:
  - name: get_skill_reference
    description: 获取 A4 简历设计规范
    params:
      - name: skill_name
        type: string
        description: 固定传 "resume-design"
      - name: reference_name
        type: string
        description: 固定传 "a4-guidelines"
    usage: 返回完整的 A4 单页简历设计规范

references:
  - name: a4-guidelines
    description: A4 单页简历设计规范，包含推荐样式、禁止样式和修改策略
```

- [ ] **Step 2: 验证 YAML 格式正确**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go run -e -c 'import "gopkg.in/yaml.v3"; ...'` 或手动检查 YAML 缩进

实际验证方式：运行加载技能的测试：
Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestExecute_SkillAsTool -v`
Expected: PASS，且 description 包含 "CSS 设计规范摘要"

- [ ] **Step 3: 添加测试验证 description 包含关键内容**

在 `tool_executor_test.go` 中添加新测试：

```go
func TestSkillTool_ResumeDesign_DescriptionContainsCSSGuidelines(t *testing.T) {
    loader, err := NewSkillLoader()
    require.NoError(t, err)
    executor := NewAgentToolExecutor(nil, loader)

    ctx := WithSessionID(context.Background(), 300)
    result, err := executor.Execute(ctx, "resume-design", nil)
    require.NoError(t, err)

    var data map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(result), &data))
    desc, ok := data["description"].(string)
    require.True(t, ok)

    assert.Contains(t, desc, "linear-gradient", "description should mention banned gradient")
    assert.Contains(t, desc, "backdrop-filter", "description should mention banned backdrop-filter")
    assert.Contains(t, desc, "Noto Sans CJK SC", "description should mention Chinese font requirement")
}
```

- [ ] **Step 4: 运行新测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestSkillTool_ResumeDesign_DescriptionContainsCSSGuidelines -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/skills/resume-design/skill.yaml backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: embed CSS guidelines summary in resume-design skill description"
```

---

### Task 4: 日志改进

**Files:**
- Modify: `backend/internal/modules/agent/debug.go`
- Modify: `backend/internal/modules/agent/tool_executor.go:388-392`
- Modify: `backend/internal/modules/agent/service.go:506-509`

- [ ] **Step 1: 在 debug.go 添加 truncateFull 辅助函数**

在 `debug.go` 中添加一个用于失败路径的完整日志函数（不截断，但标记）：

```go
// debugLogFull logs a full message without truncation, prefixed with [FULL].
// Use only on failure paths where truncated context is insufficient.
func debugLogFull(component string, label string, content string) {
    if debugEnabled() {
        log.Printf("[agent:%s] [FULL] %s:\n%s", component, label, content)
    }
}
```

- [ ] **Step 2: 在 applyEdits 失败路径添加完整日志**

在 `tool_executor.go` 的 `applyEdits` 函数中，old_string 验证失败时添加完整参数日志：

```go
// 在 return fmt.Errorf(...) 之前添加：
debugLogFull("tools", fmt.Sprintf("apply_edits 操作 %d 失败 - 完整 new_string", i), op.NewString)
```

在 `applyEdits` 函数末尾的失败日志处（line 484 附近），也添加完整参数：

```go
if err != nil {
    debugLog("tools", "apply_edits 失败，耗时 %v: %v", time.Since(start), err)
    // 打印完整的 ops 参数用于调试
    opsJSON, _ := json.Marshal(ops)
    debugLogFull("tools", "apply_edits 失败 - 完整 ops", string(opsJSON))
} else {
```

- [ ] **Step 3: 在 service.go 记录模型文本输出**

在 `service.go` 的迭代循环中，`hadText` 为 true 时记录模型输出：

找到 `if hadText {` 块（line 515 附近），在 `allThinking.WriteString(thinkingAccum.String())` 之后添加：

```go
if hadText {
    allThinking.WriteString(thinkingAccum.String())
    debugLog("service", "模型文本输出，长度 %d 字符，内容: %s", fullText.Len(), truncateDebug(fullText.String(), 500))
    // ... 原有代码继续
```

- [ ] **Step 4: 在 service.go 记录注入的提醒消息**

在 `service.go` 的搜索过多提醒注入处（line 506-509），添加提醒内容日志：

```go
// 改前：
if reminder != "" {
    toolResults = append(toolResults, Message{Role: "user", Content: reminder})
    debugLog("service", "搜索过多提醒触发，连续 %d 轮未执行 apply_edits", searchOnlyCount)
}

// 改后：
if reminder != "" {
    toolResults = append(toolResults, Message{Role: "user", Content: reminder})
    debugLog("service", "搜索过多提醒触发，连续 %d 轮未执行 apply_edits", searchOnlyCount)
    debugLogFull("service", "提醒消息内容", reminder)
}
```

- [ ] **Step 5: 在 service.go 添加迭代汇总日志**

在 `service.go` 的循环结束后（line 542 `return ErrMaxIterations` 之前），添加汇总日志。同时在正常退出路径（line 529 附近）也添加汇总。

在 `for` 循环结束后、`return ErrMaxIterations` 之前添加：

```go
} // end of for loop

// 迭代汇总
debugLog("service", "迭代汇总: 总轮次 %d，总耗时 %v", s.maxIterations*2+1, time.Since(loopStart))

return ErrMaxIterations
```

正常退出路径（line 529 附近已有 `debugLog("service", "循环结束，共 %d 轮，总耗时 %v", totalIter+1, time.Since(loopStart))`），这个已经够了。

- [ ] **Step 6: 运行全部 agent 测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -v`
Expected: 全部 PASS

- [ ] **Step 7: 运行编译检查**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go build ./...`
Expected: 编译成功

- [ ] **Step 8: Commit**

```bash
cd /home/handy/Projects/ResumeGenius
git add backend/internal/modules/agent/debug.go backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/service.go
git commit -m "feat: improve agent logging with full params on failure and iteration summary"
```

---

### Task 5: 端到端验证

- [ ] **Step 1: 运行全部后端测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./... -v 2>&1 | tail -20`
Expected: 全部 PASS

- [ ] **Step 2: 启动后端服务确认无启动错误**

Run: `cd /home/handy/Projects/ResumeGenius && docker compose up -d backend && sleep 3 && docker compose logs backend --tail 20`
Expected: 无 panic 或启动错误

- [ ] **Step 3: 检查技能加载日志**

在 docker compose logs 中确认看到：
```
[agent:skills] 技能加载完成，共 2 个技能: [resume-design resume-interview]
```

- [ ] **Step 4: 最终 Commit（如有遗漏文件）**

```bash
cd /home/handy/Projects/ResumeGenius
git status
# 如果有遗漏的文件，add 并 commit
```
