# MSW Mock 补齐 + 空白草稿创建 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 补齐 MSW browser worker 的 project/draft mock 覆盖，后端新增 POST /drafts 接口，前端空状态添加"新建草稿"按钮。

**Architecture:** 后端 workbench 模块新增 Create 方法（TDD），前端 api-client 新增 workbenchApi，EditorEmptyState 添加按钮，EditorPage 处理创建回调。MSW browser worker 注册所有 workbench 相关 handlers。

**Tech Stack:** Go + Gin + GORM (backend), TypeScript + MSW + Vitest + React (frontend)

---

## Task 0: 契约同步 — 更新 workbench contract.md

**Files:**
- Modify: `docs/modules/workbench/contract.md`

**前置条件：** 已在设计阶段完成（已同步到 contract.md）。

**确认内容：**
1. §4 API 端点表新增 `POST /api/v1/drafts` 行
2. §4 端点详情新增 `POST /api/v1/drafts` 的 Request/Response 示例（200/404/409）
3. §8 错误码表新增 `4003`（项目不存在）和 `4004`（项目已有草稿）

**Step 1: 确认 contract.md 已更新**

契约文档已在设计阶段同步完成。验证以下内容存在：

- `POST /api/v1/drafts` 端点定义
- 4003 / 4004 错误码

---

## Task 1: 后端 — 新增 Create Draft Service 方法（TDD 红灯）

**Files:**
- Modify: `backend/internal/modules/workbench/service.go`
- Test: `backend/internal/modules/workbench/service_test.go` (create)

**Step 1: 写失败的 Service 测试**

创建 `backend/internal/modules/workbench/service_test.go`：

```go
package workbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func TestCreate_Succeeds(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	require.NoError(t, tx.Create(&project).Error)

	svc := NewDraftService(tx)
	draft, err := svc.Create(project.ID)
	require.NoError(t, err)
	assert.NotZero(t, draft.ID)
	assert.Equal(t, project.ID, draft.ProjectID)
	assert.Equal(t, "", draft.HTMLContent)

	// Verify project.current_draft_id is updated
	var updated models.Project
	require.NoError(t, tx.First(&updated, project.ID).Error)
	assert.Equal(t, draft.ID, *updated.CurrentDraftID)
}

func TestCreate_ReturnsErrProjectNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	svc := NewDraftService(tx)
	_, err := svc.Create(99999)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestCreate_ReturnsErrProjectHasDraft(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	require.NoError(t, tx.Create(&project).Error)

	existingDraft := models.Draft{ProjectID: project.ID, HTMLContent: "<p>existing</p>"}
	require.NoError(t, tx.Create(&existingDraft).Error)
	tx.Model(&project).Update("current_draft_id", existingDraft.ID)

	svc := NewDraftService(tx)
	_, err := svc.Create(project.ID)
	assert.ErrorIs(t, err, ErrProjectHasDraft)
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/workbench/... -run TestCreate -v`
Expected: 编译失败，`ErrProjectNotFound` / `ErrProjectHasDraft` / `Create` 方法不存在

**Step 3: Commit 红灯测试**

```bash
git add backend/internal/modules/workbench/service_test.go
git commit -m "test: add Create draft service failing tests"
```

---

## Task 2: 后端 — 实现 Create Draft Service 方法（TDD 绿灯）

**Files:**
- Modify: `backend/internal/modules/workbench/service.go`

**Step 1: 添加错误定义和方法实现**

在 `service.go` 的 `var` block 中添加两个新错误：

```go
var (
	ErrDraftNotFound    = errors.New("draft not found")
	ErrHTMLContentEmpty = errors.New("html content empty")
	ErrProjectNotFound  = errors.New("project not found")
	ErrProjectHasDraft  = errors.New("project already has a current draft")
)
```

在文件末尾添加 `Create` 方法：

```go
// Create creates a new empty draft for the given project and sets it as the current draft.
// Returns ErrProjectNotFound if the project doesn't exist.
// Returns ErrProjectHasDraft if the project already has a current draft.
func (s *DraftService) Create(projectID uint) (*models.Draft, error) {
	var project models.Project
	if err := s.db.First(&project, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	if project.CurrentDraftID != nil {
		return nil, ErrProjectHasDraft
	}

	draft := models.Draft{
		ProjectID:   projectID,
		HTMLContent: "",
	}
	if err := s.db.Create(&draft).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&project).Update("current_draft_id", draft.ID).Error; err != nil {
		return nil, err
	}

	return &draft, nil
}
```

