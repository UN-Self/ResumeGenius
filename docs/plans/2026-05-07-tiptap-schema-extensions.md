# TipTap Schema 扩展实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 扩展 TipTap schema 使其保留 AI 生成的自定义 HTML 元素（div/section/header/span）和 class 属性，让 scoped CSS 能正确匹配 DOM 元素。

**Architecture:** 三个 TipTap 扩展：`PresetAttributes`（全局 class/style 属性）、`Div`（块级容器节点）、`Span`（行内 span 标记）。全部注册到 EditorPage 的编辑器 extensions 中。

**Tech Stack:** TipTap 3.22.x, ProseMirror, Vitest, jsdom

---

### Task 1: PresetAttributes 扩展

**Files:**
- Create: `frontend/workbench/src/components/editor/extensions/PresetAttributes.ts`
- Test: `frontend/workbench/tests/extensions/preset-attributes.test.ts`

**Step 1: 写失败测试**

```typescript
// tests/extensions/preset-attributes.test.ts
import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { PresetAttributes } from '@/components/editor/extensions/PresetAttributes'

function createEditor(content = '') {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), PresetAttributes],
    content,
  })
}

describe('PresetAttributes', () => {
  it('preserves class attribute on paragraph', () => {
    const editor = createEditor('<p class="intro">Hello</p>')
    expect(editor.getHTML()).toContain('class="intro"')
    editor.destroy()
  })

  it('preserves class attribute on heading', () => {
    const editor = createEditor('<h2 class="section-title">Title</h2>')
    expect(editor.getHTML()).toContain('class="section-title"')
    editor.destroy()
  })

  it('preserves class attribute on list item', () => {
    const editor = createEditor('<ul><li class="item">Item</li></ul>')
    expect(editor.getHTML()).toContain('class="item"')
    editor.destroy()
  })

  it('preserves style attribute on paragraph', () => {
    const editor = createEditor('<p style="color: red">Red text</p>')
    expect(editor.getHTML()).toContain('style="color: red"')
    editor.destroy()
  })

  it('does not add attributes when none present', () => {
    const editor = createEditor('<p>Plain text</p>')
    expect(editor.getHTML()).toBe('<p>Plain text</p>')
    editor.destroy()
  })

  it('preserves both class and style on the same element', () => {
    const editor = createEditor('<p class="highlight" style="font-size: 14px">Text</p>')
    expect(editor.getHTML()).toContain('class="highlight"')
    expect(editor.getHTML()).toContain('style="font-size: 14px"')
    editor.destroy()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/preset-attributes.test.ts`
Expected: FAIL — cannot find module `@/components/editor/extensions/PresetAttributes`

**Step 3: 实现 PresetAttributes 扩展**

```typescript
// src/components/editor/extensions/PresetAttributes.ts
import { Extension } from '@tiptap/core'

/**
 * Adds `class` and `style` attributes to standard block nodes
 * so that AI-generated HTML class attributes survive ProseMirror parsing.
 * Uses the same addGlobalAttributes pattern as TextAlign.
 */
export const PresetAttributes = Extension.create({
  name: 'presetAttributes',

  addGlobalAttributes() {
    return [
      {
        types: [
          'paragraph',
          'heading',
          'listItem',
          'bulletList',
          'orderedList',
          'blockquote',
          'codeBlock',
        ],
        attributes: {
          class: {
            default: null,
            parseHTML: (element: HTMLElement) => element.getAttribute('class'),
            renderHTML: (attributes: Record<string, string>) => {
              if (!attributes.class) return {}
              return { class: attributes.class }
            },
          },
          style: {
            default: null,
            parseHTML: (element: HTMLElement) => element.getAttribute('style'),
            renderHTML: (attributes: Record<string, string>) => {
              if (!attributes.style) return {}
              return { style: attributes.style }
            },
          },
        },
      },
    ]
  },
})
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/preset-attributes.test.ts`
Expected: PASS (6 tests)

---

### Task 2: Div 节点扩展

**Files:**
- Create: `frontend/workbench/src/components/editor/extensions/Div.ts`
- Test: `frontend/workbench/tests/extensions/div.test.ts`

**Step 1: 写失败测试**

