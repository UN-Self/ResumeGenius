# 模块 workbench — TipTap 可视化编辑器 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 TipTap 所见即所得编辑器 + A4 画布预览 + 自动保存，这是产品核心体验。

**Architecture:** TipTap 编辑器渲染 HTML，A4 画布用 CSS 固定 210mm x 297mm，自动保存用 debounce 2s 调用 PUT API。后端 DraftService 只负责 GET/PUT drafts。

**Tech Stack:** TipTap / @tiptap/starter-kit / React / CSS A4 layout / debounce

**Depends on:** Phase 0 共享基石完成

**契约文档:** `docs/modules/workbench/contract.md`

---

### Task 1: 后端 — DraftService GET/PUT

**Files:**
- Create: `backend/internal/modules/workbench/service.go`
- Create: `backend/internal/modules/workbench/handler.go`
- Create: `backend/internal/modules/workbench/handler_test.go`
- Modify: `backend/internal/modules/workbench/routes.go`

**Step 1: 写失败测试**

```go
// handler_test.go
package workbench

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

func TestGetDraft(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html><body>hello</body></html>"})

	svc := NewDraftService(db)
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("GET", "/drafts/1", nil)

	h.GetDraft(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["html_content"] != "<html><body>hello</body></html>" {
		t.Errorf("unexpected html_content: %v", data["html_content"])
	}
}

func TestPutDraft(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html><body>old</body></html>"})

	svc := NewDraftService(db)
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	body, _ := json.Marshal(map[string]string{"html_content": "<html><body>updated</body></html>"})
	c.Request = httptest.NewRequest("PUT", "/drafts/1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var draft models.Draft
	db.First(&draft, 1)
	if draft.HTMLContent != "<html><body>updated</body></html>" {
		t.Errorf("expected updated content, got: %s", draft.HTMLContent)
	}
}

func TestPutDraftCreateVersion(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html><body>v1</body></html>"})

	svc := NewDraftService(db)
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	body, _ := json.Marshal(map[string]interface{}{
		"html_content":  "<html><body>v2</body></html>",
		"create_version": true,
		"version_label":  "test snapshot",
	})
	c.Request = httptest.NewRequest("PUT", "/drafts/1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateDraft(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&models.Version{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 version, got %d", count)
	}
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/workbench/... -v
# Expected: FAIL
```

**Step 3: 实现 service.go + handler.go**

```go
// service.go
package workbench

import (
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type DraftService struct {
	db *gorm.DB
}

func NewDraftService(db *gorm.DB) *DraftService {
	return &DraftService{db: db}
}

func (s *DraftService) GetByID(id uint) (*models.Draft, error) {
	var draft models.Draft
	if err := s.db.First(&draft, id).Error; err != nil {
		return nil, err
	}
	return &draft, nil
}

func (s *DraftService) Update(id uint, htmlContent string, createVersion bool, versionLabel string) (*models.Draft, error) {
	var draft models.Draft
	if err := s.db.First(&draft, id).Error; err != nil {
		return nil, err
	}
	draft.HTMLContent = htmlContent
	if err := s.db.Save(&draft).Error; err != nil {
		return nil, err
	}

	if createVersion {
		label := versionLabel
		if label == "" {
			defaultLabel := "手动保存"
			label = defaultLabel
		}
		version := models.Version{
			DraftID:      draft.ID,
			HTMLSnapshot: htmlContent,
			Label:        &label,
		}
		s.db.Create(&version)
	}

	return &draft, nil
}
```

```go
// handler.go
package workbench

import (
	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

type Handler struct {
	svc *DraftService
}

func NewHandler(svc *DraftService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetDraft(c *gin.Context) {
	id := c.Param("id")
	draft, err := h.svc.GetByID(/* parse id */ uint(atoi(id)))
	if err != nil {
		response.Error(c, 4001, "草稿不存在")
		return
	}
	response.Success(c, draft)
}

func (h *Handler) UpdateDraft(c *gin.Context) {
	var req struct {
		HTMLContent   string `json:"html_content" binding:"required"`
		CreateVersion bool   `json:"create_version"`
		VersionLabel  string `json:"version_label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 4002, "HTML 内容为空")
		return
	}

	id := c.Param("id")
	draft, err := h.svc.Update(uint(atoi(id)), req.HTMLContent, req.CreateVersion, req.VersionLabel)
	if err != nil {
		response.Error(c, 4001, "草稿不存在")
		return
	}
	response.Success(c, draft)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
