# 编辑器与 PDF 导出 CSS 差异修复 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复编辑器（TipTap/ProseMirror）与 PDF 导出（chromedp）之间的 6 项 CSS 排版差异，确保两端渲染一致。

**Architecture:** 新建 `shared/typography.ts` 作为排版值单一来源，前端 editor.css 和后端 render-template.html 均对齐到该常量。修复 `wrapWithTemplate` 中用户 CSS 插入顺序。调整智能分页阈值留出亚像素安全边距。

**Tech Stack:** TypeScript (前端), Go (后端), CSS, chromedp

**Background:** 见设计文档 `docs/plans/2026-05-12-editor-pdf-css-sync-design.md`

---

### Task 1: 创建共享排版常量

**Files:**
- Create: `frontend/workbench/src/shared/typography.ts`

**Step 1: 创建 typography.ts**

```ts
export const TYPOGRAPHY = {
  fontFamily:
    '"Inter", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif',
  color: '#333333',
  base: { fontSize: '14px', lineHeight: 1.5, fontWeight: 400 },
  h1: { fontSize: '24px', lineHeight: 1.3, fontWeight: 600 },
  h2: { fontSize: '20px', lineHeight: 1.3, fontWeight: 600 },
  h3: { fontSize: '16px', lineHeight: 1.4, fontWeight: 500 },
  p: { fontSize: '14px', lineHeight: 1.5, fontWeight: 400, minHeight: '1.5em' },
  ul: { paddingLeft: '24px', listStyleType: 'disc' },
  ol: { paddingLeft: '24px', listStyleType: 'decimal' },
  li: { marginBottom: '4px' },
} as const
```

**Step 2: 验证 TypeScript 编译**

```bash
cd frontend/workbench && bunx tsc --noEmit --pretty
```

Expected: PASS（无类型错误）

---

### Task 2: 修复编辑器 CSS（#1 #2 #3 #5）

**Files:**
- Modify: `frontend/workbench/src/styles/editor.css:984-988`

**Step 1: 添加 p 元素 min-height**

在 `.ProseMirror p` 规则中添加 `min-height: 1.5em`，与 PDF 模板一致。

当前代码（第 984-988 行）：
```css
.ProseMirror p {
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
}
```

修改为：
```css
.ProseMirror p {
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
  min-height: 1.5em;
}
```

**说明：** 这确保编辑器中的空 `<p>` 标签高度与 PDF 中一致（原本 PDF 有 `min-height: 1.5em`，编辑器没有，空段落高度不同）。

**Step 2: 验证编辑器编译**

```bash
cd frontend/workbench && bun run build
```

Expected: PASS（构建成功）

---

### Task 3: 修复 PDF 模板 CSS（#1 #3）

**Files:**
- Modify: `backend/internal/modules/render/render-template.html`

**Step 1: 移除 "Open Sans" 字体回退**

第 11 行，将：
```css
font-family: "Inter", "Open Sans", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif;
```
改为：
```css
font-family: "Inter", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif;
```

**Step 2: 修复列表符号**

第 37 行，将：
```css
.resume-page ul { list-style-type: "•"; }
```
改为：
```css
.resume-page ul { list-style-type: disc; }
```

`"•"` 是字符串值（CSS Level 3），`disc` 是标准关键字。chromedp 渲染字符串值 `"•"` 时可能产生不同的 glyph，与编辑器中的 `disc` 不一致。

**Step 3: 验证 Go 编译**

```bash
cd backend && go build ./internal/modules/render/...
```

Expected: PASS（embed 的模板文件变更后编译成功）

---

### Task 4: 修复用户 CSS 优先级（#4）

**Files:**
- Modify: `backend/internal/modules/render/exporter.go:309`

**Step 1: 修改 wrapWithTemplate 插入点**

当前代码（第 309 行）：
```go
return strings.Replace(html, "<style>", styleHTML+"<style>", 1)
```

