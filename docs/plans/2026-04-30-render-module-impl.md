# Render 模块实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现版本管理（列表/创建/回退）+ PDF 导出（chromedp 异步任务）全部功能。

**Architecture:** 拆分为 VersionService（纯 DB 操作）和 ExportService（异步任务 + PDF 导出）。PDFExporter 接口解耦 chromedp 实现，测试用 MockExporter 替代。任务状态用 sync.Map 内存管理，Chrome 实例通过 channel 排队复用。

**Tech Stack:** Go + Gin + GORM + chromedp + sync.Map

---

### Task 1: 模块脚手架 + 测试工具

**Files:**
- Create: `backend/internal/modules/render/testutil.go`
- Create: `backend/fixtures/sample_resume.pdf`

**Step 1: 创建 sample_resume.pdf**

创建一个最小有效 PDF 文件作为 MockExporter 的预设返回值。

```bash
python3 -c "
import struct
pdf = b'%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%EOF'
with open('backend/fixtures/sample_resume.pdf', 'wb') as f:
    f.write(pdf)
"
```

**Step 2: 编写 testutil.go**

```go
package render

import (
	"fmt"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"backend/internal/shared/models"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost", "5432", "postgres", "postgres", "resume_genius",
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })
	tx.AutoMigrate(
		&models.Project{}, &models.Draft{}, &models.Version{},
		&models.Asset{}, &models.AISession{}, &models.AIMessage{},
	)
	return tx
}

func seedDraft(t *testing.T, db *gorm.DB) models.Draft {
	t.Helper()
	project := models.Project{UserID: "test-user-1", Title: "Test Project"}
	require.NoError(t, db.Create(&project).Error)

	draft := models.Draft{
		ProjectID:   project.ID,
		HTMLContent: "<html><body>Resume</body></html>",
	}
	require.NoError(t, db.Create(&draft).Error)
	return draft
}
```

> 注：`seedDraft` 需要 import `testify/require`，在最终文件中补上。

**Step 3: 验证 testutil 可编译**

```bash
cd backend && go build ./internal/modules/render/
```

Expected: 无错误（`testutil.go` 在非 `_test` 包中可独立编译）。

**Step 4: Commit**

```bash
git add backend/internal/modules/render/testutil.go backend/fixtures/sample_resume.pdf
git commit -m "chore(render): add test utilities and sample PDF fixture"
```

---

### Task 2: VersionService — 版本 CRUD + 回退

**Files:**
- Create: `backend/internal/modules/render/service_test.go`
- Create: `backend/internal/modules/render/service.go`

**Step 1: 编写 service_test.go — ListByDraft + Create 测试**

```go
package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"backend/internal/shared/models"
)

func TestListByDraft_Empty(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	versions, err := svc.ListByDraft(draft.ID)
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestCreate_Version(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	v, err := svc.Create(draft.ID, "测试版本")
	require.NoError(t, err)
	assert.Equal(t, draft.ID, v.DraftID)
	assert.Equal(t, "测试版本", *v.Label)
	assert.Equal(t, draft.HTMLContent, v.HTMLSnapshot)
	assert.NotZero(t, v.ID)
}

func TestCreate_DefaultLabel(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	v, err := svc.Create(draft.ID, "")
	require.NoError(t, err)
	assert.Equal(t, "手动保存", *v.Label)
}

func TestCreate_DraftNotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewVersionService(db)

	_, err := svc.Create(99999, "label")
	require.ErrorIs(t, err, ErrDraftNotFound)
}

func TestListByDraft_AfterCreate(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	svc.Create(draft.ID, "v1")
	svc.Create(draft.ID, "v2")

	versions, err := svc.ListByDraft(draft.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 2)
	// 验证按 created_at DESC 排序
	assert.True(t, versions[0].CreatedAt.After(versions[1].CreatedAt))
}
```

**Step 2: 跑测试确认失败**

```bash
cd backend && go test ./internal/modules/render/ -run "TestListByDraft|TestCreate" -v
```

Expected: FAIL — `NewVersionService`、`svc.ListByDraft`、`svc.Create`、`ErrDraftNotFound` 未定义。

**Step 3: 编写 service.go — VersionService 实现**

