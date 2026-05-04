# Parsing / Intake / Workbench — PR 开发日志（素材正文清洗、入库沉淀与左栏资产化）

**类型：** 开发日志 / PR 说明  
**模块：** parsing / intake / workbench  
**时间：** 2026-05-04  
**相关契约：** `docs/modules/parsing/contract.md`、`docs/modules/intake/contract.md`、`docs/modules/agent/contract.md`  
**相关计划：** `docs/plans/2026-05-02-parsing-cleanup-and-asset-sidebar-plan.md`

---

## 1. 背景

这轮改动的目标，不是单纯把 parsing 做成“解析一次然后返回预览”，而是把素材链路收口成一条可持续复用的正文数据链：

1. 用户上传 PDF / DOCX / Git / note
2. parsing 对文本做轻清洗，产出更适合 AI 消费的素材正文
3. 清洗后的正文写回 `assets.content`
4. 解析出的图片持久化为派生 `resume_image` 资产
5. 在正文和派生图片都已落库后，删除原始 PDF / DOCX
6. workbench 左栏直接基于 `assets` 做素材管理
7. generate 优先消费已持久化正文，而不是每次重跑整条解析链

这条链路的直接收益是：

- AI 可读素材从“临时解析结果”变成“稳定持久化正文”
- 前端左栏不再依赖一次性 `parsed_contents`
- 后续 agent / tool 如需读取项目素材，可以统一从 `assets.content` 取数

---

## 2. 本次完成内容

### 2.1 抽出共享文本清洗器

对 PDF / DOCX / Git / note 的解析文本统一做轻清洗，主要包括：

- 统一换行符
- 压缩多余空白和重复空行
- 去掉简单分隔线和页码噪声
- 规范 bullet 表达

目标不是做重型摘要，而是让正文“更整洁、更适合 AI 获取”。

### 2.2 将清洗后正文回写到 `assets.content`

当前行为已统一为：

- `note`：`content` 继续直接保存用户原文
- `resume_pdf / resume_docx / git_repo`：由 parsing 成功后回写 `assets.content`

同时会在 `metadata.parsing` 中留下：

- `status`
- `cleaned`
- `content_persisted`
- `updated_by_user`
- `last_parsed_at`

### 2.3 持久化解析图片并建立主从关系

PDF / DOCX 解析出的图片不再只存在于一次性响应里，而是会落成派生 `resume_image` 资产。

主资产的 `metadata.parsing` 中会记录：

- `derived_image_asset_ids`
- `images_persisted`
- `image_count`

重复解析同一素材时，会替换旧的派生图资产和旧文件，避免无限堆积。

### 2.4 在安全前提下删除原始 PDF / DOCX

当正文和派生图片都已持久化成功后：

- 原始 PDF / DOCX 文件会从存储中删除
- 主资产 `uri` 会被清空
- `metadata.parsing` 中会保留：
  - `source_deleted`
  - `original_uri`
  - `original_filename`
  - `original_asset_type`

后续再次 parse / generate 时，会优先从 `assets.content` 回退构造内容，不再依赖已删源文件。

### 2.5 生成链路优先消费持久化正文

`/api/v1/parsing/generate` 当前行为：

- 优先读取 `assets.content`
- 正文缺失时才回退触发解析
- 回退解析成功后会补持久化，再继续生成草稿

这样已经 parse 过的项目，在重复 generate 时不会每次都重新读原始文件。

### 2.6 素材正文支持通用编辑

intake 新增了通用接口：

- `PATCH /api/v1/assets/{asset_id}`

现在 `note / resume_pdf / resume_docx / git_repo` 都可以统一编辑：

- `content`
- `label`

同时：

- 仅改标题不会误标为手动改正文
- 改正文时会写 `metadata.parsing.updated_by_user = true`

### 2.7 编辑页左栏升级为资产管理区

workbench 编辑页左栏已从 `parsed_contents` 切到 `assets` 驱动，支持：

- 上传文件
- 接入 Git
- 添加 note
- 删除素材
- 编辑正文 / 标题
- 对 PDF / DOCX / Git 触发重新解析

左栏会过滤掉 parsing 派生的 `resume_image`，避免把它们当成普通正文素材误编辑。

### 2.8 增加“重新解析”与覆盖提示

当用户手动修改过 PDF / DOCX / Git 的正文后：

- `metadata.parsing.updated_by_user = true`
- 左栏会显示“已手动修改，重新解析将覆盖当前正文”
- 重新解析前会弹出确认提示
- parse 成功后会重置 `updated_by_user = false` 并刷新 `last_parsed_at`

---

## 3. 当前对外口径

### parsing

- `/parse`：返回预览，同时回写正文、派生图片和解析状态
- `/generate`：优先消费持久化正文，生成 draft / version

### intake

- `assets.content` 已经是素材正文统一来源
- `PATCH /assets/{asset_id}` 是通用素材正文编辑接口

### workbench

- 编辑页左栏直接展示和管理 `assets`
- 用户刷新页面后，素材正文不会因为丢失 `parsed_contents` 而消失

### agent

- 当前主对话链路仍以 `drafts.html_content` 为主上下文
- 如后续接入项目素材读取能力，应统一消费 `assets.content`

---

## 4. 测试与验证

本轮已完成以下回归：

- `go test ./internal/modules/intake/...`
- `go test ./internal/modules/parsing/...`
- `npm exec vitest run tests/EditorPage.test.tsx`
- `node_modules\\.bin\\tsc.exe --noEmit --pretty false -p tsconfig.app.json`

覆盖重点包括：

- parsing 成功后正文回写
- 派生图片落库与重复解析替换
- 原始 PDF / DOCX 删除后的回退读取
- generate 优先消费已持久化正文
- 通用资产正文编辑
- 编辑页左栏素材管理与重新解析提示

---

## 5. 当前残留点

这轮链路已经闭合，但还有一个值得后续继续补的点：

- 当前删除主素材资产时，后端删除接口还没有级联清理 parsing 派生的 `resume_image` 资产

这不影响本轮“正文入库 + 左栏资产化 + 重新解析提示”的主目标，但后面最好单独补掉。

---

## 6. PR 标题建议

建议标题：

`feat: 收口 parsing 素材正文链路并升级 workbench 左栏资产管理`

如果想更偏工程表达，也可以用：

`feat: persist cleaned asset content and move editor sidebar to asset-driven flow`

---

## 7. PR 摘要建议

```md
## Summary

这次 PR 主要收口了“素材上传 -> parsing 清洗 -> 正文入库 -> workbench 展示 -> generate/AI 复用”这条链路。

## 本次改动

- 抽出 PDF / DOCX / Git / note 的共享文本清洗器
- 将清洗后正文回写到 `assets.content`
- 将解析图片持久化为派生 `resume_image` 资产
- 在正文和图片都持久化成功后删除原始 PDF / DOCX，并保留来源 metadata
- 让 `/parsing/generate` 优先消费已持久化正文
- 新增 `PATCH /api/v1/assets/{asset_id}`，支持通用素材正文编辑
- 将编辑页左栏升级为 `assets` 驱动的素材管理区
- 增加“重新解析”入口、手动修改脏状态提示和覆盖确认

## 验证

- `go test ./internal/modules/intake/...`
- `go test ./internal/modules/parsing/...`
- `npm exec vitest run tests/EditorPage.test.tsx`
- `node_modules\\.bin\\tsc.exe --noEmit --pretty false -p tsconfig.app.json`
```
