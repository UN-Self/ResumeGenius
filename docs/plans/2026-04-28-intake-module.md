# Intake 模块实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Intake 模块全部功能 — 后端 10 个 API 端点（含匿名用户隔离）、前端 2 个页面 + 5 个组件、完整测试。

**Architecture:** 后端采用 Handler → Service → Storage 三层，FileStorage 接口化。用户隔离通过 `X-User-ID` header + 中间件实现。前端使用 shadcn/ui + Warm Editorial 主题色。

**Tech Stack:** Go/Gin/GORM (backend), React/Vite/Tailwind/shadcn (frontend), PostgreSQL (database), uuid (file dedup)

**Spec:** `docs/superpowers/specs/2026-04-28-intake-module-design.md`

---

## 文件结构总览

### 后端 — 新建文件

```
backend/internal/shared/middleware/user.go          # 用户识别中间件
backend/internal/modules/intake/storage.go          # FileStorage 接口 + 本地实现
backend/internal/modules/intake/service.go          # ProjectService + AssetService
backend/internal/modules/intake/handler.go          # 10 个 HTTP handler
backend/internal/modules/intake/routes.go           # 路由注册（重写）
backend/internal/modules/intake/testutil.go          # 测试辅助（PG 连接、事务隔离）
backend/internal/modules/intake/service_test.go     # Service 层测试
backend/internal/modules/intake/handler_test.go     # Handler 层测试
backend/internal/modules/intake/storage_test.go     # Storage 层测试
```

### 后端 — 修改文件

```
backend/internal/shared/models/models.go            # Project 新增 UserID 字段
backend/internal/shared/middleware/middleware.go     # CORS 允许 X-User-ID header
backend/cmd/server/main.go                          # 注册 UserIdentify 中间件
backend/go.mod                                      # 添加 google/uuid 依赖
```

### 前端 — 新建文件

```
frontend/workbench/src/pages/ProjectList.tsx
frontend/workbench/src/pages/ProjectList.test.tsx
frontend/workbench/src/pages/ProjectDetail.tsx
frontend/workbench/src/pages/ProjectDetail.test.tsx
frontend/workbench/src/components/intake/ProjectCard.tsx
frontend/workbench/src/components/intake/AssetList.tsx
frontend/workbench/src/components/intake/UploadDialog.tsx
frontend/workbench/src/components/intake/UploadDialog.test.tsx
frontend/workbench/src/components/intake/GitRepoDialog.tsx
frontend/workbench/src/components/intake/NoteDialog.tsx
frontend/workbench/src/components/intake/DeleteConfirm.tsx
```

### 前端 — 修改文件

```
frontend/workbench/src/App.tsx                      # 更新路由
frontend/workbench/src/lib/api-client.ts            # 添加 X-User-ID header + multipart 支持
frontend/workbench/src/index.css                    # 添加 CSS 变量 + Warm Editorial 主题
frontend/workbench/package.json                     # 添加 testing-library 依赖
```

---

### Task 1: 更新 Project Model + 用户识别中间件

**Files:**
- Modify: `backend/internal/shared/models/models.go:31-41`
- Create: `backend/internal/shared/middleware/user.go`
- Modify: `backend/internal/shared/middleware/middleware.go:10-21`
- Modify: `backend/cmd/server/main.go:20-32`

- [ ] **Step 1: 在 Project model 中添加 UserID 字段**

修改 `backend/internal/shared/models/models.go`，在 Project struct 中添加 UserID：

```go
type Project struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         string    `gorm:"size:36;not null;index" json:"user_id"`
	Title          string    `gorm:"size:200;not null" json:"title"`
	Status         string    `gorm:"size:20;not null;default:'active'" json:"status"`
	CurrentDraftID *uint     `json:"current_draft_id"`
	CurrentDraft   *Draft    `gorm:"foreignKey:CurrentDraftID" json:"current_draft,omitempty"`
	Assets         []Asset   `gorm:"foreignKey:ProjectID" json:"assets,omitempty"`
	Drafts         []Draft   `gorm:"foreignKey:ProjectID" json:"drafts,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: 创建用户识别中间件**

创建 `backend/internal/shared/middleware/user.go`：

```go
package middleware

import "github.com/gin-gonic/gin"

const ContextUserID = "user_id"

// UserIdentify 从 X-User-ID header 提取匿名用户 ID，注入 context
func UserIdentify() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}
```

- [ ] **Step 3: 更新 CORS 中间件，允许 X-User-ID header**

修改 `backend/internal/shared/middleware/middleware.go`，在 CORS 的 `Access-Control-Allow-Headers` 中添加 `X-User-ID`：

```go
c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
```

- [ ] **Step 4: 在 main.go 中注册中间件**

修改 `backend/cmd/server/main.go`，在 `setupRouter` 中添加 `middleware.UserIdentify()`：

```go
func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.UserIdentify(), middleware.Logger())

	v1 := r.Group("/api/v1")
	// ... 其余不变
}
```

- [ ] **Step 5: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过，无错误

- [ ] **Step 6: Commit**

```bash
git add backend/internal/shared/models/models.go backend/internal/shared/middleware/user.go backend/internal/shared/middleware/middleware.go backend/cmd/server/main.go
git commit -m "feat: add UserID to Project model and user identification middleware"
```

---

### Task 2: FileStorage 接口 + 本地实现 + 测试

**Files:**
- Create: `backend/internal/modules/intake/storage.go`
- Create: `backend/internal/modules/intake/storage_test.go`

- [ ] **Step 1: 安装 uuid 依赖**

Run: `cd backend && go get github.com/google/uuid`

- [ ] **Step 2: 写失败测试**

创建 `backend/internal/modules/intake/storage_test.go`：

```go
package intake

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorage_Save(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("hello world")
	path, err := s.Save(1, "resume.pdf", data)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 验证文件存在且内容正确
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(got))
	}

	// 验证路径格式: dir/1/uuid_resume.pdf
	parentDir := filepath.Dir(path)
	if filepath.Base(parentDir) != "1" {
		t.Errorf("expected project dir '1', got '%s'", filepath.Base(parentDir))
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	path, err := s.Save(1, "resume.pdf", []byte("content"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = s.Delete(path)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestLocalStorage_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	err := s.Delete("/nonexistent/file.pdf")
	if err != nil {
		t.Fatalf("Delete nonexistent file should not error, got: %v", err)
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	path, err := s.Save(1, "resume.pdf", []byte("content"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !s.Exists(path) {
		t.Error("file should exist")
	}
	if s.Exists("/nonexistent/file.pdf") {
		t.Error("nonexistent file should not exist")
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/intake/... -run TestLocalStorage -v`
Expected: FAIL（`NewLocalStorage` 未定义）

- [ ] **Step 4: 实现 storage.go**

创建 `backend/internal/modules/intake/storage.go`：

```go
package intake

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type FileStorage interface {
	Save(projectID uint, filename string, data []byte) (string, error)
	Delete(path string) error
	Exists(path string) bool
}

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

func (s *LocalStorage) Save(projectID uint, filename string, data []byte) (string, error) {
	projectDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", projectID))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("create project dir: %w", err)
	}

	uniqueName := fmt.Sprintf("%s_%s", uuid.New().String(), filename)
	fullPath := filepath.Join(projectDir, uniqueName)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fullPath, nil
}

func (s *LocalStorage) Delete(path string) error {
	if !s.Exists(path) {
		return nil
	}
	return os.Remove(path)
}

func (s *LocalStorage) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/intake/... -run TestLocalStorage -v`
Expected: PASS（4 个测试全部通过）

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/intake/storage.go backend/internal/modules/intake/storage_test.go backend/go.mod backend/go.sum
git commit -m "feat(intake): add FileStorage interface with local implementation and tests"
```

---

### Task 3: 测试辅助工具（PG 事务隔离）

**Files:**
- Create: `backend/internal/modules/intake/testutil.go`

- [ ] **Step 1: 创建测试辅助**

创建 `backend/internal/modules/intake/testutil.go`：

```go
package intake

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// SetupTestDB 连接 PostgreSQL 测试数据库，返回事务级 DB 连接
// 测试结束后自动回滚，保证测试间隔离
// 需要环境变量 DB_HOST, DB_PORT, DB_USER, DB_PASSWORD 或使用默认值
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "resume_genius"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}

	// 每个测试在事务中执行，结束后回滚
	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	// 确保 AutoMigrate（表结构存在即可）
	tx.AutoMigrate(&models.Project{}, &models.Asset{})

	return tx
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/intake/testutil.go
git commit -m "test(intake): add PostgreSQL test helper with transaction isolation"
```

---

### Task 4: ProjectService + 测试（TDD）

**Files:**
- Create: `backend/internal/modules/intake/service_test.go`（ProjectService 部分）
- Create: `backend/internal/modules/intake/service.go`（ProjectService 部分）

- [ ] **Step 1: 写 ProjectService 失败测试**

创建 `backend/internal/modules/intake/service_test.go`：

```go
package intake

