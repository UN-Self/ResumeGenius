# v2 文档一致性修复设计

日期：2026-04-24
状态：已执行
审查补充：2026-04-24（新增 Section 9-11）

## 1. 背景与目标

当前 v2 文档体系存在多处契约冲突，主要集中在：

1. 回退接口语义不一致（有文档写为仅返回，有文档写为写回）。
2. PDF 导出模式不一致（有文档写异步任务，有文档写同步二进制）。
3. 路径参数命名不一致（有文档要求语义化 ID，有文档示例使用通用 `{id}`）。
4. 错误码写法存在前导零导致的 JSON 非法示例问题。

本设计目标：

1. 在不改代码实现的前提下，统一文档口径。
2. 明确唯一语义，降低后续联调和开发歧义。
3. 采用最小变更策略，优先修复高影响冲突。

## 2. 已确认决策

### 2.1 回退接口语义（已确认）

统一为：

- 回退操作将指定版本的 `html_snapshot` 写回 `drafts.html_content`。
- 回退成功后自动创建一条新的 `versions` 快照。

接口语义：`POST /api/v1/drafts/{draft_id}/rollback` 为写操作，不是只读查询。

### 2.2 导出接口模式（已确认）

统一为：

- PDF 导出采用异步任务模式。
- `POST /api/v1/drafts/{draft_id}/export` 负责创建任务并返回 `task_id`。
- `GET /api/v1/tasks/{task_id}` 查询任务状态，完成后提供下载信息。

### 2.3 路径参数命名（已确认）

统一为语义化参数名，不使用泛化 `{id}`：

- `project_id`
- `draft_id`
- `session_id`
- `version_id`
- `task_id`

### 2.4 错误码类型（已确认）

统一为纯数字错误码：

- `code` 字段类型为 number/integer。
- 不使用带前导零表示（如 `02005`）。
- 取消 `SSCCC` 的前导零语义表达，改为纯数字分段编码。

## 3. 统一规范

### 3.1 回退接口事务语义

`POST /api/v1/drafts/{draft_id}/rollback`

建议流程（事务内）：

1. 校验 `draft_id` 与 `version_id` 存在。
2. 校验 `version_id` 归属该 `draft_id`。
3. 读取目标 `html_snapshot`。
4. 更新 `drafts.html_content`。
5. 自动创建新 `versions` 记录（回退后的当前状态）。

返回建议：

```json
{
  "code": 0,
  "data": {
    "draft_id": 1,
    "updated_at": "2026-04-24T12:00:00Z",
    "new_version_id": 10,
    "new_version_label": "回退到版本 3",
    "new_version_created_at": "2026-04-24T12:00:00Z"
  },
  "message": "ok"
}
```

### 3.2 导出异步任务语义

`POST /api/v1/drafts/{draft_id}/export`

返回示例：

```json
{
  "code": 0,
  "data": {
    "task_id": "task_abc123",
    "status": "pending"
  },
  "message": "ok"
}
```

`GET /api/v1/tasks/{task_id}`

完成示例：

```json
{
  "code": 0,
  "data": {
    "task_id": "task_abc123",
    "status": "completed",
    "progress": 100,
    "result": {
      "download_url": "/api/v1/tasks/task_abc123/file"
    }
  },
  "message": "ok"
}
```

### 3.3 错误码数字分段规则

保留模块分段思想，去除前导零语义：

- 通用：40000、40001、40100、40300、40400、40900、50000
- A 模块：1001 起
- B 模块：2001 起
- C 模块：3001 起
- D 模块：4001 起
- E 模块：5001 起

说明：文档与示例全部采用数字表示，不再出现 `02005` 等写法。

## 4. 文档修复范围

### 4.1 高优先级修复

1. `docs/modules/e-render/contract.md`
- 回退语义改为写回并新增快照。
- 导出改为异步任务语义（不再同步返回 PDF binary）。

2. `docs/modules/e-render/work-breakdown.md`
- 回退技术要点改为写操作。
- 导出流程与测试改为异步任务。

3. `docs/superpowers/specs/2026-04-23-architecture-v2-design.md`
- 导出接口总览改为异步任务口径。
- SQL DDL 调整为可执行顺序（避免先引用未创建表）。

