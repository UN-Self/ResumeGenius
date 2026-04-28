# Workbench TipTap 编辑器 — 设计文档

> 日期：2026-04-28
> 状态：已批准
> 模块：workbench
> 参考规范：`docs/01-product/ui-design-system.md`

## 范围

### 包含

- TipTap 基础编辑：粗体、斜体、下划线、标题（H1-H3）、有序列表、无序列表、文本对齐
- A4 画布预览（210mm × 297mm CSS，transform: scale() 缩放）
- 自动保存（debounce 2s，失败重试 3 次，卸载时 flush）
- 底部固定格式工具栏
- 顶部操作栏（项目名称、保存状态）
- AI 面板空占位（360px 右侧）
- 后端 DraftService（GET/PUT draft API）
- MSW Mock（前端独立开发/测试）
- TDD 测试（后端 4 个 + 前端 6 个）

### 不包含

- 高级扩展（表格/图片/链接）
- 拖拽布局（section 重排）
- 手动保存按钮
- 自定义主题
- 版本快照（create_version 参数）
- AI 对话功能
- 移动端适配（v1 仅桌面端）

## 设计原则

参考 Google NotebookLM 的界面风格：干净、克制、内容优先。

- 界面服务于内容，不喧宾夺主
- 信息层级清晰，功能区域划分明确
- 交互直觉化，减少学习成本
- 禁止装饰性动效（粒子、弹跳等）

## 设计令牌（Design Tokens）

> 来源：`docs/01-product/ui-design-system.md` 第 3-5 节

### 色彩

| 角色 | 色值 | 用途 |
|---|---|---|
| 主色 | `#1a73e8` | 按钮、链接、选中态、工具栏按钮激活态 |
| 主色悬浮 | `#1557b0` | hover 态 |
| 主色背景 | `#e8f0fe` | 选中行、标签背景、工具栏按钮激活背景 |
| 页面背景 | `#f8f9fa` | 工作台主背景 |
| 卡片/面板背景 | `#ffffff` | A4 画布、工具栏、顶部栏 |
| 分割线 | `#dadce0` | 工具栏分组分隔、区域边界 |
| 主文字 | `#202124` | 正文、标题、工具栏按钮 |
| 次文字 | `#5f6368` | 描述、辅助文字、保存状态时间 |
| 禁用文字 | `#9aa0a6` | placeholder、禁用态按钮 |
| 成功文字/背景 | `#0d652d` / `#e6f4ea` | 保存成功状态 |
| 错误文字/背景 | `#c5221f` / `#fce8e6` | 保存失败状态 |
| 警告文字/背景 | `#b06000` / `#fef7e0` | 网络异常提示 |

**使用规则**：
- 同一页面主色按钮不超过 1 个
- 语义色只用于状态反馈，不用于装饰
- 禁止出现主色调之外的彩色装饰

### 字体

```css
font-family: "Inter", "Noto Sans SC", -apple-system, BlinkMacSystemFont, sans-serif;
```

| 用途 | 字号 | 字重 | 行高 |
|---|---|---|---|
| 页面标题（项目名） | 24px | 600 | 1.3 |
| 区域标题 | 20px | 600 | 1.3 |
| 卡片标题 | 16px | 500 | 1.4 |
| 正文 | 14px | 400 | 1.5 |
| 辅助文字（保存状态） | 12px | 400 | 1.5 |
| 按钮文字 | 14px | 500 | — |

**规则**：标题不加斜体；正文不加粗（加粗仅用于强调标记）。

### 间距

基准单位：4px。所有间距必须是 4 的倍数。

| 用途 | 值 |
|---|---|
| 工具栏按钮内间距 | 8px |
| 工具栏按钮间间距 | 4px |
| 组件间间距 | 12px |
| 工具栏分组间间距 | 12px |
| 区域间间距 | 16px |
| 顶部栏内边距（水平） | 16px |
| 顶部栏内边距（垂直） | 12px |
| A4 画布外边距 | 24px |
| 页面边距 | 24px |

### 圆角

| 用途 | 值 |
|---|---|
| 面板（工具栏/顶部栏） | 8px |
| 按钮 | 6px |
| 标签 (Tag/Badge) | 4px |
| 弹窗/Modal | 12px |

