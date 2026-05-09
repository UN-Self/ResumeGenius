# Agent 日志与 AI 行为改进设计

日期: 2026-05-09
状态: 已批准

## 背景

AI Agent 在执行简历设计任务时陷入死循环：40+ 轮迭代中反复调用 `get_draft` 和 `resume-design`，偶尔调用 `apply_edits` 但因 CSS 正则验证失败（"禁止复杂渐变背景"）。日志截断严重，无法定位根因。

根因分析：
1. `resume-design` 技能分两层：skill.yaml 返回描述，a4-guidelines.yaml 返回实际 CSS 规范。模型反复加载描述但从不调用 `get_skill_reference` 获取规范
2. CSS 验证在后端用正则拦截，模型要试错才知道规则，浪费迭代
3. 日志截断到 200-300 字符，失败时看不到完整参数

## 改动 1：日志改进

### 1.1 apply_edits 失败时打印完整参数

文件: `backend/internal/modules/agent/tool_executor.go`

当前验证失败时用 `truncateParams` 截断到 300 字符。改为：验证失败时打印完整的 `new_string`，用 `[FULL]` 前缀标记。

### 1.2 记录模型文本输出

文件: `backend/internal/modules/agent/service.go`

迭代循环中模型返回 text 时，当前没有日志。添加：
```
[agent:provider] 模型文本输出，长度 N 字符，内容: 前 500 字符
```

### 1.3 记录注入的提醒消息

文件: `backend/internal/modules/agent/service.go`

`searchOnlyCount` 提醒注入时，当前只打"搜索过多提醒触发"。改为同时打印注入的完整消息内容。

### 1.4 迭代汇总

文件: `backend/internal/modules/agent/service.go`

循环结束时打印汇总：总轮次、成功编辑次数、失败次数、最终 HTML 长度。

## 改动 2：移除 CSS 正则拦截

文件: `backend/internal/modules/agent/tool_executor.go`

### 2.1 删除 validateResumeEditFragment 函数

删除 lines 593-606 的 `validateResumeEditFragment` 函数，以及 lines 573-591 的 `bannedCSSPatterns` 常量。

### 2.2 删除调用处

删除 Phase 1 设计守卫检查（line 389 附近的 `validateResumeEditFragment` 调用）。

### 2.3 保留的验证

- `old_string` 必须在当前 HTML 中存在（逻辑正确性）
- `new_string` 不能为空
- 操作数量限制

## 改动 3：CSS 规范写入技能描述

文件: `backend/internal/modules/agent/skills/resume-design/skill.yaml`

### 3.1 问题

模型调用 `resume-design` 只拿到技能描述（name, description, trigger, usage），需要再调 `get_skill_reference` 才能拿到 `a4-guidelines.yaml` 中的 CSS 规范。模型从不走第二步。

### 3.2 方案

在 `skill.yaml` 的 `description` 字段中直接包含 CSS 规范摘要。模型调用一次 `resume-design` 就能看到禁止规则。

摘要内容（来自 a4-guidelines.yaml）：
- 推荐：纯色背景、深灰/黑色正文、单一克制强调色、单栏或常规双栏、紧凑间距
- 禁止：渐变背景、backdrop-filter、动画、text-shadow、复杂 box-shadow、position fixed/absolute、vh/vw 布局、glassmorphism、aurora、3D 效果

完整规范仍通过 `get_skill_reference` 获取。

## 改动 4：改进错误消息

文件: `backend/internal/modules/agent/tool_executor.go`

### 4.1 old_string 不匹配

当前: `"old_string not found in current draft"`
改为: 包含 old_string 前 100 字符和当前 HTML 前 200 字符，方便模型定位问题。

### 4.2 其他验证

`new_string` 为空和操作数量超限的错误消息保持不变。

## 涉及文件

| 文件 | 改动类型 |
|---|---|
| `backend/internal/modules/agent/tool_executor.go` | 删除 CSS 验证、改进错误消息、改进日志 |
| `backend/internal/modules/agent/service.go` | 改进日志（模型文本、提醒内容、迭代汇总） |
| `backend/internal/modules/agent/skills/resume-design/skill.yaml` | description 加入 CSS 规范摘要 |
