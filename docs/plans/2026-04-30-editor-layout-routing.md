# 编辑器布局重构与路由修复 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复"开始解析"后路由不跳转的 bug，并重构 EditorPage 为三栏可折叠响应式布局。

**Architecture:** ProjectDetail 精简为纯 intake 页面，`handleParse` 完成后 navigate 到 `/projects/:projectId/edit`。EditorPage 重写为 CSS Grid 三栏布局，支持面板折叠，加载时通过 `current_draft_id` 做路由守卫。

**Tech Stack:** React 18 + TypeScript + Tailwind CSS + CSS Grid + TipTap + Vitest + MSW

---

### Task 1: 精简 ProjectDetail 为纯 intake 页面

**Files:**
- Modify: `frontend/workbench/src/pages/ProjectDetail.tsx`
- Test: `frontend/workbench/tests/ProjectDetail.test.tsx`

**Step 1: 写失败测试 — handleParse 调用 navigate**

在 `tests/ProjectDetail.test.tsx` 中新增测试：点击"下一步：开始解析"后，应导航到 `/projects/1/edit`。

```tsx
// tests/ProjectDetail.test.tsx — 在 describe 块内新增
it('navigates to edit page after successful parse', async () => {
  const user = userEvent.setup()
  vi.mocked(intakeApi.getProject)
    .mockResolvedValueOnce(mockProject)           // 初始加载
    .mockResolvedValueOnce({ ...mockProject })     // parse 中检查
  vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

  // 需要新增 mock
  vi.mock('@/lib/api-client', async (importOriginal) => {
    const actual = await importOriginal<typeof import('@/lib/api-client')>()
    return {
      ...actual,
      intakeApi: {
        ...actual.intakeApi,
        getProject: vi.fn()
          .mockResolvedValueOnce(mockProject)
          .mockResolvedValueOnce({ ...mockProject }),
        listAssets: vi.fn().mockResolvedValue(mockAssets),
      },
      parsingApi: {
        parseProject: vi.fn().mockResolvedValue({ parsed_contents: [] }),
      },
      workbenchApi: {
        createDraft: vi.fn().mockResolvedValue({ id: 99, project_id: 1, html_content: '', updated_at: '2026-04-28T00:00:00Z' }),
      },
    }
  })

  // 注意：使用 navigate spy 需要 MemoryRouter 包裹时传入 spy
})
```

由于 navigate 测试在 vitest 中需要 spy，实际测试更简洁的方式是验证 `handleParse` 的 API 调用顺序。先跳过导航断言，在 Step 3 实现后再补。

**Step 2: 运行测试确认当前测试通过**

Run: `cd frontend/workbench && bunx vitest run tests/ProjectDetail.test.tsx -v`
Expected: 现有 3 个测试全部 PASS

**Step 3: 精简 ProjectDetail.tsx**

删除以下内容：
- `Phase` type 和 `phase` state（行 26, 34）
- `parsedContents` 和 `parseError` state（行 35-36）
- `draftId` state（行 51）
- `useEditor` hook（行 54-67）
- `useAutoSave` hook（行 70-77）
- 编辑阶段相关的 `useEffect`（行 100-120）
- `centerSkeleton` 和 `rightSkeleton`（行 309-321）
- 底部三栏布局 JSX（行 323-354），改为只渲染 `intakeContent`

修改 `handleParse`：

```tsx
const handleParse = async () => {
  try {
    setParseLoading(true)
    setParseError('')
    const result = await parsingApi.parseProject(pid)

    // Ensure a draft exists for editing
    const proj = await intakeApi.getProject(pid)
    if (!proj.current_draft_id) {
      await workbenchApi.createDraft(pid)
    }

    navigate(`/projects/${pid}/edit`)
  } catch (err) {
    setParseError(err instanceof ApiError ? err.message : '解析失败')
  } finally {
    setParseLoading(false)
  }
}
```

新增 state：`const [parseLoading, setParseLoading] = useState(false)`

最终 JSX 简化为：

```tsx
return (
  <div className="h-screen bg-[var(--color-page-bg)] overflow-y-auto">
    {intakeContent}
  </div>
)
```

**Step 4: 更新测试**

更新 `tests/ProjectDetail.test.tsx`：
- mock 中新增 `parsingApi` 和 `workbenchApi`
- 新增测试：验证 handleParse 调用了 parsingApi.parseProject 和 workbenchApi.createDraft
- 移除任何与 editing phase 相关的断言（如有）

