# 开发工作总览

更新时间：2026-04-23

本文档是 5 人并行开发的总入口，定义公共工作、交付顺序和协作方式。每人读本文档 + 自己模块的 `work-breakdown.md` 即可开工。

## 1. 项目结构

```
ResumeGenius/
  frontend/
    marketing/               # Astro 营销站（SEO）
      src/
        pages/               # 落地页、功能介绍、定价、帮助文档
    workbench/               # Vite + React 工作台
      src/
        pages/               # 页面路由
        components/          # 共享组件 + 各模块组件
        lib/                 # 工具函数、API client
  backend/
    cmd/                     # 入口
    internal/
      modules/               # 5 个模块各自独立目录
        a_intake/
        b_parsing/
        c_agent/
        d_workbench/
        e_render/
      shared/                # 共享 Go 模型、工具
  fixtures/                  # 共享 mock 数据
  docs/                      # 本文档体系
```

## 2. 公共工作（所有人必做）

### 2.1 环境搭建

| 项目 | 说明 |
|---|---|
| Node.js | >= 18 LTS（前端构建用） |
| Go | >= 1.22 |
| PostgreSQL | >= 15 |
| Chromium | PDF 导出需要（chromedp 会自动管理，无需手动安装） |

### 2.2 公共文档必读

按顺序读：

1. [tech-stack.md](./tech-stack.md) — 技术选型
2. [api-conventions.md](./api-conventions.md) — API 规约
3. [ui-design-system.md](./ui-design-system.md) — UI 风格
4. [core-data-model.md](../02-data-models/core-data-model.md) — 数据库表结构
5. [mock-fixtures.md](../02-data-models/mock-fixtures.md) — Mock 策略
6. 自己模块的 `contract.md` + `work-breakdown.md`

### 2.3 公共代码

前端和后端各有一份公共部分需要先搭好：

**前端公共**（建议 D 负责搭建，其他人复用）：

| 内容 | 说明 |
|---|---|
| Astro 营销站脚手架 | 落地页、功能介绍等 SEO 页面 |
| Vite + React 工作台脚手架 | TypeScript、Tailwind CSS、shadcn/ui |
| 工作台布局 Shell | 左侧 A4 画布 + 右侧 AI 面板 + 底部工具栏 |
| TipTap 编辑器集成 | A4 画布 + 基础工具栏 |
| API Client 封装 | 统一请求/响应拦截、错误处理、SSE 支持 |
| 路由结构 | 各模块页面占位 |
| 色彩/字号/间距变量 | Tailwind config 中按 ui-design-system.md 配好 |

**后端公共**（建议 A 负责搭建，其他人复用）：

| 内容 | 说明 |
|---|---|
| Gin 项目脚手架 | 路由注册、中间件、CORS |
| Go 共享模型 | Project、Asset、Draft、Version、AISession、AIMessage |
| 统一响应格式 | `{code, data, message}` 封装 |
| 错误码注册表 | 纯数字 5 位分段格式（如 `1001`、`2001`） |
| 数据库连接 | GORM PostgreSQL |
| SSE 工具 | 流式响应封装 |

## 3. 模块交付物总览

| 模块 | 负责人 | 前端页面 | 后端 API 数 | 核心产出 |
|---|---|---|---|---|
| A 资料接入 | — | 3 页 | 10 个 | projects + assets 表操作 |
| B 解析初稿 | — | 2 页 | 2 个 | 文本提取 → AI 生成 HTML |
| C AI 对话 | — | 集成在工作台 | 3 个 | SSE 流式对话 |
| D 可视化编辑 | — | 工作台主体 | 2 个 | TipTap 集成 + 自动保存 |
| E 版本导出 | — | 弹窗/抽屉 | 5 个 | HTML 快照 + chromedp PDF |

## 4. 开发顺序建议

### 阶段一：基础搭建（所有人）

1. 搭前端脚手架（Astro 营销站 + Vite 工作台）+ 工作台布局
2. 搭后端脚手架 + 共享 Go 模型 + 数据库迁移
3. 确认 fixtures/ 目录的 mock 数据全部就位
4. 每人跑通自己模块的"hello world"页面 + API

### 阶段二：核心功能（并行）

- A：项目 CRUD + 文件上传
- B：PDF/DOCX 解析 + AI 初稿生成
- C：AI 对话 SSE 流式响应
- D：TipTap 编辑器集成 + 工具栏 + 自动保存
- E：版本快照 + chromedp PDF 导出

### 阶段三：联调（逐步）

1. A → B 联调：真实文件上传 → 解析 → AI 生成初稿
2. B → D 联调：AI 生成的 HTML → TipTap 加载编辑
3. B/C → D 联调：AI 对话返回 HTML → TipTap 替换内容
4. D → E 联调：编辑保存 → 版本管理 → PDF 导出
5. 端到端跑通：创建项目 → 上传 → 解析 → AI 生成 → 编辑 → AI 修改 → 导出 PDF

## 5. 协作规则

- **契约即文档**：API 和数据结构以 `contract.md` + `core-data-model.md` 为准，代码实现必须对齐
- **改契约要通知**：任何人修改共享 schema 或 API 格式，必须在群里通知所有模块负责人
- **Mock 优先**：开发阶段用 fixtures/ 下的 mock 数据，不依赖真实服务
- **环境变量切 mock**：用 `USE_MOCK=true/false` 控制是否使用 mock 数据
- **前端不直接操作数据库**：所有数据通过 API 获取
- **错误码不冲突**：A=1001–1999, B=2001–2999, C=3001–3999, D=4001–4999, E=5001–5999
- **HTML 是唯一数据源**：所有编辑路径最终操作的都是 HTML，不允许引入中间 JSON 结构

## 6. 模块工作明细

| 模块 | 工作明细文档 |
|---|---|
| A 资料接入 | [modules/a-intake/work-breakdown.md](../modules/a-intake/work-breakdown.md) |
| B 解析初稿 | [modules/b-parsing/work-breakdown.md](../modules/b-parsing/work-breakdown.md) |
| C AI 对话 | [modules/c-agent/work-breakdown.md](../modules/c-agent/work-breakdown.md) |
| D 可视化编辑 | [modules/d-workbench/work-breakdown.md](../modules/d-workbench/work-breakdown.md) |
| E 版本导出 | [modules/e-render/work-breakdown.md](../modules/e-render/work-breakdown.md) |
