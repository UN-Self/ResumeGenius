# Agent Tool System 实施计划

创建时间：2026-05-02 | 更新：2026-05-02（修正 v2）

## 目标

将 agent 模块从"简单对话透传"升级为"具备工具调用能力的 ReAct Agent"。AI 能主动获取项目资料、生成并保存简历 HTML、创建版本快照、触发 PDF 导出，完成"用户上传文件 → 解析 → AI 生成简历 → 渲染导出 PDF"的全链路闭环。

## 当前状态

### 已实现

| 模块 | 端点/能力 | 说明 |
|---|---|---|
| agent | 6 REST + SSE 流式 | Session CRUD、Chat 透传、History |
| parsing | `POST /api/v1/parsing/parse` | PDF/DOCX/Note → `ParsedContent[]` |
| render | 6 端点（版本 + 导出） | 版本快照、回退、chromedp PDF 异步导出 |
| 前端 | ChatPanel + HtmlPreview | 对话 UI、HTML iframe 预览、应用到简历 |

### agent 全链路缺口

- AI 无法获取用户上传的资料（parsing 数据）→ 对话缺少上下文
- AI 生成的 HTML 只预览不持久化 → 无法保存到 draft
- 无法创建版本快照 → 无历史追踪
- 无法触发 PDF 导出 → 全链路断裂
- 无 ReAct 推理循环 → AI 只能单次回复
- 无思考过程追踪 → 推理黑盒

## Contract 变更说明

当前 [contract.md](../../docs/modules/agent/contract.md)（2026-05-01 版）需更新以支持 ReAct Tool 模式。变更内容如下：

### 变更 1：后端需解析 AI 输出

| 项目 | 当前 contract | 变更后 | 原因 |
|---|---|---|---|
| 后端职责 | "不解析 AI 输出，透传 SSE" | 解析 AI 输出中的 `tool_calls`，执行工具后回传结果 | ReAct 模式要求后端拦截 tool_call、执行、回传 tool_result，否则循环无法进行 |

**注意**：HTML 提取仍然由前端负责（`<!--RESUME_HTML_START-->` / `<!--RESUME_HTML_END-->`），这部分不变。后端仅新增解析 `delta.tool_calls` 的能力。

### 变更 2：新增 SSE 事件类型

| 事件 | 当前 | 变更后 | 说明 |
|---|---|---|---|
| `text` | 已存在 | 不变 | AI 最终文本回复 |
| `error` | 已存在 | 不变 | 错误 |
| `done` | 已存在 | 不变 | 完成 |
| `thinking` | — | **新增** | AI 推理链（ReAct think 阶段），流式 |
| `tool_call` | — | **新增** | AI 发起工具调用，前端可显示状态 |
| `tool_result` | — | **新增** | 工具执行完成，前端消除状态 |

### 变更 3：草稿保存流程

| 操作 | 当前 contract | 变更后 | 说明 |
|---|---|---|---|
| AI 保存 HTML | 前端提取 HTML → 用户点"应用到简历" → 前端调 PUT /drafts/:id | AI 调用 `save_draft` tool 直接保存 | AI 生成后自动持久化，编辑器实时刷新 |
| HTML Preview | 前端从 SSE text 中提取 | 不变（保留） | AI 回复中仍含 HTML 标记，前端可预览 |

用户仍可通过版本回退撤销不满意的 AI 修改（render 模块已有 rollback 端点）。

### 不删除的现有端点

以下端点保持不变，不受本次变更影响：
- `POST /api/v1/ai/sessions` — 创建会话
- `GET /api/v1/ai/sessions` — 列表
- `GET /api/v1/ai/sessions/{id}` — 详情
- `DELETE /api/v1/ai/sessions/{id}` — 删除
- `GET /api/v1/ai/sessions/{id}/history` — 历史

### 不删除的现有错误码

3001–3004 保持不变，新增 3005。

---

## 架构设计

### Agent 模块内部分层

