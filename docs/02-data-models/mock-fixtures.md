# Mock Fixture 策略

更新时间：2026-04-23

本文档定义每个模块的 mock 数据策略，确保每个人可以独立开发测试，不依赖上下游模块的实际实现。

## 1. v2 简化说明

v2 砍掉了 v1 的 SourceAssetSet / EvidenceSet / ResumeDraftState / PatchEnvelope / ResolvedResumeSpec 等中间 JSON 结构，mock 策略相应大幅简化。

v2 的 mock 主要是：
- **测试文件**：sample_resume.pdf / sample_resume.docx（用于解析模块测试）
- **HTML fixture**：sample_draft.html（模拟 AI 生成的简历 HTML）
- **AI 响应 fixture**：sample_ai_response.json（模拟 AI 对话返回）

## 2. 原则

- **每个模块用文件 fixture mock**，不依赖真实服务或数据库
- **mock fixture 和数据库 schema 对齐**，不依赖中间层
- **上游 mock = 下游的输入 fixture**，上下游的 fixture 必须和 `02-data-models/core-data-model.md` 中的表结构一致
- **每个模块自己决定内部测试数据**，只要外部契约对齐即可

## 3. Fixture 文件组织

```
fixtures/
  sample_resume.pdf           # intake 消费（解析模块测试）
  sample_resume.docx          # intake 消费（解析模块测试）
  sample_draft.html           # parsing 产出 / agent/workbench/render 消费（模拟 AI 生成的简历 HTML）
  sample_ai_response.json     # agent 消费（模拟 AI 对话返回）
```

每个模块的开发目录下可以有自己的测试 fixture 副本，但**契约 fixture**（上面的文件）由数据模型负责人维护，放在仓库根目录的 `fixtures/` 下。

## 4. 各模块 mock 策略

### 模块 intake：项目管理与资料接入

**上游**：无（intake 是起点）

**自己测试用的 mock**：
- `fixtures/sample_resume.pdf` — 用于测试文件上传、存储、assets 记录创建
- `fixtures/sample_resume.docx` — 同上

**产出的 fixture（给 B 用）**：
- 磁盘上的文件路径（模拟 `assets.uri`）
- 数据库 `assets` 表记录（Go struct 或 JSON）

**Go 代码示例**：

```go
// 模拟模块 intake 的产出：创建一个 asset 记录
func mockCreateAsset(projectID uint) *Asset {
    return &Asset{
        ProjectID: projectID,
        Type:      "resume_pdf",
        URI:       ptrString("fixtures/sample_resume.pdf"),
        Metadata: JSONB{
            "filename":    "sample_resume.pdf",
            "size_bytes":  102400,
            "mime_type":   "application/pdf",
            "uploaded_at": "2026-04-23T10:00:00Z",
        },
    }
}

// 模拟补充文本 asset
func mockCreateNote(projectID uint) *Asset {
    content := "目标岗位是全栈工程师，希望突出 Go 和 React 经验"
    label := "求职意向"
    return &Asset{
        ProjectID: projectID,
        Type:      "note",
        Content:   &content,
        Label:     &label,
    }
}
```

### 模块 parsing：文件解析与 AI 初稿生成

**输入 mock（替代 intake 的产出）**：直接读 `fixtures/sample_resume.pdf` 和 `fixtures/sample_resume.docx`

**自己测试用的 mock**：
- 模拟 PDF 解析结果（纯文本提取）
- 模拟 AI 模型返回（不调真实 AI API，用预设 HTML 替代）

**产出的 fixture（给 agent/workbench/render 用）**：

