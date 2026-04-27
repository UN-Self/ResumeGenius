# 模块 intake — 项目管理与文件上传 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现项目 CRUD 和单文件上传功能，让用户能创建项目并上传 PDF/DOCX 简历文件。

**Architecture:** ProjectService 处理 CRUD 逻辑，AssetService 处理文件上传（multipart → 本地 `./uploads/`），FileStorageService 封装文件存取。前端用 react-router 管理页面路由。

**Tech Stack:** Gin / GORM / multipart form / React / react-router-dom

**Depends on:** Phase 0 共享基石完成

**契约文档:** `docs/modules/intake/contract.md`

---

### Task 1: 后端 — ProjectService CRUD

**Files:**
- Create: `backend/internal/modules/intake/handler.go`
- Create: `backend/internal/modules/intake/service.go`
- Create: `backend/internal/modules/intake/routes.go`
- Create: `backend/internal/modules/intake/handler_test.go`
- Modify: `backend/internal/modules/intake/routes.go`

**Step 1: 写失败测试**

```go
// handler_test.go
package intake

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

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&models.Project{})
	return db
}

func TestCreateProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(map[string]string{"title": "前端工程师简历"})
	c.Request = httptest.NewRequest("POST", "/api/v1/intake/projects", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateProject(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var proj models.Project
	db.First(&proj, 1)
	if proj.Title != "前端工程师简历" {
		t.Errorf("expected title 前端工程师简历, got %s", proj.Title)
	}
}

func TestListProjects(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)
	db.Create(&models.Project{Title: "项目A"})
	db.Create(&models.Project{Title: "项目B"})
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/intake/projects", nil)

	h.ListProjects(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)
	db.Create(&models.Project{Title: "待删除"})
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("DELETE", "/api/v1/intake/projects/1", nil)

	h.DeleteProject(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var count int64
	db.Model(&models.Project{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 projects, got %d", count)
	}
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/intake/... -v
# Expected: FAIL
```

**Step 3: 实现 service.go + handler.go**

```go
// service.go
package intake

import (
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type ProjectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) Create(title string) (*models.Project, error) {
	proj := models.Project{Title: title, Status: "active"}
	if err := s.db.Create(&proj).Error; err != nil {
		return nil, err
	}
	return &proj, nil
}

func (s *ProjectService) List() ([]models.Project, error) {
	var projects []models.Project
	if err := s.db.Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectService) Delete(id uint) error {
	return s.db.Delete(&models.Project{}, id).Error
}
```

```go
// handler.go
package intake

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

type Handler struct {
	projectSvc *ProjectService
}

func NewHandler(projectSvc *ProjectService) *Handler {
	return &Handler{projectSvc: projectSvc}
}

type createProjectReq struct {
	Title string `json:"title" binding:"required"`
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "title is required")
		return
	}
	proj, err := h.projectSvc.Create(req.Title)
	if err != nil {
		response.Error(c, 50000, err.Error())
		return
	}
	response.Success(c, proj)
}

func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.projectSvc.List()
	if err != nil {
		response.Error(c, 50000, err.Error())
		return
	}
	response.Success(c, projects)
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.projectSvc.Delete(uint(id)); err != nil {
		response.Error(c, 50000, err.Error())
		return
	}
	response.Success(c, nil)
}
```

**Step 4: 更新 routes.go**

```go
// routes.go
package intake

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	svc := NewProjectService(db)
	h := NewHandler(svc)

	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects", h.ListProjects)
	rg.DELETE("/projects/:id", h.DeleteProject)
}
```

**Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/intake/... -v
# Expected: PASS
```

**Step 6: Commit**

```bash
git add backend/internal/modules/intake/
git commit -m "feat(module-a): implement project CRUD with tests"
```

---

### Task 2: 后端 — 文件上传

**Files:**
- Create: `backend/internal/modules/intake/upload_handler_test.go`
- Modify: `backend/internal/modules/intake/handler.go`
- Modify: `backend/internal/modules/intake/routes.go`

**Step 1: 写失败测试**

```go
// upload_handler_test.go
package intake

