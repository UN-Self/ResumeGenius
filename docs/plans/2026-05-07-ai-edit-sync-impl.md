# AI 编辑同步修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix AI edits being overwritten by auto-save — after `done` event, fetch clean HTML from backend instead of applying dirty diff markup.

**Architecture:** Replace the frontend diff-markup approach (`<del>/<ins>`) with a simple fetch-and-replace strategy. After AI completes edits (backed by `draft.html_content`), the frontend fetches the clean HTML via `workbenchApi.getDraft()` and sets it into the editor under a `restoringContent` flag that suppresses auto-save.

**Tech Stack:** React 18, TipTap, TypeScript, Vitest, MSW

---

### Task 1: Update ChatPanel tests for new `onApplyEdits` prop

**Files:**
- Modify: `frontend/workbench/tests/ChatPanel.test.tsx`

- [ ] **Step 1: Update the `renderChatPanel` helper**

Change the second parameter from `onApplyDiffHTML` to `onApplyEdits`, make it an async mock:

```typescript
function renderChatPanel(draftId = 1, onApplyEdits = vi.fn().mockResolvedValue(undefined), onRestoreHtml = vi.fn()) {
  return render(<ChatPanel draftId={draftId} onApplyEdits={onApplyEdits} onRestoreHtml={onRestoreHtml} />)
}
```

Remove the `import type { PendingEdit } from '@/lib/api-client'` line entirely.

- [ ] **Step 2: Rewrite the edit-events test**

Replace the "handles edit events and calls onApplyDiffHTML with pending edits" test with:

```typescript
it('calls onApplyEdits when done event fires', async () => {
  const onApplyEdits = vi.fn().mockResolvedValue(undefined)
  renderChatPanel(1, onApplyEdits)
  const input = screen.getByPlaceholderText('输入你的需求...')
  await waitFor(() => {
    expect(input).toBeEnabled()
  })

  await userEvent.type(input, '优化简历')
  const sendButton = screen.getByRole('button', { name: /发送/i })
  await userEvent.click(sendButton)

  await waitFor(() => {
    expect(onApplyEdits).toHaveBeenCalledTimes(1)
  })
})
```

- [ ] **Step 3: Remove the pending-edits indicator test**

Delete the test that checks for "已生成 N 项修改" UI (if it exists). The pending edits UI is being removed.

- [ ] **Step 4: Update all other tests that call `renderChatPanel`**

Every call to `renderChatPanel(1, vi.fn(), ...)` should work with the new signature since `vi.fn()` is compatible. Update the "shows undo/redo buttons" and "calls onRestoreHtml" tests to use the new signature:

```typescript
it('shows undo/redo buttons in chat after edits are applied', async () => {
  const onRestoreHtml = vi.fn()
  renderChatPanel(1, vi.fn().mockResolvedValue(undefined), onRestoreHtml)
  // ... rest unchanged
})

it('calls onRestoreHtml when undo button is clicked', async () => {
  const restoredHTML = '<p>restored content</p>'
  const onRestoreHtml = vi.fn()
  server.use(
    http.post('/api/v1/ai/drafts/1/undo', () => {
      return HttpResponse.json({ code: 0, data: { html_content: restoredHTML } })
    }),
  )
  renderChatPanel(1, vi.fn().mockResolvedValue(undefined), onRestoreHtml)
  // ... rest unchanged
})
```

- [ ] **Step 5: Run tests to verify they fail**

Run: `cd frontend/workbench && bunx vitest run tests/ChatPanel.test.tsx`
Expected: FAIL — TypeScript errors because `onApplyEdits` prop doesn't exist yet on ChatPanel

---

### Task 2: Implement ChatPanel changes

**Files:**
- Modify: `frontend/workbench/src/components/chat/ChatPanel.tsx`

- [ ] **Step 1: Update the Props interface and imports**

Replace line 4:
```typescript
import { agentApi, undoDraft, redoDraft, type AISession, type ToolCallEntry, type PendingEdit } from '@/lib/api-client'
```
with:
```typescript
import { agentApi, undoDraft, redoDraft, type AISession, type ToolCallEntry } from '@/lib/api-client'
```

Replace the Props interface (lines 12-16):
```typescript
interface Props {
  draftId: number
  onApplyEdits?: () => Promise<void>
  onRestoreHtml?: (html: string) => void
}
```

Replace the function signature (line 18):
```typescript
export function ChatPanel({ draftId, onApplyEdits, onRestoreHtml }: Props) {
```

- [ ] **Step 2: Remove `pendingEdits` state**

Delete line 26:
```typescript
const [pendingEdits, setPendingEdits] = useState<PendingEdit[]>([])
```

Remove `setPendingEdits([])` from the `useEffect` cleanup (line 37) and `handleNewChat` (line 187). Also remove `setPendingEdits([])` from `handleSend` (line 81).

- [ ] **Step 3: Remove the `edits` accumulator and simplify the SSE loop**

In `handleSend`, delete line 102:
```typescript
const edits: PendingEdit[] = []
```

