# extract-styles.ts CSS 遍历重构 + Div 模式对齐实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复代码审查发现的 8 个问题——提取公共 CSS 遍历函数消除重复代码；修复 `promoteContainerBackground` 的 `@media` 规则丢失 bug；让 Div 扩展的 `renderHTML` 使用 `mergeAttributes` 消除死代码；补充边缘用例测试。

**Architecture:** 从 `scopeSelectors`、`stripContainerDimensions`、`promoteContainerBackground` 三个函数中提取 `walkRuleList` 通用 CSS 遍历器，将 `processStyleRule` 从副作用模式改为返回值模式。Div 扩展的 `renderHTML` 改用 `mergeAttributes(HTMLAttributes)` 与 Span 对齐。

**Tech Stack:** TypeScript, CSSOM (CSSStyleSheet), TipTap/ProseMirror, Vitest, jsdom

---

### Task 1: 重命名含空格的设计文档

**Files:**
- Rename: `docs/plans/2026-05-07-tip tap-schema-extensions-design.md` → `docs/plans/2026-05-07-tiptap-schema-extensions-design.md`

**Step 1: 重命名文件**

```bash
cd d:/Code/ResumeGenius && git mv "docs/plans/2026-05-07-tip tap-schema-extensions-design.md" "docs/plans/2026-05-07-tiptap-schema-extensions-design.md"
```

**Step 2: 提交**

```bash
git add docs/plans/
git commit -m "docs: rename design doc to kebab-case (remove space in filename)"
```

---

### Task 2: 添加 `promoteContainerBackground` 回归测试（TDD-RED）

**Files:**
- Modify: `frontend/workbench/tests/extract-styles.test.ts`

**Step 1: 添加 `@media` 内部 background 提升测试**

在 `promoteContainerBackground` describe 块末尾（最后一个 `it(...)` 之后，`})` 闭合之前）添加：

```typescript
  it('preserves background promotion inside @media screen block', () => {
    const css = '@media screen and (max-width: 600px) { .resume-document .resume { background: #f5f5f5; } }'
    const result = promoteContainerBackground(css, '.resume')
    // @media block should still exist
    expect(result).toContain('@media screen and (max-width: 600px)')
    // Background should be promoted to .resume-document at top level
    expect(result).toMatch(/\.resume-document\s*\{[^}]*background/)
    // Original container rule inside @media should NOT have background anymore
    // The @media block should contain the .resume rule (minus background)
  })

  it('preserves !important priority on promoted background', () => {
    const css = '.resume-document .resume { background: #fff !important; }'
    const result = promoteContainerBackground(css, '.resume')
    expect(result).toContain('!important')
  })
```

**Step 2: 运行测试确认失败**

```bash
cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts -t "preserves background promotion inside @media"
```

Expected: FAIL — `@media` 块丢失或 background 不在 `.resume-document` 块中

**Step 3: 提交**

```bash
git add frontend/workbench/tests/extract-styles.test.ts
git commit -m "test: add @media promoteContainerBackground regression tests"
```

---

### Task 3: 提取 `parseCss` 函数

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`

**Step 1: 提取 `parseCss`**

在文件顶部区域（`SCOPE_PREFIX` 常量附近）添加：

```typescript
/**
 * Parse a CSS string into a CSSStyleSheet.
 * Returns null if parsing fails (unparseable CSS).
 */
function parseCss(css: string): CSSStyleSheet | null {
  const sheet = new CSSStyleSheet()
  try {
    sheet.replaceSync(css)
    return sheet
  } catch {
    return null
  }
}
```

**Step 2: 替换 `scopeSelectors` 中的内联解析**

将第 99-105 行：
```typescript
  const sheet = new CSSStyleSheet()
  try {
    sheet.replaceSync(css)
  } catch {
    // Unparseable CSS should not be injected — risk of unscoped styles leaking out.
    return ''
  }
