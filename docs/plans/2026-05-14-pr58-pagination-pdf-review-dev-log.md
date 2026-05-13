# PR #58 分页与 PDF 排版同步排查开发日志

**类型：** 开发日志 / Review 问题整理  
**模块：** workbench / smart-split / render  
**时间：** 2026-05-14  
**当前分支：** `feat/pr58-pagination-pdf-fixes`  
**关联 PR：** `#58 fix: 分页多页渲染与 PDF 排版同步`  
**同步状态：** 已重新拉取 PR patch，当前仍为 14 个提交；本地已应用到 `bfd669f fix: 保留元数据事务中的 ownDispatch 状态`  
**本轮修复状态：** 已新增 4 个修复提交，优先处理 WYSIWYG 导出链路与 SmartSplit 结构性风险  

本轮新增提交：
- `fix: 修复 PDF 渲染外壳样式作用域`
- `fix: 修复导出前保存最新画布`
- `fix: 收敛智能分页拆分循环`
- `fix: 修复 PDF 字体嵌入编码`

---

## 1. 未完成与新增问题（优先处理）

### 1.1 部分修复：循环拆分仍需手动验收

**状态：本轮已收敛，但仍需手动验收。**

同事反馈“循环拆分其实没修出来”是合理风险。当前 PR 确实补了 `suppressCount` 和 `ownDispatch` 守卫，但这只能降低由自身 dispatch、PaginationPlus meta-only 更新、trailing node 等触发的级联概率，还没有证明 SmartSplit 在真实 DOM 分页重排下是幂等的。

当前代码里还有几个循环风险点：
- `performDetectionAndSplit` 在完成一次 `view.dispatch(tr)` 后，同一轮内继续调用 `syncPageBreaks`，但此时 DOM 分页边界可能还没稳定。
- `syncPageBreaks` 会先清理所有旧 `break-before`，再按当前 pageStarts 重新写入；即使最终目标没有变化，也可能制造一次 `docChanged` 事务。
- 当前测试只证明单次拆分/同步函数能跑通，没有覆盖“同一份文档在分页稳定后不会继续 dispatch”的幂等场景。

建议：
- 本轮已把 `syncPageBreaks` 改为差异更新，目标未变化时不再重复 dispatch。
- 本轮已将 split 与 page-break sync 拆成两轮：发生 split 后先 return，等待后续分页稳定。
- 仍建议增加更完整的浏览器级循环回归测试，确认真实 PaginationPlus 环境下不会持续 dispatch。

### 1.2 部分修复：两栏 / flex 容器拆分会破坏布局

**状态：本轮先阻止顶层 `.resume` 被拆；两栏内部 flex 感知仍待后续验证。**

同事指出的问题成立：当 SmartSplit 检测到 `.right-col` 跨越分页边界时，当前 `buildSplitTransaction` 会按 ProseMirror 位置拆它的父容器。若父容器是 `.main` 这类 `display: flex` 双栏容器，拆分后 `.left-col` 与 `.right-col` 会被放进两个独立的 `.main` 中，原本并排的两栏布局会变成上下堆叠，页面高度随之放大，继续诱发错误分页。

当前新增提交只缓解了级联循环和重复触发，并没有解决 flex 容器被语义错误拆开的根因。

涉及位置：
- `frontend/workbench/src/components/editor/extensions/smart-split/splitTransaction.ts`
- `frontend/workbench/src/components/editor/extensions/smart-split/detectCrossings.ts`

建议：
- 本轮已禁止 `buildSplitTransaction` 拆分顶层 `.resume/.resume-document` 根容器，避免再次生成多个同级 `.resume`。
- 后续仍建议为 `.main > .left-col + .right-col` 场景补测试，并继续完善 flex/grid 内部拆分页选择。

### 1.3 已修复：导出前没有强制保存当前编辑器内容

**状态：本轮已修复。**

当前 `handleExport` 直接调用 `exportPdf(Number(draftId))`，没有先 `await flush()`。后端导出任务从数据库读取 `drafts.html_content`，如果用户刚编辑完就立即导出，PDF 可能拿到的是上一次 autosave 成功的旧 HTML。

