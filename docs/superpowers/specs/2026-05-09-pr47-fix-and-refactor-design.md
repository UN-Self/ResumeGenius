# PR #47 修复与重构方案

更新时间：2026-05-09

## 1. 背景

PR #47 (feat/front_and_workbench) 引入了文件层级系统、设计知识库、AI 对话面板 UI 等功能。代码审查发现若干 bug 和架构问题，需要分两阶段处理：就地修复 bug，架构重构独立进行。

## 2. 就地修复（PR #47 内）

以下 5 项改动直接在 PR #47 分支上完成：

### 2.1 MoveAsset 错误消息泄露

**文件**：`backend/internal/modules/intake/handler.go:401`

**问题**：`MoveAsset` handler 默认分支使用 `err.Error()` 返回错误消息，可能泄露内部 SQL 查询、文件路径等信息。

**修复**：将 `err.Error()` 替换为固定字符串 `"failed to move asset"`，与其他 handler 保持一致。

### 2.2 错误码重复

**文件**：`backend/internal/modules/intake/handler.go:15-17`

**问题**：`CodeParamInvalid` 和 `CodeUnsupportedFormat` 都使用值 `1001`，客户端无法区分两种错误。

**修复**：将 `CodeUnsupportedFormat` 从 `1001` 改为 `1005`。

### 2.3 死代码

**文件**：`backend/internal/modules/agent/service.go:449`

**问题**：连续两行 `return err`，第二行永远不会执行。

**修复**：删除第 449 行。

### 2.4 无用变量

**文件**：`backend/cmd/server/main.go:26`

**问题**：`var _ *gorm.DB` 无实际用途，`database.Connect()` 返回值已在第 107 行使用。

**修复**：删除该行。

### 2.5 缩进混用

**文件**：`frontend/workbench/src/components/intake/AssetList.tsx:499-503`

**问题**：`renderAsset` 函数中混用 tab 和 space 缩进。

**修复**：统一为 space 缩进。

## 3. 重构任务（独立 Issue）

### 3.1 文件夹层级存储重构

**当前问题**：文件夹父子关系存储在 Asset 的 `metadata` JSONB 字段中（`folder_id` 键）。每次查子文件、计算深度、级联删除都需要加载项目全部 Asset 到内存。

**方案**：

- 在 `assets` 表新增 `parent_id` 列（`uint, nullable, index`），指向父文件夹的 Asset ID
- 新增 `path` 虚拟字段（如 "项目资料/简历素材/个人照片.jpg"），在查询时计算或在保存时冗余存储
- 迁移逻辑：从现有 `metadata.folder_id` 迁移数据到 `parent_id` 列
- 前端：从 `metadata.folder_id` 改为读 `parent_id` 字段
- 后端：`folderDepth()`、`collectFolderDescendantsForDelete()` 改为 SQL 查询而非内存 BFS

**影响范围**：
- `backend/internal/shared/models/models.go` — Asset 结构体加列
- `backend/internal/modules/intake/service.go` — 所有文件夹操作方法
- `backend/internal/modules/intake/handler.go` — 无变化（接口不变）
- `frontend/workbench/src/components/intake/AssetList.tsx` — 读 parent_id
- `frontend/workbench/src/components/intake/AssetSidebar.tsx` — 深度计算
- `frontend/workbench/src/lib/api-client.ts` — 无变化

### 3.2 设计知识库重构

**当前问题**：
- `designskill` 模块完全独立于现有 Skill 系统，未复用 `SkillLoader`
- 直接搬入 ui-ux-pro-max 开源库的 16 个 CSV，大部分数据与简历无关（landing page、dashboard、gaming 等）
- 自己实现 200 行 BM25 搜索算法，零测试
- 新增了独立的 `search_design_skill` 工具，未遵循 Issue #45 三层渐进式披露设计

**方案**：

#### Phase A：纳入三层 Skill 系统（依赖 Issue #45）

将 designskill 重构为 `resume-design` skill，遵循 Issue #45 的三层架构：

