# Workbench UI 契约统一 — Warm Editorial 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 workbench 的 15 个编辑器组件从旧 `:root` CSS 变量迁移到 Tailwind `@theme` 语义 token，统一为 Warm Editorial 暖色编辑风。

**Architecture:** 以 `src/index.css` 的 `@theme` 块为唯一色彩 token 源。编辑器组件将 `var(--color-*)` 替换为 Tailwind 语义类（如 `bg-primary-400`、`text-muted-foreground`），`editor.css` 中的 CSS 布局类保留但替换硬编码颜色为 token 引用。最终删除 `editor.css` 中的 `:root { --color-* }` 块。

**Tech Stack:** Tailwind CSS v4, React 18 + TypeScript, shadcn/ui (new-york)

**Design Doc:** `docs/plans/2026-04-30-workbench-ui-contract-design.md`

---

### Task 1: 更新 index.css @theme —— 精炼色彩 Token 源

**Files:**
- Modify: `frontend/workbench/src/index.css:1-26`

**Step 1: 替换 @theme 块**

将现有的 `@theme` 块替换为精炼后的 Warm Editorial token：

```css
@import "tailwindcss";

@theme {
  /* Primary scale */
  --color-primary-50: #faf6f2;
  --color-primary-100: #f2e8e0;
  --color-primary-200: #e5d2c2;
  --color-primary-300: #d4b99a;
  --color-primary-400: #c4956a;
  --color-primary-500: #b3804d;
  --color-primary-600: #9c6b3a;
  --color-primary-700: #7d5530;
  --color-primary-800: #5e4025;
  --color-primary-900: #3f2b1a;
  --color-primary-950: #2a1c10;

  /* Semantic colors */
  --color-background: #faf8f5;
  --color-foreground: #1a1815;
  --color-card: #ffffff;
  --color-card-foreground: #1a1815;
  --color-popover: #ffffff;
  --color-popover-foreground: #1a1815;
  --color-primary: #c4956a;
  --color-primary-foreground: #ffffff;
  --color-secondary: #e8d5c4;
  --color-secondary-foreground: #5c4a3a;
  --color-muted: #f5f1ed;
  --color-muted-foreground: #8c8279;
  --color-accent: #d4a574;
  --color-accent-foreground: #4a3020;
  --color-destructive: #d64545;
  --color-destructive-foreground: #ffffff;
  --color-border: #e4ddd5;
  --color-input: #e4ddd5;
  --color-ring: #c4956a;

  /* Editor-specific tokens */
  --color-canvas-bg: #ede8e0;
  --color-surface-hover: #faf6f2;

  /* Radii */
  --radius: 0.625rem;

  /* Fonts */
  --font-sans: "Inter", ui-sans-serif, system-ui, sans-serif;
  --font-serif: "Playfair Display", Georgia, serif;
  --font-mono: "JetBrains Mono", ui-monospace, monospace;
}
```

**Step 2: 验证构建**

Run: `cd frontend/workbench && bun run build`
Expected: PASS (Tailwind v4 从 @theme 生成对应的 `bg-primary-400` 等工具类)

---

### Task 2: 更新 editor.css 布局类 —— 硬编码颜色 → Token

**Files:**
- Modify: `frontend/workbench/src/styles/editor.css:1-98`

**Step 1: 替换 .action-bar 中的硬编码颜色**

将：
```css
.action-bar {
  background: #ffffff;
  border-bottom: 1px solid #dadce0;
}
```

改为：
```css
.action-bar {
  background: #ffffff;
  border-bottom: 1px solid var(--color-border, #e4ddd5);
}
```

**Step 2: 替换 .ai-panel 中的硬编码颜色**

将：
```css
.ai-panel {
  background: #ffffff;
}
```

改为：
```css
.ai-panel {
  background: #ffffff;
  color: var(--color-foreground, #1a1815);
}
```

**Step 3: 替换 .format-toolbar 中的硬编码颜色**

将：
```css
.format-toolbar {
  background: #ffffff;
  border-top: 1px solid #dadce0;
}
```

改为：
```css
.format-toolbar {
  background: #ffffff;
  border-top: 1px solid var(--color-border, #e4ddd5);
}
```

**Step 4: 替换骨架屏/空状态颜色**

将 `.skeleton-line` 的 `background: #e8eaed` → `background: #d4d0ca`（暖灰），`.skeleton` 中的 `#e8eaed`、`#f1f3f4` → 暖色调。

