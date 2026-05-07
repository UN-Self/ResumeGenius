# 修复：extract-styles.ts CSS 遍历重构 + Div 扩展模式对齐

## 背景

代码审查（commits `9093eef..abdca1b`，TipTap Schema 扩展 + 样式提取）发现 8 个问题。其中 2 个严重、3 个重要、3 个建议。本设计采用统一 CSS 遍历提取的方案（方案 C），从根源修复问题并消除重复代码。

## 修复一览

| # | 严重度 | 问题 | 修复方案 |
|---|--------|------|----------|
| 1 | 严重 | `promoteContainerBackground` 中非 print `@media` 内部样式规则被静默提升到顶层、媒体块丢失 | 提取 `walkRuleList` 统一遍历，`processStyleRule` 改为返回值模式 |
| 2 | 严重 | Div `renderHTML` 手动构建 attrs，`addAttributes` 的 `renderHTML` 回调成为死代码 | 改用 `mergeAttributes(HTMLAttributes)`，与 Span 一致 |
| 3 | 重要 | CSS 简写被 CSSOM 展开为多个 longhand，各 longhand 均保留导致冗余 | 对 promoted background 属性做启发式去重：`background-color` 存在时去除 `background-image: none` 等已知默认值 |
| 4 | 重要 | 文件名含空格 | 重命名为 `2026-05-07-tiptap-schema-extensions-design.md` |
| 5 | 重要 | Span/Div 渲染模式不一致 | 问题 #2 修复后自然对齐 |
| 6 | 建议 | 边缘用例测试覆盖不足 | 为 Div/PresetAttributes/promoteContainerBackground 添加 12 个测试 |
| 7 | 建议 | `originalTag` 在 `getAttrs` 和 `addAttributes.parseHTML` 双重解析 | 移除 `parseHTML` 中的 `getAttrs` 回调 |
| 8 | 建议 | 提升后 background 的级联依赖缺少注释 | 添加注释说明 `stripContainerDimensions` 与 `promoteContainerBackground` 的依赖关系 |

## 方案：提取公共 CSS 遍历 + Div 模式对齐

### 提取 1：`parseCss(css: string)` 函数

三个函数（`scopeSelectors`、`stripContainerDimensions`、`promoteContainerBackground`）的开头 5 行完全相同：

```
const sheet = new CSSStyleSheet()
try { sheet.replaceSync(css) } catch { return css }
```

提取为一个函数，消除 10 行重复。

### 提取 2：`walkRuleList()` 函数

三个函数中重复的 CSS 规则遍历逻辑：`for` 循环 → `CSSStyleRule` / `CSSMediaRule` / 其他 at-rule 三向分支。提取为：

```typescript
function walkRuleList(
  rules: CSSRuleList,
  onStyleRule: (rule: CSSStyleRule) => string | null,
  options: { skipPrintMedia: boolean; preserveOtherRules: boolean }
): string[]
```

- `onStyleRule` 返回 `null` → 丢弃该规则（如所有属性都被剥离）
- `skipPrintMedia` → `scopeSelectors` 跳过，其他两个保留
- `preserveOtherRules` → `scopeSelectors` 跳过，其他两个保留原文

### 改造 `promoteContainerBackground`

`processStyleRule` 从 `void`（副作用推到外部数组）改为返回值模式：

```typescript
// 当前：直接 push 到 promotedProps/output（副作用）
function processStyleRule(rule: CSSStyleRule): void

// 改为：返回结果，由调用方负责放置
function processStyleRule(rule: CSSStyleRule): { keptRule: string | null; bgProps: string[] }
```

顶层和 `@media` 内部的调用点统一为：
```typescript
const { keptRule, bgProps } = processStyleRule(rule)
promotedProps.push(...bgProps)
return keptRule  // walkRuleList 负责放入正确的 output/inner 数组
```

### 修复 `@media` 规则丢失

`walkRuleList` 内部递归调用自身处理 `@media` 子规则，`inner.push(keptRule)` 自然正确，因为 `keptRule` 是从 `onStyleRule` 返回的，不是直接推到顶层 `output`。

### Div `renderHTML` 改用 `mergeAttributes`

```typescript
// 当前（手动，绕过 addAttributes 的 renderHTML 回调）
renderHTML({ node }) {
  const attrs: Record<string, string> = {}
  if (node.attrs.class) attrs.class = node.attrs.class
  if (node.attrs.style) attrs.style = node.attrs.style
  return [tag, attrs, 0]
}

// 改为（与 Span 一致，自动调用各属性 renderHTML 回调）
renderHTML({ node, HTMLAttributes }) {
  const tag = node.attrs.originalTag || 'div'
  return [tag, mergeAttributes(HTMLAttributes), 0]
}
```

`originalTag` 的 `renderHTML: () => ({})` 确保它不会泄漏到 DOM。

### 移除 `parseHTML` 中的冗余 `getAttrs`

```typescript
// 当前：getAttrs 返回 originalTag，但 addAttributes 的 parseHTML 也会捕获
parseHTML() {
  return CONTAINER_TAGS.map((tag) => ({
    tag,
    getAttrs: (element) => ({ originalTag: element.tagName.toLowerCase() }),
  }))
}

// 改为：addAttributes.originalTag.parseHTML 已处理
parseHTML() {
  return CONTAINER_TAGS.map((tag) => ({ tag }))
}
```

### CSS 简写去重

`background` 简写被 CSSOM 展开为 longhand 后，对 promoted 属性做启发式清理：

```
规则：如果 promotedProps 中同时存在 background-color 和 background-image: none，
且没有 background-image: <非none值>，则移除 background-image: none
```

这消除了最常见的冗余场景（纯色背景展开出大量默认值 longhand）。

## 涉及文件

| 操作 | 文件 | 改动说明 |
|------|------|----------|
| 重构 | `frontend/workbench/src/lib/extract-styles.ts` | 提取 `parseCss` + `walkRuleList`；改造 `processStyleRule` 为返回值模式；添加简写去重；添加注释 |
| 重构 | `frontend/workbench/src/components/editor/extensions/Div.ts` | `renderHTML` 改用 `mergeAttributes`；移除 `parseHTML` 中的 `getAttrs` |
| 重命名 | `docs/plans/2026-05-07-tip tap-schema-extensions-design.md` → `2026-05-07-tiptap-schema-extensions-design.md` | 文件名去空格 |
| 新增测试 | `frontend/workbench/tests/extract-styles.test.ts` | `@media` 内部 background 提升、`!important` 保留、空 CSS 处理 |
| 新增测试 | `frontend/workbench/tests/extensions/div.test.ts` | `footer`/`article` 标签往返、空 div、多 container 标签 |
| 新增测试 | `frontend/workbench/tests/extensions/preset-attributes.test.ts` | `bulletList`/`blockquote`/`codeBlock` class 保留 |

## 验证

1. `bunx vitest run` — 所有现有 55 + 新增 ~15 测试通过
2. 现有失败的 8 个测试（FontSizeSelector 等）不受影响
3. 手动验证 `@media screen` 内部 background 样式正确保留在媒体块内
