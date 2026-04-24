# v2 文档一致性修复设计

日期：2026-04-24
状态：已确认（待执行）

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
