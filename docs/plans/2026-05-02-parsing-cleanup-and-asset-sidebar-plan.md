# Parsing 清洗与素材侧栏收口 — 开发日志（面向 AI 可消费文本）

**类型：** 开发日志 / 分步实施计划  
**模块：** parsing + intake + workbench  
**时间：** 2026-05-02  
**相关 issue：**
- `#23` Gap: agent 的 `get_project_assets` tool 只能读到 note 文本，PDF/DOCX 解析结果未落库
- `#25` 工作台左侧栏素材增删改查
**相关代码：**
- `backend/internal/modules/parsing/`
- `backend/internal/modules/intake/`
- `frontend/workbench/src/components/intake/`
- `frontend/workbench/src/pages/EditorPage.tsx`

---

## 1. 这次要解决什么

这次不再从“预算控制”切入，而是回到你同事真正想要的方向：

1. 解析后的内容要先做清洗  
   不是把 PDF / DOCX 原样抽出来就交给 AI，而是先去掉噪声，让文本更短、更干净、更像可消费素材。

2. 清洗后的内容要落库  
   不能只存在 `/parsing/parse` 的 HTTP response 里。否则一旦请求结束，agent、左侧栏、后续编辑都读不到。

3. 在文字、图片都完成持久化之后，原始文件要可以删除  
   这样后续 AI、左侧栏、素材管理都围绕“入库后的解析产物”工作，而不是继续依赖 PDF / DOCX 原件。

4. Workbench 左侧栏要从“只展示临时解析结果”升级为“可管理素材”  
   上传、查看、编辑、删除都应该围绕同一份持久化素材工作，而不是一边看 `parsed_contents`，另一边改 `assets`。

一句话概括：  
**把 parsing 产物从“临时预览”改成“可复用、可编辑、适合 AI 的素材正文”。**

---

## 2. 当前现状与两个 issue 的关系

### 2.1 issue #23 为什么成立

当前 `Asset` 模型只有：

- `uri`
- `content`
- `label`
- `metadata`

其中：

- `note` 的正文直接存在 `assets.content`
- `resume_pdf` / `resume_docx` / `git_repo` 只有 `uri`

这意味着：

- `GET /assets` 能稳定拿到 note 文本
- 但拿不到 PDF / DOCX / Git 解析后的正文

而 parsing 当前的行为是：

- 从 `assets` 读项目素材
- 解析成 `[]ParsedContent`
- 返回给 `/parsing/parse`
- 生成 draft 时再聚合送给 AI
- **没有把解析结果写回 `assets`**

所以 issue #23 说的“解析结果未落库”是准确的。

### 2.2 issue #25 为什么会被卡住

Workbench 编辑页左侧栏当前读取的是 `parsed_contents`，不是 `assets`。

这导致它天然有两个问题：

1. 它依赖重新调用 `/parsing/parse` 才能看到内容  
   页面一刷，左侧栏的数据来源又变回“临时结果”。

2. 它没有和 intake 的资产增删改查统一  
   `ProjectDetail` 页面已经有 `createNote / updateNote / deleteAsset / uploadFile` 等能力，但 `EditorPage` 左栏没有真正接过去。

所以 issue #25 其实不是纯前端问题，它依赖一个前提：

**左侧栏里展示的内容，必须先变成“资产自己的持久化正文”。**

### 2.3 你同事说的“清洗”为什么关键

当前 PDF / DOCX 的 normalize 只做了很轻的处理：

- 统一换行
- `TrimSpace`
- 合并连续空行

这对“能看”够了，但对“给 AI 用”还不够。典型噪声包括：

- PDF 被硬换行切碎的句子
- 多余空格、tab、空白段
- 重复页眉页脚、页码
- 纯装饰性的分隔线
- DOCX 导出的连续空行和段落碎片

所以这次应该把重点放在：

**解析后清洗，而不是 prompt 截断。**

---

## 3. 可改进方向

这里先列三个方向，再给出推荐方案。

### 方向 A：复用 `assets.content` 作为所有素材的“AI 可消费正文”

做法：

- `note`：继续把用户文本放在 `assets.content`
- `resume_pdf` / `resume_docx` / `git_repo`：把清洗后的解析文本也写入 `assets.content`
- `uri` 继续保留，作为原始来源和文件定位
- `metadata` 补充解析状态、来源、清洗信息、图片摘要

