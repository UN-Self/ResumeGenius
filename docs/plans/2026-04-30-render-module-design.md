# Render 模块设计文档 — 2026-04-30

## 概述

实现 render 模块全部功能：版本管理（列表/创建/回退）+ PDF 导出（chromedp 异步任务）。

选择方案 B1：内存任务管理 + 接口解耦。异步任务状态用 `sync.Map` 管理，chromedp 通过 `PDFExporter` 接口解耦，测试用 `MockExporter` 替代。

## 文件结构

```
backend/internal/modules/render/
├── routes.go              # 路由注册（替换现有 stub）
├── handler.go             # 6 个 HTTP handler
├── service.go             # VersionService（版本 CRUD + 回退）
├── exporter.go            # ExportService + PDFExporter 接口 + chromedp 实现 + 任务管理
├── exporter_test.go       # ExportService 单元测试（mock PDFExporter）
├── service_test.go        # VersionService 单元测试
├── handler_test.go        # Handler 集成测试
└── testutil.go            # 测试辅助函数

fixtures/
└── sample_resume.pdf      # mock 导出用的预设 PDF 文件
```

## API 端点（6 个）

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 1 | GET | `/api/v1/drafts/:draft_id/versions` | 版本列表 |
| 2 | POST | `/api/v1/drafts/:draft_id/versions` | 手动创建快照 |
| 3 | POST | `/api/v1/drafts/:draft_id/rollback` | 回退到指定版本（写回+自动快照） |
| 4 | POST | `/api/v1/drafts/:draft_id/export` | 创建 PDF 导出异步任务 |
| 5 | GET | `/api/v1/tasks/:task_id` | 查询任务状态 |
| 6 | GET | `/api/v1/tasks/:task_id/file` | 下载 PDF 文件 |

## 端点详情

### GET /drafts/:draft_id/versions

```
Response:
{
  "code": 0,
  "data": {
    "items": [
      { "id": 1, "label": "AI 初始生成", "created_at": "2026-04-23T20:00:00Z" }
    ],
    "total": 1
  }
}
```

按 `created_at DESC` 排序，不返回 `html_snapshot`。

### POST /drafts/:draft_id/versions

```
Request:  { "label": "精简版" }
Response: { "code": 0, "data": { "id": 4, "label": "精简版", "created_at": "..." } }
```

读取当前 draft 的 `html_content`，存入 versions 表。label 可选，默认 `"手动保存"`。

### POST /drafts/:draft_id/rollback

```
Request:  { "version_id": 1 }
Response: { "code": 0, "data": { "draft_id": 1, "updated_at": "...", "new_version_id": 5, "new_version_label": "回退到版本 1" } }
```

事务内完成：①将目标版本的 `html_snapshot` 写回 `drafts.html_content` ②自动创建新版本快照。

### POST /drafts/:draft_id/export

```
Request:  { "html_content": "<!DOCTYPE html>..." }
Response: { "code": 0, "data": { "task_id": "task_a1b2c3d4", "status": "pending" } }
```

校验 draft 存在 → 创建任务 → 启动 goroutine 执行导出 → 立即返回 task_id。

### GET /tasks/:task_id

```
Processing: { "code": 0, "data": { "task_id": "...", "status": "processing", "progress": 60 } }
Completed:  { "code": 0, "data": { "task_id": "...", "status": "completed", "progress": 100, "result": { "download_url": "/api/v1/tasks/task_xxx/file" } } }
Failed:     { "code": 5001, "data": { "task_id": "...", "status": "failed", "error": "PDF 渲染失败" } }
```

### GET /tasks/:task_id/file

```
Response: Content-Type: application/pdf, 二进制流
```

从 `exports/{task_id}.pdf` 读取文件返回。

## 核心架构

### 服务拆分

**VersionService** — 纯数据库操作：
- `ListByDraft(draftID uint) ([]models.Version, error)`
- `Create(draftID uint, label string) (*models.Version, error)`
- `Rollback(draftID, versionID uint) (*RollbackResult, error)`

**ExportService** — 异步任务 + PDF 导出：
- `CreateTask(draftID uint, htmlContent string) (taskID string, error)`
- `GetTask(taskID string) (*ExportTask, error)`
- `GetFile(taskID string) ([]byte, error)`

### PDFExporter 接口

```go
type PDFExporter interface {
    ExportHTMLToPDF(htmlContent string) ([]byte, error)
}

type ChromeExporter struct { /* chromedp 实现，长期存活 */ }
type MockExporter struct { /* 测试用 */ }
```

`ChromeExporter` 在 `routes.go` 初始化时注入 `ExportService`，服务关闭时调用 `Close()` 优雅终止。

