# AI 生成简历 A4 布局与边框污染排查开发日志

**类型：** 开发日志 / 问题整理  
**模块：** workbench / agent generated HTML / scoped CSS / PDF render  
**时间：** 2026-05-14  
**当前分支：** `feat/pr58-pagination-pdf-fixes`  
**背景：** PR #58 已修复一批分页与 PDF 导出链路问题，但用户手动测试发现 AI 生成的新简历仍会出现 A4 位置不正确、内容不居中、灰色边框或约束线被导出的问题。

---

## 1. 当前新增问题

### 1.1 生成内容没有稳定占据正确 A4 页面位置

**状态：未完成，需优先处理。**

截图现象：
- 编辑器中简历内容没有稳定铺在系统 A4 画布的正文区域内。
- 内容有时被压成偏窄的中间列，左右留白异常。
- PDF 导出后内容区域与编辑器预览仍有偏移，无法满足“所见即所得”。
- 分页条附近可见内容容器和页面背景之间的错位。

初步判断：
- AI 生成的是“完整网页/设计稿页面”样式，而不是“放进系统 A4 画布里的简历正文”样式。
- 生成 CSS 里可能包含 `body`、`html`、`.resume-document`、`.page` 这类页面级布局规则，例如 `display: flex`、`justify-content: center`、`padding`、`background`、`width: 210mm`、`min-height: 297mm`。
- 前端 `extractStyles` 会把部分全局选择器改写到 `.resume-document` 下，导致原本属于网页外壳的规则命中编辑器/PDF 的正文容器。
- 同时，系统自己的 A4 shell 已经负责纸张尺寸、页边距、分页条和缩放；AI 再生成一层 `.page` 或 `.resume-document` 页面容器，会形成“双层 A4 / 双层页面模型”，布局自然容易偏。

### 1.2 生成简历出现灰色背景、边框、阴影或约束线

**状态：未完成，需优先处理。**

截图现象：
- 编辑器预览里可以看到灰色大块背景。
- PDF 导出后灰色背景仍然存在。
- 页面内出现多条横向边框线、外框线、阴影或类似约束框的视觉元素。
- 这些元素并不是用户明确需要的简历设计，而更像 AI 为网页预览容器生成的辅助样式。

初步判断：
- AI 生成 CSS 可能把 `.page`、`.resume`、`.resume-document` 设计成卡片容器，包含 `background: #eee`、`box-shadow`、`border`、`outline` 等样式。
- 当前样式清洗没有区分“内容装饰线”和“页面容器约束线”：
  - section 标题下划线、分割线可以保留。
  - A4 外层、页面根容器、预览卡片阴影、灰色背景不应该进入最终 PDF。
- PDF 渲染本质上是忠实渲染 HTML/CSS；如果这些灰色背景和边框进入草稿内容，导出看到它们是符合当前实现的，但不符合产品预期。

### 1.3 生成模板与系统 A4 shell 职责不清

**状态：未完成，属于根因级问题。**

目前缺少明确契约：
- 谁负责 A4 纸张尺寸。
- 谁负责页边距。
- 谁负责页面背景。
- 谁负责内容宽度。
- AI 是否允许写 `.page`。
- AI 是否允许写 `body/html/.resume-document` 布局规则。
- PDF 端是否需要兜底清理这些页面级样式。

结果是 AI、编辑器和 PDF 后端都可能尝试控制页面几何，最终表现为：
- 编辑器看似可用，PDF 偏移。
- PDF 看似居中，编辑器偏移。
- 页面根容器被当成内容容器渲染。
- 灰色预览壳样式被误认为简历正文样式。

---

## 2. 建议解决方案

### 2.1 推荐方案 A：建立 AI 简历 HTML/CSS 生成契约

**目标：让 AI 只生成正文结构和正文样式，不生成网页外壳。**

建议约束：
- AI 输出的 HTML 只表示简历正文内容，不包含完整网页页面模型。
- 禁止或弱化以下选择器中的布局样式：
  - `html`
  - `body`
  - `.resume-page`
  - `.resume-document`
  - 顶层 `.page`
  - 顶层 `.resume`
- AI 不应生成 `width: 210mm`、`min-height: 297mm`、页面级 `margin auto`、页面级 `display:flex`、页面级 `box-shadow`。
- AI 不应通过灰底、阴影、外框来模拟预览卡片。
- AI 可以生成内容级样式，例如：
  - 姓名、联系方式、标题、段落、列表。
  - section 分割线。
  - 局部两栏布局。
  - 字号、颜色、行高、粗细。

落地方式：
- 更新 agent 生成简历的系统提示词和工具提示。
- 明确告诉模型：系统外层已经提供 A4 画布，生成内容必须填充系统画布的正文区域。
- 要求模型避免生成页面外壳类样式。

优点：
- 改动范围小。
- 能减少后续新生成简历继续带入灰底和约束框。
- 不直接影响已有分页算法。

风险：
- 只靠 prompt 不能完全保证模型永远遵守。
- 仍需要代码层 sanitizer 兜底。

### 2.2 推荐方案 B：在 `extractStyles` 增加页面级样式清洗

**目标：从代码层阻止 AI 页面外壳样式污染编辑器和 PDF。**

建议规则：
- 对 `html` / `body` 规则只保留文本相关属性：
  - `font-family`
  - `font-size`
  - `line-height`
  - `color`
- 从 `html` / `body` / `.resume-document` / 顶层 `.page` / 顶层 `.resume` 移除页面几何和外壳视觉属性：
  - `width`
  - `min-width`
  - `max-width`
  - `height`
  - `min-height`
  - `max-height`
  - `margin`
  - `padding`
  - `display`
  - `position`
  - `left`
  - `right`
  - `top`
  - `bottom`
  - `transform`
  - `background`
  - `box-shadow`
  - `outline`
  - 页面根容器上的 `border`
