# PDF 导出改造：后端直接渲染 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 后端直接从数据库取 HTML 片段，用渲染模板包裹后交给 Chrome 导出 PDF，实现真正的"所见即所得"。

**Architecture:** 后端新增 `render-template.html` 通过 `//go:embed` 嵌入。`ExportService.CreateTask` 改为只接收 draftID，内部从 DB 查 HTML 并用模板包裹。前端不再传 HTML，只传 draft_id。删除 `export-capture.ts`。

**Tech Stack:** Go (chromedp, gorm, gin) / TypeScript (React, Vitest)

---

### Task 1: 新增渲染模板 + wrapWithTemplate 函数

**Files:**
- Create: `backend/internal/modules/render/render-template.html`
- Modify: `backend/internal/modules/render/exporter.go:1-30` (新增 embed 和 wrapWithTemplate)
- Test: `backend/internal/modules/render/exporter_test.go`

- [ ] **Step 1: 创建渲染模板文件**

`backend/internal/modules/render/render-template.html`:

```html
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
.resume-page {
  width: 210mm;
  min-height: 297mm;
  padding: 18mm 20mm;
  background: #ffffff;
  color: #333333;
  font-family: "Inter", "Noto Sans SC", -apple-system, BlinkMacSystemFont, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  white-space: pre-wrap;
}
.resume-page h1 { font-size: 24px; font-weight: 600; line-height: 1.3; }
.resume-page h2 { font-size: 20px; font-weight: 600; line-height: 1.3; }
.resume-page h3 { font-size: 16px; font-weight: 500; line-height: 1.4; }
.resume-page p  { font-size: 14px; font-weight: 400; line-height: 1.5; }
.resume-page ul, .resume-page ol { padding-left: 24px; }
.resume-page ul { list-style-type: disc; }
.resume-page ol { list-style-type: decimal; }
.resume-page li { margin-bottom: 4px; }
.resume-page li p { display: inline; }
@page { size: A4; margin: 0; }
</style>
</head>
<body style="margin:0;padding:0;">
<div class="resume-page">{{CONTENT}}</div>
</body>
</html>
```

- [ ] **Step 2: 写 wrapWithTemplate 的失败测试**

在 `backend/internal/modules/render/exporter_test.go` 末尾追加：

```go
// ---------------------------------------------------------------------------
// wrapWithTemplate
// ---------------------------------------------------------------------------

func TestWrapWithTemplate_ReplacesPlaceholder(t *testing.T) {
	result := wrapWithTemplate("<h1>Hello</h1><p>World</p>")

	assert.Contains(t, result, "<h1>Hello</h1><p>World</p>")
	assert.Contains(t, result, `<div class="resume-page">`)
	assert.Contains(t, result, ".resume-page h1")
	assert.Contains(t, result, "@page")
	assert.NotContains(t, result, "{{CONTENT}}")
}

func TestWrapWithTemplate_EmptyContent(t *testing.T) {
	result := wrapWithTemplate("")

	assert.Contains(t, result, `<div class="resume-page"></div>`)
	assert.Contains(t, result, ".resume-page")
}
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd backend && go test ./internal/modules/render/ -run TestWrapWithTemplate -v`
Expected: FAIL — `wrapWithTemplate` undefined

- [ ] **Step 4: 在 exporter.go 中添加 embed 和 wrapWithTemplate 函数**

在 `exporter.go` 顶部 import 块已有的 `_ "embed"` 下方，添加模板 embed 声明和函数：

在现有 `//go:embed fonts/inter-regular.woff2` 之前，添加：

```go
//go:embed render-template.html
var renderTemplate string
```

在 `injectFontCSS` 函数之后，添加：

```go
// wrapWithTemplate inserts the HTML fragment into the render template,
// replacing the {{CONTENT}} placeholder.
func wrapWithTemplate(htmlFragment string) string {
	return strings.Replace(renderTemplate, "{{CONTENT}}", htmlFragment, 1)
}
```

