# 导出修复 + 智能路由 + 工作区上传 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复导出 PDF 功能、添加项目列表智能路由、在工作区左侧栏添加文件上传按钮。

**Architecture:** 纯前端改动，修复现有 hook bug 并接入 UI，一行路由分支，复用已有 UploadDialog 组件。

**Tech Stack:** React + TypeScript + Vitest + MSW + Testing Library

---

### Task 1: 修复 useExport hook 的两个 bug

**Files:**
- Modify: `frontend/workbench/src/hooks/useExport.ts:63,68`
- Test: `frontend/workbench/tests/useExport.test.ts`

**Step 1: Write the failing test**

创建 `tests/useExport.test.ts`：

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useExport, type ExportStatus } from '@/hooks/useExport'

// Mock fetch globally
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

// Mock URL.createObjectURL and revokeObjectURL
const mockCreateObjectURL = vi.fn(() => 'blob:fake-url')
const mockRevokeObjectURL = vi.fn()
vi.stubGlobal('URL', { ...URL, createObjectURL: mockCreateObjectURL, revokeObjectURL: mockRevokeObjectURL })

// Mock document.createElement to capture click behavior
const mockClick = vi.fn()
vi.spyOn(document, 'createElement').mockReturnValue({
  href: '',
  download: '',
  click: mockClick,
} as unknown as HTMLAnchorElement)

// Mock setTimeout to auto-reset status
vi.useFakeTimers()

describe('useExport', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('polls with correct task ID (template literal interpolation)', async () => {
    // The bug was using single quotes: '/tasks/${taskId}' instead of `/tasks/${taskId}`
    // This test verifies the polling URL contains the actual task ID

    // 1. Export call returns a task
    mockFetch.mockImplementation((url: string) => {
      if (url.includes('/drafts/1/export')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            code: 0,
            data: { task_id: 'abc-123', status: 'pending', progress: 0 },
            message: 'ok',
          }),
        })
      }
      // Polling call - should use actual task ID
      if (url.includes('/tasks/abc-123')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            code: 0,
            data: { task_id: 'abc-123', status: 'completed', progress: 100 },
            message: 'ok',
          }),
        })
      }
      // Download call
      if (url.includes('/tasks/abc-123/file')) {
        return Promise.resolve({
          ok: true,
          blob: () => Promise.resolve(new Blob(['pdf-data'], { type: 'application/pdf' })),
        })
      }
      return Promise.reject(new Error('Unexpected URL: ' + url))
    })

    const { result } = renderHook(() => useExport({ pollInterval: 100, maxPollDuration: 5000 }))

    await act(async () => {
      await result.current.exportPdf(1, '<p>test</p>', 'resume')
    })

    // Verify polling used actual task ID in URL, not literal '${taskId}'
    const pollingCalls = mockFetch.mock.calls.filter(
      ([url]: [string]) => url.includes('/tasks/') && !url.includes('/file')
    )
    expect(pollingCalls.length).toBeGreaterThan(0)
    expect(pollingCalls[0][0]).toContain('abc-123')
    expect(pollingCalls[0][0]).not.toContain('${')
  })

  it('handles failed status correctly (not "faild")', async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes('/drafts/1/export')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            code: 0,
            data: { task_id: 'task-fail', status: 'pending', progress: 0 },
            message: 'ok',
          }),
        })
      }
      if (url.includes('/tasks/task-fail')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            code: 0,
            data: { task_id: 'task-fail', status: 'failed', error: 'render error' },
            message: 'ok',
          }),
        })
      }
      return Promise.reject(new Error('Unexpected URL'))
    })

    const { result } = renderHook(() => useExport({ pollInterval: 100, maxPollDuration: 5000 }))

    await act(async () => {
      try {
        await result.current.exportPdf(1, '<p>test</p>')
      } catch {
        // expected
      }
    })

    expect(result.current.status).toBe('failed')
    expect(result.current.error).toBe('render error')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/useExport.test.ts -v`
Expected: FAIL — polling URL will contain literal `${taskId}` instead of `abc-123`

**Step 3: Fix the two bugs in useExport.ts**

在 `useExport.ts` 中：

Line 63: 将 `'/tasks/${taskId}'` 改为 `` `/tasks/${taskId}` ``
Line 68: 将 `"faild"` 改为 `"failed"`

**Step 4: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/useExport.test.ts -v`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/hooks/useExport.ts frontend/workbench/tests/useExport.test.ts
git commit -m "fix: 修复 useExport 模板字符串和状态拼写错误"
```

---

### Task 2: 接入导出按钮到 ActionBar

**Files:**
- Modify: `frontend/workbench/src/components/editor/ActionBar.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Test: `frontend/workbench/tests/EditorPage.test.tsx` (extend existing)

