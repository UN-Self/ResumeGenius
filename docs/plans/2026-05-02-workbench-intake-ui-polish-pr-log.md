# Workbench — PR 开发日志（素材上传视觉优化、解析结果卡片统一、侧栏展开交互修复）

**类型：** 开发日志 / PR 说明  
**模块：** workbench  
**时间：** 2026-05-02  
**相关契约：** `docs/modules/workbench/contract.md`  
**相关设计：** `docs/plans/2026-04-30-workbench-ui-contract-design.md`  
**相关代码：** `frontend/workbench/src/components/intake/`、`frontend/workbench/src/pages/EditorPage.tsx`、`frontend/workbench/src/styles/editor.css`

---

## 1. 模块职责

本次改动仍属于 workbench 编辑页范围，主要覆盖两类职责：

1. 素材上传与素材预览  
   包括上传弹窗、左侧素材列表、解析结果卡片等“编辑前的上下文展示层”。

2. 编辑页三栏交互  
   包括左侧素材栏、中央 A4 编辑区、右侧 AI 助手栏的收起与展开体验。

这次 PR 不涉及后端接口协议调整，也不改变 parsing / agent 的业务边界，重点是把已有功能的前端呈现统一起来，并修掉实际使用中已经暴露出的可用性问题。

---

## 2. 本次 PR 背景

这次 PR 的出发点不是新增一条功能线，而是对编辑页素材区做一次前端收口，原因主要有三类：

1. 上传弹窗的信息表达不够直观  
   已选文件只显示原始文件名，缺少类型图标、颜色区分和更明确的文件状态反馈。

2. 左侧素材/解析结果的视觉语言不统一  
   文件素材、备注、解析结果分别使用不同的展示方式，存在 UUID 前缀直接暴露、文件后缀重复展示、备注正文重复标题等问题。

3. 左右侧栏收起后的可恢复性不足  
   展开按钮贴边且不够明显，收起后容易只剩一条细边，用户会误以为侧栏无法再打开。

---

## 3. 本次 PR 完成内容

### 3.1 抽离统一的文件视觉映射层

本次新增了统一的文件视觉工具文件：

- `frontend/workbench/src/components/intake/fileVisuals.ts`

统一收口了以下能力：

- 按类型映射图标
- 按类型映射配色
- 按类型映射徽标文案
- 提供文件名清洗能力
- 处理存储层 UUID 前缀剥离
- 提供文件大小格式化方法

当前覆盖的视觉类型包括：

- `PDF`
- `DOCX`
- `PNG`
- `JPG`
- `JPEG`
- `GIT`
- `备注`
- 通用 `FILE`

这样做的好处是上传弹窗、素材列表、解析结果卡片都可以复用同一套规则，避免每个组件单独写一份类型判断和样式分支。

### 3.2 重做上传弹窗的已选文件预览

本次对上传弹窗做了信息结构优化：

- 已选文件会展示对应的文件图标和颜色
- 文件名展示时不再重复显示后缀
- 文件类型通过右侧徽标展示
- 图片类文件统一使用绿色视觉，但 `PNG / JPG / JPEG` 会按真实后缀区分徽标
- 增加“点击此区域可重新选择文件”的提示文案
- 重新选择同一个文件时会先清空 input value，避免浏览器不触发 change

对应实现：

- `frontend/workbench/src/components/intake/UploadDialog.tsx`

这部分解决的是“用户已经知道当前选了什么类型文件、文件名的核心信息是什么、是否可以重新选择”的即时反馈问题。

### 3.3 统一解析结果卡片的标题、图标和正文展示

本次把左侧“解析结果”卡片改成了与上传弹窗一致的视觉体系。

主要变化：

- 文件类解析结果显示专属图标和颜色
- 文件标题自动去掉 UUID 存储前缀
- 文件标题自动去掉后缀，只保留核心名称
- 文件类型通过独立徽标展示，不再让标题本身承担类型信息
- 备注卡片显示统一的备注图标和标签
- 备注正文如果首行重复标题，会自动去重
- 调整了图标尺寸，并让“图标 / 标题 / 徽标”在同一行内垂直对齐

对应实现：

- `frontend/workbench/src/components/intake/ParsedItem.tsx`
- `frontend/workbench/src/components/intake/fileVisuals.ts`

这部分的目标不是“做复杂 UI”，而是让左侧解析结果更接近一个稳定可扫读的素材侧栏，而不是原始文本堆叠。

### 3.4 补齐素材列表的视觉一致性

除了“解析结果”，编辑页左侧素材列表也同步切到了同一套图标体系。

本次调整后：

- 不同素材类型拥有不同图标和配色
- 素材标签统一使用徽标样式
- 文件素材、Git 素材、备注素材的卡片信息层级更一致
- 空状态文案保持不变，但整体列表更接近同一个设计系统

对应实现：

- `frontend/workbench/src/components/intake/AssetList.tsx`

这一步的重点是把“上传中的文件”“已存在的素材”“解析后的内容”尽量放到同一套视觉语义下，降低用户切换理解成本。

### 3.5 修复左右侧栏收起后难以重新展开的问题

本次修复了编辑页左右侧栏的展开把手问题。

具体调整：

- 收起/展开按钮统一改为更明确的 `<` / `>`
- 给按钮补上 `type="button"`，避免表单环境下的默认提交风险
- 展开把手增大点击面积
- 展开把手位置向中央编辑区内收，不再贴边隐藏
- 提高 z-index，避免被中间内容遮挡
- 调整 hover / focus 态，使其在暖色设计体系下更容易被识别

对应实现：

- `frontend/workbench/src/pages/EditorPage.tsx`
- `frontend/workbench/src/styles/editor.css`

修复后的目标很明确：侧栏可以自由收起，但不能因为视觉太弱而让用户误判成“功能失效”。