- [ ] **Step 5: 运行测试，确认通过**

Run: `cd backend && go test ./internal/modules/render/ -run TestWrapWithTemplate -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/render/render-template.html backend/internal/modules/render/exporter.go backend/internal/modules/render/exporter_test.go
git commit -m "feat(render): add render template and wrapWithTemplate function"
```

---

### Task 2: 改造 ExportService.CreateTask 从 DB 取 HTML

**Files:**
- Modify: `backend/internal/modules/render/exporter.go:76-101` (CreateTask 方法)
- Modify: `backend/internal/modules/render/exporter_test.go` (所有 CreateTask 调用处)

- [ ] **Step 1: 写失败测试 — CreateTask 从 DB 读取 HTML**

在 `exporter_test.go` 的 `TestCreateTask_ReturnsTaskID` 之前添加新测试，同时修改现有测试使其不再传 htmlContent：

```go
func TestCreateTask_ReadsHTMLFromDB(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, mock := newTestExportServiceWithDB(t, db)

	// Verify seedDraft HTML is in the DB
	var fetched models.Draft
	require.NoError(t, db.First(&fetched, draft.ID).Error)
	require.Equal(t, "<html><body><h1>Test Resume</h1></body></html>", fetched.HTMLContent)

	taskID, err := svc.CreateTask(draft.ID)
	require.NoError(t, err)

	task := waitForTask(t, svc, taskID, 3*time.Second)
	assert.Equal(t, "completed", task.Status)

	// Verify the mock exporter received HTML wrapped with the template
	// (the mock ignores input, so we verify via the task's htmlContent)
	stored, _ := svc.tasks.Load(taskID)
	assert.Contains(t, stored.(*ExportTask).htmlContent, "<h1>Test Resume</h1>")
	assert.Contains(t, stored.(*ExportTask).htmlContent, ".resume-page")
}
```

- [ ] **Step 2: 更新现有 CreateTask 测试 — 移除 htmlContent 参数**

在 `exporter_test.go` 中，所有 `svc.CreateTask(draft.ID, "..."`) 调用改为 `svc.CreateTask(draft.ID)`:

- `TestCreateTask_ReturnsTaskID`: `svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")` → `svc.CreateTask(draft.ID)`
- `TestCreateTask_DraftNotFound`: `svc.CreateTask(99999, "<html><body>Resume</body></html>")` → `svc.CreateTask(99999)`
- `TestTaskFlows_ToCompleted`: `svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")` → `svc.CreateTask(draft.ID)`
- `TestTaskFlows_ToFailed`: `svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")` → `svc.CreateTask(draft.ID)`
- `TestGetFile_CompletedTask`: `svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")` → `svc.CreateTask(draft.ID)`
- `TestGetFile_TaskNotCompleted`: `svc.CreateTask(1, "<html><body>Test</body></html>")` → `svc.CreateTask(1)`（注意这个测试 db=nil，需特殊处理）

对于 `TestGetFile_TaskNotCompleted`（db=nil 的场景），需要改写为：

```go
func TestGetFile_TaskNotCompleted(t *testing.T) {
	svc, _ := newTestExportService(t)
	// 不注入 DB，CreateTask 会跳过 draft 验证但 htmlContent 为空
	taskID, err := svc.CreateTask(1)
	require.NoError(t, err)

	_, err = svc.GetFile(taskID)
	require.ErrorIs(t, err, ErrTaskNotCompleted)
}
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd backend && go test ./internal/modules/render/ -run "TestCreateTask|TestTaskFlows|TestGetFile" -v`
Expected: FAIL — `CreateTask` signature mismatch (too many arguments)

- [ ] **Step 4: 改造 CreateTask 方法**

在 `exporter.go` 中，将 `CreateTask` 方法替换为：