**Step 1: Write the failing test**

在 `tests/EditorPage.test.tsx` 中新增一个 describe 块：

```typescript
describe('Export button', () => {
  it('renders export button that is not disabled when editor is loaded', async () => {
    server.use(
      http.get('/api/v1/projects/:projectId', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            id: 1,
            title: 'Test Project',
            status: 'active',
            current_draft_id: 1,
            created_at: '2026-04-28T12:00:00Z',
          },
          message: 'ok',
        })
      }),
      http.get('/api/v1/drafts/1', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            id: 1,
            project_id: 1,
            html_content: '<p>Hello</p>',
            updated_at: '2026-04-28T12:00:00Z',
          },
          message: 'ok',
        })
      }),
      http.post('/api/v1/parsing/parse', () => {
        return HttpResponse.json({
          code: 0,
          data: { parsed_contents: [] },
          message: 'ok',
        })
      })
    )

    renderWithRouter()
    await waitFor(() => {
      expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
    })

    const exportBtn = screen.getByText('导出 PDF')
    expect(exportBtn).toBeInTheDocument()
    expect(exportBtn).not.toBeDisabled()
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: FAIL — export button is permanently disabled

**Step 3: Implement ActionBar changes**

修改 `ActionBar.tsx`，新增 props 并接入导出逻辑：

```typescript
import { FileText } from 'lucide-react'
import type { ReactNode } from 'react'
import type { ExportStatus } from '@/hooks/useExport'

interface ActionBarProps {
  projectName: string
  saveIndicator?: ReactNode
  draftId: string | null
  getHtml: () => string
  exportStatus: ExportStatus
  onExport: () => void
}

const EXPORT_LABEL: Record<ExportStatus, string> = {
  idle: '导出 PDF',
  exporting: '导出中...',
  completed: '导出 PDF',
  failed: '导出失败',
}

export function ActionBar({
  projectName,
  saveIndicator,
  draftId,
  getHtml,
  exportStatus,
  onExport,
}: ActionBarProps) {
  return (
    <div className="action-bar">
      <div className="flex items-center gap-2">
        <FileText size={24} className="text-[var(--color-primary)]" />
      </div>
      <div className="h-6 w-px bg-[var(--color-divider)]" />
      <span className="text-base font-medium text-[var(--color-text-main)]">{projectName}</span>
      <div className="flex-1" />
      <div className="flex items-center gap-2">
        {saveIndicator}
      </div>
      <button
        type="button"
        className="px-3 py-1.5 text-sm font-medium text-[var(--color-text-main)] hover:bg-[var(--color-primary-bg)] rounded-md transition-colors cursor-pointer"
      >
        版本历史
      </button>
      <button
        type="button"
        disabled={!draftId || exportStatus === 'exporting'}
        onClick={onExport}
        className="px-3 py-1.5 text-sm font-medium text-[var(--color-text-main)] bg-[var(--color-page-bg)] border border-[var(--color-divider)] rounded-md disabled:cursor-not-allowed disabled:text-[var(--color-text-disabled)] hover:bg-[var(--color-primary-bg)] transition-colors cursor-pointer"
      >
        {EXPORT_LABEL[exportStatus]}
      </button>
    </div>
  )
}
```

**Step 4: Wire up EditorPage**

在 `EditorPage.tsx` 中：

1. 导入 `useExport`
2. 调用 hook: `const { exportPdf, status: exportStatus } = useExport()`
3. 添加 `onExport` handler:
```typescript
const handleExport = () => {
  if (draftId && editor) {
    exportPdf(Number(draftId), editor.getHTML())
  }
}
```
4. 传给 ActionBar:
```tsx
<ActionBar
  projectName={projectTitle}
  saveIndicator={<SaveIndicator status={status} lastSavedAt={lastSavedAt} onRetry={retry} />}
  draftId={draftId}
  getHtml={() => editor?.getHTML() ?? ''}
  exportStatus={exportStatus}
  onExport={handleExport}