涉及位置：
- `frontend/workbench/src/pages/EditorPage.tsx`
- `frontend/workbench/src/hooks/useAutoSave.ts`
- `backend/internal/modules/render/exporter.go`

建议：
- `handleExport` 已改为异步流程：先 `await flush()`，再主动 PUT 当前 `getPersistableHTML()`，成功后才创建导出任务。
- `flush()` 已补充等待 active save 的逻辑，避免正在保存旧内容时导出读到旧库。
- 保存失败时会阻止导出并提示“导出前保存失败”。

### 1.4 部分修复：AI 原始样式与编辑器 scoped CSS / PDF 导出链路不一致

**状态：本轮已修 PDF render shell；AI 样式规范化契约仍需后续收口。**

当前链路并不是简单地“同一份 HTML 在编辑器和 PDF 里渲染”：

1. AI 通过 `apply_edits` 直接修改数据库里的 `drafts.html_content`，可能写入完整 HTML、`<style>`、`@page`、`@media print`、root 容器尺寸、全局选择器等。
2. 前端 `applyHtmlToEditor` 会调用 `extractStyles(html)`，把 `<style>` 抽出来并改写成 scoped CSS，然后注入到 `A4Canvas`。
3. `extractStyles` 会跳过 `@page` 和 `@media print`，并移除 root 容器的 width / height / padding / margin 等尺寸属性。
4. 但 `applyHtmlToEditor` 使用 `restoringContent.current` 抑制自动保存，AI 原始 HTML 通常不会立刻被规范化后写回数据库。
5. PDF 导出后端从数据库读取 `drafts.html_content`，`wrapWithTemplate` 会把其中的 `<style>` 原样提取并注入 PDF 模板；这可能是未经前端 scoped/strip 处理的 AI 原始 CSS。

因此会出现一个很危险的分叉：
- 编辑器看到的是“处理后的 scoped CSS”。
- PDF 导出用的可能是“数据库里的 AI 原始 CSS”。

如果 AI 写了 `@page`、`@media print`、`.resume { width: 210mm; min-height: 297mm; padding: ... }`、`position`、或者全局 `body/html/*` 样式，编辑器可能显示不出来或被剥离，但 PDF 仍可能吃到这些样式，导致分页、空白、两栏布局和字距全部失真。

本次 `MEOwj1` 截图对应的数据库证据：
- 当前草稿 CSS 中存在 `.resume-document { display: flex; justify-content: center; align-items: flex-start; padding: 40px 20px; ... }`。
- PDF 模板当前使用 `<div class="resume-page resume-document">{{CONTENT}}</div>`，因此 AI/编辑器样式会直接命中 PDF 页面根容器。
- 当前草稿正文已被 SmartSplit 拆成多个兄弟 `.resume` 节点，并且最前面存在一个独立的 `<h1>陈子俊</h1>`。
- 编辑器中内容外面还有 ProseMirror / EditorContent 包装层，`.resume-document` 的 flex 直接子元素较少；PDF 中正文节点直接挂在 `.resume-page.resume-document` 下，flex 会把 `<h1>`、多个 `.resume`、空段落等兄弟节点横向排列，形成截图里的左右错位和内容重排。

这个问题比“导出前是否 flush”更直接：即使数据库是最新的，只要 PDF render shell 和编辑器 render shell DOM 结构不同，同一份 CSS 也会产生不同布局。

涉及位置：
- `frontend/workbench/src/pages/EditorPage.tsx`
- `frontend/workbench/src/lib/extract-styles.ts`
- `frontend/workbench/src/components/editor/A4Canvas.tsx`
- `backend/internal/modules/agent/tool_executor.go`
- `backend/internal/modules/render/exporter.go`

