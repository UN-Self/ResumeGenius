# 版本快照前端功能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在编辑器页面实现版本快照的浏览、HTML 预览、手动保存和回退功能。

**Architecture:** 下拉面板（Popover）锚定 ActionBar 按钮，`useVersions` hook 管理状态，编辑区在预览/编辑模式间切换。后端新增一个获取版本 HTML 的端点。

**Tech Stack:** React 18 + TypeScript + Radix Popover + Vitest + MSW + Go/Gin (后端)

---

### Task 1: 后端 — 新增 GetVersion 端点（service + handler + route）

**Files:**
- Modify: `backend/internal/modules/render/service.go` — 新增 `GetByID` 方法
- Modify: `backend/internal/modules/render/handler.go` — 新增 `GetVersion` handler + DTO
- Modify: `backend/internal/modules/render/routes.go` — 注册新路由

**Step 1: 写失败测试 — service 层**

在 `backend/internal/modules/render/service_test.go` 末尾添加：

```go
// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestGetByID_Success(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)

	svc := NewVersionService(db)

	created, err := svc.Create(draft.ID, "测试版本")
	require.NoError(t, err)

	found, err := svc.GetByID(draft.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.HTMLSnapshot, found.HTMLSnapshot)
	require.NotNil(t, found.Label)
	assert.Equal(t, "测试版本", *found.Label)
}

func TestGetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc := NewVersionService(db)

	_, err := svc.GetByID(draft.ID, 99999)
	require.ErrorIs(t, err, ErrVersionNotFound)
}

func TestGetByID_WrongDraft(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	otherDraft := seedDraft(t, db)

	svc := NewVersionService(db)

	ver, err := svc.Create(otherDraft.ID, "other")
	require.NoError(t, err)

	_, err = svc.GetByID(draft.ID, ver.ID)
	require.ErrorIs(t, err, ErrVersionNotFound)
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/render/ -run "TestGetByID" -v`
Expected: FAIL — `svc.GetByID undefined`

**Step 3: 实现 service 方法**

在 `backend/internal/modules/render/service.go` 的 `ListByDraft` 方法后添加：

```go
// GetByID returns a single version by ID, scoped to the given draft.
func (s *VersionService) GetByID(draftID, versionID uint) (*models.Version, error) {
	var version models.Version
	err := s.db.Where("id = ? AND draft_id = ?", versionID, draftID).First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	return &version, nil
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/render/ -run "TestGetByID" -v`
Expected: PASS

**Step 5: 写失败测试 — handler 层**

在 `backend/internal/modules/render/handler_test.go` 的 Version endpoints 区块末尾（`TestHandler_Rollback` 之后）添加：

```go
func TestHandler_GetVersion(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	v1 := seedVersion(t, db, draft.ID, "v1")

	w := doJSON(t, r, "GET",
		fmt.Sprintf("/drafts/%d/versions/%d", draft.ID, v1.ID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "v1", data["label"])
	assert.NotZero(t, data["id"])
	assert.Contains(t, data["html_snapshot"], "test")
}

func TestHandler_GetVersion_NotFound(t *testing.T) {
	r, _, db := setupRouter(t)
	draft := seedDraft(t, db)

	w := doJSON(t, r, "GET",
		fmt.Sprintf("/drafts/%d/versions/99999", draft.ID), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResp(t, w)
	assert.Equal(t, CodeVersionNotFound, resp.Code)
}
```

