# 模块 A 契约：项目管理与资料接入

更新时间：2026-04-23

## 1. 角色定义

**负责**：

- 项目创建与管理（CRUD）
- 文件上传（PDF / DOCX / PNG / JPG），multipart/form-data
- Git 仓库 URL 接入（可选连通性检测）
- 补充文本录入与编辑

**不负责**：

- 文件内容解析（B 的事）
- AI 初稿生成（B 的事）
- 简历编辑（C / D 的事）
- 版本管理和 PDF 导出（E 的事）

## 2. 输入契约

本模块是管线起点，输入来自用户操作：

| 输入 | 来源 | 格式 |
|---|---|---|
| 创建项目 | 用户操作 | `{ title: string }` |
| 上传文件 | 用户操作 | multipart/form-data |
| Git 仓库 | 用户操作 | `{ project_id: int, repo_url: string }` |
| 补充文本 | 用户操作 | `{ project_id: int, content: string, label?: string }` |

## 3. 输出契约

产出写入数据库：

| 表 | 说明 |
|---|---|
| `projects` | 项目记录 |
| `assets` | 资产记录（文件元信息 / Git URL / 补充文本） |

文件存储到本地磁盘：`uploads/{project_id}/{filename}`

## 4. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

### 4.1 项目管理

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/projects` | 项目列表 |
| POST | `/api/v1/projects` | 创建项目 |
| GET | `/api/v1/projects/{project_id}` | 项目详情 |
| DELETE | `/api/v1/projects/{project_id}` | 删除项目 |

### 4.2 资产管理

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/assets/upload` | 上传文件（multipart） |
| POST | `/api/v1/assets/git` | 接入 Git 仓库 |
| GET | `/api/v1/assets?project_id={project_id}` | 资产列表 |
| DELETE | `/api/v1/assets/{asset_id}` | 删除资产 |

### 4.3 补充文本

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/assets/notes` | 添加补充文本 |
| PUT | `/api/v1/assets/notes/{note_id}` | 编辑补充文本 |

### 关键端点详情

#### POST /api/v1/projects

```
Request:
{
  "title": "前端工程师求职简历"
}

Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "title": "前端工程师求职简历",
    "status": "active",
    "current_draft_id": null,
    "created_at": "2026-04-23T20:00:00Z"
  }
}
```

#### POST /api/v1/assets/upload

```
Request: multipart/form-data
  - file: (binary)
  - project_id: 1
  - type: "resume_pdf" | "resume_docx" | "resume_image"

Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "project_id": 1,
    "type": "resume_pdf",
    "uri": "uploads/1/resume.pdf",
    "metadata": {
      "filename": "resume.pdf",
      "size_bytes": 102400,
      "uploaded_at": "2026-04-23T20:00:00Z"
    },
    "created_at": "2026-04-23T20:00:00Z"
  }
}
```

> v1 限制：`resume_image` 类型的资产仅存储，B 模块解析时跳过。图片可用于前端手动引用（如头像），暂不支持 OCR 识别。

#### POST /api/v1/assets/notes

```
Request:
{
  "project_id": 1,
  "content": "目标岗位是全栈工程师，偏重后端",
  "label": "求职意向"
}

Response:
{
  "code": 0,
  "data": {
    "id": 2,
    "project_id": 1,
    "type": "note",
    "content": "目标岗位是全栈工程师，偏重后端",
    "label": "求职意向",
    "created_at": "2026-04-23T20:10:00Z"
  }
}
```

## 5. 依赖与边界

### 上游

- 无（用户直接交互）

### 下游

- 模块 B（解析与初稿生成）消费 assets 中的文件路径和文本

### 可 mock 的边界

- B 不需要启动 A 的服务，直接读 fixtures/ 中的测试文件
- A 不需要知道 B 如何消费

## 6. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 1001 | 400 | 文件格式不支持 |
| 1002 | 400 | 文件大小超限（≤ 20MB） |
| 1003 | 400 | Git 仓库 URL 无效 |
| 1004 | 404 | 项目不存在 |
| 1005 | 409 | 资料已存在（重复上传同文件） |
| 1006 | 404 | 资料不存在 |

## 7. 测试策略

### 独立测试

- 用本地测试文件（`fixtures/sample_resume.pdf` 等）测试上传和存储
- 不需要启动模块 B 的服务
- 不需要数据库（可用 SQLite 内存库替代 PostgreSQL）

### Mock 产出

确保文件上传和资产创建功能正确，模块 B 可直接使用 assets 表中的记录。

### 前端测试

- 项目创建表单
- 文件上传组件（拖拽 + 点击）
- Git 仓库输入表单
- 资料列表展示
