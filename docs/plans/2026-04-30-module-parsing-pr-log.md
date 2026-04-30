# 模块 parsing — PR 开发日志（权限收口、契约补齐、坏数据兜底）

**类型：** 开发日志 / PR 说明  
**模块：** parsing（模块 B）  
**时间：** 2026-04-30  
**相关契约：** `docs/modules/parsing/contract.md`  
**相关代码：** `backend/internal/modules/parsing/`

---

## 1. 模块职责

模块 parsing 当前负责两段连续链路：

1. 解析项目资产  
   支持 `resume_pdf`、`resume_docx`、`note`、`git_repo` 四类资产，输出统一的 `ParsedContent` 列表。
2. 生成初始简历草稿  
   将解析出的文本聚合后送入 `DraftGenerator`，生成 HTML 初稿，写入 `drafts`、`versions`，并回填 `projects.current_draft_id`。

当前模块对外只暴露两个后端端点：

- `POST /api/v1/parsing/parse`
- `POST /api/v1/parsing/generate`

边界说明：

- `/parse` 只做解析预览，不创建 draft
- `/generate` 才会创建 draft / version，并更新 `current_draft_id`
- 文件上传、项目删除属于 intake
- draft 读取/编辑属于 workbench
- AI 对话修改属于 agent

---

## 2. 本次 PR 背景

这次 PR 不是新增一条大功能线，而是对模块 B 做收口和联调对齐，原因主要有三类：

1. 接口安全边界不够严  
   之前 `/parse`、`/generate` 只按 `project_id` 工作，没有像 intake 一样按 `user_id + project_id` 收紧。

2. 返回契约和联调代码不完全对齐  
   `dev` 分支新增的 intake → parse 预览代码依赖 `parsed_contents[].label`，但 parsing 之前没有返回这个字段。

3. 坏数据处理不够稳  
   空 note、缺 URI、坏 git 资产、只有空文本的项目，之前有的会直接拖垮整次 parse/generate，有的会落成 500，不利于联调排查。

---

## 3. 本次 PR 完成内容

### 3.1 按 A 模块口径补齐项目归属校验

本次把 parsing 的项目访问方式改成与 intake 一致：

- handler 从登录上下文读取 `user_id`
- service 在业务入口先校验 `user_id + project_id`
- 非 owner 访问时，对外统一表现为 `2003 project not found`

对应实现：

- `handler.go`：`Parse` / `Generate`
- `service.go`：`ParseForUser` / `GenerateForUser` / `ensureOwnedProject`

这样模块 B 现在不会再允许“已登录用户拿别人的 `project_id` 去解析或生成草稿”。

### 3.2 补齐 `ParsedContent.label` 返回契约

本次给 `/parse` 的结果补上了 `label` 字段，规则如下：

- 优先使用 `assets.label`
- 如果没有显式 label，则回退到文件名
- note 没有 label 时返回空字符串

这样前端在解析预览阶段就能稳定显示每条解析结果的标题，不需要自行猜测文件名或类型。

对应实现：

- `types.go`：`ParsedContent.Label`
- `types.go`：`AssetLabel(...)`
- `service.go`：`attachAssetMetadata(...)`

### 3.3 调整坏资产的容错策略

本次把 `Parse(projectID)` 的行为从“单个资产报错即整项目失败”改成了“尽量保留可用结果”。

现在遇到以下可恢复错误时：

- 资产缺 URI
- note 缺内容
- PDF / DOCX 解析失败
- git 提取失败
- 不支持的资产类型

如果同一项目里还有其他可用资产，parsing 会跳过坏资产，继续返回可用的 `parsed_contents`。  
只有在全部资产都不可用时，才返回错误。

这样做的目的不是掩盖错误，而是保证模块 B 更符合“聚合已有资料”的职责，不因为一条坏 note 或一个坏 git 资产就把整个项目拖死。

### 3.4 增加“无可生成文本”判定

`/generate` 现在在进入 DraftGenerator 之前，会先检查聚合后的文本是否为空。

如果 `parsed_contents` 虽然存在，但所有 `text` 都是空字符串或空白字符，则直接返回业务错误，而不是继续尝试生成。

新增错误：

- `ErrNoGeneratableText`

这样可以避免两种不一致：

- mock 模式直接返回 fixture，看起来“生成成功”
- 真实 AI 模式却因为没有文本输入而失败

### 3.5 新增 2006 错误码，避免坏数据落成 500

本次新增：

- `2006`：项目资产数据非法，或无可生成文本

覆盖场景：

- `ErrAssetURIMissing`
- `ErrAssetContentMissing`
- `ErrNoGeneratableText`

这样这些情况现在都会返回 `400`，而不是统一落成 `50000`。

### 3.6 统一包装 git 提取错误

本次为 git 资产新增了模块内错误包装：

- `ErrGitExtractFailed`

目的有两个：

1. 让 service 层能明确区分“git 提取失败”与“数据库/系统性错误”
2. 让 parse 的“可恢复错误跳过策略”能覆盖 git 资产