```typescript
// tests/extensions/div.test.ts
import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Div } from '@/components/editor/extensions/Div'
import { PresetAttributes } from '@/components/editor/extensions/PresetAttributes'

function createEditor(content = '') {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), Div, PresetAttributes],
    content,
  })
}

describe('Div extension', () => {
  it('preserves div with class attribute', () => {
    const editor = createEditor('<div class="resume">Hello</div>')
    const html = editor.getHTML()
    expect(html).toContain('<div')
    expect(html).toContain('class="resume"')
    editor.destroy()
  })

  it('preserves div with style attribute', () => {
    const editor = createEditor('<div style="color: red">Hello</div>')
    const html = editor.getHTML()
    expect(html).toContain('<div')
    expect(html).toContain('style="color: red"')
    editor.destroy()
  })

  it('preserves nested divs', () => {
    const editor = createEditor(
      '<div class="outer"><div class="inner">Text</div></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="outer"')
    expect(html).toContain('class="inner"')
    editor.destroy()
  })

  it('wraps bare text in paragraph inside div', () => {
    const editor = createEditor('<div class="box">Plain text</div>')
    const html = editor.getHTML()
    expect(html).toContain('<p>Plain text</p>')
    editor.destroy()
  })

  it('preserves section tag as section (not div)', () => {
    const editor = createEditor('<section class="section"><p>Content</p></section>')
    const html = editor.getHTML()
    expect(html).toContain('<section')
    expect(html).toContain('class="section"')
    editor.destroy()
  })

  it('preserves header tag as header', () => {
    const editor = createEditor('<header class="profile"><p>Name</p></header>')
    const html = editor.getHTML()
    expect(html).toContain('<header')
    expect(html).toContain('class="profile"')
    editor.destroy()
  })

  it('handles div containing a list', () => {
    const editor = createEditor(
      '<div class="item"><ul><li>One</li><li>Two</li></ul></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="item"')
    expect(html).toContain('<li>One</li>')
    expect(html).toContain('<li>Two</li>')
    editor.destroy()
  })

  it('handles div containing headings', () => {
    const editor = createEditor(
      '<div class="section"><h2>Title</h2></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="section"')
    expect(html).toContain('<h2>Title</h2>')
    editor.destroy()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/div.test.ts`
Expected: FAIL — cannot find module `@/components/editor/extensions/Div`

**Step 3: 实现 Div 扩展**

```typescript
// src/components/editor/extensions/Div.ts
import { Node, mergeAttributes } from '@tiptap/core'

const CONTAINER_TAGS = ['div', 'section', 'header', 'footer', 'main', 'article', 'nav', 'aside'] as const

/**
 * Block container node that preserves <div>, <section>, <header>, etc.
 * with their class and style attributes through ProseMirror parsing.
 * Renders as the original tag name (e.g., <section> stays <section>).
 */
export const Div = Node.create({
  name: 'div',
  group: 'block',
  content: 'block*',
  selectable: false,
  draggable: false,

  parseHTML() {
    return CONTAINER_TAGS.map((tag) => ({
      tag,
      getAttrs: (element: HTMLElement) => ({
        originalTag: element.tagName.toLowerCase(),
      }),
    }))
  },

  renderHTML({ HTMLAttributes }) {
    const tag = HTMLAttributes.originalTag || 'div'
    const { originalTag: _omit, ...rest } = HTMLAttributes
    return [tag, mergeAttributes(rest), 0]
  },

  addAttributes() {
    return {
      originalTag: {
        default: 'div',
        parseHTML: (element: HTMLElement) => element.tagName.toLowerCase(),
        renderHTML: () => ({}),
      },
      class: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('class'),
      },
      style: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('style'),
      },
    }
  },
})
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/div.test.ts`
Expected: PASS (8 tests)

---

### Task 3: Span 标记扩展

**Files:**
- Create: `frontend/workbench/src/components/editor/extensions/Span.ts`
- Test: `frontend/workbench/tests/extensions/span.test.ts`

**Step 1: 写失败测试**