### 阴影

| 用途 | 值 |
|---|---|
| A4 画布 | `0 1px 3px rgba(0,0,0,0.08)` |
| A4 画布悬浮态 | `0 2px 8px rgba(0,0,0,0.12)` |
| 顶部栏/工具栏 | 无（用分割线代替） |

### 动效

| 交互 | 时长 | 缓动 |
|---|---|---|
| 工具栏按钮 hover/active | 150ms | ease-in-out |
| 保存状态切换 | 200ms | ease-in-out |
| AI 面板展开/折叠 | 200ms | ease-in-out |
| 页面切换 | 无（v1 直接切换） |
| 加载动画 | 持续旋转 | linear |

**规则**：禁止装饰性动效；所有动效需尊重 `prefers-reduced-motion`。

## 架构决策

### 方案：TipTap EditorContent + 保留原始 CSS

TipTap 直接加载完整 HTML（含 `<style>` 标签和自定义 CSS class），用户通过工具栏对选中文字应用基础格式。ProseMirror 自动将不认识的标签/属性作为原子节点保留，实现"纯文本编辑 + 结构保留"。

保存时直接取 editor 的 `getHTML()` 输出，PUT 完整 HTML 到后端。

**选择理由**：在"保留原始 HTML"和"利用 TipTap 能力"之间取得最佳平衡，无需写转换层。

## 页面布局

### ASCII 线框图

```
┌──────────────────────────────────────────────────────────┐
│  ActionBar（顶部栏）                                       │
│  [Logo]  项目名称  ···  [保存状态]  [版本] [导出]           │
│  高: auto, 背景白, 底部分割线 #dadce0                      │
├──────────────────────────────────────┬───────────────────┤
│                                      │                   │
│         A4 Canvas                    │  AI Panel         │
│    (210mm × 297mm, 居中)             │  (360px, 空占位)   │
│    transform: scale() 缩放           │  背景白, 左分割线   │
│    overflow: auto 滚动               │                   │
│    背景灰 #f8f9fa                     │  "AI 助手"         │
│                                      │  "即将推出"        │
│                                      │                   │
├──────────────────────────────────────┴───────────────────┤
│  FormatToolbar（底部固定工具栏）                            │
│  [B] [I] [U] │ [H1][H2][H3] | [UL][OL] | [←][→][≡]     │
│  高: auto, 背景白, 顶部分割线 #dadce0                      │
└──────────────────────────────────────────────────────────┘
```

### CSS Grid 定义

```css
.workbench-layout {
  display: grid;
  grid-template-rows: auto 1fr auto;
  grid-template-columns: 1fr 360px;
  grid-template-areas:
    "action-bar  action-bar"
    "canvas      ai-panel"
    "toolbar     toolbar";
  height: 100vh;
  background: #f8f9fa;
}

/* AI 面板收起时 */
.workbench-layout.ai-panel-collapsed {
  grid-template-columns: 1fr 0px;
}
```

### 布局规则

- AI 面板可收起（200ms ease-in-out 过渡），收起后 A4 画布占满宽度
- A4 画布需缩放适应，画布本身固定 210mm × 297mm
- 底部工具栏 `position: sticky; bottom: 0`，始终可见
- 滚动区域仅限 A4 Canvas 区域，工具栏和顶部栏固定

## 组件详细规范

### 6.1 ActionBar（顶部操作栏）

**位置**：页面顶部，`grid-area: action-bar`

**布局**：Flex 水平排列，左中右三段式

```
[Logo] ────── [项目名称] ────── [保存状态] [版本历史按钮] [导出按钮]
 左段              中段                    右段（按钮组）
```

| 属性 | 值 |
|---|---|
| 高度 | 56px（12px 上下内边距 + 32px 内容） |
| 背景 | `#ffffff` |
| 底部边框 | `1px solid #dadce0` |
| 水平内边距 | 16px |
| 段间距 | 16px |

**子组件**：

