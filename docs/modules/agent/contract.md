# 模块 agent 契约：AI 对话助手

更新时间：2026-05-01

## 1. 角色定义

**负责**：

- 多轮对话会话管理
- AI 流式响应（SSE）
- AI 读取当前简历 HTML 作为上下文
- AI 返回修改后的完整 HTML
- 用户确认后替换当前草稿

**不负责**：

- 文件解析（parsing 的事）
- 所见即所得编辑（workbench 的事）
- 版本管理和 PDF 导出（render 的事）

## 2. 输入契约

| 数据 | 来源 | 说明 |
|---|---|---|
| `drafts.html_content` | 模块 parsing/workbench | 当前简历 HTML |
| 用户自然语言消息 | 前端输入 | — |
| 对话历史 | `ai_messages` 表 | 最近 N 轮 |

Mock：直接用 `fixtures/sample_draft.html` 作为当前 HTML。

## 3. 输出契约

AI 返回两部分内容：

1. **自然语言回复**：对用户的文字说明
2. **修改后的 HTML**：完整的简历 HTML（可直接替换编辑器内容）

### AI 消息格式约定

assistant 消息的 content 字段格式：

```
好的，我帮你精简了项目经历部分：
主要压缩了描述文字，保留核心量化指标。

<!--RESUME_HTML_START-->
<!DOCTYPE html>...完整 HTML...
<!--RESUME_HTML_END-->
```

前端通过 `<!--RESUME_HTML_START-->` 和 `<!--RESUME_HTML_END-->` 分隔符提取 HTML 部分。

如果 AI 没有修改简历（只是回答问题），则不包含 HTML 部分。

## 4. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/ai/sessions` | 创建对话会话 |
| GET | `/api/v1/ai/sessions` | 查询会话列表（按 draft_id 过滤） |
| GET | `/api/v1/ai/sessions/{session_id}` | 获取单个会话详情 |
| DELETE | `/api/v1/ai/sessions/{session_id}` | 删除会话（级联删除消息） |
| POST | `/api/v1/ai/sessions/{session_id}/chat` | 发送消息（SSE 流式） |
| GET | `/api/v1/ai/sessions/{session_id}/history` | 获取对话历史 |

### 关键端点详情

#### POST /api/v1/ai/sessions

```
Request:
{
  "draft_id": 1
}

Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "draft_id": 1,
    "created_at": "2026-04-23T20:00:00Z"
  }
}
```

#### POST /api/v1/ai/sessions/{session_id}/chat

SSE 流式响应。后端不解析 AI 输出，将所有 chunk 统一透传为 `text` 事件。
HTML 提取由前端根据 `<!--RESUME_HTML_START-->` / `<!--RESUME_HTML_END-->` 标记完成。

```
Request:
{
  "message": "帮我把工作经历压缩得更精炼一点"
}

Response: text/event-stream
data: {"type": "text", "content": "好的，我帮你精简了工作经历部分。\n\n"}
data: {"type": "text", "content": "<!--RESUME_HTML_START-->\n<!DOCTYPE html><html>..."}
data: {"type": "text", "content": "...</html>\n<!--RESUME_HTML_END-->"}
data: {"type": "text", "content": "主要压缩了描述文字，保留核心量化指标。"}
data: {"type": "done"}
```

SSE 事件类型：

| type | payload | 说明 |
|---|---|---|
| `text` | `{"type":"text","content":"..."}` | AI 输出内容（逐字流式透传，含 HTML 标记） |
| `error` | `{"type":"error","code":3003,"message":"..."}` | 出错，code 为模块错误码 |
| `done` | `{"type":"done"}` | 响应完成 |

error 事件的 `code` 字段对应本模块错误码（见第 7 节），前端据此做差异化处理（如 3003 提示重新创建会话，3001 提示重试等）。

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
        "content": "帮我把工作经历压缩得更精炼一点",
        "created_at": "2026-04-23T20:00:00Z"
      },
      {
        "id": 2,
        "role": "assistant",
        "content": "好的，我帮你精简了...\n\n<!--RESUME_HTML_START-->...<!--RESUME_HTML_END-->",
        "created_at": "2026-04-23T20:00:05Z"
      }
    ]
  }
}
```

## 5. AI 调用规范

### 5.1 模型

- v1 通过 Provider Adapter 调用 OpenAI-compatible API
- 通过 Provider Adapter 封装

### 5.2 输入构建

每次 AI 调用传入：

- 当前 `drafts.html_content`（完整 HTML）
- 对话历史（最近 N 轮 user/assistant 消息）
- 用户当前消息

### 5.3 输出解析

AI 返回流式文本，后端透传给前端。后端不解析 AI 输出，只负责：
1. 组装 Prompt
2. 调用模型 API
3. 将流式响应透传为 SSE

HTML 提取由前端负责。

### 5.4 用户确认替换

用户在 AI 面板点击"应用到简历"后，前端调用模块 workbench 的 `PUT /api/v1/drafts/{draft_id}` 替换 HTML，请求中携带 `create_version: true` 和 `version_label: "AI 修改：{用户需求摘要}"` 以触发版本快照创建。模块 agent 不负责 HTML 替换和版本创建。

## 6. 依赖与边界

### 上游

- 模块 parsing 产出初始 drafts.html_content
- 模块 workbench 更新 drafts.html_content

### 下游

- 无。模块 agent 只负责对话和 AI 调用。

### 可 mock 的边界

- AI 调用用 mock handler 替代
- 不需要 parsing/workbench/render 的服务

## 7. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 3001 | 504 | 模型调用超时 |
| 3002 | 500 | 模型返回格式异常 |
| 3003 | 404 | 会话不存在 |
| 3004 | 400 | 草稿不存在 |

## 8. 测试策略

### 独立测试

- AI 调用用 mock handler 替代
- 测试多轮对话流程
- 测试 SSE 流式响应格式
- 测试异常场景：模型超时、连接断开

### 前端测试

- 对话 UI（聊天气泡）
- 流式输出显示
- "应用到简历" / "继续对话"按钮交互