```
┌──────────────────────────────────────────────┐
│              agent module                     │
│                                              │
│  handler.go ── SSE 事件输出                   │
│      │                                       │
│  service.go ── ReAct 循环编排                 │
│      │                                       │
│  ┌──────────┐  ┌──────────────────┐          │
│  │ provider │  │  tool_executor   │          │
│  │ (AI 调用)│  │  (工具执行)      │          │
│  └──────────┘  └──────┬───────────┘          │
│                       │                      │
│          ┌────────────┼────────────┐         │
│          ▼            ▼            ▼         │
│     [DB 直查]   [parsing API]  [render API]  │
│     draft/proj    HTTP :8080    HTTP :8080   │
│     asset 查询    /parsing/*    /drafts/*     │
└──────────────────────────────────────────────┘
```

**关键决策**：agent 不直接 import parsing/render 的 Go package。对于简单查询直接用 `*gorm.DB`，对于复杂业务逻辑通过 HTTP 调用同进程内的 parsing/render 端点。不跨模块引用，保持模块边界清晰。

### ReAct 循环（最大 3 轮）

```
用户消息
  │
  ▼
第 1 轮 ── AI think → tool_call(parse_project_assets) → tool_result
  │
  ▼
第 2 轮 ── AI think → tool_call(update_draft_html) → tool_result
  │
  ▼
第 3 轮 ── AI think → 最终文本回复 + HTML
  │
  ▼
SSE done
```

超过 3 轮仍未产出最终回复 → 强制终止，返回 `3005: max iterations exceeded`。

## Phase 1：数据模型

### 1.1 扩展 AISession

```go
type AISession struct {
    ID          uint        `gorm:"primaryKey" json:"id"`
    DraftID     uint        `gorm:"not null;index" json:"draft_id"`
    ProjectID   *uint       `gorm:"index" json:"project_id"`          // NEW
    Status      string      `gorm:"size:20;not null;default:'active'" json:"status"` // NEW
    CreatedAt   time.Time   `json:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at"`                      // NEW
}
```

### 1.2 扩展 AIMessage

```go
type AIMessage struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    SessionID   uint      `gorm:"not null;index" json:"session_id"`
    Role        string    `gorm:"size:20;not null" json:"role"`        // user | assistant | tool
    Content     string    `gorm:"type:text" json:"content"`            // 文本内容（tool_call 时可为空）
    Thinking    *string   `gorm:"type:text" json:"thinking,omitempty"` // NEW：AI 推理链
    ToolCallID  *uint     `gorm:"index" json:"tool_call_id,omitempty"` // NEW
    CreatedAt   time.Time `json:"created_at"`
}
```

### 1.3 新建 AIToolCall

```go
type AIToolCall struct {
    ID          uint       `gorm:"primaryKey" json:"id"`
    SessionID   uint       `gorm:"not null;index" json:"session_id"`
    ToolName    string     `gorm:"size:100;not null" json:"tool_name"`
    Params      JSONB      `gorm:"type:jsonb;not null" json:"params"`
    Result      *JSONB     `gorm:"type:jsonb" json:"result,omitempty"`
    Status      string     `gorm:"size:20;not null;default:'pending'" json:"status"`
    Error       *string    `gorm:"type:text" json:"error,omitempty"`
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}
```

### 1.4 数据存储位置

| 数据 | 存储位置 | 说明 |
|---|---|---|
| 用户输入 | `ai_messages.content`（role=user） | 每轮对话一条记录 |
| AI 文本回复 | `ai_messages.content`（role=assistant） | 含 HTML 标记 |
| AI 推理过程 | `ai_messages.thinking`（role=assistant） | ReAct think 阶段文本 |
| 工具调用记录 | `ai_tool_calls` 表 | 工具名、参数、结果、耗时 |
| 会话元信息 | `ai_sessions` 表 | 关联 draft_id、project_id、状态 |

### 1.5 迁移

- GORM AutoMigrate 自动新增字段和表
- `project_id` / `thinking` / `tool_call_id` 均为 nullable，旧数据不受影响
- session 删除时级联删除 messages + tool_calls

## Phase 2：Tool 系统

### 2.1 Tool 定义

| # | Tool Name | 数据来源 | 参数 | 返回值 |
|---|---|---|---|---|
| 1 | `get_project_assets` | `*gorm.DB` 直查 assets 表 | `project_id: uint` | `{assets: [{id, type, label, content, uri}]}` |
| 2 | `parse_project_assets` | HTTP → `POST /api/v1/parsing/parse` | `project_id: uint` | `{parsed_contents: [{asset_id, type, label, text}]}` |
| 3 | `get_draft` | `*gorm.DB` 直查 drafts 表 | `draft_id: uint` | `{draft_id, html_content, updated_at}` |
| 4 | `save_draft` | `*gorm.DB` 直写 drafts 表 | `draft_id: uint, html_content: string` | `{draft_id, updated: true}` |
| 5 | `create_version` | HTTP → `POST /api/v1/drafts/:id/versions` | `draft_id: uint, label: string` | `{version_id, label, created_at}` |
| 6 | `export_pdf` | HTTP → `POST /api/v1/drafts/:id/export` | `draft_id: uint, html_content: string` | `{task_id, status: "pending"}` |

### 2.2 模块间调用安全

**不跨模块 import**。agent 不引用 `parsing` 或 `render` 的 Go package。

- Tool 1/3/4 使用 agent 已有的 `*gorm.DB`，直接操作 `shared/models`（与 agent 现有代码一致）
- Tool 2 通过 `net/http` 调用 `POST /api/v1/parsing/parse`（同进程内 HTTP，localhost）
- Tool 5/6 通过 `net/http` 调用 render 端点

这保证：
- 不会出现循环 import
- 不会跨模块直接引用对方的结构体，模块边界清晰
- 安全审计时 API 调用链路可追踪（agent → parsing → db 而非 agent → db 旁路）

**为什么可以通过 HTTP 调用同进程服务？** 所有模块运行在同一个 Go 进程内，内网 `127.0.0.1:8080` 调用延迟 < 1ms，且不需要认证（v1 无认证）。更重要的是，这尊重了每个模块的 REST API 契约，不绕过接口直接操作内部实现。

### 2.3 ToolExecutor 接口

```go
// agent/tool_executor.go
type ToolDef struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

