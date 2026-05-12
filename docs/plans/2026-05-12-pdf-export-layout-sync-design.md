# PDF 导出排版同步修复设计

日期: 2026-05-12

## 问题

| # | 问题 | 根因 |
|---|------|------|
| 1 | 用户添加的空白行在 PDF 中被压缩 | render-template `<p>` 无 `min-height`，空段落高度为 0 |
| 2 | 智能分页对纯 `<p>` 结构无效 | SmartSplit 要求 `depth >= 2`，纯 `<p>` 的 depth=1（待验证是否已由 syncPageBreaks 覆盖） |
| 3 | PDF 第 2 页起无上边距 | `.resume-page` 单 div padding 只作用于第一页，`break-before: page` 后 Chrome 重置盒模型 |

## 方案：CSS 层修复

仅修改 `backend/internal/modules/render/render-template.html`。

### 改动 1：空白行保持高度

```css
.resume-page p { min-height: 1em; }
```

空 `<p></p>` 在 chromedp 中获得一行高度，不再被压缩为 0。

### 改动 2：每页自动边距

```css
/* 之前 */
.resume-page { padding: 18mm 20mm; }
@page { size: A4; margin: 0; }

/* 之后 */
.resume-page { padding: 0; }
@page { size: A4; margin: 18mm 20mm; }
```

`@page` margin 是 CSS Paged Media 原生机制，每个物理页面自动获得 18mm 上下、20mm 左右边距。与 div padding 不同，不受 `break-before` 分页影响。

### 不影响编辑器

编辑器使用 `PaginationPlus` 插件 + `layout.ts` JS 常量控制分页。前端代码中不存在 `.resume-page` 类。`@page` margin 改动仅作用于后端 chromedp 渲染管道。

## 纯文本分页验证计划

修复问题 1 和 3 后，验证纯 `<p>` 结构的分页行为：

1. `syncPageBreaks()` 使用纯 DOM 检测，应能对 `<p>` 标签正确添加 `break-before: page`
2. 如果验证通过，问题 2 无需额外代码修改
3. 如果验证失败，进入 SmartSplit depth 条件修改

## 修改文件清单

- `backend/internal/modules/render/render-template.html` — CSS 修改
