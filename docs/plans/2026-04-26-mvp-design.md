# MVP 设计文档 — 2026-04-26

## 概述

ResumeGenius MVP 采用"全模块骨架 + 核心体验优先"策略，5 人团队每人负责一个模块的全栈闭环，1-2 周交付。

## 团队与分工

| 成员 | 模块 | 核心职责 |
|---|---|---|
| 人1 | workbench — TipTap 编辑器 | 所见即所得编辑、A4 画布、自动保存 |
| 人2 | render — 版本管理 + PDF 导出 | 版本快照、chromedp PDF 导出 |
| 人3 | parsing — 文件解析 + AI 初稿 | PDF 解析、AI API 调用、HTML 初稿生成 |
| 人4 | agent — AI 对话助手 | SSE 流式对话、HTML 提取、应用到简历 |
| 人5 | intake — 项目管理 + 文件上传 | 项目 CRUD、文件上传、资产管理 |

## 开发阶段

### 阶段 1：共享基石（Day 1-2，1 人）

产出清单：

```
backend/
  go.mod                          # Gin + GORM + chromedp + pdf + docx
  cmd/server/main.go              # Gin 入口，注册所有路由组
  internal/shared/
    models/                       # 6 张表的 GORM 模型
    database/                     # PostgreSQL 连接 + AutoMigrate
    response/                     # 统一响应 {code, data, message}
    middleware/                    # CORS + 日志中间件
  internal/modules/               # 5 个空模块目录
frontend/
  workbench/
    package.json                  # Vite + React + TS + Tailwind + shadcn/ui + TipTap
    vite.config.ts
    src/
      main.tsx
      App.tsx                     # 路由定义
      lib/api-client.ts           # 统一 API client
      components/ui/              # shadcn/ui 基础组件
fixtures/
  sample_draft.html               # mock HTML 简历
  sample_ai_response.json         # mock AI 响应
docker-compose.yml                # postgres + gin
```

关键约定：
- API 路由：每个模块的 `routes.go` 接收 `*gin.RouterGroup`
- 统一响应：`response.Success(data)` / `response.Error(code, message)`
- 数据库：GORM AutoMigrate
- 前端：统一 `apiClient` 封装

### 阶段 2：并行开发（Day 3-12，5 人）

每人独立开发一个模块，全栈闭环（前端页面 + 后端 API + 单元测试）。

### 阶段 3：集成打磨（Day 13-14，全员）

联调、修 bug、部署。

## 各模块 MVP 功能范围

### 模块 intake — 项目管理 + 文件上传

包含：项目 CRUD（创建/列表/删除）、单文件上传（PDF/DOCX）、文件存本地 `./uploads/`

不包含：项目状态流转、多文件批量上传、Git URL 导入、补充文本输入、项目搜索/筛选

### 模块 parsing — 文件解析 + AI 初稿

包含：PDF 文本提取、调用 AI API 生成 HTML 初稿、返回 HTML 写入 drafts 表并自动创建 version

不包含：DOCX 解析（v2 补）、Git 仓库解析、多轮优化初稿、结构化信息提取

### 模块 agent — AI 对话

包含：创建会话、SSE 流式对话（text + html_start/chunk/end + done）、"应用到简历"（整体替换 HTML）

不包含：多会话管理、HTML diff/patch、选择性应用部分修改、对话历史翻页

### 模块 workbench — TipTap 编辑器

包含：TipTap 基础编辑（粗体/斜体/列表/标题）、A4 画布预览（210mm x 297mm CSS）、自动保存（debounce 2s）、简单工具栏

不包含：高级扩展（表格/图片/链接）、拖拽布局、手动保存按钮、自定义主题

### 模块 render — 版本管理 + PDF 导出

包含：创建版本快照、版本列表、chromedp 导出 PDF、下载 PDF 文件

不包含：版本对比 diff、版本回退、并发导出队列、导出参数自定义

## 模块间接口

> 以 `docs/modules/*/contract.md` 为唯一契约来源。

```
intake:    POST   /api/v1/projects                          → 创建 project
intake:    POST   /api/v1/assets/upload                      → 上传文件（multipart，返回 asset_id）
parsing:   POST   /api/v1/parsing/parse                      → 输入 project_id，输出 parsed_contents
parsing:   POST   /api/v1/parsing/generate                   → 输入 project_id，输出 draft_id + html_content
workbench: GET    /api/v1/drafts/{draft_id}                  → 获取草稿 HTML
workbench: PUT    /api/v1/drafts/{draft_id}                  → 保存草稿 HTML（自动保存）
agent:     POST   /api/v1/ai/sessions                        → 创建会话（绑定 draft_id）
agent:     POST   /api/v1/ai/sessions/{session_id}/chat      → SSE 流式对话
agent:     GET    /api/v1/ai/sessions/{session_id}/history   → 获取对话历史
render:    GET    /api/v1/drafts/{draft_id}/versions         → 版本列表
render:    POST   /api/v1/drafts/{draft_id}/versions         → 创建快照
render:    POST   /api/v1/drafts/{draft_id}/export           → 触发 PDF 导出
render:    GET    /api/v1/tasks/{task_id}                    → 查询导出状态
```

## 前端路由

```
/                    → 项目列表（模块 intake）
/projects/new        → 创建项目 + 上传文件（模块 intake + parsing）
/editor/:projectId   → 编辑器页面（模块 workbench 主页面）
                       左侧: A4 画布（TipTap）
                       右侧: AI 面板（模块 agent，可收起）
                       顶部: 操作栏（保存/版本/导出 → 模块 render）
```

## Mock 策略

- 后端：直接连 PostgreSQL，不需要 mock 数据库
- 前端：开发初期用 MSW（Mock Service Worker）拦截 API，后端就绪后切换
- AI 调用：`USE_MOCK=true` 环境变量，返回 fixtures 中的 mock 响应

## 集成验证里程碑

| 天数 | 里程碑 | 验证内容 |
|---|---|---|
| Day 3-4 | 各模块独立可运行 | 单元测试通过 |
| Day 8 | workbench+render 集成 | 编辑器能保存 + 导出 PDF |
| Day 10 | intake+parsing+workbench 集成 | 上传文件 → 解析 → 编辑 |
| Day 12 | intake+parsing+agent+workbench+render 全链路 | 完整用户流程 |
| Day 13-14 | 打磨 | Bug 修复 + 部署 |

## 裁剪原则

每个模块只做 happy path，砍掉所有边缘场景。5 人各 2 周内完成。砍掉的功能在 v2 补充。