### Chrome 实例复用 + Channel 排队

`ChromeExporter` 内部维护一个长期存活的 Chrome 进程和任务 channel：

```
ChromeExporter（服务启动时初始化，长期存活）
    │
    ├── ExecAllocator（Chrome 进程，启动一次，跨任务复用）
    │
    └── Worker goroutine（持续从 channel 读取任务）
         │
         ├─ 收到任务 → 创建新 Tab（chromedp.NewContext）
         │                → 加载 HTML + PrintToPDF
         │                → 关闭 Tab
         ├─ 下一个任务 → 复用同一个 Chrome 进程，开新 Tab
         └─ 收到 shutdown signal → 关闭 Chrome 进程
```

- `chromedp.ExecAllocator`（Chrome 进程）启动一次，多个导出任务复用
- 每个导出任务创建新 Tab（`chromedp.NewContext`），开销极小
- 任务通过 Go channel 排队，天然串行化
- `ChromeExporter.Close()` 用于优雅关闭，在 `main.go` 中注册 shutdown hook
- 不需要 `sync.Mutex`，channel 本身保证串行执行

### 异步任务流转

```
POST /export
    ├─ draft 不存在 → 5002
    ├─ 创建任务（status: "pending"）→ 立即返回 task_id
    └─ 发送任务到 ChromeExporter 的 channel
         └─ Worker goroutine:
             ├─ status → "processing"
             ├─ ChromeExporter.ExportHTMLToPDF(html)
             ├─ FileStorage.Save("exports/{task_id}.pdf", pdfBytes)
             ├─ status → "completed"，记录 download_url
             └─ 失败 → status → "failed"
```

多个导出请求通过 channel 排队，Worker 按顺序处理。

### 任务数据结构

```go
type ExportTask struct {
    ID          string
    DraftID     uint
    Status      string    // "pending" | "processing" | "completed" | "failed"
    Progress    int       // 0-100
    DownloadURL *string
    Error       *string
    CreatedAt   time.Time
}
```

任务 ID 格式：`task_` + UUID，如 `task_a1b2c3d4`。用 `sync.Map` 存储，进程重启丢失（MVP 可接受）。

### PDF 文件存储

路径：`exports/{task_id}.pdf`，使用 `shared/storage.FileStorage` 接口，与 intake 模块一致。

## 错误码

| 错误码 | 场景 | HTTP |
|---|---|---|
| 5001 | PDF 导出失败 | 500 |
| 5002 | 草稿不存在 | 404 |
| 5003 | 导出任务排队中 | 409（预留，当前不使用） |
| 5004 | 版本不存在 | 404 |
| 5005 | 任务不存在 | 404 |

## 前端设计

### 组件

| 组件 | 位置 | 说明 |
|---|---|---|
| `ExportButton` | `components/editor/ActionBar.tsx` 中 | 导出触发按钮，管理导出状态机 |
| `ExportStatus` | `components/editor/` | 状态提示 + 下载按钮 |

### 导出交互流程

```
用户点击"导出 PDF"
    │
    ├─ 前端发送 POST /drafts/:draft_id/export（当前编辑器 HTML）
    ├─ 按钮变为 loading + 提示"正在导出..."
    ├─ 前端开始轮询 GET /tasks/:task_id（间隔 2s）
    │
    ├─ pending → 提示"排队中..."
    ├─ processing → 提示"正在导出..."
    ├─ completed → 提示"导出完成！"
    │              显示"下载 PDF"按钮 → 用户点击后 GET /tasks/:task_id/file 下载
    └─ failed → 提示"导出失败：{error}"，显示"重试"按钮
```

导出和下载解耦：导出完成后只提醒用户，用户自行点击下载。

### 前端 API

在 `lib/api-client.ts` 新增 `renderApi`：

```typescript
export const renderApi = {
  createExport: (draftId: number, htmlContent: string) =>
    apiClient.post(`/drafts/${draftId}/export`, { html_content: htmlContent }),
  getTaskStatus: (taskId: string) =>
    apiClient.get(`/tasks/${taskId}`),
  downloadFile: (taskId: string) =>
    apiClient.get(`/tasks/${taskId}/file`, { responseType: 'blob' }),
}
```

## 测试策略

| 测试文件 | 覆盖 | mock |
|---|---|---|
| `service_test.go` | 版本创建、列表、回退、草稿不存在、版本不存在 | test DB |
| `exporter_test.go` | 任务创建、状态流转、并发排队、导出失败 | MockExporter |
| `handler_test.go` | 6 个端点请求/响应格式 | 复用 service + exporter mock |

MockExporter 实现：返回 `fixtures/sample_resume.pdf`。测试不需要 Chromium 环境。
