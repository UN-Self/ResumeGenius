# TipTap 排版功能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 TipTap 编辑器新增字体、字号、颜色（字色+高亮色）、行距选择器和右对齐按钮。

**Architecture:** 使用 `TextStyleKit` 一行注册全部扩展，4 个独立 Popover 选择器组件挂载到 FormatToolbar。

**Tech Stack:** TipTap v3, @tiptap/extension-text-style, shadcn/ui Popover, lucide-react

**Design doc:** `docs/plans/2026-04-30-tiptap-typography-design.md`

---

### Task 1: 安装 shadcn/ui Popover 组件

**Files:**
- Create: `frontend/workbench/src/components/ui/popover.tsx`
- Read: `frontend/workbench/components.json`

**Step 1: 安装 Popover**

Run: `cd frontend/workbench && bunx shadcn@latest add popover`

Expected: `src/components/ui/popover.tsx` created, `@radix-ui/react-popover` added to dependencies.

**Step 2: 验证文件存在**

Run: `ls frontend/workbench/src/components/ui/popover.tsx`

Expected: file exists.

**Step 3: Commit**

```bash
git add frontend/workbench/src/components/ui/popover.tsx frontend/workbench/package.json frontend/workbench/bun.lock
git commit -m "feat(editor): add shadcn/ui Popover component"
```

---

### Task 2: 注册 TextStyleKit 扩展

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:1-48`
- Test: `frontend/workbench/tests/TipTapEditor.test.tsx`

**Step 1: 更新测试中的 mock editor 和 TestEditorWrapper**

在 `tests/TipTapEditor.test.tsx` 中：

1. 添加 import：
```ts
import { TextStyleKit } from '@tiptap/extension-text-style'
```

2. 在 `TestEditorWrapper` 和 `TestToolbarWrapper` 的 extensions 数组末尾添加 `TextStyleKit`：

```ts
extensions: [
  StarterKit,
  Underline,
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
],
```

3. 扩展 `createMockEditor`，在 `focusMock` 中添加 text style 命令：

```ts
const focusMock = () => ({
  toggleBold: () => ({ run: runMock }),
  toggleItalic: () => ({ run: runMock }),
  toggleUnderline: () => ({ run: runMock }),
  toggleHeading: () => ({ run: runMock }),
  toggleBulletList: () => ({ run: runMock }),
  toggleOrderedList: () => ({ run: runMock }),
  setTextAlign: () => ({ run: runMock }),
  setFontFamily: () => ({ run: runMock }),
  setFontSize: () => ({ run: runMock }),
  setColor: () => ({ run: runMock }),
  setBackgroundColor: () => ({ run: runMock }),
  setLineHeight: () => ({ run: runMock }),
  unsetFontFamily: () => ({ run: runMock }),
  unsetFontSize: () => ({ run: runMock }),
  unsetColor: () => ({ run: runMock }),
  unsetBackgroundColor: () => ({ run: runMock }),
  unsetLineHeight: () => ({ run: runMock }),
})
```

4. 添加 `getAttributes` mock 到 mock editor：

```ts
getAttributes: vi.fn(() => ({})),
```

**Step 2: 运行现有测试确认不回归**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx`

Expected: All existing tests PASS.

**Step 3: 在 EditorPage.tsx 中注册 TextStyleKit**

在 `frontend/workbench/src/pages/EditorPage.tsx` 中：

1. 添加 import：
```ts
import { TextStyleKit } from '@tiptap/extension-text-style'
```