import (
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/stretchr/testify/assert"
)

// --- ProjectService 测试 ---

func TestProjectService_Create(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	proj, err := svc.Create("user-1", "前端工程师简历")
	assert.NoError(t, err)
	assert.Equal(t, "前端工程师简历", proj.Title)
	assert.Equal(t, "user-1", proj.UserID)
	assert.Equal(t, "active", proj.Status)
	assert.Greater(t, proj.ID, uint(0))
}

func TestProjectService_List_FiltersByUserID(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	svc.Create("user-1", "项目A")
	svc.Create("user-2", "项目B")
	svc.Create("user-1", "项目C")

	projects, err := svc.List("user-1")
	assert.NoError(t, err)
	assert.Len(t, projects, 2)

	titles := make([]string, len(projects))
	for i, p := range projects {
		titles[i] = p.Title
	}
	assert.Contains(t, titles, "项目A")
	assert.Contains(t, titles, "项目C")
}

func TestProjectService_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "测试项目")

	proj, err := svc.GetByID("user-1", created.ID)
	assert.NoError(t, err)
	assert.Equal(t, "测试项目", proj.Title)
}

func TestProjectService_GetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.GetByID("user-1", 9999)
	assert.Error(t, err)
}

func TestProjectService_GetByID_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	_, err := svc.GetByID("user-2", created.ID)
	assert.Error(t, err)
}

func TestProjectService_Delete(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "待删除")

	err := svc.Delete("user-1", created.ID)
	assert.NoError(t, err)

	_, err = svc.GetByID("user-1", created.ID)
	assert.Error(t, err)
}

func TestProjectService_Delete_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	err := svc.Delete("user-2", created.ID)
	assert.Error(t, err)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && docker compose up -d postgres && sleep 2 && go test ./internal/modules/intake/... -run TestProjectService -v`
Expected: FAIL（`NewProjectService` 未定义）

- [ ] **Step 3: 安装 testify**

Run: `cd backend && go get github.com/stretchr/testify`

- [ ] **Step 4: 实现 ProjectService**

创建 `backend/internal/modules/intake/service.go`：

```go
package intake

import (
	"errors"
	"fmt"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type ProjectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) Create(userID, title string) (*models.Project, error) {
	proj := models.Project{
		UserID: userID,
		Title:  title,
		Status: "active",
	}
	if err := s.db.Create(&proj).Error; err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &proj, nil
}

func (s *ProjectService) List(userID string) ([]models.Project, error) {
	var projects []models.Project
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (s *ProjectService) GetByID(userID string, id uint) (*models.Project, error) {
	var proj models.Project
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&proj).Error; err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &proj, nil
}

