# AI 自定义样式编辑器显示 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 使 AI 生成的 `<style>` 块在 TipTap 编辑器中正确显示，实现编辑器与 PDF 导出的完全 WYSIWYG 一致性。

**Architecture:** 从完整 HTML 中提取 `<style>` 块，对 CSS 选择器添加 `.resume-document` 前缀实现作用域隔离，剥离最外层容器的尺寸属性避免与 A4Canvas 冲突，将处理后的 CSS 注入编辑器容器。

**Tech Stack:** TypeScript, vitest (jsdom), DOMParser, React `dangerouslySetInnerHTML`

**Design doc:** `docs/plans/2026-05-07-ai-style-editor-display-design.md`

---

### Task 1: extractStyles 基础 — HTML 解析与内容提取

**Files:**
- Create: `frontend/workbench/src/lib/extract-styles.ts`
- Create: `frontend/workbench/tests/extract-styles.test.ts`

**Step 1: 写失败测试 — HTML 解析与 style 提取**

在 `tests/extract-styles.test.ts` 中：

```ts
import { describe, expect, it } from 'vitest'
import { extractStyles } from '@/lib/extract-styles'

const SAMPLE_HTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <style>
    .section { margin-bottom: 10pt; }
    .tag { background: #f0f0f0; }
  </style>
</head>
<body>
  <div class="resume">
    <section class="section"><h2>标题</h2></section>
    <span class="tag">标签</span>
  </div>
</body>
</html>`