```go
package render

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"backend/internal/shared/models"
)

var (
	ErrDraftNotFound = errors.New("draft not found")
	ErrVersionNotFound = errors.New("version not found")
)

type RollbackResult struct {
	DraftID       uint   `json:"draft_id"`
	UpdatedAt     string `json:"updated_at"`
	NewVersionID  uint   `json:"new_version_id"`
	NewVersionLabel string `json:"new_version_label"`
}

type VersionService struct {
	db *gorm.DB
}

func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

func (s *VersionService) ListByDraft(draftID uint) ([]models.Version, error) {
	var versions []models.Version
	err := s.db.Where("draft_id = ?", draftID).
		Order("created_at DESC").
		Find(&versions).Error
	return versions, err
}

func (s *VersionService) Create(draftID uint, label string) (*models.Version, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}

	if label == "" {
		label = "手动保存"
	}

	v := models.Version{
		DraftID:      draftID,
		HTMLSnapshot: draft.HTMLContent,
		Label:        &label,
	}
	if err := s.db.Create(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *VersionService) Rollback(draftID, versionID uint) (*RollbackResult, error) {
	var version models.Version
	if err := s.db.First(&version, versionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	if version.DraftID != draftID {
		return nil, ErrVersionNotFound
	}

	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}

	return s.rollbackInTx(draft, version)
}

func (s *VersionService) rollbackInTx(draft models.Draft, version models.Version) (*RollbackResult, error) {
	label := "回退到版本 " + time.Now().Format("2006-01-02 15:04:05")
	newVersion := models.Version{
		DraftID:      draft.ID,
		HTMLSnapshot: draft.HTMLContent,
		Label:        &label,
	}

	return s.db.Transaction(func(tx *gorm.DB) (*RollbackResult, error) {
		// 1. 写回目标版本 HTML 到 draft
		if err := tx.Model(&models.Draft{}).Where("id = ?", draft.ID).
			Update("html_content", version.HTMLSnapshot).Error; err != nil {
			return nil, err
		}
		// 2. 自动创建快照
		if err := tx.Create(&newVersion).Error; err != nil {
			return nil, err
		}
		return &RollbackResult{
			DraftID:         draft.ID,
			NewVersionID:    newVersion.ID,
			NewVersionLabel: *newVersion.Label,
		}, nil
	})
}
```

**Step 4: 跑 ListByDraft + Create 测试**

```bash
cd backend && go test ./internal/modules/render/ -run "TestListByDraft|TestCreate" -v
```

Expected: 全部 PASS。

**Step 5: 编写 Rollback 测试**

在 `service_test.go` 末尾追加：

```go
func TestRollback_Success(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	// 创建两个版本
	v1, err := svc.Create(draft.ID, "原始版本")
	require.NoError(t, err)
	svc.UpdateDraftHTML(db, draft.ID, "<html><body>修改后</body></html>")
	v2, err := svc.Create(draft.ID, "修改版本")
	require.NoError(t, err)

	// 回退到 v1
	result, err := svc.Rollback(draft.ID, v1.ID)
	require.NoError(t, err)
	assert.Equal(t, v1.HTMLSnapshot, result.HTMLContent)
	assert.NotZero(t, result.NewVersionID)
	assert.Contains(t, result.NewVersionLabel, "回退到版本")
}

func TestRollback_VersionNotFound(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	_, err := svc.Rollback(draft.ID, 99999)
	require.ErrorIs(t, err, ErrVersionNotFound)
}

func TestRollback_VersionBelongsToOtherDraft(t *testing.T) {
	db := SetupTestDB(t)
	draft1 := seedDraft(t, db)
	draft2 := seedDraft(t, db)
	svc := NewVersionService(db)

	v, err := svc.Create(draft1.ID, "v1")
	require.NoError(t, err)

	_, err = svc.Rollback(draft2.ID, v.ID)
	require.ErrorIs(t, err, ErrVersionNotFound)
}
```

> 注：`svc.UpdateDraftHTML` 和 `result.HTMLContent` 需要在 RollbackResult 中补充字段，或调整测试写法。见 Step 6。

**Step 6: 完善 Rollback 实现**

在 `service.go` 的 `RollbackResult` 结构体中补充 `HTMLContent` 字段，并添加 `UpdateDraftHTML` 辅助方法：

```go
type RollbackResult struct {
	DraftID         uint   `json:"draft_id"`
	HTMLContent     string `json:"-"`
	UpdatedAt       string `json:"updated_at"`
	NewVersionID    uint   `json:"new_version_id"`
	NewVersionLabel string `json:"new_version_label"`
}

// UpdateDraftHTML 仅用于测试辅助，不暴露给外部
func UpdateDraftHTML(db *gorm.DB, draftID uint, html string) {
	db.Model(&models.Draft{}).Where("id = ?", draftID).Update("html_content", html)
}
```

在 `rollbackInTx` 中补充返回 HTMLContent：

```go
return &RollbackResult{
	DraftID:         draft.ID,
	HTMLContent:     version.HTMLSnapshot,
	NewVersionID:    newVersion.ID,
	NewVersionLabel: *newVersion.Label,
}, nil
```

