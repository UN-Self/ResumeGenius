# AI 编辑后编辑器正确同步

## 问题

AI `apply_edits` 在后端事务中正确修改了 `draft.html_content`，但前端 `onApplyDiffHTML` 用 `<del>/<ins>` diff 标记覆盖编辑器内容，触发自动保存（2s debounce PUT）把脏 HTML 写回后端，覆盖掉正确的内容。

结果：用户看到带删除线/下划线的 diff 标记，且数据库中的 HTML 被污染。

## 方案

前端在 `done` 事件后，从后端拉取干净 HTML 设置到编辑器，用 `restoringContent` flag 跳过自动保存。

## 改动范围

### 1. `frontend/workbench/src/components/chat/ChatPanel.tsx`

- prop `onApplyDiffHTML(edits: PendingEdit[])` 改为 `onApplyEdits(): Promise<void>`
- `done` 事件中调用 `onApplyEdits()` 替代 `onApplyDiffHTML(edits)`
- 移除 `pendingEdits` 状态及"已生成 N 项修改"提示 UI
- 移除 `PendingEdit` 类型导入
- 保留 Undo/Redo 按钮不变

### 2. `frontend/workbench/src/pages/EditorPage.tsx`

- `onApplyDiffHTML` 回调改为 `onApplyEdits`：
  1. 调 `workbenchApi.getDraft(Number(draftId))` 拉取最新 HTML
  2. 用 `restoringContent = true` 包裹 `setContent`，防止触发自动保存
  3. 完成后 `restoringContent = false`
- `onRestoreHtml` 同样加上 `restoringContent` 保护
- 移除 `PendingEdit` 类型导入
- 移除 `Deletion` / `Insertion` diff 扩展（不再需要）

### 3. 不改后端

## 数据流（修复后）

```
AI apply_edits → 后端写入 draft.html_content
                → SSE edit 事件（前端忽略）
                → SSE done 事件
前端收到 done → GET /drafts/:id
             → setContent(干净 HTML)
             → restoringContent 跳过自动保存
             → 编辑器展示最终结果
```

## 涉及文件

| 文件 | 改动 |
|---|---|
| `frontend/workbench/src/components/chat/ChatPanel.tsx` | prop 接口变更，移除 diff 逻辑 |
| `frontend/workbench/src/pages/EditorPage.tsx` | 回调实现变更，移除 diff 扩展 |