func (s *ProjectService) Delete(userID string, id uint) error {
	result := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Project{})
	if result.Error != nil {
		return fmt.Errorf("delete project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("project not found")
	}
	return nil
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/intake/... -run TestProjectService -v`
Expected: PASS（7 个测试全部通过）

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/intake/service.go backend/internal/modules/intake/service_test.go backend/go.mod backend/go.sum
git commit -m "feat(intake): implement ProjectService with user isolation and tests"
```

---

### Task 5: AssetService + 测试（TDD）

**Files:**
- Modify: `backend/internal/modules/intake/service_test.go`（追加 AssetService 测试）
- Modify: `backend/internal/modules/intake/service.go`（追加 AssetService）

- [ ] **Step 1: 写 AssetService 失败测试**

追加到 `backend/internal/modules/intake/service_test.go`：

```go
// --- AssetService 测试 ---

func TestAssetService_UploadFile(t *testing.T) {
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)
	svc := NewAssetService(db, storage)

	// 创建项目
	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})

	asset, err := svc.UploadFile("user-1", 1, "resume.pdf", []byte("fake-pdf"), 102400)
	assert.NoError(t, err)
	assert.Equal(t, "resume_pdf", asset.Type)
	assert.Equal(t, uint(1), asset.ProjectID)
	assert.NotNil(t, asset.URI)
	assert.NotNil(t, asset.Label)
	assert.Equal(t, "resume.pdf", *asset.Label)
}

func TestAssetService_UploadFile_UnsupportedFormat(t *testing.T) {
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)
	svc := NewAssetService(db, storage)

	_, err := svc.UploadFile("user-1", 1, "resume.exe", []byte("bad"), 100)
	assert.Error(t, err)
	assert.Equal(t, ErrUnsupportedFormat, err)
}

func TestAssetService_UploadFile_ExceedsSizeLimit(t *testing.T) {
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)
	svc := NewAssetService(db, storage)

	largeData := make([]byte, 21*1024*1024) // 21MB
	_, err := svc.UploadFile("user-1", 1, "resume.pdf", largeData, int64(len(largeData)))
	assert.Error(t, err)
	assert.Equal(t, ErrFileTooLarge, err)
}

func TestAssetService_UploadFile_ProjectNotFound(t *testing.T) {
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)
	svc := NewAssetService(db, storage)

	_, err := svc.UploadFile("user-1", 9999, "resume.pdf", []byte("data"), 100)
	assert.Error(t, err)
}

func TestAssetService_CreateGitRepo(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})

	asset, err := svc.CreateGitRepo("user-1", 1, "https://github.com/user/repo")
	assert.NoError(t, err)
	assert.Equal(t, "git_repo", asset.Type)
	assert.NotNil(t, asset.URI)
	assert.Equal(t, "https://github.com/user/repo", *asset.URI)
}

func TestAssetService_CreateGitRepo_InvalidURL(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	_, err := svc.CreateGitRepo("user-1", 1, "not-a-url")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidGitURL, err)
}

func TestAssetService_CreateNote(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})

	asset, err := svc.CreateNote("user-1", 1, "目标岗位是全栈", "求职意向")
	assert.NoError(t, err)
	assert.Equal(t, "note", asset.Type)
	assert.NotNil(t, asset.Content)
	assert.Equal(t, "目标岗位是全栈", *asset.Content)
	assert.NotNil(t, asset.Label)
	assert.Equal(t, "求职意向", *asset.Label)
}

func TestAssetService_UpdateNote(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})
	note, _ := svc.CreateNote("user-1", 1, "旧内容", "标签")

	updated, err := svc.UpdateNote("user-1", note.ID, "新内容", "新标签")
	assert.NoError(t, err)
	assert.Equal(t, "新内容", *updated.Content)
	assert.Equal(t, "新标签", *updated.Label)
}

func TestAssetService_UpdateNote_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})
	note, _ := svc.CreateNote("user-1", 1, "内容", "标签")

	_, err := svc.UpdateNote("user-2", note.ID, "修改", "改")
	assert.Error(t, err)
}

func TestAssetService_ListByProject(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})
	db.Create(&models.Project{UserID: "user-2", Title: "other", Status: "active"})

	svc.CreateNote("user-1", 1, "笔记1", "标签")
	svc.CreateNote("user-2", 2, "笔记2", "标签")
	svc.UploadFile("user-1", 1, "resume.pdf", []byte("data"), 100)

	assets, err := svc.ListByProject("user-1", 1)
	assert.NoError(t, err)
	assert.Len(t, assets, 2)
}

func TestAssetService_DeleteAsset(t *testing.T) {
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})

	asset, _ := svc.UploadFile("user-1", 1, "resume.pdf", []byte("data"), 100)
	assert.True(t, storage.Exists(*asset.URI))

	err := svc.DeleteAsset("user-1", asset.ID)
	assert.NoError(t, err)
	assert.False(t, storage.Exists(*asset.URI))
}

func TestAssetService_DeleteAsset_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})
	asset, _ := svc.CreateNote("user-1", 1, "内容", "标签")

	err := svc.DeleteAsset("user-2", asset.ID)
	assert.Error(t, err)
}

func TestAssetService_DeleteProjectAssets(t *testing.T) {
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)
	svc := NewAssetService(db, storage)

	db.Create(&models.Project{UserID: "user-1", Title: "test", Status: "active"})

	svc.UploadFile("user-1", 1, "resume.pdf", []byte("data1"), 100)
	svc.UploadFile("user-1", 1, "doc.docx", []byte("data2"), 200)
	svc.CreateNote("user-1", 1, "笔记", "标签")

	err := svc.DeleteProjectAssets("user-1", 1)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.Asset{}).Where("project_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/intake/... -run TestAssetService -v`
Expected: FAIL（`NewAssetService` 未定义）

- [ ] **Step 3: 实现 AssetService**

追加到 `backend/internal/modules/intake/service.go`：

```go
import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

// --- 错误定义 ---

var (
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrFileTooLarge      = errors.New("file size exceeds 20MB limit")
	ErrInvalidGitURL     = errors.New("invalid git repository URL")
	ErrProjectNotFound   = errors.New("project not found")
	ErrAssetNotFound     = errors.New("asset not found")
)

var allowedExtensions = map[string]string{
	".pdf":  "resume_pdf",
	".docx": "resume_docx",
	".png":  "resume_image",
	".jpg":  "resume_image",
	".jpeg": "resume_image",
}

var maxFileSize = 20 * 1024 * 1024 // 20MB

var gitURLPattern = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

// --- AssetService ---

type AssetService struct {
	db      *gorm.DB
	storage FileStorage
}

func NewAssetService(db *gorm.DB, storage FileStorage) *AssetService {
	return &AssetService{db: db, storage: storage}
}

func (s *AssetService) UploadFile(userID string, projectID uint, filename string, data []byte, size int64) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}
	if err := s.validateFile(filename, size); err != nil {
		return nil, err
	}

	savedPath, err := s.storage.Save(projectID, filename, data)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	assetType := allowedExtensions[ext]
	uri := savedPath
	label := filename

	metadata := models.JSONB{
		"filename":   filename,
		"size_bytes": size,
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      assetType,
		URI:       &uri,
		Label:     &label,
		Metadata:  metadata,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		// 回滚文件
		s.storage.Delete(savedPath)
		return nil, fmt.Errorf("create asset record: %w", err)
	}
	return &asset, nil
}

func (s *AssetService) CreateGitRepo(userID string, projectID uint, repoURL string) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}
	if !gitURLPattern.MatchString(repoURL) {
		return nil, ErrInvalidGitURL
	}
	if _, err := url.Parse(repoURL); err != nil {
		return nil, ErrInvalidGitURL
	}

	uri := repoURL
	asset := models.Asset{
		ProjectID: projectID,
		Type:      "git_repo",
		URI:       &uri,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("create git asset: %w", err)
	}
	return &asset, nil
}

func (s *AssetService) CreateNote(userID string, projectID uint, content, label string) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      "note",
		Content:   &content,
		Label:     &label,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	return &asset, nil
}

func (s *AssetService) UpdateNote(userID string, noteID uint, content, label string) (*models.Asset, error) {
	var asset models.Asset
	if err := s.db.Where("id = ? AND type = ?", noteID, "note").First(&asset).Error; err != nil {
		return nil, ErrAssetNotFound
	}
	// 验证归属
	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return nil, err
	}

	asset.Content = &content
	asset.Label = &label
	if err := s.db.Save(&asset).Error; err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	return &asset, nil
}

func (s *AssetService) ListByProject(userID string, projectID uint) ([]models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	var assets []models.Asset
	if err := s.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&assets).Error; err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	return assets, nil
}

func (s *AssetService) DeleteAsset(userID string, assetID uint) error {
	var asset models.Asset
	if err := s.db.Where("id = ?", assetID).First(&asset).Error; err != nil {
		return ErrAssetNotFound
	}
	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return err
	}

	// 删除文件（如有）
	if asset.URI != nil && *asset.URI != "" {
		s.storage.Delete(*asset.URI)
	}

	if err := s.db.Delete(&asset).Error; err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}
	return nil
}

func (s *AssetService) DeleteProjectAssets(userID string, projectID uint) error {
	if err := s.validateProject(userID, projectID); err != nil {
		return err
	}

	var assets []models.Asset
	s.db.Where("project_id = ?", projectID).Find(&assets)

	for _, asset := range assets {
		if asset.URI != nil && *asset.URI != "" {
			s.storage.Delete(*asset.URI)
		}
	}

	return s.db.Where("project_id = ?", projectID).Delete(&models.Asset{}).Error
}

func (s *AssetService) validateProject(userID string, projectID uint) error {
	var count int64
	s.db.Model(&models.Project{}).Where("id = ? AND user_id = ?", projectID, userID).Count(&count)
	if count == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *AssetService) validateFile(filename string, size int64) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := allowedExtensions[ext]; !ok {
		return ErrUnsupportedFormat
	}
	if size > int64(maxFileSize) {
		return ErrFileTooLarge
	}
	return nil
}
```

> 注意：service.go 开头的 import 需要合并去重，最终的 import 块为：
```go
import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/intake/... -run TestAssetService -v`
Expected: PASS（13 个测试全部通过）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/intake/service.go backend/internal/modules/intake/service_test.go
git commit -m "feat(intake): implement AssetService with file upload, git, notes and tests"
```

---

### Task 6: Handler 层 + 测试（TDD）

**Files:**
- Create: `backend/internal/modules/intake/handler_test.go`
- Create: `backend/internal/modules/intake/handler.go`

- [ ] **Step 1: 写 Handler 失败测试**

创建 `backend/internal/modules/intake/handler_test.go`：

```go
package intake

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
)

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	h := NewHandler(
		NewProjectService(db),
		NewAssetService(db, storage),
	)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "test-user-1")
		c.Next()
	})

	return h, r
}

