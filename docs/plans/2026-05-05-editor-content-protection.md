# 编辑器内容保护实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 TipTap 编辑器 A4 区域添加多层内容保护（水印覆盖、剪贴板纯文本过滤、右键拦截、打印拦截），提高免费用户白嫖门槛。

**Architecture:** 四层前端阻吓：CSS `@media print` 拦截打印、React 水印节点 + MutationObserver 防截图、TipTap copy 事件拦截过滤剪贴板、右键菜单禁用。所有保护仅作用于 A4 canvas 区域，不影响 HTML 数据源和 PDF 导出。

**Tech Stack:** React 18 + TipTap (ProseMirror) + CSS `@media print` + MutationObserver API

**Design doc:** `docs/plans/2026-05-05-editor-content-protection-design.md`

---

### Task 1: 水印组件 — WatermarkOverlay

**Files:**
- Create: `frontend/workbench/src/components/editor/WatermarkOverlay.tsx`
- Test: `frontend/workbench/tests/WatermarkOverlay.test.tsx`

**Step 1: 写失败测试**

创建 `frontend/workbench/tests/WatermarkOverlay.test.tsx`：

```tsx
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { WatermarkOverlay } from '@/components/editor/WatermarkOverlay'

describe('WatermarkOverlay', () => {
  it('renders watermark text when visible=true (default)', () => {
    render(<WatermarkOverlay />)
    expect(screen.getByText('ResumeGenius 预览')).toBeInTheDocument()
  })

  it('renders upgrade hint text', () => {
    render(<WatermarkOverlay />)
    expect(screen.getByText('导出无水印 PDF 即可去除水印')).toBeInTheDocument()
  })

  it('does not render when visible=false', () => {
    const { container } = render(<WatermarkOverlay visible={false} />)
    expect(container.innerHTML).toBe('')
  })

  it('has pointer-events:none so editing is not blocked', () => {
    render(<WatermarkOverlay />)
    const overlay = screen.getByText('ResumeGenius 预览').closest('[data-testid="watermark-overlay"]')
    expect(overlay).toHaveStyle({ pointerEvents: 'none' })
  })

  it('re-creates the watermark DOM node when removed externally', () => {
    const { container } = render(<WatermarkOverlay />)
    const watermark = container.querySelector('[data-testid="watermark-overlay"]')
    expect(watermark).toBeInTheDocument()

    // Simulate external removal (e.g. DevTools delete)
    watermark!.remove()

    // MutationObserver should re-create it
    // Wait for microtask (MutationObserver is async)
    return vi.waitFor(() => {
      expect(container.querySelector('[data-testid="watermark-overlay"]')).toBeInTheDocument()
    })
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/WatermarkOverlay.test.tsx`
Expected: FAIL — module not found

**Step 3: 实现 WatermarkOverlay 组件**

创建 `frontend/workbench/src/components/editor/WatermarkOverlay.tsx`：

