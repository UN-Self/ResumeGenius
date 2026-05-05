# 营销站动画与深度感增强 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为营销站添加克制精致的动画/特效层，减少扁平化，提升品质感

**Architecture:** 渐进增强策略——通过全局 CSS 类 + 一个 IntersectionObserver 脚本注入动画层，组件上只添加 class。零额外依赖，纯 CSS + 原生 JS。

**Tech Stack:** Tailwind CSS keyframes, CSS transitions, IntersectionObserver API, Astro static HTML

---

### Task 1: 添加 Tailwind 自定义 keyframes 与 animation utilities

**Files:**
- Modify: `frontend/marketing/tailwind.config.mjs`

**Step 1: 在 tailwind.config.mjs 的 `extend` 中添加 keyframes 和 animation**

在 `fontFamily` 之后、`extend` 闭合 `}` 之前，添加 `keyframes` 和 `animation` 配置：

```js
      keyframes: {
        'fade-in-up': {
          '0%': { opacity: '0', transform: 'translateY(20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'scale-in': {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
        'float': {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-10px)' },
        },
      },
      animation: {
        'fade-in-up': 'fade-in-up 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards',
        'fade-in': 'fade-in 0.5s ease-out forwards',
        'scale-in': 'scale-in 0.5s ease-out forwards',
        'float': 'float 6s ease-in-out infinite',
        'float-delayed': 'float 8s ease-in-out infinite',
      },
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功，无错误

**Step 3: Commit**

```bash
git add frontend/marketing/tailwind.config.mjs
git commit -m "feat(marketing): add custom keyframes and animation utilities"
```

---

### Task 2: 添加全局 CSS 动画类、卡片悬浮、按钮反馈

**Files:**
- Modify: `frontend/marketing/src/styles/global.css`

**Step 1: 在 global.css 中添加所有全局动画类**

在 `@layer base { ... }` 之后添加以下内容：

```css
/* ===== Scroll-triggered animations ===== */
.animate-on-scroll {
  opacity: 0;
  transform: translateY(20px);
  transition:
    opacity 0.6s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.6s cubic-bezier(0.16, 1, 0.3, 1);
  transition-delay: var(--delay, 0ms);
}

.animate-on-scroll.is-visible {
  opacity: 1;
  transform: translateY(0);
}

/* ===== Card hover lift ===== */
.card-hover {
  transition: transform 200ms ease-out, box-shadow 200ms ease-out;
}

.card-hover:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(42, 28, 16, 0.08);
}

/* ===== Button interactions ===== */
.btn-primary {
  transition: transform 150ms ease-out, box-shadow 150ms ease-out, background-color 150ms ease-out;
}

.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(196, 149, 106, 0.3);
}

.btn-primary:active {
  transform: scale(0.98);
  box-shadow: none;
}

.btn-secondary {
  transition: transform 150ms ease-out, background-color 150ms ease-out;
}

.btn-secondary:hover {
  transform: translateY(-1px);
}

.btn-secondary:active {
  transform: scale(0.98);
}

/* ===== FAQ smooth height transition ===== */
.faq-content {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 300ms ease-out;
}

.faq-content.open {
  grid-template-rows: 1fr;
}

.faq-content > div {
  overflow: hidden;
  min-height: 0;
}

/* ===== Nav scroll border ===== */
.nav-bar {
  transition: border-color 200ms ease-out;
}

.nav-scrolled {
  border-bottom-color: rgba(228, 221, 213, 0.5);
}

