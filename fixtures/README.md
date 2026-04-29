# Fixtures

这些文件用于本地开发和模块级测试，优先服务于“能独立开发、能稳定复现”的目标。

## 文件说明

- `sample_resume.pdf`
  - 用途：模块 B 的 PDF 解析测试
  - 来源：基于 `docs/02-data-models/mock-fixtures.md` 中 `sample_draft.html` 对齐后本地生成
- `sample_resume.docx`
  - 用途：模块 B 的 DOCX 解析测试
  - 来源：基于 `docs/02-data-models/mock-fixtures.md` 中 `sample_draft.html` 对齐后本地生成
- `sample_draft.html`
  - 用途：模拟模块 B 产出的 HTML 初稿，供 C/D/E 模块消费
  - 来源：`docs/02-data-models/mock-fixtures.md`
- `sample_ai_response.json`
  - 用途：模拟 AI 对话或改写后的结构化响应，供后续 agent 模块和联调用例复用
  - 来源：`docs/02-data-models/mock-fixtures.md`

## 备注

- `sample_draft.html` 是契约型 fixture，尽量与文档保持一致。
- `sample_ai_response.json` 目前主要用于共享 mock 契约，不直接参与 parsing 的生成接口返回。
- `sample_resume.pdf` 和 `sample_resume.docx` 当前使用与 `sample_draft.html` 基本一致的中文资料，优先服务解析测试和联调。
- 如果后续你有更真实的 PDF/DOCX 简历样本，可以直接替换，或新增为独立 fixture。
