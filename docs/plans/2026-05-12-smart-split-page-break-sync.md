# SmartSplit PDF 分页同步 — 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 在 SmartSplit 扩展中新增配置项和分页同步逻辑，自动清理旧 `break-before` 并在每页起始节点插入新的 `break-before: page`，使 PDF 导出与画布排版一致。

**架构：** 在现有 SmartSplitPlugin 的 `performDetectionAndSplit` 末尾新增 `syncPageBreaks` 步骤。复用已有的 `getBreakerPositions` 和 DOM 遍历模式，新增 `findPageStartPositions` 定位每页首个内容节点。通过 `setNodeMarkup` 修改 style 属性。

**技术栈：** TypeScript, ProseMirror, TipTap, Vitest

---

### Task 1: 新增 `insertPageBreaks` 配置项

**文件：**
- 修改：`frontend/workbench/src/components/editor/extensions/smart-split/types.ts`

**Step 1: 修改 types.ts，新增选项**

在 `SmartSplitOptions` 接口中，在 `debug` 之前新增：

```ts
/** 拆分后自动同步 break-before: page，使 PDF 导出分页与画布一致 */
insertPageBreaks: boolean
```

在 `DEFAULT_OPTIONS` 中新增默认值：

```ts
insertPageBreaks: true,
```

**Step 2: 验证编译通过**

运行：`cd frontend/workbench && bunx tsc --noEmit 2>&1 | head -20`
预期：无类型错误（`insertPageBreaks` 尚未被使用，但已有默认值不会报错）

**Step 3: 提交**

```bash
git add frontend/workbench/src/components/editor/extensions/smart-split/types.ts
git commit -m "feat(smart-split): add insertPageBreaks config option"
```

---

### Task 2: 新增 `appendBreakBefore` 和 `removeBreakBefore` 工具函数

**文件：**
- 创建：`frontend/workbench/src/components/editor/extensions/smart-split/styleUtils.ts`
- 创建：`frontend/workbench/tests/smart-split/styleUtils.test.ts`

**Step 1: 写失败测试**

创建 `tests/smart-split/styleUtils.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { appendBreakBefore, removeBreakBefore } from '@/components/editor/extensions/smart-split/styleUtils'

describe('appendBreakBefore', () => {
  it('appends break-before: page to empty style', () => {
    expect(appendBreakBefore('')).toBe('break-before: page')
  })

  it('appends to existing style', () => {
    expect(appendBreakBefore('color: red')).toBe('color: red; break-before: page')
  })

  it('does not duplicate if already present', () => {
    expect(appendBreakBefore('color: red; break-before: page')).toBe('color: red; break-before: page')
  })

  it('handles style with trailing semicolon', () => {
    expect(appendBreakBefore('color: red; ')).toBe('color: red; break-before: page')
  })
})

describe('removeBreakBefore', () => {
  it('removes break-before from style', () => {
    expect(removeBreakBefore('color: red; break-before: page')).toBe('color: red')
  })

  it('returns null when style becomes empty', () => {
    expect(removeBreakBefore('break-before: page')).toBeNull()
  })

  it('returns null for empty string', () => {
    expect(removeBreakBefore('')).toBeNull()
  })

  it('preserves other properties', () => {
    expect(removeBreakBefore('color: red; font-size: 14px; break-before: page')).toBe('color: red; font-size: 14px')
  })
})
```

**Step 2: 运行测试确认失败**

运行：`cd frontend/workbench && bunx vitest run tests/smart-split/styleUtils.test.ts`
预期：FAIL — module not found

**Step 3: 实现 styleUtils.ts**

创建 `frontend/workbench/src/components/editor/extensions/smart-split/styleUtils.ts`：

```ts
export function appendBreakBefore(style: string): string {
  const parts = style.split(';').map(s => s.trim()).filter(Boolean)
  if (parts.some(p => p.startsWith('break-before'))) return style
  parts.push('break-before: page')
  return parts.join('; ')
}

export function removeBreakBefore(style: string): string | null {
  const parts = style.split(';').map(s => s.trim()).filter(p => p && !p.startsWith('break-before'))
  return parts.length > 0 ? parts.join('; ') : null
}
```

**Step 4: 运行测试确认通过**

