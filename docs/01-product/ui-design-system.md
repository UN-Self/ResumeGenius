# ResumeGenius UI 设计系统

更新时间：2026-05-01

本文档定义 ResumeGenius 的统一 UI 风格规范。所有模块的前端实现必须遵循此规范。

设计 Token 唯一数据源：`frontend/workbench/src/index.css` 中的 `@theme` 块。

---

## 1. 设计理念

### 1.1 风格定义：Warm Editorial（暖色编辑风）

三个关键词：

- **Warm**：暖色驼调（Camel Tone）主色系，营造温和、可信赖的编辑氛围
- **Editorial**：排版优先，信息层级清晰，参考杂志/编辑器界面
- **Professional**：克制、不喧宾夺主，界面服务于内容

### 1.2 设计参考

- 风格匹配：Editorial Grid Magazine + Swiss Modernism 2.0
- 以现有 Intake 模块暖色调为起点，精炼而非重写

### 1.3 核心原则

- 界面服务于内容，不喧宾夺主
- 信息层级清晰，功能区域划分明确
- 交互直觉化，减少学习成本
- 层次靠颜色和边框表达，避免滥用阴影

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
| `background` | `#faf8f5` | 页面暖白基底 |
| `foreground` | `#1a1815` | 近黑主文字 |
| `card` | `#ffffff` | 卡片/面板背景 |
| `card-foreground` | `#1a1815` | 卡片内文字 |
| `popover` | `#ffffff` | 弹出菜单背景 |
| `popover-foreground` | `#1a1815` | 弹出菜单文字 |
| `primary` | `#9c6b3a` | 主色（= primary-600，白字可达 AA） |
| `primary-foreground` | `#ffffff` | 主色上的文字 |
| `secondary` | `#e8d5c4` | 次级背景 |
| `secondary-foreground` | `#5c4a3a` | 次级文字（暖深棕） |
| `muted` | `#f5f1ed` | 弱化背景（禁用/占位） |
| `muted-foreground` | `#8c8279` | 弱化文字（暖灰） |
| `accent` | `#d4a574` | 强调色（警示/高亮） |
| `accent-foreground` | `#4a3020` | 强调色上的文字 |
| `destructive` | `#d64545` | 危险操作（暖调红） |
| `destructive-foreground` | `#ffffff` | 危险操作文字 |
| `border` | `#e4ddd5` | 通用边框 |
| `input` | `#e4ddd5` | 输入框边框（= border） |
| `ring` | `#c4956a` | 聚焦环（= primary-400） |

### 2.3 编辑器专用 Token

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
| `#ffffff` on `primary-600` | ≈4.6:1 | AA |
| `primary-600` on `#ffffff` | ≈4.6:1 | AA |
| `muted-foreground` on `background` | ≈4.7:1 | AA |
| `destructive` on `#ffffff` | ≈4.8:1 | AA |

### 2.5 使用规则

- 同一页面主色按钮不超过 1 个（引导用户关注主要操作）
- 语义色只用于状态反馈，不用于装饰
- **无 box-shadow**（Popover 除外），层次靠颜色和边框表达
- 所有颜色通过 Tailwind 语义 token 引用，禁止硬编码色值

---

## 3. 字体系统

### 3.1 字体栈

| 角色 | 字体 | Weight | 来源 |
|------|------|--------|------|
| 页面标题 | Playfair Display | 500/600 | Google Fonts (SIL OFL) |
| UI 正文/控件 | Inter | 400/500/600 | Google Fonts (SIL OFL) |
| 代码/标签 | JetBrains Mono | 400/500 | Google Fonts (SIL OFL) |

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
- Tailwind 映射：`--font-sans` / `--font-serif` / `--font-mono`

### 3.4 使用范围限定

- **Playfair Display 仅用于页面级标题**（ProjectDetail 标题、EditorPage 项目名），不进入简历编辑区或 UI 控件
- 简历编辑区（ProseMirror）使用独立字体系统（用户在编辑时可选择 SimSun/Times New Roman 等商用字体）
- JetBrains Mono 用于工具栏快捷键标签、文件名、技术标识

### 3.5 排版规则

- 标题不加斜体
- 正文不加粗（加粗只用于 table header、强调标记）
- 中英文之间不加额外空格（由排版引擎处理）

---

## 4. 间距与圆角

### 4.1 间距系统

