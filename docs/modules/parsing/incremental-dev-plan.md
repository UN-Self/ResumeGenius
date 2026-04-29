# 模块 B 增量开发拆分计划

更新时间：2026-04-28

> 这是一份给实际开发用的“小步快跑”拆分文档。
> 它不替代 [`contract.md`](./contract.md) 和 [`work-breakdown.md`](./work-breakdown.md)，而是把模块 B 拆成适合逐个实现、逐个上传的开发切片。

## 0. 当前仓库状态校准（2026-04-29）

这部分用于把本拆分计划和当前仓库实际状态对齐，避免按旧顺序从头重复判断。

### 0.1 当前完成度

- `B0-B10`：已完成。`fixtures/`、PDF/DOCX parser、note 支持、`ParsingService` 聚合解析、`/api/v1/parsing/parse`、`DraftGenerator` 的 mock/real 双模式、`/api/v1/parsing/generate`、`drafts` 写入、`projects.current_draft_id` 更新，以及自动创建 `versions` 快照和返回 `version_id` 都已经落地。
- `B11-B12`：当前代码库中仍未落地，`GitExtractor`、PDF 图片提取都还需要继续开发。

### 0.2 已有代码里可直接复用的部分

- 模块 B 可以直接复用 `backend/internal/modules/parsing/service.go` 中现有的解析分发和聚合逻辑，不需要重写 `Parse(projectID)` 主流程。
- 版本快照创建可以先参考 `backend/internal/modules/workbench/service.go` 里已经跑通的 `versions` 写库逻辑。
- `Project.current_draft_id`、`Draft`、`Version` 等模型已经在 `backend/internal/shared/models/models.go` 中齐备，B8/B9 不需要再补数据模型。
- 前端编辑器已经按 `project.current_draft_id -> GET /drafts/:draft_id` 这条链路工作，所以只要 B8/B9 打通，工作台就能消费 parsing 生成的初稿。

### 0.3 目前还不能依赖的部分

- `backend/internal/modules/agent` 目前仍是 stub，不能复用真实 AI 调用封装。
- `backend/internal/modules/render` 目前仍是 stub，不能复用独立的版本服务或导出服务。
- 因此 B9 在现阶段应先直接写 `versions` 表，后续等 render 模块成熟后再抽共享逻辑。

### 0.4 按当前仓库状态的建议顺序

如果从现在继续往下做，建议直接从 B11 开始推进主链路：

1. 做 `B11`：补 `GitExtractor`，把 `git_repo` 纳入主链路。
2. 做 `B12`：补 PDF 图片提取和增强测试。

### 0.5 当前最推荐的一批实际开发任务

如果希望下一批提交尽量小、同时又能明显推进主链路，建议下一批只做一件事：

1. `B11`：补 `GitExtractor`，把 `git_repo` 纳入主链路，但这一批先不补 PDF 图片提取。

## 1. 使用原则

- 一次只做一个切片，不同时推进 PDF、DOCX、AI、Git 多条线
- 每个切片都必须有明确的“停点”和“验收方式”
- 每次上传尽量控制在 3-6 个文件以内
- 先做“可跑通主链路”，再做“增强项”
- 以 [`contract.md`](./contract.md) 为最终范围，以本拆分文档控制实现顺序

## 2. 当前建议范围

模块 B 的完整职责包括：

- PDF 解析
- DOCX 解析
- note 补充文本处理
- Git 仓库信息抽取
- AI 初稿生成
- 写入 drafts
- 更新 `projects.current_draft_id`
- 自动创建 versions 快照

但为了避免一次改动过大，建议把功能拆成两层：

### 2.1 第一层：先跑通主链路

先做这条最小闭环：

`resume_pdf / resume_docx / note -> parse -> mock AI generate -> drafts/version`

### 2.2 第二层：后补增强项

后续再补：

- Git 抽取
- PDF 图片提取
- 真实 AI ProviderAdapter
- 更细的错误处理和质量优化

## 3. 切片总览

