# 模块 B 工作明细：文件解析与 AI 初稿生成

更新时间：2026-04-23

本文档列出模块 B 负责人的全部开发任务。契约定义见 [contract.md](./contract.md)。

## 1. 概述

模块 B 接收 A 产出的资产记录，对各类资料进行解析提取文本，然后调用 AI 生成完整简历 HTML 存入 drafts 表。

**核心交付**：用户上传的简历文件能被正确解析，AI 自动生成可编辑的初始简历 HTML。

## 2. 前端任务

### 2.1 页面

| # | 页面 | 路由建议 | 说明 |
|---|---|---|---|
| 1 | 解析结果页 | `/projects/[id]/parsing` | 显示解析进度 + 解析结果 |
| 2 | 初稿确认页 | `/projects/[id]/draft` | AI 初稿生成后的确认，展示 HTML 预览 |

### 2.2 组件

| # | 组件 | 说明 |
|---|---|---|
| 1 | `ParseStatus` | 解析状态指示（解析中 / 成功 / 失败） |
| 2 | `ParsedContent` | 解析出的文本内容预览 |
| 3 | `DraftPreview` | 初稿 HTML 预览（iframe 或直接渲染） |
| 4 | `GenerateButton` | 触发 AI 初稿生成的按钮 |

### 2.3 前端技术要点

- 解析和初稿生成都是同步请求（5-15 秒），前端显示 loading 状态
- 初稿预览可以直接渲染 HTML 或用 iframe
- 解析失败时显示错误信息和重试按钮

## 3. 后端任务

### 3.1 API 端点（2 个）

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 1 | POST | `/api/v1/parsing/parse` | 触发解析（同步） |
| 2 | POST | `/api/v1/parsing/generate` | 触发 AI 初稿生成（同步） |

### 3.2 后端服务

| # | 服务 | 说明 |
|---|---|---|
| 1 | `ParsingService` | 解析任务编排（选策略、分发、汇总结果） |
| 2 | `PdfParser` | ledongthuc/pdf 解析 PDF，提取文本块和内嵌图片 |
| 3 | `DocxParser` | nguyenthenguyen/docx 解析 DOCX，提取段落/表格/样式 |
| 4 | `GitExtractor` | clone 仓库 → 抽 README + 项目名 + 技术栈 + 目录结构 |
| 5 | `DraftGenerator` | 文本 → AI Prompt → AI 返回 HTML → 存入 drafts 表 |

### 3.3 后端技术要点

- PDF 解析：提取文本块（按顺序拼接）和内嵌图片（base64）
- DOCX 解析：提取段落文本（按顺序拼接）
- 解析结果不持久化到数据库，直接传给 AI
- AI 调用通过 ProviderAdapter 封装，返回完整 HTML
- 初稿生成失败时不创建 draft 记录，返回错误
- AI 初稿生成成功后自动创建版本快照（调用模块 E 的版本创建逻辑），响应中返回 version_id

## 4. 数据库表

| 表名 | 说明 |
|---|---|
| `drafts` | 简历草稿（id, project_id, html_content, created_at, updated_at） |

- 模块 B 负责创建 drafts 记录
- 模块 D 负责更新 html_content（自动保存）
- 模块 E 负责创建 versions 快照

## 5. 测试任务

### 5.1 后端单元测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | PDF 解析 | 用 fixtures/sample_resume.pdf 测试文本提取 |
| 2 | DOCX 解析 | 用 fixtures/sample_resume.docx 测试段落提取 |
| 3 | Git 抽取 | 用公开仓库测试 README + 技术栈提取 |
| 4 | 补充文本 | 直接使用 content 字段 |
| 5 | AI 初稿生成 | 用 mock AI 返回预设 HTML，验证存入 drafts 表 |
| 6 | 无资产 | 项目没有任何资产时返回错误 |
| 7 | 解析失败 | 损坏文件返回错误码 |

### 5.2 前端测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 解析结果展示 | 文本预览正确显示 |
| 2 | 初稿预览 | HTML 预览正确渲染 |
| 3 | 错误处理 | 解析/生成失败时显示错误信息 |

### 5.3 Mock 策略

- 不依赖 A 的服务：直接读 fixtures/ 中的测试文件
- AI 调用用 mock 替代：返回预设的 HTML
- 不需要 C/D/E 的服务
- 本地测试文件放 fixtures/：sample_resume.pdf, sample_resume.docx

## 6. 交付 Checklist

- [ ] 前端：2 个页面 + 4 个组件
- [ ] 后端：2 个 API 端点
- [ ] 后端服务：5 个核心服务（3 个解析器 + 1 个编排 + 1 个初稿生成）
- [ ] 数据库：使用 drafts 表
- [ ] 测试：7 个后端单元测试 + 3 个前端测试
- [ ] 解析策略覆盖：PDF / DOCX / Git / 补充文本
- [ ] 错误码使用 2001–2999 范围