/* ===== Reduced motion ===== */
@media (prefers-reduced-motion: reduce) {
  .animate-on-scroll {
    opacity: 1;
    transform: none;
    transition: none;
  }

  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/styles/global.css
git commit -m "feat(marketing): add global animation, card hover, button feedback CSS classes"
```

---

### Task 3: 在 BaseLayout 中添加 IntersectionObserver 和导航滚动脚本

**Files:**
- Modify: `frontend/marketing/src/layouts/BaseLayout.astro`

**Step 1: 在 `<Footer />` 之后、`</body>` 之前添加脚本**

在 `</body>` 标签前添加：

```astro
    <script>
      // Scroll-triggered animation observer
      const animObserver = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              entry.target.classList.add('is-visible');
              animObserver.unobserve(entry.target);
            }
          });
        },
        { threshold: 0.1, rootMargin: '0px 0px -50px 0px' }
      );
      document.querySelectorAll('.animate-on-scroll').forEach((el) => animObserver.observe(el));

      // Nav scroll border
      const nav = document.querySelector('.nav-bar');
      if (nav) {
        const updateNav = () => {
          nav.classList.toggle('nav-scrolled', window.scrollY > 10);
        };
        window.addEventListener('scroll', updateNav, { passive: true });
        updateNav();
      }

      // FAQ smooth toggle (grid-template-rows)
      document.querySelectorAll('details').forEach((detail) => {
        const content = detail.querySelector('.faq-content');
        if (!content) return;

        const updateOpen = () => {
          if (detail.open) {
            content.classList.add('open');
          } else {
            content.classList.remove('open');
          }
        };

        detail.addEventListener('toggle', updateOpen);
      });
    </script>
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/layouts/BaseLayout.astro
git commit -m "feat(marketing): add IntersectionObserver, nav scroll, FAQ toggle scripts"
```

---

### Task 4: 增强 Nav 组件

**Files:**
- Modify: `frontend/marketing/src/components/Nav.astro`

**Step 1: 更新 nav 元素和按钮样式**

将第 9 行的 nav 元素改为：
```astro
<nav class="nav-bar fixed top-0 left-0 right-0 z-50 bg-background/80 backdrop-blur-sm border-b border-transparent">
```

将第 23 行的"开始使用"主按钮添加 `btn-primary`：
```astro
      <a href="/app"
        class="btn-primary inline-flex items-center justify-center h-10 px-5 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 active:bg-primary-800 no-underline">
        开始使用
      </a>
```

将第 42 行的移动端主按钮同样添加 `btn-primary`：
```astro
    <a href="/app"
      class="btn-primary inline-flex items-center justify-center h-10 px-5 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 transition-colors duration-150 no-underline mt-2">
      开始使用
    </a>
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/Nav.astro
git commit -m "feat(marketing): add nav scroll border and button interaction classes"
```

---

### Task 5: 增强 Hero 组件（装饰光斑 + 按钮类）

**Files:**
- Modify: `frontend/marketing/src/components/Hero.astro`

**Step 1: 添加装饰光斑和按钮交互类**

在 `<section>` 开标签之后、`<div class="max-w-4xl...">` 之前，插入两个装饰光斑：

```astro
    <!-- Decorative blobs -->
    <div class="absolute top-10 right-10 w-[400px] h-[400px] rounded-full bg-primary-200/30 blur-[80px] animate-float pointer-events-none -z-0" aria-hidden="true"></div>
    <div class="absolute bottom-10 left-10 w-[300px] h-[300px] rounded-full bg-accent/20 blur-[60px] animate-float-delayed pointer-events-none -z-0" aria-hidden="true"></div>
```

同时需要给 section 添加 `relative overflow-hidden`：
```astro
<section class="relative overflow-hidden pt-12 pb-16 md:pt-20 md:pb-24 bg-gradient-to-b from-primary-50 to-background">
```

给两个 CTA 按钮添加交互类：

主按钮（第 31 行）改为：
```astro
        class="btn-primary inline-flex items-center justify-center h-12 px-8 text-base font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 active:bg-primary-800 no-underline"
```

次按钮（第 35 行）改为：
```astro
        class="btn-secondary inline-flex items-center justify-center h-12 px-8 text-base font-medium text-primary-600 border border-primary-300 rounded-md hover:bg-primary-50 no-underline"
