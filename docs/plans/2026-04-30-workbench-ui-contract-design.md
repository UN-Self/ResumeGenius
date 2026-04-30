# Workbench UI 契约统一设计：Warm Editorial（暖色编辑风）

> 状态：已确认  |  日期：2026-04-30

## 1. 背景与动机

### 现状问题

Workbench 前端存在 **两套并行的色彩系统**：

| 系统 | 位置 | 主色 | 覆盖范围 |
|------|------|------|----------|
| Tailwind `@theme` | `src/index.css` | `#c4956a`（暖驼色） | ProjectList、ProjectDetail、弹窗、shadcn/ui |
| `:root` CSS 变量 | `src/styles/editor.css` | `#1a73e8`（Google 蓝色） | ActionBar、A4Canvas、FormatToolbar、FontSelector 等 15 个编辑器组件 |

此外，`@theme` 中 `secondary = muted = accent = #f0ebe4`，三者完全相同，语义冗余。

### 设计目标

1. **统一为 Warm Editorial 风格**：暖色、克制、专业，符合简历编辑产品气质
2. **消除双系统**：所有颜色通过 Tailwind `@theme` 语义 token 引用
3. **补全色阶**：为 hover/active/disabled 等交互状态提供精细色阶
4. **编辑器友好**：A4 画布在暖色背景上清晰呈现，选中态/焦点态品牌一致

### 设计参考

- 参考 ui-ux-pro-max 设计系统推荐
- 风格匹配：Editorial Grid / Magazine + Swiss Modernism 2.0
- 现有 Intake 模块暖色调为起点，精炼而非重写

---

## 2. 色彩系统

### 2.1 主色阶（Primary Scale）

以 `#c4956a` 为基准色（400），向上/下各延伸 5 级：

| Token | Hex | 用途 |
|-------|-----|------|
| `primary-50` | `#faf6f2` | 最浅背景（选中行、标签底色） |
| `primary-100` | `#f2e8e0` | 浅背景（hover 态、次级按钮 hover） |
| `primary-200` | `#e5d2c2` | 轻微强调（激活标签、进度条底色） |
| `primary-300` | `#d4b99a` | 边框强调（卡片 hover 边框） |
| `primary-400` | `#c4956a` | **基准主色**（主按钮背景、链接、图标） |
| `primary-500` | `#b3804d` | 深悬停（主按钮 hover、按压态） |
| `primary-600` | `#9c6b3a` | **文字用最小对比度色**（白底上 ≈4.6:1） |
| `primary-700` | `#7d5530` | 深色背景上的文字 |
| `primary-800` | `#5e4025` | 极深强调 |
| `primary-900` | `#3f2b1a` | 最深底色（深色模式预留） |
| `primary-950` | `#2a1c10` | 最深（深色模式预留） |

### 2.2 语义色（Semantic Colors）

| Token | Hex | 说明 |
|-------|-----|------|
| `background` | `#faf8f5` | 页面暖白基底（不变） |
| `foreground` | `#1a1815` | 近黑主文字（不变） |
| `card` | `#ffffff` | 卡片/面板背景 |
| `card-foreground` | `#1a1815` | 卡片内文字 |
| `secondary` | `#e8d5c4` | 次级背景（**从 `#f0ebe4` 分化**，更暖更有辨识度） |
| `secondary-foreground` | `#5c4a3a` | 次级文字（暖深棕） |
| `muted` | `#f5f1ed` | 弱化背景（**从 `#f0ebe4` 分化**，禁用/占位） |
| `muted-foreground` | `#8c8279` | 弱化文字（暖灰） |
| `accent` | `#d4a574` | 强调色（**从 `#f0ebe4` 分化**，警示/高亮） |
| `accent-foreground` | `#4a3020` | 强调色上的文字 |
| `border` | `#e4ddd5` | 通用边框（**从 `#e7e2db` 微调**） |
| `input` | `#e4ddd5` | 输入框边框（= border） |
| `ring` | `#c4956a` | 聚焦环（= primary-400） |
| `destructive` | `#d64545` | 危险操作（**暖调红替代冷 `#ef4444`**） |
| `destructive-foreground` | `#ffffff` | 危险操作文字 |

### 2.3 编辑器专用 Token（Tailwind `@theme` 中定义）

| Token | 值 | 用途 |
|-------|-----|------|
| `canvas-bg` | `#ede8e0` | 画布区背景（比 `background` 深约 2 色阶） |
| `canvas-shadow` | `0 2px 12px rgba(0,0,0,0.08)` | A4 纸投影 |
| `selection` | `rgba(196,149,106,0.22)` | 文本选中色（主色 22% 透明） |
| `surface-hover` | `#faf6f2` | 悬停微暖色（= primary-50） |

