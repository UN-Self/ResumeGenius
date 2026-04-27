# 模块 render — 版本管理与 PDF 导出 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现版本快照（创建/列表）和 chromedp PDF 导出（异步任务模式）。

**Architecture:** VersionService 处理快照 CRUD。ExportService 用 goroutine + channel 管理异步任务，chromedp 渲染 HTML 为 A4 PDF。内存 taskStore 存储任务状态（MVP 简单方案）。

**Tech Stack:** Gin / GORM / chromedp / goroutine+channel

**Depends on:** Phase 0 共享基石完成、模块 workbench 的 DraftService

**契约文档:** `docs/modules/render/contract.md`

---

### Task 1: 后端 — VersionService 快照 CRUD

**Files:**
- Create: `backend/internal/modules/render/service.go`
- Create: `backend/internal/modules/render/handler.go`
- Create: `backend/internal/modules/render/handler_test.go`
- Modify: `backend/internal/modules/render/routes.go`

**Step 1: 写失败测试**

```go
// handler_test.go
package render

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&models.Project{}, &models.Draft{}, &models.Version{})
	return db
}

func TestCreateVersion(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html>content</html>"})

	svc := NewVersionService(db)
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "1"}}

	body, _ := json.Marshal(map[string]string{"label": "初始版本"})
	c.Request = httptest.NewRequest("POST", "/versions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateVersion(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var v models.Version
	db.First(&v, 1)
	if v.HTMLSnapshot != "<html>content</html>" {
		t.Errorf("unexpected snapshot: %s", v.HTMLSnapshot)
	}
}

func TestListVersions(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html>a</html>"})
	db.Create(&models.Version{DraftID: 1, HTMLSnapshot: "<html>a</html>", Label: strPtr("v1")})
	db.Create(&models.Version{DraftID: 1, HTMLSnapshot: "<html>b</html>", Label: strPtr("v2")})

	svc := NewVersionService(db)
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/drafts/1/versions", nil)

	h.ListVersions(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["data"].(map[string]interface{})["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 versions, got %d", len(items))
	}
}

func strPtr(s string) *string { return &s }
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/render/... -v
# Expected: FAIL
```

**Step 3: 实现 service.go + handler.go**

```go
// service.go
package render

import (
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type VersionService struct {
	db *gorm.DB
}

func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

func (s *VersionService) Create(draftID uint, label string) (*models.Version, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		return nil, err
	}
	if label == "" {
		defaultLabel := "手动保存"
		label = defaultLabel
	}
	version := models.Version{
		DraftID:      draftID,
		HTMLSnapshot: draft.HTMLContent,
		Label:        &label,
	}
	if err := s.db.Create(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (s *VersionService) ListByDraftID(draftID uint) ([]models.Version, error) {
	var versions []models.Version
	err := s.db.Where("draft_id = ?", draftID).Order("created_at DESC").Find(&versions).Error
	return versions, err
}
```

```go
// handler.go
package render

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

type Handler struct {
	versionSvc *VersionService
}

func NewHandler(versionSvc *VersionService) *Handler {
	return &Handler{versionSvc: versionSvc}
}

func (h *Handler) CreateVersion(c *gin.Context) {
	draftID, _ := strconv.Atoi(c.Param("draft_id"))

	var req struct {
		Label string `json:"label"`
	}
	c.ShouldBindJSON(&req)

	version, err := h.versionSvc.Create(uint(draftID), req.Label)
	if err != nil {
		response.Error(c, 5002, "草稿不存在")
		return
	}
	response.Success(c, version)
}

func (h *Handler) ListVersions(c *gin.Context) {
	draftID, _ := strconv.Atoi(c.Param("draft_id"))
	versions, err := h.versionSvc.ListByDraftID(uint(draftID))
	if err != nil {
		response.Error(c, 50000, err.Error())
		return
	}
	response.Success(c, gin.H{"items": versions, "total": len(versions)})
}
```

**Step 4: 更新 routes.go**

```go
package render

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	svc := NewVersionService(db)
	h := NewHandler(svc)

	rg.POST("/drafts/:draft_id/versions", h.CreateVersion)
	rg.GET("/drafts/:draft_id/versions", h.ListVersions)
}
```

**Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/render/... -v
# Expected: PASS
```

**Step 6: Commit**

```bash
git add backend/internal/modules/render/
git commit -m "feat(module-e): implement version snapshot CRUD with tests"
```

---

### Task 2: 后端 — PDF 导出（chromedp 异步任务）

**Files:**
- Create: `backend/internal/modules/render/export_service.go`
- Create: `backend/internal/modules/render/export_handler_test.go`
- Modify: `backend/internal/modules/render/handler.go`
- Modify: `backend/internal/modules/render/routes.go`

**Step 1: 安装 chromedp 依赖**

```bash
cd backend && go get github.com/chromedp/chromedp
```

**Step 2: 写失败测试**

```go
// export_handler_test.go
package render

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateExportTask(t *testing.T) {
	svc := NewExportService(":memory:")
	h := NewHandler(NewVersionService(nil))
	h.exportSvc = svc

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "1"}}

	body, _ := json.Marshal(map[string]string{"html_content": "<html><body>test</body></html>"})
	c.Request = httptest.NewRequest("POST", "/export", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateExport(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["status"] != "pending" {
		t.Errorf("expected status pending, got %v", data["status"])
	}
}

func TestGetExportStatus(t *testing.T) {
	svc := NewExportService(":memory:")
	h := NewHandler(NewVersionService(nil))
	h.exportSvc = svc

	// 先创建任务
	taskID := svc.CreateTask("<html>test</html>")

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "task_id", Value: taskID}}
	c.Request = httptest.NewRequest("GET", "/tasks/"+taskID, nil)

	h.GetExportStatus(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code, w.Body.String())
	}
}
```

**Step 3: 实现 ExportService**

```go
// export_service.go
package render

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type TaskStatus struct {
	ID       string `json:"task_id"`
	Status   string `json:"status"`    // pending, processing, completed, failed
	Progress int    `json:"progress"`
	FileURL  string `json:"download_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ExportService struct {
	tasks  sync.Map
	outputDir string
}

func NewExportService(outputDir string) *ExportService {
	os.MkdirAll(outputDir, 0755)
	return &ExportService{outputDir: outputDir}
}

func (s *ExportService) CreateTask(htmlContent string) string {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	s.tasks.Store(taskID, &TaskStatus{ID: taskID, Status: "pending", Progress: 0})

	go s.renderPDF(taskID, htmlContent)

	return taskID
}

func (s *ExportService) GetTask(taskID string) (*TaskStatus, bool) {
	v, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, false
	}
	return v.(*TaskStatus), true
}

func (s *ExportService) renderPDF(taskID, htmlContent string) {
	t, _ := s.tasks.Load(taskID)
	status := t.(*TaskStatus)

	status.Status = "processing"
	status.Progress = 10

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.SetPageContent(htmlContent),
		chromedp.EmulateMedia("screen"),
		chromedp.FullScreenshot(&buf, 100),
	)
	_ = cancel

	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		return
	}

	filePath := fmt.Sprintf("%s/%s.pdf", s.outputDir, taskID)
	if err := os.WriteFile(filePath, buf, 0644); err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		return
	}

	status.Status = "completed"
	status.Progress = 100
	status.FileURL = fmt.Sprintf("/api/v1/render/tasks/%s/file", taskID)
}

func (s *ExportService) ServeFile(taskID string, w io.Writer) error {
	t, ok := s.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found")
	}
	status := t.(*TaskStatus)
	if status.Status != "completed" {
		return fmt.Errorf("task not completed")
	}

	filePath := fmt.Sprintf("%s/%s.pdf", s.outputDir, taskID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
```

**Step 4: 追加 export handlers 到 handler.go**

```go
func (h *Handler) CreateExport(c *gin.Context) {
	var req struct {
		HTMLContent string `json:"html_content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "html_content is required")
		return
	}
	taskID := h.exportSvc.CreateTask(req.HTMLContent)
	response.Success(c, gin.H{"task_id": taskID, "status": "pending"})
}

func (h *Handler) GetExportStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	task, ok := h.exportSvc.GetTask(taskID)
	if !ok {
		response.Error(c, 5005, "任务不存在")
		return
	}
	response.Success(c, task)
}