- **Logo**：SVG 图标，24×24px，使用 Lucide `FileText` 图标
- **项目名称**：字号 16px，字重 500，颜色 `#202124`，左对齐
- **保存状态 SaveIndicator**：字号 12px，颜色 `#5f6368`，显示在项目名称右侧或独立区域
- **版本历史按钮**：Ghost 按钮（无背景无边框 + 主色文字），14px/500
- **导出按钮**：Secondary 按钮（白色背景 + 主色边框），14px/500（v1 此冲刺为禁用态）

**交互状态**（按钮）：

| 状态 | Primary | Ghost | Disabled |
|---|---|---|---|
| 默认 | 主色背景 + 白色文字 | 无背景 + 主色文字 | 背景 `#f8f9fa` + `#9aa0a6` 文字 |
| 悬浮 | `#1557b0` 背景 | `#e8f0fe` 背景 | — |
| 按下 | `scale(0.97)` + 150ms | `scale(0.97)` + 150ms | — |
| 焦点 | `2px solid #1a73e8` 外圈 | `2px solid #1a73e8` 外圈 | — |

**按钮通用规则**：
- 最小点击区域 44×44px（触摸友好）
- 所有可点击元素加 `cursor-pointer`
- 圆角 6px，高度 36px

### 6.2 A4Canvas（A4 画布）

**位置**：页面中央，`grid-area: canvas`，外层 `overflow: auto`

**画布尺寸**：

| 属性 | 值 |
|---|---|
| 宽度 | 210mm |
| 最小高度 | 297mm |
| 内边距 | 18mm（上下）× 20mm（左右） |
| 背景 | `#ffffff` |
| 阴影 | `0 1px 3px rgba(0,0,0,0.08)` |
| 外边距 | 24px auto（水平居中） |

**缩放机制**：

```css
.canvas-wrapper {
  /* 根据可视区域计算缩放比例 */
  transform: scale(var(--canvas-zoom, 0.75));
  transform-origin: top center;
  /* 缩放后保留原始尺寸占位，防止滚动区域计算错误 */
  width: 210mm;
  min-height: 297mm;
}
```

缩放比例由 JavaScript 计算：`(可用宽度 - 48px padding) / 210mm`，clamp 在 0.5-1.0 之间。

**滚动行为**：外层容器 `overflow: auto`，支持纵向和横向滚动（当缩放后内容仍超出时）。

### 6.3 TipTapEditor（编辑器实例）

**位置**：渲染在 A4Canvas 内部

**配置**：

```typescript
const editor = useEditor({
  extensions: [
    StarterKit,              // Bold, Italic, Heading, BulletList, OrderedList, History...
    Underline,
    TextAlign.configure({
      types: ['heading', 'paragraph'],
    }),
  ],
  content: initialHtml,      // 直接加载后端返回的完整 HTML（含 <style>）
  editorProps: {
    attributes: {
      class: 'resume-content outline-none',  // 移除默认聚焦边框
      style: 'min-height: 261mm;',  // 297mm - 18mm*2 上下内边距
    },
  },
})
```

**样式覆盖**（TipTap 默认样式需匹配设计系统）：

```css
/* TipTap 内容区域样式覆盖 */
.ProseMirror {
  font-family: "Inter", "Noto Sans SC", -apple-system, BlinkMacSystemFont, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  color: #202124;
  outline: none;
}

.ProseMirror h1 { font-size: 24px; font-weight: 600; line-height: 1.3; }
.ProseMirror h2 { font-size: 20px; font-weight: 600; line-height: 1.3; }
.ProseMirror h3 { font-size: 16px; font-weight: 500; line-height: 1.4; }
.ProseMirror p { font-size: 14px; font-weight: 400; line-height: 1.5; }
.ProseMirror ul, .ProseMirror ol { padding-left: 24px; }
.ProseMirror li { margin-bottom: 4px; }
```

### 6.4 FormatToolbar（底部格式工具栏）

**位置**：页面底部，`grid-area: toolbar`，`position: sticky; bottom: 0`

| 属性 | 值 |
|---|---|
| 背景 | `#ffffff` |
| 顶部边框 | `1px solid #dadce0` |
| 水平内边距 | 16px |
| 垂直内边距 | 8px |
| 对齐 | 居中 |

