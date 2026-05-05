# Editor Context Menu & BubbleMenu Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a custom right-click context menu (copy/cut/paste/undo/redo/select-all) and a compact BubbleMenu floating toolbar (full formatting) to the TipTap editor.

**Architecture:** Use `@tiptap/extension-bubble-menu` for the floating toolbar that appears on text selection. Build a custom React ContextMenu component with absolute positioning. Create an `AlignSelector` dropdown to replace 4 alignment buttons in FormatToolbar, shared between bottom toolbar and BubbleMenu.

**Tech Stack:** TipTap v3, React 19, Radix UI Popover, lucide-react icons, Vitest + Testing Library

---

### Task 1: Install @tiptap/extension-bubble-menu dependency

**Files:**
- Modify: `frontend/workbench/package.json`

**Step 1: Install the dependency**

Run: `cd frontend/workbench && bun add @tiptap/extension-bubble-menu`

**Step 2: Verify installation**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds (no import errors)

**Step 3: Commit**

```bash
git add frontend/workbench/package.json frontend/workbench/bun.lockb
git commit -m "chore: add @tiptap/extension-bubble-menu dependency"
```

---

### Task 2: Create AlignSelector component

**Files:**
- Create: `frontend/workbench/src/components/editor/AlignSelector.tsx`
- Test: `frontend/workbench/tests/AlignSelector.test.tsx`

Reference files for pattern: `src/components/editor/FontSelector.tsx`, `tests/FontSelector.test.tsx`, `tests/helpers/mock-editor.ts`

**Step 1: Write the failing test**

```tsx
// tests/AlignSelector.test.tsx
import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AlignSelector } from '@/components/editor/AlignSelector'
import { createMockEditor } from './helpers/mock-editor'

describe('AlignSelector', () => {
  it('renders trigger button with default icon', () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /对齐/ })
    expect(trigger).toBeInTheDocument()
  })

  it('shows alignment options when clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))

    expect(screen.getByText('左对齐')).toBeInTheDocument()
    expect(screen.getByText('居中')).toBeInTheDocument()
    expect(screen.getByText('右对齐')).toBeInTheDocument()
    expect(screen.getByText('两端对齐')).toBeInTheDocument()
  })

  it('calls setTextAlign when an option is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))
    await user.click(screen.getByText('居中'))

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('closes popover after selecting an option', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))
    expect(screen.getByText('左对齐')).toBeInTheDocument()

    await user.click(screen.getByText('居中'))

    await waitFor(() => {
      expect(screen.queryByText('右对齐')).not.toBeInTheDocument()
    })
  })

  it('highlights active alignment option', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
      isActive: (name: string, attrs?: Record<string, unknown>) => {
        if (name === 'textAlign' && attrs?.textAlign === 'center') return true
        return false
      },
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))

    // The "居中" item should have the active style (bg-primary-50)
    const centerItem = screen.getByText('居中').closest('button')
    expect(centerItem?.className).toContain('bg-primary-50')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/AlignSelector.test.tsx`
Expected: FAIL — module not found

**Step 3: Write the AlignSelector component**

```tsx
// src/components/editor/AlignSelector.tsx
import { useState, useEffect } from 'react'
import { AlignLeft, ChevronDown } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { DropdownTrigger, DropdownItem } from '@/components/ui/dropdown'

const ALIGNMENTS = [
  { label: '左对齐', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '右对齐', value: 'right' },
  { label: '两端对齐', value: 'justify' },
] as const

interface AlignSelectorProps {
  editor: Editor
}

export function AlignSelector({ editor }: AlignSelectorProps) {
  const [currentAlign, setCurrentAlign] = useState<string | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    if (!editor) return

    const updateAlign = () => {
      for (const { value } of ALIGNMENTS) {
        if (editor.isActive({ textAlign: value })) {
          setCurrentAlign(value)
          return
        }
      }
      setCurrentAlign(null)
    }

    updateAlign()
    editor.on('transaction', updateAlign)
    return () => { editor.off('transaction', updateAlign) }
  }, [editor])

  const handleAlignSelect = (value: string) => {
    editor.chain().focus().setTextAlign(value).run()
    setIsOpen(false)
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <DropdownTrigger aria-label="对齐方式">
          <AlignLeft size={16} />
          <ChevronDown size={14} />
        </DropdownTrigger>
      </PopoverTrigger>
      <PopoverContent side="top" className="w-32 p-1">
        <div className="flex flex-col">
          {ALIGNMENTS.map((align) => (
            <DropdownItem
              key={align.value}
              active={currentAlign === align.value}
              onClick={() => handleAlignSelect(align.value)}
            >
              {align.label}
            </DropdownItem>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/AlignSelector.test.tsx`