优点：

- 不需要新增表
- 不需要引入一套新的“parsed asset”读取模型
- `GET /assets` 天然就能被 agent、Workbench 左栏、后续 API 复用
- 前后端心智最统一：所有资产都有“给 AI 用的正文”

缺点：

- `content` 的语义会从“note 专用正文”升级成“资产正文”
- 契约文档和前端类型都要跟着改

### 方向 B：给 `assets` 新增 `parsed_content` 字段

做法：

- 保留 `content` 只给 note 用
- 新增 `parsed_content` 专门存解析结果

优点：

- 语义更直观

缺点：

- 需要迁移
- 前后端会同时面对 `content` 和 `parsed_content` 两套文本来源
- 左侧栏、agent、generate 都要多一层分支判断

### 方向 C：新建独立 `parsed_assets` / `asset_snapshots` 表

做法：

- 原始 `assets` 只存来源
- 解析后内容存新表

优点：

- 数据层最干净

缺点：

- 明显超出这轮范围
- 会把 workbench / agent / parsing 的联动复杂度一起抬高

### 推荐方案

**推荐采用方向 A：复用 `assets.content`，把它升级为所有素材的 AI 可消费正文。**

原因：

1. 最贴近 issue #23 的症结  
   当前缺的不是“更多表”，而是“文件资产没有正文落点”。

2. 最容易支撑 issue #25  
   一旦 `assets.content` 对所有素材都成立，左侧栏就能直接基于 `assets` 做展示、编辑、删除。

3. 最符合你同事的要求  
   他要的是“解析后的清洗结果”，不是“原始提取大文本”。

---

## 4. 这轮建议的目标状态

### 4.1 数据语义

调整后的 `assets` 语义建议如下：

| 字段 | 语义 |
|---|---|
| `uri` | 原始来源。文件资产保留文件路径，Git 保留 repo URL，note 可为空 |
| `content` | 给 AI / agent / 左侧栏消费的正文文本。note 为用户原文，文件/Git 为清洗后的解析文本 |
| `label` | 展示标题，可人工编辑 |
| `metadata` | 解析状态、原始文件名、清洗信息、图片摘要等附加信息 |

建议在 `metadata` 里增加一个轻量结构：

```json
{
  "parsing": {
    "status": "success",
    "source_kind": "parsed_file",
    "cleaned": true,
    "clean_rules": ["trim_blank_lines", "collapse_spaces", "merge_wrapped_lines"],
    "image_count": 1
  }
}
```

### 4.2 图片落点与源文件删除前提

如果要满足“文字、图片入库后原文件可删”，光把文本写进 `assets.content` 还不够。  
图片也必须有自己的持久化落点。

建议方向：

- `resume_pdf` / `resume_docx` 资产本身：保留一条主资产记录
- 清洗后的正文：写入主资产 `content`
- 解析出的图片：落成独立的 `resume_image` 资产，或至少落成可追踪的独立存储对象
- `metadata`：记录这些图片与主资产的来源关系

建议的最小关系模型：

```json
{
  "parsing": {
    "status": "success",
    "cleaned": true,
    "derived_image_asset_ids": [101, 102],
    "source_deleted": false
  }
}
```

只有在下面两个条件同时满足时，才允许删原文件：

1. 清洗后的正文已经成功写入数据库
2. 需要保留的图片也已经成功落库或另存为独立对象

### 4.3 parsing 行为

`POST /api/v1/parsing/parse` 建议升级为：

1. 读取项目资产
2. 解析支持的资产
3. 对提取文本做清洗
4. 将清洗后的文本写回 `assets.content`
5. 返回预览结果

这样它依然可以作为“解析预览接口”，但不再是“一次性结果”。

### 4.4 generate 行为

`POST /api/v1/parsing/generate` 建议优先使用已持久化的 `assets.content`：

- 如果资产已有清洗后正文，直接聚合
- 如果正文缺失，再按需触发 parse / refresh

这样能避免每次生成前都重复解析文件。

### 4.5 Workbench 左侧栏行为

左侧栏不再只消费 `parsed_contents`，而是改成以 `assets` 为主：

