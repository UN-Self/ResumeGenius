# A4 Multi-Page Pagination Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the single-page A4 canvas with automatic multi-page pagination using `tiptap-pagination-plus@3.1.0`.

**Architecture:** `tiptap-pagination-plus` renders page breaks purely via ProseMirror `Decoration.widget()` inside the editor DOM. No React wrapper needed — the extension manages everything through CSS variables on `.rm-with-pagination`. Our `A4Canvas` becomes a zoom-only container; the extension controls page width/height/margins.

**Tech Stack:** React 19, TipTap 3.22, tiptap-pagination-plus 3.1.0, Vitest + Testing Library

---

### Task 1: Add PaginationPlus extension to editor configuration

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:57-82`

**Step 1: Import PaginationPlus**

Add the import at the top of EditorPage.tsx (near the other tiptap imports, around line 7):

```ts
import { PaginationPlus, PAGE_SIZES } from 'tiptap-pagination-plus'
```

**Step 2: Add PaginationPlus to extensions array**

In the `useEditor()` call (line 57), add `PaginationPlus` after `Span` in the extensions array, and remove `style: 'min-height: 261mm;'` from editorProps.attributes:

```ts
const editor = useEditor({
  extensions: [
    StarterKit.configure({ strike: false }),
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
    TextStyleKit,
    PresetAttributes,
    Div,
    Span,
    PaginationPlus.configure({
      // A4 at 96dpi: 210×297mm ≈ 794×1123px
      pageHeight: 1123,
      pageWidth: 794,
      // Margins: 18mm top/bottom ≈ 68px, 20mm left/right ≈ 76px
      marginTop: 68,
      marginBottom: 68,
      marginLeft: 76,
      marginRight: 76,
      pageGap: 32,
      contentMarginTop: 0,
      contentMarginBottom: 0,
      pageBreakBackground: '#ede8e0',  // matches --color-canvas-bg
      pageGapBorderSize: 0,             // no visible border in gap
      // No headers/footers
      headerLeft: '',
      headerRight: '',
      footerLeft: '',
      footerRight: '',
    }),
  ],
  content: '',
  editorProps: {
    attributes: {
      class: 'resume-content outline-none',
      // REMOVED: style: 'min-height: 261mm;'
    },
    handleDOMEvents: {
      copy(_view, event) {
        const { from, to } = _view.state.selection
        const plainText = _view.state.doc.textBetween(from, to, '\n')
        event.preventDefault()
        event.clipboardData?.setData('text/plain', plainText)
        return true
      },
    },
  },
})
```

**Step 3: Run existing tests to verify no unexpected breakage**

```bash
cd frontend/workbench && bunx vitest run tests/A4Canvas.test.tsx tests/EditorPage.test.tsx 2>&1
```

Expected: Tests that check for `width: '210mm'` and `minHeight: '297mm'` on `a4-canvas` will FAIL — this is expected, we'll update them in Task 4.

**Step 4: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat: add PaginationPlus extension to editor, remove single-page min-height"
```

---

### Task 2: Refactor A4Canvas for multi-page layout

**Files:**
- Modify: `frontend/workbench/src/components/editor/A4Canvas.tsx`

**Step 1: Remove fixed canvas dimensions and padding**

The extension now controls page width (`--rm-page-width`) and margins (`--rm-margin-*`) via CSS variables on the editor DOM. The A4Canvas should no longer set `width`, `minHeight`, or `padding` on the `.resume-document` div.

Replace the current render block (lines 45-63) — remove `width: '210mm'`, `minHeight: '297mm'`, and `p-[18mm_20mm]` from the `.resume-document` div. Keep the zoom transform and the shadow/ring styling.

The updated `A4Canvas` return:

```tsx
return (
  <div ref={containerRef} className="canvas-area bg-canvas-bg">
    <div
      data-testid="a4-canvas"
      className={`${RESUME_DOCUMENT_CLASS} relative bg-resume-paper shadow-[0_22px_80px_rgba(2,8,23,0.24)] ring-1 ring-black/5`}
      style={{
        transform: `scale(${zoom})`,
        transformOrigin: 'top center',
      }}
    >
      {scopedCSS && <style dangerouslySetInnerHTML={{ __html: scopedCSS }} />}
      {children || (editor && <TipTapEditor editor={editor} />)}
      <WatermarkOverlay />
    </div>
  </div>
)
```

The extension sets width via `--rm-page-width: 794px` on the editor DOM (`.rm-with-pagination`), which is the `.ProseMirror` element inside `TipTapEditor`. The zoom still scales the entire `.resume-document` container.

**Step 2: Update zoom calculation**

The zoom calculation currently uses `CANVAS_WIDTH_MM * 3.7795` (210mm → 794px). This is still correct since the page width is 794px (A4). No change needed to `computeZoom()`.

However, we need to account for the margins in zoom. The editor width is now `794px + marginLeft + marginRight = 794 + 76 + 76 = 946px` (set by the extension via padding). Update `computeZoom`:

