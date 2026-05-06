# Remove AI Draft Generation & Simplify Flow

## Goal

Remove the `/api/v1/parsing/generate` endpoint and simplify the user flow so users go directly from asset upload/parse into the editor, where the AI assistant guides the conversation to understand requirements before generating content.

## Current Flow

```
Upload assets -> Auto parse -> Click "Generate Draft" -> POST /parsing/generate -> Navigate to editor
```

The editor page rejects entry if `project.current_draft_id` is null (redirects back to ProjectDetail).

## New Flow

```
Upload assets -> Auto parse -> Click "Start Editing" -> Auto-create empty draft -> Enter editor (AI-guided conversation)
```

## Backend Changes

### Delete files

- `backend/internal/modules/parsing/generator.go` - DraftGenerator implementation (AI API call)
- `backend/internal/modules/parsing/generator_test.go` - tests for generator

### Modify files

- **`parsing/routes.go`**: Remove `/parsing/generate` route registration and `DraftGenerator` instantiation
- **`parsing/handler.go`**: Remove `Generate()` method, `GenerateRequest`/`GenerateResponse` types, error codes 2005 (AI generation failed) and 2006 (no generatable text)
- **`parsing/service.go`**: Remove `GenerateForUser()` and `Generate()` methods
- **`parsing/types.go`**: Remove `DraftGeneratorInterface` interface
- **`cmd/server/main.go`**: Remove generate route from route verification test

### No changes to

- Parse functionality (PDF/DOCX/image parsing)
- `metadata.parsing` structure (intake module still uses it)
- Environment variables `AI_API_URL`/`AI_API_KEY`/`AI_MODEL` (agent module uses them)
- `intake/service.go` cross-references to `metadata.parsing.*`

## Frontend Changes

- **`api-client.ts`**: Remove `parsingApi.generateProject` method and `GenerateResult` type
- **`ProjectDetail.tsx`**:
  - Replace "下一步：生成初稿" button with "开始编辑"
  - Remove `handleParse` generate logic; button directly navigates to `/projects/{pid}/edit`
  - Remove `parseLoading`/`parseError` state variables and related error display
- **`EditorPage.tsx`**: On load, if `!project.current_draft_id`, call `workbenchApi.createDraft(pid)` to create an empty draft, then proceed to load editor (instead of redirecting back)

## Documentation Changes

- **`docs/modules/parsing/contract.md`**: Remove generate endpoint, its request/response specs, and error codes 2005/2006
