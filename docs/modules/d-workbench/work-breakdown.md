# 模块 D 工作明细：可视化编辑器

更新时间：2026-04-23

本文档列出模块 D 负责人的全部开发任务。契约定义见 [contract.md](./contract.md)。

## 1. 概述

模块 D 是用户编辑简历的核心 UI，集成 TipTap 所见即所得编辑器，提供 A4 画布和丰富的格式工具栏。所有编辑直接修改 HTML，自动保存到后端。

**核心交付**：用户能在工作台中像用 Word 一样直接编辑简历，实时预览，自动保存。

## 2. 前端任务

### 2.1 页面

| # | 页面 | 路由建议 | 说明 |
|---|---|---|---|
| 1 | 工作台 | `/projects/[id]/edit` | A4 画布 + AI 面板 + 工具栏 |

### 2.2 组件

| # | 组件 | 说明 |
|---|---|---|
| 1 | `WorkbenchLayout` | 工作台主布局：顶部操作栏 + 左侧画布 + 右侧 AI 面板 + 底部工具栏 |
| 2 | `ResumeCanvas` | A4 画布容器，负责缩放和滚动 |
| 3 | `TipTapEditor` | TipTap 编辑器实例，加载和渲染 HTML |
| 4 | `FormatToolbar` | 格式工具栏：加粗/斜体/下划线/字号/颜色/对齐/行距/列表 |
| 5 | `FontSizePicker` | 字号下拉选择器 |
| 6 | `ColorPicker` | 颜色选择器 |
| 7 | `ImageUploader` | 图片上传组件（头像等） |
| 8 | `SaveIndicator` | 保存状态指示：保存中 / 已保存 |
| 9 | `ActionBar` | 顶部操作栏：项目名 + 保存 + 版本历史 + 导出 |

### 2.3 前端技术要点

- TipTap 初始化时加载 drafts.html_content
- 监听 `update` 事件，debounce 2 秒自动保存
- A4 画布使用 CSS 固定尺寸（210mm × 297mm），通过 `transform: scale()` 缩放适应屏幕
- 工具栏使用 TipTap 的 `BubbleMenu` 或固定底部栏
- 图片上传走 `POST /api/v1/assets/upload`，返回 URL 后插入 `<img>`
- 保存失败时显示错误提示，不中断编辑
- 编辑器内容使用简历 HTML 模板骨架的 CSS class

### 2.4 前端状态管理

| 状态 | 说明 |
|---|---|
| `currentHtml` | 当前编辑器 HTML 内容 |
| `isSaving` | 是否正在保存 |
| `lastSavedAt` | 最后保存时间 |
| `hasUnsavedChanges` | 是否有未保存的变更 |

## 3. 后端任务

### 3.1 API 端点（2 个）

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 1 | GET | `/api/v1/drafts/{draft_id}` | 获取草稿 HTML |
| 2 | PUT | `/api/v1/drafts/{draft_id}` | 保存草稿 HTML |

### 3.2 后端服务

| # | 服务 | 说明 |
|---|---|---|
| 1 | `DraftService` | 草稿 CRUD，读取和更新 html_content |

### 3.3 后端技术要点

- GET 返回完整 html_content
- PUT 接收 html_content，更新 drafts 表的 html_content 和 updated_at
- PUT 请求支持可选的 `create_version` 参数（默认 false）和 `version_label` 参数
- 自动保存不触发版本创建，仅显式请求时创建版本快照
- 前端自动保存频繁调用，PUT 需要轻量高效
- 不需要复杂的校验逻辑（HTML 校验交给前端 TipTap）

## 4. 数据库表

| 表名 | 说明 |
|---|---|
| `drafts` | 简历草稿（id, project_id, html_content, created_at, updated_at） |

- 模块 B 创建，模块 D 更新 html_content，模块 E 读取

## 5. 测试任务

### 5.1 后端单元测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 获取草稿 | GET 返回正确 html_content |
| 2 | 保存草稿 | PUT 更新 html_content 和 updated_at |
| 3 | 草稿不存在 | 返回 4001 |

### 5.2 前端测试

| # | 测试 | 说明 |
|---|---|---|
| 1 | 编辑器渲染 | HTML 正确渲染为所见即所得 |
| 2 | 格式工具栏 | 加粗、斜体、字号等操作正确 |
| 3 | 自动保存 | 编辑后 debounce 触发保存 |
| 4 | A4 缩放 | 不同屏幕尺寸下正确缩放 |
| 5 | 图片上传 | 选择图片 → 上传 → 插入编辑器 |
| 6 | 撤销/重做 | 操作后撤销、撤销后重做 |

### 5.3 Mock 策略

- 不需要其他模块的服务
- 前端用 fixtures/sample_draft.html 直接加载编辑器
- 可完全独立开发和测试

## 6. 交付 Checklist

- [ ] 前端：1 个页面 + 9 个组件
- [ ] 后端：2 个 API 端点
- [ ] 后端服务：1 个核心服务（DraftService）
- [ ] 数据库：使用 drafts 表
- [ ] TipTap 集成：10+ 扩展（bold, italic, underline, color, alignment, lists 等）
- [ ] 测试：3 个后端单元测试 + 6 个前端测试
- [ ] 错误码使用 4001–4999 范围
