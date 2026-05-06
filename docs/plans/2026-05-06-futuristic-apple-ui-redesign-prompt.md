# ResumeGenius 科技感 Apple-like UI 重设计提示词

**日期**: 2026-05-06
**目标**: 将 ResumeGenius 从当前偏空白、传统的界面，重设计为具有未来科技感、Apple-like 精致感、玻璃拟态和细腻动效的简历 AI 工作台，同时完整支持日间/夜间模式与多套品牌色调。

## 决策记录

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 视觉方向 | Futuristic Cupertino Workspace | 既保留 Apple-like 的克制、留白、材质感，又加入 AI 产品应有的生成感和科技氛围 |
| 参考风格 | 图 2/3 的深色生成工作台氛围 | 半透明面板、暗色画布、发光边界、分步生成状态和右侧预览区符合目标产品气质 |
| 主题策略 | Mode x Palette 双层主题 | 模式负责明暗，色调负责品牌气质，可加入同事文档中的 Warm Editorial / Quiet Luxury |
| 动效策略 | 高级但克制 | 文字、按钮、面板、生成状态都有动效，但不影响简历编辑效率 |
| 核心约束 | 简历画布优先清晰 | A4 简历区域必须保持专业、干净、可读，不被炫光和背景特效干扰 |
| 简历卡片 | 模板预览式图库卡片 | 替代简单长条列表，让项目首页直接呈现简历成品感和模板选择感 |

## 0. 最新 dev 对齐检查

当前分支：`dev` = `origin/dev`，最新提交为 `16c7c8e fix: 修复 AI API 连通性问题`。本提示词文档是未跟踪新文件，尚未与 dev 代码产生直接 merge 冲突。

### 0.1 已发现的前端结构变化

| 最新 dev 变化 | 对美术大改的影响 | 调整结论 |
|---------------|------------------|----------|
| `FormatToolbar.tsx` 已删除 | 不能再规划底部固定格式工具栏 | 改为重设计 `BubbleToolbar` + `ContextMenu` + `AlignSelector` |
| 新增 `BubbleToolbar.tsx` | 选中文本时出现浮动格式工具条 | 美术方案应强化浮动层，而不是恢复底部工具栏 |
| 新增 `ContextMenu.tsx` | 右键菜单承载撤销/重做/剪切/复制/粘贴/全选 | 需要纳入 glass / theme / z-index 设计 |
| 新增 `AlignSelector.tsx` | 对齐从 4 个按钮改为 dropdown | 主题和动效要覆盖 dropdown trigger 和菜单项 |
| `A4Canvas.tsx` 使用 `WatermarkOverlay` | 预览水印与打印保护是当前业务约束 | 美术大改不能移除、遮挡或降低水印层级 |
| `EditorPage.tsx` 拦截 copy 为纯文本 | 内容保护依赖剪贴板逻辑 | ContextMenu 的 copy/cut 样式可以改，保护逻辑不能改弱 |
| `index.css` 使用 Tailwind v4 `@theme` | 当前 token 数据源是 `--color-*` 变量 | 多主题应复用这些变量名，通过 `[data-theme]` 覆盖，而不是另起孤立变量 |

### 0.2 冲突处理原则

- 不恢复 `FormatToolbar`，不新增底部固定格式栏。
- `BubbleToolbar` 是核心格式入口，视觉上可升级为 Apple-like 浮动玻璃工具条。
- `ContextMenu` 是核心右键入口，必须保留纯文本复制/剪切保护逻辑。
- `WatermarkOverlay` 与 print protection 不属于装饰层，不能被背景特效、玻璃层或 z-index 调整破坏。
- 多主题实现必须沿用当前 Tailwind v4 `@theme` 的 `--color-*` 命名，避免与现有 utility class 脱节。

## 1. 可直接使用的总提示词