/>
```

**Step 5: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: PASS

**Step 6: Commit**

```bash
git add frontend/workbench/src/components/editor/ActionBar.tsx frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/tests/EditorPage.test.tsx
git commit -m "feat: 接入导出 PDF 按钮到编辑器工具栏"
```

---

### Task 3: 智能路由 — 有草稿的项目直接进编辑器

**Files:**
- Modify: `frontend/workbench/src/pages/ProjectList.tsx:112`
- Test: `frontend/workbench/tests/ProjectList.test.tsx`

**Step 1: Write the failing test**

在 `tests/ProjectList.test.tsx` 中新增测试（在已有的 `describe('ProjectList', ...)` 闭合 `})` 之前）：

```typescript
it('navigates to editor when project has current_draft_id', async () => {
  const user = userEvent.setup()
  vi.mocked(intakeApi.listProjects).mockResolvedValue([
    { id: 1, title: '已有草稿的项目', status: 'active', current_draft_id: 5, created_at: '2026-04-28T00:00:00Z' },
  ])

  const { container } = renderWithRouter(<ProjectList />)
  await waitFor(() => {
    expect(screen.getByText('已有草稿的项目')).toBeInTheDocument()
  })

  await user.click(screen.getByText('已有草稿的项目'))

  // Should navigate to /projects/1/edit, not /projects/1
  await waitFor(() => {
    const links = container.querySelectorAll('a, [href]')
    const currentPath = window.location.pathname
    expect(currentPath).toBe('/projects/1/edit')
  })
})
```

注意：这个测试需要 `MemoryRouter` 来检测导航。由于 `ProjectList` 使用 `useNavigate`，需要改为用 `MemoryRouter` 包裹并读取当前路径。更新测试辅助函数：

在文件顶部，将 `renderWithRouter` 改为：

```typescript
function renderWithRouter(ui: React.ReactNode, initialEntry = '/') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      {ui}
    </MemoryRouter>
  )
}
```

同时所有已有测试中用到 `renderWithRouter` 的地方需要传 `initialEntry = '/'`（默认值已设置，无需改动）。

对于导航断言，使用 `window.location.pathname` 在 `MemoryRouter` 下无法直接工作，需要改用 `useLocation` 来验证。更简单的方式是断言路由变化 — 但 `MemoryRouter` 不渲染 `<Routes>`，所以需要检查 navigate 被调用。

最简单的方案：spy on `useNavigate`。

在测试文件中，额外 mock navigate：

```typescript
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})
```

然后断言：

```typescript
it('navigates to editor when project has current_draft_id', async () => {
  const user = userEvent.setup()
  mockNavigate.mockClear()
  vi.mocked(intakeApi.listProjects).mockResolvedValue([
    { id: 1, title: '已有草稿的项目', status: 'active', current_draft_id: 5, created_at: '2026-04-28T00:00:00Z' },
  ])

  renderWithRouter(<ProjectList />)
  await waitFor(() => {
    expect(screen.getByText('已有草稿的项目')).toBeInTheDocument()
  })

  await user.click(screen.getByText('已有草稿的项目'))
  expect(mockNavigate).toHaveBeenCalledWith('/projects/1/edit')
})

