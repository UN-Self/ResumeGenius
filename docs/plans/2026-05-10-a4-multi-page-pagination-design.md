# A4 Multi-Page Pagination Design

**Date:** 2026-05-10
**Branch:** feat/a4-multi-pages
**Package:** `tiptap-pagination-plus@3.1.0`

## Overview

Replace the single-page A4 canvas with automatic multi-page pagination using `tiptap-pagination-plus`. Content exceeding one A4 page automatically flows into subsequent pages, with visual page breaks rendered via ProseMirror decorations.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Pagination mode | Auto only, no manual breaks | Resumes should flow naturally; manual page breaks add complexity with no user benefit |
| Zoom strategy | Keep `transform: scale()` wrapper outside pagination | Existing zoom UX works well; pagination decorations scale with editor DOM automatically |
| Headers/footers | None (all empty strings) | Resumes are 1-2 pages; page numbers add noise |
| Watermarks | Per-page injection | Every page needs anti-copy protection |
| Page wrapper | Extension's native decoration rendering (no React wrapper) | `tiptap-pagination-plus` renders entirely via `Decoration.widget()`, no React component needed |

## Architecture

### Before (Single Page)

```
A4Canvas (zoom container)
  └─ .resume-document (width: 210mm, min-height: 297mm, padding: 18mm 20mm)
       ├─ <style scopedCSS>
       ├─ TipTapEditor → EditorContent
       └─ WatermarkOverlay (single)
```

### After (Multi-Page)

```
A4Canvas (zoom container only)
  └─ .resume-document (no fixed width/height/padding — extension manages via CSS vars)
       ├─ <style scopedCSS>
       ├─ TipTapEditor → EditorContent (.rm-with-pagination)
       │    ├─ [decoration: rm-first-page-header] (empty for us)
       │    └─ [decoration: #pages data-rm-pagination]
       │         ├─ .rm-page-break (page 1)
       │         │    ├─ .page (content area, float positioning)
       │         │    └─ .breaker (page gap + nav)
       │         ├─ .rm-page-break (page 2)
       │         │    ├─ .page
       │         │    └─ .breaker
       │         └─ ... (page N)
       └─ WatermarkOverlay (per-page → injected into each .rm-page-break > .page)
```

### Key Mechanism

The extension uses two `Decoration.widget(0, ..., { side: -1 })` widgets rendered before the editor content:

1. **`#pages` widget** — Contains all page break decorations: `.rm-page-break` elements with `.page` (using `float: left; clear: both` + calculated `margin-top` to position at page boundaries) and `.breaker` (gap + headers/footers)
2. **First page header widget** — Separate header decoration rendered at position 0

Page count is calculated by measuring editor content height vs page content area height.

## Files Changed

| File | Change | Complexity |
|------|--------|------------|
| `frontend/workbench/src/pages/EditorPage.tsx` | Add `PaginationPlus` extension; remove `min-height: 261mm` | Low |
| `frontend/workbench/src/components/editor/A4Canvas.tsx` | Remove fixed canvas sizing (extension manages via CSS vars); update zoom calculation; restructure watermark for per-page injection | Medium |
| `frontend/workbench/src/components/editor/WatermarkOverlay.tsx` | Multi-page support: iterate `.rm-page-break > .page`, inject watermark into each | Medium |
| `frontend/workbench/src/components/editor/editor.css` | Add styles for `.rm-page-break`, `.breaker`, `.rm-pagination-gap` (page shadow, A4 paper background, gap colors) | Low |

## Files NOT Changed

- `extract-styles.ts` — CSS scoping pipeline unchanged
- `TipTapEditor.tsx` — Pure `EditorContent` wrapper unchanged
- All custom extensions (Div, Span, PresetAttributes, TextStyleKit)
- All toolbar/menu components
- Save, AI chat, export flows

## Extension Configuration

```ts
PaginationPlus.configure({
  // A4 at 96dpi: 210×297mm ≈ 794×1123px
  pageHeight: 1123,
  pageWidth: 794,
  // Margins match current A4Canvas padding (18mm top/bottom, 20mm left/right ≈ 68px, 76px)
  marginTop: 68,
  marginBottom: 68,
  marginLeft: 76,
  marginRight: 76,
  pageGap: 32,
  contentMarginTop: 0,
  contentMarginBottom: 0,
  pageBreakBackground: "#f3f4f6",  // matches bg-canvas-bg
  // No headers/footers
  headerLeft: "",
  headerRight: "",
  footerLeft: "",
  footerRight: "",
})
```

## CSS Variable Mapping

The extension sets these CSS variables on the editor DOM (`.rm-with-pagination`):

| Variable | Value | Purpose |
|----------|-------|---------|
| `--rm-page-height` | 1123px | Page content area height |
| `--rm-page-width` | 794px | Editor width |
| `--rm-margin-top` | 68px | Page top margin |
| `--rm-margin-bottom` | 68px | Page bottom margin |
| `--rm-margin-left` | 76px | Page left margin |
| `--rm-margin-right` | 76px | Page right margin |
| `--rm-content-margin-top` | 0px | Gap between header and content |
| `--rm-content-margin-bottom` | 0px | Gap between content and footer |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Extension v3.1.0 is young, API may change | API surface is small (~5 config options we use); easy to fork or replace |
| Float-based decoration positioning may conflict with existing CSS | Test with real resume content; add CSS resets as needed |
| Page break calculation may be inaccurate with mixed font sizes | Extension uses `getBoundingClientRect()` for measurement; test with varied content |
| Watermark injection timing (decorations are async via rAF) | Use `MutationObserver` or `requestAnimationFrame` to detect page DOM ready |

## Success Criteria

1. Content exceeding one A4 page (297mm) automatically creates page 2, 3, etc.
2. Each page renders with correct A4 dimensions (210×297mm)
3. Page gaps are visually distinct (gray background matching canvas)
4. Watermark appears on every page
5. Zoom scaling works correctly with multi-page layout
6. Existing features unaffected: save, AI chat, export, text editing, toolbar
7. No regression in existing tests