func createMultipartForm(t *testing.T, fieldName, fileName string, content []byte, extraFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	assert.NoError(t, err)
	part.Write(content)

	for k, v := range extraFields {
		writer.WriteField(k, v)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

// --- Project Handlers ---

func TestHandler_CreateProject(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/projects", h.CreateProject)

	body, _ := json.Marshal(map[string]string{"title": "前端工程师简历"})
	req := httptest.NewRequest("POST", "/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "前端工程师简历", data["title"])
}

func TestHandler_CreateProject_EmptyTitle(t *testing.T) {
	h, r := setupTestHandler(t)
	r.POST("/projects", h.CreateProject)

	body, _ := json.Marshal(map[string]string{"title": ""})
	req := httptest.NewRequest("POST", "/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestHandler_ListProjects(t *testing.T) {
	h, r := setupTestHandler(t)
	r.GET("/projects", h.ListProjects)

	// 先创建几个项目
	h.projectSvc.Create("test-user-1", "项目A")
	h.projectSvc.Create("test-user-1", "项目B")

	req := httptest.NewRequest("GET", "/projects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestHandler_GetProject(t *testing.T) {
	h, r := setupTestHandler(t)
	r.GET("/projects/:project_id", h.GetProject)

	proj, _ := h.projectSvc.Create("test-user-1", "测试项目")

	req := httptest.NewRequest("GET", "/projects/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(proj.ID), data["id"])
}

func TestHandler_DeleteProject(t *testing.T) {
	h, r := setupTestHandler(t)
	r.DELETE("/projects/:project_id", h.DeleteProject)

	h.projectSvc.Create("test-user-1", "待删除")

	req := httptest.NewRequest("DELETE", "/projects/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

// --- Asset Handlers ---

func TestHandler_UploadFile(t *testing.T) {
	h, r := setupTestHandler(t)
	// 需要先创建项目
	h.projectSvc.Create("test-user-1", "测试")

	r.POST("/assets/upload", h.UploadFile)

	body, contentType := createMultipartForm(t, "file", "resume.pdf", []byte("fake-pdf"), map[string]string{
		"project_id": "1",
		"type":       "resume_pdf",
	})

	req := httptest.NewRequest("POST", "/assets/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "resume_pdf", data["type"])
}

func TestHandler_UploadFile_UnsupportedFormat(t *testing.T) {
	h, r := setupTestHandler(t)
	h.projectSvc.Create("test-user-1", "测试")

	r.POST("/assets/upload", h.UploadFile)

	body, contentType := createMultipartForm(t, "file", "virus.exe", []byte("bad"), map[string]string{
		"project_id": "1",
	})

	req := httptest.NewRequest("POST", "/assets/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(1001), resp["code"])
}

func TestHandler_CreateGitRepo(t *testing.T) {
	h, r := setupTestHandler(t)
	h.projectSvc.Create("test-user-1", "测试")

	r.POST("/assets/git", h.CreateGitRepo)

	body, _ := json.Marshal(map[string]string{
		"project_id": "1",
		"repo_url":   "https://github.com/user/repo",
	})

	req := httptest.NewRequest("POST", "/assets/git", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "git_repo", data["type"])
}

func TestHandler_ListAssets(t *testing.T) {
	h, r := setupTestHandler(t)
	h.projectSvc.Create("test-user-1", "测试")

	r.GET("/assets", h.ListAssets)

	h.assetSvc.CreateNote("test-user-1", 1, "笔记", "标签")

	req := httptest.NewRequest("GET", "/assets?project_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestHandler_DeleteAsset(t *testing.T) {
	h, r := setupTestHandler(t)
	h.projectSvc.Create("test-user-1", "测试")

	r.DELETE("/assets/:asset_id", h.DeleteAsset)

	asset, _ := h.assetSvc.CreateNote("test-user-1", 1, "内容", "标签")

	req := httptest.NewRequest("DELETE", "/assets/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestHandler_CreateNote(t *testing.T) {
	h, r := setupTestHandler(t)
	h.projectSvc.Create("test-user-1", "测试")

	r.POST("/assets/notes", h.CreateNote)

	body, _ := json.Marshal(map[string]string{
		"project_id": "1",
		"content":    "目标岗位是全栈",
		"label":      "求职意向",
	})

	req := httptest.NewRequest("POST", "/assets/notes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "note", data["type"])
}

func TestHandler_UpdateNote(t *testing.T) {
	h, r := setupTestHandler(t)
	h.projectSvc.Create("test-user-1", "测试")

	r.PUT("/assets/notes/:note_id", h.UpdateNote)

	h.assetSvc.CreateNote("test-user-1", 1, "旧内容", "旧标签")

	body, _ := json.Marshal(map[string]string{
		"content": "新内容",
		"label":   "新标签",
	})

	req := httptest.NewRequest("PUT", "/assets/notes/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "新内容", data["content"])
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/intake/... -run TestHandler -v`
Expected: FAIL（`NewHandler` 未定义）

- [ ] **Step 3: 实现 handler.go**

创建 `backend/internal/modules/intake/handler.go`：

```go
package intake

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

type Handler struct {
	projectSvc *ProjectService
	assetSvc   *AssetService
}

func NewHandler(projectSvc *ProjectService, assetSvc *AssetService) *Handler {
	return &Handler{projectSvc: projectSvc, assetSvc: assetSvc}
}

func userID(c *gin.Context) string {
	v, _ := c.Get(middleware.ContextUserID)
	return v.(string)
}

// --- Project Handlers ---

type createProjectReq struct {
	Title string `json:"title" binding:"required"`
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 10001, "title is required")
		return
	}

	proj, err := h.projectSvc.Create(userID(c), req.Title)
	if err != nil {
		response.Error(c, 50001, err.Error())
		return
	}
	response.Success(c, proj)
}

func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.projectSvc.List(userID(c))
	if err != nil {
		response.Error(c, 50001, err.Error())
		return
	}
	response.Success(c, projects)
}

func (h *Handler) GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("project_id"), 10, 64)
	if err != nil {
		response.Error(c, 10001, "invalid project_id")
		return
	}

	proj, err := h.projectSvc.GetByID(userID(c), uint(id))
	if err != nil {
		response.Error(c, 1004, "project not found")
		return
	}
	response.Success(c, proj)
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("project_id"), 10, 64)
	if err != nil {
		response.Error(c, 10001, "invalid project_id")
		return
	}

	// 先级联删除关联资产
	if delErr := h.assetSvc.DeleteProjectAssets(userID(c), uint(id)); delErr != nil {
		response.Error(c, 50001, delErr.Error())
		return
	}

	if err := h.projectSvc.Delete(userID(c), uint(id)); err != nil {
		response.Error(c, 1004, "project not found")
		return
	}
	response.Success(c, nil)
}

// --- Asset Handlers ---

func (h *Handler) UploadFile(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.PostForm("project_id"), 10, 64)
	if err != nil {
		response.Error(c, 10001, "project_id is required")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, 1001, "file is required")
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Error(c, 50001, "failed to read file")
		return
	}
	defer src.Close()

	data := make([]byte, file.Size)
	if _, err := src.Read(data); err != nil {
		response.Error(c, 50001, "failed to read file content")
		return
	}

	asset, err := h.assetSvc.UploadFile(userID(c), uint(projectID), file.Filename, data, file.Size)
	if err != nil {
		switch err {
		case ErrUnsupportedFormat:
			response.Error(c, 1001, "unsupported file format")
		case ErrFileTooLarge:
			response.Error(c, 1002, "file size exceeds 20MB")
		case ErrProjectNotFound:
			response.Error(c, 1004, "project not found")
		default:
			response.Error(c, 50001, err.Error())
		}
		return
	}
	response.Success(c, asset)
}

type createGitReq struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	RepoURL   string `json:"repo_url" binding:"required"`
}

func (h *Handler) CreateGitRepo(c *gin.Context) {
	var req createGitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 10001, "project_id and repo_url are required")
		return
	}

	asset, err := h.assetSvc.CreateGitRepo(userID(c), req.ProjectID, req.RepoURL)
	if err != nil {
		switch err {
		case ErrInvalidGitURL:
			response.Error(c, 1003, "invalid git repository URL")
		case ErrProjectNotFound:
			response.Error(c, 1004, "project not found")
		default:
			response.Error(c, 50001, err.Error())
		}
		return
	}
	response.Success(c, asset)
}

func (h *Handler) ListAssets(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Query("project_id"), 10, 64)
	if err != nil {
		response.Error(c, 10001, "project_id is required")
		return
	}

	assets, err := h.assetSvc.ListByProject(userID(c), uint(projectID))
	if err != nil {
		response.Error(c, 50001, err.Error())
		return
	}
	response.Success(c, assets)
}

func (h *Handler) DeleteAsset(c *gin.Context) {
	assetID, err := strconv.ParseUint(c.Param("asset_id"), 10, 64)
	if err != nil {
		response.Error(c, 10001, "invalid asset_id")
		return
	}

	if err := h.assetSvc.DeleteAsset(userID(c), uint(assetID)); err != nil {
		switch err {
		case ErrAssetNotFound:
			response.Error(c, 1006, "asset not found")
		case ErrProjectNotFound:
			response.Error(c, 1004, "project not found")
		default:
			response.Error(c, 50001, err.Error())
		}
		return
	}
	response.Success(c, nil)
}

type createNoteReq struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
	Label     string `json:"label"`
}

func (h *Handler) CreateNote(c *gin.Context) {
	var req createNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 10001, "project_id and content are required")
		return
	}

	asset, err := h.assetSvc.CreateNote(userID(c), req.ProjectID, req.Content, req.Label)
	if err != nil {
		switch err {
		case ErrProjectNotFound:
			response.Error(c, 1004, "project not found")
		default:
			response.Error(c, 50001, err.Error())
		}
		return
	}
	response.Success(c, asset)
}

type updateNoteReq struct {
	Content string `json:"content" binding:"required"`
	Label   string `json:"label"`
}