describe('extractStyles', () => {
  it('extracts style CSS and body HTML from a full HTML document', () => {
    const result = extractStyles(SAMPLE_HTML)
    expect(result.bodyHtml).toContain('<div class="resume">')
    expect(result.bodyHtml).toContain('<section class="section">')
    expect(result.bodyHtml).toContain('<span class="tag">标签</span>')
    expect(result.bodyHtml).not.toContain('<html')
    expect(result.bodyHtml).not.toContain('<head')
    expect(result.bodyHtml).not.toContain('<style')
  })

  it('scopes CSS selectors with .resume-document prefix', () => {
    const result = extractStyles(SAMPLE_HTML)
    expect(result.scopedCSS).toContain('.resume-document .section')
    expect(result.scopedCSS).toContain('.resume-document .tag')
    expect(result.scopedCSS).not.toContain('.section {')
  })

  it('returns empty bodyHtml and empty scopedCSS for empty input', () => {
    const result = extractStyles('')
    expect(result.bodyHtml).toBe('')
    expect(result.scopedCSS).toBe('')
  })

  it('handles plain HTML fragment without style blocks', () => {
    const html = '<p>Hello</p>'
    const result = extractStyles(html)
    expect(result.bodyHtml).toBe('<p>Hello</p>')
    expect(result.scopedCSS).toBe('')
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts`
Expected: FAIL — `Cannot find module '@/lib/extract-styles'`

**Step 3: 实现 extractStyles 基础版本**

在 `src/lib/extract-styles.ts` 中：

```ts
export interface ExtractedStyles {
  bodyHtml: string
  scopedCSS: string
}

const SCOPE_SELECTOR = '.resume-document'

// Dimension properties to strip from root container
const DIMENSION_PROPS = ['width', 'min-width', 'max-width', 'height', 'min-height', 'max-height', 'padding', 'padding-top', 'padding-right', 'padding-bottom', 'padding-left', 'margin', 'margin-top', 'margin-right', 'margin-bottom', 'margin-left']

export function extractStyles(fullHtml: string): ExtractedStyles {
  if (!fullHtml.trim()) {
    return { bodyHtml: '', scopedCSS: '' }
  }

  const parser = new DOMParser()
  const doc = parser.parseFromString(fullHtml, 'text/html')

  // Extract style content
  const styleElements = doc.querySelectorAll('style')
  let cssText = ''
  styleElements.forEach((el) => {
    cssText += el.textContent || ''
  })

  // Extract body inner HTML
  const bodyHtml = doc.body.innerHTML

  if (!cssText.trim()) {
    return { bodyHtml, scopedCSS: '' }
  }

  // Find root container selector (first child of body)
  const rootContainerClasses = getRootContainerClasses(doc)

  // Scope CSS
  const scopedCSS = scopeAndCleanCSS(cssText, SCOPE_SELECTOR, rootContainerClasses)

  return { bodyHtml, scopedCSS }
}
```

暂时导出空函数 `scopeAndCleanCSS` 和 `getRootContainerClasses`：

```ts
function getRootContainerClasses(_doc: Document): string[] {
  return []
}

function scopeAndCleanCSS(css: string, _scope: string, _rootClasses: string[]): string {
  return css
}
```

**Step 4: 运行测试确认基础用例通过**

Run: `cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts`
Expected: PASS (前4个测试通过，scopedCSS 测试需要 Task 2 实现)

**Step 5: Commit**

```bash
git add frontend/workbench/src/lib/extract-styles.ts frontend/workbench/tests/extract-styles.test.ts
git commit -m "feat(editor): add extractStyles skeleton with HTML parsing"
```

---

### Task 2: scopeSelectors — CSS 选择器作用域化

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`
- Modify: `frontend/workbench/tests/extract-styles.test.ts`

**Step 1: 写失败测试 — 选择器改写**

在 `tests/extract-styles.test.ts` 中追加 `describe('scopeSelectors', ...)`：

```ts
import { scopeSelectors } from '@/lib/extract-styles'

describe('scopeSelectors', () => {
  it('prefixes simple class selectors', () => {
    const result = scopeSelectors('.section { margin-bottom: 10pt; }', '.resume-document')
    expect(result).toBe('.resume-document .section { margin-bottom: 10pt; }')
  })

  it('rewrites body selector to scope', () => {
    const result = scopeSelectors('body { font-family: sans-serif; }', '.resume-document')
    expect(result).toBe('.resume-document { font-family: sans-serif; }')
  })

  it('rewrites html selector to scope', () => {
    const result = scopeSelectors('html { font-size: 16px; }', '.resume-document')
    expect(result).toBe('.resume-document { font-size: 16px; }')
  })

  it('rewrites * selector to scope *', () => {
    const result = scopeSelectors('* { margin: 0; padding: 0; }', '.resume-document')
    expect(result).toBe('.resume-document * { margin: 0; padding: 0; }')
  })

  it('handles comma-separated selectors', () => {
    const result = scopeSelectors('.a, .b { color: red; }', '.resume-document')
    expect(result).toBe('.resume-document .a, .resume-document .b { color: red; }')
  })

  it('handles compound selectors with combinators', () => {
    const result = scopeSelectors('.profile h1 { font-size: 18pt; }', '.resume-document')
    expect(result).toBe('.resume-document .profile h1 { font-size: 18pt; }')
  })

  it('handles child combinator', () => {
    const result = scopeSelectors('.item > h3 { font-weight: 600; }', '.resume-document')
    expect(result).toBe('.resume-document .item > h3 { font-weight: 600; }')
  })

  it('skips @page rules', () => {
    const css = '@page { size: A4; margin: 0; }\n.section { margin: 10pt; }'
    const result = scopeSelectors(css, '.resume-document')
    expect(result).not.toContain('@page')
    expect(result).toContain('.resume-document .section')
  })

  it('skips @media print blocks', () => {
    const css = '@media print { .no-print { display: none; } }\n.section { margin: 10pt; }'
    const result = scopeSelectors(css, '.resume-document')
    expect(result).not.toContain('@media print')
    expect(result).toContain('.resume-document .section')
  })

  it('preserves non-print @media blocks with scoped internal selectors', () => {
    const css = '@media screen { .tag { font-size: 9pt; } }'
    const result = scopeSelectors(css, '.resume-document')
    expect(result).toContain('@media screen')
    expect(result).toContain('.resume-document .tag')
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts`
Expected: FAIL — `scopeSelectors is not exported`

**Step 3: 实现 scopeSelectors**

在 `src/lib/extract-styles.ts` 中实现并导出：

```ts
export function scopeSelectors(cssText: string, scope: string): string {
  let result = cssText

  // Remove @page rules
  result = result.replace(/@page\s*\{[^}]*\}/g, '')

  // Remove @media print blocks
  result = result.replace(/@media\s+print\s*\{[\s\S]*?\}\s*\}/g, '')

  // Scope @media non-print blocks
  result = result.replace(
    /@media\s+([^{]+)\{([\s\S]*?)\}\s*\}/g,
    (_, query: string, body: string) => {
      const scopedBody = prefixSelectors(body, scope)
      return `@media ${query}{${scopedBody}}`
    }
  )

  // Scope remaining regular rules
  result = prefixSelectors(result, scope)

  return result.trim()
}

function prefixSelectors(cssText: string, scope: string): string {
  return cssText.replace(
    /([^{}@/+]+?)\s*\{/g,
    (match, selectorStr: string) => {
      const selectors = selectorStr.split(',')
      const scoped = selectors
        .map((s: string) => {
          s = s.trim()
          if (!s) return ''
          if (s === 'body' || s === 'html') return scope
          if (s === '*') return `${scope} *`
          return `${scope} ${s}`
        })
        .filter(Boolean)
        .join(', ')
      return `${scoped} {`
    }
  )
}
```

同时更新 `extractStyles` 函数，将 `scopeAndCleanCSS` 替换为 `scopeSelectors`：

```ts
// In extractStyles():
const scopedCSS = scopeSelectors(cssText, SCOPE_SELECTOR)
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/lib/extract-styles.ts frontend/workbench/tests/extract-styles.test.ts
git commit -m "feat(editor): implement CSS selector scoping with @page/@media handling"
```

---

### Task 3: stripContainerDimensions — 容器尺寸冲突处理

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`
- Modify: `frontend/workbench/tests/extract-styles.test.ts`

**Step 1: 写失败测试 — 容器尺寸剥离**

在 `tests/extract-styles.test.ts` 中追加：

```ts
describe('stripContainerDimensions', () => {
  it('strips dimension properties from root container in scoped CSS', () => {
    const scopedCSS = `.resume-document .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
.resume-document .section { margin-bottom: 10pt; }`
    const result = stripContainerDimensions(scopedCSS, '.resume')
    expect(result).not.toContain('width: 210mm')
    expect(result).not.toContain('min-height: 297mm')
    expect(result).not.toContain('padding: 18mm 20mm')
    expect(result).toContain('.resume-document .section')
  })

  it('preserves non-dimension properties on root container', () => {
    const scopedCSS = `.resume-document .resume { width: 210mm; background: #fff; color: #333; padding: 0; }
.resume-document .tag { background: #f0f0f0; }`
    const result = stripContainerDimensions(scopedCSS, '.resume')
    expect(result).toContain('background: #fff')
    expect(result).toContain('color: #333')
    expect(result).not.toContain('width: 210mm')
    expect(result).not.toContain('padding: 0')
  })

  it('removes rule entirely if all properties are dimension-related', () => {
    const scopedCSS = `.resume-document .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; margin: 0 auto; }
.resume-document .section { margin-bottom: 10pt; }`
    const result = stripContainerDimensions(scopedCSS, '.resume')
    expect(result).not.toContain('.resume-document .resume')
    expect(result).toContain('.resume-document .section')
  })

  it('full integration: extractStyles strips container dimensions from sample HTML', () => {
    const result = extractStyles(SAMPLE_HTML)
    // .resume has width/min-height/padding — should be stripped
    expect(result.scopedCSS).not.toContain('width: 210mm')
    expect(result.scopedCSS).not.toContain('min-height: 297mm')
    // .section styles should remain
    expect(result.scopedCSS).toContain('.resume-document .section')
    expect(result.scopedCSS).toContain('.resume-document .tag')
  })
})
```

注意：`SAMPLE_HTML` 中需要包含 `.resume { width: 210mm; ... }` 规则。需要更新 Task 1 中定义的 `SAMPLE_HTML`：

```ts
const SAMPLE_HTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <style>
    @page { size: A4; margin: 0; }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Noto Sans SC', sans-serif; font-size: 10.5pt; line-height: 1.4; color: #333; }
    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
    .section { margin-bottom: 10pt; }
    .tag { background: #f0f0f0; padding: 2pt 8pt; border-radius: 3pt; }
  </style>
</head>
<body>
  <div class="resume">
    <section class="section"><h2>标题</h2></section>
    <span class="tag">标签</span>
  </div>
</body>
</html>`
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts`
Expected: FAIL — `stripContainerDimensions is not exported`

**Step 3: 实现 stripContainerDimensions**

在 `src/lib/extract-styles.ts` 中：

```ts
export function stripContainerDimensions(scopedCSS: string, rootSelector: string): string {
  if (!rootSelector || !scopedCSS) return scopedCSS

  // Build the scoped selector pattern, e.g. ".resume-document .resume"
  const escaped = rootSelector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const scopedPattern = `${escaped}`

  // Match the rule for the root container
  const ruleRegex = new RegExp(
    `[^{}]*?${escaped}[^{]*?\\{([^}]*)\\}`,
    'g'
  )

  return scopedCSS.replace(ruleRegex, (match, props: string) => {
    // Remove dimension-related properties
    const cleanedProps = props
      .split(';')
      .map((p: string) => p.trim())
      .filter((p: string) => {
        if (!p) return false
        const prop = p.split(':')[0].trim().toLowerCase()
        return !DIMENSION_PROPS.includes(prop)
      })
      .join('; ')

    if (!cleanedProps.trim()) {
      // Remove the entire rule if no properties remain
      return ''
    }

    // Reconstruct with cleaned properties
    return match.replace(/\{[^}]*\}/, `{ ${cleanedProps}; }`)
  })
}
```

实现 `getRootContainerClasses`：

```ts
function getRootContainerClasses(doc: Document): string[] {
  const firstChild = doc.body.firstElementChild
  if (!firstChild) return []

  const classes = firstChild.className
  if (typeof classes === 'string' && classes.trim()) {
    return classes.trim().split(/\s+/)
  }
  return []
}
```

更新 `extractStyles` 函数以使用容器尺寸剥离：

```ts
export function extractStyles(fullHtml: string): ExtractedStyles {
  if (!fullHtml.trim()) {
    return { bodyHtml: '', scopedCSS: '' }
  }

  const parser = new DOMParser()
  const doc = parser.parseFromString(fullHtml, 'text/html')

  const styleElements = doc.querySelectorAll('style')
  let cssText = ''
  styleElements.forEach((el) => {
    cssText += el.textContent || ''
  })

  const bodyHtml = doc.body.innerHTML

  if (!cssText.trim()) {
    return { bodyHtml, scopedCSS: '' }
  }

  let scopedCSS = scopeSelectors(cssText, SCOPE_SELECTOR)

  // Strip dimension properties from root container
  const rootClasses = getRootContainerClasses(doc)
  for (const cls of rootClasses) {
    scopedCSS = stripContainerDimensions(scopedCSS, `.${cls}`)
  }

  return { bodyHtml, scopedCSS }
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add frontend/workbench/src/lib/extract-styles.ts frontend/workbench/tests/extract-styles.test.ts
git commit -m "feat(editor): strip container dimension styles to avoid A4Canvas conflict"
```

---

### Task 4: 集成到 A4Canvas + EditorPage

**Files:**
- Modify: `frontend/workbench/src/components/editor/A4Canvas.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`

**Step 1: 修改 A4Canvas — 接受 scopedCSS prop**

在 `A4Canvas.tsx` 中：

```ts
interface A4CanvasProps {
  editor?: Editor | null
  scopedCSS?: string
  children?: ReactNode
}
```

在 `.resume-document` div 内部、`TipTapEditor` 之前注入 `<style>`：

```tsx
<div
  data-testid="a4-canvas"
  className="resume-document relative bg-resume-paper p-[18mm_20mm] shadow-[0_22px_80px_rgba(2,8,23,0.24)] ring-1 ring-black/5"
  style={{ ... }}
>
  {scopedCSS && <style dangerouslySetInnerHTML={{ __html: scopedCSS }} />}
  {children || (editor && <TipTapEditor editor={editor} />)}
  <WatermarkOverlay />
</div>
```

**Step 2: 修改 EditorPage — 调用 extractStyles**

添加 import：

```ts
import { extractStyles } from '@/lib/extract-styles'
```

添加 state：

```ts
const [scopedCSS, setScopedCSS] = useState('')
```

修改 `pendingHtml` 设置处（共3处），将 `setPendingHtml(html)` 替换为同时设置样式：

1. 创建草稿时（约第223行）：
```ts
setPendingHtml(draft.html_content || '')
```
→ 不变，在 useEffect 中统一处理

2. 加载现有草稿时（约第247行）：
```ts
setPendingHtml(draft.html_content || '')
```
→ 不变，在 useEffect 中统一处理

3. 在 `useEffect` 中（第273-278行）统一处理：

```ts
useEffect(() => {
  if (editor && pendingHtml !== null && !hasAppliedRef.current) {
    hasAppliedRef.current = true
    const { bodyHtml, scopedCSS: css } = extractStyles(pendingHtml)
    setScopedCSS(css)
    editor.commands.setContent(bodyHtml)
  }
}, [editor, pendingHtml])
```

4. AI 编辑完成回调 `onApplyEdits`（约第427-432行）：

```ts
onApplyEdits={async () => {
  if (!editor || !draftId) return
  const draft = await workbenchApi.getDraft(Number(draftId))
  restoringContent.current = true
  const { bodyHtml, scopedCSS: css } = extractStyles(draft.html_content || '')
  setScopedCSS(css)
  editor.commands.setContent(bodyHtml)
  restoringContent.current = false
}}
```

5. `onRestoreHtml` 回调（约第434-438行）：

```ts
onRestoreHtml={(html) => {
  if (!editor) return
  restoringContent.current = true
  const { bodyHtml, scopedCSS: css } = extractStyles(html)
  setScopedCSS(css)
  editor.commands.setContent(bodyHtml)
  restoringContent.current = false
}}
```

6. 传递 scopedCSS 给 A4Canvas（约第389行）：

```tsx
<A4Canvas editor={editor} scopedCSS={scopedCSS} />
```

7. 版本预览回退 `handleRollback`（约第155-156行）：

```ts
const handleRollback = async () => {
  setRollbacking(true)
  try {
    await flush()
    const html = await rollbackVersion()
    if (editor) {
      const { bodyHtml, scopedCSS: css } = extractStyles(html)
      setScopedCSS(css)
      editor.commands.setContent(bodyHtml)
    }
    setRollbackDialogOpen(false)
  } catch (e) {
    toast(e instanceof Error ? e.message : '回滚失败')
  } finally {
    setRollbacking(false)
  }
}
```

**Step 3: 手动验证**

Run: `cd frontend/workbench && bun run dev`

验证步骤：
1. 打开一个有 AI 生成内容的草稿
2. 确认自定义样式（如 `.section` 的 border-bottom、`.tag` 的背景色）在编辑器中正确显示
3. 确认内容区域没有双重 padding/width 问题
4. 确认 TipTap 工具栏仍可正常使用
5. 确认工具栏/侧边栏不受 AI 样式影响

**Step 4: Commit**

```bash
git add frontend/workbench/src/components/editor/A4Canvas.tsx frontend/workbench/src/pages/EditorPage.tsx
git commit -m "feat(editor): integrate AI style extraction into editor pipeline"
```

---

### Task 5: 全量测试验证

**Step 1: 运行所有前端测试**

Run: `cd frontend/workbench && bunx vitest run`
Expected: ALL PASS — 无回归

**Step 2: 运行构建检查**

Run: `cd frontend/workbench && bun run build`
Expected: Build 成功，无 TypeScript 错误

**Step 3: 最终 Commit（如有修复）**

```bash
git add -A
git commit -m "test: verify AI style editor display — all tests pass"
```