```go
// CreateTask validates the draft exists, reads its HTML from the DB,
// wraps it with the render template, and queues it for async processing.
func (s *ExportService) CreateTask(draftID uint) (string, error) {
	if s.db == nil {
		return "", errors.New("database connection required for export")
	}

	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrDraftNotFound
		}
		return "", err
	}

	wrappedHTML := wrapWithTemplate(draft.HTMLContent)

	taskID := "task_" + uuid.New().String()
	task := &ExportTask{
		ID:          taskID,
		DraftID:     draftID,
		Status:      "pending",
		Progress:    0,
		CreatedAt:   time.Now().UTC(),
		htmlContent: wrappedHTML,
	}

	s.tasks.Store(taskID, task)
	s.queue <- task

	return taskID, nil
}
```

关键变化：
- 签名从 `CreateTask(draftID uint, htmlContent string)` → `CreateTask(draftID uint)`
- 不再接受 htmlContent 参数，从 DB 读取 `draft.HTMLContent`
- 调用 `wrapWithTemplate()` 包裹模板
- db 为 nil 时返回 error（不再跳过验证）

- [ ] **Step 5: 运行测试，确认通过**

Run: `cd backend && go test ./internal/modules/render/ -run "TestCreateTask|TestTaskFlows|TestGetFile" -v`
Expected: PASS

- [ ] **Step 6: 运行全部 render 包测试**

Run: `cd backend && go test ./internal/modules/render/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/modules/render/exporter.go backend/internal/modules/render/exporter_test.go
git commit -m "refactor(render): CreateTask reads HTML from DB and wraps with template"
```

---

### Task 3: 改造 Handler — 不再接收 html_content

**Files:**
- Modify: `backend/internal/modules/render/handler.go:58-61, 236-263`
- Modify: `backend/internal/modules/render/handler_test.go:259-349`

- [ ] **Step 1: 更新 handler 测试 — 不传 html_content**

在 `handler_test.go` 中，所有 `TestHandler_CreateExport` 相关的请求 body 改为 `nil`（不传 body）：

`TestHandler_CreateExport`（第 259-275 行）：

```go
func TestHandler_CreateExport(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/export", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["task_id"] != nil && data["task_id"] != "")
	assert.Equal(t, "pending", data["status"])
}
```

`TestHandler_CreateExport_DraftNotFound`（第 277-287 行）：

```go
func TestHandler_CreateExport_DraftNotFound(t *testing.T) {
	r, _, _ := setupRouter(t)

	w := doJSON(t, r, "POST", "/drafts/99999/export", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, CodeDraftNotFound, resp.Code)
}
```

`TestHandler_GetTask`（第 289-312 行）中创建 export 的部分：

```go
exportW := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/export", nil)
```

`TestHandler_DownloadFile`（第 324-349 行）中创建 export 的部分：

```go
w := doJSON(t, r, "POST", "/drafts/"+fmt.Sprintf("%d", draft.ID)+"/export", nil)
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd backend && go test ./internal/modules/render/ -run TestHandler -v`
Expected: FAIL — handler still expects `html_content` or old CreateTask signature

- [ ] **Step 3: 改造 Handler.CreateExport**

在 `handler.go` 中：

1. 删除 `createExportReq` 结构体（第 58-61 行）

2. 替换 `CreateExport` 方法（第 236-263 行）：

```go
// CreateExport handles POST /drafts/:draft_id/export.
func (h *Handler) CreateExport(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, CodeDraftNotFound, "invalid draft_id")
		return
	}

	taskID, err := h.exportSvc.CreateTask(draftID)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		response.Error(c, CodeInternalError, "failed to create export task")
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
	})
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd backend && go test ./internal/modules/render/ -run TestHandler -v`
Expected: PASS

- [ ] **Step 5: 运行全部 render 包测试**

Run: `cd backend && go test ./internal/modules/render/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/render/handler.go backend/internal/modules/render/handler_test.go
git commit -m "refactor(render): handler no longer accepts html_content from client"
```

---

### Task 4: 前端 — 简化 useExport hook