import (
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func createUploadForm(t *testing.T, fieldName, fileName, content string) (*multipart.Writer, *httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(fieldName, fileName)
	part.Write([]byte(content))
	writer.Close()

	c.Request = httptest.NewRequest("POST", "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return writer, w, c
}

func TestUploadFile(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Project{}, &models.Asset{})
	db.Create(&models.Project{Title: "test", Status: "active"})

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "1"), 0755)

	svc := NewProjectService(db)
	assetSvc := NewAssetService(db, tmpDir)
	h := NewHandler(svc)
	h.assetSvc = assetSvc

	_, w, c := createUploadForm(t, "file", "resume.pdf", "fake-pdf-content")
	c.Request.FormValue("project_id")

	// Add project_id to form
	c.Request = httptest.NewRequest("POST", "/upload", c.Request.Body)
	c.Request.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))

	h.UploadFile(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
```

**Step 2: 实现 AssetService + Upload handler**

```go
// 追加到 service.go
type AssetService struct {
	db      *gorm.DB
	uploadDir string
}

func NewAssetService(db *gorm.DB, uploadDir string) *AssetService {
	return &AssetService{db: db, uploadDir: uploadDir}
}

func (s *AssetService) Upload(projectID uint, file *multipart.FileHeader) (*models.Asset, error) {
	ext := filepath.Ext(file.Filename)
	dir := filepath.Join(s.uploadDir, strconv.Itoa(int(projectID)))
	os.MkdirAll(dir, 0755)

	uri := filepath.Join(dir, file.Filename)
	if err := c.SaveUploadedFile(file, uri); err != nil {
		return nil, err
	}

	assetType := "resume_other"
	switch ext {
	case ".pdf": assetType = "resume_pdf"
	case ".docx": assetType = "resume_docx"
	}

	uriStr := uri
	label := file.Filename
	metadata := JSONB{
		"filename":     file.Filename,
		"size_bytes":   file.Size,
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      assetType,
		URI:       &uriStr,
		Label:     &label,
		Metadata:  metadata,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}
```

```go
// 追加到 handler.go
func (h *Handler) UploadFile(c *gin.Context) {
	projectID, err := strconv.Atoi(c.PostForm("project_id"))
	if err != nil {
		response.Error(c, 40000, "project_id is required")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, 1001, "file is required")
		return
	}
	if file.Size > 20*1024*1024 {
		response.Error(c, 1002, "file size exceeds 20MB")
		return
	}

	asset, err := h.assetSvc.Upload(uint(projectID), file)
	if err != nil {
		response.Error(c, 50000, err.Error())
		return
	}
	response.Success(c, asset)
}
```

**Step 3: routes.go 追加上传路由**

```go
rg.POST("/assets/upload", h.UploadFile)
```

**Step 4: Commit**

```bash
git add backend/internal/modules/intake/
git commit -m "feat(module-a): implement file upload to local storage"
```

---

### Task 3: 前端 — 项目列表页 + 创建表单

**Files:**
- Create: `frontend/workbench/src/pages/ProjectList.tsx`
- Create: `frontend/workbench/src/pages/ProjectList.test.tsx`
- Modify: `frontend/workbench/src/App.tsx`

**Step 1: 写失败测试**

```tsx
// ProjectList.test.tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ProjectList } from './ProjectList'

vi.mock('../lib/api-client', () => ({
  apiClient: {
    get: vi.fn().mockResolvedValue([
      { id: 1, title: '项目A', status: 'active', created_at: '2026-04-26T00:00:00Z' },
    ]),
  },
}))

describe('ProjectList', () => {
  it('renders project title', async () => {
    render(<ProjectList />)
    expect(await screen.findByText('项目A')).toBeInTheDocument()
  })
})
```

**Step 2: 实现 ProjectList**

```tsx
import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '../lib/api-client'

interface Project {
  id: number
  title: string
  status: string
  created_at: string
}

export function ProjectList() {
  const [projects, setProjects] = useState<Project[]>([])
  const [title, setTitle] = useState('')

  useEffect(() => {
    apiClient.get<Project[]>('/intake/projects').then(setProjects)
  }, [])

  const handleCreate = async () => {
    const p = await apiClient.post<Project>('/intake/projects', { title })
    setProjects(prev => [p, ...prev])
    setTitle('')
  }

  return (
    <div className="max-w-2xl mx-auto p-6">
      <h1 className="text-2xl font-bold mb-6">我的简历项目</h1>
      <div className="flex gap-2 mb-6">
        <input
          value={title}
          onChange={e => setTitle(e.target.value)}
          placeholder="项目名称"
          className="border rounded px-3 py-2 flex-1"
          onKeyDown={e => e.key === 'Enter' && handleCreate()}
        />
        <button onClick={handleCreate} className="bg-blue-600 text-white px-4 py-2 rounded">
          创建
        </button>
      </div>
      <div className="space-y-3">
        {projects.map(p => (
          <Link
            key={p.id}
            to={`/editor/${p.id}`}
            className="block border rounded p-4 hover:bg-gray-50"
          >
            <div className="font-medium">{p.title}</div>
            <div className="text-sm text-gray-500">{new Date(p.created_at).toLocaleDateString()}</div>
          </Link>
        ))}
      </div>
    </div>
  )
}
```

**Step 3: 更新 App.tsx 路由**

```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { ProjectList } from './pages/ProjectList'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ProjectList />} />
        <Route path="/editor/:projectId" element={<div>Editor (coming soon)</div>} />
      </Routes>
    </BrowserRouter>
  )
}
```

**Step 4: Commit**

```bash
git add frontend/workbench/src/
git commit -m "feat(module-a): add project list page with create form"
```

---

## 验证清单

- [ ] `go test ./internal/modules/intake/... -v` 全部通过
- [ ] `curl -X POST localhost:8080/api/v1/intake/projects -d '{"title":"test"}'` 返回项目
- [ ] `curl localhost:8080/api/v1/intake/projects` 返回项目列表
- [ ] `curl -X DELETE localhost:8080/api/v1/intake/projects/1` 删除项目
- [ ] 文件上传 `curl -F "file=@test.pdf" -F "project_id=1" localhost:8080/api/v1/intake/assets/upload` 返回 asset
- [ ] 前端 `/` 页面显示项目列表，能创建新项目
