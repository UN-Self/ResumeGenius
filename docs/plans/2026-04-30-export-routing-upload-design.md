# 导出修复 + 智能路由 + 工作区上传

## 概述

三个前端改动：修复导出 PDF 功能、项目列表智能路由、工作区左侧栏添加文件上传入口。

## 一、导出 PDF 接线

### 问题

- `useExport.ts` 模板字符串用单引号（line 63），`${taskId}` 不插值
- `useExport.ts` 拼写错误 `"faild"` → `"failed"`（line 68）
- `ActionBar.tsx` 导出按钮永久 `disabled`，无 `onClick`
- `useExport` hook 从未被导入使用

### 修改

| 文件 | 改动 |
|---|---|
| `useExport.ts` | 修复单引号 → 反引号、`"faild"` → `"failed"` |
| `ActionBar.tsx` | 新增 props `draftId`、`getHtml`、`exportStatus`；去掉 `disabled`，绑定 `onClick` 调用 `exportPdf`；按钮文字随 `exportStatus` 变化 |
| `EditorPage.tsx` | 导入 `useExport`，传 `draftId` 和 `editor.getHTML()` 给 `ActionBar` |

### 按钮状态

| 状态 | 文字 | disabled |
|---|---|---|
| idle | 导出 PDF | 否 |
| exporting | 导出中... | 是 |
| failed | 导出失败 | 否 |
| completed | 导出 PDF | 否 |

## 二、智能路由

### 修改

| 文件 | 改动 |
|---|---|
| `ProjectList.tsx` | `onClick` 中判断 `project.current_draft_id`，有 draft 跳 `/projects/:id/edit`，否则跳 `/projects/:id` |

## 三、工作区上传按钮

### 行为

上传文件后触发解析，解析结果只在左侧栏展示，不做自动替换编辑器内容。

### 修改

| 文件 | 改动 |
|---|---|
| `ParsedSidebar.tsx` | 新增 `projectId`、`onParsed` props；顶部加"上传文件"按钮，点击弹出 `UploadDialog`；上传成功后调用 `parsingApi.parseProject` |
| `EditorPage.tsx` | 传 `projectId` 和 `setParsedContents` 给 `ParsedSidebar` |

### 数据流

```
用户点击上传 → UploadDialog → intakeApi.uploadFile(pid, file)
→ parsingApi.parseProject(pid) → setParsedContents(result) → 侧栏刷新
```

## 测试要点

- 导出：mock 后端 task 轮询，验证 idle → exporting → completed 状态流转
- 路由：验证有 draft 的项目直接进编辑器，无 draft 的进解析页
- 上传：验证上传后侧栏刷新显示新解析结果
