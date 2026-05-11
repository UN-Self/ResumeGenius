# SmartSplitExtension — 跨页元素智能拆分设计

**日期**: 2026-05-11
**分支**: feat/a4-multi-page
**状态**: 设计完成

## 问题

`tiptap-pagination-plus` 使用 CSS float 模拟 A4 分页，在页面之间插入 `.breaker` 浮动元素。当简历模块（如"项目经历"）横跨两页时，内容被 `.breaker` 遮挡或显示在页间间隙处。

## 方案：父容器拆分 + 空行推进

在 ProseMirror 文档层将跨页元素所在的父容器一分为二，并在两个半容器之间插入空 `<p></p>`。空行利用 float 模型自然将新容器推向下一页。

```
拆分前:                              拆分后:
<div class="skills">                <div class="skills" data-ss-parent="a1b2">
  .skill-item A                       .skill-item A
  .skill-item B  ← 跨页       →    </div>
  .skill-item C                     <p></p>                          ← 空行推进
                                    <div class="skills" data-ss-parent="a1b2">
                                      .skill-item B
                                      .skill-item C
                                    </div>
```

选择理由：
- 拆分容器让两个半段独立参与分页布局
- 空行 `<p>` 确保 float 模型将后半段推到下一页
- `data-ss-parent` 追踪拆分来源，支持后续合并或二次拆分
- 无级联风险（`ownDispatch` 标记防止循环）

## 文件结构

```
extensions/
  SmartSplitExtension.ts          ← TipTap Extension 入口
  smart-split/
    types.ts                      ← 类型定义与配置
    detectCrossings.ts            ← DOM 层检测（纯函数）
    SmartSplitPlugin.ts           ← ProseMirror Plugin（调度+防级联）
    splitTransaction.ts           ← 文档层操作（纯函数）
```

## 模块设计

### SmartSplitExtension.ts

TipTap Extension 包装器，注册 ProseMirror Plugin：

```ts
Extension.create({ name: 'smartSplit' })
  → addProseMirrorPlugins() → [smartSplitPlugin(this.options)]
```

### types.ts — 配置与类型

```ts
SmartSplitOptions {
  debounce: 300                  // 防抖 ms
  threshold: 0                   // 检测阈值 px
  parentAttr: 'data-ss-parent'   // 拆分兄弟追踪属性
}
```

| 类型 | 用途 |
|------|------|
| `BreakerPosition` | `.breaker` 的 `{ top, bottom }` Y 坐标 |
| `CrossingInfo` | 跨页元素的 `{ pos, breakerIndex }` |
| `BLOCK_TAGS` | 可跨页的块级 HTML 标签（DIV, SECTION, P, UL 等） |

### detectCrossings.ts — DOM 层检测

两个纯函数，不依赖 ProseMirror state：

| 函数 | 说明 |
|------|------|
| `getBreakerPositions(editorDom)` | `querySelectorAll('.breaker')` → `getBoundingClientRect()` |
| `findCrossingPositions(view, editorDom, breakers, threshold)` | TreeWalker 遍历 BLOCK_TAGS，`view.posAtDOM` 转 ProseMirror 位置 |

**跨页判定：**

```
rect.top < breaker.bottom && rect.bottom > breaker.top - threshold
```

`threshold` 控制"页边界上方多少像素也视为跨页"。默认 `0` = 元素底部刚好触及 breaker 顶部即判定。

### SmartSplitPlugin.ts — 调度与防级联

ProseMirror Plugin，协调检测→拆分流程：

```
view.update()
  → 检查 ownDispatch 标记（防级联）
  → debounce 300ms
  → getBreakerPositions()
  → findCrossingPositions()
  → 筛选第一个 depth≥2 且 index>0 的 crossing
  → buildSplitTransaction()
  → tr.setMeta(pluginKey, { ownDispatch: true })
  → view.dispatch(tr)
```

**防级联机制：**

```
Plugin.state.apply(tr)  → 记录 tr.getMeta(pluginKey)?.ownDispatch
view.update(view)       → 读取 pluginKey.getState()，true 则跳过
dispatch 前             → tr.setMeta(pluginKey, { ownDispatch: true })
```