| 切片 | 目标 | 建议独立上传 |
|---|---|---|
| B0 | 建立开发边界和测试素材目录 | 是 |
| B1 | 定义解析领域模型和分发骨架 | 是 |
| B2 | 只实现 PDF 文本提取 | 是 |
| B3 | 只实现 DOCX 文本提取 | 是 |
| B4 | 支持 note 补充文本 | 是 |
| B5 | 实现 ParsingService 聚合解析 | 是 |
| B6 | 暴露 `/parsing/parse` 接口 | 是 |
| B7 | 实现 mock DraftGenerator | 是 |
| B8 | 暴露 `/parsing/generate` 接口并写 drafts | 是 |
| B9 | 接入版本快照创建 | 是 |
| B10 | 接入真实 AI 调用 | 是 |
| B11 | 实现 GitExtractor | 是 |
| B12 | 补 PDF 图片提取与增强测试 | 是 |

## 4. 逐步拆解

### B0. 建立开发边界和测试素材目录

**目标**

先把模块 B 开发所需的本地基础补齐，避免后面边开发边补环境。

**建议内容**

- 创建 `fixtures/`
- 放入：
  - `fixtures/sample_resume.pdf`
  - `fixtures/sample_resume.docx`
  - `fixtures/sample_draft.html`
- 在模块 B 文档里约定：开发阶段优先走 `USE_MOCK=true`

**建议涉及文件**

- `fixtures/sample_resume.pdf`
- `fixtures/sample_resume.docx`
- `fixtures/sample_draft.html`
- 如有需要，补充 `README` 或本模块说明

**完成标志**

- fixtures 目录存在
- 后续测试不再依赖模块 A 的真实上传流程

**这一块不要顺手做的事**

- 不要在这一步实现任何 parser
- 不要开始写 handler

---

### B1. 定义解析领域模型和分发骨架

**目标**

先把“模块 B 内部如何表示解析结果”定下来，再往里塞 PDF/DOCX 解析实现。

**建议内容**

- 定义解析结果结构体，例如：
  - `ParsedContent`
  - `ParsedImage`
- 定义 parser 接口或分发约定，例如：
  - `ParsePDF(path string) (...)`
  - `ParseDOCX(path string) (...)`
  - `ParseNote(content string) (...)`
- 定义 `ParsingService` 的最小骨架，但先不接数据库写 draft

**建议涉及文件**

- `backend/internal/modules/parsing/types.go`
- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/service_test.go`

**完成标志**

- 模块 B 内部已有统一结果结构
- 后续新增解析器时不需要反复改 handler 返回格式

**这一块不要顺手做的事**

- 不要在这一步加入 AI 生成逻辑
- 不要直接写 HTTP handler

---

### B2. 只实现 PDF 文本提取

**目标**

把 PDF 文本提取单独做完并测稳，不夹带别的功能。

**建议内容**

- 引入 `github.com/ledongthuc/pdf`
- 实现 PDF 文本提取函数
- 只关注“按页顺序拿到可用文本”
- v1 可以先不做图片提取

**建议涉及文件**

- `backend/internal/modules/parsing/pdf_parser.go`
- `backend/internal/modules/parsing/pdf_parser_test.go`

**完成标志**

- 能从 `sample_resume.pdf` 提取非空文本
- 损坏 PDF 能返回模块 B 错误，而不是 panic

**验收重点**

- 文本顺序基本正确
- 空 PDF、损坏 PDF、找不到文件都能稳定返回错误

**这一块不要顺手做的事**

- 不要同时做 DOCX
- 不要在这一步接 `/parse` 接口

---

### B3. 只实现 DOCX 文本提取

**目标**

把 DOCX 提取和 PDF 分开，避免一次引两个解析器导致排查困难。

**建议内容**

- 引入 `github.com/nguyenthenguyen/docx`
- 实现 DOCX 文本提取
- v1 先只做段落顺序拼接
- 表格和样式可先转成普通文本，不必一次做复杂还原

**建议涉及文件**

- `backend/internal/modules/parsing/docx_parser.go`
- `backend/internal/modules/parsing/docx_parser_test.go`

**完成标志**

- 能从 `sample_resume.docx` 提取非空文本
- 损坏 DOCX 返回 `2002`

**这一块不要顺手做的事**

- 不要在这一步引入 AI prompt

---

### B4. 支持 note 补充文本

**目标**

把最简单、最稳定的一类输入先纳入主链路，后面聚合解析时更容易验证。

**建议内容**

- 支持 `asset.type = note`
- 直接读取 `assets.content`
- 如果有 `label`，可拼进输出文本头部

**建议涉及文件**

- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/service_test.go`