4. `docs/01-product/api-conventions.md`
- 错误码章节改为数字分段规则。
- 路径示例中的 `{id}` 改为语义化参数名。

### 4.2 中优先级修复

1. `docs/modules/a-intake/contract.md`
2. `docs/modules/b-parsing/contract.md`
3. `docs/modules/c-agent/contract.md`
4. `docs/modules/d-workbench/contract.md`
5. `docs/modules/*/work-breakdown.md`

动作：统一所有路径参数命名与错误码写法。

### 4.3 低优先级修复

1. `docs/prd_v2.md`
2. `docs/01-product/functional-breakdown.md`

动作：将描述性语句与最终口径对齐。

## 5. 非目标

1. 不在本阶段改动任何业务代码。
2. 不新增数据库表或 API 功能。
3. 不调整模块边界与职责拆分。

## 6. 验收标准

文档修复完成后，应满足：

1. 全仓不存在“回退仅返回、不修改草稿”的描述。
2. 全仓不存在“导出同步返回 PDF binary”的契约描述。
3. 全仓路径参数示例不再使用泛化 `{id}`。
4. 全仓 JSON 示例中的 `code` 均为合法数字表示。
5. 全仓不再出现带前导零错误码写法（如 `02005`）。

## 7. 风险与缓解

1. 风险：错误码体系改写影响历史认知。
- 缓解：在 API 规约中增加“v2 文档标准以纯数字为准”的说明。

2. 风险：参数重命名导致阅读者与旧习惯冲突。
- 缓解：集中修订示例并在路径规范中给出映射样例。

3. 风险：导出异步化描述与已有同步理解冲突。
- 缓解：明确说明这是统一规约，不涉及本次代码迁移。

## 8. 执行说明

- 本文档仅定义修复设计，不包含代码提交。
- 按用户要求，本次不执行 git commit。

---

## 9. 额外修复项（审查补充）

以下问题在代码审查中发现，原 Section 4 未覆盖。

### 9.1 C1 — prd_v1.md 废弃标头

**问题**：`docs/prd_v1.md` 描述 LaTeX/Patch 架构，与整个 v2 文档矛盾，且无废弃标头。

**修复**：在 `docs/prd_v1.md` 第 1 行之前插入废弃块，参照 `patch-schema.md` 的格式：

```markdown
> **此文档已废弃。** v2 架构采用"HTML 是唯一数据源"设计，不再使用 LaTeX 渲染、
> PatchEnvelope、SourceDoc、ResumeSpec、ResolvedResumeSpec 等中间概念。
>
> 请参阅 [v2 PRD](./prd_v2.md) 和 [v2 架构设计](./superpowers/specs/2026-04-23-architecture-v2-design.md)。
```

**文件**：`docs/prd_v1.md`

### 9.2 C2 — 版本自动创建触发机制

**问题**：E 契约 Section 5 期望 B/C/D 触发版本创建，但 B/C/D 契约均未记录此要求。

**已确认决策**：B/C/D 各自在完成操作后调用 E 的版本创建逻辑。

#### 模块 B（解析初稿生成后）

`POST /api/v1/parsing/generate` 成功后，服务端在事务内：
1. 写入 `drafts.html_content`
2. 调用 E 的版本创建逻辑，label = `"AI 初始生成"`
3. 返回 `draft_id` + `html_content` + `version_id`

响应增加 `version_id` 字段。

#### 模块 D（手动保存 / AI 应用保存）

前端调用 D 的 `PUT /api/v1/drafts/{draft_id}` 时：
- 自动保存（debounce 2s）：**不触发**版本创建（避免版本爆炸）
- 仅当请求显式携带 `create_version: true` 时创建版本
- `version_label` 字段可选，默认 `"手动保存"`

请求 body 扩展：

```json
{
  "html_content": "<!DOCTYPE html>...",
  "create_version": true,
  "version_label": "AI 修改：精简项目经历"
}
```