ProseMirror `view.dispatch()` 是同步的——dispatch 时立即触发所有 plugin 的 `view.update()`。`ownDispatch` 标记防止自身 dispatch 触发新一轮检测。

**Crossing 筛选条件：**

- `$pos.depth >= 2`：有足够层级定位到父容器
- `$pos.index(depth-1) > 0`：crossing 不是父容器的第一个子元素（前半段非空）

### splitTransaction.ts — 文档层操作

将跨页元素的父容器一分为二，中间插入空 `<p>`：

```
doc.resolve(crossPos) → $pos
  ↓
定位直接父容器 (parentDepth = $pos.depth - 1)
  ↓
parent.forEach() → 按交叉索引拆分为 front[] + back[]
  ↓
生成/复用 data-ss-parent ID
  ↓
frontNode = parent.type.create(attrs, front[])
backNode  = parent.type.create(attrs, back[])
  ↓
rebuildAncestors(doc, $pos, parentDepth, frontNode, <p>, backNode)
  ↓
ReplaceDocStep(newDoc) → 替换整个文档
  ↓
返回 Transaction | null
```

**深度模型（以简历为例）：**

```
doc          (depth 0)
  div.resume  (depth 1)  ← 页容器
    div.section  (depth 2)  ← parentDepth = 2，直接父容器
      div.item A  (depth 3) [index 0] → front[]
      div.item B  (depth 3) [index 1] → crossing，split point
      div.item C  (depth 3) [index 2] → back[]
```

**属性复制规则：**

| 属性 | 处理 |
|------|------|
| `type` | 复制（节点类型不变） |
| `attrs.class` | 复制 |
| `attrs.style` | 复制 |
| `attrs.id` | 删除（id 唯一） |
| `data-ss-parent` | 复用已有值，无则生成随机短码 |

**祖先重建 (`rebuildAncestors`)：**

从 parentDepth 向上遍历到 depth=1，在每一层将原始节点替换为 `[frontNode, <p>, backNode]`（最底层）或 `[rebuiltAncestor]`（上层）。最顶层直接 `doc.copy()`。

**ReplaceDocStep：**

自定义 ProseMirror Step，绕过内容匹配检查（避免 ProseMirror 自动插入填充段落）。支持 `invert()` 实现撤销。

## 数据流

```
编辑器 DOM                    ProseMirror Doc
─────────────────             ──────────────────
.breaker → Y 坐标              doc.resolve(pos)
                                       ↓
TreeWalker → 跨页 DOM 元素     $pos.depth, $pos.index()
                                       ↓
view.posAtDOM(el,0) → pos     定位 parentDepth, front[]/back[]
                                       ↓
                               rebuildAncestors(front, <p>, back)
                                       ↓
                               ReplaceDocStep(newDoc)
                                       ↓
                               view.dispatch(tr)
                                       ↓
                               ProseMirror 同步渲染 → DOM 更新
```

## 边界情况

| 场景 | 处理 |
|------|------|
| crossing depth < 2 | 返回 null，无法定位父容器 |
| crossing 是父容器首子元素 (index=0) | 返回 null，front 为空 |
| 父容器仅有 1 个子元素 | 返回 null，无法拆分 |
| 拆分后半段 nodeSize ≤ 2 | 返回 null，避免产生空壳容器 |
| 无 `.breaker` | 直接跳过，不进入检测 |
| Plugin 自身 dispatch 触发 update | `ownDispatch` 标记跳过，防止级联 |
| 空元素 / 无文本 | TreeWalker 中跳过 |
| 非块级元素 (span 等) | `BLOCK_TAGS` 过滤，跳过 |
| 已拆分容器再次跨页 | 复用已有 `data-ss-parent`，二次拆分 |

## 测试覆盖

| 文件 | 数量 | 覆盖内容 |
|------|------|----------|
| `detectCrossings.test.ts` | 10 | breaker 定位、跨页判定、阈值、空元素/内联过滤 |
| `splitTransaction.test.ts` | 6+ | 拆分+插入 `<p>`、null 边界、首元素保护、属性复制、多层嵌套 |
| `SmartSplitExtension.test.ts` | 3 | Extension 注册、options 合并 |
| `SmartSplitPlugin.test.ts` | 6 | 防级联、debounce、ownDispatch 元数据 |
