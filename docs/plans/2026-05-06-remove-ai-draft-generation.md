# Remove AI Draft Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the `/api/v1/parsing/generate` endpoint and simplify the user flow so users go directly from asset upload/parse into the editor with an auto-created empty draft.

**Architecture:** Delete the generate code path from the parsing module (handler, service, generator). On the frontend, replace the "Generate Draft" button with "Start Editing" that navigates directly to the editor. In EditorPage, auto-create an empty draft via `workbenchApi.createDraft` if none exists, instead of redirecting back.

**Tech Stack:** Go + Gin (backend), React + TypeScript (frontend), Vite, Vitest

---

### Task 1: Remove DraftGeneratorInterface from types.go

**Files:**
- Modify: `backend/internal/modules/parsing/types.go:61-63`
- Test: `backend/internal/modules/parsing/` (existing tests)

- [ ] **Step 1: Remove DraftGeneratorInterface and related sentinel errors from types.go**

Delete lines 61-63 (the `DraftGeneratorInterface` interface) and line 33 (`ErrDraftGeneratorNotConfigured`). Also delete line 25 (`ErrNoGeneratableText`) since it's only used by generate flow.

```go
// DELETE these:
type DraftGeneratorInterface interface {
	GenerateHTML(parsedText string) (string, error)
}
```

```go
// DELETE from var block:
ErrNoGeneratableText           = errors.New("project has no usable text content")
ErrAIGenerateFailed            = errors.New("ai draft generation failed")
ErrDraftGeneratorNotConfigured = errors.New("draft generator is not configured")
```

- [ ] **Step 2: Run backend tests to identify breakage**

Run: `cd backend && go build ./...`
Expected: Compilation errors in handler.go, service.go, generator.go, test files referencing deleted symbols.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/parsing/types.go
git commit -m "refactor(parsing): remove DraftGeneratorInterface and generate-related errors"
```

---

### Task 2: Remove Generate from service.go

**Files:**
- Modify: `backend/internal/modules/parsing/service.go`
- Modify: `backend/internal/modules/parsing/service_test.go`

- [ ] **Step 1: Remove generate-related code from service.go**

Remove these items:

1. The `generator DraftGeneratorInterface` field from `ParsingService` struct (line 21)
2. The `GenerateResult` struct (lines 123-127)
3. `GenerateForUser` method (lines 130-135)
4. `Generate` method (lines 138-202)
5. `loadParsedContentsForGenerate` method (lines 204-244)
6. `loadAssetParsedContentForGenerate` method (lines 246-272)
7. `aggregateParsedContents` function (lines 947-958) — only used by generate
8. Update `NewParsingService` to not pass nil generator (remove the intermediate constructors that take generator):
   - Remove `NewParsingServiceWithGenerator` (lines 32-34)
   - Simplify `NewParsingService` to not call `NewParsingServiceWithGeneratorAndStorage` with nil generator
   - Rename `NewParsingServiceWithGeneratorAndStorage` to `NewParsingServiceWithStorage`, remove `generator` parameter

The resulting constructors should be:

```go
func NewParsingService(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor) *ParsingService {
	return NewParsingServiceWithStorage(db, pdfParser, docxParser, gitExtractor, nil)
}

