# 模块 A 工作明细：项目管理与资料接入

更新时间：2026-04-23

本文档列出模块 A 负责人的全部开发任务。契约定义见 [contract.md](./contract.md)。

## 1. 概述

模块 A 是整条管线的起点，负责项目管理、文件上传、Git 仓库接入和补充文本录入。

**核心交付**：用户能创建项目、上传文件、接入 Git 仓库、录入补充文本，并查看和管理所有资料。

## 2. 前端任务

### 2.1 页面

| # | 页面 | 路由建议 | 说明 |
|---|---|---|---|
| 1 | 项目首页 | `/` | 项目列表 + 新建项目入口 |
| 2 | 项目详情页 | `/projects/[id]` | 该项目的资料列表 + 操作入口 |
| 3 | 文件上传 | `/projects/[id]/upload` | 拖拽/点击上传文件 |

### 2.2 组件

| # | 组件 | 说明 |
|---|---|---|
| 1 | `ProjectCard` | 项目卡片，显示标题、创建时间、状态 |
| 2 | `ProjectCreateDialog` | 新建项目弹窗（输入标题即可） |
| 3 | `FileUploader` | 拖拽上传组件，支持 PDF/DOCX/PNG/JPG，显示上传进度 |
| 4 | `AssetList` | 资料列表，区分文件/Git/文本类型，支持删除 |
| 5 | `GitRepoForm` | Git 仓库 URL 输入表单 + 校验 |
| 6 | `NoteEditor` | 补充文本编辑器（textarea + label 标签） |

### 2.3 前端技术要点

- 文件上传用 `multipart/form-data`，显示上传进度条
- 拖拽上传用 HTML5 Drag and Drop API
- 文件大小限制 ≤ 20MB，前端先校验，后端再校验
- 图片文件上传后仅存储，暂不支持 OCR 识别，前端提示"图片仅作参考，不会被 AI 识别"
- Git URL 前端正则校验格式，后端再做连通性校验
- 删除操作需要二次确认弹窗

## 3. 后端任务

### 3.1 API 端点（10 个）

**项目管理（4 个）**：

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 1 | GET | `/api/v1/projects` | 项目列表 |
| 2 | POST | `/api/v1/projects` | 创建项目 |
| 3 | GET | `/api/v1/projects/{project_id}` | 项目详情 |
| 4 | DELETE | `/api/v1/projects/{project_id}` | 删除项目 |

**资产管理（4 个）**：

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 5 | POST | `/api/v1/assets/upload` | 上传文件（multipart） |
| 6 | POST | `/api/v1/assets/git` | 接入 Git 仓库 |
| 7 | GET | `/api/v1/assets?project_id={project_id}` | 资产列表 |
| 8 | DELETE | `/api/v1/assets/{asset_id}` | 删除资产 |

**补充文本（2 个）**：

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 9 | POST | `/api/v1/assets/notes` | 添加补充文本 |
| 10 | PUT | `/api/v1/assets/notes/{note_id}` | 编辑补充文本 |

### 3.2 后端服务

| # | 服务 | 说明 |
|---|---|---|
| 1 | `ProjectService` | 项目 CRUD 业务逻辑 |
| 2 | `AssetService` | 资产统一管理（文件/Git/文本） |
| 3 | `FileStorageService` | 文件存储（本地文件系统，路径 `uploads/{project_id}/{filename}`） |

### 3.3 后端技术要点

- 文件存储用本地文件系统：`uploads/{project_id}/{filename}`
- 文件类型校验：只接受 `.pdf`, `.docx`, `.png`, `.jpg`, `.jpeg`
- 文件大小校验：≤ 20MB
- Git URL 校验：格式校验 + 可选的连通性检测
- 删除项目时级联删除所有关联资料和文件

## 4. 数据库表

| 表名 | 说明 |
|---|---|
| `projects` | 项目（id, title, status, current_draft_id, created_at, updated_at） |
| `assets` | 资产（id, project_id, type, uri, content, label, metadata JSONB, created_at, updated_at） |

- `type` 枚举：resume_pdf / resume_docx / resume_image / git_repo / note
- 文件类资料：`uri` 存文件路径，`metadata` 存文件名、大小等
- Git 类资料：`uri` 存仓库 URL
- 文本类资料：`content` 存文本内容，`label` 存标签

## 5. 测试任务

### 5.1 后端单元测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 项目 CRUD | 创建、查询、删除项目 |
| 2 | 文件上传 | 上传合法文件、拒绝非法格式、拒绝超大文件 |
| 3 | Git 接入 | 合法 URL、非法 URL |
| 4 | 补充文本 | 增删改查 |
| 5 | 级联删除 | 删除项目时关联资料是否清理 |

### 5.2 前端测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 项目创建表单 | 输入标题 → 提交 → 列表刷新 |
| 2 | 文件上传 | 拖拽上传、点击上传、进度条、格式校验提示 |
| 3 | 资料列表 | 展示不同类型资料、删除确认 |

### 5.3 Mock 策略

- 不依赖 B/C/D/E 任何服务
- 本地放 `fixtures/sample_resume.pdf` 等测试文件
- 数据库用 SQLite 内存库即可

## 6. 交付 Checklist

- [ ] 前端：3 个页面 + 6 个组件
- [ ] 后端：10 个 API 端点
- [ ] 后端服务：3 个核心服务
- [ ] 数据库：2 张表（projects, assets）
- [ ] 测试：5 个后端单元测试 + 3 个前端测试
- [ ] API 响应格式符合 api-conventions.md
- [ ] UI 风格符合 ui-design-system.md
- [ ] 错误码使用 1001–1999 范围