```text
请重设计 ResumeGenius 的整体 UI。产品是一个 AI 简历生成与编辑工作台，目标视觉风格是 futuristic Apple-like / Cupertino-inspired / premium AI workspace。

当前界面过于空白和传统，需要升级为有科技感、期待感和高级质感的界面。参考图 2、图 3 的感觉：深色沉浸式工作台、半透明玻璃面板、微弱发光边界、AI 正在生成的分步状态列表、左侧对话/任务流、右侧实时预览区域、浮动气泡工具条和右键菜单。整体要像 Apple Pro 应用和现代 AI IDE 的结合，但不要做成浮夸游戏 UI。

主题不要只做简单日间/夜间两套。请设计 Mode x Palette 的主题系统：Mode 分为 light / dark，Palette 包含 Futuristic Apple、Warm Editorial、Quiet Luxury 三套色调。Futuristic Apple 用于默认科技感版本；Warm Editorial 和 Quiet Luxury 来自现有同事文档，用于保留简历产品的纸张温度、暖驼色和克制专业感。主题选择器第一项是“跟随系统”：系统夜间使用科技夜，系统日间使用银白。所有主题共享语义 token，支持一键切换或后续扩展。

请重构所有相关页面和组件，不要只改首页或工作台：
1. 登录页：品牌入口、登录表单、错误提示、加载状态、主题切换。
2. 项目/简历列表页：顶部品牌栏、筛选/搜索、简历图库卡片、空状态、退出登录。
3. 新增简历入口：不要使用简单输入框 + 创建按钮；在列表网格中放一个带 `+` 的“新建简历”卡片，形态与简历卡片一致。
4. Intake / 项目详情页：上传、Git 仓库、补充说明、资料列表、解析状态、生成简历入口。
5. Workbench 编辑页：左侧资料/步骤面板，中间 A4 简历编辑画布，右侧 AI 对话面板，顶部 ActionBar，选中文字时出现 BubbleToolbar，右键出现 ContextMenu。
6. AI 生成状态：加载中、读取资料、生成 HTML、优化结构、导出 PDF 等步骤需要有进度感和完成状态。
7. 弹窗与浮层：上传弹窗、Git 弹窗、备注编辑、删除确认、版本历史、导出状态、Popover、Dropdown、Toast。
8. Marketing 页面：首页、features、pricing、help、导航、页脚，需要与工作台统一视觉语言。

交互动效要求：
- 页面首次进入使用 stagger reveal，标题、输入框、卡片依次出现。
- 标题文字可使用轻微 glow / gradient sweep / blur-to-sharp reveal，但必须保持可读。
- 主按钮需要有 magnetic hover、光泽扫过、按压缩放、焦点光环。
- 次按钮使用玻璃描边、hover 时边界发光和背景亮度变化。
- 卡片 hover 时轻微上浮、边框亮起、内部高光跟随鼠标方向移动。
- AI 生成步骤使用 spinner / progress ring / pulse dot / checkmark morph。
- 输入框 focus 时出现柔和光晕，placeholder 不能跳动。
- 右侧 AI 面板消息流使用轻微 slide-up + fade-in。
- 所有动效必须支持 prefers-reduced-motion，降级为无位移动画或纯 opacity。

布局要求：
- 不要做普通 landing page 风格的大空白 Hero。第一屏要直接像一个真实可用的 AI 简历工作台。
- Workbench 使用三栏布局：左侧任务/资料，中间 A4 简历画布，右侧 AI 对话或预览。
- 中间 A4 简历画布保持白纸质感和专业排版，夜间模式下也不要把简历纸变黑。
- 项目/简历列表不要使用简单长条卡片。改成类似简历模板市场的网格卡片：每张卡片上方是 A4 简历缩略图预览，下方是标题、更新时间、状态、更多操作。缩略图需要看起来像真实简历页面，而不是普通白色占位块。
- 新建入口使用同尺寸 `+` 简历卡片：卡片中间是发光 `+` 图标和“新建简历”，hover 时显示边框高光、轻微上浮和缩略图网格背景。
- 工具栏采用紧凑 icon button，优先使用 lucide-react 图标。
- 面板圆角控制在 8-14px，按钮 8-12px；不要过度圆润。
- 保持移动端可用：三栏在小屏变为顶部/主内容/底部抽屉或 tab。

技术实现建议：
- 使用现有 React + TypeScript + Tailwind CSS v4 架构，不引入大型 UI 框架。
- 以 `frontend/workbench/src/index.css` 的 Tailwind `@theme` 为 token 命名源，继续使用 `--color-*` 变量名。
- 多主题通过 `[data-theme="..."]` 或 `[data-palette="..."]` 覆盖同名 `--color-*` 变量，不另起一套与 Tailwind utility 脱节的 CSS 变量。
- 动效优先用 CSS transition/keyframes；复杂交互可少量使用 requestAnimationFrame，但不要引入重型动画库。
- 所有颜色、阴影、玻璃材质、发光边界都通过 token 管理，避免散落 magic value。
```