func NewParsingServiceWithStorage(
	db *gorm.DB,
	pdfParser PdfParser,
	docxParser DocxParser,
	gitExtractor GitExtractor,
	store storage.FileStorage,
) *ParsingService {
	svc := &ParsingService{
		db:           db,
		pdfParser:    pdfParser,
		docxParser:   docxParser,
		gitExtractor: gitExtractor,
		storage:      store,
	}
	svc.projectExists = svc.defaultProjectExists
	svc.listProjectAssets = svc.defaultListProjectAssets
	return svc
}
```

- [ ] **Step 2: Remove generate-related tests from service_test.go**

Delete these test functions entirely:
- `TestGenerateCreatesDraftVersionAndUpdatesCurrentDraftID` (line 856)
- `TestGenerateForUserReturnsProjectNotFoundWhenOwnershipCheckFails` (line 933)
- `TestGenerateAggregatesGitAssetTextWhenExtractorConfigured` (line 955)
- `TestGeneratePrefersPersistedFileContentWithoutInvokingParser` (line 1001)
- `TestGenerateUsesPersistedGitContentWithoutExtractor` (line 1044)
- `TestGenerateFallsBackToParseAndPersistsMissingFileContent` (line 1077)
- `TestGenerateRollsBackWhenVersionCreationFails` (line 1143)
- `TestGenerateDoesNotCreateDirtyDraftWhenGeneratorFails` (line 1213)
- `TestGenerateReturnsDatabaseNotConfiguredWhenDBMissing` (line 1258)
- `TestGenerateReturnsDraftGeneratorNotConfiguredWhenMissing` (line 1267)
- `TestGenerateReturnsNoGeneratableTextWhenParsedContentsHaveNoText` (line 1277)
- `TestAggregateParsedContents` (line 1322)

Also remove the `stubDraftGenerator` struct (around line 54) and update `TestNewParsingServiceWithGeneratorStoresGenerator` (line 83) and `TestNewParsingServiceWithGeneratorAndStorageStoresStorage` (line 93) to use the renamed constructors and not reference generator.

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/modules/parsing/... -v`
Expected: All remaining tests pass. Generate tests are gone.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/modules/parsing/service.go backend/internal/modules/parsing/service_test.go
git commit -m "refactor(parsing): remove Generate methods and tests from service"
```

---

### Task 3: Remove Generate from handler.go and routes.go, delete generator.go

**Files:**
- Modify: `backend/internal/modules/parsing/handler.go`
- Modify: `backend/internal/modules/parsing/routes.go`
- Delete: `backend/internal/modules/parsing/generator.go`
- Delete: `backend/internal/modules/parsing/generator_test.go`
- Modify: `backend/internal/modules/parsing/handler_test.go`

- [ ] **Step 1: Delete generator.go and generator_test.go**

```bash
rm backend/internal/modules/parsing/generator.go
rm backend/internal/modules/parsing/generator_test.go
```

- [ ] **Step 2: Update handler.go**

Remove:
1. Error code constants `CodeAIGenerateFailed` (line 19) and `CodeInvalidAssetData` (line 20) — `CodeInvalidAssetData` is only used for generate errors (ErrNoGeneratableText). Keep `CodeParsePDFFailed`, `CodeParseDOCXFailed`, `CodeProjectNotFound`, `CodeNoUsableAssets`, `CodeGitExtractFailed`.
2. `GenerateForUser` from the `parseService` interface (line 26)
3. The `Generate` handler method (lines 73-93)
4. Error cases for `ErrNoGeneratableText` and `ErrAIGenerateFailed` from `respondParseError` (lines 104-113)

Updated interface:
```go
type parseService interface {
	ParseForUser(userID string, projectID uint) ([]ParsedContent, error)
}
```

Updated error constants:
```go
const (
	CodeParsePDFFailed   = 2001
	CodeParseDOCXFailed  = 2002
	CodeProjectNotFound  = 2003
	CodeNoUsableAssets   = 2004
	CodeGitExtractFailed = 2007
)
```

- [ ] **Step 3: Update routes.go**

Remove the generator instantiation and generate route. Updated:

```go
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	service := NewParsingServiceWithStorage(db, NewPDFParser(), NewDocxParser(), NewGitExtractor(), store)
	handler := NewHandler(service)

	rg.POST("/parsing/parse", handler.Parse)
}
```

- [ ] **Step 4: Update handler_test.go**

Remove:
1. `generateResult` and `generateErr` fields from `stubParseService` (lines 21-22)
2. `GenerateForUser` method from `stubParseService` (lines 31-34)
3. All `TestGenerate_*` test functions (lines 233-355)
4. `newGenerateTestRouter` helper (lines 400-408)
5. `newUnauthorizedGenerateTestRouter` helper (lines 420-424)
6. `performGenerateRequest` helper (lines 436-438)
7. The generate route check in `TestRoutePaths_CorrectlyMounted` (lines 371-376)

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/modules/parsing/... -v`
Expected: All remaining tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/parsing/
git commit -m "refactor(parsing): remove Generate handler, generator, and route"
```

---

### Task 4: Update main_test.go route verification

**Files:**
- Modify: `backend/cmd/server/main_test.go:32`

- [ ] **Step 1: Remove generate route from test**

Delete line 32:
```go
{http.MethodPost, "/api/v1/parsing/generate"},
```

- [ ] **Step 2: Run test**

Run: `cd backend && go test ./cmd/server/... -v -run TestResourceRoutes`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main_test.go
git commit -m "test: remove generate route from route verification test"
```

---

### Task 5: Remove generateProject from frontend api-client.ts

**Files:**
- Modify: `frontend/workbench/src/lib/api-client.ts:160-164,220-224`

- [ ] **Step 1: Remove GenerateResult type and generateProject method**

Delete the `GenerateResult` interface (lines 160-164):
```typescript
// DELETE:
export interface GenerateResult {
  draft_id: number
  version_id: number
  html_content: string
}
```

Delete `generateProject` from `parsingApi` (lines 220-224):
```typescript
// DELETE from parsingApi:
  generateProject: (projectId: number) =>
    request<GenerateResult>('/parsing/generate', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
```

- [ ] **Step 2: Verify build**

Run: `cd frontend/workbench && bunx tsc --noEmit`
Expected: Type errors in ProjectDetail.tsx referencing `parsingApi.generateProject` — these will be fixed in Task 6.