- 对内容级元素继续允许常规样式，例如 `.section-title { border-bottom: ... }`。

建议测试：
- `body { background:#eee; display:flex; padding:40px }` 不应变成 `.resume-document` 的灰底 flex 容器。
- `.page { width:210mm; min-height:297mm; box-shadow:...; background:#eee }` 不应污染系统 A4 页面。
- section 标题的 `border-bottom` 仍可保留。
- scoped CSS 输出后，编辑器和 PDF 都不应出现页面级灰底、阴影、外框。

优点：
- 对 AI 不守约有兜底能力。
- 能直接处理截图里的灰色背景、边框、阴影问题。
- 与“所见即所得”目标一致。

风险：
- 清洗过严可能误删用户真正想要的背景或边框设计。
- 需要通过选择器层级判断“根容器边框”和“内容边框”的区别。

### 2.3 推荐方案 C：明确系统 A4 shell 的唯一所有权

**目标：系统负责纸张，用户内容只负责纸内排版。**

建议统一模型：

```html
<div class="resume-page">
  <div class="resume-document">
    <div class="resume-content ProseMirror">
      <!-- 用户/AI 生成的简历正文 -->
    </div>
  </div>
</div>
```

职责划分：
- `.resume-page`：系统 A4 纸张、分页、页面背景、导出尺寸。
- `.resume-document`：系统样式作用域，不允许 AI 当作页面 flex 容器。
- `.resume-content`：正文内容区域，AI 样式可以主要落在这里和其子元素上。
- AI 生成的 `.page`：如果出现，应视作普通内容包裹层，不能拥有 A4 尺寸、阴影、灰底、外框。

后端 PDF 兜底：
- PDF 模板中系统 shell 样式应在用户 CSS 之后再追加一小段 guard CSS，保护 `.resume-page` 和直接 shell 不被用户 CSS 改坏。
- guard CSS 只保护外壳，不覆盖正文内容排版。

示例方向：

```css
.resume-page {
  width: 210mm;
  min-height: 297mm;
  background: #fff;
}

.resume-page > .resume-document {
  background: transparent;
  box-shadow: none;
  outline: none;
}
```

注意：是否使用 `!important` 需要谨慎。可以优先通过 sanitizer 避免污染，guard 只作为最后保护。

### 2.4 推荐方案 D：生成后立即规范化并保存同一份 HTML

**目标：编辑器看到什么，PDF 就导出什么。**

建议：
- AI apply 后，不要长期保留 AI 原始 HTML/CSS。
- 前端将 AI 输出经过 `extractStyles` / sanitizer / persistable HTML 规范化后，立即保存回草稿。
- 导出前仍执行 flush，但数据库里的内容本身也应是规范化后的版本。
- 后端导出时不再直接信任原始 AI CSS，而是使用规范化后的内容。

优点：
- 减少“编辑器处理过，但 PDF 读取原始 CSS”的链路分叉。
- 更符合 WYSIWYG。

风险：
- 需要确认版本历史是否仍要保留 AI 原始输出。
- 需要避免自动保存时机和用户编辑互相覆盖。

---

## 3. 建议优先级

### P0：先修阻断所见即所得的问题

1. 在 `extractStyles` 增加页面级样式清洗。
2. 禁止 `body/html/.resume-document/.page` 的页面 shell 样式污染正文容器。
3. 保留内容级 section 分割线，不保留页面外框、灰底、阴影。
4. 增加针对灰底、边框、A4 宽度的单元测试。

### P1：再修生成源头

1. 更新 AI 生成提示词，明确系统已有 A4 画布。
2. 要求 AI 不生成完整网页壳，不生成 `.page` A4 尺寸和预览卡片样式。
3. 对生成结果增加一次轻量校验，发现页面级样式时提示或自动清洗。

### P2：最后补强 PDF 兜底

1. PDF shell 追加最小 guard CSS。
2. 确认导出 HTML 与编辑器 persistable HTML 完全一致。
3. 用真实生成简历做 1 页、2 页、多栏、长项目经历的回归测试。

---

## 4. 验收标准

必须满足：
- 编辑器中 AI 生成简历位于系统 A4 画布正确正文区域。
- PDF 导出后内容位置、宽度、分页与编辑器预览一致。
- PDF 中不出现 AI 预览容器带来的灰色背景、阴影、外框或约束边框。
- 内容级设计仍保留，例如标题颜色、section 分割线、局部两栏布局。
- 生成 1 页简历不会额外多出空白页。
- 生成 2 页简历时第二页从正确分页位置继续，不出现内容整体偏移。

建议手动用例：
- 黄应辉样例：检查首屏是否居中、是否占据正确 A4 正文宽度、导出是否仍有灰底边框。
- 陈子俊样例：检查 2 页分页是否稳定、是否没有尾部空白页。
- 双栏样例：检查左右栏是否仍在同一页内保持并排，不被分页拆成错位列。

---

## 5. 当前结论

这次截图暴露的不是单纯 PDF 字间距问题，而是 AI 生成样式、编辑器 scoped CSS、系统 A4 shell、PDF render shell 之间的职责边界仍不清晰。

推荐先采纳方案 B 和方案 A：
- 代码层先清洗页面级样式，立刻阻止灰底、边框、阴影和 A4 外壳污染。
- 生成源头再通过 prompt 约束，减少后续继续生成错误页面模型。

在这两项完成前，即使分页算法继续优化，也仍可能因为 AI 写入了错误页面外壳样式而出现“编辑器看起来不对，PDF 也跟着不对”的问题。
