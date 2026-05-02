# 模块 agent 契约：AI 对话助手

更新时间：2026-05-02

## 1. 角色定义

**负责**：

- 多轮对话会话管理
- AI 流式响应（SSE）
- AI 读取当前简历 HTML 作为上下文
- AI 调用工具（tool calling）获取资料、保存草稿、创建版本、导出 PDF
- ReAct 推理循环（Think → Act → Observe，最大 3 轮）
- AI 返回修改后的完整 HTML
- AI 推理过程记录（DB + 可选文件/日志）

**不负责**：

- 文件解析（parsing 的事）
- 所见即所得编辑（workbench 的事）
- 版本管理和 PDF 导出（render 的事，agent 仅通过 HTTP 调用其端点）

## 2. 输入契约

| 数据 | 来源 | 说明 |
|---|---|---|
| `drafts.html_content` | 模块 parsing/workbench | 当前简历 HTML |
| 用户自然语言消息 | 前端输入 | — |
| 对话历史 | `ai_messages` 表 | 最近 N 轮 |
| 项目资产解析结果 | DB 直查 assets 表 | `get_project_assets` 工具 |

Mock：`USE_MOCK=true` 时使用 `MockAdapter` 模拟 AI 响应。

## 3. 输出契约

AI 通过 ReAct 循环完成以下动作：

1. **获取资料**：调用 `get_project_assets` tool → 后端 DB 直查 assets 表 → 返回完整资产列表及内容
2. **生成/修改 HTML**：AI 生成完整简历 HTML，通过 `save_draft` tool 直接保存到 `drafts` 表
3. **创建快照**（可选）：调用 `create_version` tool → HTTP 调用 render 端点
4. **导出 PDF**（可选）：调用 `export_pdf` tool → HTTP 调用 render 端点
5. **自然语言回复**：对用户的文字说明（SSE `text` 事件流式输出）

### AI 消息格式约定

assistant 消息的 content 字段格式保持不变：

```
好的，我帮你精简了项目经历部分：
主要压缩了描述文字，保留核心量化指标。

<!--RESUME_HTML_START-->
<!DOCTYPE html>...完整 HTML...
<!--RESUME_HTML_END-->
```

前端通过 `<!--RESUME_HTML_START-->` 和 `<!--RESUME_HTML_END-->` 分隔符提取 HTML 部分进行预览。

如果 AI 没有修改简历（只是回答问题），则不包含 HTML 部分。

## 4. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/ai/sessions` | 创建对话会话 |
| GET | `/api/v1/ai/sessions` | 查询会话列表（按 draft_id 过滤） |
| GET | `/api/v1/ai/sessions/{session_id}` | 获取单个会话详情 |
| DELETE | `/api/v1/ai/sessions/{session_id}` | 删除会话（级联删除消息 + 工具调用记录） |
| POST | `/api/v1/ai/sessions/{session_id}/chat` | 发送消息（SSE 流式，ReAct 循环） |
| GET | `/api/v1/ai/sessions/{session_id}/history` | 获取对话历史（含推理过程 + 工具调用） |

### 关键端点详情

#### POST /api/v1/ai/sessions

```
Request:
{
  "draft_id": 1,
  "project_id": 1          // NEW：可选，提供后 AI 可调用项目相关工具
}

Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "draft_id": 1,
    "project_id": 1,
    "status": "active",
    "created_at": "2026-04-23T20:00:00Z"
  }
}
```

#### POST /api/v1/ai/sessions/{session_id}/chat

SSE 流式响应。后端运行 ReAct 循环（最大 3 轮），解析 AI 输出中的 tool_calls 并执行。

```
Request:
{
  "message": "帮我根据项目中的资料生成一份简历"
}

Response: text/event-stream
data: {"type":"thinking","content":"我需要先获取项目中的资料，了解用户的背景信息。"}
data: {"type":"tool_call","name":"get_project_assets","params":{"project_id":1}}
data: {"type":"tool_result","name":"get_project_assets","status":"completed"}
data: {"type":"thinking","content":"资料显示用户有3年前端开发经验...我来生成一份合适的简历HTML。"}
data: {"type":"tool_call","name":"save_draft","params":{"draft_id":1,"html_content":"<!DOCTYPE html>..."}}
data: {"type":"tool_result","name":"save_draft","status":"completed"}
data: {"type":"text","content":"我已经根据你的资料生成了简历，主要包括..."}
data: {"type":"done"}
```

SSE 事件类型：