**Files:**
- Modify: `frontend/workbench/src/hooks/useExport.ts`
- Modify: `frontend/workbench/tests/useExport.test.ts`

- [ ] **Step 1: 更新 useExport 测试 — 不传 htmlContent**

`tests/useExport.test.ts` 全部替换为：

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

vi.mock('@/lib/api-client', () => ({
  request: vi.fn(),
}))

import { useExport } from '@/hooks/useExport'
import { request } from '@/lib/api-client'
const mockRequest = vi.mocked(request)

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

describe('useExport', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetch.mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(new Blob(['pdf-data'], { type: 'application/pdf' })),
    })
  })

  it('polls with correct task ID', async () => {
    mockRequest.mockImplementation((url: string) => {
      if (url.includes('/drafts/1/export')) {
        return Promise.resolve({
          task_id: 'abc-123',
          status: 'pending',
          progress: 0,
        })
      }
      if (url.includes('/tasks/abc-123') && !url.includes('/file')) {
        return Promise.resolve({
          task_id: 'abc-123',
          status: 'completed',
          progress: 100,
        })
      }
      throw new Error('Unexpected URL: ' + url)
    })

    const { result } = renderHook(() => useExport({ pollInterval: 100, maxPollDuration: 5000 }))

    await act(async () => {
      await result.current.exportPdf(1, 'resume')
    })

    // Verify POST was called without html_content
    const exportCalls = mockRequest.mock.calls.filter(
      ([url]: [string]) => url.includes('/export')
    )
    expect(exportCalls.length).toBeGreaterThan(0)
    expect(exportCalls[0][1]?.body).toBeUndefined()

    // Verify polling used actual task ID
    const pollingCalls = mockRequest.mock.calls.filter(
      ([url]: [string]) => url.includes('/tasks/') && !url.includes('/export') && !url.includes('/file')
    )
    expect(pollingCalls.length).toBeGreaterThan(0)
    expect(pollingCalls[0][0]).toContain('abc-123')
  })

  it('handles failed status correctly', async () => {
    mockRequest.mockImplementation((url: string) => {
      if (url.includes('/drafts/1/export')) {
        return Promise.resolve({
          task_id: 'task-fail',
          status: 'pending',
          progress: 0,
        })
      }
      if (url.includes('/tasks/task-fail') && !url.includes('/file')) {
        return Promise.resolve({
          task_id: 'task-fail',
          status: 'failed',
          error: 'render error',
        })
      }
      throw new Error('Unexpected URL: ' + url)
    })

    const { result } = renderHook(() => useExport({ pollInterval: 100, maxPollDuration: 5000 }))

    await act(async () => {
      try {
        await result.current.exportPdf(1)
      } catch {
        // expected
      }
    })

    expect(result.current.status).toBe('failed')
    expect(result.current.error).toBe('render error')
  })
})
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/useExport.test.ts`
Expected: FAIL — `exportPdf` still requires 2 positional args

- [ ] **Step 3: 改造 useExport hook**

`src/hooks/useExport.ts` 全部替换为：

```typescript
import { useState, useCallback, useRef, useEffect } from 'react'
import { request } from '@/lib/api-client'

export type ExportStatus = 'idle' | 'exporting' | 'completed' | 'failed'

interface ExportTask {
  task_id: string
  status: string
  progress: number
  download_url?: string
  error?: string
}

interface UseExportOptions {
  pollInterval?: number
  maxPollDuration?: number
}

interface UseExportReturn {
  exportPdf: (draftId: number, filename?: string) => Promise<void>
  status: ExportStatus
  error: string | null
}