```tsx
it('calls parse and createDraft then navigates on parse click', async () => {
  const user = userEvent.setup()
  vi.mocked(intakeApi.getProject)
    .mockResolvedValueOnce(mockProject)           // 初始加载
    .mockResolvedValueOnce({ ...mockProject })     // parse 中检查
  vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)
  vi.mocked(parsingApi.parseProject).mockResolvedValue({ parsed_contents: [] })
  vi.mocked(workbenchApi.createDraft).mockResolvedValue({
    id: 99, project_id: 1, html_content: '', updated_at: '2026-04-28T00:00:00Z',
  })

  render(
    <MemoryRouter initialEntries={['/projects/1']}>
      <ProjectDetail />
    </MemoryRouter>,
  )

  await waitFor(() => {
    expect(screen.getByText('前端工程师简历')).toBeInTheDocument()
  })

  const parseBtn = screen.getByText('下一步：开始解析')
  await user.click(parseBtn)

  await waitFor(() => {
    expect(parsingApi.parseProject).toHaveBeenCalledWith(1)
    expect(workbenchApi.createDraft).toHaveBeenCalledWith(1)
  })
})
```

注意：mock 需要在文件顶部扩展 `vi.mock`，新增 `parsingApi` 和 `workbenchApi`。

**Step 5: 运行测试**

Run: `cd frontend/workbench && bunx vitest run tests/ProjectDetail.test.tsx -v`
Expected: 所有测试 PASS

**Step 6: Commit**

```bash
git add frontend/workbench/src/pages/ProjectDetail.tsx frontend/workbench/tests/ProjectDetail.test.tsx
git commit -m "refactor: 精简 ProjectDetail 为纯 intake 页面，handleParse 完成后导航到 /edit"
```

---

### Task 2: 重写 EditorPage — 三栏可折叠布局 + 路由守卫

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Test: `frontend/workbench/tests/EditorPage.test.tsx`
- Delete: `frontend/workbench/src/components/editor/WorkbenchLayout.tsx` (不再使用)

**Step 1: 写失败测试 — 路由守卫**

在 `tests/EditorPage.test.tsx` 中新增测试：当项目没有 `current_draft_id` 时应重定向。

```tsx
it('redirects to project detail when no current_draft_id', async () => {
  server.use(
    http.get('/api/v1/projects/:projectId', () => {
      return HttpResponse.json({
        code: 0,
        data: {
          id: 1,
          title: 'Test Project',
          status: 'active',
          current_draft_id: null,
          created_at: '2026-04-28T12:00:00Z',
        },
        message: 'ok',
      })
    })
  )

  renderWithRouter('/projects/1/edit')

  await waitFor(() => {
    expect(window.location.pathname).toBe('/projects/1')
  })
})
```

注意：MemoryRouter 中需要用 `Navigate` 组件来测试重定向。更实际的做法是在 renderWithRouter 中同时提供 `/projects/:projectId` route：

```tsx
function renderWithRouter(initialEntry = '/projects/1/edit') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/projects/:projectId" element={<div>Project Detail</div>} />
        <Route path="/projects/:projectId/edit" element={<EditorPage />} />
      </Routes>
    </MemoryRouter>
  )
}
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: 新测试 FAIL（当前 EditorPage 显示 empty state 而不是重定向）

**Step 3: 重写 EditorPage.tsx**

```tsx
import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { ActionBar } from '@/components/editor/ActionBar'
import { FormatToolbar } from '@/components/editor/FormatToolbar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { AiPanelPlaceholder } from '@/components/editor/AiPanelPlaceholder'
import { EditorSkeleton } from '@/components/editor/EditorSkeleton'
import ParsedSidebar from '@/components/intake/ParsedSidebar'
import { request, intakeApi, parsingApi, ApiError, type ParsedContent } from '@/lib/api-client'
import { useAutoSave } from '@/hooks/useAutoSave'
import type { Draft } from '@/types/editor'

