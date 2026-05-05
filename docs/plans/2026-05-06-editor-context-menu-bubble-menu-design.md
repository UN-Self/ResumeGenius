# TipTap Editor Context Menu & BubbleMenu Design

## Summary

Enhance the TipTap editor with two floating interaction layers:
1. **Custom context menu** (right-click) — basic editing operations
2. **BubbleMenu** (text selection) — compact floating formatting toolbar
3. **AlignSelector** — new dropdown component replacing 4 alignment buttons in FormatToolbar

All changes preserve the existing content protection mechanism (plain-text copy only).

## Approach

**Approach A: TipTap BubbleMenu extension + Custom React context menu**

- `@tiptap/extension-bubble-menu` for the floating toolbar (official, free, tippy.js-based)
- Custom React ContextMenu component with absolute positioning
- Minimal new dependencies, maximum reuse of existing toolbar sub-components

## Architecture

### New Files

| File | Purpose |
|---|---|
| `components/editor/ContextMenu.tsx` | Custom right-click menu component |
| `components/editor/BubbleToolbar.tsx` | BubbleMenu floating toolbar (compact layout) |
| `components/editor/AlignSelector.tsx` | Alignment dropdown selector (left/center/right/justify) |

### Modified Files

| File | Change |
|---|---|
| `pages/EditorPage.tsx` | Register BubbleMenu extension, render ContextMenu + BubbleToolbar |
| `components/editor/A4Canvas.tsx` | Remove `onContextMenu={e.preventDefault()}` |
| `components/editor/FormatToolbar.tsx` | Replace 4 alignment buttons with `AlignSelector` dropdown |

### New Dependency

- `@tiptap/extension-bubble-menu` (official TipTap extension)

### Unchanged

- All existing FormatToolbar sub-components (FontSelector, FontSizeSelector, ColorPicker, LineHeightSelector)
- Content protection logic (plain-text copy in `handleDOMEvents.copy`)

## Context Menu Design

### Menu Items

| Item | Shortcut | Disabled When |
|---|---|---|
| Undo | Ctrl+Z | `!editor.can().undo()` |
| Redo | Ctrl+Y | `!editor.can().redo()` |
| ---separator--- | | |
| Cut | Ctrl+X | No selection |
| Copy | Ctrl+C | No selection |
| Paste | Ctrl+V | (always enabled) |
| ---separator--- | | |
| Select All | Ctrl+A | Never |

### Interaction

- Right-click in editor area → show menu at cursor position
- Click menu item → execute action, close menu
- Click outside menu / press Esc → close menu
- Copy/Cut use existing plain-text extraction logic (content protection preserved)
- Paste allows rich HTML paste (inbound, needed for editing workflow)

### Styling

- White background, border, `rounded-lg`
- Box-shadow allowed (Popover floating over A4 canvas per design system)
- `z-index: 20` (panel buttons layer)
- Menu items: `min-h-9` with icon + label + shortcut hint
- Separator: `h-px bg-border`
- Disabled items: `opacity-50 cursor-not-allowed`

### Implementation

```typescript
interface ContextMenuState {
  isOpen: boolean
  x: number
  y: number
}
```

- `useEffect` listens to `contextmenu` event on editor container
- `useEffect` listens to `mousedown` (outside click) and `keydown` (Esc) to close
- Actions via `editor.chain().focus().xxx().run()`
- Copy/Cut delegate to `document.execCommand('copy'/'cut')` after setting plain-text clipboard data

## BubbleMenu Design

### Compact Layout Strategy

- **Icon-only buttons** for toggle actions (bold, italic, underline, lists) — tooltip on hover
- **Compact dropdown triggers** for selectors (font, size, alignment, line height)
- Tight spacing: `gap-0.5` within groups, `gap-2` between groups
- No ToolbarSeparator; use spacing to delineate groups

### Toolbar Layout (single row)

```
[Font▼] [Size▼] | [B] [I] [U] | [Color] [Highlight] | [Bullet] [Ordered] | [Align▼] | [LineHeight▼]
```

### Show/Hide Logic (`shouldShow`)

- Show when: selection is non-empty (text selected)
- Show during: active drag-select
- Hide when: selection is empty (cursor-only)
- Optionally hide when editor is not focused

### Positioning

- Default: above selection (`placement: "top"`)
- Auto-flip: tippy.js flips to below when insufficient space above
- tippy.js arrow disabled for cleaner look

### Styling

- White background, border, `rounded-lg`
- Box-shadow (Popover over canvas)
- `z-index: 20`
- Match design system: `bg-white border rounded-lg shadow-sm`

### Component Reuse

BubbleToolbar directly imports and renders:
- `FontSelector` from `./FontSelector`
- `FontSizeSelector` from `./FontSizeSelector`
- `ColorPicker` from `./ColorPicker`
- `LineHeightSelector` from `./LineHeightSelector`
- `AlignSelector` from `./AlignSelector`
- `ToolbarButton` from `./ToolbarButton` (icon-only mode)

## AlignSelector Component

New dropdown component replacing the 4 separate alignment buttons.

### Options

| Label | Value | Icon |
|---|---|---|
| Left | `left` | `AlignLeft` |
| Center | `center` | `AlignCenter` |
| Right | `right` | `AlignRight` |
| Justify | `justify` | `AlignJustify` |

### Behavior

- Dropdown trigger shows the currently active alignment icon + small chevron
- Click trigger → dropdown list with all 4 options
- Select option → apply alignment, close dropdown
- Same pattern as existing `FontSelector` / `FontSizeSelector`

### Usage

- Used in both `FormatToolbar` (replacing 4 buttons) and `BubbleToolbar` (compact mode)

## Content Protection

No changes to existing protection:

- Copy/Cut in context menu: extract plain text, write to clipboard as `text/plain`
- Paste: allow rich HTML paste (editing requires it)
- BubbleMenu formatting: does not affect clipboard, no protection concern
- WatermarkOverlay and print protection: unaffected

## Testing Plan

1. Context menu appears on right-click inside editor, dismissed on outside click/Esc
2. Context menu items trigger correct editor actions
3. Undo/Redo disabled states reflect editor history
4. Copy/Cut produce plain-text only (verify via clipboard API)
5. BubbleMenu appears on text selection, hides on empty selection
6. BubbleMenu formatting actions apply correctly
7. AlignSelector dropdown works in both FormatToolbar and BubbleToolbar
8. No regressions in existing FormatToolbar functionality
9. Context menu does not appear outside editor area
