# 模块 D 契约：可视化编辑器

更新时间：2026-04-23

## 1. 角色定义

**负责**：

- TipTap 所见即所得编辑器集成
- A4 尺寸编辑画布（CSS 固定 210mm × 297mm）
- 富文本编辑：加粗、斜体、下划线、字号、颜色、对齐、行距
- 列表：有序/无序
- 图片：上传头像、拖拽调整
- Section 拖拽排序
- 原生 undo/redo（ProseMirror 内置）
- 自动保存（debounce 2 秒）

**不负责**：

- AI 对话（C 的事）
- 版本管理和 PDF 导出（E 的事）
- 文件解析（B 的事）

## 2. 输入契约

| 数据 | 来源 | 说明 |
|---|---|---|
| `drafts.html_content` | 模块 B/C | 当前简历 HTML |

Mock：直接用 `fixtures/sample_draft.html` 加载到编辑器。

## 3. 输出契约

编辑后的 HTML 写入 `drafts.html_content`。

通过 `PUT /api/v1/drafts/{draft_id}` 保存。

## 4. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/drafts/{draft_id}` | 获取草稿 HTML |
| PUT | `/api/v1/drafts/{draft_id}` | 保存草稿 HTML（自动保存） |

### 关键端点详情

#### GET /api/v1/drafts/{draft_id}

```
Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "project_id": 1,
    "html_content": "<!DOCTYPE html>...",
    "updated_at": "2026-04-23T20:00:00Z"
  }
}
```

#### PUT /api/v1/drafts/{draft_id}

```
Request:
{
  "html_content": "<!DOCTYPE html>...编辑后的 HTML...",
  "create_version": true,
  "version_label": "AI 修改：精简项目经历"
}

Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "updated_at": "2026-04-23T20:05:00Z",
    "version_id": 5
  }
}
```

> `create_version`（可选，默认 false）：自动保存时不传，仅当需要创建版本快照时传 true。
> `version_label`（可选，默认 "手动保存"）：版本标签，配合 `create_version` 使用。
>
> 自动保存（debounce 2s）不触发版本创建，避免版本爆炸。

## 5. TipTap 编辑器功能

### 5.1 格式工具栏

| 功能 | TipTap 扩展 | 说明 |
|---|---|---|
| 加粗 | `@tiptap/extension-bold` | Ctrl+B |
| 斜体 | `@tiptap/extension-italic` | Ctrl+I |
| 下划线 | `@tiptap/extension-underline` | Ctrl+U |
| 字号 | 自定义 | 下拉选择 9-14pt |
| 颜色 | `@tiptap/extension-color` | 颜色选择器 |
| 对齐 | `@tiptap/extension-text-align` | 左/中/右/两端 |
| 行距 | 自定义 | 1.0 / 1.2 / 1.5 / 1.8 / 2.0 |
| 有序列表 | `@tiptap/extension-ordered-list` | — |
| 无序列表 | `@tiptap/extension-bullet-list` | — |
| 撤销 | ProseMirror 内置 | Ctrl+Z |
| 重做 | ProseMirror 内置 | Ctrl+Shift+Z |

### 5.2 图片处理

- 头像上传：通过图片选择器选择本地图片，上传到服务器，插入 `<img>` 标签
- 图片拖拽调整：使用 `@tiptap-pro/extension-resize-image` 或自定义拖拽逻辑

### 5.3 Section 排序

- 使用 `@dnd-kit/core` 或 ProseMirror 原生拖拽实现 section 级别的拖拽排序

### 5.4 自动保存

- 监听 TipTap 的 `update` 事件
- debounce 2 秒后调用 `PUT /api/v1/drafts/{draft_id}`
- 保存状态指示：保存中（旋转图标） / 已保存（勾号）

## 6. A4 画布样式

```css
.editor-canvas {
  width: 210mm;
  min-height: 297mm;
  padding: 18mm 20mm;
  background: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
  margin: 0 auto;
}

/* 缩放适应屏幕 */
.editor-wrapper {
  transform-origin: top center;
  transform: scale(var(--canvas-scale));
}
```

编辑器内容使用简历 HTML 模板骨架中定义的 CSS class（.resume, .profile, .section, .item 等）。

## 7. 依赖与边界

### 上游

- 模块 B 产出初始 drafts.html_content
- 模块 C 通过 AI 对话修改后，前端替换 HTML

### 下游

- 模块 E 消费 drafts.html_content（创建版本快照、PDF 导出）
- 模块 E（当 create_version=true 时，创建版本快照）

### 可 mock 的边界

- 不需要 B/C/E 的服务
- 前端可完全独立开发，用 mock HTML 加载编辑器

## 8. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 4001 | 404 | 草稿不存在 |
| 4002 | 400 | HTML 内容为空 |

## 9. 测试策略

### 独立测试

- 用 `fixtures/sample_draft.html` 加载编辑器
- 测试各种编辑操作的正确性
- 测试自动保存逻辑
- 不需要其他模块的服务

### 前端测试

- 编辑器渲染：HTML 正确渲染为所见即所得
- 工具栏：各格式按钮功能正确
- 自动保存：编辑后 debounce 2 秒触发保存
- 缩放：A4 画布在不同屏幕尺寸下正确缩放