## 2. 主题与色调系统

### 2.0 主题模型

主题采用两层结构：`mode` 负责明暗，`palette` 负责品牌色调。

```ts
type ThemeMode = 'light' | 'dark'
type ThemePalette = 'futuristic-apple' | 'warm-editorial' | 'quiet-luxury'
```

| Palette | 来源/用途 | 气质 |
|---------|-----------|------|
| `futuristic-apple` | 本次重设计默认 | 银白玻璃、石墨黑、冰蓝高光、Apple-like 科技感 |
| `warm-editorial` | `docs/01-product/ui-design-system.md` / Intake 同事文档 | 暖驼色、纸张温度、编辑器专业感 |
| `quiet-luxury` | Intake 文档暗色备用方案 | 深棕黑、暖金高光、克制高级 |

语义 token 保持一致，具体色值由 `mode + palette` 决定：

```css
--color-background
--color-foreground
--color-card
--color-card-foreground
--color-popover
--color-popover-foreground
--color-primary
--color-primary-foreground
--color-secondary
--color-secondary-foreground
--color-muted
--color-muted-foreground
--color-accent
--color-accent-foreground
--color-destructive
--color-border
--color-input
--color-ring
--color-canvas-bg
--color-surface-hover
--color-resume-paper
--color-border-glow
```

当前 `index.css` 已有大部分 `--color-*` token。美术大改只新增缺失的运行时 token，例如 `--color-resume-paper`、`--color-border-glow`、`--color-surface-glass`，并保持已有 Tailwind utility 可继续工作。

### 2.1 Futuristic Apple / 夜间模式

| 角色 | 建议色值 | 用途 |
|------|----------|------|
| `background` | `#05070d` | 页面基底，接近黑但保留蓝调 |
| `surface` | `rgba(18, 22, 34, 0.72)` | 主面板玻璃背景 |
| `surface-strong` | `rgba(28, 33, 48, 0.86)` | 输入框、步骤卡、弹层 |
| `border` | `rgba(180, 205, 255, 0.16)` | 默认边界 |
| `border-glow` | `rgba(104, 194, 255, 0.48)` | hover / active 发光边界 |
| `text` | `#f5f7fb` | 主文字 |
| `muted` | `#8e98aa` | 次级文字 |
| `accent` | `#6bdcff` | 主高亮，按钮、进度、焦点 |
| `accent-2` | `#9dffb0` | 成功、完成、生成步骤 |
| `danger` | `#ff6b8a` | 错误和危险操作 |

夜间模式关键词：`deep graphite`, `liquid glass`, `subtle neon edge`, `soft bloom`, `AI console`, `pro workspace`。

### 2.2 Futuristic Apple / 日间模式

| 角色 | 建议色值 | 用途 |
|------|----------|------|
| `background` | `#f5f8fc` | 银白冷调背景 |
| `surface` | `rgba(255, 255, 255, 0.76)` | 半透明玻璃面板 |
| `surface-strong` | `#ffffff` | 表单、卡片、弹层 |
| `border` | `rgba(28, 45, 72, 0.12)` | 默认边界 |
| `border-glow` | `rgba(0, 132, 255, 0.34)` | hover / focus 边界 |
| `text` | `#101522` | 主文字 |
| `muted` | `#667085` | 次级文字 |
| `accent` | `#007aff` | Apple-like 蓝色主高亮 |
| `accent-2` | `#20c997` | 成功、完成、生成步骤 |
| `danger` | `#e5484d` | 错误和危险操作 |