```tsx
import { useEffect, useRef } from 'react'

interface WatermarkOverlayProps {
  visible?: boolean
}

export function WatermarkOverlay({ visible = true }: WatermarkOverlayProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const watermarkRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!visible || !containerRef.current) return

    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        for (const removed of mutation.removedNodes) {
          if (
            (removed as Element).dataset?.testid === 'watermark-overlay' ||
            (removed as Element).querySelector?.('[data-testid="watermark-overlay"]')
          ) {
            // Watermark was removed — re-create it
            const existing = containerRef.current?.querySelector('[data-testid="watermark-overlay"]')
            if (!existing && watermarkRef.current) {
              containerRef.current?.appendChild(watermarkRef.current)
            }
          }
        }
      }
    })

    observer.observe(containerRef.current, { childList: true })
    return () => observer.disconnect()
  }, [visible])

  if (!visible) return null

  return (
    <div ref={containerRef} className="watermark-container">
      <div
        ref={watermarkRef}
        data-testid="watermark-overlay"
        className="watermark-overlay"
      >
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="watermark-item">
            <span className="watermark-brand">ResumeGenius 预览</span>
            <span className="watermark-hint">导出无水印 PDF 即可去除水印</span>
          </div>
        ))}
      </div>
    </div>
  )
}
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/WatermarkOverlay.test.tsx`
Expected: PASS (5 tests)

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/editor/WatermarkOverlay.tsx frontend/workbench/tests/WatermarkOverlay.test.tsx
git commit -m "feat: 添加 WatermarkOverlay 水印组件"
```

---

### Task 2: 水印 + 右键拦截集成到 A4Canvas

**Files:**
- Modify: `frontend/workbench/src/components/editor/A4Canvas.tsx`
- Modify: `frontend/workbench/tests/A4Canvas.test.tsx`

**Step 1: 写失败测试**

在 `frontend/workbench/tests/A4Canvas.test.tsx` 末尾追加：

```tsx
  it('renders watermark overlay on the canvas', async () => {
    const canvas = await screen.findByTestId('a4-canvas')
    expect(screen.getByTestId('watermark-overlay')).toBeInTheDocument()
  })

  it('prevents context menu on the canvas area', async () => {
    const canvas = await screen.findByTestId('a4-canvas')
    const container = canvas.closest('.canvas-area')!

    const contextEvent = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })
    container.dispatchEvent(contextEvent)

    expect(contextEvent.defaultPrevented).toBe(true)
  })
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/A4Canvas.test.tsx`
Expected: FAIL — watermark overlay not found, context menu not prevented

**Step 3: 集成到 A4Canvas**

修改 `frontend/workbench/src/components/editor/A4Canvas.tsx`，在 A4 纸容器上包裹水印 + 右键拦截：

1. 导入 `WatermarkOverlay`
2. 在 A4 纸 `<div>` 内，`{children}` 之后添加 `<WatermarkOverlay />`
3. 在最外层 `canvas-area` div 上添加 `onContextMenu={(e) => e.preventDefault()}`

修改后的关键结构：

```tsx
return (
  <div
    ref={containerRef}
    className="canvas-area bg-canvas-bg"
    onContextMenu={(e) => e.preventDefault()}
  >
    <div
      data-testid="a4-canvas"
      className="bg-white shadow-[0_2px_12px_rgba(0,0,0,0.08)] p-[18mm_20mm] relative"
      style={{
        width: '210mm',
        minHeight: '297mm',
        transform: `scale(${zoom})`,
        transformOrigin: 'top center',
      }}
    >
      {children || (editor && <TipTapEditor editor={editor} />)}
      <WatermarkOverlay />
    </div>
  </div>
)
```

注意：A4 纸 div 需要加 `relative` class 以支持水印的 `absolute` 定位。

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/A4Canvas.test.tsx`
Expected: PASS

**Step 5: 提交**

```bash
git add frontend/workbench/src/components/editor/A4Canvas.tsx frontend/workbench/tests/A4Canvas.test.tsx
git commit -m "feat: 集成水印覆盖层和右键拦截到 A4 画布"
```

---

### Task 3: 水印 CSS 样式 + @media print 拦截

**Files:**
- Modify: `frontend/workbench/src/styles/editor.css`

**Step 1: 添加水印样式和打印拦截 CSS**

在 `frontend/workbench/src/styles/editor.css` 文件末尾（第 300 行之后）追加：

```css
/* === Watermark Overlay === */
.watermark-container {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  z-index: 5;
}

.watermark-overlay {
  position: absolute;
  inset: -50%;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 60px 80px;
  transform: rotate(-30deg);
  pointer-events: none;
  user-select: none;
  opacity: 0.12;
}

.watermark-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.watermark-brand {
  font-size: 18px;
  font-weight: 600;
  color: #1a1815;
}

.watermark-hint {
  font-size: 11px;
  color: #1a1815;
}

/* === Print Protection === */
@media print {
  .canvas-area,
  .a4-page {
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

**Step 2: 运行全部前端测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run`
Expected: ALL PASS