```ts
const CANVAS_TOTAL_WIDTH_PX = 794 + 76 + 76  // page width + margins

function computeZoom(containerWidth: number): number {
  const availableWidth = containerWidth - CANVAS_PADDING_PX
  const zoom = availableWidth / CANVAS_TOTAL_WIDTH_PX
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom))
}
```

Also remove the unused `CANVAS_WIDTH_MM` constant if no longer referenced.

**Step 3: Verify the component compiles**

```bash
cd frontend/workbench && bunx tsc --noEmit src/components/editor/A4Canvas.tsx 2>&1
```

**Step 4: Commit**

```bash
git add frontend/workbench/src/components/editor/A4Canvas.tsx
git commit -m "feat: refactor A4Canvas to zoom-only container, extension controls page sizing"
```

---

### Task 3: Refactor WatermarkOverlay for multi-page injection

**Files:**
- Modify: `frontend/workbench/src/components/editor/WatermarkOverlay.tsx`

**Step 1: Understand the new approach**

Currently `WatermarkOverlay` renders a single watermark in a `.watermark-container` via React. Since the extension renders page breaks as ProseMirror decorations (native DOM, not React), we need to inject watermarks into each `.rm-page-break > .page` element using a MutationObserver.

The new `WatermarkOverlay` will:
1. Watch the parent DOM for `[data-rm-pagination]` (the pages wrapper created by the extension)
2. For each `.rm-page-break > .page`, ensure a watermark child exists
3. Clean up watermarks when pages are removed

```tsx
import { useEffect, useRef, useMemo } from 'react'

interface WatermarkOverlayProps {
  visible?: boolean
}

const SVG_CONTENT = `<svg xmlns="http://www.w3.org/2000/svg" width="280" height="100">
  <text x="140" y="35" text-anchor="middle" font-size="16" font-weight="600" fill="#1a1815">ResumeGenius 预览</text>
  <text x="140" y="60" text-anchor="middle" font-size="10" fill="#1a1815">导出无水印 PDF 即可去除水印</text>
</svg>`

export function WatermarkOverlay({ visible = true }: WatermarkOverlayProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  const bgImage = useMemo(
    () => `url("data:image/svg+xml;charset=utf-8,${encodeURIComponent(SVG_CONTENT)}")`,
    [],
  )

  useEffect(() => {
    if (!visible || !containerRef.current) return

    const container = containerRef.current

    const ensureWatermarks = () => {
      const pagesWrapper = container.querySelector('[data-rm-pagination]')
      if (!pagesWrapper) return

      const pageBreaks = pagesWrapper.querySelectorAll('.rm-page-break')
      pageBreaks.forEach((pageBreak) => {
        const page = pageBreak.querySelector(':scope > .page')
        if (!page) return

        // Check if this page already has a watermark
        const existing = page.querySelector('[data-testid="watermark-overlay"]')
        if (existing) return

        // Create watermark element for this page
        const watermark = document.createElement('div')
        watermark.setAttribute('data-testid', 'watermark-overlay')
        watermark.style.cssText = [
          'position: absolute;',
          'inset: 0;',
          'overflow: hidden;',
          'pointer-events: none;',
          'z-index: 5;',
        ].join('')

        const inner = document.createElement('div')
        inner.style.cssText = [
          'position: absolute;',
          'inset: -50%;',
          `background-image: ${bgImage};`,
          'background-repeat: repeat;',
          'background-size: 280px 100px;',
          'transform: rotate(-30deg);',
          'pointer-events: none;',
          'user-select: none;',
          'opacity: 0.12;',
        ].join('')

        watermark.appendChild(inner)
        page.appendChild(watermark)
      })
    }

    // Initial injection
    ensureWatermarks()

    // Watch for new pages being added/removed
    const observer = new MutationObserver(() => {
      ensureWatermarks()
    })

    observer.observe(container, { childList: true, subtree: true })
    return () => observer.disconnect()
  }, [visible, bgImage])

  if (!visible) return null

  return <div ref={containerRef} style={{ display: 'contents' }} />
}
```

The key changes:
- The component returns a transparent `<div style="display: contents">` as an anchor for the MutationObserver
- Watermarks are created via vanilla DOM APIs and injected into each `.rm-page-break > .page`
- The MutationObserver watches for page DOM changes and re-injects watermarks as needed

**Step 2: Run watermark tests**

```bash
cd frontend/workbench && bunx vitest run tests/WatermarkOverlay.test.tsx 2>&1
```

Expected: Tests that check for `watermark-overlay` in the React render output will need updating — the watermarks are now injected via vanilla DOM into the extension's decoration elements. Update tests to reflect the new approach.

**Step 3: Commit**

```bash
git add frontend/workbench/src/components/editor/WatermarkOverlay.tsx frontend/workbench/tests/WatermarkOverlay.test.tsx
git commit -m "feat: refactor watermark to inject per-page watermarks via MutationObserver"
```

---

### Task 4: Add page break CSS styles

**Files:**
- Modify: `frontend/workbench/src/styles/editor.css`

**Step 1: Add styles for extension-generated elements**

Append the following CSS after the `.resume-document` block (after line 69):