type ToolExecutor interface {
    // Tools returns the tool definitions for function calling.
    Tools() []ToolDef
    // Execute runs a tool and returns its result.
    Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error)
}
```

`AgentToolExecutor` 持有 `*gorm.DB` 和 `*http.Client`（指本地 baseURL），不 import 其他模块。

## Phase 3：ReAct Agent 循环

### 3.1 流程

```
1. 构建 messages：
   - system prompt（角色 + 工具说明 + 输出格式要求）
   - 历史 messages（本 session 已有记录）
   - 用户当前消息

2. 调用 AI 模型（携带 tools 定义 + stream=true）

3. 解析 SSE 流：
   ┌─ reasoning_content → 记录到 ai_messages.thinking，SSE 发 thinking 事件
   ├─ tool_calls         → 记录到 ai_tool_calls，执行工具，结果追加到 messages
   │                        回到步骤 2（最多 3 轮）
   └─ content            → 记录到 ai_messages.content，SSE 发 text 事件 → 结束

4. 发送 done 事件
```

### 3.2 迭代限制

- 最大 3 轮（环境变量 `AGENT_MAX_ITERATIONS=3`）
- 每轮超时 120s（同现有 OpenAIAdapter）
- 超限后返回 error: `3005: max tool-calling iterations (3) exceeded`

### 3.3 AI System Prompt

```
你是一个专业的简历助手。你的任务是根据用户提供的资料和要求，生成一份完整的、可直接渲染的HTML简历。

## 工作流程

1. 如果用户提到了项目/文件，首先调用 parse_project_assets 获取资料内容
2. 如果需要查看当前草稿，调用 get_draft 获取最新的简历 HTML
3. 分析资料内容，按照用户要求的格式和内容生成完整的简历 HTML
4. 生成后调用 save_draft 保存到草稿
5. 完成后，报告用户你做了哪些修改

## 轮次限制（极其重要）

你最多只有 **3 轮** 思考和工具调用机会。这意味着：
- 第 1-2 轮：获取资料、分析内容、构建简历结构
- 第 3 轮：**无论如何必须产出第一版完整简历 HTML**，即使信息不完整

