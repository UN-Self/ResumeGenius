# 计划：扩展 TipTap Schema 保留自定义 HTML 样式

## 背景

AI 生成的简历 HTML（如 `sample_draft.html`）使用了 `<div>`、`<section>`、`<header>`、`<span>` 等元素及自定义 `class` 属性。ProseMirror 的严格 schema 在 `setContent()` 时会丢弃所有未注册的元素和 class 属性，导致 `extractStyles()` 注入的 scoped CSS 选择器无法匹配任何 DOM 元素。

**根因：** StarterKit schema 没有 `div`/`section`/`header` 节点，也没有任何扩展保留标准元素上的 `class` 属性。CSS 管道本身工作正常，问题完全在 ProseMirror 解析侧。

## 方案

在 `src/components/editor/extensions/` 中创建三个 TipTap 扩展：

### 1. `Div.ts` — 块级容器节点

- `Node.create`：`group: 'block'`，`content: 'block*'`，`selectable: false`，`draggable: false`
- `parseHTML`：匹配 `div`、`section`、`header`、`footer`、`main`、`article`、`nav`、`aside`
- 通过 `originalTag` 属性记住原始标签名，渲染时输出正确的标签（不作为 DOM 属性）
- 保留 `class` 和 `style` 属性
- div 内的纯文本会被 ProseMirror 自动包裹在 `<p>` 中（scoped CSS 中的 `* { margin:0 }` 会重置段落边距）

### 2. `Span.ts` — 行内 span 标记

- `Mark.create`：`priority: 50`（低于 TextStyle 的 101）
- `parseHTML`：只匹配有 `class` 但**没有** `style` 的 `<span>`（`consuming: false` + `getAttrs` 过滤）
- 有 `style` 的 span（即使同时有 `class`）留给 TextStyle 处理
- 只保留 `class` 属性

### 3. `PresetAttributes.ts` — 为现有节点添加全局 `class`/`style` 属性

- `Extension.create`，使用 `addGlobalAttributes`（与 TextStyleKit 中 TextAlign 相同的模式）
- 为以下节点类型添加 `class` 和 `style`：`paragraph`、`heading`、`listItem`、`bulletList`、`orderedList`、`blockquote`、`codeBlock`、`div`

### 4. 集成到 `EditorPage.tsx`

在编辑器 extensions 数组中注册 `Div`、`Span`、`PresetAttributes`。在 TextAlign 的 types 中添加 `'div'`。

## 涉及文件

| 操作 | 文件 |
|---|---|
| 新建 | `src/components/editor/extensions/Div.ts` |
| 新建 | `src/components/editor/extensions/Span.ts` |
| 新建 | `src/components/editor/extensions/PresetAttributes.ts` |
| 新建 | `src/components/editor/extensions/index.ts` |
| 修改 | `src/pages/EditorPage.tsx` — 导入并注册扩展 |
| 新建 | `tests/extensions/div.test.ts` |
| 新建 | `tests/extensions/span.test.ts` |
| 新建 | `tests/extensions/preset-attributes.test.ts` |

## TDD 实施顺序

1. **PresetAttributes** — 最简单，无冲突风险。测试 `<p>`、`<h2>`、`<li>` 上的 class/style
2. **Div** — 测试 div/section/header 往返、嵌套 div、包含列表的 div
3. **Span** — 测试带 class 的 span 被保留、带 style 的 span 归 TextStyle、同时有 class 和 style 的 span 归 TextStyle
4. **集成** — 更新 EditorPage.tsx，运行完整测试套件

## 验证

1. `bunx vitest run` — 所有现有 + 新测试通过
2. 手动：加载 `sample_draft.html` fixture，在浏览器中确认 CSS 正确渲染
3. 手动：编辑文本 → 保存 → 重新加载，确认 class 属性被保留