Replace the entire `edit` case (lines 133-138):
```typescript
case 'edit':
  if (event.params?.ops) {
    edits.push(...(event.params.ops as PendingEdit[]))
    setPendingEdits([...edits])
  }
  break
```
with an empty case (or just remove it entirely — there's no default case so unhandled events are silently ignored).

Replace the `done` case (lines 152-159):
```typescript
case 'done':
  gotDone = true
  if (edits.length > 0 && onApplyDiffHTML) {
    onApplyDiffHTML(edits)
    setEditsApplied(true)
  }
  setPendingEdits([])
  break
```
with:
```typescript
case 'done':
  gotDone = true
  if (onApplyEdits) {
    await onApplyEdits()
    setEditsApplied(true)
  }
  break
```

- [ ] **Step 4: Remove the pending edits indicator UI**

Delete lines 288-293 (the "已生成 N 项修改" indicator block):
```typescript
{streaming && pendingEdits.length > 0 && (
  <div className="flex items-center gap-2 px-3 py-2 text-xs text-[var(--color-text-secondary)] bg-blue-50 border border-blue-200 rounded-lg">
    <span>已生成 {pendingEdits.length} 项修改，等待确认...</span>
  </div>
)}
```

- [ ] **Step 5: Update the `useCallback` dependency array**

In `handleSend`'s dependency array (line 174), replace `onApplyDiffHTML` with `onApplyEdits`:
```typescript
}, [input, session, streaming, onApplyEdits])
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd frontend/workbench && bunx vitest run tests/ChatPanel.test.tsx`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
cd frontend/workbench && git add src/components/chat/ChatPanel.tsx tests/ChatPanel.test.tsx && git commit -m "refactor: ChatPanel uses onApplyEdits instead of onApplyDiffHTML"
```

---

### Task 3: Remove ai-diff extension and its test

**Files:**
- Delete: `frontend/workbench/src/components/editor/extensions/ai-diff.ts`
- Delete: `frontend/workbench/tests/AiDiffExtension.test.ts`

- [ ] **Step 1: Delete the ai-diff extension file**

Run: `rm frontend/workbench/src/components/editor/extensions/ai-diff.ts`

- [ ] **Step 2: Delete the ai-diff extension test file**

Run: `rm frontend/workbench/tests/AiDiffExtension.test.ts`

- [ ] **Step 3: Remove Deletion/Insertion imports from EditorPage**

In `frontend/workbench/src/pages/EditorPage.tsx`, delete line 13:
```typescript
import { Deletion, Insertion } from '@/components/editor/extensions/ai-diff'
```

Remove `Deletion` and `Insertion` from the editor extensions array (lines 50-51):
```typescript
extensions: [
  StarterKit.configure({ strike: false }),
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
],
```

- [ ] **Step 4: Run tests to verify nothing is broken**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd frontend/workbench && git add -A && git commit -m "refactor: remove ai-diff extension (no longer needed)"
```

---

### Task 4: Update EditorPage with `onApplyEdits` callback and `restoringContent` protection

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`

- [ ] **Step 1: Remove `PendingEdit` import from EditorPage**

In line 14, change:
```typescript
import { request, intakeApi, workbenchApi, ApiError, type Asset, type PendingEdit } from '@/lib/api-client'
```
to:
```typescript
import { request, intakeApi, workbenchApi, ApiError, type Asset } from '@/lib/api-client'
```

- [ ] **Step 2: Replace the `onApplyDiffHTML` callback with `onApplyEdits`**

In the ChatPanel JSX (around line 424), replace the entire `onApplyDiffHTML` prop:
```typescript
onApplyDiffHTML={(edits: PendingEdit[]) => {
  if (!editor) return
  const currentHTML = editor.getHTML()
  let diffHTML = currentHTML
  for (const edit of edits) {
    diffHTML = diffHTML.replace(
      edit.old_string,
      `<del>${edit.old_string}</del><ins>${edit.new_string}</ins>`
    )
  }
  editor.commands.setContent(diffHTML)
}}
```
with:
```typescript
onApplyEdits={async () => {
  if (!editor || !draftId) return
  const draft = await workbenchApi.getDraft(Number(draftId))
  restoringContent.current = true
  editor.commands.setContent(draft.html_content || '')
  restoringContent.current = false
}}
```

- [ ] **Step 3: Add `restoringContent` protection to `onRestoreHtml`**

Replace line 436:
```typescript
onRestoreHtml={(html) => editor?.commands.setContent(html)}
```
with:
```typescript
onRestoreHtml={(html) => {
  if (!editor) return
  restoringContent.current = true
  editor.commands.setContent(html)
  restoringContent.current = false
}}
```

- [ ] **Step 4: Verify the existing auto-save test still passes**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.autosave.test.tsx`
Expected: PASS (ChatPanel is fully mocked in this test, so prop changes don't affect it)

- [ ] **Step 5: Commit**

```bash
cd frontend/workbench && git add src/pages/EditorPage.tsx && git commit -m "feat: EditorPage fetches clean HTML after AI edits instead of diff markup"
```

---

### Task 5: Remove `PendingEdit` type from api-client

**Files:**
- Modify: `frontend/workbench/src/lib/api-client.ts`

- [ ] **Step 1: Remove the PendingEdit type export**

Delete lines 175-179 in `api-client.ts`:
```typescript
export interface PendingEdit {
  old_string: string
  new_string: string
  description?: string
}
```

- [ ] **Step 2: Verify no remaining imports of PendingEdit**

Run: `grep -r "PendingEdit" frontend/workbench/src/ frontend/workbench/tests/`
Expected: No matches (already cleaned up in previous tasks)

- [ ] **Step 3: Run full test suite**

Run: `cd frontend/workbench && bunx vitest run`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd frontend/workbench && git add src/lib/api-client.ts && git commit -m "refactor: remove unused PendingEdit type"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run the full test suite one more time**

Run: `cd frontend/workbench && bunx vitest run`
Expected: All tests PASS

- [ ] **Step 2: Verify the build succeeds**

Run: `cd frontend/workbench && bun run build`
Expected: Build completes with no errors

- [ ] **Step 3: Verify TypeScript type-checking**

Run: `cd frontend/workbench && bunx tsc --noEmit`
Expected: No type errors