export default function EditorPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  // Route guard + data loading
  const [draftId, setDraftId] = useState<string | null>(null)
  const [projectTitle, setProjectTitle] = useState('')
  const [parsedContents, setParsedContents] = useState<ParsedContent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Panel collapse state
  const [leftOpen, setLeftOpen] = useState(true)
  const [rightOpen, setRightOpen] = useState(true)

  // TipTap editor
  const editor = useEditor({
    extensions: [
      StarterKit,
      Underline,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'resume-content outline-none',
        style: 'min-height: 261mm;',
      },
    },
  })

  // Auto-save
  const { scheduleSave, flush, retry, status, lastSavedAt } = useAutoSave({
    save: async (html: string) => {
      if (draftId) {
        await request(`/drafts/${draftId}`, { method: 'PUT', body: JSON.stringify({ html_content: html }) })
      }
    },
    saveUrl: draftId ? `/api/v1/drafts/${draftId}` : undefined,
  })

  // Load project (route guard) + draft + parsed contents
  useEffect(() => {
    if (!projectId) return

    let cancelled = false
    setLoading(true)
    setError(null)

    intakeApi.getProject(pid)
      .then((project) => {
        if (cancelled) return

        // Route guard: redirect if no draft
        if (!project.current_draft_id) {
          navigate(`/projects/${pid}`, { replace: true })
          return
        }

        setProjectTitle(project.title)
        setDraftId(String(project.current_draft_id))

        // Load draft content
        return request<Draft>(`/drafts/${project.current_draft_id}`)
      })
      .then((draft) => {
        if (cancelled || !draft) return
        if (editor && draft.html_content) {
          editor.commands.setContent(draft.html_content)
        }

        // Load parsed contents for left panel
        return parsingApi.parseProject(pid).catch(() => {
          // If parsing fails, show empty sidebar (non-blocking)
        })
      })
      .then((result) => {
        if (cancelled || !result) return
        setParsedContents(result.parsed_contents)
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => { cancelled = true }
  }, [projectId])

  // Connect editor to autosave
  useEffect(() => {
    if (!editor) return
    const handleUpdate = () => scheduleSave(editor.getHTML())
    editor.on('update', handleUpdate)
    return () => { editor.off('update', handleUpdate) }
  }, [editor, scheduleSave])

  // Flush on unmount
  useEffect(() => { return () => { flush() } }, [flush])

  if (loading) {
    return (
      <div className="h-screen bg-[var(--color-page-bg)] flex items-center justify-center">
        <p className="text-[var(--color-text-secondary)] text-sm">加载中...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="h-screen bg-[var(--color-page-bg)] flex items-center justify-center">
        <p className="text-red-500 text-sm">{error}</p>
      </div>
    )
  }

  const gridClass = [
    'editor-workspace',
    leftOpen ? 'left-open' : 'left-collapsed',
    rightOpen ? 'right-open' : 'right-collapsed',
  ].join(' ')

  return (
    <div className={gridClass}>
      {/* Left Panel — Parsed Sidebar */}
      <div className="editor-panel-left">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
            素材
          </h2>
          <button
            onClick={() => setLeftOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起左面板"
          >
            ‹
          </button>
        </div>
        <div className="panel-body">
          <ParsedSidebar contents={parsedContents} />
        </div>
      </div>

      {/* Center — A4 Canvas */}
      <div className="editor-panel-center">
        {!leftOpen && (
          <button
            onClick={() => setLeftOpen(true)}
            className="panel-expand-btn panel-expand-left"
            aria-label="展开左面板"
          >
            ›
          </button>
        )}
        {!rightOpen && (
          <button
            onClick={() => setRightOpen(true)}
            className="panel-expand-btn panel-expand-right"
            aria-label="展开右面板"
          >
            ‹
          </button>
        )}
        <div className="flex flex-col h-full">
          <ActionBar
            projectName={projectTitle}
            saveIndicator={<SaveIndicator status={status} lastSavedAt={lastSavedAt} onRetry={retry} />}
          />
          <div className="flex-1 overflow-auto">
            <A4Canvas editor={editor} />
          </div>
          <div className="format-toolbar">
            <FormatToolbar editor={editor} />
          </div>
        </div>
      </div>

      {/* Right Panel — AI */}
      <div className="editor-panel-right">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
            AI 助手
          </h2>
          <button
            onClick={() => setRightOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起右面板"
          >
            ›
          </button>
        </div>
        <div className="panel-body">
          <AiPanelPlaceholder />
        </div>
      </div>
    </div>
  )
}
```

**Step 4: 更新 EditorPage 测试**

更新 `tests/EditorPage.test.tsx`：
- 更新 `renderWithRouter` 添加 ProjectDetail 占位路由
- 更新 empty state 测试 → 现在应重定向
- 新增测试：有 `current_draft_id` 时渲染三栏布局
- 新增测试：加载 draft 内容到编辑器
- 移除旧的 `createAndLoadDraft` 相关测试

```tsx
describe('EditorPage', () => {
  describe('Route guard', () => {
    it('redirects to project detail when no current_draft_id', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 0, data: { id: 1, title: 'Test', status: 'active', current_draft_id: null, created_at: '2026-04-28T12:00:00Z' },
            message: 'ok',
          })
        })
      )
      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByText('Project Detail Page')).toBeInTheDocument()
      })
    })
  })

  describe('Editor loads', () => {
    it('renders editor when project has current_draft_id', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 0, data: { id: 1, title: 'Test', status: 'active', current_draft_id: 1, created_at: '2026-04-28T12:00:00Z' },
            message: 'ok',
          })
        }),
        http.get('/api/v1/drafts/1', () => {
          return HttpResponse.json({
            code: 0, data: { id: 1, project_id: 1, html_content: '<p>Hello</p>', updated_at: '2026-04-28T12:00:00Z' },
            message: 'ok',
          })
        }),
        http.post('/api/v1/parsing/parse', () => {
          return HttpResponse.json({
            code: 0, data: { parsed_contents: [{ asset_id: 1, type: 'text', label: 'test', text: 'content' }] },
            message: 'ok',
          })
        }),
      )
      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })
    })
  })
})
```

**Step 5: 运行测试**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: 所有测试 PASS

**Step 6: 删除不再使用的 WorkbenchLayout.tsx**

检查是否有其他文件引用 `WorkbenchLayout`。如果没有，删除文件：

```bash
rm frontend/workbench/src/components/editor/WorkbenchLayout.tsx
```

**Step 7: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/tests/EditorPage.test.tsx
git rm frontend/workbench/src/components/editor/WorkbenchLayout.tsx  # 如果确认无其他引用
git commit -m "feat: 重写 EditorPage 为三栏可折叠布局，添加路由守卫"
```