```html
<!-- fixtures/sample_draft.html -->
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
      <h1>张三</h1>
      <p>前端工程师 | zhangsan@email.com | 138xxxx1234</p>
    </header>
    <section class="section">
      <h2>工作经历</h2>
      <div class="item">
        <div class="item-header">
          <h3>ABC 科技</h3>
          <span class="date">2022.07 - 至今</span>
        </div>
        <div class="subtitle">高级前端工程师</div>
        <ul>
          <li>主导核心产品前端架构重构，将 jQuery 迁移至 React + TypeScript</li>
          <li>设计并实现组件库，覆盖 30+ 业务组件，团队效率提升 40%</li>
          <li>推动前端工程化建设，引入 CI/CD、单元测试、Code Review 机制</li>
        </ul>
      </div>
    </section>
    <section class="section">
      <h2>项目经历</h2>
      <div class="item">
        <div class="item-header">
          <h3>ResumeGenius</h3>
          <span class="date">2025.10 - 2026.04</span>
        </div>
        <div class="subtitle">个人项目</div>
        <ul>
          <li>基于 Gin + React 构建简历编辑器，支持 AI 辅助编辑</li>
          <li>使用 TipTap 实现所见即所得编辑，chromedp 实现 PDF 导出</li>
        </ul>
      </div>
    </section>
    <section class="section">
      <h2>教育经历</h2>
      <div class="item">
        <div class="item-header">
          <h3>某大学</h3>
          <span class="date">2018.09 - 2022.06</span>
        </div>
        <div class="subtitle">计算机科学与技术 本科</div>
      </div>
    </section>
    <section class="section">
      <h2>专业技能</h2>
      <div class="tags">
        <span class="tag">TypeScript</span>
        <span class="tag">React</span>
        <span class="tag">Go</span>
        <span class="tag">Node.js</span>
        <span class="tag">PostgreSQL</span>
        <span class="tag">Docker</span>
      </div>
    </section>
  </div>
</body>
</html>
```

**Go 代码示例**：

```go
// 模拟模块 parsing 的产出：读取 fixture HTML 作为初稿
func mockGenerateDraft(htmlPath string) *Draft {
    htmlContent, _ := os.ReadFile(htmlPath)
    return &Draft{
        HTMLContent: string(htmlContent),
    }
}

// 模拟 PDF 解析结果（返回纯文本）
func mockParsePDF(pdfPath string) string {
    return `张三 | 前端工程师 | zhangsan@email.com | 138xxxx1234

工作经历
ABC 科技 | 高级前端工程师 | 2022.07 - 至今
- 主导核心产品前端架构重构，将 jQuery 迁移至 React + TypeScript
- 设计并实现组件库，覆盖 30+ 业务组件，团队效率提升 40%
- 推动前端工程化建设，引入 CI/CD、单元测试、Code Review 机制

教育经历
某大学 | 计算机科学与技术 本科 | 2018.09 - 2022.06

专业技能
TypeScript, React, Go, Node.js, PostgreSQL, Docker`
}
```

### 模块 agent：AI 对话助手

**输入 mock（替代 parsing 的产出）**：直接读 `fixtures/sample_draft.html`

**自己测试用的 mock**：
- 模拟 AI 模型返回（不调真实 AI API，用预设 JSON 替代）

**产出的 fixture**：