**Step 6: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/render/ -run "TestHandler_GetVersion" -v`
Expected: FAIL — 404 route not found

**Step 7: 实现 handler + DTO + route**

在 `handler.go` 中添加 DTO（`versionItem` 之后）：

```go
// versionDetail is the response for a single version with HTML snapshot.
type versionDetail struct {
	ID           uint   `json:"id"`
	Label        string `json:"label"`
	HTMLSnapshot string `json:"html_snapshot"`
	CreatedAt    string `json:"created_at"`
}
```

在 `handler.go` 中添加 handler（`ListVersions` 之后）：

```go
// GetVersion handles GET /drafts/:draft_id/versions/:version_id.
func (h *Handler) GetVersion(c *gin.Context) {
	draftID, err := parseUintParam(c, "draft_id")
	if err != nil {
		response.Error(c, CodeDraftNotFound, "invalid draft_id")
		return
	}

	versionID, err := parseUintParam(c, "version_id")
	if err != nil {
		response.Error(c, CodeVersionNotFound, "invalid version_id")
		return
	}

	version, err := h.versionSvc.GetByID(draftID, versionID)
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, CodeVersionNotFound, "version not found")
			return
		}
		response.Error(c, CodeExportFailed, "failed to get version")
		return
	}

	label := ""
	if version.Label != nil {
		label = *version.Label
	}

	response.Success(c, versionDetail{
		ID:           version.ID,
		Label:        label,
		HTMLSnapshot: version.HTMLSnapshot,
		CreatedAt:    version.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
```

在 `routes.go` 的 version 管理区块中添加路由（`ListVersions` 之后）：

```go
		rg.GET("/drafts/:draft_id/versions/:version_id", h.GetVersion)
```

**Step 8: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/render/ -run "TestHandler_GetVersion" -v`
Expected: PASS

**Step 9: 运行全量后端测试确认无回归**

Run: `cd backend && go test ./...`
Expected: PASS

**Step 10: 提交**

```bash
git add backend/internal/modules/render/service.go backend/internal/modules/render/service_test.go backend/internal/modules/render/handler.go backend/internal/modules/render/handler_test.go backend/internal/modules/render/routes.go
git commit -m "feat: add GET /drafts/:draft_id/versions/:version_id endpoint"
```

---

### Task 2: 更新契约文档

**Files:**
- Modify: `docs/modules/render/contract.md` — 新增端点到 API 表和详情

**Step 1: 更新 API 端点表**

在 `docs/modules/render/contract.md` 的 API 端点表中，在 `GET /api/v1/drafts/{draft_id}/versions` 行之后添加：

```markdown
| GET | `/api/v1/drafts/{draft_id}/versions/{version_id}` | 获取单个版本详情（含 html_snapshot） |
```

**Step 2: 添加端点详情**

在 `GET /api/v1/drafts/{draft_id}/versions` 端点详情之后，`POST /api/v1/drafts/{draft_id}/versions` 之前，添加：

```markdown
#### GET /api/v1/drafts/{draft_id}/versions/{version_id}

获取单个版本详情，包含 `html_snapshot` 字段。

```
Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "label": "AI 初始生成",
    "html_snapshot": "<!DOCTYPE html>...",
    "created_at": "2026-04-23T20:00:00Z"
  }
}
```

版本不存在或非本 draft 的版本返回 404，错误码 5004。
```

**Step 3: 提交**

```bash
git add docs/modules/render/contract.md
git commit -m "docs: add GetVersion endpoint to render contract"
```

---

### Task 3: 前端 API 层 + MSW Mock

**Files:**
- Modify: `frontend/workbench/src/lib/api-client.ts` — 新增 `renderApi` 命名空间和类型
- Modify: `frontend/workbench/src/mocks/fixtures.ts` — 新增版本 fixture
- Create: `frontend/workbench/src/mocks/handlers/versions.ts` — 版本相关 MSW handler
- Modify: `frontend/workbench/tests/setup.ts` — 注册新 handler

**Step 1: 写失败测试 — API 客户端**

创建 `frontend/workbench/tests/render-api.test.ts`：

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderApi } from '@/lib/api-client'

// Mock fetch globally
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function mockResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve({ code: 0, data, message: 'ok' }),
  } as Response)
}

beforeEach(() => {
  mockFetch.mockReset()
})

describe('renderApi', () => {
  describe('listVersions', () => {
    it('calls GET /drafts/:draftId/versions', async () => {
      const items = [
        { id: 1, label: 'v1', created_at: '2026-05-06T10:00:00Z' },
      ]
      mockFetch.mockReturnValue(mockResponse({ items, total: 1 }))

      const result = await renderApi.listVersions(42)

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/versions',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(result.items).toHaveLength(1)
      expect(result.items[0].label).toBe('v1')
    })
  })

  describe('getVersion', () => {
    it('calls GET /drafts/:draftId/versions/:versionId', async () => {
      const detail = {
        id: 1,
        label: 'v1',
        html_snapshot: '<html>resume</html>',
        created_at: '2026-05-06T10:00:00Z',
      }
      mockFetch.mockReturnValue(mockResponse(detail))

      const result = await renderApi.getVersion(42, 1)

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/versions/1',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(result.html_snapshot).toBe('<html>resume</html>')
    })
  })

  describe('createVersion', () => {
    it('calls POST /drafts/:draftId/versions with label', async () => {
      const created = { id: 3, label: '校招版', created_at: '2026-05-06T11:00:00Z' }
      mockFetch.mockReturnValue(mockResponse(created))

      const result = await renderApi.createVersion(42, '校招版')

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/versions',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ label: '校招版' }),
        }),
      )
      expect(result.label).toBe('校招版')
    })
  })

  describe('rollback', () => {
    it('calls POST /drafts/:draftId/rollback with version_id', async () => {
      const resp = {
        draft_id: 42,
        updated_at: '2026-05-06T12:00:00Z',
        new_version_id: 5,
        new_version_label: '回退到版本 2026-05-06 10:00:00',
        new_version_created_at: '2026-05-06T12:00:00Z',
      }
      mockFetch.mockReturnValue(mockResponse(resp))

      const result = await renderApi.rollback(42, 1)

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/rollback',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ version_id: 1 }),
        }),
      )
      expect(result.new_version_id).toBe(5)
    })
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/render-api.test.ts`
Expected: FAIL — `renderApi` not found

**Step 3: 实现 API 类型和方法**

在 `frontend/workbench/src/lib/api-client.ts` 末尾添加：

```typescript
// --- Render API ---

