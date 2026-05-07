# AI 自定义样式编辑器显示设计

## 背景

AI 生成的简历 HTML 包含完整文档结构（`<!DOCTYPE html>`、`<head><style>...</style></head>`、`<body>`）。当通过 `editor.commands.setContent()` 加载到 TipTap/ProseMirror 编辑器时，`<style>` 标签被剥离，导致 AI 定义的自定义 CSS 类（如 `.section`、`.tag`、`.profile`）在编辑器中无样式显示。

PDF 导出使用原始 HTML（`<style>` 完整保留），因此编辑器与导出存在 WYSIWYG 差距。

## 目标

- **完全 WYSIWYG**：编辑器视觉效果 = PDF 导出结果
- **内容区隔离**：AI 样式不影响工具栏、侧边栏等 UI
- **TipTap 工具栏继续可用**：inline styles（工具栏产生）和 CSS classes（AI 产生）共存

## 方案：CSS 提取注入 + 选择器作用域化

### 核心数据流

```
DB (full HTML with <style>)
  → extractStyles(html)  →  { bodyHtml, scopedCSS }
  → <style> 注入到 .resume-document 容器内
  → setContent(bodyHtml)  →  ProseMirror 渲染 body
  → 样式正确应用 ✅
```

### extractStyles() 函数

新建 `src/lib/extract-styles.ts`：

```ts
interface ExtractedStyles {
  bodyHtml: string       // 纯 body 内容（去掉 <html>/<head>/<body> 包装）
  scopedCSS: string      // 作用域化后的 CSS 文本
}

function extractStyles(fullHtml: string, scopeSelector: string): ExtractedStyles
```

**处理步骤：**
1. 用 `DOMParser` 解析完整 HTML
2. 提取所有 `<style>` 块内容，拼接为 CSS 文本
3. 提取 `<body>` 内的 innerHTML 作为 bodyHtml
4. 对 CSS 文本执行选择器改写（作用域化）
5. 跳过 `@page` 等仅打印相关规则

### 选择器改写规则

将所有 CSS 选择器加上 `.resume-document` 前缀：

| AI 原始选择器 | 改写后 |
|---|---|
| `.class` | `.resume-document .class` |
| `body` / `html` | `.resume-document` |
| `#id` | `.resume-document #id` |
| `.a > .b` | `.resume-document .a > .b` |
| `.a .b, .c .d` | `.resume-document .a .b, .resume-document .c .d` |
| `@page { ... }` | 跳过（不注入编辑器） |
| `@media print { ... }` | 跳过 |

### 容器尺寸冲突处理

AI 生成的 HTML 通常有一个最外层容器（如 `<div class="resume">`），带有 A4 尺寸属性：

```css
.resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
```

编辑器的 `A4Canvas` 已经提供了 A4 容器（`width: 210mm; padding: 18mm 20mm`），直接注入会导致**双重尺寸和双重内边距**。

**处理方式：**
1. 解析 bodyHtml，找到最外层容器元素（body 的第一个子元素）
2. 提取该元素的 class/id
3. 从作用域化 CSS 中，移除该容器选择器的尺寸属性：`width`、`min-height`、`height`、`padding`、`margin`
4. 保留该容器的其他视觉样式

```css
/* 原始 */
.resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }

/* 改写后（尺寸属性被剥离） */
/* .resume 的尺寸规则被移除，由 A4Canvas 管理 */
```

### 样式注入位置

在 `TipTapEditor.tsx` 中：

```tsx
<div className="resume-document">
  <style dangerouslySetInnerHTML={{ __html: scopedCSS }} />
  <EditorContent editor={editor} />
</div>
```

`scopedCSS` 作为 prop 从 `EditorPage.tsx` 传入。

### 样式更新时机

| 场景 | 触发 | 动作 |
|---|---|---|
| 加载草稿 | `useEffect` on `pendingHtml` | 提取样式 + 注入 + setContent |
| AI 编辑完成 | SSE done → 从 DB 拉取最新 HTML | 同上，更新样式和内容 |
| 用户手动编辑 | TipTap update 事件 | 不触发样式更新 |

切换草稿时，前一个草稿的样式被新样式完全替换（无累积）。

### TipTap 工具栏兼容

- 工具栏通过 TextStyleKit 生成 **inline styles**（`style="font-size: 14pt"`）
- AI 样式通过 **CSS classes** 生效
- CSS 规范中 inline styles 优先级高于 class 选择器 → 工具栏操作自然覆盖 AI 样式
- 无需特殊处理

### PDF 导出

PDF 导出保持不变：
- 使用原始 HTML（未改写的 `<style>` 块）
- `render-template.html` 提供隔离容器（`.resume-page`）
- 改写只在编辑器侧执行

## 修改文件清单

| 文件 | 改动类型 | 说明 |
|---|---|---|
| `src/lib/extract-styles.ts` | **新建** | extractStyles() 函数，含选择器改写和容器尺寸剥离 |
| `src/components/editor/TipTapEditor.tsx` | 修改 | 添加 scopedCSS prop 和 `<style>` 标签 |
| `src/pages/EditorPage.tsx` | 修改 | 调用 extractStyles()，传递 scopedCSS |
| `tests/extract-styles.test.ts` | **新建** | extractStyles 单元测试 |

**不需要修改：**
- PDF 导出逻辑（backend `render/` 模块）
- AI agent 逻辑（backend `agent/` 模块）
- TipTap 扩展配置
- A4Canvas 组件