**完成标志**

- 给定 note 资产时，不依赖文件系统也能产出解析结果

---

### B5. 实现 ParsingService 聚合解析

**目标**

把 PDF、DOCX、note 串起来，但只做到“解析并聚合结果”，先不生成 draft。

**建议内容**

- 从 `assets` 表按 `project_id` 拉资产
- 根据 `asset.type` 分发：
  - `resume_pdf`
  - `resume_docx`
  - `note`
- 汇总成 `parsed_contents`
- 定义错误行为：
  - 项目不存在：`2003`
  - 项目无可用资产：`2004`
  - 单文件损坏：按文件类型给模块错误

**建议涉及文件**

- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/service_test.go`

**完成标志**

- `ParsingService.Parse(projectID)` 可直接给 handler 使用

**这一块不要顺手做的事**

- 不要写入 drafts
- 不要调用 AI

---

### B6. 暴露 `/api/v1/parsing/parse` 接口

**目标**

先把“解析结果可通过 API 返回”单独闭环。

**建议内容**

- 实现 handler
- 请求体使用：

```json
{
  "project_id": 1
}
```

- 返回格式严格对齐 [`contract.md`](./contract.md)
- 修正当前 stub 路由

**建议涉及文件**

- `backend/internal/modules/parsing/handler.go`
- `backend/internal/modules/parsing/handler_test.go`
- `backend/internal/modules/parsing/routes.go`

**完成标志**

- `POST /api/v1/parsing/parse` 可返回 `parsed_contents`
- 参数错误走 `40000`
- 成功响应走统一 `{code,data,message}`

**这一块不要顺手做的事**

- 不要顺手把 `/generate` 也一起做了

---

### B7. 实现 mock DraftGenerator

**目标**

先把“生成 HTML 初稿”这件事在 mock 模式下跑通，不急着接真实模型。

**建议内容**

- 新建 `DraftGenerator`
- `USE_MOCK=true` 时读取固定 HTML 或内置 mock HTML
- 先只接收“聚合后的文本”并返回完整 HTML
- 不急着引入复杂 prompt 拼装器

**建议涉及文件**

- `backend/internal/modules/parsing/generator.go`
- `backend/internal/modules/parsing/generator_test.go`

**完成标志**

- 给一段解析文本，能返回完整 HTML
- 返回内容包含 `<html` 和基础样式骨架

---

### B8. 暴露 `/api/v1/parsing/generate` 接口并写 drafts

**目标**

把“解析 -> mock 生成 -> drafts”串成最小可用主链路。

**建议内容**

- 实现 `POST /api/v1/parsing/generate`
- 流程：
  - 根据 `project_id` 找资产
  - 调用 `ParsingService.Parse`
  - 聚合文本
  - 调用 `DraftGenerator`
  - 写入 `drafts`
  - 更新 `projects.current_draft_id`
- 成功后返回：
  - `draft_id`
  - `html_content`

**建议涉及文件**

- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/generator.go`
- `backend/internal/modules/parsing/handler.go`
- `backend/internal/modules/parsing/handler_test.go`

**完成标志**

- mock 模式下可以创建 draft
- 失败时不创建脏 draft

**这一块不要顺手做的事**

- 先不要接真实 AI
- 先不要做 Git 解析

---

### B9. 接入版本快照创建

**目标**

把模块 B 的输出和模块 E 契约对上，但先只做最小版本创建逻辑。

**建议内容**