```json
// fixtures/sample_ai_response.json
{
  "text_explanation": "好的，我将项目经历部分精简为两条 bullet，使其更聚焦核心贡献。",
  "modified_html": "<!DOCTYPE html>\n<html lang=\"zh-CN\">\n<head>\n  <meta charset=\"UTF-8\" />\n  <style>\n    @page { size: A4; margin: 0; }\n    * { margin: 0; padding: 0; box-sizing: border-box; }\n    body { font-family: 'Noto Sans SC', sans-serif; font-size: 10.5pt; line-height: 1.4; color: #333; }\n    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }\n    .profile { display: flex; align-items: center; gap: 16px; margin-bottom: 12pt; }\n    .profile h1 { font-size: 18pt; font-weight: 700; }\n    .profile p { font-size: 10pt; color: #666; margin-top: 2pt; }\n    .section { margin-bottom: 10pt; }\n    .section h2 { font-size: 12pt; font-weight: 600; border-bottom: 1pt solid #ddd; padding-bottom: 3pt; margin-bottom: 6pt; }\n    .item { margin-bottom: 6pt; }\n    .item-header { display: flex; justify-content: space-between; }\n    .item h3 { font-size: 10.5pt; font-weight: 600; }\n    .item .date { font-size: 9pt; color: #888; }\n    .item .subtitle { font-size: 9.5pt; color: #555; }\n    .item ul { padding-left: 14pt; }\n    .item li { margin-bottom: 2pt; }\n    .tags { display: flex; flex-wrap: wrap; gap: 6pt; }\n    .tag { background: #f0f0f0; padding: 2pt 8pt; border-radius: 3pt; font-size: 9pt; }\n  </style>\n</head>\n<body>\n  <div class=\"resume\">\n    <header class=\"profile\">\n      <h1>张三</h1>\n      <p>前端工程师 | zhangsan@email.com | 138xxxx1234</p>\n    </header>\n    <section class=\"section\">\n      <h2>工作经历</h2>\n      <div class=\"item\">\n        <div class=\"item-header\">\n          <h3>ABC 科技</h3>\n          <span class=\"date\">2022.07 - 至今</span>\n        </div>\n        <div class=\"subtitle\">高级前端工程师</div>\n        <ul>\n          <li>主导核心产品前端架构重构，将 jQuery 迁移至 React + TypeScript</li>\n          <li>设计并实现组件库，覆盖 30+ 业务组件，团队效率提升 40%</li>\n        </ul>\n      </div>\n    </section>\n    <section class=\"section\">\n      <h2>项目经历</h2>\n      <div class=\"item\">\n        <div class=\"item-header\">\n          <h3>ResumeGenius</h3>\n          <span class=\"date\">2025.10 - 2026.04</span>\n        </div>\n        <div class=\"subtitle\">个人项目</div>\n        <ul>\n          <li>基于 Gin + React 构建简历编辑器，TipTap 编辑 + chromedp PDF 导出</li>\n        </ul>\n      </div>\n    </section>\n    <section class=\"section\">\n      <h2>教育经历</h2>\n      <div class=\"item\">\n        <div class=\"item-header\">\n          <h3>某大学</h3>\n          <span class=\"date\">2018.09 - 2022.06</span>\n        </div>\n        <div class=\"subtitle\">计算机科学与技术 本科</div>\n      </div>\n    </section>\n    <section class=\"section\">\n      <h2>专业技能</h2>\n      <div class=\"tags\">\n        <span class=\"tag\">TypeScript</span>\n        <span class=\"tag\">React</span>\n        <span class=\"tag\">Go</span>\n        <span class=\"tag\">Node.js</span>\n        <span class=\"tag\">PostgreSQL</span>\n        <span class=\"tag\">Docker</span>\n      </div>\n    </section>\n  </div>\n</body>\n</html>"
}
```

**Go 代码示例**：

```go
// 模拟 AI 响应，返回结构化结果
func mockAIResponse() *AIResponse {
    return &AIResponse{
        TextExplanation: "好的，我将项目经历部分精简为一条 bullet，使其更聚焦核心贡献。",
        ModifiedHTML:    loadFixture("sample_draft_modified.html"),
    }
}

// AI 消息存储格式（存入 ai_messages 表）
// assistant 消息中 HTML 通过分隔符标识
func formatAssistantMessage(resp *AIResponse) string {
    return resp.TextExplanation +
        "\n<!--RESUME_HTML_START-->\n" +
        resp.ModifiedHTML +
        "\n<!--RESUME_HTML_END-->\n"
}

// 从 assistant 消息中提取 HTML
func extractHTMLFromAssistant(content string) string {
    start := strings.Index(content, "<!--RESUME_HTML_START-->\n")
    end := strings.Index(content, "\n<!--RESUME_HTML_END-->")
    if start == -1 || end == -1 || end <= start {
        return ""
    }
    return content[start+len("<!--RESUME_HTML_START-->\n") : end]
}
```

### 模块 workbench：可视化编辑器

**输入 mock（替代 parsing 的产出）**：直接读 `fixtures/sample_draft.html`

**自己测试用的 mock**：
- 模拟 TipTap 编辑器内容（React 组件渲染）
- 模拟用户操作（文本修改、格式调整、拖拽排序）

**无独立产出 fixture**：编辑器直接操作 `drafts.html_content`，通过 `PUT /api/v1/drafts/{draft_id}` 保存。

**TypeScript 代码示例**：

