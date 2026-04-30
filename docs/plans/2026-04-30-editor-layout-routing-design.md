# 编辑器布局重构与路由修复设计

## 问题

1. **路由 bug**：项目详情页点击"下一步：开始解析"后没有修改路由（`/projects/:id`），刷新后回到项目详情页，丢失编辑状态
2. **布局缺失**：编辑器页面（EditorPage）缺少完整的三段式布局，无响应式适配

## 设计决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 路由策略 | 跳转到独立路由 `/edit` | 职责分离清晰，URL 语义正确 |
| 面板响应式 | 三栏可折叠（Figma 风格） | 窄屏时保留面板访问能力 |
| 画布适配 | A4 保持实际尺寸居中 + scale 缩放 | 复用现有 computeZoom 逻辑 |
| 面板内容 | 与 ProjectDetail editing 阶段一致 | 最小化额外开发 |

## 1. 路由与页面职责

### 改造后路由

| 路由 | 组件 | 职责 |
|------|------|------|
| `/projects/:projectId` | ProjectDetail | 仅 intake 阶段（上传文件、补充信息、点击"开始解析"） |
| `/projects/:projectId/edit` | EditorPage | 三栏编辑器（素材侧边栏 + A4 画布 + AI 对话面板） |

### 草稿创建流程

草稿在 ProjectDetail 的 `handleParse` 中创建（跳转前）：

```
用户点击"开始解析"
    ↓
1. parsingApi.parseProject(pid)    → 解析文件，返回素材
2. 检查 proj.current_draft_id
   ├─ 有值 → 使用已有 draft
   └─ 为 null → workbenchApi.createDraft(pid) → 后端原子创建空白草稿
    ↓
3. navigate('/projects/:projectId/edit')
```

### 路由守卫

EditorPage 加载时：
- `GET /api/v1/projects/:id` 检查 `current_draft_id`
- 为 null → 重定向到 `/projects/:projectId`
- 有值 → 用 draft ID 加载编辑器内容

## 2. 三栏可折叠布局

### 整体结构

```
┌──────────────────────────────────────────────────┐
│  ActionBar（项目名 + 操作按钮）                      │
├────────┬─────────────────────────┬────────────────┤
│ 左面板  │      A4 画布区域         │   右面板        │
│ 素材    │   （A4 纸居中 + 缩放）   │   AI 对话       │
│ 280px  │       flex: 1           │   320px        │
│ 可折叠  │                         │   可折叠        │
├────────┴─────────────────────────┴────────────────┤
│  FormatToolbar（格式化工具栏）                        │
└──────────────────────────────────────────────────┘
```

### 折叠机制

- 每个面板头部有折叠/展开按钮
- 折叠时：`grid-template-columns: 0 1fr 320px` 或 `280px 1fr 0` 或 `0 1fr 0`
- 过渡动画：`transition: grid-template-columns 300ms ease-in-out`
- 折叠状态存在组件 state 中，不持久化

### 响应式断点

| 屏幕宽度 | 默认状态 |
|----------|----------|
| ≥ 1440px | 三栏全部展开 |
| 1280px ~ 1440px | 左面板展开、右面板折叠 |
| < 1280px | 左右面板都折叠，只显示画布 |
| < 768px | 不考虑（桌面端产品） |

### 画布区域

- 中间区域 `flex: 1`，铺满剩余空间
- A4 画布保持 210mm × 297mm 实际尺寸
- 画布在容器中水平垂直居中
- 通过 CSS `transform: scale()` 缩放适配容器宽度（复用现有 computeZoom）

## 3. 组件迁移与文件变更

### 改动文件

| 文件 | 类型 | 说明 |
|------|------|------|
| `frontend/workbench/src/pages/ProjectDetail.tsx` | 修改 | 删除 editing 阶段代码，handleParse 末尾改为 navigate |
| `frontend/workbench/src/pages/EditorPage.tsx` | 重写 | 三栏 Grid 布局、面板折叠、路由守卫、加载 draft |
| `frontend/workbench/src/styles/editor.css` | 修改 | 新增折叠态 grid 规则，移除 phase-* 类 |
| `frontend/workbench/src/components/editor/WorkbenchLayout.tsx` | 评估 | 根据是否仍在使用决定保留或移除 |

### 不涉及的文件

- A4Canvas、TipTapEditor、FormatToolbar、SaveIndicator 等组件直接复用
- 后端 API 无需改动

### 数据流

```
EditorPage 加载
  ↓
GET /api/v1/projects/:id        → current_draft_id（路由守卫）
  ↓
GET /api/v1/drafts/:id          → draft HTML 内容
  ↓
GET /api/v1/parsing/...         → 解析素材（左面板）
  ↓
渲染：左面板(ParsedSidebar) + 中间(A4Canvas + TipTap) + 右面板(AI 占位)
```