```

**Step 4: 更新 routes.go**

```go
package workbench

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	svc := NewDraftService(db)
	h := NewHandler(svc)

	rg.GET("/drafts/:id", h.GetDraft)
	rg.PUT("/drafts/:id", h.UpdateDraft)
}
```

注意：需要 `import "strconv"`。

**Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/workbench/... -v
# Expected: PASS
```

**Step 6: Commit**

```bash
git add backend/internal/modules/workbench/
git commit -m "feat(module-d): implement DraftService GET/PUT with version creation"
```

---

### Task 2: 前端 — TipTap 编辑器 + A4 画布

**Files:**
- Create: `frontend/workbench/src/components/editor/TipTapEditor.tsx`
- Create: `frontend/workbench/src/components/editor/A4Canvas.tsx`
- Create: `frontend/workbench/src/components/editor/Toolbar.tsx`
- Create: `frontend/workbench/src/pages/EditorPage.tsx`
- Create: `frontend/workbench/src/components/editor/TipTapEditor.test.tsx`

**Step 1: 安装 TipTap 依赖**

```bash
cd frontend/workbench
npm install @tiptap/react @tiptap/starter-kit @tiptap/extension-underline
```

**Step 2: 写失败测试**

```tsx
// TipTapEditor.test.tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { TipTapEditor } from './TipTapEditor'

describe('TipTapEditor', () => {
  it('renders the editor area', () => {
    render(<TipTapEditor content="<p>hello</p>" onChange={() => {}} />)
    // TipTap 编辑器应渲染内容
    expect(screen.getByText('hello')).toBeInTheDocument()
  })
})
```

**Step 3: 实现 TipTapEditor**

```tsx
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import { useEffect } from 'react'

interface Props {
  content: string
  onChange: (html: string) => void
}

export function TipTapEditor({ content, onChange }: Props) {
  const editor = useEditor({
    extensions: [StarterKit, Underline],
    content,
    onUpdate: ({ editor }) => onChange(editor.getHTML()),
  })

  useEffect(() => {
    if (editor && !editor.isFocused) {
      editor.commands.setContent(content)
    }
  }, [content])

  return <EditorContent editor={editor} />
}
```

**Step 4: 实现 A4Canvas**

```tsx
import { TipTapEditor } from './TipTapEditor'

interface Props {
  html: string
  onChange: (html: string) => void
}

export function A4Canvas({ html, onChange }: Props) {
  return (
    <div className="a4-canvas">
      <div className="a4-page">
        <TipTapEditor content={html} onChange={onChange} />
      </div>
      <style>{`
        .a4-canvas {
          display: flex;
          justify-content: center;
          padding: 20px;
          background: #f0f0f0;
          overflow: auto;
          height: 100vh;
        }
        .a4-page {
          width: 210mm;
          min-height: 297mm;
          background: white;
          box-shadow: 0 2px 8px rgba(0,0,0,0.15);
          padding: 18mm 20mm;
          font-family: 'Noto Sans SC', sans-serif;
          font-size: 10.5pt;
          line-height: 1.4;
          color: #333;
        }
        .a4-page .tiptap {
          outline: none;
          min-height: 261mm;
        }
      `}</style>
    </div>
  )
}
```

**Step 5: 实现 Toolbar**