2. 在 extensions 数组末尾添加 `TextStyleKit`：
```ts
extensions: [
  StarterKit,
  Underline,
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
],
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx`

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/tests/TipTapEditor.test.tsx
git commit -m "feat(editor): register TextStyleKit extension for typography features"
```

---

### Task 3: 添加右对齐按钮

**Files:**
- Modify: `frontend/workbench/src/components/editor/FormatToolbar.tsx:1-35`
- Test: `frontend/workbench/tests/TipTapEditor.test.tsx`

**Step 1: 写失败测试 — 验证右对齐按钮存在**

在 `tests/TipTapEditor.test.tsx` 的 `FormatToolbar` describe 块内，在 `'renders all toolbar buttons'` test 中添加：

```ts
expect(screen.getByRole('button', { name: /右对齐/i })).toBeInTheDocument()
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx -t "renders all toolbar buttons"`

Expected: FAIL — 找不到"右对齐"按钮。

**Step 3: 实现 — 添加右对齐按钮到 FormatToolbar**

在 `frontend/workbench/src/components/editor/FormatToolbar.tsx` 中：

1. 添加 `AlignRight` 到 lucide-react import：
```ts
import { Bold, Italic, Underline, Heading1, Heading2, Heading3, List, ListOrdered, AlignLeft, AlignCenter, AlignRight, AlignJustify } from 'lucide-react'
```

2. 在 `ActiveStates` interface 中添加：
```ts
isAlignRight: boolean
```

3. 在 `getActiveStates` 中添加：
```ts
isAlignRight: editor.isActive({ textAlign: 'right' }),
```

4. 在对齐组 JSX 中，居中按钮和两端对齐按钮之间插入右对齐按钮：
```tsx
<ToolbarButton
  onClick={() => editor.chain().focus().setTextAlign('right').run()}
  isActive={activeStates.isAlignRight}
  icon={<AlignRight size={20} />}
  label="右对齐"
/>
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx`

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/FormatToolbar.tsx frontend/workbench/tests/TipTapEditor.test.tsx
git commit -m "feat(editor): add right-align button to format toolbar"
```

---

### Task 4: FontSelector 组件

**Files:**
- Create: `frontend/workbench/src/components/editor/FontSelector.tsx`
- Create: `frontend/workbench/tests/FontSelector.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/FontSelector.test.tsx`：

```ts
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FontSelector } from '@/components/editor/FontSelector'

const createMockEditor = () => {
  const runMock = vi.fn()
  const focusMock = () => ({
    setFontFamily: () => ({ run: runMock }),
    unsetFontFamily: () => ({ run: runMock }),
  })
  const listeners = new Map<string, Set<() => void>>()
  return {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(() => ({ fontFamily: null })),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn(),
    runMock,
  } as any
}

describe('FontSelector', () => {
  it('renders trigger button with default text', () => {
    const editor = createMockEditor()
    render(<FontSelector editor={editor} />)
    expect(screen.getByRole('button', { name: /字体/i })).toBeInTheDocument()
  })

  it('shows font list when clicked', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<FontSelector editor={editor} />)

    await user.click(screen.getByRole('button', { name: /字体/i }))
    expect(screen.getByText('宋体')).toBeInTheDocument()
    expect(screen.getByText('Arial')).toBeInTheDocument()
  })

  it('calls setFontFamily when a font is selected', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<FontSelector editor={editor} />)

    await user.click(screen.getByRole('button', { name: /字体/i }))
    await user.click(screen.getByText('黑体'))

    expect(editor.runMock).toHaveBeenCalled()
  })

  it('displays current font name when set', () => {
    const editor = createMockEditor()
    editor.getAttributes = vi.fn(() => ({ fontFamily: 'SimHei, sans-serif' }))
    render(<FontSelector editor={editor} />)
    expect(screen.getByRole('button', { name: /字体/i })).toHaveTextContent('黑体')
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/FontSelector.test.tsx`

Expected: FAIL — `FontSelector` module not found.

**Step 3: 实现 FontSelector 组件**

创建 `frontend/workbench/src/components/editor/FontSelector.tsx`：