建议：
- 明确“数据库里的草稿 HTML”必须是编辑器和 PDF 共用的规范化版本，不能让 AI 原始 CSS 直接绕过编辑器规范化进入 PDF。
- 本轮导出前已强制保存当前画布对应的 persistable HTML。
- 后端导出侧也应增加防线：拒绝或清理草稿中的 `@page`、`@media print`、root 尺寸、`position: fixed/absolute` 等破坏 A4 模型的规则。
- 本轮已让 PDF 模板模拟编辑器 DOM 包装结构，不再把正文直接作为 `.resume-document` 的直接子节点渲染，避免 `.resume-document { display:flex }` 命中 PDF 页面根外壳。
- 后续仍建议把 AI apply 后的 normalized HTML 立即写回数据库，作为更彻底的契约收口。

### 1.5 未完成：PDF 与编辑器仍存在明显 WYSIWYG 偏离

**状态：未完成，需要手动验收。**

用户手动截图显示：编辑器里内容在 A4 画布内较紧凑连续，但导出的 PDF 被拆成多页，页面内出现大量异常空白，标题、联系方式、分割线、教育经历、技术栈等位置和编辑器明显不一致。

这已经不是单一的“字间距略小”问题，而是导出链路整体未满足所见即所得。

需要继续验证：
- 导出源 HTML 是否为最新内容。
- SmartSplit 是否错误拆开 flex 双栏容器。
- PDF 模板页面模型是否与编辑器 A4 画布一致。
- 字体落点是否一致。

### 1.6 已修复：Inter 字体嵌入疑似没有真正 base64 编码

**状态：本轮已修复。**

`interFontFaceCSS()` 注释写的是生成 base64 WOFF2，但当前实现把 `[]byte` 直接用 `%s` 塞进 `data:font/woff2;base64,...`，这不等同于 base64 编码。

涉及位置：
- `backend/internal/modules/render/exporter.go`

建议：
- 已使用 `base64.StdEncoding.EncodeToString(...)` 生成真正的 base64 字符串。
- 已增加测试校验 3 个字体 data URL 都可 decode，且 WOFF2 文件头为 `wOF2`。
- 仍建议手动导出后用 PDF 查看器确认实际嵌入效果。

### 1.7 未完成：中文字体在编辑器与 PDF 环境可能不同

**状态：未完成，需要确认实际字体落点。**

当前字体栈为：

```css
"Inter", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif
```

本地 Windows 编辑器可能落到 `Microsoft YaHei`，而 PDF 渲染环境更可能落到 `Noto Sans CJK SC`。同样字号和行高下，两种中文字体的字面宽度不同，会表现为 PDF 字距更紧或更松，并进一步影响换行与分页。

建议：
- 明确产品基准中文字体。
- 编辑器和 PDF 使用同一份可分发字体，或至少确认两端实际 `fc-match` / 浏览器 computed font 一致。

### 1.8 新增：缺少双栏分页与循环稳定性回归测试

**状态：未完成。**

当前 SmartSplit 测试覆盖了边界、列表、跨页父元素跳过、`pos=0` 等场景，但缺少两类关键回归：
- 两栏 flex 简历结构的直接回归测试。
- 连续检测多轮后不再 dispatch 的循环稳定性测试。

这两类测试应该成为下一轮修复的入口，否则修复很容易只解决单点函数，不解决真实编辑器中的分页时序问题。

---

## 2. 建议解决方案（待采纳）

### 2.1 推荐方案 A：先统一“编辑器所见”和“PDF 所导”的 HTML/CSS 契约

**目标：先修导出错乱的最大链路分叉。**

建议把数据库里的 `drafts.html_content` 定义为规范化后的可渲染 HTML，而不是 AI 原始输出。

落地步骤：
1. 抽出一个前端函数，例如 `normalizeDraftHtmlForCanvas(html)`，返回：
   - `bodyHtml`
   - `scopedCSS`
   - `persistableHTML`
2. `applyHtmlToEditor` 使用同一份 normalized 结果更新编辑器和 `A4Canvas`。
3. AI apply 成功后，前端不要只 `setContent`，还要把 `persistableHTML` 立即 PUT 回 `/drafts/:id`。
4. `handleExport` 改成 `await flush()` 后再导出。
5. 调整 PDF render shell，让它和编辑器 DOM 结构一致，例如页面几何外壳与用户样式作用域分离：