```

改为：
```typescript
  const sheet = parseCss(css)
  if (!sheet) {
    // Unparseable CSS should not be injected — risk of unscoped styles leaking out.
    return ''
  }
```

**Step 3: 运行测试确认无回归**

```bash
cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts
```

Expected: all 22 extract-styles tests PASS

**Step 4: 提交**

```bash
git add frontend/workbench/src/lib/extract-styles.ts
git commit -m "refactor: extract parseCss helper from scopeSelectors"
```

---

### Task 4: 提取 `walkRuleList` 函数并替换 `scopeSelectors`

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`

**Step 1: 添加 `walkRuleList` 函数**

在 `parseCss` 之后添加：

```typescript
interface WalkOptions {
  /** Skip @media print blocks entirely */
  skipPrintMedia: boolean
  /** Preserve non-CSSStyleRule, non-CSSMediaRule at-rules as-is */
  preserveOtherRules: boolean
}

/**
 * Walk a CSSRuleList, applying onStyleRule to each CSSStyleRule.
 * CSSMediaRule blocks are recursively walked; their non-print results
 * are wrapped back into @media blocks.
 *
 * Returns an array of CSS rule text strings.
 */
function walkRuleList(
  rules: CSSRuleList,
  onStyleRule: (rule: CSSStyleRule) => string | null,
  options: WalkOptions,
): string[] {
  const output: string[] = []

  for (let i = 0; i < rules.length; i++) {
    const rule = rules[i]

    if (rule instanceof CSSStyleRule) {
      const result = onStyleRule(rule)
      if (result !== null) output.push(result)
    } else if (rule instanceof CSSMediaRule) {
      const mediaText = (rule as CSSMediaRule).media.mediaText
      if (/print/i.test(mediaText)) {
        if (options.skipPrintMedia) continue
        output.push(rule.cssText)
        continue
      }

      const inner = walkRuleList(
        (rule as CSSMediaRule).cssRules,
        onStyleRule,
        options,
      )
      if (inner.length > 0) {
        output.push(`@media ${mediaText} {\n${inner.join('\n')}\n}`)
      }
    } else {
      if (options.preserveOtherRules) {
        output.push(rule.cssText)
      }
      // skip @page and other at-rules by default
    }
  }

  return output
}
```

**Step 2: 用 `walkRuleList` 替换 `scopeSelectors` 的遍历逻辑**

将第 107-134 行的 `processRuleList` 函数和调用替换为：

```typescript
  const output = walkRuleList(
    sheet.cssRules,
    (rule: CSSStyleRule) => scopeRule(rule),
    { skipPrintMedia: true, preserveOtherRules: false },
  )

  return output.join('\n')
```

并删除 `processRuleList` 函数定义。

**Step 3: 运行测试确认无回归**

```bash
cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts
```

Expected: all 22 tests PASS

**Step 4: 提交**

```bash
git add frontend/workbench/src/lib/extract-styles.ts
git commit -m "refactor: extract walkRuleList and replace scopeSelectors traversal"
```

---

### Task 5: 用 `walkRuleList` 替换 `stripContainerDimensions` 的遍历

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`

**Step 1: 替换 `stripContainerDimensions` 的解析和遍历**

将第 172-179 行的解析代码改为使用 `parseCss`，将第 213-245 行的主循环替换为 `walkRuleList`。

解析部分（172-179）改为：
```typescript
  const sheet = parseCss(css)
  if (!sheet) {
    // If CSS can't be parsed, return as-is — better to show styled content
    // with extra container dimensions than to lose all styling.
    return css
  }
```

主循环（213-248）改为：
```typescript
  const output = walkRuleList(
    sheet.cssRules,
    (rule: CSSStyleRule) => stripStyleRule(rule),
    { skipPrintMedia: false, preserveOtherRules: true },
  )

  return output.join('\n')
