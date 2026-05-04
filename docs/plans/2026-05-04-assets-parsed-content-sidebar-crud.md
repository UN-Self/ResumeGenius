# Assets 素材正文落库、侧栏 CRUD 与 Parsing 清洗收口实施计划

> 本文是这条需求线的唯一主计划文档。  
> 已吸收并替代原来的：
> - `docs/plans/2026-05-04-assets-parsed-content-sidebar-crud.md`
> - `docs/plans/2026-05-02-parsing-cleanup-and-asset-sidebar-plan.md`

**目标：**

- 让 PDF / DOCX / Git / note 进入系统后，都能沉淀成可复用、可编辑、可供 AI 消费的素材正文
- 让编辑器左侧栏从“临时解析结果展示”升级为真正的素材管理区
- 统一 parsing / intake / workbench / agent 对素材正文的理解，避免字段语义再次分叉

**推荐最终方案：**

- 不新增 `parsed_content` 字段
- 直接复用 `assets.content` 作为所有素材的 canonical body
- parsing 负责“提取 -> 轻清洗 -> 回写 `assets.content` -> 补图片落库 -> 必要时删原文件”
- workbench / generate / 后续 agent 素材读取统一消费 `assets.content`

**当前实现状态：**

- Step 1 ~ Step 11 已完成
- Step 12 ~ Step 14 是 code review 后新增的收口项，后续逐个处理

---

## 1. 两份文档的差异结论

### 1.1 数据模型方案不同

你同事文档的核心假设是：

- 给 `assets` 新增 `parsed_content` 字段
- `content` 继续主要给 note 用
- PDF / DOCX / Git 的解析正文写入 `parsed_content`

我们现有文档与代码最终采用的是：

- 不新增 `parsed_content`
- 直接把 `assets.content` 统一升级为“素材正文”
- `note` 的 `content` 仍然是用户原文
- `resume_pdf / resume_docx / git_repo` 的 `content` 由 parsing 清洗后回写

**为什么合并后保留 `assets.content` 方案：**

1. 当前代码已经按这个方案实现完成  
   包括 parsing 回写正文、generate 优先消费持久化正文、左栏展示正文、通用正文编辑接口。

2. 这条链路对 workbench / generate / agent 更统一  
   不需要再区分“正文到底在 `content` 还是 `parsed_content`”。

3. 减少数据模型复杂度  
   如果同时存在 `content` 和 `parsed_content`，后续很容易继续出现：
   - note 改哪个
   - 文件资产改哪个
   - AI 读哪个
   - 前端预览哪个

### 1.2 关注范围不同

你同事文档主要聚焦：

- Asset 模型新增字段
- UpdateAsset 通用接口
- AssetList / AssetEditDialog / ProjectDetail CRUD 改造

我们现有文档覆盖范围更大，除了 CRUD 之外，还包括：

- parsing 文本轻清洗
- parse 结果落库
- 派生图片持久化
- 原始 PDF / DOCX 删除
- generate 优先消费持久化正文
- 编辑页左栏切换为 asset-driven
- 重新解析与脏状态提示
- 契约与 PR log 收口

**结论：**

你同事那份更像一份“围绕侧栏 CRUD 的任务实施单”；  
我们这份更像一份“围绕素材正文链路的系统性收口计划”。

### 1.3 API 设计不同

你同事文档建议：

- `PUT /assets/:asset_id`
- 更新 `label / content / parsed_content`

当前实现与现有文档采用的是：

- `PATCH /api/v1/assets/:asset_id`
- 更新 `label / content`
- 不存在 `parsed_content`
- 对 PDF / DOCX / Git 的正文人工修改会写入 `metadata.parsing.updated_by_user = true`

**为什么合并后保留 `PATCH + content`：**

1. `PATCH` 更符合“部分更新”语义  
2. 当前代码和测试已经按 `PATCH` 落地  
3. 也更符合“只修改素材正文和标题”的使用场景

### 1.4 前端入口设计不同

你同事文档主要落点在：

- `ProjectDetail.tsx`
- 通用编辑弹窗
- AssetList 对所有素材开放编辑

当前实际实现更进一步：

- 编辑页 `EditorPage` 已切到 `AssetSidebar`
- 左栏首屏直接读 `GET /assets`
- 支持上传、接入 Git、添加备注、删除、编辑、重新解析
- 对派生 `resume_image` 做过滤
- 对手改正文加脏状态提示

不过 code review 也指出了一个保留问题：

- 项目详情页入口目前仍是 `parseProject()` + `createDraft()`
- 还没有切到真正的 `/parsing/generate`

所以合并后会把这一点保留为后续 Step 13。

### 1.5 文档成熟度不同

你同事文档的优点：

- 任务拆分很细
- 接近“让 agent 一步步照着做”的执行手册
- TDD 节奏和 commit message 写得很直接

我们现有文档的优点：