日间模式关键词：`silver glass`, `frosted white`, `ice blue highlight`, `clean professional`, `calm premium`。

### 2.3 Warm Editorial 色调

来自现有同事文档，适合“简历编辑、纸张质感、可信赖”的页面，也可以作为用户可选主题。

| 角色 | 建议色值 | 用途 |
|------|----------|------|
| `background` | `#faf8f5` | 页面暖白基底 |
| `surface` | `#ffffff` | 卡片、弹窗、表单 |
| `surface-strong` | `#f5f1ed` | 弱化面板、占位区 |
| `border` | `#e4ddd5` | 通用边框 |
| `border-glow` | `#d4b99a` | hover / focus 边界 |
| `text` | `#1a1815` | 主文字 |
| `muted` | `#8c8279` | 次级文字 |
| `accent` | `#c4956a` | 主按钮、链接、选中态 |
| `accent-hover` | `#b3804d` | 主按钮 hover |
| `accent-bg` | `#faf6f2` | 标签、选中背景 |

关键词：`warm paper`, `camel tone`, `editorial grid`, `professional`, `resume craft`。

### 2.4 Quiet Luxury 色调

来自 Intake 文档的暗色备用方向，适合更低调的高级暗色主题。

| 角色 | 建议色值 | 用途 |
|------|----------|------|
| `background` | `#1c1917` | 页面深暖背景 |
| `surface` | `#292524` | 卡片、弹窗 |
| `surface-strong` | `#3a3632` | 输入框、步骤卡 |
| `border` | `#44403c` | 通用边框 |
| `border-glow` | `#d4a574` | hover / focus 边界 |
| `text` | `#fafaf9` | 主文字 |
| `muted` | `#a8a29e` | 次级文字 |
| `accent` | `#d4a574` | 主按钮、链接 |
| `accent-bg` | `#3a3632` | 标签、选中背景 |

关键词：`quiet luxury`, `warm graphite`, `champagne accent`, `premium editor`。

### 2.5 色调使用规则

- 主题选择器必须提供“跟随系统”，夜间解析为 `futuristic-apple + dark`，日间解析为 `futuristic-apple + light`。
- 日常编辑可提供 `futuristic-apple + light` 或 `warm-editorial + light`。
- 简历 A4 纸永远使用 `--resume-paper: #ffffff`，不随暗色模式变黑。
- 背景可以有玻璃、发光、粒子、网格，但不能穿透干扰 A4 简历内容。
- 所有 palette 都必须覆盖 hover / active / focus / disabled / loading / selected 状态。

## 3. 动效规范

| 元素 | 动效 | 参数建议 |
|------|------|----------|
| 页面入场 | stagger reveal | `opacity 0 -> 1`, `translateY(12px -> 0)`, 60ms 间隔 |
| 标题 | blur-to-sharp + subtle glow | 500-700ms，夜间 glow 更明显，日间更克制 |
| 主按钮 | hover 光泽扫过 + 上浮 | `translateY(-1px)`, `scale(1.01)`, active `scale(0.98)` |
| 次按钮 | 玻璃描边亮起 | border / background / color 180ms |
| 卡片 | 上浮 + 边框发光 | `translateY(-3px)`, border-glow, 背景高光 |
| 输入框 | focus 光环 | `box-shadow: 0 0 0 4px color-mix(...)` |
| AI 步骤 | spinner -> check morph | 生成中 pulse，完成后 checkmark fade/scale |
| 消息流 | slide-up fade-in | 180-260ms，避免大幅位移 |
| 主题切换 | cross-fade token transition | 200-300ms，避免闪屏 |