```tsx
import { useState, useEffect, useCallback } from 'react'
import type { Editor } from '@tiptap/react'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'
import { ChevronDown } from 'lucide-react'

interface FontSelectorProps {
  editor: Editor
}

const FONTS = [
  { label: '默认字体', value: '' },
  { label: '宋体', value: 'SimSun, serif' },
  { label: '黑体', value: 'SimHei, sans-serif' },
  { label: '楷体', value: 'KaiTi, serif' },
  { label: '仿宋', value: 'FangSong, serif' },
  { label: 'Times New Roman', value: '"Times New Roman", serif' },
  { label: 'Arial', value: 'Arial, sans-serif' },
  { label: 'Georgia', value: 'Georgia, serif' },
] as const

function getFontLabel(value: string | null | undefined): string {
  if (!value) return '字体'
  return FONTS.find(f => f.value === value)?.label ?? '字体'
}

export function FontSelector({ editor }: FontSelectorProps) {
  const [currentFont, setCurrentFont] = useState<string | null>(null)

  useEffect(() => {
    const update = () => {
      setCurrentFont(editor.getAttributes('textStyle').fontFamily ?? null)
    }
    update()
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  const handleSelect = useCallback((value: string) => {
    if (value) {
      editor.chain().focus().setFontFamily(value).run()
    } else {
      editor.chain().focus().unsetFontFamily().run()
    }
  }, [editor])

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="字体"
          className="flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
        >
          <span className="max-w-[80px] truncate">{getFontLabel(currentFont)}</span>
          <ChevronDown size={14} />
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" align="start" className="w-40 p-1">
        {FONTS.map(font => (
          <button
            key={font.value || 'default'}
            type="button"
            onClick={() => handleSelect(font.value)}
            className="w-full text-left px-2 py-1.5 text-sm rounded hover:bg-[#f8f9fa] transition-colors"
            style={{ fontFamily: font.value || undefined }}
          >
            {font.label}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/FontSelector.test.tsx`

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/FontSelector.tsx frontend/workbench/tests/FontSelector.test.tsx
git commit -m "feat(editor): add FontSelector component with 8 font options"
```

---

### Task 5: FontSizeSelector 组件

**Files:**
- Create: `frontend/workbench/src/components/editor/FontSizeSelector.tsx`
- Create: `frontend/workbench/tests/FontSizeSelector.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/FontSizeSelector.test.tsx`：

```ts
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FontSizeSelector } from '@/components/editor/FontSizeSelector'

const createMockEditor = () => {
  const runMock = vi.fn()
  const focusMock = () => ({
    setFontSize: () => ({ run: runMock }),
    unsetFontSize: () => ({ run: runMock }),
  })
  const listeners = new Map<string, Set<() => void>>()
  return {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(() => ({ fontSize: null })),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn(),
    runMock,
  } as any
}

