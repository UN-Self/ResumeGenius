# Workbench TipTap 编辑器 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现已批准的 Workbench TipTap 编辑器、A4 画布、自动保存、顶部/底部工具栏、AI 占位面板，以及后端 Draft GET/PUT 草稿接口。

**Architecture:** 采用 HTML-first 的单向数据流：后端只负责读写 `drafts.html_content`，前端把草稿 HTML 直接喂给 TipTap，编辑后再把 `editor.getHTML()` 原样送回后端。页面结构保持扁平：路由页负责拉取数据和组装状态，布局组件负责视觉结构，TipTap 组件和自动保存 hook 只处理编辑与同步，不引入额外的数据层或转换层。样式集中在一份 editor stylesheet 和少量组件级 React 结构里，避免把 v1 做成过度抽象的框架。

**Tech Stack:** Go 1.24, Gin, GORM, PostgreSQL, React 19, Vite, TypeScript, TipTap, MSW, Vitest, Testing Library, Lucide, Bun

**Read first:** `docs/modules/workbench/contract.md`, `docs/01-product/ui-design-system.md`, `docs/plans/2026-04-28-workbench-tiptap-editor-design.md`, `backend/cmd/server/main_test.go`, `frontend/workbench/src/App.tsx`, `frontend/workbench/src/lib/api-client.ts`, `backend/internal/modules/workbench/routes.go`

**Operational note:** 按用户要求，本轮计划只写实施步骤，不包含 git commit 步骤。

---

### Task 1: 后端草稿接口与服务骨架

**Files:**
- Create: `backend/internal/modules/workbench/testdb_test.go`
- Create: `backend/internal/modules/workbench/handler_test.go`
- Create: `backend/internal/modules/workbench/service.go`
- Create: `backend/internal/modules/workbench/handler.go`
- Modify: `backend/internal/modules/workbench/routes.go`
- Test: `backend/internal/modules/workbench/handler_test.go`
- Test: `backend/cmd/server/main_test.go`

**Step 1: Write the failing tests**

先把 GET/PUT 草稿契约写死，包含成功、404、空内容 4002 三类断言。测试直接连接本地 PostgreSQL，并用每个测试一个事务回滚来隔离数据。

```go
func TestGetDraft_SucceedsAndReturnsHtmlContent(t *testing.T) {
	db := mustOpenTestDB(t)
	defer rollbackTestDB(t, db)

	project := models.Project{Title: "test", Status: "active"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.Draft{ProjectID: project.ID, HTMLContent: "<html><body>hello</body></html>"}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(NewDraftService(db))
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "draft_id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/drafts/1", nil)

	h.GetDraft(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

同时补一个 contract guard，确保路由仍然挂在 `/api/v1/drafts/:draft_id`，而不是旧的模块前缀路径。

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/workbench -run 'Test(GetDraft|UpdateDraft)' -v`

Expected: FAIL，因为 `service.go`、`handler.go` 还没有实现，或者 handler 返回的响应体还不符合契约。

**Step 3: Write the minimal test helper**

把 PostgreSQL 连接和事务回滚辅助函数收进 `testdb_test.go`，只做一件事：让后续 handler 测试可以稳定创建项目和 draft。

```go
func mustOpenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := database.Connect()
	return db
}

func rollbackTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	// 开启事务，测试结束后回滚
}
```

**Step 4: Re-run the same tests**

Run: `cd backend && go test ./internal/modules/workbench -run 'Test(GetDraft|UpdateDraft)' -v`

Expected: 仍然 FAIL，但这次失败应该只剩下行为断言或未实现的 handler/service，而不是测试骨架本身。

---

### Task 2: 后端 DraftService 和 Handler 实现

**Files:**
- Create: `backend/internal/modules/workbench/service.go`
- Create: `backend/internal/modules/workbench/handler.go`
- Modify: `backend/internal/modules/workbench/routes.go`
- Test: `backend/internal/modules/workbench/handler_test.go`

**Step 1: Write the minimal implementation**