func (h *Handler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("note_id"), 10, 64)
	if err != nil {
		response.Error(c, 10001, "invalid note_id")
		return
	}

	var req updateNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 10001, "content is required")
		return
	}

	asset, err := h.assetSvc.UpdateNote(userID(c), uint(noteID), req.Content, req.Label)
	if err != nil {
		switch err {
		case ErrAssetNotFound:
			response.Error(c, 1006, "note not found")
		case ErrProjectNotFound:
			response.Error(c, 1004, "project not found")
		default:
			response.Error(c, 50001, err.Error())
		}
		return
	}
	response.Success(c, asset)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/intake/... -run TestHandler -v`
Expected: PASS（15 个 handler 测试全部通过）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/intake/handler.go backend/internal/modules/intake/handler_test.go
git commit -m "feat(intake): implement all 10 handlers with request validation and error mapping"
```

---

### Task 7: 路由注册 + 集成测试

**Files:**
- Modify: `backend/internal/modules/intake/routes.go`（重写）
- Modify: `backend/cmd/server/main_test.go`（扩展路由测试）

- [ ] **Step 1: 重写 routes.go**

重写 `backend/internal/modules/intake/routes.go`：

```go
package intake

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, uploadDir string) {
	storage := NewLocalStorage(uploadDir)
	projectSvc := NewProjectService(db)
	assetSvc := NewAssetService(db, storage)
	h := NewHandler(projectSvc, assetSvc)

	// Project CRUD
	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects", h.ListProjects)
	rg.GET("/projects/:project_id", h.GetProject)
	rg.DELETE("/projects/:project_id", h.DeleteProject)

	// Asset management
	rg.POST("/assets/upload", h.UploadFile)
	rg.POST("/assets/git", h.CreateGitRepo)
	rg.GET("/assets", h.ListAssets)
	rg.DELETE("/assets/:asset_id", h.DeleteAsset)

	// Notes
	rg.POST("/assets/notes", h.CreateNote)
	rg.PUT("/assets/notes/:note_id", h.UpdateNote)
}
```

- [ ] **Step 2: 更新 main.go 传入 uploadDir**

修改 `backend/cmd/server/main.go`，更新 `intake.RegisterRoutes` 调用：

```go
// 在 setupRouter 或 main 中添加
uploadDir := os.Getenv("UPLOAD_DIR")
if uploadDir == "" {
	uploadDir = "./uploads"
}
os.MkdirAll(uploadDir, 0755)

// 在路由注册处
intake.RegisterRoutes(v1, db, uploadDir)
```

需要在 main.go 顶部添加 `"os"` import。

- [ ] **Step 3: 更新 main_test.go 添加新路由断言**

追加到 `backend/cmd/server/main_test.go` 的 `cases` 列表中：

```go
{http.MethodPost, "/api/v1/projects"},
{http.MethodGet, "/api/v1/projects/1"},
{http.MethodDelete, "/api/v1/projects/1"},
{http.MethodPost, "/api/v1/assets/upload"},
{http.MethodPost, "/api/v1/assets/git"},
{http.MethodGet, "/api/v1/assets"},
{http.MethodDelete, "/api/v1/assets/1"},
{http.MethodPost, "/api/v1/assets/notes"},
{http.MethodPut, "/api/v1/assets/notes/1"},
```

> 注意：`setupRouter(nil)` 现在需要 uploadDir，测试中传空字符串即可，storage 创建空目录不会报错。需要同步更新 `setupRouter` 签名。

- [ ] **Step 4: 运行全部后端测试**

Run: `cd backend && go test ./... -v`
Expected: 所有测试通过

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/intake/routes.go backend/cmd/server/main.go backend/cmd/server/main_test.go
git commit -m "feat(intake): wire up all 10 routes with integration tests"
```

---

### Task 8: 前端 — 依赖安装 + CSS 主题变量

**Files:**
- Modify: `frontend/workbench/package.json`
- Modify: `frontend/workbench/src/index.css`

- [ ] **Step 1: 安装前端测试依赖**

Run:
```bash
cd frontend/workbench && bun add -d @testing-library/react @testing-library/user-event @testing-library/jest-dom jsdom
```

- [ ] **Step 2: 安装 shadcn/ui 基础组件**

Run:
```bash
cd frontend/workbench && bunx shadcn@latest add button input dialog textarea
```

- [ ] **Step 3: 配置 CSS 变量 + Warm Editorial 主题**

重写 `frontend/workbench/src/index.css`：

```css
@import "tailwindcss";

@theme {
  --color-bg-page: #faf8f5;
  --color-bg-card: #ffffff;
  --color-text-primary: #1a1815;
  --color-text-secondary: #5c5550;
  --color-text-muted: #9c9590;
  --color-accent: #c4956a;
  --color-accent-hover: #b5855a;
  --color-accent-bg: #f0ebe4;
  --color-border: #e8e4df;
  --color-border-focus: #c4956a;
  --color-success: #0d652d;
  --color-success-bg: #e6f4ea;
  --color-error: #c5221f;
  --color-error-bg: #fce8e6;
  --color-warning: #b06000;
  --color-warning-bg: #fef7e0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background-color: var(--color-bg-page);
  color: var(--color-text-primary);
}

.font-serif {
  font-family: Georgia, 'Times New Roman', serif;
}
```

- [ ] **Step 4: 验证前端构建**

Run: `cd frontend/workbench && bun run build`
Expected: 构建成功

- [ ] **Step 5: Commit**

```bash
git add frontend/workbench/package.json frontend/workbench/bun.lock frontend/workbench/src/index.css frontend/workbench/src/components/ui/
git commit -m "feat(frontend): install shadcn/ui components and add Warm Editorial theme tokens"
```

---

### Task 9: 前端 — API Client 更新（X-User-ID + multipart）

**Files:**
- Modify: `frontend/workbench/src/lib/api-client.ts`

- [ ] **Step 1: 更新 api-client.ts**

重写 `frontend/workbench/src/lib/api-client.ts`：

```typescript
const BASE = '/api/v1'

function getUserID(): string {
  let id = localStorage.getItem('user_id')
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem('user_id', id)
  }
  return id
}

function headers(extra?: Record<string, string>): Record<string, string> {
  return {
    'X-User-ID': getUserID(),
    ...extra,
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: headers(options?.headers as Record<string, string>),
  })
  const json = await res.json()
  if (json.code !== 0) {
    throw new ApiError(json.code, json.message)
  }
  return json.data as T
}

export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

export const apiClient = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    }),

  put: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    }),

  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),

  upload: <T>(path: string, formData: FormData) =>
    request<T>(path, {
      method: 'POST',
      body: formData,
      // 不设置 Content-Type，让浏览器自动设置 boundary
    }),
}
```

- [ ] **Step 2: 更新已有测试通过**

修改 `frontend/workbench/tests/api-client.test.ts`，适配新的 header 逻辑（mock localStorage 和 crypto）：

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockFetch = vi.fn()

vi.stubGlobal('fetch', mockFetch)
vi.stubGlobal('localStorage', {
  getItem: vi.fn(() => 'test-uuid'),
  setItem: vi.fn(),
})
vi.stubGlobal('crypto', { randomUUID: vi.fn(() => 'test-uuid') })

// 导入需要 mock 之后
const { apiClient, ApiError } = await import('../src/lib/api-client')

describe('apiClient', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('sends GET with X-User-ID header', async () => {
    mockFetch.mockResolvedValue({
      json: () => ({ code: 0, data: 'ok', message: 'ok' }),
    })
    await apiClient.get('/test')
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/test'),
      expect.objectContaining({
        headers: expect.objectContaining({ 'X-User-ID': 'test-uuid' }),
      }),
    )
  })

  it('throws ApiError on non-zero code', async () => {
    mockFetch.mockResolvedValue({
      json: () => ({ code: 1004, data: null, message: 'not found' }),
    })
    await expect(apiClient.get('/test')).rejects.toThrow('not found')
  })

  it('ApiError carries code', async () => {
    mockFetch.mockResolvedValue({
      json: () => ({ code: 1004, data: null, message: 'not found' }),
    })
    try {
      await apiClient.get('/test')
    } catch (e) {
      expect((e as ApiError).code).toBe(1004)
    }
  })
})
```

- [ ] **Step 3: 运行前端测试**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/workbench/src/lib/api-client.ts frontend/workbench/tests/api-client.test.ts
git commit -m "feat(frontend): add X-User-ID header and multipart upload support to api-client"
```

---

### Task 10: 前端 — ProjectList 页面 + 测试

**Files:**
- Create: `frontend/workbench/src/pages/ProjectList.tsx`
- Create: `frontend/workbench/src/pages/ProjectList.test.tsx`
- Modify: `frontend/workbench/src/App.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/workbench/src/pages/ProjectList.test.tsx`：

```tsx
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