Expected: All 5 tests PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/AlignSelector.tsx frontend/workbench/tests/AlignSelector.test.tsx
git commit -m "feat: add AlignSelector dropdown component with tests"
```

---

### Task 3: Update FormatToolbar to use AlignSelector

**Files:**
- Modify: `frontend/workbench/src/components/editor/FormatToolbar.tsx`

**Step 1: Write the failing test**

No new test file needed — the existing FormatToolbar behavior is implicitly tested via AlignSelector tests. We'll verify the build.

**Step 2: Replace alignment buttons with AlignSelector in FormatToolbar**

In `FormatToolbar.tsx`:

1. Remove the `AlignLeft, AlignCenter, AlignRight, AlignJustify` imports from lucide-react
2. Add `import { AlignSelector } from './AlignSelector'`
3. Remove the 4 alignment-related properties from `ActiveStates` interface and `getActiveStates` function:
   - `isAlignLeft`, `isAlignCenter`, `isAlignRight`, `isAlignJustify`
4. Replace the 4 `<ToolbarButton>` elements for alignment (lines 111-134) with `<AlignSelector editor={editor} />`

The final "Line Height + Alignment" group becomes:

```tsx
{/* Line Height + Alignment group */}
<div role="group" aria-label="行距和对齐" className="flex items-center gap-1">
  <LineHeightSelector editor={editor} />
  <AlignSelector editor={editor} />
</div>
```

**Step 3: Verify build**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds

**Step 4: Run existing tests to verify no regressions**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx tests/EditorPage.test.tsx tests/A4Canvas.test.tsx`
Expected: All pass

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/FormatToolbar.tsx
git commit -m "refactor: replace alignment buttons with AlignSelector dropdown in FormatToolbar"
```

---

### Task 4: Create ContextMenu component

**Files:**
- Create: `frontend/workbench/src/components/editor/ContextMenu.tsx`
- Test: `frontend/workbench/tests/ContextMenu.test.tsx`

**Step 1: Write the failing test**

```tsx
// tests/ContextMenu.test.tsx
import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ContextMenu } from '@/components/editor/ContextMenu'
import { createMockEditor } from './helpers/mock-editor'

