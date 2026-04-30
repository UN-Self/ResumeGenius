# ResumeGenius v2 架构设计

日期：2026-04-23
状态：已批准

## 1. 背景

v1 架构采用"结构化 JSON → Patch 协议 → 解算引擎 → LaTeX 编译"管线，存在以下问题：

- 中间层过多（6 层数据结构 + Patch 协议 + 解算引擎），开发复杂度高
- LaTeX 作为渲染层带来沉重依赖（TeX Live 1-5GB）、编译延迟（1-3 秒/次）、字体管理复杂
- ResumeDraftState 把简历结构写死，扩展性差
- 编辑体验不够直观（结构化表单 vs 所见即所得）

v2 的核心改变：**HTML 是唯一的数据源**，砍掉所有中间层。

## 2. 核心原则

1. **HTML 是唯一数据源**：不维护中间 JSON 结构，不维护 Patch 协议
2. **所见即所得**：用户像用 Word 一样直接编辑简历 HTML
3. **AI 直接操作 HTML**：AI 对话修改直接返回 HTML，不经过 Patch 协议
4. **导出走服务端**：chromedp 按需启动生成 PDF，用于商业化权限控制
5. **部署友好**：目标 2C2G 服务器，Docker Compose 一键部署

## 3. 技术栈

| 层 | 技术 |
|---|---|
| 营销站（SEO） | Astro |
| 工作台（编辑器） | Vite + React + TypeScript + Tailwind CSS + shadcn/ui |
| 富文本编辑器 | TipTap（基于 ProseMirror） |
| 后端 | Gin + Go |
| ORM | GORM |
| 数据库 | PostgreSQL >= 15 |
| 文件存储 | 本地文件系统（起步） |
| PDF 解析 | ledongthuc/pdf（纯 Go） |
| DOCX 解析 | nguyenthenguyen/docx（纯 Go） |
| PDF 导出 | chromedp（Go 原生库，按需启动 Chromium） |
| AI 模型 | OpenAI-compatible API（Provider Adapter 解耦） |

## 4. 系统架构

```
┌─────────────────────────────────────────────────────────┐
│                      用户浏览器                          │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  TipTap 工具栏 │  │  A4 简历画布  │  │  AI 对话面板   │  │
│  └──────────────┘  └──────────────┘  └───────────────┘  │
│         编辑 + 预览是一体的（所见即所得）                  │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP API
┌──────────────────────▼──────────────────────────────────┐
│  Gin (Go)                                              │
│  ┌─────────┐ ┌─────────┐ ┌──────┐ ┌────────┐ ┌────────┐│
│  │ 项目管理  │ │ 文件解析  │ │ AI   │ │ 草稿   │ │ 导出   ││
│  │ (A)     │ │ (B)     │ │ (C)  │ │ 存版本  │ │ (E)    ││
│  └─────────┘ └─────────┘ └──────┘ └────────┘ └────────┘│
└──────────────────────┬──────────────────────────────────┘
                       │
              ┌────────▼────────┐
              │  PostgreSQL     │
              └─────────────────┘
```

## 5. 数据流

```
用户上传文件 → [A] 存储文件到磁盘
                    ↓
              [B] Go 解析文件，提取文本内容和头像
                    ↓
              [B] 将文本发送给 AI，附上简历 HTML 模板骨架
                    ↓
              AI 返回完整简历 HTML → 存入 drafts 表
                    ↓
              用户打开工作台 → TipTap 加载 HTML → 直接编辑
                    ↓
         ┌──────┬────────────────┐
         │      │                │
    [C] AI对话修改   [D] 手动编辑      [E] 保存快照
    AI 返回 HTML    TipTap 直接改     HTML snapshot
                    DOM
         └──────┴───────┬────────┘
                        ↓
              [E] chromedp 导出 PDF（付费控制）
```

## 6. 各模块职责

### 6.1 模块 intake：项目管理与资料接入

职责：
- 项目 CRUD（创建、列表、详情、删除）
- 文件上传（PDF / DOCX / PNG / JPG），multipart/form-data
- Git 仓库接入（URL + 可选连通性检测）
- 补充文本录入

产出：
- 磁盘上的原始文件
- assets 表记录

API 端点：与 v1 基本一致，详见 docs/modules/intake/contract.md

### 6.2 模块 parsing：文件解析与 AI 初稿生成