// mock api-client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    get: vi.fn().mockResolvedValue([
      { id: 1, title: '前端工程师简历', status: 'active', created_at: '2026-04-28T10:00:00Z' },
      { id: 2, title: '产品经理简历', status: 'active', created_at: '2026-04-27T10:00:00Z' },
    ]),
    post: vi.fn().mockResolvedValue({
      id: 3, title: '新项目', status: 'active', created_at: '2026-04-28T12:00:00Z',
    }),
  },
  ApiError: class extends Error { code = 0 },
}))

import { ProjectList } from './ProjectList'
import { apiClient } from '../lib/api-client'

const mockedGet = vi.mocked(apiClient.get)
const mockedPost = vi.mocked(apiClient.post)

describe('ProjectList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // 默认 mock 返回值
    mockedGet.mockResolvedValue([
      { id: 1, title: '前端工程师简历', status: 'active', created_at: '2026-04-28T10:00:00Z' },
    ])
    mockedPost.mockResolvedValue({
      id: 2, title: '测试项目', status: 'active', created_at: '2026-04-28T12:00:00Z',
    })
  })

  it('renders project list', async () => {
    render(<ProjectList />)
    await waitFor(() => {
      expect(screen.getByText('前端工程师简历')).toBeInTheDocument()
    })
  })

  it('creates project on Enter key', async () => {
    const user = userEvent.setup()
    render(<ProjectList />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('输入项目名称，按回车创建...')).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText('输入项目名称，按回车创建...')
    await user.type(input, '测试项目{Enter}')

    await waitFor(() => {
      expect(mockedPost).toHaveBeenCalledWith('/projects', { title: '测试项目' })
    })
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/pages/ProjectList.test.tsx`
Expected: FAIL（模块不存在）

- [ ] **Step 3: 实现 ProjectList**

创建 `frontend/workbench/src/pages/ProjectList.tsx`：

```tsx
import { useState, useEffect, type KeyboardEvent } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '../lib/api-client'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'

interface Project {
  id: number
  title: string
  status: string
  created_at: string
  asset_count?: number
}

export function ProjectList() {
  const [projects, setProjects] = useState<Project[]>([])
  const [title, setTitle] = useState('')

  useEffect(() => {
    loadProjects()
  }, [])

  async function loadProjects() {
    const data = await apiClient.get<Project[]>('/projects')
    setProjects(data)
  }

  async function handleCreate() {
    if (!title.trim()) return
    await apiClient.post<Project>('/projects', { title: title.trim() })
    setTitle('')
    await loadProjects()
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleCreate()
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="min-h-screen" style={{ backgroundColor: 'var(--color-bg-page)' }}>
      <div className="mx-auto max-w-2xl px-6 py-10">
        <h1 className="font-serif text-2xl" style={{ color: 'var(--color-text-primary)' }}>
          ResumeGenius
        </h1>
        <p className="mt-1 text-sm" style={{ color: 'var(--color-text-muted)' }}>
          AI 辅助简历编辑
        </p>

        <div className="mt-8 flex gap-2">
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入项目名称，按回车创建..."
            className="flex-1"
          />
          <Button onClick={handleCreate} disabled={!title.trim()}>
            创建
          </Button>
        </div>

        <div className="mt-6" style={{ border: '1px solid var(--color-border)', borderRadius: '8px', overflow: 'hidden', background: 'var(--color-bg-card)' }}>
          {projects.length === 0 ? (
            <div className="px-6 py-12 text-center text-sm" style={{ color: 'var(--color-text-muted)' }}>
              还没有项目，创建一个开始吧
            </div>
          ) : (
            projects.map((p) => (
              <Link
                key={p.id}
                to={`/projects/${p.id}`}
                className="flex items-center justify-between px-5 py-3 transition-colors"
                style={{
                  borderBottom: '1px solid var(--color-border)',
                  textDecoration: 'none',
                  color: 'var(--color-text-primary)',
                }}
                onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = 'var(--color-bg-page)')}
                onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
              >
                <div>
                  <div className="text-sm font-medium">{p.title}</div>
                  <div className="mt-0.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    {p.created_at && formatDate(p.created_at)}
                  </div>
                </div>
                <span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>›</span>
              </Link>
            ))
          )}
        </div>

        <div className="mt-3 text-center text-xs" style={{ color: 'var(--color-text-muted)' }}>
          {projects.length > 0 ? `共 ${projects.length} 个项目` : ''}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 4: 更新 App.tsx 路由**

重写 `frontend/workbench/src/App.tsx`：

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ProjectList } from './pages/ProjectList'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ProjectList />} />
        <Route path="/projects/:projectId" element={<div>Project Detail (coming soon)</div>} />
        <Route path="/editor/:projectId" element={<div>Editor</div>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/pages/ProjectList.test.tsx -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/workbench/src/pages/ProjectList.tsx frontend/workbench/src/pages/ProjectList.test.tsx frontend/workbench/src/App.tsx
git commit -m "feat(frontend): implement ProjectList page with create functionality and test"
```

---

### Task 11: 前端 — ProjectDetail 页面 + AssetList + ProjectCard

**Files:**
- Create: `frontend/workbench/src/pages/ProjectDetail.tsx`
- Create: `frontend/workbench/src/pages/ProjectDetail.test.tsx`
- Create: `frontend/workbench/src/components/intake/ProjectCard.tsx`
- Create: `frontend/workbench/src/components/intake/AssetList.tsx`
- Create: `frontend/workbench/src/components/intake/DeleteConfirm.tsx`

- [ ] **Step 1: 实现 ProjectCard 组件**

创建 `frontend/workbench/src/components/intake/ProjectCard.tsx`：

```tsx
import { Link } from 'react-router-dom'

interface Project {
  id: number
  title: string
  status: string
  created_at: string
}

export function ProjectCard({ project }: { project: Project }) {
  return (
    <Link
      to={`/projects/${project.id}`}
      className="block transition-colors"
      style={{
        padding: '14px 18px',
        borderBottom: '1px solid var(--color-border)',
        textDecoration: 'none',
        color: 'var(--color-text-primary)',
      }}
      onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = 'var(--color-bg-page)')}
      onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
    >
      <div className="text-sm font-medium">{project.title}</div>
      <div className="mt-0.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
        {project.created_at && new Date(project.created_at).toLocaleDateString('zh-CN')}
      </div>
    </Link>
  )
}
```

- [ ] **Step 2: 实现 DeleteConfirm 组件**

创建 `frontend/workbench/src/components/intake/DeleteConfirm.tsx`：

```tsx
import { Button } from '../ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog'

interface DeleteConfirmProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  onConfirm: () => void
}