先补最小可用实现：`DraftService.GetByID` 负责查草稿，`DraftService.Update` 负责更新 `html_content` 和 `updated_at`；`Handler.GetDraft` 和 `Handler.UpdateDraft` 只做参数解析、空串校验和 response 包封装。

```go
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

func (s *DraftService) Update(id uint, htmlContent string) (*models.Draft, error) {
	var draft models.Draft
	if err := s.db.First(&draft, id).Error; err != nil {
		return nil, err
	}
	draft.HTMLContent = htmlContent
	if err := s.db.Save(&draft).Error; err != nil {
		return nil, err
	}
	return &draft, nil
}
```

`handler.go` 里返回的数据结构要严格对齐设计文档：GET 带 `id / project_id / html_content / updated_at`，PUT 只回 `id / updated_at`。

**Step 2: Run the backend tests**

Run: `cd backend && go test ./internal/modules/workbench -run 'Test(GetDraft|UpdateDraft)' -v`

Expected: PASS。

**Step 3: Tighten error handling**

把 404 映射到 `4001 draft not found`，把空 HTML 映射到 `4002 html content empty`，并确认 `response.Error` 的 HTTP 状态码和错误码都符合 `docs/modules/workbench/contract.md`。

**Step 4: Run the focused backend verification again**

Run: `cd backend && go test ./internal/modules/workbench -run 'Test(GetDraft|UpdateDraft|Empty)' -v`

Expected: PASS，且 `backend/cmd/server/main_test.go` 中的资源路径断言仍然通过。

---

### Task 3: 前端依赖、MSW 和 draft mock

**Files:**
- Modify: `frontend/workbench/package.json`
- Modify: `frontend/workbench/vitest.config.ts`
- Modify: `frontend/workbench/src/main.tsx`
- Create: `frontend/workbench/src/mocks/browser.ts`
- Create: `frontend/workbench/src/mocks/fixtures.ts`
- Create: `frontend/workbench/src/mocks/handlers/drafts.ts`
- Create: `frontend/workbench/tests/setup.ts`
- Create: `frontend/workbench/tests/drafts.test.ts`

**Step 1: Write the failing mock test**

先写一个只验证 mock handler 的测试，确保 GET/PUT 草稿请求在前端独立运行时能被 MSW 拦住，并返回 `sample_draft.html` 对应内容。

```ts
it('returns the sample draft html through the mock handler', async () => {
  const draft = await apiClient.get('/drafts/1')
  expect(draft.html_content).toContain('sample draft')
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/drafts.test.ts`

Expected: FAIL，因为 `msw`、`jsdom`、测试初始化和 browser worker 还没接上。

**Step 3: Install and wire the mock stack**

更新 `package.json`，加上 TipTap、MSW、Testing Library 和 `jsdom`；然后执行 `bun install`，让 lockfile 同步。

建议安装命令：
`cd frontend/workbench && bun add @tiptap/react @tiptap/starter-kit @tiptap/extension-underline @tiptap/extension-text-align msw @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom`

然后在 `src/mocks/handlers/drafts.ts` 里实现 `GET /api/v1/drafts/:draftId` 和 `PUT /api/v1/drafts/:draftId`，`src/mocks/browser.ts` 里做 `setupWorker(...)`，`src/main.tsx` 里按 `import.meta.env.VITE_USE_MOCK === 'true'` 条件启动。

**Step 4: Re-run the mock test**

Run: `cd frontend/workbench && bunx vitest run tests/drafts.test.ts`

Expected: PASS。

---

### Task 4: 工作台页面壳子和 A4 画布布局

**Files:**
- Modify: `frontend/workbench/src/App.tsx`
- Create: `frontend/workbench/src/pages/EditorPage.tsx`
- Create: `frontend/workbench/src/components/editor/WorkbenchLayout.tsx`
- Create: `frontend/workbench/src/components/editor/ActionBar.tsx`
- Create: `frontend/workbench/src/components/editor/A4Canvas.tsx`
- Create: `frontend/workbench/src/components/editor/AiPanelPlaceholder.tsx`
- Create: `frontend/workbench/src/components/editor/EditorSkeleton.tsx`
- Create: `frontend/workbench/src/components/editor/EditorEmptyState.tsx`
- Create: `frontend/workbench/src/styles/editor.css`
- Create: `frontend/workbench/tests/A4Canvas.test.tsx`