export interface Version {
  id: number
  label: string
  created_at: string
}

export interface VersionDetail extends Version {
  html_snapshot: string
}

export const renderApi = {
  listVersions: (draftId: number) =>
    request<{ items: Version[]; total: number }>(`/drafts/${draftId}/versions`),
  getVersion: (draftId: number, versionId: number) =>
    request<VersionDetail>(`/drafts/${draftId}/versions/${versionId}`),
  createVersion: (draftId: number, label: string) =>
    request<Version>(`/drafts/${draftId}/versions`, {
      method: 'POST',
      body: JSON.stringify({ label }),
    }),
  rollback: (draftId: number, versionId: number) =>
    request<{
      draft_id: number
      updated_at: string
      new_version_id: number
      new_version_label: string
      new_version_created_at: string
    }>(`/drafts/${draftId}/rollback`, {
      method: 'POST',
      body: JSON.stringify({ version_id: versionId }),
    }),
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/render-api.test.ts`
Expected: PASS

**Step 5: 添加 fixture 数据**

在 `frontend/workbench/src/mocks/fixtures.ts` 末尾添加：

```typescript
export const sampleVersions = [
  { id: 3, label: 'AI 修改：精简项目经历', created_at: '2026-05-06T10:15:00Z' },
  { id: 2, label: '手动保存', created_at: '2026-05-06T10:10:00Z' },
  { id: 1, label: 'AI 初始生成', created_at: '2026-05-06T10:00:00Z' },
]
```

**Step 6: 创建 MSW handler**

创建 `frontend/workbench/src/mocks/handlers/versions.ts`：

```typescript
import { http } from 'msw'
import { sampleVersions, sampleDraftHtml } from '../fixtures'

export const versionHandlers = [
  // GET /api/v1/drafts/:draftId/versions
  http.get('/api/v1/drafts/:draftId/versions', async () => {
    await new Promise((resolve) => setTimeout(resolve, 100))
    return Response.json({
      code: 0,
      data: { items: sampleVersions, total: sampleVersions.length },
      message: 'ok',
    })
  }),

  // GET /api/v1/drafts/:draftId/versions/:versionId
  http.get('/api/v1/drafts/:draftId/versions/:versionId', async ({ params }) => {
    const versionId = Number(params.versionId)
    const found = sampleVersions.find((v) => v.id === versionId)
    if (!found) {
      return Response.json({ code: 5004, data: null, message: 'version not found' }, { status: 404 })
    }
    return Response.json({
      code: 0,
      data: { ...found, html_snapshot: sampleDraftHtml.trim() },
      message: 'ok',
    })
  }),

  // POST /api/v1/drafts/:draftId/versions
  http.post('/api/v1/drafts/:draftId/versions', async ({ request }) => {
    const body = (await request.json()) as { label?: string }
    return Response.json({
      code: 0,
      data: {
        id: Date.now(),
        label: body.label || '手动保存',
        created_at: new Date().toISOString(),
      },
      message: 'ok',
    })
  }),

  // POST /api/v1/drafts/:draftId/rollback
  http.post('/api/v1/drafts/:draftId/rollback', async ({ params }) => {
    const draftId = Number(params.draftId)
    return Response.json({
      code: 0,
      data: {
        draft_id: draftId,
        updated_at: new Date().toISOString(),
        new_version_id: Date.now(),
        new_version_label: '回退到版本 2026-05-06 10:00:00',
        new_version_created_at: new Date().toISOString(),
      },
      message: 'ok',
    })
  }),
]
```

**Step 7: 注册 handler 到测试 setup**

在 `frontend/workbench/tests/setup.ts` 中导入并注册新 handler：

```typescript
import { versionHandlers } from '../src/mocks/handlers/versions'
```

在 `setupServer(...)` 调用中添加 `...versionHandlers`：

```typescript
export const server = setupServer(
  ...projectHandlers,
  ...draftHandlers,
  ...agentHandlers,
  ...versionHandlers,
)
```

**Step 8: 运行全量测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

**Step 9: 提交**

```bash
git add frontend/workbench/src/lib/api-client.ts frontend/workbench/tests/render-api.test.ts frontend/workbench/src/mocks/fixtures.ts frontend/workbench/src/mocks/handlers/versions.ts frontend/workbench/tests/setup.ts
git commit -m "feat: add renderApi types, methods, tests and MSW handlers"
```

---

### Task 4: useVersions Hook

**Files:**
- Create: `frontend/workbench/src/hooks/useVersions.ts`
- Create: `frontend/workbench/tests/useVersions.test.ts`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/useVersions.test.ts`：

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'

vi.mock('@/lib/api-client', () => ({
  renderApi: {
    listVersions: vi.fn(),
    getVersion: vi.fn(),
    createVersion: vi.fn(),
    rollback: vi.fn(),
  },
}))

import { useVersions } from '@/hooks/useVersions'
import { renderApi } from '@/lib/api-client'

const mockListVersions = vi.mocked(renderApi.listVersions)
const mockGetVersion = vi.mocked(renderApi.getVersion)
const mockCreateVersion = vi.mocked(renderApi.createVersion)
const mockRollback = vi.mocked(renderApi.rollback)

const sampleVersions = [
  { id: 2, label: 'v2', created_at: '2026-05-06T10:10:00Z' },
  { id: 1, label: 'v1', created_at: '2026-05-06T10:00:00Z' },
]

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useVersions', () => {
  it('loads version list on mount', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.versions).toEqual(sampleVersions)
    expect(result.current.previewMode).toBe('editing')
  })

  it('enters preview mode on startPreview', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    const html = '<html>resume v1</html>'
    mockGetVersion.mockResolvedValue({
      ...sampleVersions[1],
      html_snapshot: html,
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(() => result.current.startPreview(sampleVersions[1]))

    expect(result.current.previewMode).toBe('previewing')
    expect(result.current.previewVersion).toEqual(sampleVersions[1])
    expect(result.current.previewHtml).toBe(html)
  })

  it('exits preview mode on exitPreview', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    mockGetVersion.mockResolvedValue({
      ...sampleVersions[1],
      html_snapshot: '<html>v1</html>',
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => result.current.startPreview(sampleVersions[1]))
    expect(result.current.previewMode).toBe('previewing')

    act(() => result.current.exitPreview())

    expect(result.current.previewMode).toBe('editing')
    expect(result.current.previewVersion).toBeNull()
    expect(result.current.previewHtml).toBeNull()
  })

  it('creates snapshot and refreshes list', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    mockCreateVersion.mockResolvedValue({
      id: 3,
      label: '校招版',
      created_at: '2026-05-06T11:00:00Z',
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(() => result.current.createSnapshot('校招版'))

    expect(mockCreateVersion).toHaveBeenCalledWith(1, '校招版')
    expect(mockListVersions).toHaveBeenCalledTimes(2) // mount + after create
  })

  it('rollback returns html and refreshes list', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    const rollbackHtml = '<html>restored</html>'
    mockGetVersion.mockResolvedValue({
      ...sampleVersions[1],
      html_snapshot: rollbackHtml,
    })
    mockRollback.mockResolvedValue({
      draft_id: 1,
      updated_at: '2026-05-06T12:00:00Z',
      new_version_id: 5,
      new_version_label: '回退到版本 2026-05-06 10:00:00',
      new_version_created_at: '2026-05-06T12:00:00Z',
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => result.current.startPreview(sampleVersions[1]))

    const html = await act(() => result.current.rollback())

    expect(html).toBe(rollbackHtml)
    expect(result.current.previewMode).toBe('editing')
    expect(mockRollback).toHaveBeenCalledWith(1, 1)
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/useVersions.test.ts`
Expected: FAIL — `useVersions` not found

**Step 3: 实现 hook**

创建 `frontend/workbench/src/hooks/useVersions.ts`：

```typescript
import { useState, useEffect, useCallback } from 'react'
import { renderApi, type Version } from '@/lib/api-client'

type PreviewMode = 'editing' | 'previewing'

export interface UseVersionsReturn {
  versions: Version[]
  loading: boolean
  previewMode: PreviewMode
  previewVersion: Version | null
  previewHtml: string | null
  refreshList: () => Promise<void>
  startPreview: (version: Version) => Promise<void>
  exitPreview: () => void
  createSnapshot: (label: string) => Promise<void>
  rollback: () => Promise<string>
}

export function useVersions(draftId: number | null): UseVersionsReturn {
  const [versions, setVersions] = useState<Version[]>([])
  const [loading, setLoading] = useState(true)
  const [previewMode, setPreviewMode] = useState<PreviewMode>('editing')
  const [previewVersion, setPreviewVersion] = useState<Version | null>(null)
  const [previewHtml, setPreviewHtml] = useState<string | null>(null)

  const refreshList = useCallback(async () => {
    if (!draftId) return
    try {
      const data = await renderApi.listVersions(draftId)
      setVersions(data.items)
    } finally {
      setLoading(false)
    }
  }, [draftId])

  useEffect(() => {
    refreshList()
  }, [refreshList])

  const startPreview = useCallback(async (version: Version) => {
    if (!draftId) return
    const detail = await renderApi.getVersion(draftId, version.id)
    setPreviewVersion(version)
    setPreviewHtml(detail.html_snapshot)
    setPreviewMode('previewing')
  }, [draftId])

  const exitPreview = useCallback(() => {
    setPreviewMode('editing')
    setPreviewVersion(null)
    setPreviewHtml(null)
  }, [])

  const createSnapshot = useCallback(async (label: string) => {
    if (!draftId) return
    await renderApi.createVersion(draftId, label)
    await refreshList()
  }, [draftId, refreshList])

  const rollback = useCallback(async (): Promise<string> => {
    if (!draftId || !previewVersion) throw new Error('No version to rollback to')
    await renderApi.rollback(draftId, previewVersion.id)
    const html = previewHtml!
    exitPreview()
    await refreshList()
    return html
  }, [draftId, previewVersion, previewHtml, exitPreview, refreshList])

  return {
    versions,
    loading,
    previewMode,
    previewVersion,
    previewHtml,
    refreshList,
    startPreview,
    exitPreview,
    createSnapshot,
    rollback,
  }
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/useVersions.test.ts`
Expected: PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/hooks/useVersions.ts frontend/workbench/tests/useVersions.test.ts
git commit -m "feat: add useVersions hook with list, preview, create and rollback"
```

---

### Task 5: SaveSnapshotDialog 组件

**Files:**
- Create: `frontend/workbench/src/components/version/SaveSnapshotDialog.tsx`
- Create: `frontend/workbench/tests/SaveSnapshotDialog.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/SaveSnapshotDialog.test.tsx`：

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SaveSnapshotDialog } from '@/components/version/SaveSnapshotDialog'

describe('SaveSnapshotDialog', () => {
  it('renders input and buttons when open', () => {
    render(<SaveSnapshotDialog open onClose={vi.fn()} onConfirm={vi.fn()} />)

    expect(screen.getByPlaceholderText(/可选/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '取消' })).toBeInTheDocument()
  })

  it('renders nothing when closed', () => {
    const { container } = render(
      <SaveSnapshotDialog open={false} onClose={vi.fn()} onConfirm={vi.fn()} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('calls onConfirm with input value', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(<SaveSnapshotDialog open onClose={vi.fn()} onConfirm={onConfirm} />)

    await user.type(screen.getByPlaceholderText(/可选/), '校招版')
    await user.click(screen.getByRole('button', { name: '确认' }))

    expect(onConfirm).toHaveBeenCalledWith('校招版')
  })

  it('calls onConfirm with empty string when no input', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(<SaveSnapshotDialog open onClose={vi.fn()} onConfirm={onConfirm} />)

    await user.click(screen.getByRole('button', { name: '确认' }))

    expect(onConfirm).toHaveBeenCalledWith('')
  })

  it('calls onClose on cancel button click', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<SaveSnapshotDialog open onClose={onClose} onConfirm={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '取消' }))

    expect(onClose).toHaveBeenCalled()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/SaveSnapshotDialog.test.tsx`
Expected: FAIL — module not found

**Step 3: 实现组件**

创建 `frontend/workbench/src/components/version/SaveSnapshotDialog.tsx`：

```typescript
import { useState } from 'react'
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface SaveSnapshotDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: (label: string) => void
}

export function SaveSnapshotDialog({ open, onClose, onConfirm }: SaveSnapshotDialogProps) {
  const [label, setLabel] = useState('')

  const handleConfirm = () => {
    onConfirm(label)
    setLabel('')
  }

  const handleClose = () => {
    setLabel('')
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>保存快照</ModalHeader>
      <ModalBody>
        <Input
          placeholder="可选，如「校招版」「精简版」"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleConfirm()
          }}
        />
      </ModalBody>
      <ModalFooter>
        <Button variant="secondary" size="sm" onClick={handleClose}>
          取消
        </Button>
        <Button size="sm" onClick={handleConfirm}>
          确认
        </Button>
      </ModalFooter>
    </Modal>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/SaveSnapshotDialog.test.tsx`
Expected: PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/version/SaveSnapshotDialog.tsx frontend/workbench/tests/SaveSnapshotDialog.test.tsx
git commit -m "feat: add SaveSnapshotDialog component"
```

---

### Task 6: RollbackConfirmDialog 组件

**Files:**
- Create: `frontend/workbench/src/components/version/RollbackConfirmDialog.tsx`
- Create: `frontend/workbench/tests/RollbackConfirmDialog.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/RollbackConfirmDialog.test.tsx`：

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RollbackConfirmDialog } from '@/components/version/RollbackConfirmDialog'

describe('RollbackConfirmDialog', () => {
  it('renders confirmation message when open', () => {
    render(
      <RollbackConfirmDialog open onClose={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByText(/覆盖当前编辑内容/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认回退' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '取消' })).toBeInTheDocument()
  })

  it('renders nothing when closed', () => {
    const { container } = render(
      <RollbackConfirmDialog open={false} onClose={vi.fn()} onConfirm={vi.fn()} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('calls onConfirm on confirm button click', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(<RollbackConfirmDialog open onClose={vi.fn()} onConfirm={onConfirm} />)

    await user.click(screen.getByRole('button', { name: '确认回退' }))

    expect(onConfirm).toHaveBeenCalled()
  })

  it('calls onClose on cancel button click', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<RollbackConfirmDialog open onClose={onClose} onConfirm={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '取消' }))

    expect(onClose).toHaveBeenCalled()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/RollbackConfirmDialog.test.tsx`
Expected: FAIL — module not found

**Step 3: 实现组件**

创建 `frontend/workbench/src/components/version/RollbackConfirmDialog.tsx`：

```typescript
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'

interface RollbackConfirmDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
}

export function RollbackConfirmDialog({ open, onClose, onConfirm }: RollbackConfirmDialogProps) {
  return (
    <Modal open={open} onClose={onClose}>
      <ModalHeader>确认回退</ModalHeader>
      <ModalBody>
        <p className="text-sm text-muted-foreground">
          回退将覆盖当前编辑内容，是否继续？
        </p>
      </ModalBody>
      <ModalFooter>
        <Button variant="secondary" size="sm" onClick={onClose}>
          取消
        </Button>
        <Button variant="danger" size="sm" onClick={onConfirm}>
          确认回退
        </Button>
      </ModalFooter>
    </Modal>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/RollbackConfirmDialog.test.tsx`
Expected: PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/version/RollbackConfirmDialog.tsx frontend/workbench/tests/RollbackConfirmDialog.test.tsx
git commit -m "feat: add RollbackConfirmDialog component"
```

---

### Task 7: VersionPreviewBanner 组件

**Files:**
- Create: `frontend/workbench/src/components/version/VersionPreviewBanner.tsx`
- Create: `frontend/workbench/tests/VersionPreviewBanner.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/VersionPreviewBanner.test.tsx`：

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { VersionPreviewBanner } from '@/components/version/VersionPreviewBanner'

describe('VersionPreviewBanner', () => {
  const version = { id: 3, label: 'AI 修改', created_at: '2026-05-06T10:15:00Z' }

  it('displays version label', () => {
    render(
      <VersionPreviewBanner
        version={version}
        onRollback={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByText(/AI 修改/)).toBeInTheDocument()
    expect(screen.getByText(/正在预览/)).toBeInTheDocument()
  })

  it('calls onRollback when rollback button clicked', async () => {
    const user = userEvent.setup()
    const onRollback = vi.fn()

    render(
      <VersionPreviewBanner
        version={version}
        onRollback={onRollback}
        onClose={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '回退到此版本' }))

    expect(onRollback).toHaveBeenCalled()
  })

  it('calls onClose when close button clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(
      <VersionPreviewBanner
        version={version}
        onRollback={vi.fn()}
        onClose={onClose}
      />,
    )

    await user.click(screen.getByRole('button', { name: '关闭预览' }))

    expect(onClose).toHaveBeenCalled()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/VersionPreviewBanner.test.tsx`
Expected: FAIL — module not found

**Step 3: 实现组件**

创建 `frontend/workbench/src/components/version/VersionPreviewBanner.tsx`：

```typescript
import type { Version } from '@/lib/api-client'
import { Button } from '@/components/ui/button'

interface VersionPreviewBannerProps {
  version: Version
  onRollback: () => void
  onClose: () => void
}

export function VersionPreviewBanner({ version, onRollback, onClose }: VersionPreviewBannerProps) {
  return (
    <div className="flex items-center gap-3 bg-blue-50 border-b border-blue-200 px-4 py-2 text-sm text-blue-800">
      <span>
        正在预览: <strong>{version.label}</strong>
      </span>
      <div className="flex-1" />
      <Button variant="secondary" size="sm" onClick={onRollback}>
        回退到此版本
      </Button>
      <Button variant="ghost" size="sm" onClick={onClose}>
        关闭预览
      </Button>
    </div>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/VersionPreviewBanner.test.tsx`
Expected: PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/version/VersionPreviewBanner.tsx frontend/workbench/tests/VersionPreviewBanner.test.tsx
git commit -m "feat: add VersionPreviewBanner component"
```

---

### Task 8: VersionDropdown 组件

**Files:**
- Create: `frontend/workbench/src/components/version/VersionDropdown.tsx`
- Create: `frontend/workbench/tests/VersionDropdown.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/VersionDropdown.test.tsx`：

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { VersionDropdown } from '@/components/version/VersionDropdown'
import type { Version } from '@/lib/api-client'

const sampleVersions: Version[] = [
  { id: 3, label: 'AI 修改：精简项目经历', created_at: '2026-05-06T10:15:00Z' },
  { id: 2, label: '手动保存', created_at: '2026-05-06T10:10:00Z' },
  { id: 1, label: 'AI 初始生成', created_at: '2026-05-06T10:00:00Z' },
]

describe('VersionDropdown', () => {
  it('shows version history button', () => {
    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: '版本历史' })).toBeInTheDocument()
  })

  it('shows loading state', async () => {
    const user = userEvent.setup()

    render(
      <VersionDropdown
        versions={[]}
        loading={true}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))

    expect(screen.getByText('加载中...')).toBeInTheDocument()
  })

  it('lists versions when opened', async () => {
    const user = userEvent.setup()

    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))

    expect(screen.getByText('AI 修改：精简项目经历')).toBeInTheDocument()
    expect(screen.getByText('手动保存')).toBeInTheDocument()
    expect(screen.getByText('AI 初始生成')).toBeInTheDocument()
  })

  it('calls onPreview when clicking a version', async () => {
    const user = userEvent.setup()
    const onPreview = vi.fn()

    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={onPreview}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))
    await user.click(screen.getByText('手动保存'))

    expect(onPreview).toHaveBeenCalledWith(sampleVersions[1])
  })

  it('shows save snapshot button', async () => {
    const user = userEvent.setup()
    const onSaveSnapshot = vi.fn()

    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={onSaveSnapshot}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))
    await user.click(screen.getByRole('button', { name: '保存快照' }))

    expect(onSaveSnapshot).toHaveBeenCalled()
  })

  it('shows empty state', async () => {
    const user = userEvent.setup()

    render(
      <VersionDropdown
        versions={[]}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))

    expect(screen.getByText('暂无版本记录')).toBeInTheDocument()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/VersionDropdown.test.tsx`
Expected: FAIL — module not found

**Step 3: 实现组件**

创建 `frontend/workbench/src/components/version/VersionDropdown.tsx`：

```typescript
import { useState } from 'react'
import { History, Plus } from 'lucide-react'
import type { Version } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'

interface VersionDropdownProps {
  versions: Version[]
  loading: boolean
  onPreview: (version: Version) => void
  onSaveSnapshot: () => void
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffSec = Math.floor((now - then) / 1000)

  if (diffSec < 60) return '刚刚'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)} 小时前`
  return `${Math.floor(diffSec / 86400)} 天前`
}

export function VersionDropdown({
  versions,
  loading,
  onPreview,
  onSaveSnapshot,
}: VersionDropdownProps) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="secondary" size="sm" type="button">
          <History size={14} className="mr-1" />
          版本历史
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 p-0">
        <div className="max-h-80 overflow-y-auto">
          {loading ? (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">
              加载中...
            </div>
          ) : versions.length === 0 ? (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">
              暂无版本记录
            </div>
          ) : (
            versions.map((v) => (
              <button
                key={v.id}
                type="button"
                className="flex w-full items-center gap-2 px-4 py-2.5 text-left hover:bg-accent transition-colors cursor-pointer"
                onClick={() => {
                  onPreview(v)
                  setOpen(false)
                }}
              >
                <span className="flex-1 text-sm truncate">{v.label}</span>
                <span className="text-xs text-muted-foreground whitespace-nowrap">
                  {formatRelativeTime(v.created_at)}
                </span>
              </button>
            ))
          )}
        </div>
        <div className="border-t border-border p-2">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start"
            onClick={() => {
              onSaveSnapshot()
              setOpen(false)
            }}
          >
            <Plus size={14} className="mr-1" />
            保存快照
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/VersionDropdown.test.tsx`
Expected: PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/version/VersionDropdown.tsx frontend/workbench/tests/VersionDropdown.test.tsx
git commit -m "feat: add VersionDropdown component with Popover"
```

---

### Task 9: 集成到 EditorPage + ActionBar

**Files:**
- Modify: `frontend/workbench/src/components/editor/ActionBar.tsx` — 替换占位按钮为 VersionDropdown
- Modify: `frontend/workbench/src/pages/EditorPage.tsx` — 集成 useVersions hook + 预览模式

**Step 1: 修改 ActionBar — 接收版本相关 props**

将 `ActionBar.tsx` 中的占位「版本历史」按钮替换为接收 `children` prop 来渲染版本组件。

替换 `ActionBarProps` 接口：

```typescript
interface ActionBarProps {
  projectName: string
  saveIndicator?: ReactNode
  draftId: string | null
  exportStatus: ExportStatus
  onExport: () => void
  children?: ReactNode  // 版本历史组件 slot
}
```

替换组件签名和渲染部分：

```typescript
export function ActionBar({
  projectName,
  saveIndicator,
  draftId,
  exportStatus,
  onExport,
  children,
}: ActionBarProps) {
```

将第 47-50 行的占位版本历史按钮替换为：

```typescript
      {/* Version History */}
      {children}
```

**Step 2: 修改 EditorPage — 集成 hook 和组件**

在 `EditorPage.tsx` 中：

1. 新增 imports：

```typescript
import { useVersions } from '@/hooks/useVersions'
import { VersionDropdown } from '@/components/version/VersionDropdown'
import { VersionPreviewBanner } from '@/components/version/VersionPreviewBanner'
import { SaveSnapshotDialog } from '@/components/version/SaveSnapshotDialog'
import { RollbackConfirmDialog } from '@/components/version/RollbackConfirmDialog'
```

2. 在 `EditorPage` 组件内，`const { exportPdf, status: exportStatus } = useExport()` 之后添加：

```typescript
  const {
    versions,
    loading: versionsLoading,
    previewMode,
    previewVersion,
    previewHtml,
    startPreview,
    exitPreview,
    createSnapshot,
    rollback: rollbackVersion,
  } = useVersions(draftId ? Number(draftId) : null)

  const [saveDialogOpen, setSaveDialogOpen] = useState(false)
  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false)

  const handleRollback = async () => {
    try {
      const html = await rollbackVersion()
      editor?.commands.setContent(html)
    } catch (e) {
      console.error('Rollback failed:', e)
    }
    setRollbackDialogOpen(false)
  }
```

3. 修改编辑区渲染逻辑。将 `<div className="flex-1 overflow-auto" onContextMenu={handleContextMenu}>` 内部替换为：

```typescript
          <div className="flex-1 overflow-auto" onContextMenu={handleContextMenu}>
            {previewMode === 'previewing' && previewHtml && previewVersion ? (
              <>
                <VersionPreviewBanner
                  version={previewVersion}
                  onRollback={() => setRollbackDialogOpen(true)}
                  onClose={exitPreview}
                />
                <A4Canvas>
                  <div dangerouslySetInnerHTML={{ __html: previewHtml }} />
                </A4Canvas>
              </>
            ) : (
              <>
                <A4Canvas editor={editor} />
                {editor && (
                  <BubbleMenu
                    editor={editor}
                    options={{
                      placement: 'top',
                      arrow: false,
                    }}
                    shouldShow={({ editor: e }) => {
                      const { from, to } = e.state.selection
                      return from !== to
                    }}
                  >
                    <BubbleToolbar editor={editor} />
                  </BubbleMenu>
                )}
              </>
            )}
          </div>
```

4. 修改 ActionBar，传入版本组件作为 children：

```typescript
          <ActionBar
            projectName={projectTitle}
            saveIndicator={<SaveIndicator status={status} lastSavedAt={lastSavedAt} onRetry={retry} />}
            draftId={draftId}
            exportStatus={exportStatus}
            onExport={handleExport}
          >
            <VersionDropdown
              versions={versions}
              loading={versionsLoading}
              onPreview={startPreview}
              onSaveSnapshot={() => setSaveDialogOpen(true)}
            />
          </ActionBar>
```

5. 在 `</ContextMenu>` 之前添加对话框：

```typescript
      <SaveSnapshotDialog
        open={saveDialogOpen}
        onClose={() => setSaveDialogOpen(false)}
        onConfirm={(label) => {
          createSnapshot(label)
          setSaveDialogOpen(false)
        }}
      />
      <RollbackConfirmDialog
        open={rollbackDialogOpen}
        onClose={() => setRollbackDialogOpen(false)}
        onConfirm={handleRollback}
      />
```

**Step 3: 运行全量测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

**Step 4: 构建检查**

Run: `cd frontend/workbench && bun run build`
Expected: 成功

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/editor/ActionBar.tsx frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat: integrate version snapshot UI into EditorPage"
```

---

### Task 10: 全量测试 + 最终验证

**Step 1: 运行后端全量测试**

Run: `cd backend && go test ./...`
Expected: PASS

**Step 2: 运行前端全量测试**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

**Step 3: 前端构建验证**

Run: `cd frontend/workbench && bun run build`
Expected: 成功，无 TypeScript 错误

**Step 4: 运行后端编译检查**

Run: `cd backend && go build ./cmd/server/...`
Expected: 成功