如果第 3 轮结束时你还没有调用 save_draft 保存简历，本次对话将失败。
不要追求完美——先用已有信息生成一版可用的简历，用户可以后续让你修改。
信息不足时，用合理的推断填充，并在回复中说明哪些部分是推断的。

## 输出格式

- 生成的 HTML 必须是一份完整的、独立的 HTML 文档（含 <!DOCTYPE html>、CSS 样式）
- 页面尺寸为 A4（210mm × 297mm），使用 @page { size: A4; margin: 0; }
- 使用语义化 HTML 标签（header、section、h1-h3、ul/li）
- CSS 必须内联在 <style> 标签中，不引用外部资源
- 字体：font-family: 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif
- 简历内容应简洁、专业，突出关键信息
- 仅在用户明确要求时才创建版本快照或导出 PDF

## 重要规则

- 生成 HTML 前必须先获取资料或查看当前草稿
- 生成 HTML 后必须调用 save_draft 保存
- 不要编造用户没有提供的信息（信息不足时合理推断并标注）
- 如果资料不足以生成完整简历，在第 3 轮直接生成最佳可用版本
```

### 3.4 Provider Adapter 扩展

```go
type ProviderAdapter interface {
    // 现有：向后兼容的简单流式对话
    StreamChat(ctx context.Context, messages []Message, sendChunk func(string) error) error

    // NEW：ReAct 流式对话（支持 function calling）
    StreamChatReAct(
        ctx context.Context,
        messages []Message,
        tools []ToolDef,
        onReasoning func(chunk string) error,  // AI 推理链回调
        onToolCall func(call ToolCallRequest) error, // 工具调用回调
        onText func(chunk string) error,       // 文本回调
    ) error
}
```

豆包 2.0 Pro 模型兼容 OpenAI function calling 协议，完全支持此接口。

## Phase 4：思考过程记录

### 4.1 三层记录

| 层级 | 方式 | 默认 | 环境变量 | 用途 |
|---|---|---|---|---|
| DB | `ai_messages.thinking` 字段 | ON | — | 持久化，可在前端回放 |
| 文件 | `.thinking/{session_id}.md` 实时写入 | OFF | `AGENT_THINKING_FILE=true` | 防止推理中断丢失 |
| 日志 | `log.Printf` 输出 | OFF | `AGENT_THINKING_LOG=true` | 开发调试 |

### 4.2 SSE 事件类型

| type | 方向 | payload | 说明 |
|---|---|---|---|
| `thinking` | → | `{"type":"thinking","content":"..."}` | AI 推理链，流式 |
| `tool_call` | → | `{"type":"tool_call","name":"parse_project_assets","params":{...}}` | AI 发起工具调用 |
| `tool_result` | → | `{"type":"tool_result","name":"parse_project_assets","status":"completed"}` | 工具执行结果摘要 |
| `text` | → | `{"type":"text","content":"..."}` | 最终文本回复（同现有） |
| `error` | → | `{"type":"error","code":3005,"message":"..."}` | 错误（扩展错误码） |
| `done` | → | `{"type":"done"}` | 完成（同现有） |

### 4.3 ThinkingRecorder

```go
type ThinkingRecorder struct {
    sessionID uint
    filePath  string    // .thinking/{session_id}.md
    logger    *log.Logger
    mu        sync.Mutex
    file      *os.File  // nil if file mode disabled
}
```

`Sync()` 每次写入后强制刷盘，防止进程异常退出导致数据丢失。

## Phase 5：前端适配

### 5.1 ChatPanel 新增

1. **思考过程折叠区**：`thinking` 事件流式追加，默认折叠，用户可展开查看 AI 推理过程
2. **工具调用指示器**：显示 `<Spinner /> + "正在分析项目资料..."`，`tool_result` 后消失
3. **全链路步骤条**（可选）：解析资料 → 生成简历 → 保存草稿 → 导出 PDF

### 5.2 新 SSE 事件解析

```typescript
case 'thinking':
  setThinking(prev => prev + event.content)
  break
case 'tool_call':
  setActiveTool(`正在执行：${TOOL_LABELS[event.name] || event.name}`)
  break
case 'tool_result':
  setActiveTool(null)
  break
