# 模块 parsing 契约：文件解析与文本清洗

更新时间：2026-05-11

> 说明：本文件描述当前分支已实现的 parsing 素材正文链路。
> 重点包括：文本清洗、正文回写、派生图片持久化、原始文件删除。

## 1. 角色定义

**负责**：

- PDF 文件解析（提取文本块和内嵌图片）
- DOCX 文件解析（提取段落、表格、样式）
- Git 仓库 AI 深度分析（生成 `repository_{name}.md`，含常用命令、高层架构、规则合并；AI 不可用时 fallback 到正则 README 提取）
- 文本清洗和持久化回写

**不负责**：

- 文件上传和存储（intake 的事）
- AI 对话修改（agent 的事）
- 所见即所得编辑（workbench 的事）
- 版本管理和 PDF 导出（render 的事）

## 2. 输入契约

| 数据 | 来源 | 说明 |
|---|---|---|
| assets 表记录 | 模块 intake | 原始来源（`uri`）与已持久化素材正文（`content`） |

Mock：直接用 `fixtures/sample_resume.pdf` 等测试文件。

## 3. 输出契约

产出写入数据库：

| 表 | 字段 | 说明 |
|---|---|---|
| `assets` | `content` | 清洗后的素材正文。note 保留用户原文，PDF / DOCX / Git 回写 AI 可消费文本 |
| `assets` | `metadata` | 解析状态、清洗标记、派生图片关系、人工修改标记、最近解析时间、原件删除状态等信息 |

## 4. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/parsing/parse` | 触发解析（同步） |

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
        "label": "sample_resume.pdf",
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

Notes:
- `parsed_contents[].label` uses `assets.label` when present; otherwise it falls back to the asset file name.
- `/parse` 在返回预览的同时，会将 PDF / DOCX / Git 的清洗后正文回写到 `assets.content`，并更新 `metadata.parsing`。

## 5. 解析策略

| 输入类型 | 策略 |
|---|---|
| PDF | ledongthuc/pdf 原生解析（文本 + 图片提取）→ 文本清洗 → 回写 `assets.content` |
| DOCX | nguyenthenguyen/docx 段落/表格/样式提取 → 文本清洗 → 回写 `assets.content` |
| 图片 | 上传得到的 `resume_image` 在 v1 仍跳过解析；PDF / DOCX 解析出的图片会持久化成派生 `resume_image` 资产供前端引用 |
| Git | clone → 收集上下文（README、技术栈、目录结构、`.cursor/rules/`、构建配置）→ AI 单次请求生成 `repository_{name}.md` → 回写 `assets.content`。AI 未配置时 fallback 到正则 README + 技术栈 + 目录结构提取 |
| 补充文本 | 直接使用 `content` 字段，并做轻量清洗 |

清洗目标：

- 去掉多余空行、空白、分隔线
- 保留段落边界和 bullet 结构
- 输出"适合 AI 消费"的素材正文，而不是原始抽取大文本

源文件处理行为：

- 当正文与需要保留的派生图片都已成功持久化后，原始 PDF / DOCX 文件会被删除
- 删除后资产的 `uri` 会被清空，同时在 `metadata.parsing` 中保留 `source_deleted`、`original_uri`、`original_filename`
- 后续重新解析这类资产时，会优先从已持久化的 `assets.content` 回退构建 `ParsedContent`

## 6. 依赖与边界

### 上游

- 模块 intake（项目管理）提供 assets 表中的原始来源与资产记录壳

### 下游

- 模块 agent（AI 对话）消费 `assets.content` 作为 AI 输入
- 模块 workbench（编辑器）消费 `drafts.html_content`，并以 `assets.content` 驱动左栏素材管理

### 可 mock 的边界

- 不需要 intake 的服务：直接读 fixtures/ 中的测试文件

## 7. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 2001 | 400 | PDF 解析失败 |
| 2002 | 400 | DOCX 解析失败 |
| 2003 | 404 | 项目不存在 |
| 2004 | 400 | 项目无可用资产 |
| 2007 | 400 | Git 仓库提取失败（clone 或读取异常） |
| 2008 | 400 | Git 仓库 AI 分析失败（AI API 调用异常） |

## 8. 测试策略

### 独立测试

- 用本地测试文件（`fixtures/sample_resume.pdf` 等）测试各种格式的解析
- 不需要启动模块 intake、agent、workbench、render 的服务

### 前端测试

- 解析结果展示（文本预览 + 图片预览）