```

给 3 个 mini feature cards（第 43 行）添加 `card-hover`：
```astro
        <div class="card-hover bg-card border border-border rounded-lg px-6 py-5 text-left">
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/Hero.astro
git commit -m "feat(marketing): add decorative blobs and interaction classes to Hero"
```

---

### Task 6: 增强 Section 组件（标题渐入动画）

**Files:**
- Modify: `frontend/marketing/src/components/Section.astro`

**Step 1: 给标题和描述添加 `.animate-on-scroll`**

将第 17-21 行改为：

```astro
        <h2 class="animate-on-scroll font-serif font-semibold text-3xl md:text-4xl text-foreground mb-4">
          {title}
        </h2>
        {subtitle && (
          <p class="animate-on-scroll text-lg text-muted-foreground max-w-2xl mx-auto" style="--delay: 100ms">
            {subtitle}
          </p>
        )}
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/Section.astro
git commit -m "feat(marketing): add scroll-triggered fade-in to Section titles"
```

---

### Task 7: 增强 FeatureSection 组件（图片占位 + 动画类）

**Files:**
- Modify: `frontend/marketing/src/components/FeatureSection.astro`

**Step 1: 增强图片占位区 + 添加动画类**

将整个模板改为：

```astro
<div {id} class="animate-on-scroll grid grid-cols-1 md:grid-cols-2 gap-12 items-center py-8">
  <!-- Image placeholder -->
  <div class={`relative overflow-hidden bg-muted border border-border rounded-lg h-64 flex items-center justify-center text-muted-foreground text-sm shadow-[inset_0_2px_8px_rgba(0,0,0,0.06)] ${isLeft ? '' : 'md:order-2'}`} aria-hidden="true">
    <div class="absolute inset-0 bg-gradient-to-br from-primary-50/50 to-transparent pointer-events-none"></div>
    <span class="relative z-10">示意图</span>
  </div>

  <!-- Text -->
  <div class={isLeft ? 'md:order-2' : ''}>
    <h3 class="font-serif font-semibold text-2xl text-foreground mb-3">{title}</h3>
    <p class="text-base text-muted-foreground leading-relaxed">{subtitle}</p>
  </div>
</div>
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/FeatureSection.astro
git commit -m "feat(marketing): enhance feature image placeholder and add scroll animation"
```

---

### Task 8: 增强 PricingCard 组件（卡片悬浮 + 按钮类）

**Files:**
- Modify: `frontend/marketing/src/components/PricingCard.astro`

**Step 1: 添加 card-hover 和按钮交互类**

将第 23 行改为（添加 `card-hover`）：
```astro
<div class={`card-hover bg-card border rounded-lg px-8 py-8 flex flex-col ${highlighted ? 'border-primary-400 ring-1 ring-primary-400/20' : 'border-border'}`}>
```

将第 37-42 行的按钮改为：
```astro
  <a href={ctaHref}
    class={`btn-primary inline-flex items-center justify-center h-11 px-6 text-sm font-medium rounded-md no-underline text-center ${
      highlighted
        ? 'bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800'
        : 'bg-card text-foreground border border-border hover:bg-muted active:bg-muted'
    }`}>
    {ctaText}
  </a>
```

注意：次按钮也使用 `btn-primary` 的 scale 效果，但背景/文字色不同。

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/PricingCard.astro
git commit -m "feat(marketing): add card hover and button interaction to PricingCard"
```

---

### Task 9: 增强 FaqGroup 组件（平滑高度过渡 + 卡片悬浮）

**Files:**
- Modify: `frontend/marketing/src/components/FaqGroup.astro`

**Step 1: 改为 grid 高度过渡结构**

将整个模板改为：

```astro
<div {id} class="mb-10 scroll-mt-20">
  <h3 class="font-serif font-semibold text-xl text-foreground mb-4">{title}</h3>
  <div class="space-y-2">
    {items.map(item => (
      <details class="card-hover bg-card border border-border rounded-lg group">
        <summary class="px-5 py-4 cursor-pointer text-sm font-medium text-foreground hover:text-primary-600 transition-colors duration-150 list-none flex items-center justify-between">
          {item.q}
          <svg class="w-4 h-4 flex-shrink-0 ml-2 text-muted-foreground group-open:rotate-180 transition-transform duration-200" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
        </summary>
        <div class="faq-content">
          <div>
            <div class="px-5 pb-4 text-sm text-muted-foreground leading-relaxed">
              {item.a}
            </div>
          </div>
        </div>
      </details>
    ))}
  </div>
</div>
```