**工具栏按钮分组**：

```
[B] [I] [U] │ [H1] [H2] [H3] │ [UL] [OL] │ [←] [→] [≡]
 粗体组        标题组             列表组        对齐组
```

**按钮规范**：

| 属性 | 值 |
|---|---|
| 尺寸 | 32×32px 图标区域，44×44px 点击区域（padding 扩展） |
| 圆角 | 6px |
| 图标 | Lucide 图标，20×20px，stroke-width 2 |
| 间距 | 按钮间 4px，分组间 12px |
| 分隔符 | `1px solid #dadce0`，高度 20px，垂直居中 |

**工具栏按钮交互状态**：

| 状态 | 非激活 | 激活（格式已应用） |
|---|---|---|
| 默认 | 透明背景 + `#5f6368` 图标 | `#e8f0fe` 背景 + `#1a73e8` 图标 |
| 悬浮 | `#f8f9fa` 背景 | `#d2e3fc` 背景 |
| 按下 | `scale(0.95)` + 150ms | `scale(0.95)` + 150ms |
| 焦点 | `2px solid #1a73e8` 外圈 | `2px solid #1a73e8` 外圈 |

**图标映射**：

| 按钮 | Lucide 图标 | TipTap 命令 |
|---|---|---|
| Bold (B) | `Bold` | `editor.chain().focus().toggleBold().run()` |
| Italic (I) | `Italic` | `editor.chain().focus().toggleItalic().run()` |
| Underline (U) | `Underline` | `editor.chain().focus().toggleUnderline().run()` |
| H1 | `Heading1` | `editor.chain().focus().toggleHeading({level: 1}).run()` |
| H2 | `Heading2` | `editor.chain().focus().toggleHeading({level: 2}).run()` |
| H3 | `Heading3` | `editor.chain().focus().toggleHeading({level: 3}).run()` |
| UL | `List` | `editor.chain().focus().toggleBulletList().run()` |
| OL | `ListOrdered` | `editor.chain().focus().toggleOrderedList().run()` |
| Align Left | `AlignLeft` | `editor.chain().focus().setTextAlign('left').run()` |
| Align Center | `AlignCenter` | `editor.chain().focus().setTextAlign('center').run()` |
| Align Justify | `AlignJustify` | `editor.chain().focus().setTextAlign('justify').run()` |

**键盘快捷键**（TipTap StarterKit 默认支持）：

| 快捷键 | 操作 |
|---|---|
| Ctrl+B | 粗体 |
| Ctrl+I | 斜体 |
| Ctrl+U | 下划线 |
| Ctrl+Z | 撤销 |
| Ctrl+Shift+Z | 重做 |
| Ctrl+Shift+7 | 有序列表 |
| Ctrl+Shift+8 | 无序列表 |

### 6.5 SaveIndicator（保存状态指示）

**位置**：ActionBar 右侧，项目名称右边

**状态流转**：

```
idle → saving → saved → idle
                → error → retry → saving → ...
```

| 状态 | 显示 | 颜色 | 图标 |
|---|---|---|---|
| idle | 无显示（透明占位） | — | — |
| saving | 旋转加载图标 + "保存中..." | `#5f6368` | Lucide `Loader2`（旋转动画） |
| saved | 对勾 + "已保存 HH:MM" | `#0d652d` | Lucide `Check`，5 秒后淡出回 idle |
| error | 警告 + "保存失败，点击重试" | `#c5221f` | Lucide `AlertCircle`，可点击触发重试 |

**动画**：
- 旋转加载：`animation: spin 1s linear infinite`
- saved → idle 淡出：`opacity 1→0`，200ms ease-in-out
- 状态切换：200ms ease-in-out

**可访问性**：使用 `aria-live="polite"` 屏幕阅读器播报保存状态变化。

### 6.6 AiPanelPlaceholder（AI 面板空占位）

**位置**：`grid-area: ai-panel`，右侧 360px

| 属性 | 值 |
|---|---|
| 宽度 | 360px |
| 背景 | `#ffffff` |
| 左边框 | `1px solid #dadce0` |
| 内容 | 居中显示占位信息 |

**内容布局**：