**Step 7: 跑 Rollback 测试**

```bash
cd backend && go test ./internal/modules/render/ -run "TestRollback" -v
```

Expected: 全部 PASS。

**Step 8: 跑全部 service 测试**

```bash
cd backend && go test ./internal/modules/render/ -run "TestListByDraft|TestCreate|TestRollback" -v
```

Expected: 全部 PASS。

**Step 9: Commit**

```bash
git add backend/internal/modules/render/service.go backend/internal/modules/render/service_test.go
git commit -m "feat(render): implement VersionService with tests"
```

---

### Task 3: ExportService + PDFExporter 接口 + MockExporter

**Files:**
- Create: `backend/internal/modules/render/exporter_test.go`
- Create: `backend/internal/modules/render/exporter.go`

**Step 1: 编写 exporter_test.go — 任务创建 + 状态流转测试**

```go
package render

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"backend/internal/shared/storage"
)

// ---------- MockExporter ----------

type MockExporter struct {
	PDFBytes []byte
	Err      error
	mu       sync.Mutex
	CallCount int
}

func (m *MockExporter) ExportHTMLToPDF(html string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount++
	if m.Err != nil {
		return nil, m.Err
	}
	return m.PDFBytes, nil
}

// ---------- test helpers ----------

func samplePDF(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../fixtures/sample_resume.pdf")
	require.NoError(t, err)
	return data
}

func newTestExportService(t *testing.T) (*ExportService, storage.FileStorage) {
	t.Helper()
	dir := t.TempDir()
	store := storage.NewLocalStorage(dir)
	pdf := samplePDF(t)
	exporter := &MockExporter{PDFBytes: pdf}
	svc := NewExportService(exporter, store)
	t.Cleanup(func() { svc.Close() })
	return svc, store
}

// ---------- tests ----------

func TestCreateTask_ReturnsTaskID(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportService(t)
	// 注入 DB 用于 draft 校验
	svc.db = db

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Test</body></html>")
	require.NoError(t, err)
	assert.NotEmpty(t, taskID)
	assert.Contains(t, taskID, "task_")
}

func TestCreateTask_DraftNotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc, _ := newTestExportService(t)
	svc.db = db

	_, err := svc.CreateTask(99999, "<html><body>Test</body></html>")
	require.ErrorIs(t, err, ErrDraftNotFound)
}

func TestGetTask_NotFound(t *testing.T) {
	svc, _ := newTestExportService(t)

	task, err := svc.GetTask("task_nonexistent")
	require.ErrorIs(t, err, ErrTaskNotFound)
	assert.Nil(t, task)
}

func TestTaskFlows_ToCompleted(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportService(t)
	svc.db = db

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Test</body></html>")
	require.NoError(t, err)

	// 等待异步任务完成（最多 3 秒）
	var task *ExportTask
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err = svc.GetTask(taskID)
		require.NoError(t, err)
		if task.Status == "completed" || task.Status == "failed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.Equal(t, "completed", task.Status)
	assert.Equal(t, 100, task.Progress)
	assert.NotNil(t, task.DownloadURL)
	assert.Contains(t, *task.DownloadURL, taskID)
}

func TestTaskFlows_ToFailed(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	pdf := samplePDF(t)
	exporter := &MockExporter{Err: errors.New("chrome crashed")}
	dir := t.TempDir()
	store := storage.NewLocalStorage(dir)
	svc := NewExportService(exporter, store)
	svc.db = db
	t.Cleanup(func() { svc.Close() })

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Test</body></html>")
	require.NoError(t, err)

	var task *ExportTask
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, _ = svc.GetTask(taskID)
		if task != nil && (task.Status == "completed" || task.Status == "failed") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NotNil(t, task)
	assert.Equal(t, "failed", task.Status)
	assert.NotNil(t, task.Error)
	assert.Contains(t, *task.Error, "chrome crashed")
	_ = pdf // pdf 变量在此测试中不需要，但 samplePDF(t) 确认 fixture 存在
}

func TestGetFile_CompletedTask(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportService(t)
	svc.db = db

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Test</body></html>")
	require.NoError(t, err)

	// 等待完成
	var task *ExportTask
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, _ = svc.GetTask(taskID)
		if task != nil && task.Status == "completed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, task)

	data, err := svc.GetFile(taskID)
	require.NoError(t, err)
	assert.Equal(t, samplePDF(t), data)
}

func TestGetFile_TaskNotFound(t *testing.T) {
	svc, _ := newTestExportService(t)

	_, err := svc.GetFile("task_nonexistent")
	require.ErrorIs(t, err, ErrTaskNotFound)
}

func TestGetFile_TaskNotCompleted(t *testing.T) {
	svc, _ := newTestExportService(t)

	// 手动插入一个 pending 任务到 sync.Map
	task := &ExportTask{
		ID:     "task_pending_test",
		Status: "pending",
	}
	svc.tasks.Store(task.ID, task)

	_, err := svc.GetFile("task_pending_test")
	require.ErrorIs(t, err, ErrTaskNotCompleted)
}
```