- 先做了方案比较
- 把 issue #23 / #25 和同事口径都串起来了
- 风险、取舍、联调口径、Reviewer 视角更完整
- 已经回写了“哪些步骤完成了、当前真实代码行为是什么”

**合并策略：**

- 保留我们文档里的方案决策、行为口径和完成状态
- 保留你同事文档里“按任务拆分、可交给 AI 执行”的表达方式
- 最终形成一份“既能指导实现，又能对齐架构口径”的统一文档

---

## 2. 最终采用的统一设计

### 2.1 数据语义

| 字段 | 最终语义 |
|---|---|
| `assets.uri` | 原始来源。文件资产为上传路径，Git 为 repo URL，note 通常为空 |
| `assets.content` | 素材正文的统一落点。note 为用户原文；PDF / DOCX / Git 为 parsing 清洗后正文 |
| `assets.label` | 展示标题，可人工调整 |
| `assets.metadata` | 解析状态、派生图片、人工修改、源文件删除信息等元数据 |

明确不采用：

- `assets.parsed_content`

原因：

- 会与 `content` 语义重叠
- 会让前端、AI、后端继续分叉
- 当前分支已经走通 `assets.content` 方案

### 2.2 Parsing 目标行为

最终链路是：

1. 读取项目 assets
2. 对 PDF / DOCX / Git / note 做轻清洗
3. 将 PDF / DOCX / Git 的正文回写到 `assets.content`
4. 将解析图片持久化为派生 `resume_image`
5. 对已成功落库的 PDF / DOCX 删除原始文件
6. `generate` 优先消费持久化正文

### 2.3 Workbench 目标行为

编辑页左栏应基于 `assets` 工作，而不是临时 `parsed_contents`：

- 显示持久化素材正文预览
- 支持新增、删除、编辑
- 支持重新解析文件类素材
- 支持脏状态提示
- 过滤 parsing 派生图，避免误编辑

### 2.4 Agent 目标行为

当前 v1 主对话仍以 `drafts.html_content` 为主上下文。  
但如需读取项目素材，统一来源应是：

- `assets.content`

而不是：

- 原始 PDF / DOCX 文件
- Git 仓库原地址
- 一次性 `parsed_contents`

---

## 3. 你同事文档与现有实现的任务映射

### Task 1：Asset 模型新增 `parsed_content` 字段

**差异：**

- 你同事文档建议加字段
- 当前实现没有加

**合并结论：**

- 废弃此任务
- 用“统一 `assets.content` 语义 + parsing 回写正文”替代

**对应到现有计划：**

- Step 1：统一契约口径
- Step 3：把 parse 结果持久化到 assets

### Task 2：AssetService 新增 UpdateAsset

**差异：**

- 你同事文档想更新 `label / parsed_content / content`
- 当前实现更新的是 `label / content`
- 同时会记录 `updated_by_user`

**合并结论：**

- 保留通用 `UpdateAsset`
- 不保留 `parsed_content`
- 增加 metadata 行为，标记人工修改

**对应到现有计划：**

- Step 7：补一个通用资产正文更新接口

### Task 3：新增 `PUT /assets/:asset_id`

**差异：**

- 你同事文档用 `PUT`
- 当前实现用 `PATCH`

**合并结论：**

- 保留当前 `PATCH /api/v1/assets/:asset_id`

**对应到现有计划：**

- Step 7：补一个通用资产正文更新接口

### Task 4：前端 Asset 类型新增 `parsed_content`

**差异：**

- 你同事文档前端仍围绕 `parsed_content`
- 当前实现前端已经围绕 `content`

**合并结论：**

- 废弃 `parsed_content` 前端字段设计
- 统一让 `Asset.content` 成为前端素材正文字段

**对应到现有计划：**

- Step 1：统一契约
- Step 8：Editor 左栏切到 Asset 驱动

### Task 5 / Task 6 / Task 7：AssetList、AssetEditDialog、ProjectDetail 改造

**差异：**

- 你同事文档的目标是“在项目详情页先把编辑通了”
- 当前实现已经更进一步走到了编辑页左栏资产管理

**合并结论：**

- 保留“素材可编辑”的目标
- 但最终落点以 `EditorPage + AssetSidebar + AssetEditorDialog` 为主
- `ProjectDetail` 只保留项目入口和素材准备职责

**对应到现有计划：**

- Step 8：把 Editor 左侧栏从 ParsedContent 改成 Asset 驱动
- Step 9：补重新解析与脏状态提示

### Task 8：更新 intake 契约文档

**差异：**

- 你同事文档只改 intake contract
- 当前实现同时改了 parsing / intake / agent 三份契约

**合并结论：**

- 保留文档收口，但范围扩大到三份契约 + PR log

**对应到现有计划：**

- Step 10：文档、联调口径、回归测试收口

### Task 9：运行全部测试

**差异：**

- 你同事文档只写了“最后统一跑”
- 当前实现每个关键阶段都有测试补齐，并在 Step 10 汇总回归

**合并结论：**

- 保留统一回归动作
- 但在计划中按阶段穿插测试更合理