### 2.4 对比度合规

| 组合 | 对比度 | WCAG |
|------|--------|------|
| `foreground` on `background` | ≈18:1 | AAA |
| `primary-600` on `#ffffff` | ≈4.6:1 | AA |
| `muted-foreground` on `background` | ≈4.7:1 | AA |
| `destructive` on `#ffffff` | ≈4.8:1 | AA |

---

## 3. 字体系统

### 3.1 字体栈

| 角色 | 字体 | Weight | 来源 |
|------|------|--------|------|
| 页面标题 | Playfair Display | 500/600 | Google Fonts (OFL) |
| UI 正文/控件 | Inter | 400/500/600 | Google Fonts (OFL) |
| 代码/标签 | JetBrains Mono | 400/500 | Google Fonts (OFL) |

### 3.2 字号规格

| 层级 | 字体 | Weight | Size | Line-height |
|------|------|--------|------|-------------|
| 页面标题 | Playfair Display | 600 | 28px | 1.3 |
| 区块标题 | Playfair Display | 500 | 22px | 1.35 |
| 卡片标题 | Inter | 600 | 16px | 1.4 |
| 正文/UI | Inter | 400 | 14px | 1.6 |
| 辅助说明 | Inter | 400 | 12px | 1.5 |
| 代码/标签 | JetBrains Mono | 400 | 12px | 1.5 |

### 3.3 加载策略

```css
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&family=Playfair+Display:wght@500;600&display=swap');
```

- `display=swap`：防止 FOIT（不可见文本闪烁）
- 仅加载需用 weight，总加载量约 80KB（woff2）
- Tailwind 映射：`--font-sans: "Inter", ...` / `--font-serif: "Playfair Display", Georgia, serif` / `--font-mono: "JetBrains Mono", ...`

### 3.4 使用范围限定

- Playfair Display **仅用于页面级标题**（ProjectDetail 标题、EditorPage 项目名），不进入简历编辑区或 UI 控件
- 简历编辑区（ProseMirror）使用独立字体系统（用户在编辑时可选择 SimSun/Times New Roman 等商用字体）
- JetBrains Mono 用于工具栏快捷键标签、文件名、技术标识

---

## 4. A4 画布与布局

### 4.1 画布呈现

- **纸面**：`#ffffff`（纯白），210×297mm 比例
- **投影**：`0 2px 12px rgba(0,0,0,0.08)`（唯一使用阴影的元素）
- **画布区背景**：`#ede8e0`（比页面背景深约 2 色阶，利用同时对比原理衬托白色纸面）

### 4.2 三栏布局

```
左侧面板 (2.5fr)  |  中央画布区 (5fr)  |  右侧面板 (2.5fr)
bg-white          |  bg-canvas-bg      |  bg-white
border-r border   |                    |  border-l border
```

- 面板折叠：`width: 0` / `transition: 300ms ease`
- 折叠按钮：32×32px，位于面板边缘
- 现有 CSS Grid 类名（`.editor-workspace`、`.action-bar` 等）**保持不变**

### 4.3 响应式断点

| 宽度 | 行为 |
|------|------|
| ≥1440px | 左右两栏展开 |
| 1280-1439px | 仅左栏展开 |
| 1024-1279px | 左栏自动折叠，中央全宽 |
| <1024px | 单栏全宽，面板变为底部抽屉 |

### 4.4 画布缩放

- 默认自动适应（fit-to-width），手动缩放 50%-200%
- 缩放工具栏：底部浮层，32px 高，`bg-white`，`border border-border`
- 档位：50% / 75% / 100% / 125% / 150% / 200%

### 4.5 z-index 层级

| 层级 | 用途 |
|------|------|
| 0 | 内容区 |
| 10 | 悬浮缩放工具栏 |
| 20 | 面板折叠按钮 |
| 40 | 弹窗遮罩 / 弹窗 |
| 100 | Toast 通知 |

---

## 5. 组件规格

### 5.1 通用规则

- 所有交互元素 `min-h-10`（40px），触摸目标 ≥ 44px
- 过渡统一 `150ms ease`，弹窗出入 `200ms`
- 禁用态：`opacity-50 cursor-not-allowed`
- 聚焦：`focus-visible: ring-2 ring-ring/20 ring-offset-1`
- 圆角：`rounded-md`（6px 小元素）/ `rounded-lg`（8px 卡片）/ `rounded-xl`（12px 弹窗）
- **无 box-shadow**（Popover 除外），层次靠颜色和边框表达

### 5.2 按钮