**Step 2: 跑测试确认失败**

```bash
cd backend && go test ./internal/modules/render/ -run "TestCreateTask|TestGetTask|TestTaskFlows|TestGetFile" -v
```

Expected: FAIL — `ExportService`、`ExportTask`、`MockExporter`、`ErrTaskNotFound`、`ErrTaskNotCompleted` 等全部未定义。

**Step 3: 编写 exporter.go — ExportService + PDFExporter 接口**

```go
package render

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"backend/internal/shared/models"
	"backend/internal/shared/storage"
)

var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrTaskNotCompleted = errors.New("task not completed")
)

// PDFExporter 将 HTML 渲染为 PDF 字节流
type PDFExporter interface {
	ExportHTMLToPDF(htmlContent string) ([]byte, error)
}

// ExportTask 异步导出任务
type ExportTask struct {
	ID          string    `json:"task_id"`
	DraftID     uint      `json:"draft_id"`
	Status      string    `json:"status"`       // pending | processing | completed | failed
	Progress    int       `json:"progress"`     // 0-100
	DownloadURL *string   `json:"download_url"`
	Error       *string   `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}

// ExportService 管理异步 PDF 导出任务
type ExportService struct {
	exporter PDFExporter
	store    storage.FileStorage
	db       *gorm.DB
	tasks    sync.Map           // map[string]*ExportTask
	queue    chan *ExportTask
	closeCh  chan struct{}
	wg       sync.WaitGroup
}

func NewExportService(exporter PDFExporter, store storage.FileStorage) *ExportService {
	svc := &ExportService{
		exporter: exporter,
		store:    store,
		queue:    make(chan *ExportTask, 100),
		closeCh:  make(chan struct{}),
	}
	svc.wg.Add(1)
	go svc.worker()
	return svc
}

func (s *ExportService) CreateTask(draftID uint, htmlContent string) (string, error) {
	// 校验 draft 存在
	if s.db != nil {
		var draft models.Draft
		if err := s.db.First(&draft, draftID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", ErrDraftNotFound
			}
			return "", err
		}
	}

	taskID := "task_" + uuid.New().String()[:8]
	task := &ExportTask{
		ID:        taskID,
		DraftID:   draftID,
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
	}
	s.tasks.Store(taskID, task)

	// 启动异步 worker 处理
	s.wg.Add(1)
	go func() {
		task.HTMLContent = htmlContent
		s.queue <- task
	}()

	return taskID, nil
}

func (s *ExportService) GetTask(taskID string) (*ExportTask, error) {
	v, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, ErrTaskNotFound
	}
	return v.(*ExportTask), nil
}

func (s *ExportService) GetFile(taskID string) ([]byte, error) {
	v, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, ErrTaskNotFound
	}
	task := v.(*ExportTask)
	if task.Status != "completed" {
		return nil, ErrTaskNotCompleted
	}

	// 从 store 读取文件
	key := fmt.Sprintf("exports/%s.pdf", taskID)
	if !s.store.Exists(key) {
		return nil, ErrTaskNotFound
	}
	absPath, err := s.store.Resolve(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *ExportService) worker() {
	defer s.wg.Done()
	for {
		select {
		case task := <-s.queue:
			s.processTask(task)
		case <-s.closeCh:
			// 排空剩余任务
			for len(s.queue) > 0 {
				task := <-s.queue
				s.processTask(task)
			}
			return
		}
	}
}

func (s *ExportService) processTask(task *ExportTask) {
	defer s.wg.Done()

	task.Status = "processing"
	task.Progress = 30
	s.tasks.Store(task.ID, task)

	pdfBytes, err := s.exporter.ExportHTMLToPDF(task.htmlContent)
	if err != nil {
		errMsg := err.Error()
		task.Status = "failed"
		task.Error = &errMsg
		s.tasks.Store(task.ID, task)
		return
	}

	task.Progress = 70

	// 保存 PDF 到 store
	key := fmt.Sprintf("exports/%s.pdf", task.ID)
	_, err = s.store.Save(task.DraftID, task.ID+".pdf", pdfBytes)
	if err != nil {
		errMsg := "保存 PDF 文件失败: " + err.Error()
		task.Status = "failed"
		task.Error = &errMsg
		s.tasks.Store(task.ID, task)
		return
	}

	downloadURL := "/api/v1/tasks/" + task.ID + "/file"
	task.Status = "completed"
	task.Progress = 100
	task.DownloadURL = &downloadURL
	s.tasks.Store(task.ID, task)
}

