# 模块 library 契约：组件库与模板库

更新时间：2026-05-07

## 1. 模块职责

**负责**：

- 简历组件库的查询、预览、插入。
- 简历模板库的查询、预览、应用。
- 组件/模板元数据、标签、分类、风格描述。
- AI 助手对组件/模板的查询与推荐。

**不负责**：

- 文件上传与解析，属于 intake/parsing。
- 简历 HTML 自动保存，属于 workbench。
- PDF 导出，属于 render。
- AI 对话编排，属于 agent。

## 2. 数据模型草案

### `resume_components`

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 组件 ID，如 `timeline_compact` |
| `name` | string | 展示名称 |
| `category` | string | 分类：experience、project、skills、education、certificate、decoration |
| `description` | string | 描述 |
| `style_tags` | string[] | 风格标签 |
| `ai_tags` | string[] | AI 检索标签 |
| `preview_image` | string | 预览图 URL |
| `html_snippet` | string | 可插入 HTML |
| `schema` | json | 插槽定义 |
| `source` | string | system/user/team |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

### `resume_templates`

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 模板 ID |
| `name` | string | 展示名称 |
| `industry` | string[] | 适用行业 |
| `role_tags` | string[] | 岗位标签 |
| `style_tags` | string[] | 风格标签 |
| `ai_tags` | string[] | AI 检索标签 |
| `preview_image` | string | 模板预览图 |
| `html_snapshot` | string | 模板 HTML |
| `recommended_components` | string[] | 推荐搭配组件 |
| `source` | string | system/user/team |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

## 3. API 草案

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

### 查询组件

`GET /api/v1/library/components`

Query：

| 参数 | 说明 |
|---|---|
| `q` | 搜索关键词 |
| `category` | 组件分类 |
| `style_tags` | 逗号分隔 |
| `ai_tags` | 逗号分隔 |

Response：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "timeline_compact",
        "name": "紧凑时间线",
        "category": "experience",
        "description": "适合展示工作经历和项目时间线",
        "style_tags": ["professional", "compact"],
        "ai_tags": ["工作经历", "项目经历"],
        "preview_image": "/library/components/timeline_compact.png"
      }
    ],
    "total": 1
  },
  "message": "ok"
}
```

### 获取组件详情

`GET /api/v1/library/components/{component_id}`

返回 `html_snippet` 与 `schema`。

### 查询模板

`GET /api/v1/library/templates`

Query：

| 参数 | 说明 |
|---|---|
| `q` | 搜索关键词 |
| `industry` | 行业 |
| `role_tags` | 岗位标签 |
| `style_tags` | 风格标签 |

### 获取模板详情

`GET /api/v1/library/templates/{template_id}`

返回 `html_snapshot`、`recommended_components`。

## 4. 前端契约

### 组件库面板

组件：

```text
ComponentLibraryPanel
```

能力：

- 分类筛选
- 搜索
- 预览
- 插入到当前光标位置或指定 section

### 模板库面板

组件：

```text
TemplateLibraryPanel
```

能力：

- 行业/岗位/风格筛选
- 模板预览
- 应用模板
- 应用前确认覆盖范围

## 5. AI Tool 契约

### `library.search_components`

Input：

```json
{
  "query": "项目经历卡片",
  "category": "project",
  "style_tags": ["professional", "compact"],
  "limit": 5
}
```

Output：

```json
{
  "items": [
    {
      "id": "project_cards_compact",
      "name": "紧凑项目卡片",
      "reason": "适合技术简历项目经历，信息密度高"
    }
  ]
}
```

### `library.search_templates`

Input：

```json
{
  "query": "前端工程师 React",
  "industry": "software",
  "style_tags": ["modern", "professional"],
  "limit": 3
}
```

### `library.insert_component`

Input：

```json
{
  "component_id": "project_cards_compact",
  "target": "current_selection",
  "payload": {
    "title": "项目经历",
    "items": []
  }
}
```

约束：

- 用户确认前不得直接写入。
- 插入前建议创建版本快照。

### `library.apply_template`

Input：

```json
{
  "template_id": "frontend_modern_compact",
  "mode": "preserve_content"
}
```

约束：

- 必须弹出确认。
- `replace_all` 模式会覆盖当前 HTML。
- `preserve_content` 模式尽量保留现有文本并填入新模板。

## 6. Skill 数据契约

外部 skill 位于：

```text
backend/internal/modules/library/skill/ui-ux-pro-max/
```

Go 查询器位于：

```text
backend/cmd/uxsearch/
```

运行时业务 API 通过 Go embed 只读查询 skill CSV。CSV 用于：

- 设计调研
- seed 数据生成
- 风格标签标准化
- AI prompt/tool 描述补充