- [ ] **Step 3: Commit**

```bash
git add frontend/workbench/src/lib/api-client.ts
git commit -m "refactor(frontend): remove generateProject from api-client"
```

---

### Task 6: Simplify ProjectDetail.tsx — replace "Generate Draft" with "Start Editing"

**Files:**
- Modify: `frontend/workbench/src/pages/ProjectDetail.tsx`

- [ ] **Step 1: Remove generate-related state and imports**

1. Remove `parsingApi` from the import on line 4 (keep `intakeApi`)
2. Remove `parseLoading` state (line 25)
3. Remove `parseError` state (line 26)
4. Delete the entire `handleParse` function (lines 120-136)

- [ ] **Step 2: Replace the button section**

Replace lines 218-231 with:

```tsx
{assets.length > 0 && (
  <Button
    size="lg"
    className="mt-6 w-full h-11"
    onClick={() => navigate(`/projects/${pid}/edit`)}
  >
    {project.current_draft_id ? '继续编辑' : '开始编辑'}
  </Button>
)}
```

- [ ] **Step 3: Remove parseError display**

Delete line 176-177:
```tsx
{parseError && (
  <Alert className="mb-4">生成初稿失败：{parseError}</Alert>
)}
```

- [ ] **Step 4: Verify build**

Run: `cd frontend/workbench && bunx tsc --noEmit`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/workbench/src/pages/ProjectDetail.tsx
git commit -m "refactor(frontend): replace generate draft button with direct edit navigation"
```

---

### Task 7: Auto-create empty draft in EditorPage when none exists

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:145-148`

- [ ] **Step 1: Replace the redirect-when-no-draft logic**

In `EditorPage.tsx`, the current logic at lines 145-148 redirects back to ProjectDetail when there's no draft:

```typescript
if (!project.current_draft_id) {
  navigate(`/projects/${pid}`, { replace: true })
  return
}
```

Replace with auto-create logic:

```typescript
if (!project.current_draft_id) {
  try {
    const draft = await workbenchApi.createDraft(pid)
    if (cancelled) return
    setDraftId(String(draft.id))
    setPendingHtml(draft.html_content || '')
  } catch (createErr) {
    if (cancelled) return
    setError(createErr instanceof ApiError ? createErr.message : '创建草稿失败')
  }
  setLoading(false)
  return
}
```

- [ ] **Step 2: Verify build**

Run: `cd frontend/workbench && bunx tsc --noEmit`
Expected: No errors.

- [ ] **Step 3: Test manually**

1. Create a project and upload a file
2. Click "开始编辑" — should navigate to editor without a draft
3. EditorPage should auto-create an empty draft and load the editor
4. Verify the AI panel is visible and functional

- [ ] **Step 4: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat(frontend): auto-create empty draft when entering editor"
```

---

### Task 8: Update AiPanelPlaceholder text

**Files:**
- Modify: `frontend/workbench/src/components/editor/AiPanelPlaceholder.tsx:10`

- [ ] **Step 1: Update placeholder text**

Replace line 10:
```tsx
AI 助手将帮助您优化简历内容，提供智能建议，并自动生成简历初稿
```
With:
```tsx
AI 助手将帮助您根据需求生成和优化简历内容，提供智能建议
```

- [ ] **Step 2: Commit**

```bash
git add frontend/workbench/src/components/editor/AiPanelPlaceholder.tsx
git commit -m "refactor(frontend): update AI panel placeholder text"
```

---

### Task 9: Update parsing contract docs

**Files:**
- Modify: `docs/modules/parsing/contract.md`

- [ ] **Step 1: Remove generate endpoint section**

Read the file and remove:
1. The `POST /api/v1/parsing/generate` endpoint definition section
2. Error codes 2005 and 2006 from the error code table
3. Any references to `GenerateResult`, `GenerateRequest`, `draft_id`/`version_id`/`html_content` in the generate response
4. The description of the generate flow in any overview section

- [ ] **Step 2: Commit**

```bash
git add docs/modules/parsing/contract.md
git commit -m "docs: remove generate endpoint from parsing contract"
```

---

### Task 10: Final verification

- [ ] **Step 1: Run all backend tests**

Run: `cd backend && go test ./...`
Expected: All pass.

- [ ] **Step 2: Run all frontend tests**

Run: `cd frontend/workbench && bunx vitest run`
Expected: All pass.

- [ ] **Step 3: Run frontend build**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds.

- [ ] **Step 4: Verify no remaining references to generate**

Run: `grep -rn "generate\|GenerateResult\|GenerateHTML\|DraftGenerator" backend/internal/modules/parsing/ --include="*.go" | grep -v "_test.go.bak"`
Expected: No output (no remaining references).

Run: `grep -rn "generateProject\|GenerateResult" frontend/workbench/src/`
Expected: No output.