export function DeleteConfirm({ open, onOpenChange, title, description, onConfirm }: DeleteConfirmProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button
            variant="destructive"
            onClick={() => {
              onConfirm()
              onOpenChange(false)
            }}
          >
            删除
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 3: 实现 AssetList 组件**

创建 `frontend/workbench/src/components/intake/AssetList.tsx`：

```tsx
interface Asset {
  id: number
  project_id: number
  type: string
  uri?: string | null
  content?: string | null
  label?: string | null
  metadata?: Record<string, unknown>
  created_at: string
}

interface AssetListProps {
  assets: Asset[]
  onDelete: (id: number) => void
}

const typeConfig: Record<string, { bg: string; color: string; text: string }> = {
  resume_pdf:  { bg: 'var(--color-error-bg)',   color: 'var(--color-error)',   text: 'PDF' },
  resume_docx: { bg: 'var(--color-accent-bg)',  color: 'var(--color-accent)', text: 'DOC' },
  resume_image:{ bg: 'var(--color-success-bg)', color: 'var(--color-success)', text: 'IMG' },
  git_repo:    { bg: 'var(--color-success-bg)', color: 'var(--color-success)', text: 'Git' },
  note:        { bg: 'var(--color-accent-bg)',  color: 'var(--color-accent)', text: 'MEMO' },
}

function formatMeta(type: string, asset: Asset): string {
  if (type === 'note') return asset.content?.slice(0, 50) || ''
  if (type === 'git_repo') return asset.uri || ''
  const meta = asset.metadata as Record<string, unknown> | undefined
  if (meta?.size_bytes) {
    const bytes = Number(meta.size_bytes)
    if (bytes > 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
    return `${(bytes / 1024).toFixed(0)} KB`
  }
  return ''
}

export function AssetList({ assets, onDelete }: AssetListProps) {
  if (assets.length === 0) return null

  return (
    <div style={{ border: '1px solid var(--color-border)', borderRadius: '8px', overflow: 'hidden', background: 'var(--color-bg-card)' }}>
      {assets.map((asset, i) => {
        const cfg = typeConfig[asset.type] || typeConfig.note
        const displayLabel = asset.label || asset.uri?.split('/').pop() || '未命名'
        return (
          <div
            key={asset.id}
            className="flex items-center justify-between px-5 py-3"
            style={{ borderBottom: i < assets.length - 1 ? '1px solid var(--color-border)' : 'none' }}
          >
            <div className="flex items-center gap-3">
              <div
                style={{
                  width: '32px',
                  height: '32px',
                  borderRadius: '4px',
                  background: cfg.bg,
                  color: cfg.color,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '11px',
                  fontWeight: 600,
                  flexShrink: 0,
                }}
              >
                {cfg.text}
              </div>
              <div>
                <div className="text-sm font-medium">{displayLabel}</div>
                <div className="mt-0.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                  {formatMeta(asset.type, asset)}
                </div>
              </div>
            </div>
            <button
              onClick={() => onDelete(asset.id)}
              style={{ background: 'none', border: 'none', color: 'var(--color-text-muted)', fontSize: '12px', cursor: 'pointer' }}
            >
              删除
            </button>
          </div>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 4: 实现 ProjectDetail 页面**

创建 `frontend/workbench/src/pages/ProjectDetail.tsx`：

```tsx
import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { apiClient, ApiError } from '../lib/api-client'
import { Button } from '../components/ui/button'
import { AssetList } from '../components/intake/AssetList'
import { DeleteConfirm } from '../components/intake/DeleteConfirm'
import { UploadDialog } from '../components/intake/UploadDialog'
import { GitRepoDialog } from '../components/intake/GitRepoDialog'
import { NoteDialog } from '../components/intake/NoteDialog'

interface Project {
  id: number
  title: string
  status: string
  created_at: string
}

interface Asset {
  id: number
  project_id: number
  type: string
  uri?: string | null
  content?: string | null
  label?: string | null
  metadata?: Record<string, unknown>
  created_at: string
}

export function ProjectDetail() {
  const { projectId } = useParams()
  const navigate = useNavigate()
  const [project, setProject] = useState<Project | null>(null)
  const [assets, setAssets] = useState<Asset[]>([])
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [noteEditId, setNoteEditId] = useState<number | null>(null)

  const pid = Number(projectId)

  useEffect(() => {
    if (!projectId) return
    loadProject()
    loadAssets()
  }, [projectId])

  async function loadProject() {
    const data = await apiClient.get<Project>(`/projects/${projectId}`)
    setProject(data)
  }

  async function loadAssets() {
    const data = await apiClient.get<Asset[]>(`/assets?project_id=${projectId}`)
    setAssets(data)
  }

  async function handleDeleteAsset(id: number) {
    await apiClient.del(`/assets/${id}`)
    setDeleteTarget(null)
    loadAssets()
  }

  async function handleUploadComplete() {
    setUploadOpen(false)
    loadAssets()
  }

  async function handleGitComplete() {
    setGitOpen(false)
    loadAssets()
  }

  async function handleNoteComplete() {
    setNoteOpen(false)
    setNoteEditId(null)
    loadAssets()
  }

  async function handleDeleteProject() {
    await apiClient.del(`/projects/${pid}`)
    navigate('/')
  }

  function handleEditNote(id: number) {
    setNoteEditId(id)
    setNoteOpen(true)
  }

  if (!project) return <div className="p-6">加载中...</div>

  return (
    <div className="min-h-screen" style={{ backgroundColor: 'var(--color-bg-page)' }}>
      <div className="mx-auto max-w-2xl px-6 py-10">
        <Link to="/" className="text-sm" style={{ color: 'var(--color-text-muted)', textDecoration: 'none' }}>
          ← 返回项目列表
        </Link>
        <h1 className="font-serif mt-4 text-2xl" style={{ color: 'var(--color-text-primary)' }}>
          {project.title}
        </h1>

        <div className="mt-6 flex gap-2">
          <Button onClick={() => setUploadOpen(true)}>上传文件</Button>
          <Button variant="outline" onClick={() => setGitOpen(true)}>接入 Git</Button>
          <Button variant="outline" onClick={() => { setNoteEditId(null); setNoteOpen(true) }}>添加备注</Button>
        </div>

        <div className="mt-6">
          <AssetList
            assets={assets}
            onDelete={setDeleteTarget}
          />
        </div>

        {assets.length === 0 && (
          <div className="mt-6 text-center text-sm" style={{ color: 'var(--color-text-muted)' }}>
            暂无资料，上传文件或添加备注开始
          </div>
        )}

        <div className="mt-10 border-t pt-6" style={{ borderColor: 'var(--color-border)' }}>
          <Button variant="ghost" style={{ color: 'var(--color-error)' }} onClick={handleDeleteProject}>
            删除此项目
          </Button>
        </div>
      </div>

      <UploadDialog open={uploadOpen} onOpenChange={setUploadOpen} projectId={pid} onComplete={handleUploadComplete} />
      <GitRepoDialog open={gitOpen} onOpenChange={setGitOpen} projectId={pid} onComplete={handleGitComplete} />
      <NoteDialog open={noteOpen} onOpenChange={setNoteOpen} projectId={pid} noteId={noteEditId} onComplete={handleNoteComplete} />

      <DeleteConfirm
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="删除资料"
        description="确定要删除这条资料吗？文件将从服务器永久移除。"
        onConfirm={() => deleteTarget && handleDeleteAsset(deleteTarget)}
      />
    </div>
  )
}
```

> 注意：`UploadDialog`、`GitRepoDialog`、`NoteDialog` 在 Task 12 实现。此步骤可先用空壳占位让页面可编译。

- [ ] **Step 5: 更新 App.tsx 添加 ProjectDetail 路由**

在 `App.tsx` 中导入 `ProjectDetail`：

```tsx
import { ProjectDetail } from './pages/ProjectDetail'

// 在 Routes 中替换占位：
<Route path="/projects/:projectId" element={<ProjectDetail />} />
```

- [ ] **Step 6: Commit**

```bash
git add frontend/workbench/src/pages/ProjectDetail.tsx frontend/workbench/src/components/intake/ProjectCard.tsx frontend/workbench/src/components/intake/AssetList.tsx frontend/workbench/src/components/intake/DeleteConfirm.tsx frontend/workbench/src/App.tsx
git commit -m "feat(frontend): implement ProjectDetail page with AssetList and DeleteConfirm"
```

---

### Task 12: 前端 — UploadDialog / GitRepoDialog / NoteDialog

**Files:**
- Create: `frontend/workbench/src/components/intake/UploadDialog.tsx`
- Create: `frontend/workbench/src/components/intake/UploadDialog.test.tsx`
- Create: `frontend/workbench/src/components/intake/GitRepoDialog.tsx`
- Create: `frontend/workbench/src/components/intake/NoteDialog.tsx`

- [ ] **Step 1: 写 UploadDialog 失败测试**

创建 `frontend/workbench/src/components/intake/UploadDialog.test.tsx`：

```tsx
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../lib/api-client', () => ({
  apiClient: {
    upload: vi.fn().mockResolvedValue({ id: 1, type: 'resume_pdf' }),
  },
  ApiError: class extends Error { code = 0 },
}))

import { UploadDialog } from './UploadDialog'
import { apiClient } from '../../lib/api-client'

const mockedUpload = vi.mocked(apiClient.upload)

