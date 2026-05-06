# Resume AI Agent V2 设计文档

> "简历界的 Claude Code" — 让 AI 像 Claude Code 编辑代码一样精准编辑简历

## 已确定的设计决策

### 1. 产品愿景

用户不需要理解系统的内部复杂性。产品形态：

```
上传文件/资料 → 输入需求 → 系统内部智能编排 → 得到优化后的简历
```

- 用户不是技术用户，不暴露 slash commands 或底层概念
- 系统内部的规划、工具调用、多步编排对用户透明
- 用户只需要看到"简历变了"以及"变成了什么样"

### 2. 编辑范式：Search/Replace

采用和 Claude Code 完全一致的编辑机制 — **Search/Replace Edit**。

**核心原理**：HTML 就是代码，TipTap 就是 IDE，AI 就是 Claude Code。

AI 不再全量输出 HTML。AI 输出的是 Search/Replace 操作：

```typescript
interface SearchReplaceOp {
  old_string: string    // 必须在当前 draft HTML 中精确匹配
  new_string: string    // 替换后的内容
  description?: string  // AI 对这个修改的自然语言说明
}
```

**选择理由**（相比 Full HTML + Diff 或结构化数据层）：
- 简单可靠 — 不需要解析 AST，纯文本操作
- 通用 — 对任何 HTML 格式都有效
- Token 高效 — 只传输变化的部分
- 可验证 — old_string 必须精确匹配，不匹配就报错
- 模型友好 — LLM 天然擅长"找到这段文字并替换"

### 3. 变更可视化：Rendered Diff

AI 的修改不是以代码 diff 展示，而是以**渲染后的可视化变更**展示：

- 用户看到的是：字体变了、文字改了、条目删了、排版调整了
- 用户感知不到底层的 HTML 代码变动
- 每个变更是一个"变更卡片"，用户可以逐条 accept/reject
- 类似 PR Review 的体验，但是渲染后的效果而非代码

**变更分类**（后端自动判断，通过对比 old/new 的 HTML 差异）：
| 判断逻辑 | 变更类型 | 可视化样式 |
|---------|---------|-----------|
| 只有文本节点变化 | 内容修改 | 红色删除 → 绿色新增 |
| `<style>` 相关变化 | 样式修改 | "字体从 A 变为 B" |
| 整个 `<div class="item">` 被删除 | 条目删除 | 整条红色划线 |
| 新增了 `<div class="item">` | 条目新增 | 整条绿色新增 |

### 4. 版本控制：独立 Edit History 表

每个 Draft 维护独立的编辑链（edit chain），支持 Undo/Redo。

**使用独立表** `draft_edits`（而非 JSONB 字段），理由：
- 更干净 — 每条记录是一行，结构清晰
- 查询方便 — SQL 直接查，不需要解析 JSONB
- 不会有行过大问题
- 未来可扩展（对比任意两个版本、AI 回顾修改历史等）

**与现有 versions 表的关系**：
- `draft_edits` — 细粒度编辑历史，每次 AI 或用户修改都记录，用于 undo/redo
- `versions` — 显式版本快照（用户手动创建或重要节点自动创建），用于长期版本管理
- 两者共存，类似 git commits vs git tags

**清理策略**：超过 50 条或超时 30 分钟无 undo/redo，压缩 edit chain。

### 5. 上下文管理：Claude Code 哲学

**固定 System Prompt**：不动态拼接内容，保证前缀一致、缓存命中率高。

**AI 自主探索**：不预注入简历 HTML 或用户资料，AI 通过工具按需获取需要的信息。

**简历直接全量读取**：一份简历最多两页 A4（20-50KB），不需要搜索工具，直接 get_draft 全量读取。

**用户资料按需搜索**：用户的资料库（旧简历、Git 摘要、Notes 等）可能很大，需要搜索/过滤工具让 AI 自主制定搜索策略。

**Compaction 而非截断**：对话历史超出阈值时，像 Claude Code 一样自动 compact — 旧的对话被压缩为摘要，不是简单丢弃。

### 6. 交互模式

- **增强型聊天面板**：右侧面板保留，但增强为显示 diff 预览、逐条 accept/reject
- **选区触发 AI**：在编辑器中选中文字，右键或快捷键触发 AI 操作
- **无 Slash Commands**：用户不学习命令，自然语言输入即可

### 7. 模型能力假设

面向最先进的模型能力设计（Claude、GPT-4、DeepSeek 等高端模型），不因当前使用的 doubao-2.0-pro 限制设计。

---

## 待设计部分

以下部分尚未确定，需要继续讨论：

### A. 工具集设计（已确定）

对标 Claude Code 的工具哲学：少而精，每个工具足够通用，通过组合覆盖所有场景。版本保存和 PDF 导出由前端 UI 触发，不暴露给 AI。