```

### 5.3 修复已有 bug

修复 HTML 跨 chunk 提取 bug（ChatPanel.tsx:111-133）：`inHTML` 分支需要追加新 chunk 到 `currentHTML`。

## Phase 6：错误码扩展

| 错误码 | HTTP | 含义 |
|---|---|---|
| 3001 | 504 | 模型调用超时 |
| 3002 | 500 | 模型返回格式异常 |
| 3003 | 404 | 会话不存在 |
| 3004 | 400 | 草稿不存在 |
| 3005 | 422 | 超过最大工具调用轮数（3 轮） |

## Phase 7：实施步骤

### Step 0：更新 Contract（预计 0.5 天，先于代码实施）
- [ ] 更新 `docs/modules/agent/contract.md`：
  - 后端职责：新增"解析 AI 输出中的 tool_calls"
  - SSE 事件类型：新增 `thinking`、`tool_call`、`tool_result`
  - 草稿保存：AI 通过 tool 直接保存，前端仍保留 HTML 预览
  - 轮次限制：最大 3 轮
  - 错误码：新增 3005
- [ ] 与上下游模块确认变更影响

### Step 1：数据模型（预计 1 天）
- [ ] 扩展 `models.go`：AISession + AIMessage + 新建 AIToolCall
- [ ] AutoMigrate 验证
- [ ] 编写 model 单元测试

### Step 2：Tool 执行器（预计 1 天）
- [ ] 创建 `agent/tool_executor.go`（ToolDef + ToolExecutor 接口 + AgentToolExecutor）
- [ ] 实现 6 个 tool：DB 直查（1/3/4）+ HTTP 调用（2/5/6）
- [ ] 单元测试（mock HTTP server）

### Step 3：Provider 升级（预计 1 天）
- [ ] 扩展 `ProviderAdapter` 接口
- [ ] `OpenAIAdapter` 实现 function calling 流式协议
- [ ] `MockAdapter` 实现 ReAct mock（模拟 thinking + tool_call + text 序列）
- [ ] 单元测试

### Step 4：ReAct 循环 + ThinkingRecorder（预计 1 天）
- [ ] `ChatService.StreamChatReAct` 实现
- [ ] `ThinkingRecorder` 实现（DB + 文件 + 日志）
- [ ] 3 轮限制 + 错误处理
- [ ] 集成测试（MockAdapter 模拟完整 ReAct 流程）

### Step 5：Handler + Routes 适配（预计 0.5 天）
- [ ] 更新 `routes.go`：组装 ToolExecutor + ThinkingRecorder
- [ ] 更新 `handler.go`：Chat 端点使用 ReAct 循环
- [ ] 新的 SSE 事件类型输出
- [ ] 向后兼容旧客户端

### Step 6：前端适配（预计 0.5 天）
- [ ] ChatPanel：新增 thinking 折叠区
- [ ] ChatPanel：新增 tool_call 状态指示器
- [ ] 修复 HTML 跨 chunk 提取 bug
- [ ] 解析新 SSE 事件类型

### Step 7：集成测试（预计 0.5 天）
- [ ] 全链路 mock 测试：用户消息 → parse → AI 生成 → save draft → export PDF
- [ ] 异常场景：tool 失败、超时、3 轮限制
- [ ] SSE 事件序列验证

## 文件变更清单

```
backend/internal/shared/models/models.go          # 扩展 AISession/AIMessage + 新建 AIToolCall
backend/internal/modules/agent/
  routes.go                                         # 组装 ToolExecutor + ThinkingRecorder 依赖
  handler.go                                        # Chat 端点 → ReAct 循环 + 新 SSE 事件
  service.go                                        # ChatService → ReAct 循环编排
  provider.go                                       # ProviderAdapter 接口 + OpenAI function calling
  tool_executor.go                                  # NEW：Tool 定义 + 执行器（DB + HTTP）
  thinking_recorder.go                              # NEW：思考记录三层写入
  service_test.go                                   # 扩展
  provider_test.go                                  # 扩展
  tool_executor_test.go                             # NEW
  thinking_recorder_test.go                         # NEW

