# AI Agent 系统化改进设计

> 日期: 2026-05-12
> 状态: 设计确认，待实施
> 目标: 解决 apply_edits 反复失败问题，引入 OA-Reader 可复用的架构模式

---

## 一、问题分析

### 1.1 核心问题

AI 反复使用 apply_edits 失败，表现多样：搜索匹配失败、陷入搜索循环、修改后 HTML 破损。这不是单一 bug，而是系统性的架构缺陷。

### 1.2 根因

1. **错误反馈太薄** — apply_edits 失败时只返回 "未找到匹配" 字符串，AI 不知道当前 HTML 长什么样，无法调整策略
2. **上下文膨胀** — get_draft 返回完整 HTML（3000-8000 字符），多次调用后占满上下文窗口
3. **缺乏恢复策略** — 没有降级方案，AI 只能盲目重试
4. **AI 缺乏结构化工作流** — 不该上来就改，应该先理解、再对齐需求、最后执行

---

## 二、改进方案

### 2.1 工具执行层：错误恢复机制

#### 2.1.1 结构化错误反馈

当 `strings.Contains(html, op.OldString)` 匹配失败时：

- 用 goquery 遍历所有文本节点，找到包含 op.OldString 最多公共子串的节点
- 截取该节点所在元素的 OuterHTML（±200 字符）
- 如果完全找不到相似内容，返回 draft 的 structure 模式概览
- 返回包含以下信息的错误消息：
  - AI 搜索的内容
  - 失败原因（可能：缩进/换行不一致、HTML 已被修改、片段跨越标签边界）
  - 建议（使用更短唯一片段重新搜索）
  - 附近 HTML 片段供参考

#### 2.1.2 apply_edits 分步执行

- 逐个执行 op，失败的 op 立即返回错误（含上下文）
- 已成功的 ops 保留（不回滚）
- 错误反馈中明确告知 AI 哪些 op 已成功、哪个失败

#### 2.1.3 JSON 参数解析重试

当 AI 返回的 tool_call 参数 JSON 解析失败时：
- 将错误信息 + 正确格式示例注入 tool_result
- 让 LLM 自我修正（最多重试 3 次）

### 2.2 get_draft 增强：多模式查询

增强 get_draft 工具，支持多种查询模式：

```go
type GetDraftParams struct {
    Mode     string `json:"mode"`                // structure | section | search | full
    Selector string `json:"selector,omitempty"`  // mode=section 时使用
    Query    string `json:"query,omitempty"`     // mode=search 时使用
}
```

| 模式 | 用途 | 返回内容 |
|------|------|---------|
| `structure` | 了解简历结构 | HTML 标签树概览，格式如：`<html><head><style>(CSS 摘要)</style></head><body><div.header><div.experience><div.education>...</div></body></html>`，不包含文本内容 |
| `section` | 获取指定区域 | 按 CSS selector 返回指定片段 |
| `search` | grep 式搜索 | 包含关键词的 HTML 片段（±200字符上下文），最多返回 5 条匹配 |
| `full` | 完整 HTML | 完整内容，会被截断到 5000 字符 |

**典型流程**：
1. 首次 `get_draft(mode="structure")` → 获得结构概览
2. `get_draft(mode="section", selector=".experience")` → 获取精确片段
3. `apply_edits` → 执行修改

### 2.3 三层上下文管理

#### Layer 1: Token 预算实时监控

```go
type ContextBudget struct {
    maxTokens         int     // 128000
    warnThreshold     float64 // 0.75
    compactThreshold  float64 // 0.85
    blockThreshold    float64 // 0.95
}
```

每次调用 LLM 前检查 token 估算：
- 75%: SSE 发送 `compact_warning` 事件
- 85%: 触发压缩
- 95%: 阻断新消息，强制压缩

#### Layer 2: 工具输出差异化截断

| 工具 | 截断策略 | 阈值 |
|------|---------|------|
| get_draft (full) | 按段落边界截断，保留 head style + body 首尾 | 5000 字符 |
| get_draft (structure) | 不截断 | -- |
| get_draft (section) | 不截断 | -- |
| get_draft (search) | 限制匹配数，每条 ±200 字符 | 5 条 |
| apply_edits 结果 | 硬截断 | 1000 字符 |
| search_assets | 限制条数 + 每条摘要长度 | 10 条 × 300 字符 |
| 技能内容 | 不截断 | -- |
| 错误消息 | 硬截断 | 2000 字符 |

#### Layer 3: 压缩改进

- 使用结构化摘要模板（对话摘要 / 关键结论 / 待继续事项）
- 压缩失败 fallback：保留原始消息 + warn 日志
- 压缩后 SSE 发送 `compact_done` 事件

### 2.4 AI 结构化工作流

将 AI 工作流从"被动响应"改为"主动引导"（借鉴 Superpowers 流程）：

```
Phase 1: 理解上下文（必须）
  → get_draft(mode="structure")
  → search_assets 了解用户背景
  → 理解用户意图

Phase 2: 需求对齐（非原子修改必须）
  → 复杂修改：给出 2-3 种方案 + 权衡
  → 等待用户确认
  → 原子修改（改词、调间距）可跳过

Phase 3: 执行修改
  → get_draft(mode="section") 获取精确片段
  → apply_edits 执行
  → 失败 → 恢复策略 → 降级

Phase 4: 总结
  → 告知修改了什么
  → 后续建议
```

**System Prompt 规则**：
- 首次进入会话必须先调用 get_draft(mode="structure")
- 复杂修改（多区域、结构调整）必须分步执行、逐步确认
- 原子修改可跳过方案展示，直接执行