职责：
- 用 ledongthuc/pdf 解析 PDF：提取文本块、布局信息、内嵌图片（头像）
- 用 nguyenthenguyen/docx 解析 DOCX：提取段落、表格、样式
- 将提取的纯文本内容发送给 AI
- AI 根据简历 HTML 模板骨架 + 用户资料生成完整简历 HTML
- 将 HTML 存入 drafts 表作为初始草稿

AI 输入：
- 简历 HTML 模板骨架（包含 CSS 样式和语义结构）
- 用户上传文件中提取的文本内容
- 用户补充的文本资料

AI 输出：
- 完整的简历 HTML（可直接在浏览器中渲染）

与 v1 的区别：
- 砍掉 EvidenceSet、ResumeDraftState JSON、ParsedBlock 等中间结构
- AI 直接生成 HTML，不再生成结构化数据
- 初稿生成是同步等待还是异步任务，v1 先用同步（AI 调用通常 5-15 秒）

### 6.3 模块 agent：AI 对话助手

职责：
- 多轮对话（SSE 流式响应）
- AI 能读取当前简历的 HTML 内容作为上下文
- 用户描述需求，AI 返回修改后的完整 HTML
- 用户确认后替换当前草稿

AI 交互模式：
- 用户发送消息 → AI 流式回复（先给文字说明，再给 HTML）
- 用户点击"应用" → 前端用 AI 返回的 HTML 替换编辑器内容
- 用户点击"拒绝" → 继续对话

与 v1 的区别：
- 砍掉 PatchEnvelope、意图识别、PatchOp、SuggestionBuilder 等复杂协议
- AI 直接返回修改后的 HTML，不返回 Patch 操作列表
- 不需要 propose/apply 模式，只有"返回 HTML → 用户确认替换"

### 6.4 模块 workbench：可视化编辑器

技术选型：TipTap（基于 ProseMirror）

功能：
- A4 尺寸编辑画布（CSS 固定 210mm × 297mm）
- 文本编辑：加粗、斜体、下划线、字号、颜色、对齐、行距
- 列表：有序列表、无序列表
- 图片：上传头像、拖拽调整位置和大小
- Section 拖拽排序
- 原生 undo/redo（ProseMirror 内置历史管理）
- 自动保存（debounce 2 秒）

与 v1 的区别：
- 从结构化表单编辑器改为 TipTap 所见即所得编辑器
- 砍掉 Patch 映射逻辑（不再需要把用户操作翻译成 PatchOp）
- 砍掉三栏编辑器布局（左侧 section 导航 + 中间编辑 + 右侧样式面板）
- 编辑和预览是同一个东西，不需要单独的预览机制

### 6.5 模块 render：版本管理与导出

版本管理：
- 每次用户手动保存或 AI 确认修改后，自动创建 HTML 快照
- 版本列表（版本号 + 时间 + 标签）
- 回退：将历史快照写回 `drafts.html_content` 并自动新增快照
- 版本快照只存 HTML，一份简历 HTML 约 5-10KB

PDF 导出（异步任务模式）：
- 前端将当前编辑器 HTML 发送到后端
- 后端创建异步导出任务，立即返回 `task_id`
- 后端校验用户导出权限（付费用户）
- 后端启动 chromedp（Go 原生库控制 Chromium）
- Chromium 以固定 A4 尺寸渲染 HTML
- 调用 page.PrintToPDF() 生成 PDF
- PDF 文件存储至本地文件系统
- 客户端通过 `GET /api/v1/tasks/{task_id}` 轮询任务状态
- 导出完成后从 `result.download_url` 获取 PDF 下载链接
- 释放 Chromium 进程

并发控制：同一时间只允许一个导出任务，其余排队等待。

与 v1 的区别：
- 砍掉 Patch 应用引擎
- 砍掉解算引擎（DraftState → ResolvedResumeSpec）
- 砍掉 LaTeX 模板和 xelatex 编译
- 版本管理从 revision 记录简化为 HTML 快照

## 7. 数据库设计