---

## 4. 合并后的实施状态

### 已完成

#### Step 1：统一契约口径（已完成）

- 把 `assets.content` 的语义从“note 专用”升级成“素材正文”
- intake / parsing / agent 契约已统一

#### Step 2：抽出共享清洗器（已完成）

- PDF / DOCX / Git / note 已接共享轻清洗
- 包含换行归一、空白压缩、bullet 规范、简单页码 / 分隔线移除

#### Step 3：把 parse 结果持久化到 assets（已完成）

- PDF / DOCX / Git 的清洗后正文已回写 `assets.content`
- `metadata.parsing.status` 已区分 `success / failed / skipped`

#### Step 4：给解析出的图片补持久化落点（已完成）

- 解析图片已落为派生 `resume_image`
- 主资产 `metadata.parsing.derived_image_asset_ids` 记录关联

#### Step 5：在持久化成功后安全删除原始文件（已完成）

- 正文和图片落库后，原始 PDF / DOCX 会被删除
- 删除标记写入 `metadata.parsing.source_deleted`
- 之后可从 `assets.content` 回退构造解析正文

#### Step 6：让 generate 优先消费持久化正文（已完成）

- `generate` 已优先用 `assets.content`
- 正文缺失时才回退触发 parse

#### Step 7：补一个通用资产正文更新接口（已完成）

- 已有 `PATCH /api/v1/assets/:asset_id`
- 支持通用更新 `label / content`
- 手动改正文会写 `metadata.parsing.updated_by_user = true`

#### Step 8：把 Editor 左侧栏从 ParsedContent 改成 Asset 驱动（已完成）

- 编辑页左栏首屏已直接加载 `assets`
- 新增 `AssetSidebar`、`AssetEditorDialog`
- 支持上传、接入 Git、添加备注、删除、编辑正文

#### Step 9：补“重新解析”与脏状态提示（已完成）

- 文件类素材支持重新解析
- 手改正文后会有覆盖确认
- 已展示“最近解析时间”和“已手动修改”提示

#### Step 10：文档、联调口径、回归测试收口（已完成）

- parsing / intake / agent 契约已回写
- PR log 已补齐
- 关键测试已通过

---

## 5. 合并后的后续修复清单

### Step 11：删除主素材时级联清理 parsing 派生图片（已完成）

问题：

- 当前删除主 PDF / DOCX 时，派生 `resume_image` 不会一起删

目标：

- 删除主素材时，同时删除派生图片资产和文件

完成情况：

- intake 的 `DeleteAsset(...)` 现在会读取主素材 `metadata.parsing.derived_image_asset_ids`
- 删除主 PDF / DOCX 时，会把关联的派生 `resume_image` 资产一并删除
- 派生图片文件也会随着删除一起清理
- 已补服务层测试，覆盖“删除主素材会清理派生图，但不会误删无关图片资产”

建议提交备注：

```bash
git commit -m "fix(intake): 删除主素材时级联清理派生图片"
```

### Step 12：收紧删除链路错误处理

问题：

- 当前删除接口会吞掉文件存储删除失败

目标：

- 不再出现“DB 删了但文件还在”的静默半成功状态

建议提交备注：

```bash
git commit -m "fix(intake): 收紧素材删除链路的存储错误处理"
```

### Step 13：前端入口切到真正的 `/parsing/generate`

问题：

- 当前项目详情页仍走 `parseProject()` + `createDraft()`
- 绕过了后端已经补好的 generate 路径

目标：

- 点击“开始解析”后直接生成带 HTML 的初稿并跳编辑页

建议提交备注：

```bash
git commit -m "feat(workbench): 将项目入口切换到 parsing generate"
```

### Step 14：清理死代码与重复 helper

问题：

- 当前 parsing 中 `derivedImageLabel(...)` 未使用

目标：

- 清理无引用 helper，减少维护歧义

建议提交备注：

```bash
git commit -m "refactor(parsing): 清理未使用的派生图片辅助函数"
```

---

## 6. 合并后的统一结论

如果只保留一句话，这两份文档合并后的结论是：

**这条线真正要做的，不是给 Asset 再加一个 `parsed_content` 字段，而是把 `assets.content` 升级成所有素材统一的正文落点，并围绕它完成 parsing 清洗、正文回写、图片落库、源文件删除、左栏 CRUD、重新解析和 AI 可消费链路收口。**

---

## 7. Reviewer / 同事快速阅读版

1. 你同事文档提出的问题方向是对的：素材解析后要落库，左栏要支持 CRUD。  
2. 但最终实现不采用 `parsed_content`，而是统一复用 `assets.content`。  
3. 当前这条链路已经完成到 Step 10，包括清洗、正文回写、图片落库、删原文件、左栏资产化、重新解析提示。  
4. 剩下的是 code review 后补的 4 个收口项，尤其是删除派生图和项目入口切 generate。  
5. 后续继续开发时，应以本合并版文档为准，不再单独沿用 `parsed_content` 方案。
