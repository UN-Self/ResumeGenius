# TipTap 排版功能设计

> 日期：2026-04-30
> 状态：Approved

## 目标

为 TipTap 编辑器新增 5 个排版功能：字体选择器、字号选择器、字体/高亮颜色选择器、行距调整、右对齐。

## 技术方案

**架构：** TextStyleKit + 独立 Popover 子组件

- 唯一新增依赖：`@tiptap/extension-text-style@^3.22.4`
- 使用 `TextStyleKit` 一行注册全部 6 个内置扩展（TextStyle、Color、BackgroundColor、FontFamily、FontSize、LineHeight）
- 零自定义 Mark
- UI 组件：shadcn/ui Popover

## 扩展注册

**变更文件：** `frontend/workbench/src/pages/EditorPage.tsx`

```ts
import { TextStyleKit } from '@tiptap/extension-text-style'

extensions: [
  StarterKit,
  Underline,
  TextAlign.configure({ types: ['heading', 'paragraph'] }),
  TextStyleKit,
]
```

右对齐由现有 `TextAlign` 扩展支持（`'right'`），仅需在工具栏添加按钮。

## 工具栏布局

```
[字体▾ | 字号▾] ─ [B I U | A色 🖍高亮] ─ [H1 H2 H3] ─ [·列表 1.列表] ─ [行距▾ | ←左 | ↗居中 | →右 | ≡两端]
```

## 组件结构

### 新增文件

| 文件 | 职责 |
|------|------|
| `components/editor/FontSelector.tsx` | 字体下拉选择 |
| `components/editor/FontSizeSelector.tsx` | 字号下拉选择 |
| `components/editor/ColorPicker.tsx` | 字体色 + 高亮色选择 |
| `components/editor/LineHeightSelector.tsx` | 行距下拉选择 |
| `components/ui/popover.tsx` | shadcn/ui Popover（CLI 安装） |

### 变更文件

| 文件 | 变更 |
|------|------|
| `pages/EditorPage.tsx` | extensions 数组新增 TextStyleKit |
| `components/editor/FormatToolbar.tsx` | 引入 4 个选择器组件 + 右对齐按钮 |

### 统一组件接口

```ts
interface SelectorProps {
  editor: Editor
}
```

## 数据配置

### 字体（8 款中英混合）

| 显示名 | font-family 值 |
|--------|----------------|
| 默认字体 | (unset) |
| 宋体 | SimSun, serif |
| 黑体 | SimHei, sans-serif |
| 楷体 | KaiTi, serif |
| 仿宋 | FangSong, serif |
| Times New Roman | "Times New Roman", serif |
| Arial | Arial, sans-serif |
| Georgia | Georgia, serif |

### 字号（6 档）

10、12、14、16、18、24（单位 pt）

### 颜色（预设 + 自定义）

预设色板 18 色：
```
#000000 #434343 #666666 #999999 #b7b7b7 #cccccc #d9d9d9 #efefef #f3f3f3 #ffffff
#e06666 #f6b26b #ffd966 #93c47d #76a5af #6fa8dc #8e7cc3 #c27ba0
```
底部附加原生 `<input type="color">` 自定义选色。

### 行距（6 档）

1.0、1.15、1.5、1.75、2.0、2.5

## TipTap Commands

| 功能 | 设置命令 | 重置命令 |
|------|---------|---------|
| 字体 | `setFontFamily(value)` | `unsetFontFamily()` |
| 字号 | `setFontSize('10pt')` | `unsetFontSize()` |
| 字体色 | `setColor('#000000')` | `unsetColor()` |
| 高亮色 | `setBackgroundColor('#ffd966')` | `unsetBackgroundColor()` |
| 行距 | `setLineHeight('1.5')` | `unsetLineHeight()` |
| 右对齐 | `setTextAlign('right')` | — |

## 样式与交互

### 下拉触发按钮

- 复用 ToolbarButton 视觉风格（44px 最小高度，蓝色激活态）
- 带文本标签的按钮显示当前值 + 三角箭头

### Popover 行为

- `side="top"`（工具栏在底部，向上弹出）
- 选择后自动关闭
- 点击外部关闭（默认行为）

### Active 状态追踪

在 `getActiveStates()` 中通过 `editor.getAttributes('textStyle')` 读取当前样式属性，监听 `transaction` 事件更新。

## 不涉及的变更

- 不改动 ToolbarButton、ToolbarSeparator 组件
- 不改动 editor.css ProseMirror 样式
- 不新增 shadcn/ui 以外的 UI 依赖