- 展示所有资产
- 展示每个资产的 `content` 预览
- 允许新增
- 允许删除
- 允许编辑正文
- 文件类资产可编辑“清洗后的文本”，不是编辑原 PDF / DOCX

这正好承接 issue #25 里的“编辑、删除、新增”需求。

---

## 5. 清洗策略建议

这部分是这轮的核心，不追求“摘要化”，先追求“去噪后可读、可喂给 AI”。

### 5.1 通用清洗规则

适用于 PDF / DOCX / Git：

1. 统一换行符  
   `\r\n` / `\r` 全部转为 `\n`

2. 清理首尾空白和多余空行  
   连续空行压成最多一行

3. 合并多余空格 / tab  
   避免出现“单词之间一大串空格”“整段只剩 tab”的情况

4. 规范项目符号  
   把异常 bullet、破折号、符号列表归一到稳定形态

5. 去掉无意义分隔  
   如纯由 `----`、`====`、`_____` 组成的装饰线

### 5.2 PDF / DOCX 定向规则

建议优先做这些轻量但收益高的规则：

1. 去掉重复页眉页脚  
   如果某一行在多页重复出现，且像页眉/页脚，清洗时去掉

2. 去掉纯页码行  
   如 `1 / 2`、`Page 2`、单独数字页码

3. 适度合并硬换行  
   对明显是同一句被截断的场景，合并成一行  
   但不要把 bullet list、时间线、标题段落全揉成一大段

4. 压缩连续空白段  
   保留段落边界，但避免一页简历被导成大量空段

### 5.3 Git 定向规则

Git 素材本来就不是原文档，建议继续坚持“摘要优先”：

1. 保留仓库名
2. 保留 README 关键段
3. 保留技术栈
4. 保留顶层结构摘要
5. 不把整个 README 或代码片段全文灌进去

### 5.4 Note 规则

note 不做“重写”，只做轻清洗：

1. 去掉首尾空白
2. 压缩连续空行
3. 保留用户原意和换段

### 5.5 本轮明确不做

这轮不做下面这些更重的事：

- OCR
- 大模型摘要 / 改写
- token 预算截断
- 语义去重

说明：

- “删除原始 PDF / DOCX 文件”现在已经是新增需求，因此会进入本计划
- 但不会在第一刀就做，而是放在“正文 / 图片持久化”之后单独实现

---

## 6. 分步实施计划

下面按“你之后一个个改”的节奏拆开，每一步都尽量单一、可提交。

### Step 1：先统一契约口径（已完成）

目标：

- 把 `assets.content` 的语义从“note 专用”升级成“资产正文”
- 明确 `content` 对 agent / workbench / generate 都是主消费字段

需要改的地方：

- `docs/modules/intake/contract.md`
- `docs/modules/parsing/contract.md`
- `docs/modules/agent/contract.md`
- 前端 `api-client.ts` 的 `Asset` 类型注释

验收：

- 文档里不再写“只有 note 才有 content”
- issue #23 的根因在契约层有明确解释和收口方案

建议提交备注：

```bash
git commit -m "docs: 统一 assets.content 为素材正文契约"
```

### Step 2：抽出共享清洗器（已完成）

目标：

- 把 PDF / DOCX 当前分散的 normalize 收口成共享 cleaner
- 先实现轻量去噪，不做摘要

建议新增：

- `backend/internal/modules/parsing/text_cleaner.go`
- `backend/internal/modules/parsing/text_cleaner_test.go`

优先规则：

- 统一换行
- Trim
- 压缩连续空行
- 压缩多余空格 / tab
- 去掉纯分隔线
- 规范 bullet
- 识别简单页码行

验收：

- PDF / DOCX / note / git 都能共用清洗入口
- 单测覆盖中英文、bullet、空白、页码、分隔线

建议提交备注：

```bash
git commit -m "feat(parsing): 新增共享文本清洗器"
```

### Step 3：把 parse 结果持久化到 assets

目标：

- `parse` 不再只返回临时结果
- 对可解析资产，把清洗后的正文写回 `assets.content`

建议实现：

- `ParsingService.Parse` 内部在成功解析后更新对应 `asset`
- `metadata.parsing.status` 至少区分 `success` / `failed` / `skipped`
- `metadata.parsing.image_count` 可以先落轻量摘要