| type | payload | 说明 |
|---|---|---|
| `text` | `{"type":"text","content":"..."}` | AI 文本回复（逐字流式，含 HTML 标记） |
| `thinking` | `{"type":"thinking","content":"..."}` | AI 推理链（ReAct think 阶段，流式） |
| `tool_call` | `{"type":"tool_call","name":"...","params":{...}}` | AI 发起工具调用 |
| `tool_result` | `{"type":"tool_result","name":"...","status":"completed\|failed"}` | 工具执行结果 |
| `error` | `{"type":"error","code":3003,"message":"..."}` | 出错，code 为模块错误码 |
| `done` | `{"type":"done"}` | 响应完成 |

#### GET /api/v1/ai/sessions/{session_id}/history

```
Response:
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 1,
        "role": "user",
        "content": "帮我根据项目中的资料生成一份简历",
        "thinking": null,
        "created_at": "2026-04-23T20:00:00Z"
      },
      {
        "id": 2,
        "role": "assistant",
        "content": "我已经根据你的资料生成了简历...",
        "thinking": "我需要先获取项目中的资料...\n资料显示用户有3年前端开发经验...",
        "tool_call": {
          "id": 1,
          "tool_name": "get_project_assets",
          "params": {"project_id": 1},
          "result": {"parsed_contents": [...]},
          "status": "completed"
        },
        "created_at": "2026-04-23T20:00:05Z"
      }
    ]
  }
}
```

## 5. AI 调用规范

### 5.1 模型

- v1 通过 Provider Adapter 调用 OpenAI-compatible API（当前使用豆包 2.0 Pro）
- 支持 function calling 协议
- 通过 `USE_MOCK=true` 切换到 MockAdapter

### 5.2 ReAct 循环

每次用户消息触发 ReAct 循环（最大 3 轮）：

1. 构建 messages（system prompt + 历史 + 用户消息 + 已有 tool_results）
2. 调用 AI 模型（携带 tools 定义 + stream=true）
3. 解析流式响应：
   - `reasoning_content` → SSE `thinking` 事件 + 写入 `ai_messages.thinking`
   - `tool_calls` → 记录到 `ai_tool_calls`，执行工具，结果追加到 messages，回到步骤 1
   - `content`（最终文本）→ SSE `text` 事件 + 写入 `ai_messages.content`
4. 超过 3 轮仍未产出最终回复 → 返回错误 `3005`

### 5.3 System Prompt

```
你是一个专业的简历助手。你的任务是根据用户提供的资料和要求，生成一份完整的、可直接渲染的HTML简历。

## 工作流程
1. 如果用户提到了项目/文件，首先调用 get_project_assets 获取资料内容
2. 如果需要查看当前草稿，调用 get_draft 获取最新的简历 HTML
3. 分析资料内容，按照用户要求的格式和内容生成完整的简历 HTML
4. 生成后调用 save_draft 保存到草稿
5. 完成后，报告用户你做了哪些修改

## 轮次限制（极其重要）
你最多只有 **3 轮** 思考和工具调用机会：
- 第 1-2 轮：获取资料、分析内容、构建简历结构
- 第 3 轮：**无论如何必须产出第一版完整简历 HTML**，即使信息不完整
如果第 3 轮结束时你还没有调用 save_draft 保存简历，本次对话将失败。
不要追求完美——先用已有信息生成一版可用的简历，用户可以后续让你修改。
信息不足时，用合理的推断填充，并在回复中说明哪些部分是推断的。

## 输出格式
- 生成的 HTML 必须是完整的、独立的 HTML 文档（含 <!DOCTYPE html>、CSS 样式）
- 页面尺寸为 A4（210mm × 297mm），使用 @page { size: A4; margin: 0; }
- 使用语义化 HTML 标签（header、section、h1-h3、ul/li）
- CSS 内联在 <style> 标签中，不引用外部资源
- 字体：font-family: 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif
- 简历内容应简洁、专业，突出关键信息
- 仅在用户明确要求时才创建版本快照或导出 PDF

## 重要规则
- 生成 HTML 前必须先获取资料（get_project_assets）或查看当前草稿（get_draft）
- 生成 HTML 后必须调用 save_draft 保存
- 不要编造用户没有提供的信息（信息不足时合理推断并标注）
- 如果资料不足以生成完整简历，在第 3 轮直接生成最佳可用版本
```

### 5.4 可用工具

AI 可在 ReAct 循环中调用以下工具。所有工具由后端 agent 模块执行，不暴露给前端。