export function useExport({
  pollInterval = 800,
  maxPollDuration = 30000,
}: UseExportOptions = {}): UseExportReturn {
  const [status, setStatus] = useState<ExportStatus>('idle')
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef(false)

  const clearState = useCallback(() => {
    abortRef.current = true
    setStatus('idle')
    setError(null)
  }, [])

  const downloadFile = useCallback(async (taskId: string, filename: string) => {
    const res = await fetch(`/api/v1/tasks/${taskId}/file`, { credentials: 'include' })
    if (!res.ok) throw new Error('下载失败')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename.endsWith('.pdf') ? filename : `${filename}.pdf`
    a.click()
    URL.revokeObjectURL(url)
  }, [])

  const pollUntilDone = useCallback(async (
    taskId: string,
    filename: string,
  ): Promise<void> => {
    const deadline = Date.now() + maxPollDuration

    while (!abortRef.current && Date.now() < deadline) {
      await new Promise((r) => setTimeout(r, pollInterval))

      if (abortRef.current) return

      const task = await request<ExportTask>(`/tasks/${taskId}`)
      if (task.status === 'completed') {
        await downloadFile(taskId, filename)
        return
      }
      if (task.status === 'failed') {
        throw new Error(task.error)
      }
    }

    throw new Error('导出超时')
  }, [pollInterval, maxPollDuration, downloadFile])

  useEffect(() => clearState, [clearState])

  const exportPdf = useCallback(async (
    draftId: number,
    filename = 'resume',
  ) => {
    abortRef.current = false
    setStatus('exporting')
    setError(null)

    try {
      const task = await request<ExportTask>(`/drafts/${draftId}/export`, {
        method: 'POST',
      })

      await pollUntilDone(task.task_id, filename)

      if (!abortRef.current) {
        setStatus('completed')
        setTimeout(() => setStatus('idle'), 3000)
      }
    } catch (err) {
      if (abortRef.current) return
      setStatus('failed')
      setError(err instanceof Error ? err.message : '导出失败')
      setTimeout(() => {
        setStatus('idle')
        setError(null)
      }, 5000)
    }
  }, [pollUntilDone])

  return { exportPdf, status, error }
}
```

关键变化：
- `exportPdf` 签名从 `(draftId, htmlContent, filename?)` → `(draftId, filename?)`
- POST body 不再包含 `html_content`
- `downloadFile` 加入 `pollUntilDone` 的 deps 数组（修复 React exhaustive-deps 警告）

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/useExport.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/workbench/src/hooks/useExport.ts frontend/workbench/tests/useExport.test.ts
git commit -m "refactor(frontend): useExport no longer sends HTML to backend"
```

---

### Task 5: 前端 — 清理 EditorPage 和删除 export-capture.ts

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:16, 146-149`
- Delete: `frontend/workbench/src/lib/export-capture.ts`

- [ ] **Step 1: 修改 EditorPage.tsx**

1. 删除第 16 行的 import：
```typescript
// 删除这一行
import { captureExportHTML } from '@/lib/export-capture'
```

2. 替换 `handleExport` 函数（第 146-149 行）：
```typescript
const handleExport = () => {
  if (draftId) {
    exportPdf(Number(draftId))
  }
}
```

- [ ] **Step 2: 删除 export-capture.ts**

```bash
rm frontend/workbench/src/lib/export-capture.ts
```

- [ ] **Step 3: 运行前端测试**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

- [ ] **Step 4: 运行前端构建验证**

Run: `cd frontend/workbench && bun run build`
Expected: 构建成功，无 import 错误

- [ ] **Step 5: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx
git rm frontend/workbench/src/lib/export-capture.ts
git commit -m "refactor(frontend): remove export-capture, EditorPage only sends draft_id"
```

---

### Task 6: 后端编译验证 + 全量测试

**Files:**
- No new files

- [ ] **Step 1: 后端编译检查**

Run: `cd backend && go build ./cmd/server/...`
Expected: 编译成功

- [ ] **Step 2: 后端全量测试**

Run: `cd backend && go test ./... -v`
Expected: PASS

- [ ] **Step 3: 前端全量测试**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

- [ ] **Step 4: 前端构建**

Run: `cd frontend/workbench && bun run build`
Expected: 构建成功