**Step 5: 替换 .ProseMirror 中的硬编码颜色**

将 `.ProseMirror` 的 `color: #202124` → `color: var(--color-foreground, #1a1815)`

---

### Task 3: 重构 ActionBar.tsx —— var() → Tailwind 类

**Files:**
- Modify: `frontend/workbench/src/components/editor/ActionBar.tsx`

**Step 1: 替换所有 `var(--color-*)` 引用**

将 ActionBar.tsx 中所有 `var(--color-*)` 替换为 Tailwind 语义类：

| 旧 | 新 |
|----|-----|
| `text-[var(--color-primary)]` | `text-primary` |
| `bg-[var(--color-divider)]` | `bg-border` |
| `text-[var(--color-text-main)]` | `text-foreground` |
| `text-[var(--color-text-disabled)]` | `text-muted-foreground/50` |
| `border-[var(--color-divider)]` | `border-border` |
| `hover:bg-[var(--color-primary-bg)]` | `hover:bg-primary-50` |
| `bg-[var(--color-page-bg)]` | `bg-background` |

完整替换后的 ActionBar.tsx：

```tsx
import { FileText } from 'lucide-react'
import type { ReactNode } from 'react'
import type { ExportStatus } from '@/hooks/useExport'

interface ActionBarProps {
  projectName: string
  saveIndicator?: ReactNode
  draftId: string | null
  getHtml: () => string
  exportStatus: ExportStatus
  onExport: () => void
}

const EXPORT_LABEL: Record<ExportStatus, string> = {
  idle: '导出 PDF',
  exporting: '导出中...',
  completed: '导出 PDF',
  failed: '导出失败',
}

export function ActionBar({
  projectName,
  saveIndicator,
  draftId,
  getHtml,
  exportStatus,
  onExport,
}: ActionBarProps) {
  return (
    <div className="action-bar">
      <div className="flex items-center gap-2">
        <FileText size={24} className="text-primary" />
      </div>

      <div className="h-6 w-px bg-border" />
      <span className="text-base font-medium text-foreground">{projectName}</span>

      <div className="flex-1" />

      <div className="flex items-center gap-2">
        {saveIndicator}
      </div>

      <button
        type="button"
        className="px-3 py-1.5 text-sm font-medium text-foreground hover:bg-primary-50 rounded-md transition-colors cursor-pointer"
      >
        版本历史
      </button>

      <button
        type="button"
        disabled={!draftId || exportStatus === 'exporting'}
        onClick={onExport}
        className="px-3 py-1.5 text-sm font-medium text-foreground bg-background border border-border rounded-md disabled:cursor-not-allowed disabled:text-muted-foreground/50 hover:bg-primary-50 transition-colors cursor-pointer"
      >
        {EXPORT_LABEL[exportStatus]}
      </button>
    </div>
  )
}
```

---

### Task 4: 重构 ToolbarButton.tsx + ToolbarSeparator.tsx

**Files:**
- Modify: `frontend/workbench/src/components/editor/ToolbarButton.tsx`
- Modify: `frontend/workbench/src/components/editor/ToolbarSeparator.tsx`

**Step 1: ToolbarButton.tsx**

将 `var(--color-*)` 替换为 Tailwind 类：

```tsx
import { type ReactNode } from 'react'

interface ToolbarButtonProps {
  onClick: () => void
  isActive: boolean
  icon: ReactNode
  label: string
  disabled?: boolean
}

export function ToolbarButton({ onClick, isActive, icon, label, disabled }: ToolbarButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      aria-pressed={isActive}
      className={`
        p-2 rounded-md transition-all duration-150 ease-in-out
        min-w-[44px] min-h-[44px] flex items-center justify-center
        ${isActive
          ? 'bg-primary-50 text-primary hover:bg-primary-100'
          : 'text-muted-foreground hover:bg-surface-hover'
        }
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        focus:outline-none focus:ring-2 focus:ring-ring
      `}
    >
      {icon}
    </button>
  )
}
```

**Step 2: ToolbarSeparator.tsx**

将硬编码 `#dadce0` 替换为 Tailwind 类：

```tsx
export function ToolbarSeparator() {
  return (
    <div
      className="w-px h-5 bg-border mx-1.5"
      role="separator"
    />
  )
}
```

---

### Task 5: 重构 A4Canvas.tsx —— 画布投影