建议文件：

- `backend/internal/modules/parsing/service.go`
- `backend/internal/shared/models/models.go`（如果只复用现有字段，可不改结构体）
- `backend/internal/modules/parsing/service_test.go`

验收：

- PDF / DOCX / Git parse 后，数据库里的 `assets.content` 不再为空
- note 资产行为不被破坏
- `/parse` 仍然返回 `parsed_contents`

建议提交备注：

```bash
git commit -m "feat(parsing): 持久化清洗后的素材正文"
```

### Step 4：给解析出的图片补持久化落点

目标：

- 不让图片只存在 `ParsedContent.Images` 的临时返回里
- 为后续删原文件建立前提条件

建议实现：

- 对 PDF / DOCX 提取出的图片，落成独立 `resume_image` 资产
- 在主资产 `metadata.parsing.derived_image_asset_ids` 里记录关联
- 图片摘要信息可继续保留在 `ParsedContent.Images` 里做预览

建议文件：

- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/pdf_parser.go`
- 如有需要，补 `intake/storage` 相关写入能力
- `backend/internal/modules/parsing/service_test.go`

验收：

- 解析后图片不再只是响应内临时数据
- 后续即使删掉 PDF / DOCX 原件，图片仍可被左侧栏或 AI 侧消费

建议提交备注：

```bash
git commit -m "feat(parsing): 持久化解析出的素材图片"
```

### Step 5：在持久化成功后安全删除原始文件

目标：

- 对 `resume_pdf` / `resume_docx`，在解析成功且正文/图片都已落点后，删除原始上传文件

建议实现：

- 只对文件资产做这一步，`note` / `git_repo` 不涉及
- 删除前必须检查：
  - `assets.content` 已写入
  - 派生图片已完成持久化（如果有）
- 删除成功后：
  - 调用存储层删除原文件
  - 清空或废弃 `uri`
  - 在 `metadata.parsing.source_deleted = true` 中留下标记
  - 保留 `original_filename` / `original_asset_type` 供展示和排查

建议文件：

- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/intake/service.go` 或共享存储接口
- `backend/internal/modules/parsing/service_test.go`

验收：

- parse 成功后，原 PDF / DOCX 文件会被删除
- 但左侧栏和 AI 仍可通过入库正文/图片正常工作

建议提交备注：

```bash
git commit -m "feat(parsing): 解析入库后删除原始文件"
```

### Step 6：让 generate 优先消费持久化正文

目标：

- `generate` 优先使用 `assets.content`
- 缺正文时再决定是否补 parse

建议实现：

- 聚合逻辑从“聚合 `ParsedContent[]`”切到“聚合资产正文”
- 如果某个资产没有正文且可解析，再触发一次 parse fallback

好处：

- 避免每次 generate 都重跑一遍文件解析
- agent、generate、左侧栏统一消费同一份正文

建议文件：

- `backend/internal/modules/parsing/service.go`
- `backend/internal/modules/parsing/service_test.go`

验收：

- 已 parse 过的项目再次 generate 时，不依赖重新提取文件正文
- issue #23 里“解析即丢弃”的链路被打断

建议提交备注：

```bash
git commit -m "feat(parsing): 让生成链路优先消费持久化正文"
```

### Step 7：补一个通用资产正文更新接口

目标：

- 不再只有 note 能编辑
- 文件 / Git / note 的“清洗后正文”都可在左侧栏修改

建议新增接口：

- `PATCH /api/v1/assets/:asset_id`

建议支持字段：

- `label`
- `content`

不支持字段：

- `uri`
- `type`

建议文件：

- `backend/internal/modules/intake/handler.go`
- `backend/internal/modules/intake/service.go`
- `backend/internal/modules/intake/routes.go`
- 对应 tests

验收：

- note 仍可正常编辑
- PDF / DOCX / Git 资产的正文也可人工修正

建议提交备注：

```bash
git commit -m "feat(intake): 支持通用资产正文编辑"
```

### Step 8：把 Editor 左侧栏从 ParsedContent 改成 Asset 驱动

目标：

- 左侧栏不再只展示临时 parse 结果
- 改成真正的素材管理区

建议调整：