---

### Task 3: CSS Grid 三栏布局 + 折叠样式

**Files:**
- Modify: `frontend/workbench/src/styles/editor.css`

**Step 1: 写失败测试 — 折叠状态渲染**

在 `tests/EditorPage.test.tsx` 中新增测试：

```tsx
it('renders collapse buttons for left and right panels', async () => {
  server.use(
    http.get('/api/v1/projects/:projectId', () => HttpResponse.json({
      code: 0, data: { id: 1, title: 'Test', status: 'active', current_draft_id: 1, created_at: '2026-04-28T12:00:00Z' }, message: 'ok',
    })),
    http.get('/api/v1/drafts/1', () => HttpResponse.json({
      code: 0, data: { id: 1, project_id: 1, html_content: '', updated_at: '2026-04-28T12:00:00Z' }, message: 'ok',
    })),
    http.post('/api/v1/parsing/parse', () => HttpResponse.json({
      code: 0, data: { parsed_contents: [] }, message: 'ok',
    })),
  )
  renderWithRouter()
  await waitFor(() => {
    expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
  })
  // 折叠按钮存在
  expect(screen.getByLabelText('收起左面板')).toBeInTheDocument()
  expect(screen.getByLabelText('收起右面板')).toBeInTheDocument()
})

it('collapses left panel on button click', async () => {
  // 同上 mock setup
  const user = userEvent.setup()
  // ... render, wait for canvas ...
  await user.click(screen.getByLabelText('收起左面板'))
  // 验证展开按钮出现
  expect(screen.getByLabelText('展开左面板')).toBeInTheDocument()
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: FAIL（CSS class 或按钮尚未添加）

**Step 3: 添加三栏 Grid CSS**

在 `editor.css` 中新增（替换旧的 `.workbench-layout` 和 `.workspace` 相关样式）：

```css
/* === Editor Workspace — Three-column collapsible grid === */
.editor-workspace {
  display: grid;
  height: 100vh;
  transition: grid-template-columns 300ms ease-in-out;
}

/* Default: both open */
.editor-workspace.left-open.right-open {
  grid-template-columns: 280px 1fr 320px;
}

/* Left collapsed */
.editor-workspace.left-collapsed.right-open {
  grid-template-columns: 0 1fr 320px;
}

/* Right collapsed */
.editor-workspace.left-open.right-collapsed {
  grid-template-columns: 280px 1fr 0;
}

/* Both collapsed */
.editor-workspace.left-collapsed.right-collapsed {
  grid-template-columns: 0 1fr 0;
}

/* Panel base styles */
.editor-panel-left,
.editor-panel-right {
  overflow: hidden;
  background: var(--color-card);
  display: flex;
  flex-direction: column;
}

.editor-panel-left {
  border-right: 1px solid var(--color-divider);
}

.editor-panel-right {
  border-left: 1px solid var(--color-divider);
}

/* Panel header */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-divider);
  flex-shrink: 0;
}