describe('ContextMenu', () => {
  const mockEditor = () => createMockEditor({
    chainCommands: ['undo', 'redo', 'selectAll'],
  })

  it('renders nothing when closed', () => {
    render(<ContextMenu editor={mockEditor()} isOpen={false} x={0} y={0} onClose={() => {}} />)
    expect(screen.queryByText('撤销')).not.toBeInTheDocument()
  })

  it('renders menu items when open', () => {
    render(<ContextMenu editor={mockEditor()} isOpen={true} x={100} y={200} onClose={() => {}} />)

    expect(screen.getByText('撤销')).toBeInTheDocument()
    expect(screen.getByText('重做')).toBeInTheDocument()
    expect(screen.getByText('剪切')).toBeInTheDocument()
    expect(screen.getByText('复制')).toBeInTheDocument()
    expect(screen.getByText('粘贴')).toBeInTheDocument()
    expect(screen.getByText('全选')).toBeInTheDocument()
  })

  it('shows shortcut hints', () => {
    render(<ContextMenu editor={mockEditor()} isOpen={true} x={0} y={0} onClose={() => {}} />)

    expect(screen.getByText('Ctrl+Z')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+Y')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+X')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+C')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+V')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+A')).toBeInTheDocument()
  })

  it('calls onClose when a menu item is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<ContextMenu editor={mockEditor()} isOpen={true} x={0} y={0} onClose={onClose} />)

    await user.click(screen.getByText('撤销'))
    expect(onClose).toHaveBeenCalled()
  })

  it('calls editor.undo when undo item is clicked', async () => {
    const user = userEvent.setup()
    const editor = mockEditor()
    render(<ContextMenu editor={editor} isOpen={true} x={0} y={0} onClose={() => {}} />)

    await user.click(screen.getByText('撤销'))
    expect(editor.runMock).toHaveBeenCalled()
  })

  it('calls editor.selectAll when select-all item is clicked', async () => {
    const user = userEvent.setup()
    const editor = mockEditor()
    render(<ContextMenu editor={editor} isOpen={true} x={0} y={0} onClose={() => {}} />)

    await user.click(screen.getByText('全选'))
    expect(editor.runMock).toHaveBeenCalled()
  })

  it('positions menu at given coordinates', () => {
    render(<ContextMenu editor={mockEditor()} isOpen={true} x={150} y={300} onClose={() => {}} />)

    const menu = screen.getByRole('menu')
    expect(menu.style.left).toBe('150px')
    expect(menu.style.top).toBe('300px')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend/workbench && bunx vitest run tests/ContextMenu.test.tsx`
Expected: FAIL — module not found

**Step 3: Write the ContextMenu component**

```tsx
// src/components/editor/ContextMenu.tsx
import type { Editor } from '@tiptap/react'
import { Undo2, Redo2, Scissors, Copy, ClipboardPaste, SelectAll } from 'lucide-react'

interface ContextMenuProps {
  editor: Editor | null
  isOpen: boolean
  x: number
  y: number
  onClose: () => void
}

interface MenuItem {
  label: string
  shortcut: string
  icon: React.ReactNode
  action: () => void
  disabled?: boolean
  separator?: false
}

interface MenuSeparator {
  separator: true
}

const MENU_ITEMS = (editor: Editor): (MenuItem | MenuSeparator)[] => [
  {
    label: '撤销',
    shortcut: 'Ctrl+Z',
    icon: <Undo2 size={16} />,
    action: () => editor.chain().focus().undo().run(),
    disabled: !editor.can().undo(),
  },
  {
    label: '重做',
    shortcut: 'Ctrl+Y',
    icon: <Redo2 size={16} />,
    action: () => editor.chain().focus().redo().run(),
    disabled: !editor.can().redo(),
  },
  { separator: true },
  {
    label: '剪切',
    shortcut: 'Ctrl+X',
    icon: <Scissors size={16} />,
    action: () => {
      const { view } = editor
      const { from, to } = view.state.selection
      if (from === to) return
      const plainText = view.state.doc.textBetween(from, to, '\n')
      navigator.clipboard.writeText(plainText)
      editor.chain().focus().deleteSelection().run()
    },
    disabled: editor.state.selection.from === editor.state.selection.to,
  },
  {
    label: '复制',
    shortcut: 'Ctrl+C',
    icon: <Copy size={16} />,
    action: () => {
      const { view } = editor
      const { from, to } = view.state.selection
      if (from === to) return
      const plainText = view.state.doc.textBetween(from, to, '\n')
      navigator.clipboard.writeText(plainText)
    },
    disabled: editor.state.selection.from === editor.state.selection.to,
  },
  {
    label: '粘贴',
    shortcut: 'Ctrl+V',
    icon: <ClipboardPaste size={16} />,
    action: () => {
      navigator.clipboard.readText().then((text) => {
        editor.chain().focus().insertContent(text).run()
      })
    },
  },
  { separator: true },
  {
    label: '全选',
    shortcut: 'Ctrl+A',
    icon: <SelectAll size={16} />,
    action: () => editor.chain().focus().selectAll().run(),
  },
]

export function ContextMenu({ editor, isOpen, x, y, onClose }: ContextMenuProps) {
  if (!isOpen || !editor) return null

  const items = MENU_ITEMS(editor)

  return (
    <div
      role="menu"
      className="fixed z-20 min-w-[180px] bg-white border border-border rounded-lg shadow-sm py-1"
      style={{ left: `${x}px`, top: `${y}px` }}
    >
      {items.map((item, idx) => {
        if ('separator' in item && item.separator) {
          return <div key={`sep-${idx}`} className="h-px bg-border my-1 mx-1" />
        }
        return (
          <button
            key={item.label}
            type="button"
            role="menuitem"
            disabled={item.disabled}
            onClick={() => {
              item.action()
              onClose()
            }}
            className={`
              w-full flex items-center gap-3 px-3 min-h-9 text-sm
              transition-colors duration-150
              ${item.disabled
                ? 'opacity-50 cursor-not-allowed'
                : 'text-foreground hover:bg-surface-hover cursor-pointer'
              }
            `}
          >
            <span className="text-muted-foreground">{item.icon}</span>
            <span className="flex-1 text-left">{item.label}</span>
            <span className="text-xs text-muted-foreground ml-auto">{item.shortcut}</span>
          </button>
        )
      })}
    </div>
  )
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend/workbench && bunx vitest run tests/ContextMenu.test.tsx`
Expected: All 7 tests PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/ContextMenu.tsx frontend/workbench/tests/ContextMenu.test.tsx
git commit -m "feat: add ContextMenu component with undo/redo/cut/copy/paste/select-all"
```

---

### Task 5: Integrate ContextMenu into EditorPage

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Modify: `frontend/workbench/src/components/editor/A4Canvas.tsx`

**Step 1: Remove context menu prevention from A4Canvas**

In `A4Canvas.tsx` line 44, remove the `onContextMenu={(e) => e.preventDefault()}` prop from the container div:

Before:
```tsx
<div ref={containerRef} className="canvas-area bg-canvas-bg" onContextMenu={(e) => e.preventDefault()}>
```

After:
```tsx
<div ref={containerRef} className="canvas-area bg-canvas-bg">
```

**Step 2: Add ContextMenu state and wiring to EditorPage**

In `EditorPage.tsx`, add the following:

1. Import ContextMenu:
```tsx
import { ContextMenu } from '@/components/editor/ContextMenu'
```

2. Add state after existing useState hooks:
```tsx
const [contextMenu, setContextMenu] = useState<{ isOpen: boolean; x: number; y: number }>({
  isOpen: false, x: 0, y: 0,
})
```

3. Add context menu event handlers (after `handleExport`):
```tsx
const handleContextMenu = useCallback((e: React.MouseEvent) => {
  e.preventDefault()
  setContextMenu({ isOpen: true, x: e.clientX, y: e.clientY })
}, [])

const closeContextMenu = useCallback(() => {
  setContextMenu((prev) => ({ ...prev, isOpen: false }))
}, [])
```

4. Add `useEffect` to close context menu on scroll/resize:
```tsx
useEffect(() => {
  const close = () => closeContextMenu()
  document.addEventListener('scroll', close, true)
  return () => document.removeEventListener('scroll', close, true)
}, [closeContextMenu])
```

5. Wire `onContextMenu` onto the A4Canvas container area. In the JSX, change the `<A4Canvas>` to accept the event:
```tsx
<div className="flex-1 overflow-auto" onContextMenu={handleContextMenu}>
  <A4Canvas editor={editor} />
</div>
```

6. Add the ContextMenu component before the closing `</div>` of the main grid, after all panels:
```tsx
<ContextMenu
  editor={editor}
  isOpen={contextMenu.isOpen}
  x={contextMenu.x}
  y={contextMenu.y}
  onClose={closeContextMenu}
/>
```

7. Add a click-outside handler. Add another `useEffect`:
```tsx
useEffect(() => {
  if (!contextMenu.isOpen) return

  const handleClick = (e: MouseEvent) => {
    const target = e.target as HTMLElement
    if (!target.closest('[role="menu"]')) {
      closeContextMenu()
    }
  }

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') closeContextMenu()
  }

  document.addEventListener('mousedown', handleClick)
  document.addEventListener('keydown', handleKeyDown)
  return () => {
    document.removeEventListener('mousedown', handleClick)
    document.removeEventListener('keydown', handleKeyDown)
  }
}, [contextMenu.isOpen, closeContextMenu])
```

**Step 3: Verify build**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds

**Step 4: Run existing tests**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx tests/A4Canvas.test.tsx tests/ContextMenu.test.tsx`
Expected: All pass

**Step 5: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/src/components/editor/A4Canvas.tsx
git commit -m "feat: integrate ContextMenu into editor with right-click handling"
```

---

### Task 6: Create BubbleToolbar component

**Files:**
- Create: `frontend/workbench/src/components/editor/BubbleToolbar.tsx`

**Step 1: Write the BubbleToolbar component**

```tsx
// src/components/editor/BubbleToolbar.tsx
import { useState, useEffect } from 'react'
import { Bold, Italic, Underline, List, ListOrdered } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { ToolbarButton } from './ToolbarButton'
import { FontSelector } from './FontSelector'
import { FontSizeSelector } from './FontSizeSelector'
import { ColorPicker } from './ColorPicker'
import { LineHeightSelector } from './LineHeightSelector'
import { AlignSelector } from './AlignSelector'

interface ActiveStates {
  isBold: boolean
  isItalic: boolean
  isUnderline: boolean
  isBulletList: boolean
  isOrderedList: boolean
}

function getActiveStates(editor: Editor): ActiveStates {
  return {
    isBold: editor.isActive('bold'),
    isItalic: editor.isActive('italic'),
    isUnderline: editor.isActive('underline'),
    isBulletList: editor.isActive('bulletList'),
    isOrderedList: editor.isActive('orderedList'),
  }
}

interface BubbleToolbarProps {
  editor: Editor
}

export function BubbleToolbar({ editor }: BubbleToolbarProps) {
  const [activeStates, setActiveStates] = useState<ActiveStates | null>(null)

  useEffect(() => {
    if (!editor) return

    const update = () => setActiveStates(getActiveStates(editor))
    update()
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  if (!editor || !activeStates) return null

  return (
    <div className="flex items-center gap-2 bg-white border border-border rounded-lg shadow-sm px-1 py-0.5">
      {/* Font & Size group */}
      <div className="flex items-center gap-0.5">
        <FontSelector editor={editor} />
        <FontSizeSelector editor={editor} />
      </div>

      {/* Text Format + Color group */}
      <div className="flex items-center gap-0.5">
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleBold().run()}
          isActive={activeStates.isBold}
          icon={<Bold size={16} />}
          label="粗体"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleItalic().run()}
          isActive={activeStates.isItalic}
          icon={<Italic size={16} />}
          label="斜体"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleUnderline().run()}
          isActive={activeStates.isUnderline}
          icon={<Underline size={16} />}
          label="下划线"
        />
        <ColorPicker editor={editor} />
      </div>

      {/* List group */}
      <div className="flex items-center gap-0.5">
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          isActive={activeStates.isBulletList}
          icon={<List size={16} />}
          label="无序列表"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          isActive={activeStates.isOrderedList}
          icon={<ListOrdered size={16} />}
          label="有序列表"
        />
      </div>

      {/* Alignment + Line Height group */}
      <div className="flex items-center gap-0.5">
        <AlignSelector editor={editor} />
        <LineHeightSelector editor={editor} />
      </div>
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/workbench/src/components/editor/BubbleToolbar.tsx
git commit -m "feat: add BubbleToolbar component with compact formatting layout"
```

---

### Task 7: Integrate BubbleMenu into EditorPage

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`

**Step 1: Add BubbleMenu to EditorPage**

In `EditorPage.tsx`:

1. Import BubbleMenu extension and React component:
```tsx
import { BubbleMenu } from '@tiptap/extension-bubble-menu'
```

2. Import BubbleToolbar:
```tsx
import { BubbleToolbar } from '@/components/editor/BubbleToolbar'
```

3. Add `BubbleMenu` to the extensions array:
```tsx
extensions: [
  StarterKit,
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
  BubbleMenu.configure({
    element: document.createElement('div'), // placeholder, actual rendering via React <BubbleMenu>
  }),
],
```

Wait — actually `@tiptap/extension-bubble-menu` provides both an extension AND a React component `<BubbleMenu>`. The correct approach is:

1. Import the React component `BubbleMenu` from `@tiptap/react` (it re-exports it when the extension is installed)
2. Do NOT add the extension to the extensions array — the `<BubbleMenu>` React component handles everything internally

Correction: Check `@tiptap/react` exports. The `<BubbleMenu>` component from `@tiptap/react` auto-manages the extension. We only need to add it as JSX.

Updated approach:

1. Import only:
```tsx
import { BubbleMenu } from '@tiptap/react'
```

2. No change to extensions array — `<BubbleMenu>` component auto-registers.

3. Add `<BubbleMenu>` JSX in the center panel, right before the A4Canvas:
```tsx
<BubbleMenu
  editor={editor}
  tippyOptions={{
    duration: 150,
    placement: 'top',
    arrow: false,
  }}
  shouldShow={({ editor }) => {
    const { from, to } = editor.state.selection
    return from !== to
  }}
>
  <BubbleToolbar editor={editor} />
</BubbleMenu>
```

**Step 2: Verify build**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds

**Step 3: Run all tests**

Run: `cd frontend/workbench && bunx vitest run`
Expected: All pass

**Step 4: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat: integrate BubbleMenu with compact formatting toolbar on text selection"
```

---

### Task 8: Final verification — run all tests

**Step 1: Run full test suite**

Run: `cd frontend/workbench && bunx vitest run`
Expected: All tests pass

**Step 2: Run build**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds with no warnings

**Step 3: Manual smoke test**

Run: `cd frontend/workbench && bun run dev`

Verify:
1. Right-click inside editor → context menu appears with undo/redo/cut/copy/paste/select-all
2. Click outside menu or press Esc → menu closes
3. Click undo/redo → editor state changes
4. Select text → BubbleMenu appears above selection
5. Use BubbleMenu to change font, bold, etc. → formatting applies
6. Bottom FormatToolbar alignment is now a dropdown → works correctly
7. Copy from context menu → pastes plain text only (content protection)
8. No regressions in existing toolbar functionality