### 3.6 补充回归测试

本次新增了两组前端测试，覆盖这次最容易回归的 UI 规则：

1. `UploadDialog.test.tsx`
   验证已选文件会显示清洗后的文件名和正确的类型徽标，并校验不支持格式时的报错文案。

2. `ParsedItem.test.tsx`
   验证解析结果卡片会去掉 UUID 前缀与多余后缀，并验证备注正文会去掉重复标题。

对应文件：

- `frontend/workbench/tests/UploadDialog.test.tsx`
- `frontend/workbench/tests/ParsedItem.test.tsx`

---

## 4. 当前前端对外表现

本次 PR 合入后，用户在编辑页会感知到以下变化：

1. 上传文件时  
   已选文件不再只是“文件名一行字”，而是包含图标、颜色、类型徽标、体积信息和重新选择提示。

2. 查看左侧解析结果时  
   文件类素材会显示更干净的标题，不再暴露 UUID 前缀，也不会同时在标题和后缀里重复表达类型。

3. 查看备注素材时  
   标题与正文的层级更清楚，正文不会重复首行标题。

4. 收起左右栏后  
   用户可以更容易找到并点击展开把手，不会再出现“收起后像是打不开了”的错觉。

---

## 5. 本次改动涉及的核心文件

前端实现：

- `frontend/workbench/src/components/intake/AssetList.tsx`
- `frontend/workbench/src/components/intake/ParsedItem.tsx`
- `frontend/workbench/src/components/intake/UploadDialog.tsx`
- `frontend/workbench/src/components/intake/fileVisuals.ts`
- `frontend/workbench/src/pages/EditorPage.tsx`
- `frontend/workbench/src/styles/editor.css`

测试：

- `frontend/workbench/tests/ParsedItem.test.tsx`
- `frontend/workbench/tests/UploadDialog.test.tsx`

---

## 6. 测试与验证

本次 PR 已完成以下验证：

- `bun install --frozen-lockfile`
- `bun run build`
- 本地访问 `http://127.0.0.1:3000/projects/2/edit` 返回 `200`
- 手工验证上传弹窗的文件图标、文件名清洗、类型徽标显示
- 手工验证左侧解析结果卡片的图标、标题、备注正文去重
- 手工验证左右侧栏收起后仍可重新展开

说明：

- 本次未通过 Docker 重新构建前端镜像完成最终验证，原因是当前环境的镜像源 `docker.1ms.run` 出现 DNS 解析失败
- 因此这次 UI 验证采用了本地 `bun + vite` 方式完成
- 后端与数据库仍保持 Docker 运行，不影响本次前端交互验证

---

## 7. 当前已知但不属于本 PR 范围的问题

以下问题与编辑页体验相关，但不属于本次 PR 的直接修改范围：

- 左右侧栏的展开/收起状态当前不会跨刷新持久化
- 左侧解析结果正文仍是纯文本预览，不包含结构化折叠或高亮
- Git 素材目前仍以文本信息为主，未单独设计仓库摘要卡片
- Docker 镜像源解析失败属于当前环境问题，不属于本次 workbench UI 改动本身

这些点可以在后续独立的 workbench 体验优化中继续收口，不建议与这次 PR 混在一起。

---

## 8. PR Reviewer 快速结论

如果 reviewer 只关心“这次 workbench PR 到底做了什么”，可以直接看下面这 5 条：

1. 上传弹窗现在会按文件类型显示图标、颜色和准确徽标，文件名也去掉了冗余后缀。
2. 左侧解析结果卡片统一接入文件视觉体系，并清理了 UUID 前缀、重复后缀和备注标题重复问题。
3. 素材列表与解析结果、上传弹窗使用了同一套视觉映射，整体前端风格更一致。
4. 左右侧栏收起后的展开把手已做可用性修复，避免“收起后打不开”的误判。
5. 本次补了前端测试，覆盖文件名清洗、徽标显示和备注正文去重等核心回归点。

---

## 9. PR 标题建议

建议标题：

`feat: 优化 Workbench 素材上传、解析展示与侧栏展开交互`

如果想更偏设计优化语气，也可以用：

`feat: polish workbench intake visuals and panel expand interactions`

---

## 10. PR 摘要建议

可直接用于 GitHub PR 描述：

```md
## 背景

这次 PR 主要是对 Workbench 编辑页的素材区做一次前端收口，重点解决三个问题：

- 上传弹窗缺少更直观的文件类型表达
- 左侧素材/解析结果的视觉语言不统一，且存在 UUID 前缀、重复后缀、备注标题重复等展示问题
- 左右侧栏收起后展开把手不明显，容易误判成“无法重新打开”

## 本次改动

- 抽离 `fileVisuals.ts`，统一文件图标、配色、类型徽标和文件名清洗逻辑
- 优化上传弹窗的已选文件预览，支持按类型展示图标和颜色，并去掉文件名冗余后缀
- 区分 `PNG / JPG / JPEG` 的真实后缀徽标，同时保持图片类统一视觉风格
- 优化左侧解析结果卡片，去掉 UUID 前缀和多余后缀，备注正文自动去重首行标题
- 统一左侧素材列表的图标和标签样式
- 修复左右侧栏收起后展开把手不明显的问题，提升可发现性和点击命中率
- 补充前端测试，覆盖上传弹窗和解析结果卡片的关键展示规则

## 验证

- `bun install --frozen-lockfile`
- `bun run build`
- 手工验证 `http://127.0.0.1:3000/projects/2/edit`

## 说明

- 本次属于前端 UI / UX 优化，不涉及后端接口协议调整
- Docker 镜像源 `docker.1ms.run` 当前解析失败，因此最终页面验证使用本地 `bun + vite` 完成
```
