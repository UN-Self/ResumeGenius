# PDFTotext 替换 PDF 文本提取

## 背景

`ledongthuc/pdf` 的 `GetPlainText()` 无法正确处理使用 CID 字体（Identity-H 编码）的中文 PDF。
该库读取原始 glyph ID 而非经过 ToUnicode CMap 映射的 Unicode 字符，导致中文简历上传后提取出乱码文本。

## 方案

用系统 `pdftotext`（poppler-utils）替换文本提取，保留 `ledongthuc/pdf` 用于图片提取。

## 改动范围

### 1. `backend/internal/modules/parsing/pdf_parser.go`

`ExtractTextFromPDF` 改为调用 `exec.Command("pdftotext", "-layout", "-enc", "UTF-8", path, "-")`。

参数：
- `-layout`：保持排版布局
- `-enc UTF-8`：强制 UTF-8
- `-`：输出到 stdout

`ExtractImagesFromPDF` 及所有图片处理函数不变。

### 2. `backend/Dockerfile`

runtime stage `apk add` 加 `poppler-utils`。

### 3. `backend/internal/modules/parsing/pdf_parser_test.go`

现有 `TestExtractTextFromPDFFixture` 自动覆盖中文提取验证（断言 "张三"、"工作经历"）。
补充 `pdftotext` 可用性检查。

## 错误处理

- `pdftotext` 不存在 → `exec.Error` 返回明确错误信息
- 非零退出码 → 包含 stderr
- 空输出 → 正常返回空字符串