```

删除旧的 `for` 循环和 `CSSMediaRule` 分支代码。

**Step 2: 运行测试确认无回归**

```bash
cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts
```

Expected: all 22 tests PASS

**Step 3: 提交**

```bash
git add frontend/workbench/src/lib/extract-styles.ts
git commit -m "refactor: replace stripContainerDimensions traversal with walkRuleList"
```

---

### Task 6: 改造 `promoteContainerBackground` — 修复 `@media` 规则丢失 bug

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`

这是核心修复。`processStyleRule` 从副作用模式改为返回值模式。

**Step 1: 改造 `processStyleRule` 签名**

将第 282-314 行的 `processStyleRule` 从 `void` 改为返回值模式：

```typescript
  function processStyleRule(rule: CSSStyleRule): { keptRule: string | null; bgProps: string[] } {
    const selectors = rule.selectorText.split(',').map((s) => s.trim())
    const matchesContainer = selectors.some(
      (s) => selectorEndsWith(s, containerSelector),
    )

    if (!matchesContainer) {
      return { keptRule: rule.cssText, bgProps: [] }
    }

    const bgProps: string[] = []
    const keptProps: string[] = []
    for (let i = 0; i < rule.style.length; i++) {
      const prop = rule.style[i]
      const value = rule.style.getPropertyValue(prop)
      const priority = rule.style.getPropertyPriority(prop)
      const propStr = `${prop}: ${value}${priority ? ` !${priority}` : ''};`

      if (isBackgroundProperty(prop)) {
        bgProps.push(propStr)
      } else {
        keptProps.push(propStr)
      }
    }

    const keptRule =
      keptProps.length > 0
        ? `${rule.selectorText} {\n  ${keptProps.join('\n  ')}\n}`
        : null

    return { keptRule, bgProps }
  }
```

**Step 2: 构建 `onStyleRule` 适配器并替换遍历**

用 `walkRuleList` 替换第 316-344 行：

```typescript
  const output = walkRuleList(
    sheet.cssRules,
    (rule: CSSStyleRule) => {
      const { keptRule, bgProps } = processStyleRule(rule)
      promotedProps.push(...bgProps)
      return keptRule
    },
    { skipPrintMedia: false, preserveOtherRules: true },
  )
```

**Step 3: 替换解析代码**

将第 272-277 行的解析改为使用 `parseCss`：

```typescript
  const sheet = parseCss(css)
  if (!sheet) return css
```

**Step 4: 运行测试确认通过**

```bash
cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts
```

Expected: all 24 tests PASS（包括 Task 2 的新增测试）

**Step 5: 添加级联依赖注释**

在 `promoteContainerBackground` 的 JSDoc 注释末尾添加：

```typescript
/**
 * ...
 *
 * NOTE: Relies on stripContainerDimensions being called first to remove
 * dimension properties from the container rule. Without that step, the
 * original scoped rule (more specific selector) could override the
 * promoted background on `.resume-document` (less specific selector).
 */
```

**Step 6: 提交**

```bash
git add frontend/workbench/src/lib/extract-styles.ts frontend/workbench/tests/extract-styles.test.ts
git commit -m "fix: promoteContainerBackground now correctly preserves @media blocks

Refactored processStyleRule to return {keptRule, bgProps} instead of
pushing directly to output. Replaced inline CSS traversal with walkRuleList.
This fixes a bug where style rules inside non-print @media blocks were
silently promoted out of their media context."
```

---

### Task 7: 添加 background longhand 去重（修复 #3）

**Files:**
- Modify: `frontend/workbench/src/lib/extract-styles.ts`

**Step 1: 在 `promotedProps` 收集完成后添加去重逻辑**

在 `promoteContainerBackground` 的第 346 行（`if (promotedProps.length > 0)` 所在的 `const uniqueProps` 行之前）添加：