describe('FontSizeSelector', () => {
  it('renders trigger button with default text', () => {
    const editor = createMockEditor()
    render(<FontSizeSelector editor={editor} />)
    expect(screen.getByRole('button', { name: /字号/i })).toBeInTheDocument()
  })

  it('shows size list when clicked', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<FontSizeSelector editor={editor} />)

    await user.click(screen.getByRole('button', { name: /字号/i }))
    expect(screen.getByText('12pt')).toBeInTheDocument()
    expect(screen.getByText('24pt')).toBeInTheDocument()
  })

  it('calls setFontSize when a size is selected', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<FontSizeSelector editor={editor} />)

    await user.click(screen.getByRole('button', { name: /字号/i }))
    await user.click(screen.getByText('16pt'))

    expect(editor.runMock).toHaveBeenCalled()
  })

  it('displays current size when set', () => {
    const editor = createMockEditor()
    editor.getAttributes = vi.fn(() => ({ fontSize: '14pt' }))
    render(<FontSizeSelector editor={editor} />)
    expect(screen.getByRole('button', { name: /字号/i })).toHaveTextContent('14pt')
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/FontSizeSelector.test.tsx`

Expected: FAIL — `FontSizeSelector` module not found.

**Step 3: 实现 FontSizeSelector 组件**

创建 `frontend/workbench/src/components/editor/FontSizeSelector.tsx`：

```tsx
import { useState, useEffect, useCallback } from 'react'
import type { Editor } from '@tiptap/react'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'
import { ChevronDown } from 'lucide-react'

interface FontSizeSelectorProps {
  editor: Editor
}

const SIZES = ['10pt', '12pt', '14pt', '16pt', '18pt', '24pt'] as const

export function FontSizeSelector({ editor }: FontSizeSelectorProps) {
  const [currentSize, setCurrentSize] = useState<string | null>(null)

  useEffect(() => {
    const update = () => {
      setCurrentSize(editor.getAttributes('textStyle').fontSize ?? null)
    }
    update()
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  const handleSelect = useCallback((value: string) => {
    editor.chain().focus().setFontSize(value).run()
  }, [editor])

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="字号"
          className="flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
        >
          <span className="w-[36px] text-center">{currentSize ?? '12'}</span>
          <ChevronDown size={14} />
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" align="start" className="w-24 p-1">
        {SIZES.map(size => (
          <button
            key={size}
            type="button"
            onClick={() => handleSelect(size)}
            className={`w-full text-left px-2 py-1.5 text-sm rounded hover:bg-[#f8f9fa] transition-colors ${
              currentSize === size ? 'bg-[#e8f0fe] text-[#1a73e8]' : ''
            }`}
          >
            {size}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/FontSizeSelector.test.tsx`

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/FontSizeSelector.tsx frontend/workbench/tests/FontSizeSelector.test.tsx
git commit -m "feat(editor): add FontSizeSelector component with 6 size options"
```

---

### Task 6: ColorPicker 组件

**Files:**
- Create: `frontend/workbench/src/components/editor/ColorPicker.tsx`
- Create: `frontend/workbench/tests/ColorPicker.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/ColorPicker.test.tsx`：

```ts
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ColorPicker } from '@/components/editor/ColorPicker'

const createMockEditor = () => {
  const runMock = vi.fn()
  const focusMock = () => ({
    setColor: () => ({ run: runMock }),
    unsetColor: () => ({ run: runMock }),
    setBackgroundColor: () => ({ run: runMock }),
    unsetBackgroundColor: () => ({ run: runMock }),
  })
  const listeners = new Map<string, Set<() => void>>()
  return {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(() => ({})),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn(),
    runMock,
  } as any
}

describe('ColorPicker', () => {
  it('renders both color trigger buttons', () => {
    const editor = createMockEditor()
    render(<ColorPicker editor={editor} />)
    expect(screen.getByRole('button', { name: /字体颜色/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /高亮颜色/i })).toBeInTheDocument()
  })

  it('shows color swatches when font color button is clicked', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<ColorPicker editor={editor} />)

    await user.click(screen.getByRole('button', { name: /字体颜色/i }))
    // Should show preset color swatches (18 colors + custom input)
    expect(screen.getAllByRole('button').length).toBeGreaterThan(2)
  })

  it('calls setColor when a preset color is selected', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<ColorPicker editor={editor} />)

    await user.click(screen.getByRole('button', { name: /字体颜色/i }))
    // First swatch button (excluding the trigger)
    const swatches = screen.getAllByRole('button').filter(b => b.getAttribute('aria-label') !== '字体颜色' && b.getAttribute('aria-label') !== '高亮颜色')
    await user.click(swatches[0])

    expect(editor.runMock).toHaveBeenCalled()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/ColorPicker.test.tsx`

Expected: FAIL — `ColorPicker` module not found.

**Step 3: 实现 ColorPicker 组件**

创建 `frontend/workbench/src/components/editor/ColorPicker.tsx`：

```tsx
import { useState, useEffect, useCallback } from 'react'
import type { Editor } from '@tiptap/react'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'
import { Type, Highlighter } from 'lucide-react'

interface ColorPickerProps {
  editor: Editor
}

const PRESET_COLORS = [
  '#000000', '#434343', '#666666', '#999999', '#b7b7b7', '#cccccc',
  '#d9d9d9', '#efefef', '#f3f3f3', '#ffffff',
  '#e06666', '#f6b26b', '#ffd966', '#93c47d', '#76a5af', '#6fa8dc',
  '#8e7cc3', '#c27ba0',
] as const

type ColorTarget = 'font' | 'background'

export function ColorPicker({ editor }: ColorPickerProps) {
  const [fontColor, setFontColor] = useState<string | null>(null)
  const [bgColor, setBgColor] = useState<string | null>(null)
  const [target, setTarget] = useState<ColorTarget | null>(null)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const update = () => {
      const attrs = editor.getAttributes('textStyle')
      setFontColor(attrs.color ?? null)
      setBgColor(attrs.backgroundColor ?? null)
    }
    update()
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  const handleOpen = useCallback((t: ColorTarget) => {
    setTarget(t)
    setOpen(true)
  }, [])

  const handleColorSelect = useCallback((color: string) => {
    if (!target) return
    if (target === 'font') {
      editor.chain().focus().setColor(color).run()
    } else {
      editor.chain().focus().setBackgroundColor(color).run()
    }
    setOpen(false)
  }, [editor, target])

  const handleCustomColor = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    handleColorSelect(e.target.value)
  }, [handleColorSelect])

  const handleUnset = useCallback(() => {
    if (!target) return
    if (target === 'font') {
      editor.chain().focus().unsetColor().run()
    } else {
      editor.chain().focus().unsetBackgroundColor().run()
    }
    setOpen(false)
  }, [editor, target])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <div className="flex items-center gap-1">
        {/* Font color button */}
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label="字体颜色"
            onClick={() => handleOpen('font')}
            className="relative p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-md text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
          >
            <Type size={20} />
            {fontColor && (
              <span
                className="absolute bottom-1 left-1/2 -translate-x-1/2 w-4 h-1 rounded-full"
                style={{ backgroundColor: fontColor }}
              />
            )}
          </button>
        </PopoverTrigger>

        {/* Highlight color button */}
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label="高亮颜色"
            onClick={() => handleOpen('background')}
            className="relative p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-md text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
          >
            <Highlighter size={20} />
            {bgColor && (
              <span
                className="absolute bottom-1 left-1/2 -translate-x-1/2 w-4 h-1 rounded-full"
                style={{ backgroundColor: bgColor }}
              />
            )}
          </button>
        </PopoverTrigger>
      </div>

      <PopoverContent side="top" align="start" className="w-auto p-2">
        <div className="grid grid-cols-9 gap-1">
          {PRESET_COLORS.map(color => (
            <button
              key={color}
              type="button"
              aria-label={color}
              onClick={() => handleColorSelect(color)}
              className="w-6 h-6 rounded border border-[#dadce0] hover:scale-110 transition-transform"
              style={{ backgroundColor: color }}
            />
          ))}
        </div>
        <div className="mt-2 flex items-center gap-2">
          <button
            type="button"
            onClick={handleUnset}
            className="text-xs text-[#5f6368] hover:text-[#1a73e8] px-1"
          >
            重置
          </button>
          <label className="flex items-center gap-1 text-xs text-[#5f6368] cursor-pointer">
            自定义
            <input
              type="color"
              onChange={handleCustomColor}
              className="w-5 h-5 cursor-pointer border-0 p-0"
            />
          </label>
        </div>
      </PopoverContent>
    </Popover>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/ColorPicker.test.tsx`

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/ColorPicker.tsx frontend/workbench/tests/ColorPicker.test.tsx
git commit -m "feat(editor): add ColorPicker component with font and highlight color"
```

---

### Task 7: LineHeightSelector 组件

**Files:**
- Create: `frontend/workbench/src/components/editor/LineHeightSelector.tsx`
- Create: `frontend/workbench/tests/LineHeightSelector.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/LineHeightSelector.test.tsx`：

```ts
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LineHeightSelector } from '@/components/editor/LineHeightSelector'

const createMockEditor = () => {
  const runMock = vi.fn()
  const focusMock = () => ({
    setLineHeight: () => ({ run: runMock }),
    unsetLineHeight: () => ({ run: runMock }),
  })
  const listeners = new Map<string, Set<() => void>>()
  return {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(() => ({ lineHeight: null })),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn(),
    runMock,
  } as any
}

describe('LineHeightSelector', () => {
  it('renders trigger button', () => {
    const editor = createMockEditor()
    render(<LineHeightSelector editor={editor} />)
    expect(screen.getByRole('button', { name: /行距/i })).toBeInTheDocument()
  })

  it('shows line height options when clicked', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<LineHeightSelector editor={editor} />)

    await user.click(screen.getByRole('button', { name: /行距/i }))
    expect(screen.getByText('1.5')).toBeInTheDocument()
    expect(screen.getByText('2.0')).toBeInTheDocument()
  })

  it('calls setLineHeight when an option is selected', async () => {
    const user = userEvent.setup()
    const editor = createMockEditor()
    render(<LineHeightSelector editor={editor} />)

    await user.click(screen.getByRole('button', { name: /行距/i }))
    await user.click(screen.getByText('1.5'))

    expect(editor.runMock).toHaveBeenCalled()
  })

  it('displays current line height when set', () => {
    const editor = createMockEditor()
    editor.getAttributes = vi.fn(() => ({ lineHeight: '1.5' }))
    render(<LineHeightSelector editor={editor} />)
    expect(screen.getByRole('button', { name: /行距/i })).toHaveTextContent('1.5')
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/LineHeightSelector.test.tsx`

Expected: FAIL — `LineHeightSelector` module not found.

**Step 3: 实现 LineHeightSelector 组件**

创建 `frontend/workbench/src/components/editor/LineHeightSelector.tsx`：

```tsx
import { useState, useEffect, useCallback } from 'react'
import type { Editor } from '@tiptap/react'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'
import { ChevronDown } from 'lucide-react'

interface LineHeightSelectorProps {
  editor: Editor
}

const LINE_HEIGHTS = ['1.0', '1.15', '1.5', '1.75', '2.0', '2.5'] as const

export function LineHeightSelector({ editor }: LineHeightSelectorProps) {
  const [currentLineHeight, setCurrentLineHeight] = useState<string | null>(null)

  useEffect(() => {
    const update = () => {
      setCurrentLineHeight(editor.getAttributes('textStyle').lineHeight ?? null)
    }
    update()
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  const handleSelect = useCallback((value: string) => {
    editor.chain().focus().setLineHeight(value).run()
  }, [editor])

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="行距"
          className="flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
        >
          <span className="w-[28px] text-center">{currentLineHeight ?? '—'}</span>
          <ChevronDown size={14} />
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" align="start" className="w-24 p-1">
        {LINE_HEIGHTS.map(height => (
          <button
            key={height}
            type="button"
            onClick={() => handleSelect(height)}
            className={`w-full text-left px-2 py-1.5 text-sm rounded hover:bg-[#f8f9fa] transition-colors ${
              currentLineHeight === height ? 'bg-[#e8f0fe] text-[#1a73e8]' : ''
            }`}
          >
            {height}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/LineHeightSelector.test.tsx`

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/LineHeightSelector.tsx frontend/workbench/tests/LineHeightSelector.test.tsx
git commit -m "feat(editor): add LineHeightSelector component with 6 options"
```

---

### Task 8: 集成全部选择器到 FormatToolbar

**Files:**
- Modify: `frontend/workbench/src/components/editor/FormatToolbar.tsx`
- Modify: `frontend/workbench/tests/TipTapEditor.test.tsx`

**Step 1: 更新 FormatToolbar**

在 `frontend/workbench/src/components/editor/FormatToolbar.tsx` 中：

1. 添加 imports：
```ts
import { FontSelector } from './FontSelector'
import { FontSizeSelector } from './FontSizeSelector'
import { ColorPicker } from './ColorPicker'
import { LineHeightSelector } from './LineHeightSelector'
```

2. 在 return JSX 的最前面，Bold/Italic/Underline group 之前，插入字体和字号组：

```tsx
{/* Font & Size group */}
<div role="group" aria-label="字体和字号" className="flex items-center gap-1">
  <FontSelector editor={editor} />
  <FontSizeSelector editor={editor} />
</div>

<ToolbarSeparator />
```

3. 在 Bold/Italic/Underline group 和 Heading group 之间插入颜色选择器：

```tsx
{/* After Bold/Italic/Underline closing </div>, before <ToolbarSeparator /> for heading */}
<ColorPicker editor={editor} />
```

4. 在 List group 和 Alignment group 之间插入行距选择器：

```tsx
{/* After List group closing </div> and its <ToolbarSeparator /> */}
<div role="group" aria-label="行距" className="flex items-center gap-1">
  <LineHeightSelector editor={editor} />
</div>

<ToolbarSeparator />
```

5. 移除旧的 Bold/Italic/Underline 和 Heading 之间的 `<ToolbarSeparator />`（因为现在它们之间有 ColorPicker）。最终布局应该是：

```
[字体▾ | 字号▾] ─ [B I U | A色 🖍高亮] ─ [H1 H2 H3] ─ [·列表 1.列表] ─ [行距▾ | ←左 | ↗居中 | →右 | ≡两端]
```

**Step 2: 更新测试以覆盖新增组件**

在 `tests/TipTapEditor.test.tsx` 中：

1. 在 `TestToolbarWrapper` 的 extensions 数组中确保包含 `TextStyleKit`（Task 2 已做）。

2. 添加一个新的集成测试，验证全部 toolbar 按钮和选择器都渲染：

```ts
it('renders all selectors and toolbar buttons', async () => {
  let editorRef: Editor | null = null

  render(
    <TestToolbarWrapper
      content="<p>Hello</p>"
      onEditor={(e) => { editorRef = e }}
    />,
  )

  // Selectors
  expect(await screen.findByRole('button', { name: /字体/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /字号/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /字体颜色/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /高亮颜色/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /行距/i })).toBeInTheDocument()

  // Alignment (including new right-align)
  expect(screen.getByRole('button', { name: /右对齐/i })).toBeInTheDocument()
})
```

**Step 3: 运行全部测试**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx tests/FontSelector.test.tsx tests/FontSizeSelector.test.tsx tests/ColorPicker.test.tsx tests/LineHeightSelector.test.tsx`

Expected: All tests PASS.

**Step 4: 构建验证**

Run: `cd frontend/workbench && bun run build`

Expected: Build succeeds without errors.

**Step 5: Commit**

```bash
git add frontend/workbench/src/components/editor/FormatToolbar.tsx frontend/workbench/tests/TipTapEditor.test.tsx
git commit -m "feat(editor): integrate all typography selectors into FormatToolbar"
```

---

### Task 9: 运行全量测试 + 构建验证

**Step 1: 运行全部前端测试**

Run: `cd frontend/workbench && bunx vitest run`

Expected: All tests PASS.

**Step 2: 运行生产构建**

Run: `cd frontend/workbench && bun run build`

Expected: Build succeeds without errors.

**Step 3: （可选）启动开发服务器手动验证**

Run: `cd frontend/workbench && bun run dev`

Manual verification checklist:
- [ ] 字体选择器弹出、选择字体、光标处文本字体变化
- [ ] 字号选择器弹出、选择字号、光标处文本大小变化
- [ ] 字体颜色选择器弹出、选择颜色、光标处文本颜色变化
- [ ] 高亮颜色选择器弹出、选择颜色、光标处文本背景色变化
- [ ] 行距选择器弹出、选择行距、光标处段落行距变化
- [ ] 右对齐按钮点击、段落右对齐
- [ ] 各选择器 Popover 向上弹出（side="top"）
- [ ] 点击外部关闭 Popover
