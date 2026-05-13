# AI 页数反馈与 Skill 发现修复设计

日期：2026-05-13

## 问题

1. **AI 无法感知页数**：`apply_edits` 返回 `{"applied":N,"failed":N,"new_sequence":N}`，没有任何页数信息。AI 不知道内容溢出。
2. **Skill 发现机制断裂**：`SkillLoader` 只传给了 `AgentToolExecutor`，`ChatService` 没有引用。`DefaultPromptSections` 的 `skillListing` 参数永远为空。AI 看到 `load_skill` 工具但不知道有什么 skill 可用。
3. **A4 尺寸指导缺乏精确数据**：`a4-guidelines.yaml` 只有文字描述，没有像素尺寸、内容容量参考和模板骨架。

## 方案

### 改动 1：修复 Skill 发现机制

`ChatService` 增加 `skillLoader` 字段，在构建系统提示词时调用 `buildSkillListing()` 生成技能清单，传入 `DefaultPromptSections(assetInfo, skillListing)`。

技能清单格式：
```
## 可用技能
- resume-design: A4 单页简历设计规范。用户要求调整样式、排版、配色、模板时加载使用。
  调用 load_skill(skill_name="resume-design") 获取完整规范和模板。
```

改动文件：
- `agent/service.go` — ChatService 增加 skillLoader 字段，构建 skillListing
- `agent/routes.go` — 传 skillLoader 给 NewChatService
- `agent/skill_loader.go` — 新增 buildSkillListing() 方法

### 改动 2：A4 模板参考文档

在 `resume-design` skill 中新增 `a4-template.yaml` 参考，包含：
- 精确像素尺寸（794×1123px 画布，642×987px 有效内容区）
- 内容容量参考（~47 行、~1500-2000 字/页）
- HTML 模板骨架结构
- 控制篇幅的具体操作清单

改动文件：
- `skills/resume-design/references/a4-template.yaml` — 新增
- `skills/resume-design/skill.yaml` — 注册新 reference

### 改动 3：前端页数回传 + get_draft 返回页数

流程：
1. Draft 模型增加 `page_count` 字段
2. 前端 TipTap onUpdate 回调中读取 PaginationPlus 页数，debounce 后 PATCH 回传
3. `get_draft` 工具返回值从纯 HTML 改为 `{ html, page_count }`
4. AI 读简历时自然看到当前页数，结合用户需求自行判断是否压缩

不改系统提示词。页数作为工具返回值，不硬编码目标页数。

改动文件：
- `shared/models/draft.go` — 加 PageCount 字段
- `agent/tool_executor.go` — get_draft 返回值增加 page_count
- `workbench/handler.go` 或新增路由 — PATCH /drafts/:id/meta 端点
- `frontend/workbench/src/` — TipTap onUpdate 回传页数

## 不做的事情

- 不在 apply_edits 返回值中加页数估算（精度不够，且需要后端渲染）
- 不在系统提示词中硬编码"目标 1 页"（用户可能需要多页）
- 不改 chromedp 渲染逻辑
- 不改现有 a4-guidelines.yaml 内容

## 参考

- Claude-Code 的 skill 发现机制：每 turn 注入 `<system-reminder>` 技能清单，工具 description 引导模型查看清单
- TipTap PaginationPlus：已有精确分页逻辑，pageHeight=1123px、marginTop/MarginBottom=68px
- render-template.html：`@page { size: A4; margin: 18mm 20mm }`，有效可打印区 170mm×261mm