---

## 4. 当前模块 B 的对外行为

### 4.1 `POST /api/v1/parsing/parse`

输入：

```json
{
  "project_id": 1
}
```

输出：

```json
{
  "code": 0,
  "data": {
    "parsed_contents": [
      {
        "asset_id": 1,
        "type": "resume_pdf",
        "label": "sample_resume.pdf",
        "text": "..."
      }
    ]
  }
}
```

行为说明：

- 需要登录
- 只能操作自己的项目
- 返回的是解析预览，不创建 draft
- 遇到坏资产时，优先保留同项目内其他可用解析结果

### 4.2 `POST /api/v1/parsing/generate`

输入：

```json
{
  "project_id": 1
}
```

输出：

```json
{
  "code": 0,
  "data": {
    "draft_id": 1,
    "version_id": 1,
    "html_content": "<!DOCTYPE html><html>..."
  }
}
```

行为说明：

- 需要登录
- 只能操作自己的项目
- 内部链路为：`Parse -> aggregate text -> DraftGenerator -> drafts -> versions -> current_draft_id`
- 如果没有可生成文本，会返回 `2006`

### 4.3 当前错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 2001 | 400 | PDF 解析失败 |
| 2002 | 400 | DOCX 解析失败 |
| 2003 | 404 | 项目不存在或不属于当前用户 |
| 2004 | 400 | 项目无可用资产 |
| 2005 | 500 | AI 初稿生成失败 |
| 2006 | 400 | 项目资产数据非法，或无可生成文本 |

---

## 5. 与其他模块的联调口径

### 5.1 与 intake 的关系

parsing 依赖 intake 提供 `assets` 数据，但 parsing 不负责上传或存储文件。  
联调时只假设：

- project 已存在
- assets 已写入数据库
- 文件型资产的 `uri` 可读取
- note 资产的 `content` 非空

### 5.2 与 workbench 的关系

workbench 不负责“从无到有创建初稿”，它负责消费已有 draft。

因此联调口径应为：

1. intake 创建 project + assets
2. parsing 调 `/generate`
3. parsing 创建 draft / version 并写 `projects.current_draft_id`
4. workbench 通过 `current_draft_id` 进入编辑器

重点说明：

- `/parse` 只是解析预览，不会创建 draft
- 如果前端只调用 `/parse`，那它不能假设后续一定能进入编辑器
- 如果前端想从“解析结果预览”继续进入编辑器，应该显式调用 `/generate`

### 5.3 与 agent / render 的关系

- agent 消费的是已有 `drafts.html_content`
- render 消费的是已有 `versions` / `drafts.html_content`

因此 parsing 的职责是把初始 draft 和第一条 version 准备好，而不是负责后续编辑和导出。

---

## 6. 本次改动涉及的核心文件

后端实现：

- `backend/internal/modules/parsing/handler.go`
- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/types.go`
- `backend/internal/modules/parsing/generator.go`

测试：

- `backend/internal/modules/parsing/handler_test.go`
- `backend/internal/modules/parsing/service_test.go`
- `backend/internal/modules/parsing/generator_test.go`

文档：

- `docs/modules/parsing/contract.md`

---

## 7. 测试与验证

本次 PR 已验证：

- `go test ./internal/modules/parsing/...`
- `go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1`

重点回归覆盖：

- 非 owner 不能访问别人的项目
- `/parse` 返回 `label`
- 非法资产数据不再落成 500
- 有坏资产但也有好资产时，`Parse` 仍可成功
- 空聚合文本时，`Generate` 直接返回业务错误
- mock / 真实 AI 模式对空输入行为一致

说明：

- `go test ./...` 目前仍然不会全绿，剩余失败来自 workbench 现有测试依赖本地 PostgreSQL `localhost:5432`
- 这不是本次 parsing 改动引入的问题

---

## 8. 当前已知但不属于本模块 PR 范围的问题

以下问题与联调有关，但不属于本次模块 B PR 的直接修改范围：

- intake 上传文件时对 multipart 文件只做一次 `Read`
- intake 允许创建空 note
- workbench 的 draft 读取/更新仍未按用户归属限权
- workbench 目前没有 `POST /api/v1/drafts`
- 删除 project 时，draft / version 的级联清理尚未统一

这些问题需要 A / D 模块分别收口，不应与本次 parsing PR 混为一谈。

---

## 9. PR Reviewer 快速结论

如果 reviewer 只关心“模块 B 这次 PR 到底做了什么”，可以直接看下面这 5 条：

1. parsing 现在和 intake 一样，按 `user_id + project_id` 做项目归属校验。
2. `/parse` 现在返回 `parsed_contents[].label`，前端预览契约补齐。
3. 坏资产不会轻易拖垮整项目，能保留的解析结果会尽量保留。
4. `/generate` 对“没有可用文本”的项目会稳定返回业务错误，不再 mock/真实模式表现不一致。
5. 非法资产数据现在走 `2006 / 400`，不再默认回 `50000`。