**Step 2: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/workbench/... -run TestCreate -v`
Expected: 3 tests PASS

**Step 3: 运行全部测试确保无回归**

Run: `cd backend && go test ./internal/modules/workbench/... -v`
Expected: 全部 PASS

**Step 4: Commit**

```bash
git add backend/internal/modules/workbench/service.go
git commit -m "feat(workbench): implement Create draft service method"
```

---

## Task 3: 后端 — 新增 CreateDraft Handler + 路由（TDD）

**Files:**
- Modify: `backend/internal/modules/workbench/handler.go`
- Modify: `backend/internal/modules/workbench/routes.go`
- Modify: `backend/internal/modules/workbench/handler_test.go`

**Step 1: 写失败的 Handler 测试**

在 `handler_test.go` 末尾添加：

```go
func TestCreateDraft_SucceedsAndReturnsDraft(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(`{"project_id": `+strconv.FormatUint(uint64(project.ID), 10)+`}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["project_id"].(float64) != float64(project.ID) {
		t.Fatalf("expected project_id %d, got %v", project.ID, data["project_id"])
	}
	if data["html_content"].(string) != "" {
		t.Fatalf("expected empty html_content, got %v", data["html_content"])
	}
	if data["id"] == nil {
		t.Fatalf("expected id to be set")
	}
	if data["updated_at"] == nil {
		t.Fatalf("expected updated_at to be set")
	}

	// Verify project.current_draft_id updated
	var updated models.Project
	if err := tx.First(&updated, project.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.CurrentDraftID == nil {
		t.Fatal("expected project current_draft_id to be set")
	}
	if *updated.CurrentDraftID != uint(data["id"].(float64)) {
		t.Fatalf("expected current_draft_id %d, got %d", uint(data["id"].(float64)), *updated.CurrentDraftID)
	}
}

func TestCreateDraft_Returns404WhenProjectNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(`{"project_id": 99999}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["code"].(float64) != 4003 {
		t.Fatalf("expected code 4003, got %v", resp["code"])
	}
}

func TestCreateDraft_Returns409WhenProjectHasDraft(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<p>existing</p>"}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	tx.Model(&project).Update("current_draft_id", draft.ID)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(`{"project_id": `+strconv.FormatUint(uint64(project.ID), 10)+`}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["code"].(float64) != 4004 {
		t.Fatalf("expected code 4004, got %v", resp["code"])
	}
}

func TestCreateDraft_Returns400WhenRequestBodyInvalid(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	h := NewHandler(NewDraftService(tx))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(`invalid`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDraft(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["code"].(float64) != 40000 {
		t.Fatalf("expected code 40000, got %v", resp["code"])
	}
}

func TestCreateDraft_RouteMountedCorrectly(t *testing.T) {
	db := mustOpenTestDB(t)
	tx := rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterRoutes(v1, tx)

	body := `{"project_id": ` + strconv.FormatUint(uint64(project.ID), 10) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "POST /api/v1/drafts should be mounted")
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/modules/workbench/... -run TestCreateDraft -v`
Expected: 编译失败，`CreateDraft` handler 不存在

**Step 3: 实现 Handler 和路由**

在 `handler.go` 添加 request/response 结构体和 handler 方法：

```go
// CreateDraftRequest represents the request body for POST /drafts
type CreateDraftRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
}

// CreateDraft handles POST /drafts
func (h *Handler) CreateDraft(c *gin.Context) {
	var req CreateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "invalid request body")
		return
	}

	draft, err := h.service.Create(req.ProjectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			response.ErrorWithStatus(c, http.StatusNotFound, 4003, "project not found")
			return
		}
		if errors.Is(err, ErrProjectHasDraft) {
			response.ErrorWithStatus(c, http.StatusConflict, 4004, "project already has a current draft")
			return
		}
		response.Error(c, 50000, "internal server error")
		return
	}

	response.Success(c, GetDraftResponse{
		ID:          draft.ID,
		ProjectID:   draft.ProjectID,
		HTMLContent: draft.HTMLContent,
		UpdatedAt:   draft.UpdatedAt.UTC().Format(time.RFC3339),
	})
}
```

在 `routes.go` 添加路由：

```go
rg.POST("/drafts", handler.CreateDraft)
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/modules/workbench/... -v`
Expected: 全部 PASS（包括新增的 5 个 CreateDraft 测试 + 原有的 10 个测试）

**Step 5: Commit**

```bash
git add backend/internal/modules/workbench/handler.go backend/internal/modules/workbench/routes.go backend/internal/modules/workbench/handler_test.go
git commit -m "feat(workbench): add POST /drafts endpoint with handler and tests"
```

---

## Task 4: 前端 — api-client 添加 workbenchApi + workbenchApi 测试

**Files:**
- Modify: `frontend/workbench/src/lib/api-client.ts`
- Modify: `frontend/workbench/tests/api-client.test.ts`

**Step 1: 写失败的 workbenchApi 测试**

在 `tests/api-client.test.ts` 末尾的 `describe('apiClient', ...)` 闭合括号前添加：

```typescript
  it('workbenchApi.createDraft calls POST /drafts', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({
        code: 0,
        data: { id: 2, project_id: 1, html_content: '', updated_at: '2026-04-29T12:00:00Z' },
      }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await workbenchApi.createDraft(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/drafts', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.id).toBe(2)
    expect(result.project_id).toBe(1)
    expect(result.html_content).toBe('')
  })

  it('workbenchApi.getDraft calls GET /drafts/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({
        code: 0,
        data: { id: 1, project_id: 1, html_content: '<p>hello</p>', updated_at: '2026-04-29T12:00:00Z' },
      }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await workbenchApi.getDraft(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/drafts/1', expect.objectContaining({
      credentials: 'include',
    }))
    expect(result.html_content).toBe('<p>hello</p>')
  })