**Files:**
- Modify: `frontend/workbench/src/components/editor/A4Canvas.tsx:44-50`

**Step 1: 替换画布投影**

将：
```tsx
className="bg-white shadow-[0_1px_3px_rgba(0,0,0,0.08)] p-[18mm_20mm]"
```

改为（使用设计文档中的画布投影值）：
```tsx
className="bg-white shadow-[0_2px_12px_rgba(0,0,0,0.08)] p-[18mm_20mm]"
```

**Step 2: 为画布区容器添加暖色背景**

在 `containerRef` 的 div 上添加背景色：
```tsx
<div ref={containerRef} className="canvas-area bg-canvas-bg">
```

---

### Task 6: 重构 ColorPicker.tsx

**Files:**
- Modify: `frontend/workbench/src/components/editor/ColorPicker.tsx`

**Step 1: 替换所有 `var(--color-*)` 引用**

映射：
| 旧 | 新 |
|----|-----|
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |
| `hover:bg-[var(--color-page-bg)]` | `hover:bg-surface-hover` |
| `focus:ring-[var(--color-primary)]` | `focus:ring-ring` |
| `border-[var(--color-divider)]` | `border-border` |
| `text-[var(--color-text-main)]` | `text-foreground` |

完整替换后的 JSX 部分（只改类名，逻辑不变）：

```tsx
{/* Font color trigger */}
<button
  type="button"
  onClick={handleFontClick}
  aria-label="字体颜色"
  aria-haspopup="dialog"
  aria-expanded={open && target === 'font'}
  className="relative p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-md text-muted-foreground hover:bg-surface-hover transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
>
  ...
</button>

{/* Highlight color trigger — 同上替换模式 */}

{/* PopoverContent 内部 */}
<button className="w-6 h-6 rounded border border-border hover:scale-110 transition-transform" ... />

<button className="text-sm text-muted-foreground hover:text-foreground transition-colors">
  重置
</button>

<label className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer">
  自定义
  <input ... />
</label>
```

---

### Task 7: 重构 FontSelector.tsx + FontSizeSelector.tsx + LineHeightSelector.tsx

**Files:**
- Modify: `frontend/workbench/src/components/editor/FontSelector.tsx`
- Modify: `frontend/workbench/src/components/editor/FontSizeSelector.tsx`
- Modify: `frontend/workbench/src/components/editor/LineHeightSelector.tsx`

**Step 1: FontSelector.tsx —— 全局替换 `var(--color-*)`**

映射（FontSelector.tsx 只有 3 种变量引用）：
| 旧 | 新 |
|----|-----|
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |
| `hover:bg-[var(--color-page-bg)]` | `hover:bg-surface-hover` |
| `focus:ring-[var(--color-primary)]` | `focus:ring-ring` |

**Step 2: FontSizeSelector.tsx —— 替换模式**

在 FontSizeSelector.tsx 中，额外有激活态：
| 旧 | 新 |
|----|-----|
| `bg-[var(--color-primary-bg)]` | `bg-primary-50` |
| `text-[var(--color-primary)]` | `text-primary` |
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |
| `hover:bg-[var(--color-page-bg)]` | `hover:bg-surface-hover` |
| `focus:ring-[var(--color-primary)]` | `focus:ring-ring` |

**Step 3: LineHeightSelector.tsx —— 同 FontSizeSelector 模式**

同样的变量 → 同样的类名替换。

---

### Task 8: 重构 SaveIndicator.tsx —— 硬编码颜色 → Tailwind

**Files:**
- Modify: `frontend/workbench/src/components/editor/SaveIndicator.tsx`

**Step 1: 替换所有硬编码颜色**

| 旧硬编码 | 新 Tailwind |
|----------|------------|
| `text-[#5f6368]` | `text-muted-foreground` |
| `text-[#0d652d]` | `text-primary` |
| `text-[#c5221f]` | `text-destructive` |

完整替换后的 SaveIndicator：