**3 个核心工具：**

| 工具 | 对标 | 职责 |
|------|------|------|
| `get_draft` | Read | 读取当前简历 HTML，支持 CSS selector 指定范围或全量读取 |
| `apply_edits` | Edit | 提交 SearchReplaceOps，后端验证精确匹配并原子应用 |
| `search_assets` | Grep + Read | 按关键词/类型搜索用户资料，短内容返回全文，长内容返回匹配片段 |

**`get_draft` 参数：**
```typescript
// 无参数 → 全量返回
// 有 selector → 只返回匹配的 HTML 片段
get_draft(selector?: string): string
```

**`apply_edits` 参数：**
```typescript
interface SearchReplaceOp {
  old_string: string    // 必须在当前 draft HTML 中精确匹配
  new_string: string    // 替换后的内容
  description?: string  // AI 对这个修改的自然语言说明
}
apply_edits(ops: SearchReplaceOp[]): ApplyResult
```

**`search_assets` 参数：**
```typescript
interface AssetFilter {
  query?: string        // 自然语言搜索关键词
  type?: string         // 资料类型过滤：resume | git_summary | note
  limit?: number        // 返回数量上限，默认 5
}
search_assets(filter: AssetFilter): AssetResult[]
```

---

### B. 前端架构（已确定）

**1. 可视化 Diff：Inline 修订模式**
- 变更直接高亮在 TipTap 画布上，类似 Word 修订模式（删除红划线、新增绿色）
- Accept/Reject 粒度跟随 AI 返回的 SearchReplaceOps，不额外分组
- 变更类型由后端通过对比 old/new HTML 自动判断

**2. 选区触发 AI**
- 待后续细聊，暂不设计

**3. 增强型 Chat Panel**
- 工具调用日志：对标 Claude Code，展开可看 input/output，收起显示一行摘要
- AI 计划/TODO：展示 AI 的任务规划和执行进度
- 复用现有 SSE 事件类型（thinking、tool_call、tool_result）渲染

**4. Undo/Redo**
- 通过按钮触发（非快捷键），后端基于 draft_edits edit chain 实现

### C. 后端架构（已确定）

**1. apply_edits 流程**

```
AI 提交 SearchReplaceOps[]
  → 开事务，读取当前 HTML
  → 逐条验证 old_string 是否精确匹配
  → 全部匹配 → 依次应用，保存新 HTML，写入 draft_edits
  → 任一不匹配 → 回滚，返回失败的那条及上下文，AI 重试
```

原子操作：要么全成功，要么全不动。AI 拿到失败信息后自己修正重试。

**2. Compaction**

对话 token 数达到上下文窗口 80% 时触发（上下文窗口大小通过 env 配置，适配不同模型）。调用 AI 将旧消息压缩为摘要，保留：讨论了什么、做了哪些修改、当前简历状态。

**3. draft_edits schema**

```sql
CREATE TABLE draft_edits (
  id            BIGSERIAL PRIMARY KEY,
  draft_id      BIGINT NOT NULL REFERENCES drafts(id),
  sequence      INT    NOT NULL,          -- 操作序号，用于 undo/redo 定位
  op_type       VARCHAR(20) NOT NULL,     -- search_replace | user_edit
  old_string    TEXT,                     -- 被替换的内容
  new_string    TEXT,                     -- 替换后的内容
  description   TEXT,                     -- AI/用户的修改说明
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

Undo 回退到指定 sequence 的状态，Redo 正向重放。超过 50 条或 30 分钟无 undo/redo 时压缩 edit chain。

**4. 与现有 Agent 集成**

保留 `StreamChatReAct` 循环骨架，替换内容：
- 工具集从 5 个换成新的 3 个（get_draft, apply_edits, search_assets）
- System prompt 固定化，不动态拼接
- 新增 compaction 触发逻辑（80% 上下文窗口阈值）
- SSE 事件扩展 edit 类型，供前端 inline diff 渲染
- Provider adapter（OpenAI-compatible）不变

### D. 实施路径（已确定）

- 单阶段直接替换现有 agent，不分期
- 不考虑向后兼容，产品未上线
- 集成测试使用 docker compose 启动环境
- 完成后提交 commit

---

## 实施顺序建议

1. **后端：工具层** — 实现 get_draft、apply_edits、search_assets 三个工具
2. **后端：draft_edits 表 + undo/redo API**
3. **后端：System prompt 固定化 + compaction 逻辑**
4. **后端：SSE 事件扩展（edit 类型）**
5. **前端：Inline diff 组件（TipTap 修订模式）**
6. **前端：增强型 Chat Panel（工具日志 + TODO）**
7. **前端：Undo/Redo 按钮**