改为：
```go
return strings.Replace(html, "</style>", "</style>"+styleHTML, 1)
```

**说明：** 核心修复。用户自定义 CSS（被 `extractStyles` 重写为 `.resume-document .xxx` 选择器）需要出现在模板默认样式**之后**，这样同特异性时用户样式胜出——与编辑器中行为一致。PDF 渲染 div 同时有 `resume-page` 和 `resume-document` class，用户选择器仍能匹配。

**Step 2: 验证 Go 编译**

```bash
cd backend && go build ./internal/modules/render/...
```

Expected: PASS

---

### Task 5: 调整智能分页阈值（#6）

**Files:**
- Modify: `frontend/workbench/src/components/editor/extensions/smart-split/types.ts:32`

**Step 1: 修改 threshold 默认值**

第 32 行，将：
```ts
threshold: 0,
```
改为：
```ts
threshold: 4,
```

**说明：** `threshold` 语义是"当 `rect.bottom > breaker.top - threshold` 时视为越界"。值越大，越早触发分页。4px 为 PDF 渲染中亚像素膨胀（chromedp vs 浏览器 `getBoundingClientRect` 的 96dpi 差异）留缓冲，不会导致明显的过度分页。

**Step 2: 验证编辑器编译**

```bash
cd frontend/workbench && bunx tsc --noEmit --pretty
```

Expected: PASS

---

### Task 6: 更新 Go 测试

**Files:**
- Modify: `backend/internal/modules/render/exporter_test.go`

**Step 1: 更新 TestWrapWithTemplate_ExtractsFullDocumentBodyAndStyles 验证顺序**

当前测试第 300-310 行验证 `<style>` 标签存在于结果中。修改后，用户样式在 `</style>` 之后插入，需要验证样式出现在模板样式闭合标签之后。

在现有断言（第 305 行）后添加顺序验证：

```go
// 验证用户 CSS 在模板 </style> 之后（优先级高于模板默认值）
assert.Regexp(t, `</style>\s*<style>\.resume-document \.name\{font-size:22px\}</style>`, result,
    "user CSS must appear after template </style> to take priority")
```

原断言 `assert.Contains(t, result, ...)` 仍保留，新增正则验证顺序。

**Step 2: 运行测试**

```bash
cd backend && go test ./internal/modules/render/... -v -run "TestWrapWithTemplate|TestExtractRenderableHTML"
```

Expected: 全部 PASS

---

### Task 7: 运行全部测试验证无回归

**Step 1: 后端测试**

```bash
cd backend && go test ./internal/modules/render/... -v
```

Expected: 全部 PASS

**Step 2: 前端类型检查 + 构建**

```bash
cd frontend/workbench && bunx tsc --noEmit --pretty && bun run build
```

Expected: PASS

---

### Task 8: 提交

```bash
git add frontend/workbench/src/shared/typography.ts
git add frontend/workbench/src/styles/editor.css
git add frontend/workbench/src/components/editor/extensions/smart-split/types.ts
git add backend/internal/modules/render/render-template.html
git add backend/internal/modules/render/exporter.go
git add backend/internal/modules/render/exporter_test.go
git commit -m "fix: 修复编辑器与 PDF 导出 CSS 排版差异（6项）"
```

---

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `frontend/workbench/src/shared/typography.ts` | **新增** — 排版常量单一来源 |
| `frontend/workbench/src/styles/editor.css` | `.ProseMirror p` 添加 `min-height: 1.5em` |
| `backend/internal/modules/render/render-template.html` | 移除 `"Open Sans"`、`"•"` 改为 `disc` |
| `backend/internal/modules/render/exporter.go` | 用户 CSS 插入点改为 `</style>` 之后 |
| `frontend/workbench/src/components/editor/extensions/smart-split/types.ts` | `threshold` 默认值 0 → 4 |
| `backend/internal/modules/render/exporter_test.go` | 添加 CSS 顺序断言 |
