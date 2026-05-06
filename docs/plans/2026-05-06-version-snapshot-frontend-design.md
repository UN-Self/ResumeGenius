# 版本快照前端功能设计

日期：2026-05-06

## 1. 目标

在编辑器页面实现版本快照的四个核心功能：
1. 版本历史浏览
2. HTML 预览（在编辑区切换为只读渲染）
3. 手动保存快照（支持自定义标签）
4. 版本回退（确认弹窗 → rollback API → 刷新编辑器）

## 2. UI 方案：ActionBar 下拉面板

点击 ActionBar 已有的「版本历史」按钮，弹出 Popover 下拉面板。选中某版本后，编辑区临时切换为该版本的只读 HTML 渲染，顶部显示蓝色预览提示栏。

对现有三栏布局影响最小，类似 Google Docs 版本历史体验。

## 3. 架构

```
ActionBar [版本历史▼]
      │
      ▼ 点击
VersionDropdown (Popover)
  ├── 版本列表（从 GET /versions 加载）
  ├── [+ 保存快照] → SaveSnapshotDialog
  │
  └── 点击某版本 → 编辑区切换为只读预览
                    ↓
              VersionPreviewBanner
              [回退到此版本] → RollbackConfirmDialog
              [关闭预览]
```

### 3.1 新增前端组件

| 组件 | 文件 | 职责 |
|---|---|---|
| `VersionDropdown` | `components/version/VersionDropdown.tsx` | Popover 下拉面板，显示版本时间线，触发预览/保存 |
| `VersionPreviewBanner` | `components/version/VersionPreviewBanner.tsx` | 蓝色提示栏，显示当前预览版本信息，含回退和关闭按钮 |
| `SaveSnapshotDialog` | `components/version/SaveSnapshotDialog.tsx` | 保存快照对话框，含可选标签输入框 |
| `RollbackConfirmDialog` | `components/version/RollbackConfirmDialog.tsx` | 回退确认弹窗 |

### 3.2 新增 Hook

`hooks/useVersions.ts` — 封装版本列表加载、预览状态管理、创建快照、回退操作。

### 3.3 API 层

在 `lib/api-client.ts` 新增 `renderApi` 命名空间：

```typescript
export interface Version {
  id: number
  label: string
  created_at: string
}

export interface VersionDetail extends Version {
  html_snapshot: string
}

export const renderApi = {
  listVersions: (draftId: number) =>
    request<{ items: Version[]; total: number }>(`/drafts/${draftId}/versions`),
  getVersion: (draftId: number, versionId: number) =>
    request<VersionDetail>(`/drafts/${draftId}/versions/${versionId}`),
  createVersion: (draftId: number, label: string) =>
    request<Version>(`/drafts/${draftId}/versions`, {
      method: 'POST',
      body: JSON.stringify({ label }),
    }),
  rollback: (draftId: number, versionId: number) =>
    request<{ draft_id: number; updated_at: string; new_version_id: number; new_version_label: string; new_version_created_at: string }>(`/drafts/${draftId}/rollback`, {
      method: 'POST',
      body: JSON.stringify({ version_id: versionId }),
    }),
}
```

## 4. 后端新增端点

当前 `GET /drafts/{draft_id}/versions` 只返回 `{id, label, created_at}`，不包含 `html_snapshot`。需要新增一个获取单个版本详情的端点。

### 4.1 新增端点

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/drafts/{draft_id}/versions/{version_id}` | 获取单个版本详情（含 html_snapshot） |

### 4.2 响应格式

```json
{
  "code": 0,
  "data": {
    "id": 3,
    "label": "AI 修改：精简项目经历",
    "html_snapshot": "<!DOCTYPE html>...",
    "created_at": "2026-04-23T20:15:00Z"
  }
}
```

### 4.3 后端改动

- `handler.go`：新增 `GetVersion` handler
- `service.go`：新增 `GetByID` 方法
- `routes.go`：注册 `GET /drafts/:draft_id/versions/:version_id` 路由

## 5. 契约文档同步

新增端点需同步更新以下文档：
- `docs/modules/render/contract.md` — 在 API 端点表中新增行，添加关键端点详情
- `docs/modules/render/work-breakdown.md` — 如有相关任务分解

## 6. 状态设计

### 6.1 EditorPage 状态扩展

```typescript
// 预览模式状态
previewMode: 'editing' | 'previewing'
previewVersion: Version | null     // 当前预览的版本元数据
previewHtml: string | null         // 懒加载的 HTML 快照
```

### 6.2 编辑区渲染逻辑

```
if (previewMode === 'previewing' && previewHtml)
  → A4Canvas + dangerouslySetInnerHTML(previewHtml)  // 只读渲染
  + VersionPreviewBanner
else
  → 正常 TipTapEditor
```

### 6.3 useVersions Hook 接口

```typescript
interface UseVersionsReturn {
  versions: Version[]
  loading: boolean
  previewMode: 'editing' | 'previewing'
  previewVersion: Version | null
  previewHtml: string | null
  refreshList: () => Promise<void>
  startPreview: (version: Version) => Promise<void>
  exitPreview: () => void
  createSnapshot: (label: string) => Promise<void>
  rollback: () => Promise<string>  // 返回新的 html_content
}
```

## 7. 用户流程

### 浏览版本
1. 点击 ActionBar「版本历史」按钮
2. Popover 下拉面板显示版本列表（按时间倒序，最新在上）
3. 每项显示：标签 + 相对时间（如「5 分钟前」）

### HTML 预览
1. 点击某个版本项
2. Popover 关闭，编辑区切换为只读 HTML 渲染
3. 顶部蓝色 banner 显示「正在预览: v3 - AI 修改」
4. banner 含「回退到此版本」和「关闭预览」按钮
5. 关闭预览后恢复 TipTap 编辑器

### 手动保存快照
1. Popover 底部点击「+ 保存快照」
2. 弹出 SaveSnapshotDialog
3. 可选输入标签（placeholder: "可选，如「校招版」"）
4. 确认 → 调 createVersion API → 刷新版本列表
5. 空标签时后端默认为「手动保存」

### 版本回退
1. 预览模式下点击「回退到此版本」
2. 弹出 RollbackConfirmDialog：「回退将覆盖当前编辑内容，是否继续？」
3. 确认 → 调 rollback API
4. 用返回的新版本 HTML 刷新编辑器（editor.commands.setContent）
5. 退出预览模式，恢复正常编辑
6. 刷新版本列表

## 8. 错误处理

| 场景 | 处理 |
|---|---|
| 版本列表加载失败 | 下拉面板显示错误状态 + 重试按钮 |
| 获取版本 HTML 失败 | 预览区显示错误提示 |
| 创建快照失败 | toast 提示错误信息 |
| 回退失败 | toast 提示错误信息，保持当前状态 |
| 网络断开 | 复用 ApiError 的统一错误处理 |

## 9. 测试要点

### 前端测试
- VersionDropdown 渲染和交互
- 版本列表加载（成功/失败）
- 预览模式切换（进入/退出）
- SaveSnapshotDialog 标签输入和提交
- RollbackConfirmDialog 确认/取消
- useVersions hook 的状态转换

### 后端测试
- GetVersion 端点（正常/不存在/非本 draft 的 version）