```typescript
// tests/extensions/span.test.ts
import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { Span } from '@/components/editor/extensions/Span'

function createEditor(content = '') {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), TextStyleKit, Span],
    content,
  })
}

describe('Span mark', () => {
  it('preserves span with class attribute', () => {
    const editor = createEditor('<p><span class="tag">TypeScript</span></p>')
    const html = editor.getHTML()
    expect(html).toContain('class="tag"')
    editor.destroy()
  })

  it('does not match span with style (left for TextStyle)', () => {
    const editor = createEditor('<p><span style="color: red">Red</span></p>')
    const html = editor.getHTML()
    // TextStyle handles this — should render as <span style="...">
    expect(html).toContain('style="color: red"')
    editor.destroy()
  })

  it('leaves span with both class and style to TextStyle', () => {
    const editor = createEditor('<p><span class="highlight" style="color: red">Text</span></p>')
    const html = editor.getHTML()
    // TextStyle has higher priority and should handle this span
    expect(html).toContain('style="color: red"')
    editor.destroy()
  })

  it('does not create span for bare span without class or style', () => {
    const editor = createEditor('<p><span>Bare</span></p>')
    const html = editor.getHTML()
    // No class and no style → neither Span nor TextStyle matches → plain text
    expect(html).not.toContain('<span')
    expect(html).toContain('Bare')
    editor.destroy()
  })

  it('preserves multiple spans with different classes', () => {
    const editor = createEditor(
      '<p><span class="tag">Go</span> and <span class="tag">React</span></p>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="tag"')
    editor.destroy()
  })

  it('preserves span class across paragraph boundary', () => {
    const editor = createEditor(
      '<div><p><span class="label">Name:</span> Alice</p></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="label"')
    expect(html).toContain('Alice')
    editor.destroy()
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/span.test.ts`
Expected: FAIL — cannot find module `@/components/editor/extensions/Span`

**Step 3: 实现 Span 扩展**

```typescript
// src/components/editor/extensions/Span.ts
import { Mark, mergeAttributes } from '@tiptap/core'

/**
 * Inline span mark that preserves <span> elements with a `class` attribute
 * through ProseMirror parsing. Only matches spans with `class` but NO `style`
 * — spans with `style` are handled by the TextStyle mark (priority 101).
 */
export const Span = Mark.create({
  name: 'span',
  priority: 50,

  parseHTML() {
    return [
      {
        tag: 'span',
        consuming: false,
        getAttrs: (element: HTMLElement) => {
          if (element.hasAttribute('class') && !element.hasAttribute('style')) {
            return {}
          }
          return false
        },
      },
    ]
  },

  renderHTML({ HTMLAttributes }) {
    return ['span', mergeAttributes(HTMLAttributes), 0]
  },

  addAttributes() {
    return {
      class: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('class'),
      },
    }
  },
})
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/span.test.ts`
Expected: PASS (6 tests)

---

### Task 4: 导出桶文件

**Files:**
- Create: `frontend/workbench/src/components/editor/extensions/index.ts`

**Step 1: 创建导出文件**

```typescript
// src/components/editor/extensions/index.ts
export { Div } from './Div'
export { Span } from './Span'
export { PresetAttributes } from './PresetAttributes'
```

**Step 2: 运行全部新测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/extensions/`
Expected: PASS (20 tests)

---

### Task 5: 集成到 EditorPage

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:1-67` (imports + extensions array)

**Step 1: 添加扩展导入**

在 `EditorPage.tsx` 顶部 import 区域添加：

```typescript
import { Div, Span, PresetAttributes } from '@/components/editor/extensions'
```

**Step 2: 注册扩展**

修改 `useEditor` 的 `extensions` 数组（约第 46 行），从：

```typescript
extensions: [
  StarterKit.configure({ strike: false }),
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
],
```

改为：

```typescript
extensions: [
  StarterKit.configure({ strike: false }),
  TextAlign.configure({ types: ['heading', 'paragraph', 'div'] }),
  TextStyleKit,
  Div,
  Span,
  PresetAttributes,
],
```

**Step 3: 运行全部测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run`
Expected: ALL PASS (existing + new)

---

### Task 6: 端到端验证

**Step 1: 用 sample_draft.html 验证 CSS 渲染**

在浏览器中打开编辑器，加载 `tests/fixtures/sample_draft.html` 的内容，确认：
- `.resume` 容器的 padding/margin 被正确剥离（由 stripContainerDimensions）
- `.profile` 的 flex 布局生效
- `.section h2` 的标题样式生效
- `.item .date` 的日期颜色/字号生效
- `.tag` 的标签背景色生效

**Step 2: 验证编辑保存往返**

编辑文本内容 → 保存 → 刷新页面 → 确认 class 属性保留、CSS 仍然生效。

---

## 补充说明

- **不提交代码** — 在当前 dev 分支编码，不 git commit
- **jsdom 环境** — TipTap 扩展测试在 jsdom 中运行，不需要 React 渲染
- **CSSStyleSheet polyfill** — `extract-styles.test.ts` 已经依赖 `CSSStyleSheet`，jsdom 环境已支持
- **PresetAttributes 不包含 div** — Div 节点自身通过 `addAttributes` 定义 class/style，不需要在 PresetAttributes 中重复