```
┌─────────────────────┐
│                     │
│    [AI 图标]        │
│    AI 助手          │
│    即将推出          │
│                     │
│    用 AI 智能优化    │
│    你的简历内容       │
│                     │
└─────────────────────┘
```

- AI 图标：Lucide `Sparkles`，48×48px，颜色 `#9aa0a6`
- "AI 助手"：20px/600，`#202124`
- "即将推出"：14px/400，`#5f6368`
- 描述文字：12px/400，`#9aa0a6`
- 内容垂直水平居中

### 6.7 加载状态

**页面初始加载**（GET draft 请求中）：

| 区域 | 行为 |
|---|---|
| A4 Canvas | 显示骨架屏（白色矩形 + 3 行灰色条带模拟文字，`animate-pulse`） |
| 工具栏 | 正常渲染，但所有按钮处于禁用态（`#9aa0a6` + `pointer-events: none`） |
| 保存状态 | 显示 idle（无内容） |

**加载反馈规范**（UX 规则）：操作超过 300ms 时必须显示反馈。

### 6.8 空状态

**无 draft 数据时**：

- A4 Canvas 区域显示空状态提示
- 图标：Lucide `FileEdit`，64×64px，`#9aa0a6`
- 主文字："暂无简历内容"，16px/500，`#5f6368`
- 副文字："开始编辑你的简历，或使用 AI 助手生成初稿"，12px/400，`#9aa0a6`

### 6.9 错误状态

**加载失败**（GET draft 返回错误）：

| 元素 | 显示 |
|---|---|
| 图标 | Lucide `AlertTriangle`，48×48px，`#c5221f` |
| 主文字 | "加载失败"，16px/500，`#c5221f` |
| 副文字 | 错误信息文本，12px/400，`#5f6368` |
| 按钮 | Secondary 按钮"重新加载"，点击重试 GET 请求 |

**自动保存失败**（PUT 返回错误）：
- 见 6.5 SaveIndicator 的 error 状态
- 提供"点击重试"的恢复路径

## 组件结构

```
src/
├── pages/
│   └── EditorPage.tsx            # 页面入口，组合所有组件
├── components/
│   └── editor/
│       ├── WorkbenchLayout.tsx   # 整体布局（CSS Grid）
│       ├── ActionBar.tsx         # 顶部操作栏
│       ├── A4Canvas.tsx          # A4 画布容器（缩放 + 滚动）
│       ├── TipTapEditor.tsx      # TipTap 编辑器实例
│       ├── FormatToolbar.tsx     # 底部格式工具栏
│       ├── ToolbarButton.tsx     # 工具栏按钮（复用组件）
│       ├── ToolbarSeparator.tsx  # 工具栏分组分隔符
│       ├── SaveIndicator.tsx     # 保存状态指示
│       ├── AiPanelPlaceholder.tsx # AI 面板占位
│       ├── EditorSkeleton.tsx    # 加载骨架屏
│       └── EditorEmptyState.tsx  # 空状态
├── hooks/
│   └── useAutoSave.ts           # 自动保存 hook
├── mocks/
│   ├── handlers/
│   │   └── drafts.ts            # MSW: GET/PUT draft API mock
│   ├── browser.ts               # MSW browser worker setup
│   └── fixtures.ts              # mock 数据
└── styles/
    └── editor.css               # TipTap 编辑器样式覆盖 + A4 画布样式
```

## 后端 API

### GET /api/v1/drafts/:draft_id

查询 draft，返回：

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "project_id": 1,
    "html_content": "<div class=\"resume\">...</div>",
    "updated_at": "2026-04-28T12:00:00Z"
  },
  "message": "ok"
}
```

draft 不存在：`{code: 4001, message: "draft not found"}`, HTTP 404。

### PUT /api/v1/drafts/:draft_id

请求体：`{html_content: string}`

更新 `html_content` 和 `updated_at`，返回：

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "updated_at": "2026-04-28T12:00:01Z"
  },
  "message": "ok"
}
```

`html_content` 为空：`{code: 4002, message: "html content empty"}`, HTTP 400。

### 实现结构

