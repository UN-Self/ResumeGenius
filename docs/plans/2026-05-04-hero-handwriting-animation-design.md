# Hero 手写动画设计

## 概述

首页 Hero 区域标题动画：CSS mask 遮罩渐显文字，三段文案无限循环切换。

## 需求

- 动画风格：文字从左到右逐步显现（CSS mask 遮罩）
- 三段轮播文案：
  1. `ResumeGenius`
  2. `有简历？上传优化。`
  3. `没简历？AI 帮你从零生成。`
- 无限循环：写完 → 停留 → 淡出 → 写下一段

## 技术方案：CSS Mask + requestAnimationFrame

不引入第三方动画库，全部自包含在 `Hero.astro` 组件内。

### 架构

```
Hero.astro
├── <style>          — CSS mask、淡入淡出
├── HTML             — h1 内含三个 <span> 文案
└── <script>         — 动画状态机 + requestAnimationFrame 循环
```

### 动画状态机

| 阶段 | 时长 | 文字 |
|------|------|------|
| WRITING | ~1.8s | CSS mask 从左到右展开 |
| HOLDING | ~2.5s | 完全显示 |
| FADING | ~0.5s | opacity → 0 淡出 |
| 间隔 | ~0.3s | 空白 |

循环顺序：0 → 1 → 2 → 0 → ...

### CSS Mask 文字显现

```css
.handwriting-text {
  mask-image: linear-gradient(to right, black var(--reveal), transparent var(--reveal));
  -webkit-mask-image: linear-gradient(to right, black var(--reveal), transparent var(--reveal));
}
```

JS 动态更新 `--reveal` CSS 变量从 `0%` → `100%`。使用 cubic ease-in-out 缓动函数控制显现节奏。

### 文案布局

```html
<h1 class="font-serif ...">
  <span class="handwriting-text" data-index="0">ResumeGenius</span>
  <span class="handwriting-text hidden" data-index="1">有简历？上传优化。</span>
  <span class="handwriting-text hidden" data-index="2">没简历？AI 帮你从零生成。</span>
</h1>
```

同一时刻只有一段文案可见。

### 响应式

- 移动端（< md）：`text-4xl`
- 桌面端（≥ md）：`text-5xl lg:text-[3.5rem]`

### 无障碍

- 文字始终存在于 DOM，屏幕阅读器可读
- `prefers-reduced-motion`：直接显示文字，不做动画

## 约束

- 不引入新的 npm 依赖
- 修改范围仅限 `frontend/marketing/src/components/Hero.astro`
- 不影响现有 Hero 的副标题、按钮、功能卡片等元素
