# MSW Browser Worker 补齐 + 空白草稿自动创建

**日期：** 2026-04-29
**状态：** 已批准

## 背景

当前 MSW browser worker 仅注册了 `draftHandlers`（GET/PUT），缺少 `projectHandlers`，导致 `VITE_USE_MOCK=true` 时 `GET /api/v1/projects/:id` 请求穿透到真实后端。此外，当项目没有草稿时，编辑器显示空白状态但无操作入口，用户无法开始编辑。

## 方案选择：方案 A（最小改动）

精准解决两个问题：MSW mock 断裂 + 无草稿空状态。

## 1. MSW Browser Worker 补齐

### 改动文件

| 文件 | 变更 |
|---|---|
| `frontend/workbench/src/mocks/browser.ts` | 注册 `projectHandlers` |
| `frontend/workbench/src/mocks/handlers/drafts.ts` | 新增 `POST /api/v1/drafts` mock |

### 具体变更

**browser.ts：**
```typescript
import { projectHandlers } from './handlers/projects'
import { draftHandlers } from './handlers/drafts'
setupWorker(...projectHandlers, ...draftHandlers)
```

**drafts.ts 新增：**
```
POST /api/v1/drafts
  Request:  { "project_id": 1 }
  Response: { id: 2, project_id: 1, html_content: "", updated_at: "2026-04-29T12:00:00Z" }
```

Mock 的 `sampleProject` 已有 `current_draft_id: 1`，加载流程在 mock 下自然走通。

## 2. 后端：新增 `POST /api/v1/drafts`

### 接口契约

```
POST /api/v1/drafts
Request:  { "project_id": 1 }
Response: { "code": 0, "data": { "id": 2, "project_id": 1, "html_content": "", "updated_at": "..." }, "message": "ok" }
```

### 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 4003 | 404 | 项目不存在 |
| 4004 | 409 | 项目已有当前草稿 |

### 实现逻辑（TDD）

1. **Service 层** 新增 `Create(projectID uint) (*models.Draft, error)`
   - 校验 project 存在 → 不存在返回 `ErrProjectNotFound` (4003)
   - 校验 project 无 current_draft → 已有返回 `ErrProjectHasDraft` (4004)
   - 创建 `Draft{ ProjectID, HTMLContent: "" }`
   - 更新 `project.CurrentDraftID = draft.ID`
2. **Handler 层** 新增 `CreateDraft` handler
3. **路由注册：** `rg.POST("/drafts", handler.CreateDraft)`

## 3. 前端：空状态添加"新建草稿"按钮

### 改动文件

| 文件 | 变更 |
|---|---|
| `frontend/workbench/src/lib/api-client.ts` | 添加 `workbenchApi.createDraft` |
| `frontend/workbench/src/components/editor/EditorEmptyState.tsx` | 添加"新建草稿"按钮 |
| `frontend/workbench/src/pages/EditorPage.tsx` | 添加 `createDraft` 回调 |

### 交互流程

```
加载项目 → current_draft_id 为 null
  → 显示 EditorEmptyState（含"新建草稿"按钮）
  → 点击 → POST /api/v1/drafts { project_id }
  → 成功 → 设置 draftId → loadDraft → 'ready' 状态
  → 失败 → 显示错误提示
```

### API Client 新增

```typescript
workbenchApi.createDraft(projectId: number): Promise<Draft>
  → POST /drafts { project_id: projectId }
```

## 4. 数据流

```
Mock 模式 (VITE_USE_MOCK=true):
  GET /projects/1 → { current_draft_id: 1 }
  GET /drafts/1   → { html_content: "Sample Draft..." }
  → 编辑器进入 ready 状态

真实模式 (VITE_USE_MOCK=false):
  GET /projects/1 → { current_draft_id: null }
  → 显示空状态 + "新建草稿"按钮
  → 点击 → POST /drafts → GET /drafts → 进入 ready
  → 用户输入 → auto-save PUT /drafts/:id
```