```html
<div class="resume-page">
  <div class="resume-document">
    <div class="resume-content ProseMirror">
      {{CONTENT}}
    </div>
  </div>
</div>
```

6. 后端 `wrapWithTemplate` 只接受规范化后的 scoped CSS；对 `@page`、`@media print`、root 尺寸、fixed/absolute 定位加测试或清理。
7. 将 `body/html` 规则的作用目标重新评估：不要让 AI 的网页级 `body { display:flex; padding... }` 直接覆盖 PDF 页面根容器。

优点：
- 编辑器显示、数据库内容、PDF 导出三者开始共享同一份样式语义。
- 能解释并解决“AI 样式编辑器显示不出来，但 PDF 导出乱”的问题。
- 能解决本次 `MEOwj1` 里 `.resume-document display:flex` 命中 PDF 根容器导致的横向错排。
- 改动比重写分页算法小，收益最大。

风险：
- 需要小心 React state 异步问题，不能在 `setScopedCSS(nextScopedCSS)` 后立刻用旧 state 拼保存内容；应直接用 normalize 函数返回的 `persistableHTML` 保存。

### 2.2 推荐方案 B：SmartSplit 改成幂等的两阶段流程

**目标：修循环拆分。**

建议把当前“检测 -> split -> 立刻 syncPageBreaks”的流程改成稳定的两阶段：

1. 第一阶段只做跨页拆分。
   - 如果本轮发生 split，立刻 return。
   - 不在同一轮继续 syncPageBreaks。
   - 等下一轮 DOM / PaginationPlus breaker 稳定后再判断。

2. 第二阶段只做 page-break 同步。
   - 先根据当前 DOM 算出目标 pageStart positions。
   - 读取文档中已有 `break-before` 位置。
   - 只有目标集合和现有集合不同才 dispatch。
   - 不再“先删全部再加全部”。

3. 增加循环保护。
   - 记录最近一次处理的 doc fingerprint / pageStart fingerprint。
   - 同一 fingerprint 下不重复 dispatch。
   - 单轮最多一次 split，最多一次 sync。

验收标准：
- 同一份文档连续触发检测 3 次，后两次不得产生 docChanged transaction。
- 开启 PaginationPlus 后手动编辑一处跨页内容，不出现持续插入空段落、重复 break-before 或页数持续增长。

### 2.3 推荐方案 C：SmartSplit 增加 flex/grid 感知拆分

**目标：修两栏简历被拆坏。**

建议策略：
1. `findCrossingPositions` 或 `buildSplitTransaction` 在候选位置附近读取 DOM computed style。
2. 如果候选节点是 flex/grid 容器的直接子元素，不把父 flex/grid 容器拆开。
3. 优先在该列内部寻找段落、列表项、section 等子块作为拆分页点。
4. 如果列内部找不到安全拆分页点，本轮跳过，不要破坏布局。

验收标准：
- `.main { display:flex }` 下 `.left-col` 和 `.right-col` 不会被拆到两个 `.main`。
- 右栏跨页时，拆分发生在 `.right-col` 内部的安全块级节点，或安全跳过。

### 2.4 推荐方案 D：字体与 PDF 渲染环境收口

**目标：修字距、行宽和分页偏移。**

建议：
1. 修复 `interFontFaceCSS()`，用真正的 base64 编码嵌入 Inter。
2. 明确中文字体基准，最好使用同一份可分发中文字体同时服务编辑器和 PDF。
3. 增加字体落点验证：编辑器 computed font 与 PDF/chromedp 环境字体一致。
4. 保留当前 `white-space: break-spaces` 对齐，并用长中文、英文数字混排、连续空格做手动验收。

---

## 3. 已解决 / 部分解决问题（PR #58 与本轮提交已覆盖）

### 3.1 部分完成：SmartSplit 抑制逻辑已补，但循环未验收

**状态：部分完成。**