/* Collapse/Expand buttons */
.panel-collapse-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-divider);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  color: var(--color-text-secondary);
  transition: background-color 150ms;
}

.panel-collapse-btn:hover {
  background: var(--color-page-bg);
}

/* Expand buttons (floating in center panel) */
.panel-expand-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 24px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-card);
  border: 1px solid var(--color-divider);
  cursor: pointer;
  font-size: 14px;
  color: var(--color-text-secondary);
  z-index: 10;
  transition: background-color 150ms;
}

.panel-expand-btn:hover {
  background: var(--color-page-bg);
}

.panel-expand-left {
  left: 8px;
  border-radius: 0 6px 6px 0;
}

.panel-expand-right {
  right: 8px;
  border-radius: 6px 0 0 6px;
}

/* Panel body */
.panel-body {
  flex: 1;
  overflow-y: auto;
}

/* Center panel */
.editor-panel-center {
  position: relative;
  background: var(--color-page-bg);
}

/* Responsive: fold right panel at < 1440px, both at < 1280px */
@media (max-width: 1439px) {
  .editor-workspace.left-open.right-open {
    grid-template-columns: 280px 1fr 0;
  }
}

@media (max-width: 1279px) {
  .editor-workspace.left-open.right-open {
    grid-template-columns: 0 1fr 0;
  }
}
```

**Step 4: 运行测试**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: 所有测试 PASS

**Step 5: 清理旧 CSS**

从 `editor.css` 中移除不再使用的样式：
- `.workbench-layout` 及其子规则（`.action-bar`, `.canvas-area`, `.ai-panel` 在 workbench-layout context 下的）
- `.workspace` 及 `.phase-intake`, `.phase-parsing`, `.phase-editing` 相关规则
- `.workspace .panel-left`, `.workspace .panel-center`, `.workspace .panel-right` 规则

保留以下仍需要的样式：
- `.a4-canvas`（可能需要检查是否被 Tailwind class 替代）
- `.format-toolbar`
- `.skeleton-line`, `.skeleton`, shimmer 动画
- `.empty-state`
- `.ProseMirror` 排版样式
- Design tokens (`:root` 变量)
- `@media (prefers-reduced-motion)` 无障碍规则

**Step 6: Commit**

```bash
git add frontend/workbench/src/styles/editor.css
git commit -m "style: 添加三栏可折叠 CSS Grid 布局，支持响应式断点"
```

---

### Task 4: 响应式断点 + 画布居中优化

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx` (初始化面板状态基于窗口宽度)
- Test: `frontend/workbench/tests/EditorPage.test.tsx`

**Step 1: 写失败测试 — 默认面板状态**

```tsx
it('initializes panel state based on window width', async () => {
  // Mock window.innerWidth
  const originalWidth = window.innerWidth
  window.innerWidth = 1200  // < 1280px → both collapsed

  // ... 同上 mock setup + render ...

  await waitFor(() => {
    expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
  })

  // 宽度 < 1280 时，两个展开按钮应该可见（面板默认折叠）
  // 这取决于 useEffect 是否在 mount 时读取 window.innerWidth
  expect(screen.getByLabelText('展开左面板')).toBeInTheDocument()
  expect(screen.getByLabelText('展开右面板')).toBeInTheDocument()

  window.innerWidth = originalWidth
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`

**Step 3: 实现基于窗口宽度的默认面板状态**

在 EditorPage 中，将面板 state 初始化改为：

```tsx
function getDefaultPanelState(): { left: boolean; right: boolean } {
  const w = window.innerWidth
  if (w >= 1440) return { left: true, right: true }
  if (w >= 1280) return { left: true, right: false }
  return { left: false, right: false }
}

// 在组件中：
const defaults = getDefaultPanelState()
const [leftOpen, setLeftOpen] = useState(defaults.left)
const [rightOpen, setRightOpen] = useState(defaults.right)
```

**Step 4: 运行测试**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/tests/EditorPage.test.tsx
git commit -m "feat: 面板默认状态根据窗口宽度初始化"
```

---

### Task 5: 集成验证 + 全量测试

**Files:** 无新文件

**Step 1: 运行全量前端测试**

Run: `cd frontend/workbench && bunx vitest run -v`
Expected: 所有测试 PASS，无 regressions

**Step 2: 构建检查**

Run: `cd frontend/workbench && bun run build`
Expected: 构建成功，无 TypeScript 错误

**Step 3: 最终 Commit（如有修复）**

```bash
git add -A
git commit -m "test: 确保全量测试通过，集成验证"
```