docs/modules/agent/contract.md                      # 更新契约
docs/01-product/api-conventions.md                   # 更新 SSE 事件类型（Section 7.1）
docs/modules/workbench/contract.md                   # 更新 Section 7（agent 新增直接保存路径）
docs/modules/parsing/contract.md                     # 标注 POST /api/v1/parsing/generate 待实现/废弃

frontend/workbench/src/components/chat/
  ChatPanel.tsx                                     # 新增 thinking/tool_call 事件处理 + 修复 bug
```

总计 4.5 天，6 个新增/修改的 Go 文件 + 1 个前端文件 + 4 个文档文件。

---

## 跨模块契约冲突分析

### 冲突 1：`api-conventions.md` Section 7.1 — SSE 事件类型过时 ⚠️ 需修复

| 项目 | 当前值 | agent 变更后 | 影响 |
|---|---|---|---|
| SSE 事件 | `text`, `html_start`, `html_chunk`, `html_end`, `done` | `text`, `thinking`, `tool_call`, `tool_result`, `error`, `done` | API 规约是全局规范，SSE 事件定义已落后于 agent contract |

**解决方案**：将 `api-conventions.md` Section 7.1 改为引用 agent contract 作为 SSE 权威来源，避免两处定义不一致：

```markdown
## 7. 流式响应

### 7.1 SSE 模式（AI 对话）

AI 对话使用 Server-Sent Events（SSE）流式响应。SSE 事件类型以
[docs/modules/agent/contract.md](../modules/agent/contract.md) 为准。

主要事件：`text`（文本）、`thinking`（推理链）、`tool_call`（工具调用）、
`tool_result`（工具结果）、`error`（错误）、`done`（完成）。
```

### 冲突 2：`parsing/contract.md` — `POST /api/v1/parsing/generate` 未实现 ⚠️ 待决策

contract 定义了 `POST /api/v1/parsing/generate`（AI 初稿生成），但 `parsing/routes.go` 中只有 `POST /api/v1/parsing/parse`。agent 升级后，初稿生成职责转移到 agent 的 ReAct 循环（AI 调用 `save_draft` tool）。

| 选项 | 操作 | 影响 |
|---|---|---|
| A（推荐） | parsing contract 标记 `generate` 为 deprecated，注明由 agent 接管 | agent 全权负责生成，parsing 专注解析 |
| B | 实现 `generate` 端点，agent tool 额外调用 | 增加复杂度，初稿 + 对话两个入口 |

**推荐选项 A**。parsing 保持"纯解析"角色（输入文件，输出文本），agent 负责所有 AI 生成逻辑。符合模块边界。

### 冲突 3：`workbench/contract.md` Section 7 — agent 保存路径变更 ℹ️ 需注明

当前写的是"模块 agent 通过 AI 对话修改后，前端替换 HTML"。agent 升级后新增一条路径：

| 路径 | 触发方式 | 保存入口 |
|---|---|---|
| 原有（保留） | 前端"应用到简历"按钮 | `PUT /api/v1/drafts/{draft_id}`（workbench） |
| 新增 | AI 调用 `save_draft` tool | 后端直写 `drafts` 表（agent） |

**解决方案**：在 workbench contract 中注明 agent 模块可能直接写入 `drafts.html_content`（通过 tool），workbench 的 GET 端点可读取到最新内容，编辑器通过轮询或事件刷新。

### 无冲突确认 ✅

| 模块 | 检查项 | 结论 |
|---|---|---|
| intake | agent 不调用 intake 端点，不操作 assets 写入 | 无冲突 |
| render | agent 通过 HTTP 调用 render 端点（create_version / export_pdf），与标准客户端行为一致 | 无冲突 |
| api-conventions Section 4 | agent 新增 3005 在 3001-3999 范围内 | 无冲突 |
| 路由注册 | agent 的 6 个端点 `/api/v1/ai/*` 与其他模块无路径重叠 | 无冲突 |

---

## 后续演进（v2）

- Plan-and-Execute 模式：先规划步骤，再逐步执行（替代 ReAct）
- 多轮对话上下文管理
- Tool 权限控制（HTML 修改类 tool 需要用户确认）
- HTML Diff/Patch：AI 只返回修改片段
- 流式 tool result（大文件逐步返回解析内容）
