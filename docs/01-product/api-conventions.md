# ResumeGenius API 规约

更新时间：2026-04-23

本文档定义所有模块的统一 API 规范。所有模块必须遵循此规约，不得自行定义冲突规则。

## 1. 总则

### 1.1 风格

- RESTful API，JSON 序列化
- HTTP 方法语义：GET 读取、POST 创建、PUT 全量更新、DELETE 删除

### 1.2 版本

- 所有 API 路径以 `/api/v1/` 开头
- v1 稳定后不在此版本路径下做破坏性变更

### 1.3 字符编码

- 请求和响应统一 UTF-8
- Content-Type: `application/json; charset=utf-8`

## 2. 路径规范

### 2.1 格式

```
/api/v1/{module}/{resource}
```

各模块前缀：

| 模块 | 前缀 | 示例 |
|---|---|---|
| A 资料接入 | `/api/v1/projects/`、`/api/v1/assets/` | `GET /api/v1/projects/{project_id}` |
| B 解析初稿 | `/api/v1/parsing/` | `POST /api/v1/parsing/parse` |
| C AI 对话 | `/api/v1/ai/` | `POST /api/v1/ai/sessions/{session_id}/chat` |
| D 可视化编辑 | `/api/v1/drafts/` | `GET /api/v1/drafts/{draft_id}` |
| E 版本导出 | `/api/v1/drafts/{draft_id}/`（drafts 子资源） | `POST /api/v1/drafts/{draft_id}/export` |

### 2.2 命名规则

- 路径用小写 + 短横线：`/ai-sessions`
- 资源名用复数：`/projects`、`/drafts`
- 路径参数用单数：`/projects/{project_id}`
- ID 类参数带 `_id` 后缀：`project_id`、`draft_id`

## 3. 统一响应格式

### 3.1 成功响应

```json
{
  "code": 0,
  "data": { ... },
  "message": "ok"
}
```

### 3.2 列表响应（带分页）

```json
{
  "code": 0,
  "data": {
    "items": [ ... ],
    "total": 42,
    "page": 1,
    "page_size": 20
  },
  "message": "ok"
}
```

### 3.3 错误响应

```json
{
  "code": 40001,
  "data": null,
  "message": "文件格式不支持"
}
```

## 4. 错误码体系

### 4.1 错误码结构

纯数字分段编码：

- 通用错误码：40000、40001、40100、40300、40400、40900、50000
- 模块 A（资料接入）：1001–1999
- 模块 B（解析初稿）：2001–2999
- 模块 C（AI 对话）：3001–3999
- 模块 D（可视化编辑）：4001–4999
- 模块 E（版本导出）：5001–5999

各模块在 contract.md 中定义自己的错误码明细。

### 4.2 通用错误码（00xxx）

| 错误码 | HTTP 状态 | 含义 |
|---|---|---|
| 0 | 200 | 成功（`code` 字段类型为整数，固定值 `0`） |
| 40000 | 400 | 请求参数错误 |
| 40001 | 400 | 数据校验失败 |
| 40100 | 401 | 未认证（v1 预留） |
| 40300 | 403 | 无权限（v1 预留，用于 PDF 导出权限控制） |
| 40400 | 404 | 资源不存在 |
| 40900 | 409 | 资源冲突 |
| 50000 | 500 | 服务内部错误 |

### 4.3 模块错误码

| 模块 | 编号范围 | 示例 |
|---|---|---|
| A 资料接入 | 1001–1999 | 1001 = 文件格式不支持 |
| B 解析初稿 | 2001–2999 | 2001 = PDF 解析失败 |
| C AI 对话 | 3001–3999 | 3001 = 模型调用超时 |
| D 可视化编辑 | 4001–4999 | 4001 = 草稿不存在 |
| E 版本导出 | 5001–5999 | 5001 = PDF 导出失败 |

各模块在 contract.md 中定义自己的错误码明细。

## 5. 分页规范

- 请求参数：`page`（从 1 开始）、`page_size`（默认 20，最大 100）
- 排序参数：`sort_by`（字段名）、`sort_order`（`asc` / `desc`）
- 不支持游标分页（v1 不需要）

## 6. 认证

- v1 不实现认证，所有 API 无需 token
- 所有请求头预留 `Authorization: Bearer <token>` 位
- 认证上线后，未携带 token 的请求返回 40100
- PDF 导出端点预留付费权限校验位（40300）

## 7. 流式响应

### 7.1 SSE 模式（AI 对话）

AI 对话使用 Server-Sent Events（SSE）流式响应：

```
POST /api/v1/ai/sessions/{session_id}/chat
Accept: text/event-stream

Response:
data: {"type": "text", "content": "好的，我来帮你..."}
data: {"type": "text", "content": "建议将项目经历精简为："}
data: {"type": "html_start"}
data: {"type": "html_chunk", "content": "<div class=\"resume\">..."}
data: {"type": "html_end"}
data: {"type": "done"}
```

事件类型：
- `text`：AI 文字说明（逐字流式）
- `html_start`：HTML 内容开始标记
- `html_chunk`：HTML 内容片段
- `html_end`：HTML 内容结束标记
- `done`：响应完成

## 8. 异步任务

### 8.1 模式

长时间任务（PDF 导出）采用异步模式：

1. 客户端 POST 触发任务
2. 服务端立即返回任务 ID
3. 客户端轮询任务状态

### 8.2 任务创建响应

```json
{
  "code": 0,
  "data": {
    "task_id": "task_abc123",
    "status": "pending"
  },
  "message": "ok"
}
```

### 8.3 任务状态查询

`GET /api/v1/tasks/{task_id}`

```json
{
  "code": 0,
  "data": {
    "task_id": "task_abc123",
    "status": "completed",
    "progress": 100,
    "result": { ... }
  },
  "message": "ok"
}
```

任务状态枚举：`pending` → `processing` → `completed` / `failed`

> `GET /api/v1/tasks/{task_id}` 归属于模块 E（详见 E 契约）。

## 9. 请求/字段命名

- JSON 字段统一用 `snake_case`：`project_id`、`created_at`
- 日期时间用 ISO 8601：`2026-04-22T20:00:00Z`
- 布尔值用 `is_` / `has_` 前缀：`is_visible`、`has_education`
- 枚举值用 `snake_case`：`type = "resume_pdf"`

## 10. 各模块 API 端点

详细端点定义见各模块的 `contract.md` 文件。本文档只定义规约，不定义具体端点。
