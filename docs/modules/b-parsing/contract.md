# 模块 B 契约：文件解析与 AI 初稿生成

更新时间：2026-04-23

## 1. 角色定义

**负责**：

- PDF 文件解析（提取文本块和内嵌图片）
- DOCX 文件解析（提取段落、表格、样式）
- Git 仓库信息抽取（README、技术栈）
- 将提取的文本发送给 AI
- AI 根据简历 HTML 模板骨架生成完整简历 HTML
- 将 HTML 存入 drafts 表

**不负责**：

- 文件上传和存储（A 的事）
- AI 对话修改（C 的事）
- 所见即所得编辑（D 的事）
- 版本管理和 PDF 导出（E 的事）

## 2. 输入契约

| 数据 | 来源 | 说明 |
|---|---|---|
| assets 表记录 | 模块 A | 文件路径（uri 字段）或补充文本（content 字段） |

Mock：直接用 `fixtures/sample_resume.pdf` 等测试文件。

## 3. 输出契约

产出写入数据库：

| 表 | 字段 | 说明 |
|---|---|---|
| `drafts` | `html_content` | 完整的简历 HTML（可直接在浏览器中渲染） |
| `projects` | `current_draft_id` | 关联最新草稿 |

AI 输入：
- 简历 HTML 模板骨架（包含 CSS 样式和语义结构）
- 用户上传文件中提取的文本内容
- 用户补充的文本资料

AI 输出：
- 完整的简历 HTML（可直接在浏览器中渲染）

版本快照：AI 初稿生成成功后，服务端自动调用版本创建逻辑，写入 versions 表一条记录，label = `"AI 初始生成"`。

## 4. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/parsing/parse` | 触发解析（同步） |
| POST | `/api/v1/parsing/generate` | 触发 AI 初稿生成（同步） |

### 关键端点详情

#### POST /api/v1/parsing/parse

解析项目中的所有文件，提取文本内容。

```
Request:
{
  "project_id": 1
}

Response (成功):
{
  "code": 0,
  "data": {
    "parsed_contents": [
      {
        "asset_id": 1,
        "type": "resume_pdf",
        "text": "张三 | 前端工程师\n\n工作经历\nABC科技 高级前端工程师 2022.07-至今\n· 主导核心产品前端架构重构...",
        "images": [
          {
            "description": "头像",
            "data_base64": "..."
          }
        ]
      }
    ]
  }
}
```

#### POST /api/v1/parsing/generate

将提取的文本发送给 AI，生成初始简历 HTML。

```
Request:
{
  "project_id": 1
}

Response (成功):
{
  "code": 0,
  "data": {
    "draft_id": 1,
    "version_id": 1,
    "html_content": "<!DOCTYPE html><html>..."
  }
}

Response (AI 调用失败):
{
  "code": 2005,
  "data": null,
  "message": "AI 初稿生成失败：模型调用超时"
}
```

## 5. 解析策略

| 输入类型 | 策略 |
|---|---|
| PDF | ledongthuc/pdf 原生解析（文本 + 图片提取） |
| DOCX | nguyenthenguyen/docx 段落/表格/样式提取 |
| 图片 | v1 跳过，不发送给 AI。图片存储在 assets 表供前端手动引用（如头像），后续通过云端 OCR API 兜底 |
| Git | clone → 抽 README + 项目名 + 技术栈 + 目录结构 |
| 补充文本 | 直接使用 content 字段 |

## 6. AI 初稿生成

### 6.1 Prompt 构建

将以下内容组装成 AI Prompt：

1. **系统角色**：简历优化助手
2. **HTML 模板骨架**：包含 CSS 样式和语义 class 的 HTML 结构
3. **用户资料**：从文件中提取的文本 + 补充文本
4. **输出要求**：返回完整简历 HTML

### 6.2 HTML 模板骨架

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <style>
    @page { size: A4; margin: 0; }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Noto Sans SC', sans-serif; font-size: 10.5pt; line-height: 1.4; color: #333; }
    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
    .profile { display: flex; align-items: center; gap: 16px; margin-bottom: 12pt; }
    .avatar { width: 48pt; height: 48pt; border-radius: 50%; object-fit: cover; }
    .profile h1 { font-size: 18pt; font-weight: 700; }
    .profile p { font-size: 10pt; color: #666; margin-top: 2pt; }
    .section { margin-bottom: 10pt; }
    .section h2 { font-size: 12pt; font-weight: 600; border-bottom: 1pt solid #ddd; padding-bottom: 3pt; margin-bottom: 6pt; }
    .item { margin-bottom: 6pt; }
    .item-header { display: flex; justify-content: space-between; }
    .item h3 { font-size: 10.5pt; font-weight: 600; }
    .item .date { font-size: 9pt; color: #888; }
    .item .subtitle { font-size: 9.5pt; color: #555; }
    .item ul { padding-left: 14pt; }
    .item li { margin-bottom: 2pt; }
    .tags { display: flex; flex-wrap: wrap; gap: 6pt; }
    .tag { background: #f0f0f0; padding: 2pt 8pt; border-radius: 3pt; font-size: 9pt; }
  </style>
</head>
<body>
  <div class="resume">
    <header class="profile">
      <!-- AI 填充：头像、姓名、职位、联系方式 -->
    </header>
    <!-- AI 自由生成 section -->
  </div>
</body>
</html>
```

AI 不受固定 section 类型的约束，可以自由生成内容结构。

## 7. 依赖与边界

### 上游

- 模块 A（项目管理）提供 assets 表中的文件路径和文本

### 下游

- 模块 D（编辑器）消费 drafts.html_content
- 模块 C（AI 对话）消费 drafts.html_content
- 模块 E（版本快照：初稿生成后自动创建版本）

### 可 mock 的边界

- 不需要 A 的服务：直接读 fixtures/ 中的测试文件
- AI 调用：用预设 HTML 替代真实模型调用

## 8. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 2001 | 400 | PDF 解析失败 |
| 2002 | 400 | DOCX 解析失败 |
| 2003 | 404 | 项目不存在 |
| 2004 | 400 | 项目无可用资产 |
| 2005 | 500 | AI 初稿生成失败 |

## 9. 测试策略

### 独立测试

- 用本地测试文件（`fixtures/sample_resume.pdf` 等）测试各种格式的解析
- AI 调用用 mock 替代
- 不需要启动模块 A、C、D、E 的服务

### Mock 产出

确保产出的 HTML 可以直接在浏览器中渲染，且结构符合简历模板骨架。

### 前端测试

- 解析结果展示（文本预览 + 图片预览）
- 初稿生成进度（同步等待 loading）