```typescript
// 前端示例：用 fixture 替代 API 响应
import sampleDraft from '../../fixtures/sample_draft.html?raw';

function EditorWithMock() {
  const editor = useEditor({
    extensions: [StarterKit],
    content: sampleDraft, // 直接加载 fixture HTML
  });

  return <EditorContent editor={editor} />;
}

// 模拟自动保存
const autoSave = useDebounceFn(async (html: string) => {
  await fetch(`/api/v1/drafts/${draftId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ html_content: html }),
  });
}, 2000);
```

### 模块 render：版本管理与 PDF 导出

**输入 mock（替代 agent/workbench 的产出）**：直接读 `fixtures/sample_draft.html`

**自己测试用的 mock**：
- 模拟版本快照创建
- 模拟 chromedp 导出（用预设 PDF 文件替代真实渲染）

**无独立产出 fixture**：版本管理操作 `versions` 表，导出直接从 HTML 生成 PDF。

**Go 代码示例**：

```go
// 模拟创建版本快照
func mockCreateVersion(draftID uint, htmlContent string) *Version {
    label := "AI 初始生成"
    return &Version{
        DraftID:      draftID,
        HTMLSnapshot: htmlContent,
        Label:        &label,
    }
}

// 模拟 PDF 导出（测试时返回固定文件）
func mockExportPDF(htmlContent string) ([]byte, error) {
    // 集成测试时使用真实 chromedp
    // 单元测试时返回固定 PDF
    if os.Getenv("USE_MOCK") != "false" {
        return os.ReadFile("fixtures/sample_export.pdf")
    }
    return chromedpExport(htmlContent)
}
```

## 5. 使用方式

### 开发阶段

```go
// 后端示例：用 fixture 替代上游模块
import (
    "os"
)

// USE_MOCK 默认 true，集成测试时设为 false
useMock := os.Getenv("USE_MOCK") != "false"

var draftHTML string
if useMock {
    draftHTML = string(mustReadFile("fixtures/sample_draft.html"))
} else {
    draftHTML = getDraftFromDB(draftID)
}
```

```typescript
// 前端示例：用 fixture 替代 API 响应
import sampleDraft from '../../fixtures/sample_draft.html?raw';

// 开发时不需要后端运行，直接用 mock
function EditorWithMock() {
  return <Editor initialHTML={sampleDraft} />;
}
```

### 集成测试阶段

当上下游都完成后，通过环境变量切换 mock 和真实服务：

```go
import "os"

useMock := os.Getenv("USE_MOCK") != "false" // 默认 true

var htmlContent string
if useMock {
    htmlContent = string(mustReadFile("fixtures/sample_draft.html"))
} else {
    // 调用真实的解析 + AI 生成接口
    resp, _ := http.Post("/api/v1/parsing/generate", "application/json", body)
    htmlContent = parseResponse(resp).HTMLContent
}
```

## 6. 与 v1 mock 的对比

| 维度 | v1 | v2 |
|---|---|---|
| fixture 文件数 | 6 个 JSON | 1 个 HTML + 1 个 JSON + 2 个测试文件 |
| fixture 总大小 | ~10KB JSON | ~8KB HTML + ~5KB JSON |
| 中间层 fixture | SourceAssetSet, EvidenceSet, ResumeDraftState, PatchEnvelope, ResolvedResumeSpec | 无 |
| 数据库 mock | 需要 ORM 模型对齐 JSON fixture | fixture 直接对应数据库表 |
| AI 响应 mock | PatchOp 列表（复杂结构） | 纯 HTML（简单直接） |

## 7. 维护规则

- `fixtures/` 目录下的文件是**共享契约 fixture**，修改需通知所有模块负责人
- 每个模块内部可以有自己的 `test_fixtures/`，自由修改不影响他人
- fixture 文件必须符合 `02-data-models/core-data-model.md` 中定义的数据库 schema
- `sample_draft.html` 必须是可直接在浏览器渲染的完整 HTML（含 `<style>`）
- `sample_ai_response.json` 必须包含 `text_explanation` 和 `modified_html` 两个字段
- 任何人发现 fixture 与 schema 不一致，应立即修复并通知