必须实现：

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
  }
}
```

## 4. 组件提示词

### 4.1 项目列表页

```text
把项目列表页设计成一个 AI resume gallery，而不是简单长条列表。顶部是紧凑品牌栏和筛选/搜索/主题切换；主区域是响应式网格，每个项目都是一张带 A4 简历缩略图的卡片，类似简历模板市场。卡片底部显示简历标题、更新时间、生成状态、更多操作。hover 时卡片轻微上浮、缩略图边框发光、右上角出现快捷操作。空状态直接展示一个“新建简历”卡片，而不是大段文案。
```

### 4.2 简历图库卡片

```text
重设计简历/项目卡片：不要使用横向长条，不要只有标题和日期。卡片采用 3:4 或 A4 比例缩略图，上方 80% 是真实简历预览，下方 20% 是元信息。缩略图需要模拟真实简历：标题栏、个人信息块、教育经历、项目经历、技能标签、头像占位等，按不同模板显示不同布局和主色。卡片支持选中、hover、loading、生成中、导出中、失败、更多菜单状态。

新增简历入口也必须是一张同尺寸卡片：中心是发光 `+` 图标，标题“新建简历”，副标题“上传文件或从零开始”。hover 时显示轻微网格背景、边框流光和 `+` 图标 pulse。点击后可以进入创建流程或打开创建弹窗。
```

卡片结构建议：

```text
┌────────────────────┐
│                    │
│   A4 resume preview│
│   thumbnail        │
│                    │
├────────────────────┤
│ 简历标题      ⋯    │
│ 更新时间 · 状态     │
└────────────────────┘
```

缩略图模板建议：

| 模板 | 视觉 | 适用 |
|------|------|------|
| `classic-blue` | 蓝色侧边栏、清晰分区、头像右上 | 技术岗、应届生、项目型简历 |
| `compact-black` | 黑白高密度、粗分隔条 | 传统企业、金融、咨询 |
| `modern-sidebar` | 左侧个人信息栏、右侧经历内容 | 设计、运营、产品 |
| `warm-editorial` | 暖驼标题、纸张质感、杂志排版 | 通用专业简历 |
| `minimal-apple` | 大留白、细线、系统字体 | 高级简洁风 |

### 4.3 登录页

```text
登录页也要纳入重构。左侧或背景可以使用柔和玻璃光、动态网格或简历缩略图阵列，右侧是紧凑登录卡片。表单控件与全局主题一致，错误提示不要突兀，使用 inline alert + 轻微 shake 或 glow。登录中按钮显示 spinner 和 loading 文案。移动端居中单栏。
```

### 4.4 Intake / 项目详情页

```text
把资料接入页设计成 AI intake pipeline。左侧是资料来源卡片：上传文件、Git 仓库、补充说明；右侧或下方是已接入资料列表和解析状态。每个资料项有文件类型图标、状态、摘要、操作菜单。解析过程显示步骤流和进度，完成后显示“生成简历”主操作。整体保持科技感，但信息密度要适合长期使用。
```

### 4.5 Workbench 编辑页

```text
把工作台设计成三栏专业 AI 编辑器。左侧是资料和处理步骤流，中间是 A4 简历白纸画布，右侧是 AI 对话面板。顶部 `ActionBar` 像 Apple Pro app 一样克制、紧凑、半透明。不要恢复底部 `FormatToolbar`；格式操作由选中文本时出现的 `BubbleToolbar` 承载，右键操作由 `ContextMenu` 承载。AI 面板需要显示生成步骤、消息、输入框和快捷按钮。整体有科技感，但 A4 简历内容区域必须保持纯净、专业、可读。
```

### 4.5.1 编辑器浮动交互层

```text
重设计 BubbleToolbar、ContextMenu、AlignSelector。BubbleToolbar 采用单行玻璃浮层，包含字体、字号、加粗、斜体、下划线、颜色、列表、对齐、行距。ContextMenu 采用紧凑右键菜单，包含撤销、重做、剪切、复制、粘贴、全选，支持禁用态和剪贴板错误提示。AlignSelector 是 dropdown，不要退回 4 个散开的对齐按钮。所有浮动层必须有清晰 z-index，不能被 A4 水印、面板折叠按钮或 modal 遮挡，也不能遮挡正在编辑的选区。
```

### 4.6 AI 生成状态

```text
生成状态参考图 2/3：居中或面板内显示一个轻量 AI 图标、动态粒子短线、标题“正在生成你的简历”、副标题“正在整理资料、生成结构和优化表达”。步骤列表包含“读取资料”“分析岗位方向”“生成简历 HTML”“优化排版”“准备预览”，每一步都有进行中 pulse、完成 check、失败 retry 状态。
```

### 4.7 Marketing 页面

```text
营销页不再是普通大留白 landing page。首页第一屏直接展示产品工作台视觉：左侧 AI 生成步骤，右侧简历预览缩略图，中间突出“AI 帮你生成可编辑简历”。features / pricing / help 保持内容清晰，但使用同一套 glass surface、主题色调、按钮动效和简历缩略图资产。导航、页脚、FAQ、定价卡都要支持多主题。
```

### 4.8 弹窗、浮层和基础组件

```text
统一重构 button、input、textarea、modal、popover、dropdown、alert、full-page-state 等基础组件。所有组件必须具备 light/dark/palette 适配、hover/focus/active/disabled/loading 状态，以及 reduced-motion 降级。弹窗使用玻璃面板 + 清晰边框 + 背景模糊；Popover 保持高可读，不使用过强阴影。
```

### 4.9 内容保护与预览水印

```text
保留 A4Canvas 内的 WatermarkOverlay、打印保护和 copy 纯文本保护。可以重设计水印透明度、颜色和与主题的适配，但不能移除水印、不能让装饰背景覆盖水印、不能让右键菜单绕过纯文本复制逻辑。导出 PDF 仍然是获得无水印简历的路径。
```

## 5. 重构范围

### 5.1 Workbench 前端

| 范围 | 文件/组件 |
|------|-----------|
| 页面 | `LoginPage.tsx`, `ProjectList.tsx`, `ProjectDetail.tsx`, `EditorPage.tsx` |
| 基础组件 | `button.tsx`, `input.tsx`, `textarea.tsx`, `modal.tsx`, `popover.tsx`, `dropdown.tsx`, `alert.tsx`, `full-page-state.tsx` |
| Intake | `ProjectCard.tsx`, `AssetList.tsx`, `AssetSidebar.tsx`, `ParsedItem.tsx`, `UploadDialog.tsx`, `GitRepoDialog.tsx`, `NoteDialog.tsx`, `DeleteConfirm.tsx` |
| Editor | `ActionBar.tsx`, `A4Canvas.tsx`, `BubbleToolbar.tsx`, `ContextMenu.tsx`, `AlignSelector.tsx`, `ToolbarButton.tsx`, `ToolbarSeparator.tsx`, `SaveIndicator.tsx`, `AiPanelPlaceholder.tsx`, `FontSelector.tsx`, `FontSizeSelector.tsx`, `ColorPicker.tsx`, `LineHeightSelector.tsx`, `WatermarkOverlay.tsx` |
| Chat | `ChatPanel.tsx`, `HtmlPreview.tsx` |
| 样式 | `index.css`, `styles/editor.css` |

### 5.2 Marketing 前端

| 范围 | 文件/组件 |
|------|-----------|
| 页面 | `index.astro`, `features.astro`, `pricing.astro`, `help.astro` |
| 布局 | `BaseLayout.astro` |
| 组件 | `Nav.astro`, `Hero.astro`, `FeatureSection.astro`, `PricingCard.astro`, `FaqGroup.astro`, `CTASection.astro`, `Footer.astro`, `Section.astro` |

### 5.3 实施优先级

| 优先级 | 内容 |
|--------|------|
| P0 | 主题 token、基础组件、项目/简历图库卡片、新建卡片入口 |
| P1 | Login、ProjectList、ProjectDetail、EditorPage 全页面重构 |
| P2 | BubbleToolbar、ContextMenu、AlignSelector、AI 生成步骤动效、ChatPanel、HtmlPreview、导出/版本状态 |
| P3 | Marketing 四页统一视觉、响应式细节、主题切换体验 |

## 6. 非目标

- 不要把简历 A4 画布做成黑色或霓虹风。
- 不要使用大面积纯紫/纯蓝渐变铺满页面，避免廉价 AI 工具感。
- 不要让动画影响输入、编辑、拖拽和导出效率。
- 不要堆叠太多卡片套卡片，面板层级要清晰。
- 不要只改营销页，工作台才是核心体验。
- 不要再使用简单横向长条作为简历/项目主卡片。
- 不要恢复或重建已删除的 `FormatToolbar`。
- 不要破坏 `WatermarkOverlay`、打印保护、copy/cut 纯文本保护和右键菜单剪贴板错误提示。
- 不要把多主题实现成一套与当前 Tailwind `@theme` 无关的新变量系统。

## 7. 验收标准

- `futuristic-apple`, `warm-editorial`, `quiet-luxury` 至少有明确 token 方案，其中默认主题完整实现 light / dark。
- 主题覆盖登录页、项目列表、项目详情、工作台、AI 面板、弹窗、按钮、输入框、卡片、空状态和错误状态。
- 主题实现沿用当前 `--color-*` token，现有 Tailwind utility class 不需要大面积重写。
- 项目/简历列表使用模板预览式网格卡片，不再使用简单长条卡片。
- 新建入口是同尺寸 `+` 简历卡片，不再是顶部简单输入框 + 创建按钮的唯一入口。
- `BubbleToolbar`、`ContextMenu`、`AlignSelector` 在新视觉下仍可用，且 `FormatToolbar` 不被恢复。
- `WatermarkOverlay` 可见、print protection 有效、copy/cut 仍只写入纯文本。
- 所有按钮、输入框、卡片、步骤项、消息项至少有 hover / focus / active / loading 其中两类状态。
- 首屏进入和 AI 生成流程具备可感知的动效，不出现突兀闪烁。
- A4 简历区域在两种主题下都保持白纸、清晰边界和高可读性。
- 移动端不出现文字溢出、按钮挤压、面板互相遮挡。

## 8. 本分支落地记录

**分支**: `feat/frontend-futuristic-apple-ui`

### 8.1 已实现范围

- Workbench 全局主题 token：基于当前 Tailwind v4 `@theme` / `--color-*` 命名扩展，不另起孤立变量系统。
- 主题切换：新增跟随系统、科技夜、银白、暖驼、静奢五个选项；跟随系统不锁定具体主题，会根据 OS 明暗自动解析，手动选择具体主题后持久化到 `localStorage`。
- 登录页：改为玻璃登录卡片 + 左侧 AI 简历工作台氛围。
- 项目列表页：移除顶部简单创建输入框主入口，改为简历图库网格和同尺寸 `+` 新建简历卡片。
- 项目详情页：改为 AI intake pipeline，上传、Git、备注入口以三张操作卡呈现。
- 编辑页：保留三栏结构，升级 ActionBar、A4Canvas、BubbleToolbar、ContextMenu、ToolbarButton、AI ChatPanel 的视觉。
- 内容保护：保留 `WatermarkOverlay`、打印保护、copy/cut 纯文本保护，不恢复 `FormatToolbar`。
- Marketing 首页：改为产品工作台视觉 Hero，导航和全局背景同步科技玻璃风。

### 8.2 验证

- `frontend/workbench`: `npm run build` 通过。
- `frontend/marketing`: `npm run build` 通过。
- `frontend/workbench`: 新增主题组件单独 lint 通过：`npx eslint src/components/ui/theme-switcher.tsx src/lib/theme.ts`。

### 8.3 已知情况

- `frontend/workbench npm run lint` 仍会被 dev 既有 lint 问题阻塞，包括旧测试文件未使用变量、`useAutoSave` hook 规则、`ChatPanel` / `ContextMenu` / `NoteDialog` / `AssetEditorDialog` 的 `set-state-in-effect` 规则等。本次只修复新增主题组件相关 lint。
- `frontend/marketing` 依赖按仓库已有 `bun.lock` 使用 `bun install` 安装；未保留 `npm install` 超时产生的 `package-lock.json`。
