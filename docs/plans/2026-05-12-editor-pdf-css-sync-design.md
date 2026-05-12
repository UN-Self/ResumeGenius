# 编辑器与 PDF 导出 CSS 差异修复设计

日期: 2026-05-12

## 问题

| # | 问题 | 风险 | 根因 |
|---|------|------|------|
| 4 | 用户 CSS 优先级方向在 PDF 中反转 | 高 | `wrapWithTemplate` 将用户 `<style>` 插入在模板 `<style>` 之前 |
| 6 | 亚像素分页精度差异 | 中高 | 编辑器用 `getBoundingClientRect()` 96dpi 检测，PDF 在 chromedp 中字体渲染可能产生亚像素偏移 |
| 3 | `disc` vs `"•"` 列表符号不一致 | 中 | 两端 `list-style-type` 值不同 |
| 5 | 双选择器维护成本 | 中 | `.ProseMirror` 和 `.resume-page` 各维护一套排版值 |
| 1 | `"Open Sans"` 字体回退间断 | 低 | PDF 模板多一个回退字体，编辑器没有 |
| 2 | `min-height` 单边存在 | 低 | 仅 PDF 有 `min-height: 1.5em`，编辑器靠 ProseMirror `<br>` 占位 |
| 7 | 颜色变量 vs 硬编码 | 低 | 编辑器用 CSS 变量，PDF 硬编码 `#333333` |

## 方案：共享常量 + 精准修复

### 改动 1：共享排版常量（解决 #1 #2 #3 #5）

新建 `shared/typography.ts`，作为排版值的 single source of truth：

```ts
export const TYPOGRAPHY = {
  fontFamily: '"Inter", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif',
  color: '#333333',
  base: { fontSize: '14px', lineHeight: 1.5, fontWeight: 400 },
  h1: { fontSize: '24px', lineHeight: 1.3, fontWeight: 600 },
  h2: { fontSize: '20px', lineHeight: 1.3, fontWeight: 600 },
  h3: { fontSize: '16px', lineHeight: 1.4, fontWeight: 500 },
  p:  { fontSize: '14px', lineHeight: 1.5, fontWeight: 400, minHeight: '1.5em' },
  ul: { paddingLeft: '24px', listStyleType: 'disc' },
  ol: { paddingLeft: '24px', listStyleType: 'decimal' },
  li: { marginBottom: '4px' },
} as const
```

**消费方式：**

- **前端**：`editor.css` 中 `.ProseMirror` 排版块引用 CSS 自定义属性（`var(--typo-h1-size, 24px)`），`index.css` 的 `@theme` 块定义这些变量。`typography.ts` 作为 JS 端常量源供 `layout.ts` 等模块引用。
- **后端**：`render-template.html` 中 `.resume-page` 排版块的值对齐到 `typography.ts`。通过文档约束 + 脚本校验（或后续引入构建步骤）保持同步。

**具体修复：**

- #1：移除 PDF 模板 `"Open Sans"`，统一为 `typography.ts` 定义的 `fontFamily`
- #2：编辑器 `.ProseMirror p` 添加 `min-height: 1.5em`，与 PDF 一致
- #3：PDF 模板 `list-style-type: "•"` 改为 `disc`，与编辑器统一
- #5：将来排版值只需改 `typography.ts` 一处

### 改动 2：用户 CSS 优先级修复（解决 #4）

修改 `backend/internal/modules/render/exporter.go` 中 `wrapWithTemplate`：

```go
// 之前：用户 CSS 在模板前，被模板覆盖
return strings.Replace(html, "<style>", styleHTML+"<style>", 1)

// 之后：用户 CSS 在模板后，覆盖模板默认值
return strings.Replace(html, "</style>", "</style>"+styleHTML, 1)
```

用户 CSS 选择器被编辑器重写为 `.resume-document .xxx`。PDF 中 div 同时有 `resume-page` 和 `resume-document` 类，所以用户选择器仍能匹配。用户 CSS 在模板之后声明，同特异性时用户样式胜出——与编辑器行为一致。

### 改动 3：分页安全边距（解决 #6）

修改 `frontend/workbench/src/components/editor/extensions/smart-split/types.ts` 中 `DEFAULT_OPTIONS.threshold`：

```ts
// 之前
threshold: 0,

// 之后：4px 安全边距，为 PDF 渲染中的亚像素膨胀留出缓冲
threshold: 4,
```

`threshold` 的语义：当 `rect.bottom > breaker.top - threshold` 时视为越界。值越大，越早触发分页。4px 为 PDF 中的亚像素差异留出缓冲，同时不会导致明显的过度分页。

### 不修改项

- #7（颜色变量 vs 硬编码）：PDF 硬编码 `#333333` 是正确的。简历 PDF 不应受 UI 主题影响。用户内容颜色通过内联样式或用户 CSS 传达，在 PDF 中正常生效。

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `shared/typography.ts` | **新增** — 排版常量 |
| `frontend/workbench/src/styles/editor.css` | `.ProseMirror p` 添加 `min-height: 1.5em` |
| `backend/internal/modules/render/render-template.html` | 移除 `"Open Sans"`、`"•"` 改 `disc`、对齐常量 |
| `backend/internal/modules/render/exporter.go` | 用户 CSS 插入点改为 `</style>` 之后 |
| `frontend/workbench/src/components/editor/extensions/smart-split/types.ts` | `threshold` 默认值改为 2 |
| `backend/internal/modules/render/exporter_test.go` | 更新相关测试断言 |