```sql
-- 项目（先创建，不含外键）
CREATE TABLE projects (
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(200) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',  -- active / archived
    current_draft_id INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 草稿（核心表，HTML 是唯一数据源）
CREATE TABLE drafts (
    id           SERIAL PRIMARY KEY,
    project_id   INTEGER NOT NULL REFERENCES projects(id),
    html_content TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 补充 projects 外键（drafts 表已存在）
ALTER TABLE projects ADD CONSTRAINT fk_projects_current_draft
    FOREIGN KEY (current_draft_id) REFERENCES drafts(id);

-- 资产（文件、Git、文本）
CREATE TABLE assets (
    id          SERIAL PRIMARY KEY,
    project_id  INTEGER NOT NULL REFERENCES projects(id),
    type        VARCHAR(50) NOT NULL,   -- resume_pdf / resume_docx / resume_image / git_repo / note
    uri         TEXT,                    -- 文件路径或 Git URL
    content     TEXT,                    -- 补充文本内容
    label       VARCHAR(100),            -- 补充文本标签
    metadata    JSONB,                   -- {filename, size_bytes, uploaded_at}
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 版本快照
CREATE TABLE versions (
    id           SERIAL PRIMARY KEY,
    draft_id     INTEGER NOT NULL REFERENCES drafts(id),
    html_snapshot TEXT NOT NULL,
    label        VARCHAR(200),           -- "AI 初始生成" / "手动保存" / "AI 修改：精简项目经历"
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI 对话会话
CREATE TABLE ai_sessions (
    id          SERIAL PRIMARY KEY,
    draft_id    INTEGER NOT NULL REFERENCES drafts(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI 对话消息
CREATE TABLE ai_messages (
    id          SERIAL PRIMARY KEY,
    session_id  INTEGER NOT NULL REFERENCES ai_sessions(id),
    role        VARCHAR(20) NOT NULL,   -- user / assistant
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 8. API 端点总览

### 模块 intake（项目管理）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/projects | 项目列表 |
| POST | /api/v1/projects | 创建项目 |
| GET | /api/v1/projects/{project_id} | 项目详情 |
| DELETE | /api/v1/projects/{project_id} | 删除项目 |
| POST | /api/v1/assets/upload | 上传文件 |
| POST | /api/v1/assets/git | 接入 Git 仓库 |
| GET | /api/v1/assets?project_id={project_id} | 资产列表 |
| DELETE | /api/v1/assets/{asset_id} | 删除资产 |
| POST | /api/v1/assets/notes | 添加补充文本 |
| PUT | /api/v1/assets/notes/{note_id} | 编辑补充文本 |

### 模块 parsing（解析与初稿）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/parsing/parse | 触发解析（同步） |
| POST | /api/v1/parsing/generate | 触发 AI 初稿生成（同步） |

### 模块 agent（AI 对话）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/ai/sessions | 创建对话会话 |
| POST | /api/v1/ai/sessions/{session_id}/chat | 发送消息（SSE 流式） |
| GET | /api/v1/ai/sessions/{session_id}/history | 获取对话历史 |

### 模块 workbench（草稿编辑）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/drafts/{draft_id} | 获取草稿 HTML |
| PUT | /api/v1/drafts/{draft_id} | 保存草稿 HTML（自动保存） |

### 模块 render（版本与导出）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/drafts/{draft_id}/versions | 版本列表 |
| POST | /api/v1/drafts/{draft_id}/versions | 手动创建快照 |
| POST | /api/v1/drafts/{draft_id}/rollback | 回退到指定版本（写回 + 自动快照） |
| POST | /api/v1/drafts/{draft_id}/export | 创建 PDF 导出异步任务 |

## 9. 简历 HTML 模板骨架

AI 初稿生成时使用的模板骨架：

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <style>
    @page { size: A4; margin: 0; }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Noto Sans SC', sans-serif; font-size: 10.5pt; line-height: 1.4; color: #333; }
    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
    .profile { display: flex; align-items: center; gap: 16px; margin-bottom: 12pt; }
    .avatar { width: 48pt; height: 48pt; border-radius: 50%; object-fit: cover; }
    .profile h1 { font-size: 18pt; font-weight: 700; }
    .profile p { font-size: 10pt; color: #666; margin-top: 2pt; }
    .section { margin-bottom: 10pt; }
    .section h2 { font-size: 12pt; font-weight: 600; border-bottom: 1pt solid #ddd; padding-bottom: 3pt; margin-bottom: 6pt; }
    .item { margin-bottom: 6pt; }
    .item-header { display: flex; justify-content: space-between; }
    .item h3 { font-size: 10.5pt; font-weight: 600; }
    .item .date { font-size: 9pt; color: #888; }
    .item .subtitle { font-size: 9.5pt; color: #555; }
    .item ul { padding-left: 14pt; }
    .item li { margin-bottom: 2pt; }
    .tags { display: flex; flex-wrap: wrap; gap: 6pt; }
    .tag { background: #f0f0f0; padding: 2pt 8pt; border-radius: 3pt; font-size: 9pt; }
  </style>
</head>
<body>
  <div class="resume">
    <header class="profile">
      <!-- AI 填充：头像、姓名、职位、联系方式 -->
    </header>
    <!-- AI 自由生成 section，可按需增减 -->
  </div>
</body>
</html>
```

