# SmartSplit PDF 分页同步设计

**日期：** 2026-05-12
**分支：** feat/a4-multi-page

## 问题

编辑器画布通过 `tiptap-pagination-plus` 渲染视觉分页（`.breaker` 元素）。SmartSplit 检测跨页内容并拆分。但 HTML 导出为 PDF 时（chromedp），文档中没有 CSS 分页属性。Chrome 自行分页，可能与编辑器的视觉分页不一致。

## 方案

在 SmartSplit 插件中新增 `syncPageBreaks` 步骤，每次检测周期结束后执行（无论是否发生拆分）。步骤：

1. 清理文档中所有已有的 `break-before` 样式
2. 找到每个 `.breaker` 边界后的第一个内容节点
3. 通过 `setNodeMarkup` 给这些节点添加 `break-before: page`
4. 以 `ownDispatch: true` 派发 transaction，防止重复触发

## 设计

### 配置项

在 `SmartSplitOptions` 中新增：

```ts
insertPageBreaks: boolean  // default: true
```

### 数据流

```
performDetectionAndSplit (每次 debounce 触发)
  ├─ Step 1: 现有拆分逻辑（不变）
  └─ Step 2: syncPageBreaks（新增，始终执行）
       ├─ 清理文档中所有旧的 break-before 样式
       ├─ getBreakerPositions → 所有 .breaker 的 Y 坐标
       ├─ findPageStartPositions → 每个 breaker 后的第一个内容节点
       └─ Transaction: setNodeMarkup 添加 break-before: page
           └─ 以 ownDispatch: true 派发
```

### 新函数：`findPageStartPositions`（detectCrossings.ts）

- 单次 TreeWalker 遍历 block 元素
- 对每个 breaker bottom，找第一个非空 block 元素满足 `rect.top >= breaker.bottom`
- 返回 ProseMirror position 数组（通过 `posAtDOM`，装饰元素会 try/catch 失败）
- 复用已有 `BLOCK_TAGS` 和 `getBreakerPositions()`

### 新函数：`syncPageBreaks`（SmartSplitPlugin.ts）

1. `doc.descendants()` 遍历：移除所有含 `break-before` 的 `style` 属性
2. `findPageStartPositions()`：获取每页起始节点的 pos
3. 对每个 pos：读取当前 `style`，追加 `break-before: page`
4. 同一个 Transaction 中执行多个 `setNodeMarkup`
5. 仅 `tr.docChanged` 时 dispatch

### 集成方式

在 `performDetectionAndSplit` 末尾：

- 发生拆分后：重新读取 breaker 位置（DOM 同步更新），然后 sync
- 未发生拆分：直接用已有 breakers 调用 sync
- 无 breakers（单页）：仅执行清理

### 样式工具函数

```ts
appendBreakBefore(style: string): string   // 追加 "break-before: page"
removeBreakBefore(style: string): string | null  // 移除 "break-before: ..."
```

使用 `break-before: page`（现代 CSS 标准，chromedp `PreferCSSPageSize: true` 支持）。

## 改动文件

| 文件 | 改动 |
|------|------|
| `smart-split/types.ts` | 新增 `insertPageBreaks` 选项 |
| `smart-split/detectCrossings.ts` | 新增 `findPageStartPositions()` |
| `smart-split/SmartSplitPlugin.ts` | 新增 `syncPageBreaks()`，集成到 `performDetectionAndSplit` |

无需改动后端、CSS 或其他扩展。

## 边界情况

- **单页**（无 breakers）：`syncPageBreaks` 仅清理旧分页
- **sync 未找到节点**：仅清理，不添加新分页
- **同一周期内拆分 + sync**：拆分 dispatch 后重新读取 breaker 位置
- **空元素/隐藏元素**：跳过（与 `findCrossingPositions` 相同过滤逻辑）
- **装饰元素**：`posAtDOM` 失败时 try/catch 处理