- **Layer 1**（System Prompt）：`resume-design: 当用户需要简历设计参考或模板时使用`
- **Layer 2**（`load_skill("resume-design")`）：描述文档，暴露两个二级工具：
  - `get_skill_reference` — 获取完整简历模板 HTML/CSS
  - `search_design_components` — 搜索组件素材（配色、字体、风格参考）
- **Layer 3**（`get_skill_reference("minimal-single-column")`）：具体模板内容

数据保留简历相关的 4 个域：style、color、typography、ux。去掉 landing、product、chart。

搜索引擎（BM25 或简化为关键词匹配）作为 `search_design_components` 的后端。

#### Phase B：用户上传模板 + 组件提取

用户可上传简历 HTML 模板：

1. **CSS 解析提取**：从模板 HTML 中解析出颜色值、字体声明、间距值等硬数据
2. **AI 描述提取**：调用 AI 分析模板，输出风格描述、适用行业、适用场景等软数据
3. **数据库存储**：提取结果存入数据库，按维度分表：
   - `template_color_schemes` — 配色方案（参考 colors.csv 列结构）
   - `template_typography` — 字体搭配（参考 typography.csv 列结构）
   - `template_styles` — 风格标签（参考 styles.csv 列结构）
   - `resume_templates` — 完整模板 HTML/CSS
4. **AI 按需查询**：通过 `search_design_components` 工具按维度搜索

**依赖**：Issue #45（三层 Skill 系统实现）必须先完成。

**影响范围**：
- 删除 `backend/internal/modules/designskill/` 模块
- 新增 `backend/internal/modules/agent/skills/resume-design/` 目录
- 修改 `backend/internal/modules/agent/tool_executor.go` — 替换 `search_design_skill` 为 `search_design_components`
- 修改 `backend/internal/modules/agent/service.go` — system prompt 更新
- 新增数据库表：`resume_templates`、`template_color_schemes`、`template_typography`、`template_styles`
- 新增模板上传 API 端点
- 新增 CSS 解析 + AI 提取逻辑

### 3.3 前端测试修复

PR #47 引入的 7 个前端测试失败：

1. **AssetSidebar** ×2 — PR 新增"上传文件夹"按钮，导致 `getByRole("button", /上传文件/i)` 匹配到多个元素。需改用更精确的查询或给按钮加 `data-testid`。
2. **UploadDialog** ×1 — 同名文件替换确认行为变更，测试断言需同步。
3. **FontSizeSelector / LineHeightSelector / AlignSelector** ×3 — CSS 类名从 `bg-primary-50` 改为 `bg-surface-hover`，测试断言需同步。
4. **EditorPage.autosave** ×1 — draft id 相关逻辑变更，测试需适配。

### 3.4 CSS 拆分

`frontend/workbench/src/styles/editor.css`（1485 行）按关注点拆分：

- `editor-typography.css` — 编辑器排版样式
- `chat-panel.css` — AI 对话面板样式
- `chat-animations.css` — 发送按钮起飞/飞行/坠毁动效
- `asset-workspace.css` — 文件工作区布局样式

## 4. 执行计划

| 阶段 | 任务 | 依赖 |
|------|------|------|
| PR #47 就地修复 | 5 项 bug 修复和代码清理 | 无 |
| Issue: 文件夹重构 | parent_id 列 + path 字段 | 无 |
| Issue: 测试修复 | 7 个前端测试 | 无 |
| Issue: CSS 拆分 | editor.css 拆分 | 无 |
| Issue: 设计知识库重构 Phase A | 纳入三层 Skill 系统 | Issue #45 完成 |
| Issue: 设计知识库重构 Phase B | 用户上传 + 组件提取 | Phase A 完成 |

## 5. 设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| Bug 修复范围 | 只修 5 项明确的 bug | PR #47 是别人的 PR，架构改动应独立进行 |
| 文件夹存储 | parent_id 列而非 JSONB | 标准 SQL 查询，不需要加载全部记录到内存 |
| 设计知识库 | 纳入三层 Skill 系统 | 复用现有架构，避免维护两套 skill 系统 |
| 组件提取 | CSS 解析 + AI 描述 | 硬数据确定性高，软数据需要 AI 理解能力 |
| Issue 合并 | 所有重构任务合并为一个 Issue | 都属于 PR #47 后续优化，方便追踪 |