| Tool Name | 说明 | 参数 | 数据来源 |
|---|---|---|---|
| `get_project_assets` | 获取项目资产列表及解析内容（含 note 文本内容） | `project_id: uint` | DB 直查 assets 表 |
| `get_draft` | 获取当前草稿 HTML | `draft_id: uint` | DB 直查 drafts 表 |
| `save_draft` | 保存/更新草稿 HTML | `draft_id: uint, html_content: string` | DB 直写 drafts 表 |
| `create_version` | 创建版本快照 | `draft_id: uint, label: string` | HTTP `POST /api/v1/drafts/:id/versions` |
| `export_pdf` | 触发 PDF 异步导出 | `draft_id: uint, html_content: string` | HTTP `POST /api/v1/drafts/:id/export` |

### 5.5 输出处理

后端负责：
1. 组装 System Prompt + 注入上下文（当前 HTML、对话历史）
2. 调用模型 API（携带 tools 定义）
3. 解析流式响应中的 `tool_calls`，执行工具，回传 tool_result
4. 将最终文本响应透传为 SSE `text` 事件
5. 将推理过程透传为 SSE `thinking` 事件

前端负责：
- HTML 提取（通过 `<!--RESUME_HTML_START-->` / `<!--RESUME_HTML_END-->` 标记）
- HTML 预览渲染
- 推理过程展示（折叠区）

## 6. 依赖与边界

### 上游

- 模块 parsing 产出初始 drafts.html_content
- 模块 workbench 更新 drafts.html_content
- 模块 intake 管理项目资产（agent 通过 tool 间接获取）

### 下游

- 无。模块 agent 只负责对话和 AI 调用。

### 模块间调用

agent 通过 **同进程 HTTP** 调用其他模块的 REST 端点：
- ~~`POST /api/v1/parsing/parse`~~ 已移除，改为 DB 直查 assets 表（`get_project_assets` 工具）
- `POST /api/v1/drafts/{draft_id}/versions`（创建版本快照）
- `POST /api/v1/drafts/{draft_id}/export`（触发 PDF 导出）

agent **不直接 import** parsing/render/intake 的 Go package，保持模块边界清晰。

简单数据操作（查/写 drafts、查 assets、查 projects）使用 agent 自有的 `*gorm.DB`，与现有代码一致。

### 可 mock 的边界

- AI 调用用 MockAdapter 替代（`USE_MOCK=true`）
- 不需要 parsing/workbench/render 的服务运行

## 7. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 3001 | 504 | 模型调用超时 |
| 3002 | 500 | 模型返回格式异常 |
| 3003 | 404 | 会话不存在 |
| 3004 | 400 | 草稿不存在 |
| 3005 | 422 | 超过最大工具调用轮数（3 轮） |

## 8. 测试策略

### 独立测试

- AI 调用用 mock handler 替代
- 测试 ReAct 循环：thinking → tool_call → tool_result → text
- 测试 SSE 流式响应格式（含新增事件类型）
- 测试异常场景：模型超时、tool 执行失败、超过最大轮数
- 测试 ThinkingRecorder：DB + 文件 + 日志三层记录

### 前端测试

- 对话 UI（聊天气泡）
- 流式输出显示（text + thinking）
- 工具调用状态指示（tool_call / tool_result）
- "应用到简历"按钮交互
- HTML 预览提取

## 9. 数据模型

（参考 `backend/internal/shared/models/models.go`）

### ai_sessions

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint | 主键 |
| draft_id | uint | 关联草稿 |
| project_id | uint? | 关联项目（供 tool 使用上下文） |
| status | string | active / completed |
| created_at | timestamp | — |
| updated_at | timestamp | — |

### ai_messages

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint | 主键 |
| session_id | uint | 关联会话 |
| role | string | user / assistant / tool |
| content | text | 消息正文 |
| thinking | text? | AI 推理链（ReAct think 阶段内容） |
| tool_call_id | uint? | 关联的 ai_tool_calls 记录 |
| created_at | timestamp | — |

### ai_tool_calls（新增）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint | 主键 |
| session_id | uint | 关联会话 |
| tool_name | string | 工具名称 |
| params | jsonb | 工具参数 |
| result | jsonb? | 工具返回结果 |
| status | string | pending / running / completed / failed |
| error | text? | 错误信息 |
| started_at | timestamp? | 开始执行时间 |
| completed_at | timestamp? | 完成时间 |
| created_at | timestamp | — |