响应扩展（仅当创建了版本时）：

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "updated_at": "2026-04-23T20:05:00Z",
    "version_id": 5
  }
}
```

#### 模块 C

C 不负责 HTML 替换和版本创建。用户点击"应用到简历"后，前端调用 D 的 PUT（携带 `create_version: true` 和 `version_label`）。

#### 模块 E

`POST /api/v1/drafts/{draft_id}/versions` 保持不变。新增一个内部版本创建函数（供 B/D handler 调用），不暴露为独立 API。

**文件**：
- `docs/modules/b-parsing/contract.md` — 输出契约加 version_id，下游加"模块 E"
- `docs/modules/b-parsing/work-breakdown.md` — 交付清单更新
- `docs/modules/c-agent/contract.md` — Section 5.4 补充说明前端调用 D 时携带版本参数
- `docs/modules/d-workbench/contract.md` — PUT 端点扩展请求/响应，下游加"模块 E"
- `docs/modules/d-workbench/work-breakdown.md` — 交付清单更新

### 9.3 I1 — api-conventions.md 错误码章节内部自相矛盾

**问题**：同一文件 Section 4.1 描述 `SSCCC` 格式（示例 `01001`），Section 4.3 列表使用纯数字 `1001-1999`。

**修复**：重写 Section 4.1，删除 `SSCCC` 和 `01001` 描述，改为与 Section 4.3 一致的纯数字分段。同时修复 Section 4.2 通用错误码中的 `0`（成功码）为显式说明。

**文件**：`docs/01-product/api-conventions.md`

### 9.4 I2 — work-breakdown 回滚测试描述

**问题**：`e-render/work-breakdown.md` 第 86 行写"返回指定版本的 html_snapshot"（只读语义），与同文件第 55 行和 E 契约的写回语义矛盾。

**修复**：将第 86 行测试描述改为"回退到指定版本，验证 drafts.html_content 被更新为新版本快照"。

**文件**：`docs/modules/e-render/work-breakdown.md`

### 9.5 I3 — 任务状态查询端点归入模块 E

**问题**：`GET /api/v1/tasks/{task_id}` 在 api-conventions 和 E 契约中被引用，但不属于任何模块的正式 API 列表。

**已确认决策**：归入模块 E。

**修复**：
- E 契约 API 端点表增加第 5 个端点：`GET /api/v1/tasks/{task_id}`
- 补充完整的请求/响应契约（成功、失败、轮询状态）
- E work-breakdown 交付清单更新为"5 个 API 端点"
- api-conventions.md Section 8 注明此端点归属于模块 E

**文件**：
- `docs/modules/e-render/contract.md`
- `docs/modules/e-render/work-breakdown.md`
- `docs/01-product/api-conventions.md`

### 9.6 I4 — 模块前缀规则更新

**问题**：api-conventions 定义"每个模块一个前缀"，但 D（`/api/v1/drafts/`）和 E（`/api/v1/drafts/{draft_id}/`）共享前缀。

**修复**：更新 api-conventions.md Section 2.1 模块前缀表，明确说明 E 的端点是 drafts 资源的子路径，由 D 和 E 共享 `/api/v1/drafts/` 前缀：

```
| D 可视化编辑 | `/api/v1/drafts/` | `GET /api/v1/drafts/{draft_id}` |
| E 版本导出   | `/api/v1/drafts/{draft_id}/...` | （drafts 子资源） |
```

**文件**：`docs/01-product/api-conventions.md`

### 9.7 I5 — 图片上传 v1 限制说明

**问题**：A 模块接受 `resume_image` 上传，但 B 模块不处理图片解析。用户上传图片后系统静默忽略。

**已确认决策**：v1 保留图片上传能力，标注为"仅作参考/头像提取，暂不支持 OCR 识别"。

**修复**：
- A 契约上传端点说明中增加 v1 限制：图片资产仅存储，B 模块解析时跳过
- B 契约解析策略表中图片行增加说明："v1 跳过，不发送给 AI。图片存储在 assets 表供前端手动引用（如头像）。"

**文件**：
- `docs/modules/a-intake/contract.md`
- `docs/modules/b-parsing/contract.md`

### 9.8 I6 — 回滚响应格式对齐

**问题**：E 契约回滚响应格式（`{draft_id, version_id, new_version_id, message}`）与 fix-design Section 3.1 定义的新格式不一致。

**修复**：将 E 契约回滚响应替换为 Section 3.1 定义的格式。

**文件**：`docs/modules/e-render/contract.md`

### 9.9 M2 — README 目录列表补全

**问题**：`docs/README.md` 目录列表遗漏 `prd_v1.md`（已废弃）和 `patch-schema.md`（已废弃）。

**修复**：在目录列表中增加这两个文件，标注"已废弃"。

**文件**：`docs/README.md`

### 9.10 M3 — 模块标准显示名统一

**问题**：模块显示名在不同文档中不一致（如 E 模块有 4 种叫法）。

**已确认标准名**：

| 模块 | 标准显示名 | 目录名 | Go 包名 |
|------|-----------|--------|---------|
| A | 资料接入 | `a-intake` | `a_intake` |
| B | 解析初稿 | `b-parsing` | `b_parsing` |
| C | AI 对话 | `c-agent` | `c_agent` |
| D | 可视化编辑 | `d-workbench` | `d_workbench` |
| E | 版本导出 | `e-render` | `e_render` |

**修复**：在 README.md、api-conventions.md、functional-breakdown.md、dev-work-breakdown.md 中统一使用标准显示名。

**文件**：
- `docs/README.md`
- `docs/01-product/api-conventions.md`
- `docs/01-product/functional-breakdown.md`
- `docs/01-product/dev-work-breakdown.md`

### 9.11 prd_v2 / functional-breakdown 回滚描述对齐

**问题**：`prd_v2.md` 和 `functional-breakdown.md` 中回滚描述为"加载历史快照到编辑器"（模棱两可），应明确为写回语义。

**修复**：将两处描述改为"回退到指定版本（写回当前草稿并自动创建新快照）"。

**文件**：
- `docs/prd_v2.md`
- `docs/01-product/functional-breakdown.md`

## 10. 更新后的验收标准

除原 Section 6 的 5 项标准外，新增：

6. `docs/prd_v1.md` 顶部包含废弃标头，指向 v2 文档。
7. `docs/README.md` 目录列表包含所有 `docs/` 根目录文件（含已废弃标注）。
8. B/C/D 契约明确记录版本创建触发机制（输出契约或 API 端点中）。
9. `GET /api/v1/tasks/{task_id}` 归属于模块 E 的正式 API 端点表。
10. `api-conventions.md` 模块前缀表反映 D/E 共享前缀的现实。
11. A 契约和 B 契约包含图片上传的 v1 限制说明。
12. 全仓模块显示名与 Section 9.10 标准名一致。

## 11. 更新后的文件修改清单

| # | 文件 | 修复项 |
|---|------|--------|
| 1 | `docs/prd_v1.md` | 9.1 |
| 2 | `docs/README.md` | 9.9, 9.10 |
| 3 | `docs/prd_v2.md` | 9.11 |
| 4 | `docs/01-product/api-conventions.md` | 4.1, 9.3, 9.5, 9.6, 9.10 |
| 5 | `docs/01-product/functional-breakdown.md` | 4.3, 9.10, 9.11 |
| 6 | `docs/01-product/dev-work-breakdown.md` | 4.2, 9.10 |
| 7 | `docs/02-data-models/core-data-model.md` | 4.2 |
| 8 | `docs/modules/a-intake/contract.md` | 4.2, 9.7 |
| 9 | `docs/modules/a-intake/work-breakdown.md` | 4.2 |
| 10 | `docs/modules/b-parsing/contract.md` | 4.2, 9.2, 9.7 |
| 11 | `docs/modules/b-parsing/work-breakdown.md` | 4.2, 9.2 |
| 12 | `docs/modules/c-agent/contract.md` | 4.2, 9.2 |
| 13 | `docs/modules/c-agent/work-breakdown.md` | 4.2 |
| 14 | `docs/modules/d-workbench/contract.md` | 4.2, 9.2 |
| 15 | `docs/modules/d-workbench/work-breakdown.md` | 4.2, 9.2 |
| 16 | `docs/modules/e-render/contract.md` | 4.1, 9.5, 9.8 |
| 17 | `docs/modules/e-render/work-breakdown.md` | 4.1, 9.4, 9.5 |
| 18 | `docs/superpowers/specs/2026-04-23-architecture-v2-design.md` | 4.1 |

共 18 个文件。
