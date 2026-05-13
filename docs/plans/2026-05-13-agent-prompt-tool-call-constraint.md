# 2026-05-13 强化 Agent Prompt 工具调用约束

## 问题

用户在 AI 对话中要求修改简历时，glm-4.7 模型有时返回纯文本声称"已修改"但未调用 `apply_edits` 工具，导致用户看不到任何实际变化。用户确认（"可以"/"好的"）时，模型将其理解为任务完成而非继续执行指令。

**日志证据：**
- 17:03:03 — 用户说"每个部分靠太远了，占三页"，模型返回 475 字符文本声称"已压缩到一页"，但无 tool_call/tool_result 事件
- 17:09:12 — 用户说"可以"，stall 保护触发 2 次后模型返回纯文本确认

## 根因

1. System prompt 的 `ironRulesSection` 缺少强制工具调用的明确约束
2. `tool_choice: "auto"` 让模型自主决定是否调用工具
3. 用户确认性输入被模型理解为任务结束信号

## 方案

**变体 1：纯 prompt 强化**（已选定）

只修改 `backend/internal/modules/agent/prompt.go` 的 `ironRulesSection`，新增"工具调用铁律"段落。

参考 Claude-Code 的 MUST/NEVER/禁止 指令风格，Claude-Code 验证该风格对模型指令遵从性有效。

## 设计

### 改动文件

`backend/internal/modules/agent/prompt.go` — `ironRulesSection` 常量

### 新增内容

在现有核心铁律之后追加：

```
## 工具调用铁律
- 任何对简历的修改（包括调整样式、压缩布局、修改内容、改变字号、优化排版等）MUST 通过调用 apply_edits 工具完成，禁止仅用文字声称已修改而不调用工具
- apply_edits 是搜索替换，不是全文重写：old_string 必须精确匹配当前 HTML 中的已有片段
- 禁止说"已修改"/"已调整"/"已优化"/"已压缩"却未调用 apply_edits — 这是严重错误，用户看不到任何变化
- 如果修改涉及多处，可以在一次 apply_edits 中提交多个 ops
- 用户说"可以"/"好的"/"确认"/"继续"等确认性回复时，如果上一轮你提出了修改建议但未通过 apply_edits 执行，MUST 立即调用 apply_edits 执行该修改
- 只有当用户明确表示满意且你已通过 apply_edits 完成了所有修改后，才可返回纯文本回复
```

### 设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| 改动位置 | `ironRulesSection` | 最高优先级段落，模型最先看到 |
| 检测方式 | 纯 prompt | 对标 Claude-Code 做法，零代码侵入 |
| 语言风格 | MUST/NEVER/禁止 | Claude-Code 验证有效的指令风格 |
| 确认处理 | 明确"确认≠完成" | 解决日志中"可以"不触发工具的问题 |

### 不做的事

- 不改 `service.go` — 不增加代码层面的检测逻辑
- 不改 `provider.go` — 不修改 `tool_choice` 参数
- 不改 `tool_executor.go` — 不修改工具定义
- 不改前端 — 不需要新的 SSE event 类型

## 验收标准

1. 用户要求修改简历时，模型必须调用 `apply_edits` 工具
2. 用户说"可以"/"好的"确认时，如果上一轮有未执行的修改建议，模型应立即调用 `apply_edits`
3. 模型不得在未调用 `apply_edits` 的情况下声称已修改简历