```
backend/internal/modules/workbench/
├── routes.go        # 路由注册（已有，需更新）
├── handler.go       # Handler: GetDraft, UpdateDraft
├── service.go       # DraftService: GetByID, Update
└── handler_test.go  # 测试（PostgreSQL + 事务回滚隔离）
```

## 自动保存

### useAutoSave Hook

```
状态: idle → saving → saved → idle
                     → error → retry → saving → ...
```

- **debounce 2s**：内容停止变化 2 秒后触发 PUT
- **保存中**：显示"保存中..."，带旋转图标（Lucide `Loader2`，`spin 1s linear infinite`）
- **保存成功**：显示"已保存 HH:MM"（Lucide `Check`，`#0d652d`），5 秒后 200ms 淡出回 idle
- **保存失败**：重试 3 次（间隔 1s/2s/4s 指数退避），全部失败后显示"保存失败，点击重试"（`#c5221f`），提供手动重试按钮
- **卸载时**：有未保存变更立即 flush（`beforeunload` 事件 + React `useEffect` cleanup）
- **防重复**：保存中不重复触发请求
- **可访问性**：`aria-live="polite"` 播报状态变化

## MSW Mock

- 安装 `msw`，在 `src/mocks/` 下创建 handler
- GET `/api/v1/drafts/:draftId` 返回 `sample_draft.html` 内容（模拟 200ms 延迟）
- PUT `/api/v1/drafts/:draftId` 返回成功（模拟 100ms 延迟）
- 通过 `VITE_USE_MOCK=true/false` 环境变量控制启用/禁用
- `main.tsx` 中条件启动 MSW worker

## 可访问性要求

- 所有工具栏按钮有 `aria-label`（如 `aria-label="粗体 (Ctrl+B)"`）
- 工具栏分组用 `role="group"` + `aria-label`
- 保存状态区域 `aria-live="polite"`
- 键盘导航：Tab 顺序与视觉顺序一致
- 焦点状态：所有可交互元素有 `2px solid #1a73e8` 焦点环
- 尊重 `prefers-reduced-motion`：禁用或减少动画
- 不依赖颜色传达信息（图标 + 文字双重指示）
- 最小点击区域 44×44px

## 测试计划

### 后端（Go, handler_test.go）

| # | 测试 | 说明 |
|---|---|---|
| 1 | GetDraft 成功 | 插入 draft → GET → 验证返回 html_content |
| 2 | GetDraft 不存在 | GET 不存在的 ID → 404 + code 4001 |
| 3 | UpdateDraft 成功 | PUT 新 html_content → 验证 updated_at 更新 |
| 4 | UpdateDraft 空内容 | PUT 空字符串 → 400 + code 4002 |

连接 docker-compose 提供的 PostgreSQL 实例（`DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=resume_genius`），每个测试用独立的事务回滚保证隔离。运行测试前需确保 `docker compose up -d postgres` 已启动。TDD 红绿重构。

### 前端（Vitest + Testing Library）

| # | 测试 | 文件 | 说明 |
|---|---|---|---|
| 1 | TipTap 渲染 HTML | TipTapEditor.test.tsx | 加载 sample_draft.html → 验证包含关键文本 |
| 2 | 工具栏格式化 | FormatToolbar.test.tsx | 选中文字 → 点击 Bold → 验证 `<strong>` |
| 3 | 自动保存 debounce | useAutoSave.test.tsx | 编辑 → 2s 内无 PUT → 2s 后触发 PUT |
| 4 | 保存失败重试 | useAutoSave.test.tsx | 模拟 PUT 失败 → 验证重试 3 次 |
| 5 | MSW mock 拦截 | drafts.test.ts | 验证 mock handler 返回正确数据 |
| 6 | A4 Canvas 样式 | A4Canvas.test.tsx | 验证画布尺寸 210mm × 297mm |

## 新增依赖

### 前端

- `@tiptap/react` — React 绑定
- `@tiptap/starter-kit` — 基础扩展包
- `@tiptap/extension-underline` — 下划线
- `@tiptap/extension-text-align` — 文本对齐
- `msw` — Mock Service Worker
- `@testing-library/react` — 组件测试
- `@testing-library/jest-dom` — DOM 断言