```css
/* === Multi-Page Pagination (tiptap-pagination-plus) === */

/* Page shadows — each .page gets an A4 paper look */
.rm-page-break .page {
  background: #ffffff;
  box-shadow: 0 4px 24px rgba(2, 8, 23, 0.06), 0 1px 3px rgba(0, 0, 0, 0.04);
}

/* Ensure breaker fills the editor width */
.rm-page-break .breaker {
  clear: both;
}

/* Page gap area — blend with canvas background */
.rm-pagination-gap {
  background-color: var(--color-canvas-bg, #ede8e0) !important;
}
```

**Step 2: Update print protection to cover paginated pages**

Update the `@media print` block at line 1459 to also hide pagination elements:

```css
@media print {
  .canvas-area,
  .a4-page,
  .rm-pagination-gap,
  .rm-page-break .breaker,
  .rm-page-header,
  .rm-page-footer {
    display: none !important;
  }

  body::after {
    content: "导出无水印 PDF 即可获得完整简历 — ResumeGenius";
    display: block;
    font-size: 20px;
    text-align: center;
    padding-top: 40vh;
    color: #1a1815;
  }
}
```

**Step 3: Commit**

```bash
git add frontend/workbench/src/styles/editor.css
git commit -m "feat: add CSS styles for multi-page pagination decorations"
```

---

### Task 5: Update tests

**Files:**
- Modify: `frontend/workbench/tests/A4Canvas.test.tsx`
- Modify: `frontend/workbench/tests/WatermarkOverlay.test.tsx`

**Step 1: Update A4Canvas tests**

The test currently asserts `width: '210mm'` and `minHeight: '297mm'` on the canvas. Since the extension now controls sizing, update the assertion:

```ts
// OLD assertions (remove):
// expect(canvas).toHaveStyle({ width: '210mm', minHeight: '297mm' })

// NEW assertions:
// The canvas should still exist but sizing is handled by the extension
it('renders the editor page with an A4 canvas', async () => {
  const canvas = await screen.findByTestId('a4-canvas')
  expect(canvas).toBeInTheDocument()
  // The canvas has the resume-document class and zoom transform
  expect(canvas).toHaveClass('resume-document')
  expect(canvas.style.transform).toMatch(/scale\(/)
})
```

**Step 2: Update WatermarkOverlay tests**

The watermark now uses `display: contents` as anchor and injects watermarks into extension-generated DOM via MutationObserver. Update tests:

```ts
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { WatermarkOverlay } from '@/components/editor/WatermarkOverlay'

describe('WatermarkOverlay', () => {
  it('renders a display:contents anchor element when visible=true (default)', () => {
    const { container } = render(<WatermarkOverlay />)
    // The component renders a div with display:contents as observer anchor
    const anchor = container.firstElementChild as HTMLElement
    expect(anchor).toBeInTheDocument()
    expect(anchor.style.display).toBe('contents')
  })

  it('has SVG background-image ready for injection', () => {
    // The bgImage is pre-computed via useMemo — we verify it exists
    // by checking that the component renders without errors
    render(<WatermarkOverlay />)
    // No watermark-overlay in React tree (it's injected via vanilla DOM)
    expect(screen.queryByTestId('watermark-overlay')).toBeNull()
  })

  it('does not render when visible=false', () => {
    const { container } = render(<WatermarkOverlay visible={false} />)
    expect(container.innerHTML).toBe('')
  })

  it('injects watermarks into extension page elements when present', async () => {
    const { container } = render(
      <div>
        <WatermarkOverlay />
        <div data-rm-pagination="true" id="pages">
          <div class="rm-page-break">
            <div class="page" style="position: relative;"></div>
            <div class="breaker"></div>
          </div>
          <div class="rm-page-break">
            <div class="page" style="position: relative;"></div>
            <div class="breaker"></div>
          </div>
        </div>
      </div>
    )

    // Wait for MutationObserver to fire
    await vi.waitFor(() => {
      const watermarks = container.querySelectorAll('[data-testid="watermark-overlay"]')
      expect(watermarks.length).toBe(2)
    })
  })
})
```

**Step 3: Run all tests**

```bash
cd frontend/workbench && bunx vitest run tests/A4Canvas.test.tsx tests/WatermarkOverlay.test.tsx 2>&1
```

Expected: All tests PASS.

**Step 4: Commit**

```bash
git add frontend/workbench/tests/A4Canvas.test.tsx frontend/workbench/tests/WatermarkOverlay.test.tsx
git commit -m "test: update tests for multi-page pagination layout"
```

---

### Task 6: Manual verification

**Files:** No changes — verification only.

**Step 1: Start the dev server and verify visually**

```bash
cd frontend/workbench && bun run dev 2>&1
```

Open `http://localhost:3000/app/projects/1/edit` and verify:

1. Editor loads with A4 page dimensions
2. Add enough content to overflow one page → second page appears automatically
3. Watermarks appear on every page
4. Zoom in/out works correctly with multi-page layout
5. Save, AI chat, export still function

**Step 2: Run the full test suite**

```bash
cd frontend/workbench && bunx vitest run 2>&1
```

Expected: All tests PASS.

**Step 3: Commit any final adjustments**

```bash
git add -A
git commit -m "chore: final adjustments after manual verification"
```