```tsx
import { Editor } from '@tiptap/react'

interface Props {
  editor: Editor | null
}

export function Toolbar({ editor }: Props) {
  if (!editor) return null

  const btn = (cmd: () => void, label: string, active = false) => (
    <button
      onClick={cmd}
      className={`px-2 py-1 rounded text-sm ${active ? 'bg-gray-200' : 'hover:bg-gray-100'}`}
    >
      {label}
    </button>
  )

  return (
    <div className="flex items-center gap-1 border-b px-4 py-2 bg-white">
      {btn(() => editor.chain().focus().toggleBold().run(), 'B', editor.isActive('bold'))}
      {btn(() => editor.chain().focus().toggleItalic().run(), 'I', editor.isActive('italic'))}
      {btn(() => editor.chain().focus().toggleUnderline().run(), 'U', editor.isActive('underline'))}
      <span className="mx-1 text-gray-300">|</span>
      {btn(() => editor.chain().focus().toggleHeading({ level: 2 }).run(), 'H2', editor.isActive('heading', { level: 2 }))}
      {btn(() => editor.chain().focus().toggleHeading({ level: 3 }).run(), 'H3', editor.isActive('heading', { level: 3 }))}
      <span className="mx-1 text-gray-300">|</span>
      {btn(() => editor.chain().focus().toggleBulletList().run(), 'UL', editor.isActive('bulletList'))}
      {btn(() => editor.chain().focus().toggleOrderedList().run(), 'OL', editor.isActive('orderedList'))}
      <span className="mx-1 text-gray-300">|</span>
      {btn(() => editor.chain().focus().undo().run(), 'Undo')}
      {btn(() => editor.chain().focus().redo().run(), 'Redo')}
    </div>
  )
}
```

**Step 6: 实现 EditorPage**

```tsx
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import { apiClient } from '../lib/api-client'
import { Toolbar } from '../components/editor/Toolbar'

export function EditorPage() {
  const { projectId } = useParams()
  const [draftId, setDraftId] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const timerRef = useRef<number>()

  const editor = useEditor({
    extensions: [StarterKit, Underline],
    content: '',
    onUpdate: () => {
      if (!editor || !draftId) return
      clearTimeout(timerRef.current)
      timerRef.current = window.setTimeout(async () => {
        setSaving(true)
        try {
          await apiClient.put(`/workbench/drafts/${draftId}`, {
            html_content: editor.getHTML(),
          })
        } finally {
          setSaving(false)
        }
      }, 2000)
    },
  })

  useEffect(() => {
    apiClient.get<{ id: number; html_content: string }>(`/workbench/drafts/${projectId}`)
      .then(d => {
        setDraftId(d.id)
        editor?.commands.setContent(d.html_content)
      })
      .catch(() => {
        // 用 fixture 作为 fallback
        fetch('/fixtures/sample_draft.html')
          .then(r => r.text())
          .then(html => editor?.commands.setContent(html))
      })
  }, [projectId])

  return (
    <div className="flex flex-col h-screen">
      <header className="flex items-center justify-between border-b px-4 py-2 bg-white">
        <span className="font-medium">ResumeGenius</span>
        <span className="text-sm text-gray-500">
          {saving ? '保存中...' : '已保存'}
        </span>
      </header>
      <Toolbar editor={editor} />
      <div className="flex-1 overflow-auto bg-gray-100 flex justify-center p-5">
        <div className="a4-page bg-white shadow-lg" style={{ width: '210mm', minHeight: '297mm', padding: '18mm 20mm' }}>
          <EditorContent editor={editor} />
        </div>
      </div>
    </div>
  )
}
```

**Step 7: 更新 App.tsx**

```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { ProjectList } from './pages/ProjectList'
import { EditorPage } from './pages/EditorPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ProjectList />} />
        <Route path="/editor/:projectId" element={<EditorPage />} />
      </Routes>
    </BrowserRouter>
  )
}
```

**Step 8: Commit**

```bash
git add frontend/workbench/src/
git commit -m "feat(module-d): implement TipTap editor with A4 canvas and auto-save"
```

---

## 验证清单

- [ ] `go test ./internal/modules/workbench/... -v` 全部通过
- [ ] `curl localhost:8080/api/v1/workbench/drafts/1` 返回 HTML 内容
- [ ] `curl -X PUT localhost:8080/api/v1/workbench/drafts/1 -d '{"html_content":"<p>test</p>"}'` 保存成功
- [ ] `curl -X PUT localhost:8080/api/v1/workbench/drafts/1 -d '{"html_content":"<p>v2</p>","create_version":true,"version_label":"snap"}'` 创建版本
- [ ] 前端 `/editor/1` 显示 A4 画布 + TipTap 编辑器
- [ ] 在编辑器中输入文字，2 秒后显示"已保存"
- [ ] 工具栏按钮（粗体/斜体/列表等）正常工作
