# 模块 C 工作明细：AI 对话助手

更新时间：2026-04-23

本文档列出模块 C 负责人的全部开发任务。契约定义见 [contract.md](./contract.md)。

## 1. 概述

模块 C 是 AI 编辑链路的核心，负责多轮对话、AI 流式响应，以及 HTML 替换确认。AI 读取当前简历 HTML 作为上下文，用户描述需求后 AI 返回修改后的完整 HTML。

**核心交付**：用户能通过自然语言对话修改简历，AI 流式返回 HTML 预览，确认后替换编辑器内容。

## 2. 前端任务

### 2.1 组件

| # | 组件 | 说明 |
|---|---|---|
| 1 | `ChatPanel` | AI 对话面板容器（右侧栏），可收起/展开 |
| 2 | `ChatBubble` | 消息气泡：区分用户消息和 AI 回复 |
| 3 | `ChatInput` | 消息输入框：多行输入 + 发送按钮 |
| 4 | `StreamRenderer` | 流式输出渲染：逐字显示 AI 回复 |
| 5 | `HtmlPreview` | AI 返回的 HTML 预览（内嵌渲染或 iframe） |
| 6 | `ApplyActions` | 操作按钮："应用到简历" / "继续对话" |

### 2.2 前端技术要点

- SSE 连接管理：创建、监听、错误处理、重连
- AI 回复逐字流式显示
- 通过 `<!--RESUME_HTML_START-->` / `<!--RESUME_HTML_END-->` 分隔符提取 HTML
- 提取到 HTML 后渲染预览，显示"应用到简历"和"继续对话"按钮
- 点击"应用到简历"调用 `PUT /api/v1/drafts/{draft_id}`，携带 `create_version: true` 和 `version_label` 参数
- 对话面板可收起/展开
- AI 正在回复时禁用输入框

## 3. 后端任务

### 3.1 API 端点（3 个）

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 1 | POST | `/api/v1/ai/sessions` | 创建对话会话 |
| 2 | POST | `/api/v1/ai/sessions/{session_id}/chat` | 发送消息（SSE 流式） |
| 3 | GET | `/api/v1/ai/sessions/{session_id}/history` | 获取对话历史 |

### 3.2 后端服务

| # | 服务 | 说明 |
|---|---|---|
| 1 | `SessionService` | 会话 CRUD，关联 draft |
| 2 | `ChatService` | 消息收发，对话历史管理，SSE 流式透传 |
| 3 | `ProviderAdapter` | AI 模型调用封装（GLM-5 / GLM-5-Turbo），Prompt 构建 |

### 3.3 后端技术要点

- SSE 响应：Gin 框架的 `c.Stream()` 或 `c.Writer.Flush()` 实现流式输出
- 每次 AI 调用传入：当前 HTML + 对话历史（最近 N 轮）+ 用户消息
- AI 返回流式文本，后端透传为 SSE 事件
- 后端不解析 AI 输出，HTML 提取由前端负责
- 对话历史存 ai_messages 表
- Prompt 模板：角色定义 + 输出格式要求（先文字后 HTML，用分隔符标识）

### 3.4 AI Prompt 设计

| # | Prompt | 说明 |
|---|---|---|
| 1 | System Prompt | 角色定义（简历优化助手）+ 输出格式（先文字说明，再 HTML，用分隔符标识） |
| 2 | Context Injection | 注入当前 HTML + 对话历史 |
| 3 | User Message | 包裹用户消息 |

## 4. 数据库表

| 表名 | 说明 |
|---|---|
| `ai_sessions` | AI 对话会话（id, draft_id, created_at） |
| `ai_messages` | AI 对话消息（id, session_id, role, content, created_at） |

## 5. 测试任务

### 5.1 后端单元测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 会话创建 | 创建会话关联 draft |
| 2 | 消息收发 | 发送消息 → SSE 流式响应 |
| 3 | 对话历史 | 历史消息正确返回 |
| 4 | 模型超时 | 模拟 AI 调用超时，返回错误 |
| 5 | Prompt 构建 | 验证不同场景下 prompt 注入的上下文正确 |

### 5.2 前端测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 聊天界面 | 消息发送、气泡渲染 |
| 2 | 流式输出 | 逐字显示 AI 回复 |
| 3 | HTML 提取 | 正确从分隔符中提取 HTML |
| 4 | 应用操作 | 点击"应用到简历"→ 调用 API → 编辑器更新 |

### 5.3 Mock 策略

- AI 调用用 mock handler 替代：返回预设的流式文本 + HTML
- 不需要 B/D/E 的服务
- 当前 HTML 用 fixtures/sample_draft.html

## 6. 交付 Checklist

- [ ] 前端：6 个组件（集成在工作台右侧面板）
- [ ] 后端：3 个 API 端点
- [ ] 后端服务：3 个核心服务（会话 + 聊天 + ProviderAdapter）
- [ ] AI Prompt：1 个 system prompt + context injection 模板
- [ ] 数据库：2 张表（ai_sessions, ai_messages）
- [ ] 测试：5 个后端单元测试 + 4 个前端测试
- [ ] SSE 流式响应正确实现
- [ ] 错误码使用 3001–3999 范围