```typescript
  if (promotedProps.length > 0) {
    // Deduplicate props (same property may appear from multiple container classes)
    const uniqueProps = [...new Set(promotedProps)]

    // Clean up CSSOM shorthand expansion artifacts:
    // When `background: #fff` is parsed by CSSOM, it expands to many longhands
    // (background-color, background-image: none, background-repeat: repeat, etc.).
    // If we have a background-color, remove default-valued longhands to keep
    // the output concise.
    const hasBgColor = uniqueProps.some((p) => p.startsWith('background-color:'))
    if (hasBgColor) {
      const defaultLonghands = new Set([
        'background-image: none;',
        'background-repeat: repeat;',
        'background-attachment: scroll;',
        'background-position: 0% 0%;',
        'background-size: auto;',
        'background-origin: padding-box;',
        'background-clip: border-box;',
      ])
      const cleaned = uniqueProps.filter((p) => !defaultLonghands.has(p))
      const bgRule = `.resume-document {\n  ${cleaned.join('\n  ')}\n}`
      output.unshift(bgRule)
    } else {
      const bgRule = `.resume-document {\n  ${uniqueProps.join('\n  ')}\n}`
      output.unshift(bgRule)
    }
  }
```

**Step 2: 运行测试确认无回归**

```bash
cd frontend/workbench && bunx vitest run tests/extract-styles.test.ts
```

Expected: all 24 tests PASS

**Step 3: 提交**

```bash
git add frontend/workbench/src/lib/extract-styles.ts
git commit -m "feat: deduplicate CSSOM-expanded background longhands in promoteContainerBackground"
```

---

### Task 8: 修复 Div 扩展 — `renderHTML` 改用 `mergeAttributes` + 移除死代码（修复 #2, #5, #7）

**Files:**
- Modify: `frontend/workbench/src/components/editor/extensions/Div.ts`

**Step 1: 更新 import**

将第 1 行：
```typescript
import { Node } from '@tiptap/core'
```

改为：
```typescript
import { Node, mergeAttributes } from '@tiptap/core'
```

**Step 2: 移除 `parseHTML` 中冗余的 `getAttrs`**

将第 17-24 行：
```typescript
  parseHTML() {
    return CONTAINER_TAGS.map((tag) => ({
      tag,
      getAttrs: (element: HTMLElement) => ({
        originalTag: element.tagName.toLowerCase(),
      }),
    }))
  },
```

改为：
```typescript
  parseHTML() {
    return CONTAINER_TAGS.map((tag) => ({ tag }))
  },
```

**Step 3: 改造 `renderHTML` 使用 `mergeAttributes`**

将第 26-32 行：
```typescript
  renderHTML({ node }) {
    const tag = node.attrs.originalTag || 'div'
    const attrs: Record<string, string> = {}
    if (node.attrs.class) attrs.class = node.attrs.class
    if (node.attrs.style) attrs.style = node.attrs.style
    return [tag, attrs, 0]
  },
```

改为：
```typescript
  renderHTML({ node, HTMLAttributes }) {
    const tag = node.attrs.originalTag || 'div'
    // mergeAttributes calls each attribute's renderHTML callback:
    // - class/style → rendered normally
    // - originalTag → renderHTML returns {} (suppressed from DOM)
    return [tag, mergeAttributes(HTMLAttributes), 0]
  },
```

**Step 4: 运行测试确认无回归**

```bash
cd frontend/workbench && bunx vitest run tests/extensions/div.test.ts
```

Expected: all 8 div tests PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/editor/extensions/Div.ts
git commit -m "fix: Div renderHTML now uses mergeAttributes, removing dead code

- renderHTML changed from manual attr construction to mergeAttributes(HTMLAttributes)
  which properly calls each attribute's renderHTML callback
- Removed redundant getAttrs from parseHTML (addAttributes.originalTag.parseHTML already handles it)
- Aligns with Span's renderHTML pattern (mergeAttributes)"
```

---

### Task 9: 添加 Div 和 PresetAttributes 边缘用例测试（修复 #6）

**Files:**
- Modify: `frontend/workbench/tests/extensions/div.test.ts`
- Modify: `frontend/workbench/tests/extensions/preset-attributes.test.ts`

**Step 1: 添加 Div 边缘用例测试**

