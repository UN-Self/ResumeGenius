# PDF 导出排版同步修复 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复 PDF 导出中空白行压缩、第 2 页起无上边距的问题。

**Architecture:** 纯 CSS 修复，仅修改 render-template.html。将 `.resume-page` padding 替换为 `@page` margin，为 `<p>` 添加 `min-height`。

**Tech Stack:** Go testing (existing `exporter_test.go`), HTML/CSS

---

### Task 1: 为 `min-height` 和 `@page` margin 添加测试

**Files:**
- Modify: `backend/internal/modules/render/exporter_test.go`

**Step 1: Write the failing test for @page margin**

在 `TestWrapWithTemplate_ReplacesPlaceholder` 之后添加：

```go
func TestWrapWithTemplate_PageMarginInsteadOfDivPadding(t *testing.T) {
	result := wrapWithTemplate("<h1>Hello</h1>")

	// @page should carry the margins, not the .resume-page div
	assert.Contains(t, result, "@page { size: A4; margin: 18mm 20mm; }")
	assert.NotContains(t, result, "padding: 18mm 20mm")
}
```

**Step 2: Write the failing test for p min-height**

```go
func TestWrapWithTemplate_ParagraphMinHeight(t *testing.T) {
	result := wrapWithTemplate("<p></p>")

	assert.Contains(t, result, ".resume-page p")
	assert.Contains(t, result, "min-height: 1em")
}
```

**Step 3: Run tests to verify they fail**

Run: `cd backend && go test ./internal/modules/render/... -run "TestWrapWithTemplate_PageMargin|TestWrapWithTemplate_ParagraphMinHeight" -v`
Expected: FAIL — `@page` still has `margin: 0`, `.resume-page` still has `padding: 18mm 20mm`, no `min-height: 1em`

**Step 4: Commit**

```bash
git add backend/internal/modules/render/exporter_test.go
git commit -m "test: add failing tests for @page margin and p min-height"
```

---

### Task 2: 修改 render-template.html CSS

**Files:**
- Modify: `backend/internal/modules/render/render-template.html`

**Step 1: Replace padding with @page margin and add p min-height**

将 render-template.html 中的 CSS 修改为：

```html
.resume-page {
  box-sizing: border-box;
  width: 210mm;
  min-height: 297mm;
  padding: 0;
  background: #ffffff;
  color: #333333;
  font-family: "Inter", "Open Sans", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif;
  font-size: 14px;
  line-height: 1.5;
  white-space: pre-wrap;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}
```

将 `.resume-page p` 规则添加 `min-height`：

```css
.resume-page p  { font-size: 14px; font-weight: 400; line-height: 1.5; min-height: 1em; }
```

将 `@page` 规则改为带 margin：

```css
@page { size: A4; margin: 18mm 20mm; }
```

**Step 2: Run tests to verify they pass**

Run: `cd backend && go test ./internal/modules/render/... -run "TestWrapWithTemplate" -v`
Expected: ALL PASS

**Step 3: Run full render tests to check no regressions**

Run: `cd backend && go test ./internal/modules/render/... -v`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add backend/internal/modules/render/render-template.html
git commit -m "fix: 修复 PDF 空白行压缩和第2页无上边距问题"
```

---

### Task 3: 验证纯文本分页行为

**Files:** 无代码修改，纯验证

**Step 1: 启动后端服务**

Run: `cd backend && DB_HOST=localhost USE_MOCK=true go run cmd/server/main.go`

**Step 2: 手动验证**

在编辑器中创建只含 `<p>` 标签的简历（无容器），内容超过一页，导出 PDF，检查：
- 空白行是否保持高度
- 第 2 页是否有 18mm 上边距
- 纯文本分页是否正常

**Step 3: 根据验证结果决定是否需要修改 SmartSplit**

如果纯文本分页仍有问题，创建后续计划修改 SmartSplit depth 条件。