基准单位：4px。所有间距必须是 4 的倍数。

| 用途 | 值 |
|---|---|
| 组件内部间距 | 8px (2x) |
| 组件间间距 | 12px (3x) |
| 区域间间距 | 16px (4x) |
| 区块间间距 | 24px (6x) |
| 页面边距 | 24px (6x) |

### 4.2 圆角

| 用途 | 值 | Tailwind |
|---|---|---|
| 标签 (Tag/Badge) | 4px | `rounded-sm` |
| 按钮、输入框 | 6px | `rounded-md` |
| 卡片、面板、Popover | 8px | `rounded-lg` |
| 弹窗/Modal | 12px | `rounded-xl` |

---

## 5. 布局系统

### 5.1 工作台主布局

```
┌─────────────────────────────────────────────────────────────┐
│  Logo   项目名   |  保存状态  版本历史  导出PDF(付费)         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌───────────────┐  ┌──────────────────┐  ┌──────────────┐ │
│  │ 左侧面板      │  │                  │  │ AI 助手      │ │
│  │ (2.5fr)       │  │   A4 简历画布    │  │ (2.5fr)     │ │
│  │               │  │  （TipTap 编辑）  │  │              │ │
│  │ bg-white      │  │  bg-canvas-bg    │  │ bg-white    │ │
│  │ border-r      │  │                  │  │ border-l    │ │
│  │               │  │  ┌────────────┐  │  │              │ │
│  │               │  │  │ 简历内容    │  │  │  用户消息    │ │
│  │               │  │  │ #ffffff    │  │  │  AI 回复     │ │
│  │               │  │  └────────────┘  │  │              │ │
│  └───────────────┘  │                  │  └──────────────┘ │
│                     └──────────────────┘                   │
├─────────────────────────────────────────────────────────────┤
│  TipTap 工具栏：B I U | 字号 | 颜色 | 对齐 | 行距          │
└─────────────────────────────────────────────────────────────┘
```

- 三栏 CSS Grid：左侧面板 (2.5fr) | 中央画布区 (5fr) | 右侧面板 (2.5fr)
- 面板折叠：`width: 0` / `transition: 300ms ease`，折叠按钮 32×32px 位于面板边缘
- TipTap 工具栏在画布底部或浮动
- 现有 CSS Grid 类名（`.editor-workspace`、`.action-bar` 等）保持不变

### 5.2 各页面布局

| 页面 | 主区域 | 说明 |
|---|---|---|
| 项目首页 | 项目卡片列表 | 新建项目入口 |
| 工作台 | A4 画布 + AI 面板 | 核心编辑页面 |
| 版本历史 | 弹窗或侧边抽屉 | 版本列表 + 回退操作 |

### 5.3 响应式断点

| 宽度 | 行为 |
|------|------|
| ≥1440px | 左右两栏展开 |
| 1280-1439px | 仅左栏展开 |
| 1024-1279px | 左栏自动折叠，中央全宽 |
| <1024px | 单栏全宽，面板变为底部抽屉 |

### 5.4 A4 画布

- **纸面**：`#ffffff`（纯白），210×297mm 比例
- **投影**：`0 2px 12px rgba(0,0,0,0.08)`（唯一使用阴影的元素）
- **画布区背景**：`#ede8e0`（比页面背景深约 2 色阶，利用同时对比原理衬托白色纸面）
- **缩放**：默认自动适应（fit-to-width），手动缩放 50%-200%
- **缩放工具栏**：底部浮层，32px 高，`bg-white`，`border border-border`

### 5.5 z-index 层级

| 层级 | 用途 |
|------|------|
| 0 | 内容区 |
| 10 | 悬浮缩放工具栏 |
| 20 | 面板折叠按钮 |
| 40 | 弹窗遮罩 / 弹窗 |
| 100 | Toast 通知 |

---

## 6. 组件规格

### 6.1 通用规则

- 所有交互元素 `min-h-10`（40px），触摸目标 ≥ 44px
- 过渡统一 `150ms ease`，弹窗出入 `200ms`
- 禁用态：`opacity-50 cursor-not-allowed`
- 聚焦：`focus-visible:ring-2 ring-ring/20 ring-offset-1`
- **无 box-shadow**（Popover 除外），层次靠颜色和边框表达

### 6.2 按钮