**Step 1: Write the failing page/layout test**

先让路由页和 A4 画布的关键样式失败，防止只做出一个静态 div。

```tsx
it('renders the editor page shell with a 210mm by 297mm canvas', () => {
  render(<EditorPage />)
  expect(screen.getByText('AI 助手')).toBeInTheDocument()
  expect(screen.getByTestId('a4-canvas')).toHaveStyle({ width: '210mm', minHeight: '297mm' })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/A4Canvas.test.tsx`

Expected: FAIL，因为页面路由、布局组件和 editor 样式还没落地。

**Step 3: Write the minimal page shell**

把 `App.tsx` 的占位路由替换成真正的编辑页路由，建议对齐 workbench 文档里的 `/projects/:projectId/edit`；`EditorPage.tsx` 负责拉取 draft、组装 loading / empty / error / ready 四种状态，`WorkbenchLayout.tsx` 负责顶部栏、左侧画布、右侧 AI 占位和底部工具栏的网格骨架。

`A4Canvas.tsx` 先只做固定尺寸容器和缩放占位，不要急着接 TipTap。

**Step 4: Re-run the layout test**

Run: `cd frontend/workbench && bunx vitest run tests/A4Canvas.test.tsx`

Expected: PASS。

---

### Task 5: TipTap 编辑器和格式工具栏

**Files:**
- Create: `frontend/workbench/src/components/editor/TipTapEditor.tsx`
- Create: `frontend/workbench/src/components/editor/FormatToolbar.tsx`
- Create: `frontend/workbench/src/components/editor/ToolbarButton.tsx`
- Create: `frontend/workbench/src/components/editor/ToolbarSeparator.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Modify: `frontend/workbench/src/components/editor/A4Canvas.tsx`
- Create: `frontend/workbench/tests/TipTapEditor.test.tsx`
- Create: `frontend/workbench/tests/FormatToolbar.test.tsx`

**Step 1: Write the failing editor/toolbar tests**

先锁定两个最关键行为：初始 HTML 能渲染进编辑器，以及选中文字后点击 Bold 会生成 `<strong>`。

```tsx
it('renders the sample draft html', async () => {
  render(<TipTapEditor content={sampleDraftHtml} />)
  expect(await screen.findByText(/sample draft/i)).toBeInTheDocument()
})