关键变化：
- `<details>` 添加 `card-hover`
- 答案内容用 `.faq-content > div > div` 三层结构实现 grid 动画

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/FaqGroup.astro
git commit -m "feat(marketing): add smooth height transition and card hover to FAQ"
```

---

### Task 10: 增强 CTASection 组件（按钮类）

**Files:**
- Modify: `frontend/marketing/src/components/CTASection.astro`

**Step 1: 给 CTA 按钮添加 `btn-primary`**

将第 25-26 行的按钮改为：
```astro
    <a href={buttonHref}
      class="btn-primary inline-flex items-center justify-center h-12 px-8 text-base font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 active:bg-primary-800 no-underline">
      {buttonText}
    </a>
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/CTASection.astro
git commit -m "feat(marketing): add button interaction class to CTASection"
```

---

### Task 11: 增强 Footer 组件（渐变分隔线）

**Files:**
- Modify: `frontend/marketing/src/components/Footer.astro`

**Step 1: 将 border-t 改为渐变边框**

将第 1 行的 footer 改为：
```astro
<footer class="bg-card" style="border-top: 1px solid transparent; border-image: linear-gradient(to right, transparent, #e4ddd5, transparent) 1;">
```

注意：Tailwind 的 `border-image` 不支持直接用 class 设置，所以用 inline style。

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/components/Footer.astro
git commit -m "feat(marketing): replace flat footer border with gradient border"
```

---

### Task 12: 增强首页 index.astro（动画类添加到各区块）

**Files:**
- Modify: `frontend/marketing/src/pages/index.astro`

**Step 1: 给 feature cards 和 steps 添加动画类**

将 Feature cards 的 3 个 `<div>` （第 13、20、27 行）各添加 `card-hover animate-on-scroll` 和延迟：

第一个 card（第 13 行）：
```astro
      <div class="card-hover animate-on-scroll bg-card border border-border rounded-lg px-6 py-6">
```

第二个 card（第 20 行）：
```astro
      <div class="card-hover animate-on-scroll bg-card border border-border rounded-lg px-6 py-6" style="--delay: 100ms">
```

第三个 card（第 27 行）：
```astro
      <div class="card-hover animate-on-scroll bg-card border border-border rounded-lg px-6 py-6" style="--delay: 200ms">
```

给 "三步完成你的简历" section 标题（第 39 行）添加动画：
```astro
      <h2 class="animate-on-scroll font-serif font-semibold text-3xl md:text-4xl text-foreground text-center mb-12">
```

给 3 个 step cards（第 43、48、53 行）添加动画：
```astro
        <div class="animate-on-scroll text-center">
        <div class="animate-on-scroll text-center" style="--delay: 100ms">
        <div class="animate-on-scroll text-center" style="--delay: 200ms">
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/pages/index.astro
git commit -m "feat(marketing): add scroll animations and card hover to homepage"
```

---

### Task 13: 增强定价页 pricing.astro（积分卡片 + FAQ 动画）

**Files:**
- Modify: `frontend/marketing/src/pages/pricing.astro`

**Step 1: 给积分充值信息卡片添加动画类**

将第 68 行改为：
```astro
    <div class="animate-on-scroll max-w-md mx-auto mt-12 p-6 bg-card border border-border rounded-lg text-center">
```

**Step 2: 验证构建通过**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/marketing/src/pages/pricing.astro
git commit -m "feat(marketing): add scroll animation to pricing credit info card"
```

---

### Task 14: 最终构建验证 + 视觉检查

**Step 1: 完整构建**

Run: `cd frontend/marketing && bun run build`
Expected: 构建成功，无警告

**Step 2: 启动开发服务器**

Run: `cd frontend/marketing && bun run dev`
Expected: 开发服务器启动在本地端口

**Step 3: 视觉检查清单**

在浏览器中打开以下页面，逐项检查：

- [ ] **首页 Hero**: 两个模糊光斑可见并缓慢浮动
- [ ] **首页 Hero**: 两个 CTA 按钮 hover 时有阴影浮起 + active 时缩放
- [ ] **首页 Feature cards**: hover 时浮起 + 阴影，进入视口时交错淡入
- [ ] **首页 Steps**: 标题和 3 个步骤卡片交错淡入
- [ ] **首页 CTASection**: 按钮有交互反馈
- [ ] **Features 页**: 5 个 FeatureSection 依次淡入，图片占位有内阴影和渐变
- [ ] **Pricing 页**: 两张 PricingCard hover 浮起，积分信息卡片淡入
- [ ] **Pricing 页 FAQ**: 展开/收起有平滑高度过渡
- [ ] **Help 页**: FAQ 展开平滑，侧边栏链接正常
- [ ] **Nav**: 向下滚动时出现底部半透明边框
- [ ] **Footer**: 顶部渐变分隔线（两侧淡出）
- [ ] **所有页面**: reduced-motion 模式下无动画

**Step 4: 最终 commit（如有调整）**

如有修复，单独提交。全部完成后无需额外 commit。