在 `tests/extensions/div.test.ts` 的 describe 块末尾添加：

```typescript
  it('preserves footer tag as footer', () => {
    const editor = createEditor('<footer class="page-footer"><p>Footer</p></footer>')
    const html = editor.getHTML()
    expect(html).toContain('<footer')
    expect(html).toContain('class="page-footer"')
    editor.destroy()
  })

  it('preserves article tag as article', () => {
    const editor = createEditor('<article class="post"><p>Content</p></article>')
    const html = editor.getHTML()
    expect(html).toContain('<article')
    expect(html).toContain('class="post"')
    editor.destroy()
  })

  it('handles empty div', () => {
    const editor = createEditor('<div class="spacer"></div>')
    const html = editor.getHTML()
    // ProseMirror may insert an empty paragraph inside the div
    expect(html).toContain('<div')
    expect(html).toContain('class="spacer"')
    editor.destroy()
  })
```

**Step 2: 添加 PresetAttributes 边缘用例测试**

在 `tests/extensions/preset-attributes.test.ts` 的 describe 块末尾添加：

```typescript
  it('preserves class attribute on bulletList', () => {
    const editor = createEditor('<ul class="skills-list"><li><p>Item</p></li></ul>')
    expect(editor.getHTML()).toContain('class="skills-list"')
    editor.destroy()
  })

  it('preserves class attribute on blockquote', () => {
    const editor = createEditor('<blockquote class="quote"><p>Text</p></blockquote>')
    expect(editor.getHTML()).toContain('class="quote"')
    editor.destroy()
  })
```

**Step 3: 运行测试确认通过**

```bash
cd frontend/workbench && bunx vitest run tests/extensions/
```

Expected: all extension tests PASS（div: 11, span: 6, preset-attributes: 8）

**Step 4: 提交**

```bash
git add frontend/workbench/tests/extensions/div.test.ts frontend/workbench/tests/extensions/preset-attributes.test.ts
git commit -m "test: add edge case coverage for Div and PresetAttributes extensions"
```

---

### Task 10: 全量回归测试 + 最终提交

**Step 1: 运行所有测试**

```bash
cd frontend/workbench && bunx vitest run tests/extensions/ tests/extract-styles.test.ts
```

Expected: all ~65 tests PASS

**Step 2: 检查 8 个已有失败测试不受影响**

```bash
cd frontend/workbench && bunx vitest run 2>&1 | grep -E "(FAIL|PASS|Tests)"
```

确认 FontSizeSelector、LineHeightSelector、AlignSelector、ProjectList 的 8 个失败仍为相同测试，无新增失败。

**Step 3: 提交**

```bash
git add -A
git commit -m "chore: final verification — all refactor and regression tests pass"
```

---

### 受影响文件总览

| 操作 | 文件 | Task |
|------|------|------|
| Rename | `docs/plans/2026-05-07-tip tap-schema-extensions-design.md` → `2026-05-07-tiptap-schema-extensions-design.md` | 1 |
| Refactor | `frontend/workbench/src/lib/extract-styles.ts` | 3-7 |
| Refactor | `frontend/workbench/src/components/editor/extensions/Div.ts` | 8 |
| Tests | `frontend/workbench/tests/extract-styles.test.ts` | 2 |
| Tests | `frontend/workbench/tests/extensions/div.test.ts` | 9 |
| Tests | `frontend/workbench/tests/extensions/preset-attributes.test.ts` | 9 |

### 修复覆盖

| 审查问题 | Task |
|----------|------|
| #1 @media 规则丢失 | 6 |
| #2 renderHTML 绕过 addAttributes | 8 |
| #3 CSS 简写展开冗余 | 7 |
| #4 文件名空格 | 1 |
| #5 Span/Div 模式不一致 | 8 |
| #6 边缘测试覆盖不足 | 2, 9 |
| #7 originalTag 双重解析 | 8 |
| #8 级联依赖注释 | 6 末尾 |