**Step 3: 提交**

```bash
git add frontend/workbench/src/styles/editor.css
git commit -m "feat: 添加水印覆盖层样式和打印拦截 CSS"
```

---

### Task 4: 剪贴板纯文本过滤

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Modify: `frontend/workbench/tests/EditorPage.test.tsx`

**Step 1: 写失败测试**

在 `frontend/workbench/tests/EditorPage.test.tsx` 末尾追加：

```tsx
describe('Editor content protection — clipboard', () => {
  it('intercepts copy event and writes only plain text to clipboard', async () => {
    server.use(
      http.get('/api/v1/assets', () =>
        HttpResponse.json({ code: 0, data: [], message: 'ok' })
      )
    )

    render(
      <MemoryRouter initialEntries={['/projects/1/edit']}>
        <Routes>
          <Route path="/projects/:projectId" element={<div>Detail</div>} />
          <Route path="/projects/:projectId/edit" element={<EditorPage />} />
        </Routes>
      </MemoryRouter>
    )

    // Wait for editor to load
    const canvas = await screen.findByTestId('a4-canvas')

    // Find the ProseMirror editable area
    const editorEl = canvas.querySelector('.ProseMirror') as HTMLElement
    expect(editorEl).toBeTruthy()

    // Dispatch a copy event
    const copyEvent = new ClipboardEvent('copy', {
      bubbles: true,
      cancelable: true,
      clipboardData: new DataTransfer(),
    })
    Object.defineProperty(copyEvent, 'clipboardData', {
      value: {
        setData: vi.fn(),
        getData: vi.fn(() => ''),
      },
    })

    editorEl.dispatchEvent(copyEvent)

    // Verify setData was called with text/plain only
    expect(copyEvent.clipboardData.setData).toHaveBeenCalledWith(
      'text/plain', expect.any(String)
    )
    // Verify default was prevented (no HTML clipboard data)
    expect(copyEvent.defaultPrevented).toBe(true)
  })
})
```

**Step 2: 运行测试确认失败**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx`
Expected: FAIL — setData not called, defaultPrevented is false

**Step 3: 在 EditorPage.tsx 添加 copy 事件拦截**

修改 `frontend/workbench/src/pages/EditorPage.tsx` 中的 `useEditor` 配置，在 `editorProps` 中添加 `handleDOMEvents`：

```tsx
  const editor = useEditor({
    extensions: [
      StarterKit,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TextStyleKit,
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'resume-content outline-none',
        style: 'min-height: 261mm;',
      },
      handleDOMEvents: {
        copy(_view, event) {
          const { from, to } = _view.state.selection
          const plainText = _view.state.doc.textBetween(from, to, '\n')
          event.preventDefault()
          event.clipboardData.setData('text/plain', plainText)
          return true
        },
      },
    },
  })
```

**Step 4: 运行测试确认通过**

Run: `cd frontend/workbench && bunx vitest run tests/EditorPage.test.tsx`
Expected: PASS

**Step 5: 运行全部前端测试确认无回归**

Run: `cd frontend/workbench && bunx vitest run`
Expected: ALL PASS

**Step 6: 提交**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/tests/EditorPage.test.tsx
git commit -m "feat: 添加剪贴板纯文本过滤 — 复制仅保留无格式文本"
```

---

### Task 5: 全量回归验证

**Step 1: 运行全部前端测试**

Run: `cd frontend/workbench && bunx vitest run`
Expected: ALL PASS

**Step 2: 构建检查**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds

**Step 3: 手动验证清单**

在浏览器 `bun run dev` 中验证：
1. 编辑器正常编辑（输入、格式化、AI 对话）—— 不受水印影响
2. A4 纸上方可见半透明水印文字
3. 右键点击 A4 区域无菜单弹出
4. 选中文字后 Ctrl+C，在记事本中粘贴只有纯文本
5. Ctrl+P 打印预览中简历内容被隐藏，显示付费引导
6. PDF 导出按钮正常工作（导出无水印）