- 初稿生成成功后自动创建版本
- label 固定为 `"AI 初始生成"`
- 返回 `version_id`

**建议涉及文件**

- `backend/internal/modules/parsing/generator.go`
- 如有需要，可提取共享版本创建 helper
- `backend/internal/modules/parsing/handler_test.go`

**完成标志**

- 生成 draft 后，`versions` 表自动新增一条快照
- `/generate` 响应中带 `version_id`

**注意**

- 如果模块 E 还没实现独立服务层，B 可以先直接写 `versions` 表
- 等模块 E 稳定后，再收敛成共享的版本创建逻辑。

---

### B10. 接入真实 AI 调用

**目标**

把 mock 生成替换成真实模型调用，但不和前面的 parse/generate 主链路开发混在一起。

**建议内容**

- 读取：
  - `AI_API_URL`
  - `AI_API_KEY`
- 组装 prompt：
  - 系统角色
  - HTML 模板骨架
  - 聚合后的用户资料
  - 输出约束
- 解析 OpenAI-compatible 响应

**建议涉及文件**

- `backend/internal/modules/parsing/generator.go`
- `backend/internal/modules/parsing/generator_test.go`
- 如有需要，提取 `provider_adapter.go`

**完成标志**

- `USE_MOCK=false` 时可走真实 API
- 模型失败能稳定返回 `2005`

---

### B11. 实现 GitExtractor

**目标**

把 Git 解析放到主链路稳定之后再做，避免它拖慢 PDF/DOCX 主流程。

**建议内容**

- 识别 `asset.type = git_repo`
- 从仓库抽取：
  - 项目名
  - README
  - 技术栈关键词
  - 目录结构摘要
- v1 不追求深度源码理解

**建议涉及文件**

- `backend/internal/modules/parsing/git_extractor.go`
- `backend/internal/modules/parsing/git_extractor_test.go`
- `backend/internal/modules/parsing/service.go`

**完成标志**

- Git 资产可转成一段可供 AI 使用的文本摘要

---

### B12. 补 PDF 图片提取与增强测试

**目标**

把契约里提到但 MVP 不是最优先的能力放到最后做。

**建议内容**

- 从 PDF 中提取内嵌图片
- 以 `base64` 形式挂到解析结果
- 补更真实的 fixture 和边界测试

**建议涉及文件**

- `backend/internal/modules/parsing/pdf_parser.go`
- `backend/internal/modules/parsing/pdf_parser_test.go`

**完成标志**

- `parsed_contents.images` 有稳定输出结构
- 没有图片的 PDF 不报错

## 5. 推荐开发顺序

如果你想严格控制每次改动大小，建议按下面顺序推进：

1. B0
2. B1
3. B2
4. B3
5. B4
6. B5
7. B6
8. B7
9. B8
10. B9
11. B10
12. B11
13. B12

其中最核心的第一阶段停点是：

`B0 -> B6`

做完后，你就已经拥有“可通过接口返回解析结果”的模块 B 半成品。

第二阶段停点是：

`B7 -> B9`

做完后，你就拥有“可生成初稿并写入 drafts/version”的 MVP 主链路。

## 6. 每次上传前自检

每完成一个切片，至少检查这几件事：

- 只改了当前切片需要的文件
- 当前切片的测试已补上
- 没有顺手引入下一个切片的半成品代码
- 接口返回格式仍符合统一响应约定
- 错误码仍在 `2001-2999` 范围

## 7. 不建议一次混做的组合

- 不要把 `PDF + DOCX + Git` 一次一起做
- 不要把 `parse + generate + real AI` 一次一起做
- 不要把 `draft 写库 + version 快照 + Git 抽取` 一次一起做
- 不要在同一批里同时改大量文档、模型、handler、前端页面

## 8. 建议的第一批实际开发任务

如果你现在马上开工，我建议第一批只做这三件事：

1. B0：补齐 `fixtures/`
2. B1：定义 `ParsedContent` 和 `ParsingService` 骨架
3. B2：只把 PDF 文本提取做完

这样第一批上传会很小，失败面也最可控。