it('applies bold formatting from the toolbar', async () => {
  render(<FormatToolbar editor={editor} />)
  await user.click(screen.getByRole('button', { name: /粗体/i }))
  expect(editor.chain().focus().toggleBold).toHaveBeenCalled()
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx tests/FormatToolbar.test.tsx`

Expected: FAIL，因为 TipTap extensions 和按钮命令还没接好。

**Step 3: Implement the editor and toolbar minimally**

`TipTapEditor.tsx` 只装载设计文档列出的基础扩展：`StarterKit`、`Underline`、`TextAlign.configure({ types: ['heading', 'paragraph'] })`。`FormatToolbar.tsx` 只实现粗体、斜体、下划线、H1/H2/H3、无序/有序列表、左/中/右对齐按钮，按钮状态由 `editor.isActive(...)` 驱动。

**Step 4: Re-run the focused editor tests**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx tests/FormatToolbar.test.tsx`

Expected: PASS。

---

### Task 6: 自动保存、保存状态和页面状态流转

**Files:**
- Create: `frontend/workbench/src/hooks/useAutoSave.ts`
- Create: `frontend/workbench/src/components/editor/SaveIndicator.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Modify: `frontend/workbench/src/components/editor/TipTapEditor.tsx`
- Create: `frontend/workbench/tests/useAutoSave.test.ts`

**Step 1: Write the failing autosave test**

把 2 秒 debounce、失败重试 3 次、卸载 flush 这三条契约一次性写死。

```ts
it('debounces draft saves for 2 seconds', async () => {
  const save = vi.fn().mockResolvedValue(undefined)
  const { result } = renderHook(() => useAutoSave({ save }))

  act(() => result.current.scheduleSave('<p>changed</p>'))
  expect(save).not.toHaveBeenCalled()

  vi.advanceTimersByTime(2000)
  await waitFor(() => expect(save).toHaveBeenCalledTimes(1))
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/useAutoSave.test.ts`

Expected: FAIL，因为 hook、计时器逻辑和状态机都还没有实现。

**Step 3: Implement the autosave hook**

`useAutoSave.ts` 需要暴露最少这些能力：`scheduleSave(html)`, `flush()`, `retry()`, `status`, `lastSavedAt`, `error`。状态流转按设计文档执行：`idle -> saving -> saved -> idle`，失败则按 `1s / 2s / 4s` 退避重试三次，全部失败后进入 `error`。

`SaveIndicator.tsx` 只负责渲染状态，不参与请求；`EditorPage.tsx` 负责把 TipTap 的 `onUpdate` 接到 `scheduleSave`，并在卸载时调用 `flush()`。

**Step 4: Re-run the autosave test**

Run: `cd frontend/workbench && bunx vitest run tests/useAutoSave.test.ts`

Expected: PASS。

---

### Task 7: 空状态、错误状态和可访问性收尾

**Files:**
- Create: `frontend/workbench/src/components/editor/EditorErrorState.tsx`
- Modify: `frontend/workbench/src/components/editor/ActionBar.tsx`
- Modify: `frontend/workbench/src/components/editor/WorkbenchLayout.tsx`
- Modify: `frontend/workbench/src/components/editor/FormatToolbar.tsx`
- Modify: `frontend/workbench/src/components/editor/SaveIndicator.tsx`
- Modify: `frontend/workbench/src/styles/editor.css`
- Create: `frontend/workbench/tests/EditorPage.test.tsx`

**Step 1: Write the failing accessibility/state test**

把加载中、空草稿、加载失败三种状态都补上断言，确保没有内容时不会把空白页面误当作成功。

```tsx
it('shows the empty state when draft html is missing', () => {
  render(<EditorPage draft={null} />)
  expect(screen.getByText('暂无简历内容')).toBeInTheDocument()
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx`

Expected: FAIL，因为状态组件和 aria 语义还不完整。

**Step 3: Add the final UI details**

补齐 `aria-label`、`role="group"`、`aria-live="polite"`、44×44 点击区域、`prefers-reduced-motion` 降级，以及错误状态里的“重新加载”路径。`ActionBar` 显示项目名和保存状态，`EditorErrorState` 显示 404 / 500 文案，`EditorEmptyState` 只负责空草稿提示。

**Step 4: Re-run the state and accessibility test**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx`

Expected: PASS。

---

### Task 8: 全量验证

**Files:**
- Test: `backend/cmd/server/main_test.go`
- Test: `backend/internal/modules/workbench/handler_test.go`
- Test: `frontend/workbench/tests/*.test.tsx`
- Test: `frontend/workbench/tests/setup.ts`

**Step 1: Run the backend suite**

Run: `cd backend && go test ./...`

Expected: PASS，尤其要确认 `backend/cmd/server/main_test.go` 仍然守住资源路径契约。

**Step 2: Run the frontend test suite**

Run: `cd frontend/workbench && bunx vitest run`

Expected: PASS，覆盖 editor、toolbar、autosave、mock、A4 canvas。

**Step 3: Run the frontend production build**

Run: `cd frontend/workbench && bun run build`

Expected: PASS，说明 TipTap、MSW、样式和路由没有把 Vite 打包链路搞坏。

**Step 4: Stop and inspect any remaining diff locally**

如果还有失败，只修当前 slice，不要扩展到版本快照、图片上传、AI 对话或移动端适配，这些都明确不在本次范围内。
