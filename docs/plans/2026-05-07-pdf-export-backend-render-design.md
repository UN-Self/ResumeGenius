# PDF 导出改造：后端直接从数据库取 HTML 渲染

## 背景

当前 PDF 导出流程是前端克隆 DOM → 粗暴转内联样式 → 传 HTML 给后端 → Chrome 渲染。这个过程中样式信息大量丢失，导致"所见不为所得"。

核心问题：`captureExportHTML()` 把 TipTap 编辑器的 CSS class 全扒掉，只塞几个写死的内联样式，排版基本全丢。

## 目标

后端直接从数据库取 `draft.html_content`，用模板包裹后交给 Chrome 渲染。导出结果和前端编辑器看到的一模一样，唯一区别是不渲染水印。

## 方案

**方案 B：后端持有 CSS 副本**。后端维护一份和前端 `editor.css` 一致的渲染模板，从 DB 取出 HTML 后用模板包裹再渲染。

理由：改动最小，不涉及数据库 schema 变更、AI 生成逻辑变更或前端保存逻辑变更。CSS 基本不变，后端放一份副本即可。

## 改动详情

### 1. 后端：新增渲染模板

文件：`backend/internal/modules/render/render-template.html`

用 `//go:embed` 嵌入到 Go 二进制中。模板内容：

```html
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
/* Inter 字体 @font-face（由 injectFontCSS 注入） */

/* 从 editor.css 提取的排版样式（不含 .ProseMirror 前缀，改为 .resume-page 选择器） */
.resume-page {
  width: 210mm;
  min-height: 297mm;
  padding: 18mm 20mm;
  background: #ffffff;
  color: #333333;
  font-family: "Inter", "Noto Sans SC", -apple-system, BlinkMacSystemFont, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  white-space: pre-wrap;
}

.resume-page h1 { font-size: 24px; font-weight: 600; line-height: 1.3; }
.resume-page h2 { font-size: 20px; font-weight: 600; line-height: 1.3; }
.resume-page h3 { font-size: 16px; font-weight: 500; line-height: 1.4; }
.resume-page p  { font-size: 14px; font-weight: 400; line-height: 1.5; }
.resume-page ul, .resume-page ol { padding-left: 24px; }
.resume-page ul { list-style-type: disc; }
.resume-page ol { list-style-type: decimal; }
.resume-page li { margin-bottom: 4px; }
.resume-page li p { display: inline; }

@page { size: A4; margin: 0; }
</style>
</head>
<body style="margin:0;padding:0;">
<div class="resume-page">{{CONTENT}}</div>
</body>
</html>
```

样式来源：`frontend/workbench/src/styles/editor.css` 中的 `.ProseMirror` 规则（第 139-191 行），选择器改为 `.resume-page`。A4 容器尺寸和内边距来自 `A4Canvas.tsx` 的 `resume-document` div（宽 210mm、min-height 297mm、padding 18mm 20mm）。

### 2. 后端：改造 ExportService

文件：`backend/internal/modules/render/exporter.go`

改动点：
- `CreateTask` 签名从 `CreateTask(draftID uint, htmlContent string)` 改为 `CreateTask(draftID uint)`
- 方法内部从 DB 查 `draft.html_content`（已有 DB 验证逻辑，扩展为读取内容）
- `ExportTask.htmlContent` 字段由 DB 查询结果填充，而非从前端传入
- 新增 `wrapWithTemplate(htmlFragment string) string` 方法，将 HTML 片段嵌入渲染模板

### 3. 后端：改造 Handler

文件：`backend/internal/modules/render/handler.go`

改动点：
- `CreateExport` 不再绑定 `createExportReq`（删掉 `html_content` 字段解析）
- 调用 `exportSvc.CreateTask(draftID)` 不再传 HTML

### 4. 前端：简化导出

文件改动：

**删除** `frontend/workbench/src/lib/export-capture.ts`（整个文件）

**改造** `frontend/workbench/src/hooks/useExport.ts`：
- `exportPdf` 签名从 `(draftId, htmlContent, filename?)` 改为 `(draftId, filename?)`
- POST body 不再包含 `html_content`，改为 `{}` 或空 body

**改造** `frontend/workbench/src/pages/EditorPage.tsx`：
- `handleExport` 从 `exportPdf(Number(draftId), captureExportHTML())` 改为 `exportPdf(Number(draftId))`
- 删除 `import { captureExportHTML }` 引用

### 5. 不变的部分

- 异步任务队列（CreateTask → worker → processTask → 轮询 → 下载）
- chromedp 渲染引擎
- Inter 字体注入（`injectFontCSS` 逻辑保留）
- `ChromeExporter.ExportHTMLToPDF` 签名不变（仍是接收完整 HTML 文档字符串）
- 前端轮询和下载逻辑
- PDF 存储和文件下载接口

### 6. 水印处理

后端渲染模板中没有 WatermarkOverlay 组件，导出的 PDF 天然无水印。无需额外处理。

## 数据流（改造后）

```
用户点"导出"
    |
    v
EditorPage.handleExport()
    |-- exportPdf(draftId)  // 只传 draft_id
    |
    v
POST /api/v1/drafts/{draftId}/export  (body: {})
    |
    v
Handler.CreateExport
    |-- exportSvc.CreateTask(draftID)  // 不传 HTML
    |   |-- 从 DB 查 draft.html_content
    |   |-- wrapWithTemplate(htmlContent)  // 包裹渲染模板
    |   |-- 创建 ExportTask，htmlContent = 模板包裹后的完整文档
    |   |-- 入队
    |
    v
Worker 处理
    |-- ChromeExporter.ExportHTMLToPDF(wrappedHTML)
    |   |-- injectFontCSS(wrappedHTML)  // 注入 Inter 字体
    |   |-- Chrome 渲染 → PDF bytes
    |-- 存储 PDF，返回下载 URL
    |
    v
前端轮询 → 下载 PDF
```

## 测试计划

- 单元测试：`wrapWithTemplate` 正确替换 `{{CONTENT}}` 占位符
- 单元测试：`CreateTask` 从 DB 读取 HTML 并包裹模板
- 集成测试：端到端导出流程（需 chromedp 环境）
- 手动验证：对比编辑器显示和导出 PDF 的排版一致性
