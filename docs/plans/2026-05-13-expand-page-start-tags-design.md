# 扩展 findPageStartPositions 标签集

## 问题

`findPageStartPositions` 使用 `BLOCK_TAGS` 过滤 TreeWalker。当页面边界后的第一个可见元素不在 `BLOCK_TAGS` 中（如 `<ul>`、`<ol>`、`<li>`），TreeWalker 跳过它，找到后面更远的块级元素——导致 `break-before: page` 被设在错误位置，PDF 分页出错。

## 方案

在 `findPageStartPositions` 中使用比 `BLOCK_TAGS` 更宽泛的 `PAGE_START_TAGS` 标签集，覆盖列表等常见非 BLOCK_TAGS 元素。拆分逻辑（`findCrossingPositions`）继续使用 `BLOCK_TAGS` 不变。

## 改动

### types.ts

新增 `PAGE_START_TAGS`：

```ts
const PAGE_START_EXTRA_TAGS = [
  'UL', 'OL', 'LI', 'DL', 'DT', 'DD',
] as const

export const PAGE_START_TAGS = new Set([
  ...BLOCK_TAGS,
  ...PAGE_START_EXTRA_TAGS,
])
```

### detectCrossings.ts

`findPageStartPositions` 的 TreeWalker filter 改为 `PAGE_START_TAGS`：

```ts
acceptNode: (node: Element) =>
  PAGE_START_TAGS.has(node.tagName)
    ? NodeFilter.FILTER_ACCEPT
    : NodeFilter.FILTER_SKIP,
```

### 不改动

- `findCrossingPositions` — 拆分逻辑继续用 `BLOCK_TAGS`
- `syncPageBreaks` — 消费 findPageStartPositions 结果，无变化
- `styleUtils.ts` — 无变化

## 测试

在 detectCrossings.test.ts 新增：

- 页面起始是 `<ul>` → 正确返回位置
- 页面起始是 `<li>` → 正确返回位置
- 页面起始是 `<ol>` → 正确返回位置
- `<ul>` 和 `<p>` 都在 breaker 后 → 返回 `<ul>` 位置（更早的元素）