| 变体 | Tailwind 类 |
|------|-------------|
| Primary | `bg-primary-600 text-white h-10 px-5 rounded-md hover:bg-primary-700 active:bg-primary-800 transition-colors` |
| Secondary | `bg-secondary text-secondary-foreground hover:bg-primary-100 border border-border` |
| Ghost | `bg-transparent text-muted-foreground hover:bg-surface-hover hover:text-foreground` |
| Destructive | `bg-destructive text-white hover:brightness-110` |

### 6.3 卡片

```
bg-card border border-border rounded-lg px-5 py-4
hover:border-primary-300 transition-border 150ms
```

### 6.4 输入框

```
bg-white border border-border rounded-md h-10 px-3
focus:border-primary-400 focus:ring-2 focus:ring-ring/20
placeholder:text-muted-foreground
```

- 高度 36-40px
- 错误态：边框变 `destructive` + 下方错误文字

### 6.5 标签页 (Tabs)

- 底部指示线样式，非填充式
- 选中态：`text-primary-400` 指示线 + 主色文字
- 未选中态：无指示线 + `text-muted-foreground`

### 6.6 工具栏

| 工具栏 | 样式 |
|--------|------|
| ActionBar（顶部） | `h-14 bg-white border-b border-border`，按钮 `36×36px hover:bg-surface-hover` |
| FormatToolbar（底部） | `h-12 bg-white border-t border-border`，分隔符 `w-[1px] h-5 bg-border mx-1` |

### 6.7 弹窗 (Modal)

```
遮罩: fixed inset-0 bg-black/35 z-40
弹窗: bg-white rounded-xl max-w-md w-full p-6 z-40
```

无投影（扁平微交互）。

### 6.8 Popover（弹出菜单）

```
bg-white border border-border rounded-lg shadow-sm
animate-in fade-in-0 zoom-in-95 duration-150
```

Popover 是**唯一保留 `shadow-sm` 的组件**——字体选择器、颜色选择器等浮在画布上方，无投影会导致与 A4 纸内容视觉混合。

### 6.9 AI 对话面板

- 宽度 360px（右侧面板）
- 聊天气泡样式：用户消息靠右（`bg-primary-400`），AI 回复靠左（`bg-muted`）
- "应用到简历"按钮为 Primary
- "继续对话"按钮为 Ghost
- 流式输出时显示打字动画

### 6.10 状态指示

| 状态 | 表现 |
|------|------|
| 保存中 | 文字 `text-muted-foreground` + 旋转图标 16px |
| 已保存 | "✓ 已保存" `text-primary-600` |
| 错误 | "⚠ 保存失败" `text-destructive` + 重试按钮 |
| 空状态 | 居中文案 `text-muted-foreground` + 浅色图标 |

---

## 7. 动效

| 类型 | 时间 | 缓动 |
|------|------|------|
| 交互反馈（hover、focus、press） | 150ms | ease |
| 面板展开/折叠 | 300ms | ease |
| 弹窗出入 | 200ms | ease |
| AI 流式输出 | — | 逐字显示 |
| 页面切换 | 无动画 | v1 直接切换 |

- 禁止装饰性动效（粒子、弹跳等）
- 尊重 `prefers-reduced-motion` 用户偏好

---

## 8. 技术实现

### 8.1 技术栈

| 层 | 技术 |
|---|---|
| CSS 框架 | Tailwind CSS v4（通过 `@tailwindcss/vite` 插件集成） |
| 组件库 | shadcn/ui（new-york 风格，zinc 基础色，lucide 图标） |
| 富文本编辑器 | TipTap（基于 ProseMirror） |
| 禁止引入 | Ant Design、Material UI 等其他 UI 框架 |

### 8.2 单一数据源

所有设计 Token 定义在 `frontend/workbench/src/index.css` 的 `@theme` 块中。

- 颜色、字号、间距通过 Tailwind CSS 变量统一管理
- 不使用 `:root` CSS 变量或硬编码色值
- TipTap 编辑器自定义主题以匹配整体风格
- ProseMirror 简历编辑区使用独立字体系统（用户可选字体）

### 8.3 禁止事项

- 禁止在组件中硬编码 `#xxx` 色值，必须使用 Tailwind 语义 token
- 禁止引入 Tailwind `@theme` 以外的自定义 CSS 变量
- 禁止使用 box-shadow（Popover 除外）
- 禁止装饰性动效
