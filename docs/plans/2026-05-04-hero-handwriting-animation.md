# Hero 手写动画 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在首页 Hero 区域实现 CSS mask 遮罩渐显动画标题，三段文案无限循环切换。

**Architecture:** CSS mask 遮罩渐显 + JS requestAnimationFrame 状态机。全部自包含在 Hero.astro 组件内，不引入新依赖。

**Tech Stack:** Astro + Tailwind CSS + 原生 JS（内联 `<script>`）

**设计文档:** `docs/plans/2026-05-04-hero-handwriting-animation-design.md`

**修改范围:** 仅 `frontend/marketing/src/components/Hero.astro`

**测试方式:** 视觉验证（`cd frontend/marketing && bun run dev`），无自动化测试框架。

---

### Task 1: 替换 h1 HTML 结构

**Files:**
- Modify: `frontend/marketing/src/components/Hero.astro:20-22`

**Step 1: 替换 h1 内容**

将第 20-22 行的静态 h1 替换为三段轮播文案容器：

```astro
    <h1 class="hero-title font-serif font-semibold text-4xl md:text-5xl lg:text-[3.5rem] leading-tight text-foreground mb-6">
      <span class="handwriting-text" data-index="0">ResumeGenius</span>
      <span class="handwriting-text hidden" data-index="1">有简历？上传优化。</span>
      <span class="handwriting-text hidden" data-index="2">没简历？AI 帮你从零生成。</span>
    </h1>
```

**Step 2: 验证页面不报错**

Run: `cd frontend/marketing && bun run dev`
Expected: 页面加载无报错，显示 "ResumeGenius"，其余两段隐藏

---

### Task 2: 添加 CSS 样式

**Files:**
- Modify: `frontend/marketing/src/components/Hero.astro`

**Step 1: 在 `</section>` 之后添加 style 块**

```css
<style>
  .hero-title {
    min-height: 2.5em;
  }

  .handwriting-text {
    display: inline-block;
    -webkit-mask-image: linear-gradient(to right, black var(--reveal, 100%), transparent var(--reveal, 100%));
    mask-image: linear-gradient(to right, black var(--reveal, 100%), transparent var(--reveal, 100%));
    transition: opacity 0.5s ease;
  }

  .handwriting-text.hidden {
    display: none;
  }

  .handwriting-text.fading {
    opacity: 0;
  }

  @media (prefers-reduced-motion: reduce) {
    .handwriting-text {
      -webkit-mask-image: none !important;
      mask-image: none !important;
    }
  }
</style>
```

---

### Task 3: 添加 JS 动画状态机

**Files:**
- Modify: `frontend/marketing/src/components/Hero.astro`

**Step 1: 在 `</section>` 之后添加 script 块**

JS 状态机驱动三阶段循环：WRITING → HOLDING → FADING → 下一段。

- WRITING: requestAnimationFrame 循环更新 `--reveal` CSS 变量，使用 cubic ease-in-out 缓动
- HOLDING: setTimeout 停留展示
- FADING: CSS transition opacity → 0
- GAP: 短暂间隔后切到下一段

包含 `prefers-reduced-motion` 检测：直接显示文字，不做动画。

---

### Task 4: 视觉调优与响应式验证

**Step 1: 桌面端验证**
- 确认标题 `text-[3.5rem]`，动画流畅
- 确认文案居中对齐，不影响下方布局

**Step 2: 移动端验证**
- 确认标题 `text-4xl`
- 确认文字换行正常

**Step 3: 验证 prefers-reduced-motion**
- 确认文字直接显示，无动画