AI 不受固定 section 类型的约束，可以自由生成内容结构。CSS 类名（.section, .item, .profile 等）是建议的语义化命名，AI 可以遵循也可以自行扩展。

## 10. 工作台 UI 布局

```
┌─────────────────────────────────────────────────────┐
│  Logo   项目名   |  保存  版本历史  导出PDF(付费)     │
├──────────────────────────────┬──────────────────────┤
│                              │  AI 助手    [收起/展开] │
│                              │ ─────────────────── │
│                              │  用户: 帮我精简项目经历  │
│       A4 简历画布             │                      │
│    （TipTap 编辑器）          │  AI: 好的，我建议...    │
│                              │  （流式输出）           │
│    ┌────────────────────┐    │                      │
│    │  张三              │    │  [应用到简历] [继续对话] │
│    │  前端工程师         │    │                      │
│    │  ──────────        │    │                      │
│    │  工作经历           │    │                      │
│    │  ABC科技           │    │                      │
│    │  · 做了什么事       │    │                      │
│    └────────────────────┘    │                      │
│                              │                      │
│                              │                      │
├──────────────────────────────┴──────────────────────┤
│  TipTap 工具栏：B I U | 字号 | 颜色 | 对齐 | 行距   │
└─────────────────────────────────────────────────────┘
```

左侧：A4 画布（TipTap 编辑器），支持缩放适应屏幕宽度
右侧：AI 对话面板（可收起/展开）
底部：TipTap 格式工具栏（浮动或固定）

## 11. 部署方案

```yaml
# docker-compose.yml
services:
  nginx:
    image: nginx:alpine
    ports: ["80:80"]
    volumes:
      - ./frontend/marketing/dist:/usr/share/nginx/html/marketing
      - ./frontend/workbench/dist:/usr/share/nginx/html/app
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
    depends_on: [gin]

  gin:
    build: ./backend
    ports: ["8080:8080"]
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/resumegenius
      - ZHIPU_API_KEY=${ZHIPU_API_KEY}
      - CHROMIUM_PATH=/usr/bin/chromium
    depends_on: [postgres]

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=resumegenius
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./uploads:/uploads

volumes:
  pgdata:
```

资源预估：
- 空闲时：nginx ~10MB + gin ~50MB + postgres ~500MB = ~560MB
- 导出 PDF 时：+ chromium ~300MB（临时，2-5 秒后释放）
- 2C2G 完全够用

## 12. 与 v1 方案对比

| 维度 | v1 | v2 |
|---|---|---|
| 数据源 | ResumeDraftState JSON + PatchEnvelope | HTML（唯一） |
| 编辑方式 | 结构化表单 + Patch 协议 | TipTap 所见即所得 |
| 预览 | 需要渲染步骤 | 直接看（零延迟） |
| AI 编辑 | 生成 PatchOp → 应用 → 渲染 | 直接返回 HTML |
| 导出 | LaTeX → xelatex → PDF | chromedp → PDF |
| Docker 镜像 | TeX Live 1-5GB | Go 二进制 ~50MB |
| 中间数据结构 | 6 层 | 0 层 |
| 开发复杂度 | 高 | 低 |
| 部署复杂度 | 高（TeX Live + Python + Node） | 低（Go 单二进制） |
| 2C2G 可行性 | 勉强（Python + TeX Live 吃内存） | 宽裕 |

## 13. 砍掉的内容清单

以下 v1 的设计在 v2 中不再需要：

- **数据结构**：ResumeDraftState、ResolvedResumeSpec、EvidenceSet、PatchEnvelope、PatchOp、TargetRef、RevisionRecord
- **协议文档**：patch-schema.md（已删除，v2 不需要 Patch 协议）
- **后端服务**：PatchEngine、ResolveEngine、TemplateManager、EvidenceBuilder、SuggestionBuilder、PatchComposer、IntentRecognizer
- **前端组件**：三栏编辑器（section 导航 + 编辑区 + 样式面板）、建议卡片（SuggestionCard）、确认/拒绝按钮
- **依赖**：LaTeX、TeX Live、ctexart、PyMuPDF（替换为 ledongthuc/pdf）、python-docx（替换为 nguyenthenguyen/docx）
- **运行时**：Python（零 Python 依赖）、Node.js 运行时（Vite 构建后是纯静态文件）