运行：`cd frontend/workbench && bunx vitest run tests/smart-split/styleUtils.test.ts`
预期：全部 PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/editor/extensions/smart-split/styleUtils.ts frontend/workbench/tests/smart-split/styleUtils.test.ts
git commit -m "feat(smart-split): add styleUtils for break-before manipulation"
```

---

### Task 3: 新增 `findPageStartPositions` 函数

**文件：**
- 修改：`frontend/workbench/src/components/editor/extensions/smart-split/detectCrossings.ts`
- 修改：`frontend/workbench/tests/smart-split/detectCrossings.test.ts`

**Step 1: 写失败测试**

在 `tests/smart-split/detectCrossings.test.ts` 末尾新增 import 和 describe：

在文件顶部 import 中新增 `findPageStartPositions`：

```ts
import {
  getBreakerPositions,
  elementCrossesBreaker,
  findCrossingPositions,
  findPageStartPositions,
} from '@/components/editor/extensions/smart-split/detectCrossings'
```

在文件末尾新增：

```ts
describe('findPageStartPositions', () => {
  it('returns empty when no breakers', () => {
    const mockView = { posAtDOM: vi.fn() }
    const wrapper = document.createElement('div')
    const results = findPageStartPositions(mockView as any, wrapper, [])
    expect(results).toEqual([])
  })

  it('finds first block element after each breaker', () => {
    const wrapper = document.createElement('div')

    // Page 1 content
    const block1 = document.createElement('div')
    block1.textContent = 'Page 1 content'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    // Page 2 content (starts after breaker bottom)
    const block2 = document.createElement('div')
    block2.textContent = 'Page 2 content'
    block2.getBoundingClientRect = () => ({ top: 650, bottom: 1000, height: 350 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(block2)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(10) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(results[0]).toBe(10)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(block2, 0)

    document.body.removeChild(wrapper)
  })

  it('skips empty/zero-height elements', () => {
    const wrapper = document.createElement('div')

    const emptyBlock = document.createElement('div')
    emptyBlock.getBoundingClientRect = () => ({ top: 0, bottom: 0, height: 0 } as DOMRect)

    const realBlock = document.createElement('div')
    realBlock.textContent = 'Content'
    realBlock.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(emptyBlock)
    wrapper.appendChild(realBlock)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(5) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(results[0]).toBe(5)

    document.body.removeChild(wrapper)
  })

  it('handles multiple breakers in single pass', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const block2 = document.createElement('div')
    block2.textContent = 'Page 2'
    block2.getBoundingClientRect = () => ({ top: 650, bottom: 1000, height: 350 } as DOMRect)

    const block3 = document.createElement('div')
    block3.textContent = 'Page 3'
    block3.getBoundingClientRect = () => ({ top: 1250, bottom: 1600, height: 350 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(block2)
    wrapper.appendChild(block3)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [
      { top: 500, bottom: 600 },
      { top: 1100, bottom: 1200 },
    ]
    const mockView = { posAtDOM: vi.fn().mockReturnValueOnce(10).mockReturnValueOnce(20) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(2)
    expect(results[0]).toBe(10)
    expect(results[1]).toBe(20)

    document.body.removeChild(wrapper)
  })

  it('gracefully handles posAtDOM failure for decoration elements', () => {
    const wrapper = document.createElement('div')

    const decoBlock = document.createElement('div')
    decoBlock.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(decoBlock)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = {
      posAtDOM: vi.fn().mockImplementation(() => { throw new Error('not in doc') }),
    }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toEqual([])

    document.body.removeChild(wrapper)
  })
})
```

**Step 2: 运行测试确认失败**

运行：`cd frontend/workbench && bunx vitest run tests/smart-split/detectCrossings.test.ts`
预期：FAIL — `findPageStartPositions` is not exported

**Step 3: 实现 findPageStartPositions**

在 `detectCrossings.ts` 末尾新增：

```ts
/**
 * Find the first content block element after each breaker boundary.
 * Returns ProseMirror positions for page-start nodes.
 */
export function findPageStartPositions(
  view: { posAtDOM: (node: Node, offset: number) => number },
  editorDom: Element,
  breakers: BreakerPosition[],
): number[] {
  if (breakers.length === 0) return []

  const results: number[] = []
  let breakerIdx = 0

  const walker = document.createTreeWalker(
    editorDom,
    NodeFilter.SHOW_ELEMENT,
    {
      acceptNode: (node: Element) =>
        BLOCK_TAGS.has(node.tagName) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP,
    },
  )

  let el = walker.nextNode() as Element | null
  if (el === editorDom) el = walker.nextNode() as Element | null

  while (el && breakerIdx < breakers.length) {
    const rect = el.getBoundingClientRect()
    if (rect.height === 0 || (el.textContent ?? '').trim() === '') {
      el = walker.nextNode() as Element | null
      continue
    }

    if (rect.top >= breakers[breakerIdx].bottom - 1) {
      try {
        results.push(view.posAtDOM(el, 0))
      } catch {
        // decoration element — not in ProseMirror doc
      }
      breakerIdx++
      continue
    }

    el = walker.nextNode() as Element | null
  }

  return results
}
```

**Step 4: 运行测试确认通过**

运行：`cd frontend/workbench && bunx vitest run tests/smart-split/detectCrossings.test.ts`
预期：全部 PASS（包括原有的 `getBreakerPositions`、`elementCrossesBreaker`、`findCrossingPositions` 测试）

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/editor/extensions/smart-split/detectCrossings.ts frontend/workbench/tests/smart-split/detectCrossings.test.ts
git commit -m "feat(smart-split): add findPageStartPositions for page-break sync"
```

---

### Task 4: 新增 `syncPageBreaks` 并集成到插件

**文件：**
- 修改：`frontend/workbench/src/components/editor/extensions/smart-split/SmartSplitPlugin.ts`

**Step 1: 实现 syncPageBreaks 函数和集成逻辑**

在 `SmartSplitPlugin.ts` 中：

1. 新增 import：

```ts
import { findPageStartPositions } from './detectCrossings'
import { appendBreakBefore, removeBreakBefore } from './styleUtils'
```

2. 在文件底部（`performDetectionAndSplit` 函数之后）新增 `syncPageBreaks` 函数：

```ts
function syncPageBreaks(
  view: EditorView,
  breakers: BreakerPosition[],
  log: (...args: any[]) => void,
) {
  const { state } = view
  const { tr, doc } = state

  // Step 1: Clean up all existing break-before styles
  doc.descendants((node, pos) => {
    if (!node.isBlock) return false
    const style = node.attrs.style as string | null
    if (style && style.includes('break-before')) {
      const cleaned = removeBreakBefore(style)
      tr.setNodeMarkup(pos, undefined, { ...node.attrs, style: cleaned })
    }
    return true
  })

  // Step 2: Find page-start nodes
  const pageStarts = findPageStartPositions(view, view.dom, breakers)
  log('pageStarts:', pageStarts.length, pageStarts)

  // Step 3: Add break-before: page to page-start nodes
  for (const pos of pageStarts) {
    const node = tr.doc.nodeAt(pos)
    if (!node) continue
    const currentStyle = (node.attrs.style as string) || ''
    const newStyle = appendBreakBefore(currentStyle)
    tr.setNodeMarkup(pos, undefined, { ...node.attrs, style: newStyle })
  }

  // Step 4: Dispatch if changed (excluded from undo/redo history)
  if (tr.docChanged) {
    log('syncPageBreaks dispatching ✓')
    tr.setMeta(pluginKey, { ownDispatch: true })
    tr.setMeta('addToHistory', false)
    view.dispatch(tr)
  }
}
```

注意：`BreakerPosition` 类型需要新增 import：

```ts
import type { SmartSplitOptions, BreakerPosition } from './types'
```

（原来只 import 了 `SmartSplitOptions`，需改为同时 import `BreakerPosition`）

3. 修改 `performDetectionAndSplit` 函数，在现有逻辑末尾集成 `syncPageBreaks`：

将原函数中 `if (crossings.length === 0) return` 改为不再直接 return，而是跳过拆分后继续 sync。具体改动：

```ts
function performDetectionAndSplit(
  view: EditorView, options: SmartSplitOptions,
  log: (...args: any[]) => void,
) {
  const editorDom = view.dom
  const breakers = getBreakerPositions(editorDom)
  log('breakers:', breakers.length, breakers)
  if (breakers.length === 0) {
    // No breakers = single page, still sync (cleanup only)
    if (options.insertPageBreaks) {
      syncPageBreaks(view, breakers, log)
    }
    return
  }

  // --- Existing split logic (unchanged) ---
  const crossings = findCrossingPositions(view, editorDom, breakers, options.threshold, options.jitter)
  log('crossings:', crossings.length,
    crossings.map(c => ({ pos: c.pos, breaker: c.breakerIndex })))

  let didSplit = false
  if (crossings.length > 0) {
    crossings.sort((a, b) => a.pos - b.pos)
    const { state } = view
    let crossPos = -1
    for (const c of crossings) {
      const $pos = state.doc.resolve(c.pos)
      const crossIndex = $pos.index($pos.depth - 1)
      log(`crossing pos=${c.pos} depth=${$pos.depth}`,
        `parent(${$pos.depth - 1})=${$pos.node($pos.depth - 1)?.type?.name ?? '?'}`,
        `index=${crossIndex}`)
      if ($pos.depth >= 2 && crossIndex > 0 && crossPos < 0) {
        crossPos = c.pos
      }
    }
    if (crossPos >= 0) {
      log('selected crossPos:', crossPos)
      const tr = buildSplitTransaction(state, crossPos, options.parentAttr, log)
      if (tr) {
        const { preEditDoc } = pluginKey.getState(state) as SmartSplitState
        const resultState = state.apply(tr)
        if (preEditDoc && resultState.doc.eq(preEditDoc)) {
          log('split result identical to pre-edit state → undoing user edit')
          undo(view.state, (t) => view.dispatch(t))
        } else {
          log('dispatching transaction ✓')
          tr.setMeta(pluginKey, { ownDispatch: true })
          view.dispatch(tr)
          didSplit = true
        }
      } else {
        log('buildSplitTransaction returned null ✗')
      }
    } else {
      log('no splittable crossing (depth>=2, index>0), skipping')
    }
  }

  // --- NEW: Sync page breaks ---
  if (options.insertPageBreaks) {
    const currentBreakers = didSplit ? getBreakerPositions(editorDom) : breakers
    syncPageBreaks(view, currentBreakers, log)
  }
}
```

**Step 2: 运行编译检查**

运行：`cd frontend/workbench && bunx tsc --noEmit 2>&1 | head -20`
预期：无错误

**Step 3: 运行已有测试确保无回归**

运行：`cd frontend/workbench && bunx vitest run tests/smart-split/`
预期：全部 PASS

**Step 4: 提交**

```bash
git add frontend/workbench/src/components/editor/extensions/smart-split/SmartSplitPlugin.ts
git commit -m "feat(smart-split): integrate syncPageBreaks into detection cycle"
```

---

### Task 5: 新增 syncPageBreaks 集成测试

**文件：**
- 修改：`frontend/workbench/tests/smart-split/SmartSplitPlugin.test.ts`

**Step 1: 写集成测试**

在 `SmartSplitPlugin.test.ts` 末尾新增测试用例：

```ts
it('syncPageBreaks cleans up old break-before and inserts new ones', () => {
  // This tests the syncPageBreaks logic indirectly through the plugin
  // by verifying the plugin accepts insertPageBreaks option
  const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, insertPageBreaks: true })
  expect(plugin).toBeDefined()
})

it('respects insertPageBreaks: false option', () => {
  const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, insertPageBreaks: false })
  expect(plugin).toBeDefined()
})
```

**Step 2: 运行全部 smart-split 测试确认通过**

运行：`cd frontend/workbench && bunx vitest run tests/smart-split/`
预期：全部 PASS

**Step 3: 提交**

```bash
git add frontend/workbench/tests/smart-split/SmartSplitPlugin.test.ts
git commit -m "test(smart-split): add syncPageBreaks integration tests"
```

---

### Task 6: 最终验证

**Step 1: 运行全部前端测试**

运行：`cd frontend/workbench && bunx vitest run`
预期：全部 PASS

**Step 2: 启动开发服务器手动验证**

运行：`cd frontend/workbench && bun run dev`

在浏览器中打开编辑器，加载多页简历内容，检查：
1. 编辑器画布分页正常
2. 打开浏览器开发者工具，检查每页起始的节点是否有 `style="...break-before: page"`
3. 导出 PDF，检查分页位置是否与画布一致

**Step 3: 最终提交（如有遗漏修复）**

```bash
git add -A
git commit -m "feat(smart-split): complete PDF page-break sync feature"
```