```tsx
import { Loader2, Check, AlertCircle } from 'lucide-react'
import type { SaveStatus } from '@/hooks/useAutoSave'

interface SaveIndicatorProps {
  status: SaveStatus
  lastSavedAt: Date | null
  onRetry?: () => void
}

export function SaveIndicator({ status, lastSavedAt, onRetry }: SaveIndicatorProps) {
  if (status === 'idle') return null

  const timeStr = lastSavedAt
    ? lastSavedAt.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    : ''

  return (
    <div aria-live="polite" className="flex items-center gap-1 text-xs">
      {status === 'saving' && (
        <>
          <Loader2 size={14} className="animate-spin text-muted-foreground" />
          <span className="text-muted-foreground">保存中...</span>
        </>
      )}
      {status === 'saved' && (
        <>
          <Check size={14} className="text-primary" />
          <span className="text-primary">已保存 {timeStr}</span>
        </>
      )}
      {status === 'error' && (
        <button
          onClick={onRetry}
          className="flex items-center gap-1 text-destructive hover:underline cursor-pointer"
        >
          <AlertCircle size={14} />
          <span>保存失败，点击重试</span>
        </button>
      )}
    </div>
  )
}
```

---

### Task 9: 重构 AiPanelPlaceholder.tsx

**Files:**
- Modify: `frontend/workbench/src/components/editor/AiPanelPlaceholder.tsx`

**Step 1: 替换 var() 引用**

| 旧 | 新 |
|----|-----|
| `text-[var(--color-text-disabled)]` | `text-muted-foreground/50` |
| `text-[var(--color-text-main)]` | `text-foreground` |
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |

```tsx
import { Sparkles } from 'lucide-react'

export function AiPanelPlaceholder() {
  return (
    <div className="ai-panel">
      <Sparkles size={48} className="text-muted-foreground/50 mb-4" />
      <h2 className="text-xl font-semibold text-foreground mb-2">AI 助手</h2>
      <p className="text-sm font-normal text-muted-foreground mb-4">即将推出</p>
      <p className="text-xs font-normal text-muted-foreground/50 max-w-xs">
        AI 助手将帮助您优化简历内容，提供智能建议，并自动生成简历初稿
      </p>
    </div>
  )
}
```

---

### Task 10: 重构 ParsedSidebar.tsx + ParsedItem.tsx

**Files:**
- Modify: `frontend/workbench/src/components/intake/ParsedSidebar.tsx`
- Modify: `frontend/workbench/src/components/intake/ParsedItem.tsx`

**Step 1: ParsedSidebar.tsx**

| 旧 | 新 |
|----|-----|
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |
| `text-[var(--color-primary)]` | `text-primary` |

```tsx
<h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">素材</h2>
<button className="text-xs text-primary hover:underline cursor-pointer">上传文件</button>
<h3 className="mb-2 text-xs font-semibold text-muted-foreground">解析结果</h3>
```

**Step 2: ParsedItem.tsx**

| 旧 | 新 |
|----|-----|
| `bg-[var(--color-page-bg)]` | `bg-background` |
| `text-[var(--color-primary)]` | `text-primary` |
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |

```tsx
<div className="rounded-lg bg-background p-3">
  <div className="mb-1.5 text-xs font-medium text-primary">
    {icon} {content.label}
  </div>
  <div className="max-h-48 overflow-y-auto text-[13px] leading-relaxed text-muted-foreground whitespace-pre-wrap">
    {content.text}
  </div>
</div>
```

---

### Task 11: 重构 EditorPage.tsx + ProjectDetail.tsx

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Modify: `frontend/workbench/src/pages/ProjectDetail.tsx`

**Step 1: EditorPage.tsx**

| 旧 | 新 |
|----|-----|
| `bg-[var(--color-page-bg)]` | `bg-background` |
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |

两处 loading/error wrapper 和面板 header 文字（共 5 处）。

**Step 2: ProjectDetail.tsx**

| 旧 | 新 |
|----|-----|
| `bg-[var(--color-page-bg)]` | `bg-background` |
| `text-[var(--color-text-secondary)]` | `text-muted-foreground` |
| `text-[var(--color-text-main)]` | `text-foreground` |
| `border-[var(--color-divider)]` | `border-border` |
| `bg-[var(--color-primary)]` | `bg-primary` |
| `hover:bg-[var(--color-primary-hover)]` | `hover:bg-primary-500` |

注意：还需将 `font-serif` 应用于项目标题：

```tsx
<h1 className="font-serif text-xl font-semibold text-foreground truncate">
  {project.title}
</h1>
```

导出/版本按钮也需要调整——无 `bg-[var(--color-primary)]` 和 `text-[var(--color-text-main)]` 的映射。

---

### Task 12: 清理 editor.css —— 删除旧 :root 块 + 替换 var() 引用

**Files:**
- Modify: `frontend/workbench/src/styles/editor.css`

