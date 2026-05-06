# 移除底部 FormatToolbar 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除编辑器底部固定的 FormatToolbar，简化界面，让编辑区域占满空间。

**Architecture:** 删除 FormatToolbar 组件文件及其在 EditorPage 中的渲染代码和 CSS 样式。格式控制功能由 BubbleToolbar（选中文本时浮动）和 ContextMenu（右键）完整覆盖。

**Tech Stack:** React + TypeScript + TipTap + CSS

---

### Task 1: 更新测试文件 — 移除 FormatToolbar 相关测试

**Files:**
- Modify: `frontend/workbench/tests/TipTapEditor.test.tsx:10`（移除 import）
- Modify: `frontend/workbench/tests/TipTapEditor.test.tsx:41-50`（移除 createFormatToolbarMockEditor）
- Modify: `frontend/workbench/tests/TipTapEditor.test.tsx:73-194`（移除两个 describe 块和 TestToolbarWrapper）
- Modify: `frontend/workbench/tests/EditorPage.autosave.test.tsx:57-59`（移除 FormatToolbar mock）
- Modify: `frontend/workbench/tests/BubbleToolbar.test.tsx:75`（更新注释引用）

**Step 1: 运行现有测试确认全部通过**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx tests/EditorPage.autosave.test.tsx tests/BubbleToolbar.test.tsx`
Expected: All tests PASS

**Step 2: 在 TipTapEditor.test.tsx 中移除 FormatToolbar import 和所有相关代码**

从 `TipTapEditor.test.tsx` 中删除：
- 第 10 行 `import { FormatToolbar } from '@/components/editor/FormatToolbar'`
- 第 41-50 行 `createFormatToolbarMockEditor` 函数
- 第 73-150 行 `describe('FormatToolbar', ...)` 整个 describe 块
- 第 152-170 行 `TestToolbarWrapper` 函数
- 第 172-194 行 `describe('FormatToolbar integration (real editor)', ...)` 整个 describe 块

**Step 3: 在 EditorPage.autosave.test.tsx 中移除 FormatToolbar mock**

删除第 57-59 行：
```typescript
vi.mock('@/components/editor/FormatToolbar', () => ({
  FormatToolbar: () => <div>mock-format-toolbar</div>,
}))
```

**Step 4: 在 BubbleToolbar.test.tsx 中更新注释**

将第 75 行注释从：
```
// so it should render immediately with button states, unlike FormatToolbar
// which starts with null and returns null on first render.
```
改为：
```
// so it should render immediately with button states without a null flash.
```

**Step 5: 运行测试确认 TipTapEditor 测试仍然通过**

Run: `cd frontend/workbench && bunx vitest run tests/TipTapEditor.test.tsx tests/EditorPage.autosave.test.tsx tests/BubbleToolbar.test.tsx`
Expected: All tests PASS（TipTapEditor describe 块保留）

**Step 6: Commit**

```bash
git add frontend/workbench/tests/TipTapEditor.test.tsx frontend/workbench/tests/EditorPage.autosave.test.tsx frontend/workbench/tests/BubbleToolbar.test.tsx
git commit -m "test: 移除 FormatToolbar 相关测试代码"
```

---

### Task 2: 移除 FormatToolbar 组件和引用

**Files:**
- Delete: `frontend/workbench/src/components/editor/FormatToolbar.tsx`
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:10`（移除 import）
- Modify: `frontend/workbench/src/pages/EditorPage.tsx:293-295`（移除渲染代码）
- Modify: `frontend/workbench/src/styles/editor.css:33-41`（移除 .format-toolbar 样式）

**Step 1: 运行测试确认当前状态通过**

Run: `cd frontend/workbench && bunx vitest run`
Expected: All tests PASS

**Step 2: 从 EditorPage.tsx 移除 FormatToolbar import**

删除第 10 行：
```typescript
import { FormatToolbar } from '@/components/editor/FormatToolbar'
```

**Step 3: 从 EditorPage.tsx 移除 FormatToolbar 渲染代码**

删除第 293-295 行：
```tsx
        <div className="format-toolbar">
          <FormatToolbar editor={editor} />
        </div>
```

**Step 4: 从 editor.css 移除 .format-toolbar 样式**

删除第 33-41 行：
```css
/* Format Toolbar */
.format-toolbar {
  background: #ffffff;
  border-top: 1px solid var(--color-border, #e4ddd5);
  padding: 8px 16px;
  min-height: 48px;
  display: flex;
  justify-content: center;
}
```

**Step 5: 删除 FormatToolbar.tsx 文件**

```bash
rm frontend/workbench/src/components/editor/FormatToolbar.tsx
```

**Step 6: 运行测试确认全部通过**

Run: `cd frontend/workbench && bunx vitest run`
Expected: All tests PASS

**Step 7: 运行构建确认无编译错误**

Run: `cd frontend/workbench && bun run build`
Expected: Build succeeds without errors

**Step 8: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/src/styles/editor.css
git rm frontend/workbench/src/components/editor/FormatToolbar.tsx
git commit -m "refactor: 移除底部 FormatToolbar 组件"
```
