# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## TOP RULES

MUST USE SUPERPOWER
MUST SHOW A FEW PLAN AND ALSO RECOMMEND LOWER TECH-DEBT ONES
USE TDD TO DEVELOP THE PROJECT (REG)

## 项目概览

ResumeGenius 是一个 AI 辅助简历编辑产品。核心理念：**HTML 是唯一数据源**，AI 直接操作 HTML，用户通过 TipTap 所见即所得编辑器直接编辑简历，最终通过 chromedp 服务端渲染导出 PDF。

## 常用命令

### 数据库

```bash
docker compose up -d postgres          # 启动 PostgreSQL
docker compose exec postgres psql -U postgres -d resume_genius  # 连接数据库
```

### 后端（Go）

```bash
cd backend
go run cmd/server/main.go              # 启动 API 服务 :8080（自动连接 DB + AutoMigrate）
go test ./...                          # 运行全部测试
go test ./internal/shared/models/... -v  # 运行指定包测试
go build ./cmd/server/...              # 编译检查
```

### 前端工作台（Vite + React）

```bash
cd frontend/workbench
bun install                            # 安装依赖（使用 bun，不用 npm）
bun run dev                            # 开发服务器 :3000（/api 代理到 :8080）
bun run build                          # 生产构建
bunx vitest run                        # 运行全部前端测试
bunx vitest run tests/api-client.test.ts  # 运行单个测试文件
```

### 营销站（Astro）

```bash
cd frontend/marketing
bun install
bun run dev                            # 开发服务器
bun run build                          # 构建纯静态 HTML 到 dist/
```

### 环境变量

后端通过环境变量配置（见 `backend/internal/shared/database/database.go`）：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DB_HOST` | localhost | PostgreSQL 主机 |
| `DB_PORT` | 5432 | PostgreSQL 端口 |
| `DB_USER` | postgres | 数据库用户 |
| `DB_PASSWORD` | postgres | 数据库密码 |
| `DB_NAME` | resume_genius | 数据库名 |
| `USE_MOCK` | false | AI 调用使用 mock 响应 |
| `AI_API_URL` | — | AI 模型 API 地址 |
| `AI_API_KEY` | — | AI 模型 API 密钥 |

## 架构：v2（HTML 单一数据源）

```
[A 项目管理] → 文件/资料 → [B 解析初稿] → HTML 初稿
                                       │
                              ┌─────────┴─────────┐
                              ▼                   ▼
                        [C AI 对话]          [D TipTap 编辑]
                        AI 返回 HTML          直接编辑 HTML
                              │                   │
                              └─────────┬─────────┘
                                        ▼
                              [E 版本管理 + PDF 导出]
                                HTML 快照 / chromedp
```

零中间层：HTML 直接存数据库，直接编辑，直接导出。

### 后端架构

Gin 路由分组注册，每个模块统一签名 `func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB)`：

```
/api/v1/projects       → a_intake    （项目管理）
/api/v1/assets/*       → a_intake    （文件上传、Git 接入、补充文本）
/api/v1/parsing/*      → b_parsing   （解析、AI 初稿生成）
/api/v1/ai/*           → c_agent     （SSE 流式 AI 对话）
/api/v1/drafts/*       → d_workbench （TipTap 编辑、草稿 CRUD）
/api/v1/drafts/*       → e_render    （版本快照）
/api/v1/tasks/*        → e_render    （PDF 导出任务）
```

**路由与端点定义以 docs/modules/*/contract.md 为唯一契约来源。**

共享层 `backend/internal/shared/`：models（6 表 GORM 结构体）、database（连接 + AutoMigrate）、response（统一响应）、middleware（CORS + Logger）。

### 前端架构

两个独立前端项目：

- `frontend/workbench/` — React SPA，Vite 构建，路径别名 `@/` → `src/`，测试在 `tests/` 目录
- `frontend/marketing/` — Astro 静态站，SEO 纯 HTML 输出

部署路由：`/` → 营销站，`/app/*` → 工作台，`/api/*` → Go 后端。

### 数据库

6 张核心表：projects → drafts → versions, projects → assets, drafts → ai_sessions → ai_messages

循环外键引用（Project.CurrentDraft ↔ Draft）通过 GORM `DisableForeignKeyConstraintWhenMigrating: true` 处理。

## 技术栈

| 层 | 技术 |
|---|---|
| 营销站（SEO） | Astro |
| 工作台（编辑器） | Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui |
| 富文本编辑器 | TipTap（基于 ProseMirror） |
| 后端 | Go + Gin + GORM |
| 数据库 | PostgreSQL >= 15 |
| PDF 解析 | ledongthuc/pdf / nguyenthenguyen/docx（纯 Go） |
| PDF 导出 | chromedp |
| AI 模型 | OpenAI-compatible API（Provider Adapter 解耦） |
| 包管理器 | bun（前端），go mod（后端） |

## API 规约要点

- 统一前缀 `/api/v1/{module}/{resource}`
- 响应格式 `{code: 0, data: {...}, message: "ok"}`，错误码 5 位 `SSCCC`（SS=模块 01-05）
- JSON 字段 `snake_case`，日期 ISO 8601
- AI 对话用 websocket 流式响应（event types: text, html_start, html_chunk, html_end, done）
- PDF 导出用异步任务模式
- v1 无认证

## 文档体系

契约驱动开发，文档是 source of truth：

- **共享规范层** `docs/01-product/`：tech-stack、api-conventions、ui-design-system
- **数据模型** `docs/02-data-models/`：core-data-model、mock-fixtures
- **模块契约** `docs/modules/{a-intake,...,e-render}/`：各模块 contract.md + work-breakdown.md
- **实施计划** `docs/plans/`：phase0-shared-foundation + 5 个模块计划

开发前必读顺序：tech-stack → api-conventions → ui-design-system → core-data-model → 对应模块 contract.md

## 协作规则

- 契约即文档：代码必须对齐 contract.md + core-data-model.md
- 错误码段：A=01xxx, B=02xxx, C=03xxx, D=04xxx, E=05xxx
- 前端不直接操作数据库，所有数据通过 API 获取
- Commit 消息前缀：`feat:` / `fix:` / `docs:` / `refactor:` / `test:`