- `EditorPage` 加载 `assets`
- 左栏展示 `assets.content`
- 继续保留上传入口
- 增加删除入口
- 增加编辑入口
- note 和文件资产共用同一套正文编辑体验

建议文件：

- `frontend/workbench/src/lib/api-client.ts`
- `frontend/workbench/src/pages/EditorPage.tsx`
- `frontend/workbench/src/components/intake/ParsedSidebar.tsx`
- 可视情况拆新组件，如 `AssetSidebar.tsx` / `AssetEditorDialog.tsx`

验收：

- issue #25 的“新增、删除、编辑”在编辑页左侧可走通
- 页面刷新后仍能看到上次 parse 后的正文

建议提交备注：

```bash
git commit -m "feat(workbench): 将编辑页左侧栏升级为素材管理区"
```

### Step 9：补“重新解析”与脏状态提示

目标：

- 当用户重新上传文件或想刷新提取结果时，有明确入口
- 避免人工编辑过的正文被静默覆盖

建议行为：

- 左栏给文件资产一个“重新解析”动作
- 如果该资产 `content` 被人工修改过，重新解析前给确认提示
- `metadata.parsing` 可记录 `updated_by_user` / `last_parsed_at`

验收：

- 用户能区分“AI 可读正文是解析生成的”还是“我手改过的”

建议提交备注：

```bash
git commit -m "feat(workbench): 增加素材重新解析与覆盖确认"
```

### Step 10：文档、联调口径、回归测试收口

目标：

- 收口所有契约和测试，避免模块理解再分叉

需要更新：

- `docs/modules/parsing/contract.md`
- `docs/modules/intake/contract.md`
- `docs/modules/agent/contract.md`
- 如有需要，补一份 PR log

回归重点：

- `parse` 后 `assets.content` 已持久化
- `generate` 能直接消费持久化正文
- 左栏编辑后刷新页面不丢
- note / PDF / DOCX / Git 行为一致

建议提交备注：

```bash
git commit -m "docs: 收口 parsing 素材正文链路契约"
```

---

## 7. 风险与取舍

### 风险 1：把 `content` 复用给所有资产，会不会语义变混

会有一点，但这是可控的。  
关键不是字段名是否完美，而是全链路是否统一。

只要文档写清：

- `content` = 资产正文
- note 的正文来自用户输入
- 文件 / Git 的正文来自 parsing 清洗结果

那这个字段就是有价值的统一抽象。

### 风险 2：人工修改过的正文会不会被重新 parse 覆盖

会，所以 Step 7 必须补“覆盖确认”或“脏状态提示”。

### 风险 3：图片是否要同步入库

如果源文件要删，那图片就不能只做“轻量摘要”。  
至少要有一个可追踪的持久化落点，否则删掉原 PDF / DOCX 后，图片就丢了。

建议：

- 不把完整 base64 直接塞进主资产 `content`
- 优先落成独立 `resume_image` 资产或独立存储对象
- 主资产只保留关联关系和摘要信息

### 风险 4：删除原始文件后的可回溯性

一旦删掉原文件，后续排查和重跑解析会受影响。  
所以删除动作必须满足两个条件：

- 解析正文与图片都已稳定持久化
- `metadata` 里保留必要的来源信息和删除标记

---

## 8. Reviewer 快速结论

如果 reviewer 只想快速知道这份计划的核心，请看这 6 条：

1. 这次不做预算控制，改做“解析后清洗 + 正文落库”。
2. 推荐直接复用 `assets.content`，把它升级成所有素材的 AI 可消费正文。
3. `parse` 要从“返回临时结果”升级成“写回资产正文并返回预览”。
4. `generate`、agent、Workbench 左侧栏都应该统一消费这份正文。
5. 左侧栏后续可以基于 `assets` 实现新增、删除、编辑，而不再依赖易失的 `parsed_contents`。
6. “源文件删除”已经纳入计划，但会放在正文/图片持久化之后单独完成。

---

## 9. 本轮推荐的第一刀

如果只做第一步，我建议先从 **Step 1 + Step 2** 开始：

1. 统一契约：明确 `assets.content` 是所有素材的正文落点
2. 抽共享清洗器：先把文本质量收口

这样后面的“落库”和“左侧栏编辑”就会顺很多，也最符合你同事现在强调的方向。