describe('UploadDialog', () => {
  beforeEach(() => vi.clearAllMocks())

  it('calls apiClient.upload when file is selected', async () => {
    const onComplete = vi.fn()
    render(
      <UploadDialog open={true} onOpenChange={vi.fn()} projectId={1} onComplete={onComplete} />,
    )

    // 找到 input[type=file] 并触发文件选择
    const input = screen.getByLabelText('选择文件')
    const file = new File(['content'], 'resume.pdf', { type: 'application/pdf' })
    await userEvent.upload(input, file)

    await waitFor(() => {
      expect(mockedUpload).toHaveBeenCalled()
    })
  })
})
```

- [ ] **Step 2: 实现 UploadDialog**

创建 `frontend/workbench/src/components/intake/UploadDialog.tsx`：

```tsx
import { useState, useRef, type ChangeEvent } from 'react'
import { apiClient, ApiError } from '../../lib/api-client'
import { Button } from '../ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog'

const ALLOWED_TYPES = ['.pdf', '.docx', '.png', '.jpg', '.jpeg']
const MAX_SIZE = 20 * 1024 * 1024

interface UploadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: number
  onComplete: () => void
}

export function UploadDialog({ open, onOpenChange, projectId, onComplete }: UploadDialogProps) {
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  async function handleFile(file: File) {
    setError('')

    const ext = '.' + file.name.split('.').pop()?.toLowerCase()
    if (!ALLOWED_TYPES.includes(ext)) {
      setError('不支持的文件格式，请上传 PDF / DOCX / PNG / JPG')
      return
    }
    if (file.size > MAX_SIZE) {
      setError('文件大小不能超过 20MB')
      return
    }

    setUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('project_id', String(projectId))
      await apiClient.upload('/assets/upload', formData)
      onComplete()
    } catch (e) {
      if (e instanceof ApiError) {
        setError(e.message)
      } else {
        setError('上传失败')
      }
    } finally {
      setUploading(false)
    }
  }

  function handleInputChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) handleFile(file)
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFile(file)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>上传文件</DialogTitle>
        </DialogHeader>

        <div
          onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
          onClick={() => inputRef.current?.click()}
          style={{
            border: `2px dashed ${dragOver ? 'var(--color-accent)' : 'var(--color-border)'}`,
            borderRadius: '8px',
            padding: '32px',
            textAlign: 'center',
            cursor: 'pointer',
            backgroundColor: dragOver ? 'var(--color-accent-bg)' : 'transparent',
            transition: 'all 150ms ease',
          }}
        >
          <input
            ref={inputRef}
            type="file"
            accept=".pdf,.docx,.png,.jpg,.jpeg"
            onChange={handleInputChange}
            className="hidden"
            aria-label="选择文件"
          />
          <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
            拖拽文件到此处，或<span style={{ color: 'var(--color-accent)' }}> 点击选择</span>
          </div>
        </div>

        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
          支持 PDF / DOCX / PNG / JPG，≤ 20MB
        </div>

        {error && (
          <div className="text-sm" style={{ color: 'var(--color-error)' }}>{error}</div>
        )}

        {uploading && (
          <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>上传中...</div>
        )}
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 3: 实现 GitRepoDialog**

创建 `frontend/workbench/src/components/intake/GitRepoDialog.tsx`：

```tsx
import { useState } from 'react'
import { apiClient, ApiError } from '../../lib/api-client'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog'

interface GitRepoDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: number
  onComplete: () => void
}

export function GitRepoDialog({ open, onOpenChange, projectId, onComplete }: GitRepoDialogProps) {
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')

  async function handleSubmit() {
    setError('')
    try {
      await apiClient.post('/assets/git', { project_id: projectId, repo_url: url })
      setUrl('')
      onComplete()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '保存失败')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>接入 Git 仓库</DialogTitle>
        </DialogHeader>
        <div>
          <Input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://github.com/username/repo"
          />
        </div>
        {error && <div className="text-sm" style={{ color: 'var(--color-error)' }}>{error}</div>}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={!url.trim()}>保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 4: 实现 NoteDialog**

创建 `frontend/workbench/src/components/intake/NoteDialog.tsx`：

```tsx
import { useState, useEffect } from 'react'
import { apiClient, ApiError } from '../../lib/api-client'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog'

interface NoteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: number
  noteId: number | null  // null=新建, 非null=编辑
  onComplete: () => void
}

export function NoteDialog({ open, onOpenChange, projectId, noteId, onComplete }: NoteDialogProps) {
  const [content, setContent] = useState('')
  const [label, setLabel] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (noteId && open) {
      // 加载已有笔记
      apiClient.get<Asset>(`/assets?project_id=${projectId}`).then((assets) => {
        const note = assets.find((a) => a.id === noteId && a.type === 'note')
        if (note) {
          setContent(note.content || '')
          setLabel(note.label || '')
        }
      })
    } else if (!open) {
      setContent('')
      setLabel('')
    }
  }, [open, noteId])

  async function handleSubmit() {
    setError('')
    try {
      if (noteId) {
        await apiClient.put(`/assets/notes/${noteId}`, { content, label })
      } else {
        await apiClient.post('/assets/notes', { project_id: projectId, content, label })
      }
      onComplete()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '保存失败')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{noteId ? '编辑备注' : '添加备注'}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Input
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="标签（如：求职意向）"
          />
          <Textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="输入备注内容..."
            rows={4}
          />
        </div>
        {error && <div className="text-sm" style={{ color: 'var(--color-error)' }}>{error}</div>}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={!content.trim()}>保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

> 注意：NoteDialog 中引用了 `Asset` 类型。需要从 ProjectDetail 或 api-client 导出，或在 NoteDialog 内部定义。建议在 `api-client.ts` 旁新建 `types.ts`，或直接在 NoteDialog 内定义 `interface Asset { id: number; content?: string | null; label?: string | null }`。

- [ ] **Step 5: 运行全部前端测试**

Run: `cd frontend/workbench && bunx vitest run -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/workbench/src/components/intake/UploadDialog.tsx frontend/workbench/src/components/intake/UploadDialog.test.tsx frontend/workbench/src/components/intake/GitRepoDialog.tsx frontend/workbench/src/components/intake/NoteDialog.tsx
git commit -m "feat(frontend): implement UploadDialog, GitRepoDialog, and NoteDialog components"
```

---

### Task 13: 端到端验证

**Files:** 无新文件

- [ ] **Step 1: 启动 PostgreSQL**

Run: `docker compose up -d postgres && sleep 3`

- [ ] **Step 2: 启动后端**

Run: `cd backend && go run cmd/server/main.go`
Expected: `server starting on :8080`

- [ ] **Step 3: 运行全部后端测试**

Run: `cd backend && go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 4: 启动前端**

Run: `cd frontend/workbench && bun run dev`
Expected: dev server on `:3000`

- [ ] **Step 5: 运行全部前端测试**

Run: `cd frontend/workbench && bunx vitest run -v`
Expected: 全部 PASS

- [ ] **Step 6: 手动验证完整流程**

1. 打开 `http://localhost:3000`，看到项目列表页（空状态）
2. 输入项目名称，按回车，列表刷新显示新项目
3. 点击项目进入详情页
4. 点击"上传文件"，弹窗出现，拖拽/选择文件，上传成功后列表刷新
5. 点击"添加备注"，输入内容保存，列表刷新
6. 点击删除按钮，确认弹窗出现，确认后资料被删除
7. 返回首页，删除项目

- [ ] **Step 7: Commit（如有修复）**

```bash
git add -A && git commit -m "fix(intake): end-to-end verification fixes"
```

---

## 自检清单

| Spec 章节 | 对应 Task |
|---|---|
| 用户隔离（user_id + 中间件） | Task 1, Task 4 |
| 文件存储（FileStorage 接口） | Task 2 |
| PG 测试隔离 | Task 3 |
| ProjectService CRUD + user_id 过滤 | Task 4 |
| AssetService（上传/Git/文本/删除/级联） | Task 5 |
| 10 个 Handler（参数校验 + 错误码映射） | Task 6 |
| 路由注册 | Task 7 |
| Warm Editorial CSS 主题 | Task 8 |
| API Client（X-User-ID + multipart） | Task 9 |
| ProjectList 页面 | Task 10 |
| ProjectDetail + AssetList + DeleteConfirm | Task 11 |
| UploadDialog / GitRepoDialog / NoteDialog | Task 12 |
| 端到端验证 | Task 13 |
| 错误码 1001-1006 | Task 5, Task 6 |