### 2.5 Agent 循环策略改进

#### 2.5.1 步骤漏调检测

定义关键步骤序列：`get_draft → search_assets → skill → apply_edits → summary`

连续 N 轮没有调用 apply_edits（而用户请求的是修改）时：
1. 第 1 次：注入提醒消息
2. 第 2 次：加强提醒
3. 第 3 次：降级放行（AI 返回纯文本解释，结束循环）

#### 2.5.2 工具失败恢复策略

apply_edits 连续失败 2 次时：
- 自动注入当前 HTML 结构摘要
- 提供附近 HTML 片段
- 第 3 次仍失败 → 降级为文本建议

#### 2.5.3 API 调用重试

LLM API 调用失败（网络超时、5xx）时：
- 指数退避重试，最多 3 次（间隔 1s/2s/4s）
- 429 (rate limit) 不重试，直接返回错误事件

#### 2.5.4 降级放行策略

| 场景 | 降级行为 |
|------|---------|
| apply_edits 连续 3 次失败 | AI 返回文本建议 |
| 压缩失败 | 保留原始消息，warn 日志 |
| LLM API 3 次重试仍失败 | error SSE 事件 |
| 步骤漏调连续 3 次 | 降级放行，纯文本 |

### 2.6 可观测性体系

#### 2.6.1 slog 结构化日志

替换所有 `debugLog` 调用为 `log/slog`：

| 级别 | 场景 |
|------|------|
| DEBUG | 回复预览、工具参数详情 |
| INFO | 对话完成（含统计）、工具执行、会话创建 |
| WARN | 压缩失败回退、搜索匹配降级、LLM 重试 |
| ERROR | 工具调用异常、数据库错误 |

#### 2.6.2 requestID 链路追踪

Gin 中间件为每个请求生成 `X-Request-ID`，注入到 context 中的 logger。

#### 2.6.3 AI 决策链路追踪

每一轮 ReAct 迭代产出一个结构化 trace 日志，包含：
- `iteration`: 当前轮次
- `reasoning_summary`: AI 思考摘要
- `tool_calls`: 调用的工具列表及参数
- `tool_results`: 工具执行结果摘要
- `decision`: AI 的下一步决策

示例：
```
[requestID:abc123] [session:42] ReAct iteration 1/7
  → reasoning: "用户想修改教育经历..."
  → tool_calls: [get_draft(mode=section, selector=.education)]
  → tool_result: get_draft → 832 chars, 0.04s
  → decision: need more context
```

#### 2.6.4 降级日志模式

所有 fallback/降级场景统一用 warn 级别记录，包含：
- 失败原因
- 降级到什么策略
- 关键上下文

#### 2.6.5 敏感信息脱敏

- API Key: 前4后4
- Token/JWT: 前12字符
- 密码: 不记录

### 2.7 接口抽象与测试体系

#### 2.7.1 核心接口抽象

```go
type LLMProvider interface {
    StreamChatReAct(ctx context.Context, req LLMRequest) (<-chan LLMEvent, error)
}

type HTMLRepository interface {
    GetDraft(ctx context.Context, draftID uint) (string, error)
    ApplyEdits(ctx context.Context, draftID uint, ops []EditOp) ([]EditResult, error)
    GetDraftStructure(ctx context.Context, draftID uint) (string, error)
    SearchInDraft(ctx context.Context, draftID uint, query string) (string, error)
    GetDraftSection(ctx context.Context, draftID uint, selector string) (string, error)
}

type AssetRepository interface {
    SearchAssets(ctx context.Context, projectID uint, query string, assetType string) ([]Asset, error)
}
```

保留现有 `ProviderAdapter` 和 `ToolExecutor` 接口，新增 Repository 层接口。

#### 2.7.2 测试分层

| 层级 | 覆盖 | 依赖 |
|------|------|------|
| 单元测试 | ChatService 循环逻辑、token 估算、截断策略、错误恢复 | Mock LLM + Mock Repo |
| 集成测试 | apply_edits 端到端、会话持久化、压缩流程 | 真实 DB |
| E2E 测试 | SSE 流式对话全链路 | httptest + Mock LLM |

#### 2.7.3 关键测试用例

- apply_edits 搜索失败 → 错误反馈包含上下文片段
- 连续 3 次 apply_edits 失败 → 降级为文本建议
- Token 超过 85% → 触发压缩
- 压缩失败 → fallback 保留原始消息
- get_draft mode=structure → 返回标签树概览
- get_draft mode=search → grep 搜索返回片段
- 步骤漏调检测 → 注入提醒
- LLM API 失败 → 指数退避重试

---

## 三、实施优先级

按依赖关系和影响排序：

1. **P0 — 工具执行层改进**（错误反馈、get_draft 增强、分步执行）
2. **P1 — 可观测性**（slog 替换、requestID、决策链路追踪）
3. **P1 — 上下文管理**（token 预算、截断策略、压缩改进）
4. **P2 — Agent 循环策略**（步骤漏调、恢复策略、API 重试）
5. **P2 — 接口抽象**（Repository 层、依赖注入改造）
6. **P3 — 测试体系**（单元 → 集成 → E2E）

---

## 四、不在本次范围内

以下 OA-Reader 模式经评估不适合本次改进：

- **两步式记忆系统** — 简历项目的信息收集通过 preloadAssets 已基本满足，不需要独立的 Extract+Merge 记忆系统
- **用户画像** — 不适用
- **并发控制（Semaphore）** — 当前只有 LLM 一个外部调用，不需要多 lane 并发控制
- **会话双表设计** — 当前 AISession + AIMessage 已够用，列表查询性能不是瓶颈