新增 `suppressCount` 与 apply 守卫后，SmartSplit 自己 dispatch 引发的二次检测被抑制，元数据事务也不会再错误重置 `ownDispatch`。这能防止一次分页修正触发连续多轮拆分。

但这不是完整循环修复。当前仍缺少幂等 diff、两阶段时序和真实循环回归测试，因此不能作为“循环拆分已解决”的依据。

涉及提交：
- `fix: 修复 SmartSplit 插件级联循环问题`
- `fix: 保留元数据事务中的 ownDispatch 状态`

### 3.2 已完成：编辑器与 PDF 的 `white-space` 对齐

**状态：完成。**

编辑器 `.ProseMirror` 与 PDF `.resume-page` 已统一到 `white-space: break-spaces`，解决了 review 中指出的 `pre-wrap` 与 `break-spaces` 行尾空格处理不一致问题。

涉及位置：
- `frontend/workbench/src/styles/editor.css`
- `backend/internal/modules/render/render-template.html`

### 3.3 已完成：`typography.ts` 死代码移除

**状态：完成。**

`frontend/workbench/src/shared/typography.ts` 已删除。当前不再声称 TypeScript 常量是真实的单一排版来源，避免误导维护者。

### 3.4 已完成：`resolveToBlockPos` 增加 `pos=0` 防护

**状态：完成。**

`resolveToBlockPos` 已使用 `Math.max(0, ...)` 防止返回 `-1`，并已有对应测试覆盖。

涉及位置：
- `frontend/workbench/src/components/editor/extensions/smart-split/styleUtils.ts`
- `frontend/workbench/tests/smart-split/styleUtils.test.ts`

### 3.5 已完成：PDF 模板 magic number 已补注释

**状态：完成。**

PDF 模板中的 `68px`、`76px` 已补充与 96dpi 下毫米换算相关的注释，后续维护者能看懂这些值和 A4 边距的关系。

涉及位置：
- `backend/internal/modules/render/render-template.html`

### 3.6 已完成：threshold 文档与实现已对齐

**状态：完成。**

设计文档已对齐当前 `threshold: 4` 的实际实现，并说明该值用于给 chromedp / 浏览器亚像素差异留缓冲。

涉及位置：
- `docs/plans/2026-05-12-editor-pdf-css-sync-design.md`
- `frontend/workbench/src/components/editor/extensions/smart-split/types.ts`

---

## 4. 当前验证记录

### 4.1 PR 同步

已重新下载 `https://github.com/UN-Self/ResumeGenius/pull/58.patch`，当前 patch 仍为 14 个提交，没有发现新的第 15 个提交。本地分支已包含 PR #58 当前内容。

### 4.2 自动化测试

前端 SmartSplit 测试：

```bash
cd frontend/workbench
bunx vitest run tests/smart-split/
```

结果：通过，5 个测试文件，60 个测试全部通过。

后端 render 测试：

```bash
cd backend
$env:DB_PORT='55432'
go test ./internal/modules/render/...
```

结果：通过。

---

## 5. 当前结论

PR #58 的新增提交已经修掉了 review 里一批确定性问题，尤其是 `white-space`、`typography.ts`、`pos=0`、模板注释和 threshold 文档同步。SmartSplit 循环方面只能算“补了抑制机制”，还不能算真正修复。

但合并前仍有四个核心风险：

1. 循环拆分没有幂等验收，真实编辑器里仍可能持续 dispatch。
2. AI 原始 CSS、编辑器 scoped CSS、PDF 导出 CSS 可能不是同一份契约，这是导出排版错乱的高概率根因。
3. 两栏 flex 容器仍可能被错误拆开，这是当前截图里分页和排版混乱的另一个高概率根因。
4. 导出前未 flush，字体链路未闭环，仍会影响“所见即所得”。

下一步建议优先采纳方案 A 和 B：先统一 AI/编辑器/PDF 的 HTML/CSS 契约，再把 SmartSplit 改成幂等两阶段流程。两栏 flex 感知和字体收口作为下一层修复推进。