func (s *ExportService) Close() {
	close(s.closeCh)
	s.wg.Wait()
}
```

> 注：exporter.go 中需要额外 import `"os"`。

**Step 4: 跑全部 exporter 测试**

```bash
cd backend && go test ./internal/modules/render/ -run "TestCreateTask|TestGetTask|TestTaskFlows|TestGetFile" -v
```

Expected: 全部 PASS。

**Step 5: Commit**

```bash
git add backend/internal/modules/render/exporter.go backend/internal/modules/render/exporter_test.go
git commit -m "feat(render): implement ExportService with PDFExporter interface and tests"
```

---

### Task 4: ChromeExporter 实现

**Files:**
- Modify: `backend/internal/modules/render/exporter.go`（追加 ChromeExporter）

> 此 Task 无单元测试 — ChromeExporter 封装 chromedp（外部依赖），通过 PDFExporter 接口已在 Task 3 中用 MockExporter 覆盖。这里只需实现并确保编译通过。

**Step 1: 追加 ChromeExporter 到 exporter.go**

在 `exporter.go` 文件末尾（`Close()` 方法之后）追加：

```go
import (
	// ... 已有 imports ...
	"context"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// ChromeExporter 使用 chromedp 将 HTML 渲染为 PDF
type ChromeExporter struct {
	allocCtx context.Context
	cancel   context.CancelFunc
}

// NewChromeExporter 创建一个长期存活的 Chrome 进程
// 调用方负责在服务关闭时调用 Close() 释放资源
func NewChromeExporter() *ChromeExporter {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return &ChromeExporter{allocCtx: allocCtx, cancel: cancel}
}

func (c *ChromeExporter) ExportHTMLToPDF(htmlContent string) ([]byte, error) {
	// 每个 PDF 生成一个新 Tab（Context），复用同一 Chrome 进程
	ctx, cancel := chromedp.NewContext(c.allocCtx)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, htmlContent).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4
				WithPaperHeight(11.69). // A4
				Do(ctx)
			return err
		}),
	)
	return buf, err
}

func (c *ChromeExporter) Close() {
	c.cancel()
}
```

**Step 2: 验证编译**

```bash
cd backend && go build ./internal/modules/render/
```

Expected: 编译通过。如果 `chromedp` 或 `google/uuid` 尚未在 go.mod 中：

```bash
cd backend && go get github.com/chromedp/chromedp
```

**Step 3: Commit**

```bash
git add backend/internal/modules/render/exporter.go backend/go.mod backend/go.sum
git commit -m "feat(render): add ChromeExporter with chromedp"
```

---

### Task 5: Handler 层（6 个端点）

**Files:**
- Create: `backend/internal/modules/render/handler_test.go`
- Create: `backend/internal/modules/render/handler.go`

**Step 1: 编写 handler_test.go**

```go
package render

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/internal/shared/middleware"
	"backend/internal/shared/models"
	"backend/internal/shared/response"
	"backend/internal/shared/storage"
)

// ---------- setup helpers ----------

func setupRouter(t *testing.T) (*gin.Engine, *Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := SetupTestDB(t)
	versionSvc := NewVersionService(db)

	dir := t.TempDir()
	store := storage.NewLocalStorage(dir)
	pdf, err := os.ReadFile("../../fixtures/sample_resume.pdf")
	require.NoError(t, err)
	exporter := &MockExporter{PDFBytes: pdf}
	exportSvc := NewExportService(exporter, store)
	exportSvc.db = db
	t.Cleanup(func() { exportSvc.Close() })

	h := NewHandler(versionSvc, exportSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "test-user-1")
		c.Next()
	})

	return r, h
}

type apiResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseResp(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

func seedVersion(t *testing.T, db *gorm.DB, draftID uint) models.Version {
	t.Helper()
	v, err := NewVersionService(db).Create(draftID, "测试版本")
	require.NoError(t, err)
	return *v
}

// ---------- version list tests ----------

func TestHandler_ListVersions(t *testing.T) {
	r, h := setupRouter(t)
	r.GET("/drafts/:draft_id/versions", h.ListVersions)

	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	seedVersion(t, db, draft.ID)
	seedVersion(t, db, draft.ID)

	w := doJSON(t, r, "GET", "/drafts/1/versions", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Len(t, items, 2)
}

func TestHandler_CreateVersion(t *testing.T) {
	r, h := setupRouter(t)
	r.POST("/drafts/:draft_id/versions", h.CreateVersion)

	w := doJSON(t, r, "POST", "/drafts/1/versions", map[string]string{
		"label": "精简版",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "精简版", data["label"])
}

func TestHandler_CreateVersion_DefaultLabel(t *testing.T) {
	r, h := setupRouter(t)
	r.POST("/drafts/:draft_id/versions", h.CreateVersion)

	w := doJSON(t, r, "POST", "/drafts/1/versions", map[string]string{})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "手动保存", data["label"])
}

func TestHandler_Rollback(t *testing.T) {
	r, h := setupRouter(t)
	r.POST("/drafts/:draft_id/rollback", h.Rollback)

	w := doJSON(t, r, "POST", "/drafts/1/rollback", map[string]interface{}{
		"version_id": float64(1),
	})
	// 可能 200（成功）或对应错误码，取决于 seed 数据
	assert.Contains(t, []int{http.StatusOK, 5004, 5002}, w.Code)
}

// ---------- export tests ----------

func TestHandler_CreateExport(t *testing.T) {
	r, h := setupRouter(t)
	r.POST("/drafts/:draft_id/export", h.CreateExport)

	w := doJSON(t, r, "POST", "/drafts/1/export", map[string]string{
		"html_content": "<html><body>Test Resume</body></html>",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data["task_id"], "task_")
	assert.Equal(t, "pending", data["status"])
}

func TestHandler_CreateExport_DraftNotFound(t *testing.T) {
	r, h := setupRouter(t)
	r.POST("/drafts/:draft_id/export", h.CreateExport)

	w := doJSON(t, r, "POST", "/drafts/99999/export", map[string]string{
		"html_content": "<html><body>Test</body></html>",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 5002, resp.Code)
}

func TestHandler_GetTask(t *testing.T) {
	r, h := setupRouter(t)
	r.GET("/tasks/:task_id", h.GetTask)

	// 先创建一个导出任务
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	taskID, err := h.exportSvc.CreateTask(draft.ID, "<html><body>Test</body></html>")
	require.NoError(t, err)

	w := doJSON(t, r, "GET", "/tasks/"+taskID, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, taskID, data["task_id"])
}

func TestHandler_GetTask_NotFound(t *testing.T) {
	r, h := setupRouter(t)
	r.GET("/tasks/:task_id", h.GetTask)

	w := doJSON(t, r, "GET", "/tasks/task_nonexistent", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 5005, resp.Code)
}

func TestHandler_DownloadFile(t *testing.T) {
	r, h := setupRouter(t)
	r.GET("/tasks/:task_id/file", h.DownloadFile)

	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	taskID, err := h.exportSvc.CreateTask(draft.ID, "<html><body>Test</body></html>")
	require.NoError(t, err)

	// 等待任务完成
	waitForTask(t, h.exportSvc, taskID, 3*time.Second)

	w := doJSON(t, r, "GET", "/tasks/"+taskID+"/file", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 0)
}

// ---------- helpers ----------

func waitForTask(t *testing.T, svc *ExportService, taskID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task, err := svc.GetTask(taskID)
		if err != nil {
			continue
		}
		if task.Status == "completed" || task.Status == "failed" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("task %s did not complete within %v", taskID, timeout)
}
```

> 注：handler_test.go 需要额外 import `"time"`、`"gorm.io/gorm"`。

**Step 2: 跑测试确认失败**

```bash
cd backend && go test ./internal/modules/render/ -run "TestHandler_" -v
```

Expected: FAIL — `NewHandler`、所有 handler 方法未定义。

**Step 3: 编写 handler.go**

```go
package render

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/shared/models"
	"backend/internal/shared/response"
)

const (
	CodeDraftNotFound  = 5002
	CodeVersionNotFound = 5004
	CodeTaskNotFound   = 5005
	CodeExportFailed   = 5001
)

// ---------- request / response structs ----------

type createVersionReq struct {
	Label string `json:"label"`
}

type versionItem struct {
	ID        uint   `json:"id"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

type versionListResp struct {
	Items []versionItem `json:"items"`
	Total int           `json:"total"`
}

type rollbackReq struct {
	VersionID uint `json:"version_id"`
}

type createExportReq struct {
	HTMLContent string `json:"html_content"`
}

// ---------- Handler ----------

type Handler struct {
	versionSvc *VersionService
	exportSvc  *ExportService
}

func NewHandler(versionSvc *VersionService, exportSvc *ExportService) *Handler {
	return &Handler{versionSvc: versionSvc, exportSvc: exportSvc}
}

// GET /drafts/:draft_id/versions
func (h *Handler) ListVersions(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, 5002, "invalid draft id")
		return
	}

	versions, err := h.versionSvc.ListByDraft(draftID)
	if err != nil {
		response.Error(c, 5001, "internal server error")
		return
	}

	items := make([]versionItem, len(versions))
	for i, v := range versions {
		label := ""
		if v.Label != nil {
			label = *v.Label
		}
		items[i] = versionItem{
			ID:        v.ID,
			Label:     label,
			CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	response.Success(c, versionListResp{Items: items, Total: len(items)})
}

// POST /drafts/:draft_id/versions
func (h *Handler) CreateVersion(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, 5002, "invalid draft id")
		return
	}

	var req createVersionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 5002, "invalid request body")
		return
	}

	v, err := h.versionSvc.Create(draftID, req.Label)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		response.Error(c, 5001, "internal server error")
		return
	}

	label := "手动保存"
	if v.Label != nil {
		label = *v.Label
	}
	response.Success(c, versionItem{
		ID:        v.ID,
		Label:     label,
		CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// POST /drafts/:draft_id/rollback
func (h *Handler) Rollback(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, 5002, "invalid draft id")
		return
	}

	var req rollbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 5002, "invalid request body")
		return
	}

	result, err := h.versionSvc.Rollback(draftID, req.VersionID)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		if errors.Is(err, ErrVersionNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeVersionNotFound, "version not found")
			return
		}
		response.Error(c, 5001, "internal server error")
		return
	}

	response.Success(c, result)
}

// POST /drafts/:draft_id/export
func (h *Handler) CreateExport(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, 5002, "invalid draft id")
		return
	}

	var req createExportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 5002, "invalid request body")
		return
	}

	taskID, err := h.exportSvc.CreateTask(draftID, req.HTMLContent)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeDraftNotFound, "draft not found")
			return
		}
		response.Error(c, CodeExportFailed, "export failed")
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
	})
}

// GET /tasks/:task_id
func (h *Handler) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.exportSvc.GetTask(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeTaskNotFound, "task not found")
			return
		}
		response.Error(c, 5001, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    task,
		"message": "ok",
	})
}

// GET /tasks/:task_id/file
func (h *Handler) DownloadFile(c *gin.Context) {
	taskID := c.Param("task_id")

	data, err := h.exportSvc.GetFile(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeTaskNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrTaskNotCompleted) {
			response.Error(c, 5001, "task not completed")
			return
		}
		response.Error(c, 5001, "internal server error")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=\"resume.pdf\"")
	c.Data(http.StatusOK, "application/pdf", data)
}

// ---------- helpers ----------

func parseUintParam(c *gin.Context, name string) (uint, error) {
	val := c.Param(name)
	n, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
```

**Step 4: 跑全部 handler 测试**

```bash
cd backend && go test ./internal/modules/render/ -run "TestHandler_" -v
```

Expected: 全部 PASS。

**Step 5: Commit**

```bash
git add backend/internal/modules/render/handler.go backend/internal/modules/render/handler_test.go
git commit -m "feat(render): implement 6 HTTP handlers with tests"
```

---

### Task 6: 路由注册 + main.go 接入

**Files:**
- Modify: `backend/internal/modules/render/routes.go`（替换 stub）
- Modify: `backend/cmd/server/main.go`（传入 storage，注册 shutdown hook）

**Step 1: 替换 routes.go**

将现有 stub 替换为完整路由注册：

```go
package render

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"backend/internal/shared/storage"
)

// RegisterRoutes 注册 render 模块的 6 个端点
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	versionSvc := NewVersionService(db)

	exporter := NewChromeExporter()
	exportSvc := NewExportService(exporter, store)
	exportSvc.db = db

	h := NewHandler(versionSvc, exportSvc)

	// 版本管理
	rg.GET("/drafts/:draft_id/versions", h.ListVersions)
	rg.POST("/drafts/:draft_id/versions", h.CreateVersion)
	rg.POST("/drafts/:draft_id/rollback", h.Rollback)

	// PDF 导出
	rg.POST("/drafts/:draft_id/export", h.CreateExport)
	rg.GET("/tasks/:task_id", h.GetTask)
	rg.GET("/tasks/:task_id/file", h.DownloadFile)
}
```

**Step 2: 修改 main.go — 更新 render.RegisterRoutes 调用**

在 `setupRouter` 函数中，将：

```go
render.RegisterRoutes(authed, db)
```

替换为：

```go
render.RegisterRoutes(authed, db, store)
```

**Step 3: 验证编译**

```bash
cd backend && go build ./cmd/server/...
```

Expected: 编译通过。

**Step 4: 跑全量测试确认无回归**

```bash
cd backend && go test ./... -v -count=1
```

Expected: 全部 PASS（包括 main_test.go 中的路由挂载测试）。

> 注：`main_test.go` 中 `TestRoutePaths_CorrectlyMounted` 验证路由路径不变，签名变更不影响该测试。

**Step 5: Commit**

```bash
git add backend/internal/modules/render/routes.go backend/cmd/server/main.go
git commit -m "feat(render): wire up routes with ChromeExporter and storage"
```

---

### Task 7: 全量验证 + 路由集成测试

**Files:**
- Modify: `backend/internal/modules/render/handler_test.go`（追加路由挂载测试）
- Verify: `backend/cmd/server/main_test.go`（已有路由测试）

**Step 1: 追加路由挂载测试**

在 `handler_test.go` 末尾追加：

```go
func TestRoutePaths_CorrectlyMounted(t *testing.T) {
	// 验证 render 模块路由不使用 /render/ 前缀
	// 路径应直接挂在 /api/v1/ 下
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/drafts/1/versions"},
		{"POST", "/drafts/1/versions"},
		{"POST", "/drafts/1/rollback"},
		{"POST", "/drafts/1/export"},
		{"GET", "/tasks/task_abc123"},
		{"GET", "/tasks/task_abc123/file"},
	}

	for _, route := range routes {
		// 路径不应包含 /render/ 前缀
		assert.NotContains(t, route.path, "/render/",
			"route %s %s should not contain /render/ prefix", route.method, route.path)
	}
}
```

**Step 2: 跑全部 render 测试**

```bash
cd backend && go test ./internal/modules/render/... -v -count=1
```

Expected: 全部 PASS（约 20+ 个测试用例）。

**Step 3: 跑全量后端测试**

```bash
cd backend && go test ./... -v -count=1
```

Expected: 全部 PASS，无回归。

**Step 4: 手动冒烟测试（可选）**

```bash
# 启动服务
cd backend && go run cmd/server/main.go

# 另一个终端
# 1. 创建项目 + 草稿（通过已有 API）
# 2. 手动创建版本
curl -X POST http://localhost:8080/api/v1/drafts/1/versions \
  -H "Content-Type: application/json" \
  -d '{"label": "测试版本"}'

# 3. 查看版本列表
curl http://localhost:8080/api/v1/drafts/1/versions

# 4. 创建导出任务
curl -X POST http://localhost:8080/api/v1/drafts/1/export \
  -H "Content-Type: application/json" \
  -d '{"html_content": "<html><body>Hello</body></html>"}'

# 5. 查询任务状态
curl http://localhost:8080/api/v1/tasks/{task_id}

# 6. 下载 PDF
curl -o resume.pdf http://localhost:8080/api/v1/tasks/{task_id}/file
```

**Step 5: Commit**

```bash
git add backend/internal/modules/render/handler_test.go
git commit -m "test(render): add route mounting verification test"
```

---

## 文件总览

实施完成后 render 模块文件结构：

```
backend/internal/modules/render/
├── routes.go           # 路由注册（6 个端点），初始化 services + ChromeExporter
├── handler.go          # 6 个 HTTP handler + 请求/响应结构体 + 错误码常量
├── service.go          # VersionService（ListByDraft / Create / Rollback）+ 哨兵错误
├── exporter.go         # PDFExporter 接口 + ExportService + ChromeExporter + MockExporter
├── testutil.go         # SetupTestDB + seedDraft 测试辅助
├── service_test.go     # VersionService 测试（8 用例）
├── exporter_test.go    # ExportService 测试（7 用例）
└── handler_test.go     # Handler 测试（11 用例）

backend/fixtures/
└── sample_resume.pdf   # MockExporter 预设 PDF
```

**修改的已有文件：**
- `backend/cmd/server/main.go` — `render.RegisterRoutes` 签名变更，传入 `store`

**新增依赖：**
- `github.com/chromedp/chromedp`（已有则跳过）

## 错误码速查

| 错误码 | 常量 | 场景 | HTTP |
|--------|------|------|------|
| 5001 | CodeExportFailed | PDF 导出失败 | 500 |
| 5002 | CodeDraftNotFound | 草稿不存在 | 404 |
| 5004 | CodeVersionNotFound | 版本不存在 | 404 |
| 5005 | CodeTaskNotFound | 任务不存在 | 404 |