func (h *Handler) DownloadExport(c *gin.Context) {
	taskID := c.Param("task_id")
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", taskID))
	if err := h.exportSvc.ServeFile(taskID, c.Writer); err != nil {
		response.Error(c, 5001, "PDF 文件不存在")
		return
	}
}
```

**Step 5: routes.go 追加导出路由**

```go
rg.POST("/drafts/:draft_id/export", h.CreateExport)
rg.GET("/tasks/:task_id", h.GetExportStatus)
rg.GET("/tasks/:task_id/file", h.DownloadExport)
```

**Step 6: Commit**

```bash
git add backend/internal/modules/render/
git commit -m "feat(module-e): implement async PDF export with chromedp"
```

---

### Task 3: 前端 — 版本列表 + 导出按钮

**Files:**
- Create: `frontend/workbench/src/components/version/VersionList.tsx`
- Create: `frontend/workbench/src/components/version/VersionList.test.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`（顶部操作栏加导出按钮）

**Step 1: 写失败测试**

```tsx
// VersionList.test.tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { VersionList } from './VersionList'

vi.mock('../../lib/api-client', () => ({
  apiClient: {
    get: vi.fn().mockResolvedValue({
      items: [
        { id: 1, label: 'v1', created_at: '2026-04-26T00:00:00Z' },
        { id: 2, label: 'v2', created_at: '2026-04-26T01:00:00Z' },
      ],
      total: 2,
    }),
    post: vi.fn(),
  },
}))

describe('VersionList', () => {
  it('renders version labels', async () => {
    render(<VersionList draftId={1} />)
    expect(await screen.findByText('v1')).toBeInTheDocument()
    expect(await screen.findByText('v2')).toBeInTheDocument()
  })
})
```

**Step 2: 实现 VersionList**

```tsx
import { useState, useEffect } from 'react'
import { apiClient } from '../../lib/api-client'

interface Version {
  id: number
  label: string
  created_at: string
}

interface Props {
  draftId: number
}

export function VersionList({ draftId }: Props) {
  const [versions, setVersions] = useState<Version[]>([])
  const [exporting, setExporting] = useState(false)

  useEffect(() => {
    apiClient.get<{ items: Version[] }>(`/render/drafts/${draftId}/versions`)
      .then(data => setVersions(data.items))
  }, [draftId])

  const handleExport = async () => {
    setExporting(true)
    try {
      const task = await apiClient.post<{ task_id: string; status: string }>(
        `/render/drafts/${draftId}/export`,
        {} // 后端从 draft 读取当前 HTML
      )
      // 轮询任务状态
      const poll = setInterval(async () => {
        const status = await apiClient.get<{ status: string; download_url: string }>(
          `/render/tasks/${task.task_id}`
        )
        if (status.status === 'completed') {
          clearInterval(poll)
          window.open(status.download_url)
          setExporting(false)
        }
      }, 1000)
    } catch {
      setExporting(false)
    }
  }

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-medium text-sm">版本历史</h3>
        <button
          onClick={handleExport}
          disabled={exporting}
          className="text-sm bg-blue-600 text-white px-3 py-1 rounded disabled:opacity-50"
        >
          {exporting ? '导出中...' : '导出 PDF'}
        </button>
      </div>
      <div className="space-y-2 max-h-64 overflow-y-auto">
        {versions.map(v => (
          <div key={v.id} className="text-sm text-gray-600 py-1 border-b last:border-0">
            <span className="font-medium">{v.label}</span>
            <span className="ml-2 text-gray-400">
              {new Date(v.created_at).toLocaleString()}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
```

**Step 3: Commit**

```bash
git add frontend/workbench/src/
git commit -m "feat(module-e): add version list and PDF export button"
```

---

## 验证清单

- [ ] `go test ./internal/modules/render/... -v` 全部通过
- [ ] `curl -X POST localhost:8080/api/v1/render/drafts/1/versions -d '{"label":"snap"}'` 创建版本
- [ ] `curl localhost:8080/api/v1/render/drafts/1/versions` 返回版本列表
- [ ] `curl -X POST localhost:8080/api/v1/render/drafts/1/export -d '{"html_content":"<html>test</html>"}'` 返回 task_id
- [ ] `curl localhost:8080/api/v1/render/tasks/<task_id>` 返回任务状态（最终 completed）
- [ ] `curl localhost:8080/api/v1/render/tasks/<task_id>/file` 下载 PDF
- [ ] 前端版本列表显示历史版本
- [ ] 点击"导出 PDF"按钮能下载 PDF 文件
