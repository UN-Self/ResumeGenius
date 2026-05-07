# 模块 library 设计方案：组件库、模板库与 UI/UX Skill 融合

更新时间：2026-05-07

## 1. 背景

ResumeGenius 后续需要两类可复用资源：

- **图案组件库**：技能条、时间线、项目卡片、证书块、分隔线、图标组等可插入简历局部的组件。
- **简历模板库**：面向岗位、行业、风格的整页简历模板。

`ui-ux-pro-max` 是外部 UI/UX Agent Skill，包含设计风格、配色、字体、UX guidelines、技术栈最佳实践等 CSV 数据和查询脚本。它适合作为“设计知识来源”，不适合作为线上业务的直接运行依赖。

## 2. 目录调整

本地 skill 放置在：

```text
backend/internal/modules/library/skill/ui-ux-pro-max/
```

Go 查询器放置在：

```text
backend/cmd/uxsearch/
```

原因：

- `backend/internal/modules/library` 明确属于组件库/模板库建设工具链。
- skill CSV 会被 Go embed 打进后端，避免 Docker runtime 丢失数据文件。
- 方便后续构建 seed 数据、离线查询和生成文档。

## 3. 角色边界

### `ui-ux-pro-max` 负责

- 提供 UI/UX 设计参考。
- 提供可搜索的 CSV 数据：
  - styles
  - colors
  - typography
  - ux guidelines
  - product recommendations
  - stack best practices

### ResumeGenius library 模块负责

- 定义组件与模板的数据模型。
- 提供用户可浏览、搜索、预览、插入的面板。
- 提供 AI 可调用的组件/模板查询和应用工具。
- 维护用户自定义组件、收藏、版本和权限。

## 4. 推荐数据流

```text
ui-ux-pro-max CSV
  -> Go 查询器 uxsearch
  -> 人工/脚本生成 ResumeGenius seed 数据
  -> resume_components / resume_templates
  -> 左侧组件库和模板库
  -> AI tools 与用户交互共同调用
```

运行时不直接调用 Python skill 脚本。

## 5. Go 查询器

命令位置：

```text
backend/cmd/uxsearch/main.go
```

示例：

```powershell
cd backend
go run ./cmd/uxsearch "professional resume dashboard" --domain style --max 3
go run ./cmd/uxsearch "React component accessibility" --stack react --json
```

能力：

- 读取 `backend/internal/modules/library/skill/ui-ux-pro-max/data`。
- 支持 domain 查询：`style`、`prompt`、`color`、`chart`、`landing`、`product`、`ux`、`typography`。
- 支持 stack 查询：`html-tailwind`、`react`、`nextjs`、`vue`、`svelte`、`swiftui`、`react-native`、`flutter`。
- 使用 Go 标准库实现 BM25 风格排序。

## 6. 前端落点

当前左侧 Activity Bar 已预留：

- 文件
- 图案组件库
- 简历模板库

后续新增组件：

```text
frontend/workbench/src/components/library/ComponentLibraryPanel.tsx
frontend/workbench/src/components/library/TemplateLibraryPanel.tsx
frontend/workbench/src/components/library/LibraryCard.tsx
frontend/workbench/src/components/library/LibraryPreviewDialog.tsx
```

## 7. AI 调用模式

AI 不直接读 CSV。AI 通过后端 tool 调用 library API：

- 查询组件
- 查询模板
- 推荐组件组合
- 插入组件
- 应用模板

用户确认前，AI 不应直接覆盖当前简历。

## 8. 分阶段落地

### Phase 1：静态库

- 用 TypeScript seed 文件提供 5-10 个组件和 3-5 个模板。
- 左侧面板支持搜索、分类、预览。

### Phase 2：后端库

- 建表 `resume_components`、`resume_templates`。
- 提供列表、详情、搜索 API。

### Phase 3：AI tools

- Agent 注册 library tools。
- AI 可查询和推荐，但插入/应用前必须用户确认。

### Phase 4：用户资产

- 支持用户收藏、自定义组件、团队共享、版本管理。