```

同时在文件顶部的 import 中添加 `workbenchApi`：

```typescript
import { intakeApi, authApi, workbenchApi, ApiError } from '@/lib/api-client'
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/api-client.test.ts`
Expected: FAIL — `workbenchApi` is not exported

**Step 3: 在 api-client.ts 末尾添加 workbenchApi**

```typescript
// --- Workbench API ---

export interface Draft {
  id: number
  project_id: number
  html_content: string
  updated_at: string
}

export const workbenchApi = {
  getDraft: (id: number) => request<Draft>(`/drafts/${id}`),
  createDraft: (projectId: number) =>
    request<Draft>('/drafts', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
}
```

注意：`Draft` 类型已经在 `api-client.ts` 中不存在（它在 `types/editor.ts` 中），所以需要在 api-client.ts 中添加。或者，如果 `api-client.ts` 和 `types/editor.ts` 都有 `Draft`，需要消除重复。我们选择在 `api-client.ts` 中定义并导出，然后 `types/editor.ts` 改为从 `api-client.ts` re-export。

实际上 `types/editor.ts` 已经有 `Draft`，而 `api-client.ts` 没有导入它。最简方案：在 `api-client.ts` 中定义 `Draft` 并导出，然后让 `types/editor.ts` 从 `api-client.ts` 重新导出以保持向后兼容。

但为了最小改动，直接在 `api-client.ts` 中添加 `Draft` 接口，`types/editor.ts` 的 `Draft` 保持不变。`EditorPage.tsx` 从 `types/editor` 导入 `Draft`，这在功能上等价。

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/api-client.test.ts`
Expected: 全部 PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/lib/api-client.ts frontend/workbench/tests/api-client.test.ts
git commit -m "feat(workbench): add workbenchApi with createDraft and getDraft"
```

---

## Task 5: 前端 — EditorEmptyState 添加"新建草稿"按钮

**Files:**
- Modify: `frontend/workbench/src/components/editor/EditorEmptyState.tsx`

**Step 1: 修改 EditorEmptyState 组件**

将 `EditorEmptyState.tsx` 改为接受 `onCreateDraft` 回调 prop：

```tsx
import { FileEdit, Plus } from 'lucide-react'

interface EditorEmptyStateProps {
  onCreateDraft?: () => void
  loading?: boolean
}

export function EditorEmptyState({ onCreateDraft, loading }: EditorEmptyStateProps) {
  return (
    <div className="empty-state">
      <FileEdit size={64} className="text-[var(--color-text-disabled)]" />
      <h3 className="text-base font-medium text-[var(--color-text-secondary)]">暂无简历内容</h3>
      <p className="text-xs font-normal text-[var(--color-text-disabled)] max-w-sm">
        开始编辑你的简历，或使用 AI 助手生成初稿
      </p>
      {onCreateDraft && (
        <button
          className="mt-4 inline-flex items-center gap-2 rounded-md bg-[var(--color-primary)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-primary-hover)] disabled:opacity-50"
          onClick={onCreateDraft}
          disabled={loading}
        >
          <Plus size={16} />
          新建草稿
        </button>
      )}
    </div>
  )
}
```

**Step 2: 运行前端构建确认无编译错误**

Run: `cd frontend/workbench && bun run build`
Expected: 编译通过（`onCreateDraft` 是可选 prop，现有调用无需改动）

**Step 3: Commit**

```bash
git add frontend/workbench/src/components/editor/EditorEmptyState.tsx
git commit -m "feat(editor): add create draft button to EditorEmptyState"
```

---

## Task 6: 前端 — EditorPage 接入 createDraft 逻辑

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`

**Step 1: 修改 EditorPage.tsx**

1. 在 import 中添加 `workbenchApi`：
```typescript
import { request, intakeApi, workbenchApi } from '@/lib/api-client'
```

2. 添加 `createAndLoadDraft` 回调（在 `loadProject` 后面）：
```typescript
  const [creating, setCreating] = useState(false)

  const createAndLoadDraft = useCallback(() => {
    if (!projectId) return

    setCreating(true)
    workbenchApi
      .createDraft(Number(projectId))
      .then((draft) => {
        setDraftId(String(draft.id))
      })
      .catch((err) => {
        console.error('Failed to create draft:', err)
        setErrorMessage(err instanceof Error ? err.message : 'Failed to create draft')
        setState('error')
      })
      .finally(() => {
        setCreating(false)
      })
  }, [projectId])
```

3. 修改 `renderContent` 中的 `empty` case，传递 props：
```typescript
      case 'empty':
        return (
          <A4Canvas>
            <EditorEmptyState onCreateDraft={createAndLoadDraft} loading={creating} />
          </A4Canvas>
        )
```

4. 修改 `loadDraft` 回调：当 `html_content` 为空时（新建草稿），也应该进入 `ready` 状态（因为空白草稿允许编辑）：

```typescript
  const loadDraft = useCallback(() => {
    if (!draftId) return

    setState('loading')
    request<Draft>(`/drafts/${draftId}`)
      .then((data) => {
        if (editor && data.html_content) {
          editor.commands.setContent(data.html_content)
        }
        // 空白草稿也进入 ready 状态，允许用户编辑
        setState('ready')
      })
      .catch((err) => {
        console.error('Failed to load draft:', err)
        setErrorMessage(err instanceof Error ? err.message : 'Failed to load draft')
        setState('error')
      })
  }, [editor, draftId])
```

**Step 2: 运行构建确认无编译错误**

Run: `cd frontend/workbench && bun run build`
Expected: 编译通过

**Step 3: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat(editor): wire up create draft button in EditorPage"
```

---

## Task 7: MSW — Browser Worker 注册 projectHandlers + 新增 POST /drafts mock

**Files:**
- Modify: `frontend/workbench/src/mocks/browser.ts`
- Modify: `frontend/workbench/src/mocks/handlers/drafts.ts`

**Step 1: 修改 browser.ts 注册 projectHandlers**

```typescript
import { setupWorker } from 'msw/browser'
import { projectHandlers } from './handlers/projects'
import { draftHandlers } from './handlers/drafts'

// Export the worker instance for the main app to start
export const worker = setupWorker(...projectHandlers, ...draftHandlers)

// Start the worker in development mode when needed
export async function startMockWorker() {
  if (import.meta.env.DEV && import.meta.env.VITE_USE_MOCK === 'true') {
    await worker.start({
      onUnhandledRequest: 'bypass',
    })
    console.log('[MSW] Mock worker started')
  }
}
```

**Step 2: 在 drafts.ts 添加 POST /drafts mock handler**

在 `draftHandlers` 数组末尾添加：

```typescript
  // POST /api/v1/drafts
  http.post('/api/v1/drafts', async ({ request }) => {
    await request.json()

    // Simulate 200ms delay
    await new Promise((resolve) => setTimeout(resolve, 200))

    return Response.json({
      code: 0,
      data: {
        id: 2,
        project_id: 1,
        html_content: '',
        updated_at: new Date().toISOString(),
      },
      message: 'ok',
    })
  }),
```

**Step 3: 运行前端测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run`
Expected: 全部 PASS

**Step 4: 运行前端构建确认无编译错误**

Run: `cd frontend/workbench && bun run build`
Expected: 编译通过

**Step 5: Commit**

```bash
git add frontend/workbench/src/mocks/browser.ts frontend/workbench/src/mocks/handlers/drafts.ts
git commit -m "feat(mock): register projectHandlers and add POST /drafts mock"
```

---

## Task 8: 端到端验证

**Step 1: 确保后端可运行**

```bash
docker compose up -d postgres
cd backend && go run cmd/server/main.go
```

**Step 2: 测试 POST /api/v1/drafts 接口**

```bash
# 先创建项目（需要 auth token，这里用 dev 模式跳过或手动登录）
curl -X POST http://localhost:8080/api/v1/drafts \
  -H "Content-Type: application/json" \
  -d '{"project_id": 1}'
# 期望: {"code":0,"data":{"id":...,"project_id":1,"html_content":"","updated_at":"..."},"message":"ok"}
```

**Step 3: 测试 MSW mock 模式**

```bash
cd frontend/workbench
VITE_USE_MOCK=true bun run dev
# 浏览器打开 http://localhost:3000/app/projects/1/edit
# 期望: 控制台显示 [MSW] Mock worker started，编辑器加载 sample draft 内容
```

**Step 4: 测试真实模式（无草稿时）**

```bash
cd frontend/workbench
VITE_USE_MOCK=false bun run dev
# 浏览器打开项目页面（current_draft_id 为 null 的项目）
# 期望: 显示"暂无简历内容" + "新建草稿"按钮
# 点击按钮 → 进入空白编辑器（ready 状态）
```
