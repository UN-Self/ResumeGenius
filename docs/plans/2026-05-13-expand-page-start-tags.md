# 扩展 findPageStartPositions 标签集 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让 `findPageStartPositions` 能检测 UL/OL/LI 等非 BLOCK_TAGS 元素作为页面起始位置，修复 PDF 导出分页错误。

**Architecture:** 在 types.ts 中新增 `PAGE_START_TAGS`（BLOCK_TAGS 超集），在 detectCrossings.ts 的 `findPageStartPositions` 中替换使用。拆分逻辑不变。

**Tech Stack:** TypeScript, Vitest, jsdom

---

### Task 1: 写失败测试 — UL 作为页面起始

**Files:**
- Test: `frontend/workbench/tests/smart-split/detectCrossings.test.ts`

**Step 1: 在 `findPageStartPositions` describe 块末尾新增测试**

在 `detectCrossings.test.ts` 的 `describe('findPageStartPositions')` 最后一个 it 块之后，`})` 闭合之前添加：

```ts
  it('detects UL element as page start (not in BLOCK_TAGS but in PAGE_START_TAGS)', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1 content'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const ul = document.createElement('ul')
    const li = document.createElement('li')
    li.textContent = 'List item on page 2'
    ul.appendChild(li)
    ul.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(ul)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(15) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(results[0]).toBe(15)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(ul, 0)

    document.body.removeChild(wrapper)
  })
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/smart-split/detectCrossings.test.ts`

Expected: FAIL — `findPageStartPositions` 当前用 `BLOCK_TAGS` 过滤，`<ul>` 被跳过，测试找不到 page start。

---

### Task 2: 实现 — 新增 PAGE_START_TAGS 并替换

**Files:**
- Modify: `frontend/workbench/src/components/editor/extensions/smart-split/types.ts`
- Modify: `frontend/workbench/src/components/editor/extensions/smart-split/detectCrossings.ts`

**Step 1: 在 types.ts 的 `BLOCK_TAGS` 定义之后添加 `PAGE_START_TAGS`**

在 `types.ts` 第 51 行（`BLOCK_TAGS` 的闭合 `)` 之后）添加：

```ts
/** Extended tag set for page-start detection (broader than BLOCK_TAGS) */
const PAGE_START_EXTRA_TAGS = [
  'UL', 'OL', 'LI', 'DL', 'DT', 'DD',
] as const

export const PAGE_START_TAGS = new Set([
  ...BLOCK_TAGS,
  ...PAGE_START_EXTRA_TAGS,
])
```

**Step 2: 在 detectCrossings.ts 中导入 `PAGE_START_TAGS` 替换 `BLOCK_TAGS`**

修改 import 行：

```ts
// 旧
import { BLOCK_TAGS } from './types'
// 新
import { PAGE_START_TAGS } from './types'
```

修改 `findPageStartPositions` 内的 TreeWalker filter（第 27-28 行）：

```ts
// 旧
BLOCK_TAGS.has(node.tagName) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP,
// 新
PAGE_START_TAGS.has(node.tagName) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP,
```

注意：`findCrossingPositions` 仍然导入 `BLOCK_TAGS`，所以需要同时导入两者：

```ts
import { BLOCK_TAGS, PAGE_START_TAGS } from './types'
```

**Step 3: 运行全部测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/smart-split/detectCrossings.test.ts`

Expected: ALL PASS

---

### Task 3: 补充测试覆盖 — OL、LI、混合场景

**Files:**
- Test: `frontend/workbench/tests/smart-split/detectCrossings.test.ts`

**Step 1: 在 `findPageStartPositions` describe 块中追加三个测试**

```ts
  it('detects OL element as page start', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const ol = document.createElement('ol')
    const li = document.createElement('li')
    li.textContent = 'Ordered item'
    ol.appendChild(li)
    ol.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(ol)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(20) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(ol, 0)

    document.body.removeChild(wrapper)
  })

  it('detects LI element as page start when LI is direct child of wrapper', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const li = document.createElement('li')
    li.textContent = 'Standalone list item on page 2'
    li.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(li)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(25) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(li, 0)

    document.body.removeChild(wrapper)
  })

  it('prefers UL over later P when both are after breaker', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const ul = document.createElement('ul')
    const li = document.createElement('li')
    li.textContent = 'List at page start'
    ul.appendChild(li)
    ul.getBoundingClientRect = () => ({ top: 650, bottom: 800, height: 150 } as DOMRect)

    const p = document.createElement('p')
    p.textContent = 'Paragraph after list'
    p.getBoundingClientRect = () => ({ top: 820, bottom: 950, height: 130 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(ul)
    wrapper.appendChild(p)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(30) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(ul, 0)

    document.body.removeChild(wrapper)
  })
```

**Step 2: 运行全部 smart-split 测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run tests/smart-split/`

Expected: ALL PASS

---

### Task 4: 运行全部前端测试 + 编译检查

**Step 1: 运行全部前端测试**

Run: `cd frontend/workbench && bunx vitest run`

Expected: ALL PASS

**Step 2: TypeScript 编译检查**

Run: `cd frontend/workbench && bunx tsc --noEmit`

Expected: 无错误

**Step 3: Commit**

```bash
git add frontend/workbench/src/components/editor/extensions/smart-split/types.ts frontend/workbench/src/components/editor/extensions/smart-split/detectCrossings.ts frontend/workbench/tests/smart-split/detectCrossings.test.ts
git commit -m "feat: 扩展 findPageStartPositions 标签集以支持 UL/OL/LI 分页检测"
```