**Step 1: 删除 :root 块（第 155-166 行）**

删除：
```css
/* Design Tokens */
:root {
  --color-primary: #1a73e8;
  --color-primary-hover: #1557b0;
  --color-primary-bg: #e8f0fe;
  --color-page-bg: #f8f9fa;
  --color-card: #ffffff;
  --color-divider: #dadce0;
  --color-text-main: #202124;
  --color-text-secondary: #5f6368;
  --color-text-disabled: #9aa0a6;
}
```

**Step 2: 替换编辑器布局 CSS 类中的 var(--color-*)**

将 `.editor-workspace`、`.editor-panel-left`、`.editor-panel-center`、`.editor-panel-right`、`.panel-header`、`.panel-collapse-btn`、`.panel-expand-btn` 中的 `var(--color-*)` 引用替换为对应的 Tailwind token：

| 旧 var() | 新 CSS 值或 var() |
|----------|-------------------|
| `var(--color-page-bg)` | `#faf8f5`（或 `var(--color-background, #faf8f5)`）|
| `var(--color-card)` | `#ffffff` |
| `var(--color-divider)` | `var(--color-border, #e4ddd5)` |
| `var(--color-text-secondary)` | `#8c8279` |
| `var(--color-text-main)` | `#1a1815` |
| `var(--color-primary)` | `var(--color-primary, #c4956a)` |

由于这些 CSS Grid 布局类保留在 editor.css 中而非迁移到 Tailwind，这里用 CSS 变量回退值写法（`var(--color-border, #e4ddd5)`），与 index.css 中的 `@theme` token 名保持一致。

如果 Tailwind v4 的 `@theme` 变量对全局 CSS 文件也可见（Tailwind v4 会将 `@theme` 变量输出为 `:root` 级别的 CSS 自定义属性），则可以直接用 `var(--color-border)` 不带回退值。为安全起见，使用带回退值的写法。

---

### Task 13: 验证 —— 确保无残留

**Step 1: 搜索旧变量残留**

Run: `cd frontend/workbench && grep -r "var(--color-primary)" src/ --include="*.tsx" --include="*.css"`

Expected: NO RESULTS（所有 `var(--color-*)` 已被替换）

**Step 2: 搜索硬编码旧颜色**

Run: `cd frontend/workbench && grep -r "#1a73e8\|#dadce0\|#f8f9fa\|#5f6368\|#202124\|#9aa0a6\|#e8f0fe\|#1557b0" src/ --include="*.tsx" --include="*.css"`

Expected: NO RESULTS（editor.css 中的 `#dadce0` 和 `#e8eaed` 已被 Task 2 替换）

**Step 3: 运行构建**

Run: `cd frontend/workbench && bun run build`

Expected: PASS（Tailwind v4 生成 CSS，无缺失类引用警告）

**Step 4: 运行前端测试**

Run: `cd frontend/workbench && bunx vitest run`

Expected: PASS（样式变更不影响测试逻辑）

---

### Task 14: 添加 Google Fonts 加载

**Files:**
- Modify: `frontend/workbench/index.html`

**Step 1: 在 `<head>` 中添加字体 CSS import**

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&family=Playfair+Display:wght@500;600&display=swap" rel="stylesheet">
```

---

## 迁移顺序总结

| 序号 | 任务 | 影响范围 |
|------|------|----------|
| 1 | 更新 index.css @theme | 所有 Tailwind 类 |
| 2 | 更新 editor.css 布局样式的硬编码颜色 | 编辑布局 CSS |
| 3 | ActionBar.tsx | 1 组件 |
| 4 | ToolbarButton.tsx + ToolbarSeparator.tsx | 2 组件（基础） |
| 5 | A4Canvas.tsx | 1 组件 |
| 6 | ColorPicker.tsx | 1 组件 |
| 7 | FontSelector + FontSizeSelector + LineHeightSelector | 3 组件 |
| 8 | SaveIndicator.tsx | 1 组件 |
| 9 | AiPanelPlaceholder.tsx | 1 组件 |
| 10 | ParsedSidebar + ParsedItem | 2 组件 |
| 11 | EditorPage.tsx + ProjectDetail.tsx | 2 页面 |
| 12 | 清理 editor.css :root 块 + 替换布局类 var() | 1 CSS 文件 |
| 13 | 验证 | 全项目 |
| 14 | 添加 Google Fonts | index.html |
