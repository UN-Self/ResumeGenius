# 移除 TipTap 编辑器底部 FormatToolbar

## 背景

编辑器底部固定了一个 `FormatToolbar`，与浮动 `BubbleToolbar` 功能高度重叠（都包含字体、字号、加粗、斜体、颜色、列表、对齐等格式控制）。移除底部工具栏可简化界面，让编辑区域占满更多空间。

## 设计决策

- 完全删除 `FormatToolbar`（不保留隐藏/降级选项）
- 保留 `BubbleToolbar`（选中文本时出现）和 `ContextMenu`（右键）
- 保留 `ToolbarButton.tsx`、`ToolbarSeparator.tsx` 等共享组件

## 变更范围

### 删除文件

- `frontend/workbench/src/components/editor/FormatToolbar.tsx`

### 修改文件

1. `frontend/workbench/src/pages/EditorPage.tsx` — 移除 FormatToolbar 的 import 和 `<div className="format-toolbar">` 渲染代码
2. `frontend/workbench/src/styles/editor.css` — 移除 `.format-toolbar` 样式规则

### 不受影响

- `BubbleToolbar.tsx`、`ContextMenu.tsx`
- `ToolbarButton.tsx`、`ToolbarSeparator.tsx`（BubbleToolbar 仍在使用）
- 无数据库、API、测试文件变更

## 移除后的编辑器布局

```
+---------------------------------------+
|  ActionBar          (顶部, 56px)       |
+---------------------------------------+
|                                       |
|  A4Canvas           (flex-1, 可滚动)   |
|    +---------------------------+      |
|    |  BubbleMenu (浮动)        |      |
|    +---------------------------+      |
|                                       |
+---------------------------------------+
|  ContextMenu        (右键, fixed)      |
+---------------------------------------+
```