| 变体 | Tailwind 类 |
|------|-------------|
| Primary | `bg-primary-400 text-white h-10 px-5 rounded-md hover:bg-primary-500 active:bg-primary-600 transition-colors` |
| Secondary | `bg-secondary text-secondary-foreground hover:bg-primary-100 border border-border` |
| Ghost | `bg-transparent text-muted-foreground hover:bg-surface-hover hover:text-foreground` |
| Destructive | `bg-destructive text-white hover:brightness-110` |

### 5.3 卡片

```
bg-card border border-border rounded-lg px-5 py-4
hover:border-primary-300 transition-border 150ms
```

### 5.4 输入框

```
bg-white border border-border rounded-md h-10 px-3
focus:border-primary-400 focus:ring-2 focus:ring-ring/20
placeholder:text-muted-foreground
```

### 5.5 工具栏

- **ActionBar**（顶部）：`h-14 bg-white border-b border-border`，按钮 `36×36px hover:bg-surface-hover`
- **FormatToolbar**（底部）：`h-12 bg-white border-t border-border`，分隔符 `w-[1px] h-5 bg-border mx-1`

### 5.6 弹窗

```
遮罩: fixed inset-0 bg-black/35 z-40
弹窗: bg-white rounded-xl max-w-md w-full p-6 z-40
```

无投影（扁平微交互）。

### 5.7 Popover（弹出菜单）

```
bg-white border border-border rounded-lg shadow-sm
animate-in fade-in-0 zoom-in-95 duration-150
```

Popover 是**唯一保留 `shadow-sm` 的组件**——字体选择器、颜色选择器等浮在画布上方，无投影会导致与 A4 纸内容视觉混合。

### 5.8 状态指示

| 状态 | 表现 |
|------|------|
| 保存中 | 文字 `text-muted-foreground` + 旋转图标 16px |
| 已保存 | "✓ 已保存" `text-primary-600` |
| 错误 | "⚠ 保存失败" `text-destructive` + 重试按钮 |
| 空状态 | 居中文案 `text-muted-foreground` + 浅色图标 |

---

## 6. 迁移策略

### 6.1 变量映射表

| 旧 `var(--color-*)` | 新 Tailwind 类 |
|---------------------|----------------|
| `--color-primary` | `bg-primary-400`（背景）/ `text-primary-600`（文字） |
| `--color-primary-hover` | `hover:bg-primary-500` |
| `--color-primary-bg` | `bg-primary-50` |
| `--color-page-bg` | `bg-background` |
| `--color-card` | `bg-card` |
| `--color-divider` | `border-border` |
| `--color-text-main` | `text-foreground` |
| `--color-text-secondary` | `text-muted-foreground` |
| `--color-text-disabled` | `text-muted-foreground/50` |

### 6.2 实施步骤

| Step | 内容 | 文件 |
|------|------|------|
| 1 | 更新 `index.css` `@theme` 块——精炼色值、补全 primary 色阶、新增 canvas/surface token、新增 `--font-serif` | `src/index.css` |
| 2 | 重构编辑器组件——将 `var(--color-*)` 替换为 Tailwind 语义类 | 见下表 |
| 3 | 删除 `editor.css` 中的 `:root { --color-* }` 块 | `src/styles/editor.css` |
| 4 | 验证——检查无 `var(--color-*)` 残留、确认对比度、运行测试 | 全项目 |

### 6.3 受影响文件及迁移顺序

| 优先级 | 文件 |
|--------|------|
| ① | `ActionBar.tsx`、`ToolbarButton.tsx`、`ToolbarSeparator.tsx`（基础组件） |
| ② | `A4Canvas.tsx`（核心视觉锚点） |
| ③ | `FontSelector.tsx`、`FontSizeSelector.tsx`、`LineHeightSelector.tsx`、`ColorPicker.tsx` |
| ④ | `AiPanelPlaceholder.tsx`、`SaveIndicator.tsx`、`ParsedSidebar.tsx`、`ParsedItem.tsx` |
| ⑤ | `EditorPage.tsx`、`ProjectDetail.tsx`（页面级） |
| ⑥ | 清理 `editor.css` |

### 6.4 不做的事

- **不改变**现有 CSS Grid 布局类名（`.editor-workspace`、`.action-bar`、`.canvas-area` 等）
- **不改变** shadcn/ui `components.json` 配置（`zinc` 基础色保留——shadcn 用它生成内部 CSS 变量）
- **不改变** ProseMirror 排版样式（简历编辑区独立字体系统）
- **不改变** Intake 模块中已使用 Tailwind 类的页面（ProjectList、ProjectDetail 弹窗等）

---

## 7. 参考

- ui-ux-pro-max 设计系统：Exaggerated Minimalism / Editorial Grid Magazine
- 字体许可：Playfair Display、Inter、JetBrains Mono 均为 SIL Open Font License（免费可商用）
- WCAG 2.1 AA 对比度标准：正常文字 ≥ 4.5:1