it('navigates to project detail when project has no draft', async () => {
  const user = userEvent.setup()
  mockNavigate.mockClear()
  vi.mocked(intakeApi.listProjects).mockResolvedValue([
    { id: 2, title: '新项目', status: 'active', current_draft_id: null, created_at: '2026-04-28T00:00:00Z' },
  ])

  renderWithRouter(<ProjectList />)
  await waitFor(() => {
    expect(screen.getByText('新项目')).toBeInTheDocument()
  })

  await user.click(screen.getByText('新项目'))
  expect(mockNavigate).toHaveBeenCalledWith('/projects/2')
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/ProjectList.test.tsx -v`
Expected: FAIL — navigate called with `/projects/1` not `/projects/1/edit`

**Step 3: Implement smart routing**

在 `ProjectList.tsx` line 112，将：
```typescript
onClick={(id) => navigate(`/projects/${id}`)}
```
改为：
```typescript
onClick={(id) => navigate(
  project.current_draft_id ? `/projects/${id}/edit` : `/projects/${id}`
)}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/ProjectList.test.tsx -v`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/pages/ProjectList.tsx frontend/workbench/tests/ProjectList.test.tsx
git commit -m "feat: 有草稿的项目直接导航到编辑器"
```

---

### Task 4: 工作区左侧栏添加上传按钮

**Files:**
- Modify: `frontend/workbench/src/components/intake/ParsedSidebar.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Test: `frontend/workbench/tests/EditorPage.test.tsx` (extend existing)

**Step 1: Write the failing test**

在 `tests/EditorPage.test.tsx` 中新增：

```typescript
describe('Upload button in sidebar', () => {
  it('renders upload button in left panel', async () => {
    server.use(
      http.get('/api/v1/projects/:projectId', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            id: 1,
            title: 'Test Project',
            status: 'active',
            current_draft_id: 1,
            created_at: '2026-04-28T12:00:00Z',
          },
          message: 'ok',
        })
      }),
      http.get('/api/v1/drafts/1', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            id: 1,
            project_id: 1,
            html_content: '',
            updated_at: '2026-04-28T12:00:00Z',
          },
          message: 'ok',
        })
      }),
      http.post('/api/v1/parsing/parse', () => {
        return HttpResponse.json({
          code: 0,
          data: { parsed_contents: [] },
          message: 'ok',
        })
      })
    )

    renderWithRouter()
    await waitFor(() => {
      expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
    })

    expect(screen.getByText('上传文件')).toBeInTheDocument()
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: FAIL — no "上传文件" text in document

**Step 3: Implement ParsedSidebar changes**

修改 `ParsedSidebar.tsx`：

```typescript
import { useState } from 'react'
import { intakeApi, parsingApi, type ParsedContent } from '@/lib/api-client'
import ParsedItem from './ParsedItem'
import UploadDialog from './UploadDialog'

interface ParsedSidebarProps {
  projectId: number
  contents: ParsedContent[]
  onParsed: (contents: ParsedContent[]) => void
}

export default function ParsedSidebar({ projectId, contents, onParsed }: ParsedSidebarProps) {
  const [uploadOpen, setUploadOpen] = useState(false)

  const handleUpload = async (file: File) => {
    await intakeApi.uploadFile(projectId, file)
    const result = await parsingApi.parseProject(projectId)
    onParsed(result.parsed_contents)
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
          素材
        </h2>
        <button
          onClick={() => setUploadOpen(true)}
          className="text-xs text-[var(--color-primary)] hover:underline cursor-pointer"
        >
          上传文件
        </button>
      </div>
      <h3 className="mb-2 text-xs font-semibold text-[var(--color-text-secondary)]">
        解析结果
      </h3>
      <div className="flex flex-col gap-2">
        {contents.map((c) => (
          <ParsedItem key={c.asset_id} content={c} />
        ))}
      </div>
      <UploadDialog open={uploadOpen} onClose={() => setUploadOpen(false)} onUpload={handleUpload} />
    </div>
  )
}
```

**Step 4: Wire up EditorPage**

在 `EditorPage.tsx` 中修改 ParsedSidebar 调用：

```tsx
<ParsedSidebar
  projectId={pid}
  contents={parsedContents}
  onParsed={setParsedContents}
/>
```

**Step 5: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx -v`
Expected: PASS

**Step 6: Run full test suite**

Run: `cd frontend/workbench && bunx vitest run -v`
Expected: ALL PASS

**Step 7: Commit**

```bash
git add frontend/workbench/src/components/intake/ParsedSidebar.tsx frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/tests/EditorPage.test.tsx
git commit -m "feat: 在工作区左侧栏添加文件上传按钮"
```
