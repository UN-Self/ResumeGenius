# Marketing Site Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a complete 4-page Astro marketing site with Warm Editorial design system for ResumeGenius (customer acquisition + help docs).

**Architecture:** Pure Astro + Tailwind CSS 3, zero JS except mobile hamburger menu (~15 lines vanilla). All components are `.astro` files. Design tokens translated from `ui-design-system.md` into Tailwind config. Pages: index, features, pricing, help.

**Tech Stack:** Astro 5, Tailwind CSS 3, Google Fonts (Playfair Display + Inter + JetBrains Mono), Lucide SVG icons

**Design Doc:** `docs/plans/2026-05-04-marketing-site-design.md`

---

### Task 1: Foundation — Tailwind Config + Global CSS + Google Fonts

**Files:**
- Modify: `frontend/marketing/tailwind.config.mjs`
- Create: `frontend/marketing/src/styles/global.css`
- Modify: `frontend/marketing/src/layouts/BaseLayout.astro`

**Step 1: Update Tailwind config with Warm Editorial tokens**

Replace the empty config in `tailwind.config.mjs`:

```js
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#faf6f2',
          100: '#f2e8e0',
          200: '#e5d2c2',
          300: '#d4b99a',
          400: '#c4956a',
          500: '#b3804d',
          600: '#9c6b3a',
          700: '#7d5530',
          800: '#5e4025',
          900: '#3f2b1a',
          950: '#2a1c10',
        },
        background: '#faf8f5',
        foreground: '#1a1815',
        card: {
          DEFAULT: '#ffffff',
          foreground: '#1a1815',
        },
        muted: {
          DEFAULT: '#f5f1ed',
          foreground: '#8c8279',
        },
        secondary: {
          DEFAULT: '#e8d5c4',
          foreground: '#5c4a3a',
        },
        accent: {
          DEFAULT: '#d4a574',
          foreground: '#4a3020',
        },
        destructive: {
          DEFAULT: '#d64545',
          foreground: '#ffffff',
        },
        border: '#e4ddd5',
        input: '#e4ddd5',
        ring: '#c4956a',
      },
      fontFamily: {
        serif: ['Playfair Display', 'serif'],
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
}
```

**Step 2: Create global CSS**

Create `frontend/marketing/src/styles/global.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&family=Playfair+Display:wght@500;600&display=swap');

@layer base {
  html {
    font-family: 'Inter', system-ui, sans-serif;
    color: #1a1815;
    background-color: #faf8f5;
  }

  body {
    min-height: 100vh;
  }
}
```

**Step 3: Update BaseLayout to include global CSS and proper structure**

Replace `frontend/marketing/src/layouts/BaseLayout.astro`:

```astro
---
export interface Props {
  title: string
  description?: string
}

const { title, description = 'ResumeGenius - AI 辅助简历生成与优化平台' } = Astro.props
---

<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content={description} />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <title>{title} | ResumeGenius</title>
  </head>
  <body class="bg-background text-foreground font-sans antialiased">
    <slot />
  </body>
</html>
```

Note: `global.css` import will be added by Astro + Tailwind integration automatically. If not, add `<link>` in head.

**Step 4: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds with Tailwind generating proper CSS.

---

### Task 2: Shared Layout Components — Nav + Footer

**Files:**
- Create: `frontend/marketing/src/components/Nav.astro`
- Create: `frontend/marketing/src/components/Footer.astro`
- Modify: `frontend/marketing/src/layouts/BaseLayout.astro`

**Step 1: Create Nav component**

Create `frontend/marketing/src/components/Nav.astro`:

```astro
---
const navLinks = [
  { href: '/features', label: '功能介绍' },
  { href: '/pricing', label: '定价' },
  { href: '/help', label: '帮助文档' },
]
---

<nav class="fixed top-0 left-0 right-0 z-50 bg-background/80 backdrop-blur-sm border-b border-border">
  <div class="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
    <a href="/" class="flex items-center gap-2 no-underline">
      <span class="text-xl font-serif font-semibold text-foreground">ResumeGenius</span>
    </a>

    <!-- Desktop nav -->
    <div class="hidden md:flex items-center gap-8" id="desktop-nav">
      {navLinks.map(link => (
        <a href={link.href} class="text-sm text-muted-foreground hover:text-foreground transition-colors duration-150">
          {link.label}
        </a>
      ))}
      <a href="/app"
        class="inline-flex items-center justify-center h-10 px-5 text-sm font-medium text-white bg-primary-400 rounded-md hover:bg-primary-500 active:bg-primary-600 transition-colors duration-150 no-underline">
        开始使用
      </a>
    </div>

    <!-- Mobile hamburger -->
    <button id="menu-toggle" class="md:hidden flex items-center justify-center w-10 h-10 rounded-md hover:bg-muted transition-colors duration-150" aria-label="菜单">
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
  </div>

  <!-- Mobile menu (hidden by default) -->
  <div id="mobile-menu" class="hidden md:hidden border-t border-border bg-background px-6 py-4 flex flex-col gap-3">
    {navLinks.map(link => (
      <a href={link.href} class="text-sm text-muted-foreground hover:text-foreground transition-colors duration-150 no-underline py-1">
        {link.label}
      </a>
    ))}
    <a href="/app"
      class="inline-flex items-center justify-center h-10 px-5 text-sm font-medium text-white bg-primary-400 rounded-md hover:bg-primary-500 transition-colors duration-150 no-underline mt-2">
      开始使用
    </a>
  </div>
</nav>

<!-- Spacer for fixed nav -->
<div class="h-16"></div>

<script>
  const toggle = document.getElementById('menu-toggle');
  const menu = document.getElementById('mobile-menu');
  toggle?.addEventListener('click', () => {
    menu?.classList.toggle('hidden');
  });
</script>
```

**Step 2: Create Footer component**

Create `frontend/marketing/src/components/Footer.astro`:

```astro
<footer class="border-t border-border bg-card">
  <div class="max-w-6xl mx-auto px-6 py-12 flex flex-col sm:flex-row items-center justify-between gap-4">
    <p class="text-sm text-muted-foreground">
      &copy; {new Date().getFullYear()} ResumeGenius. All rights reserved.
    </p>
    <nav class="flex items-center gap-6">
      <a href="/features" class="text-sm text-muted-foreground hover:text-foreground transition-colors duration-150 no-underline">功能介绍</a>
      <a href="/pricing" class="text-sm text-muted-foreground hover:text-foreground transition-colors duration-150 no-underline">定价</a>
      <a href="/help" class="text-sm text-muted-foreground hover:text-foreground transition-colors duration-150 no-underline">帮助文档</a>
    </nav>
  </div>
</footer>
```

**Step 3: Update BaseLayout to include Nav and Footer**

Update `frontend/marketing/src/layouts/BaseLayout.astro` — add Nav before `<slot />` and Footer after:

```astro
---
import Nav from '../components/Nav.astro'
import Footer from '../components/Footer.astro'

export interface Props {
  title: string
  description?: string
}

const { title, description = 'ResumeGenius - AI 辅助简历生成与优化平台' } = Astro.props
---

<!-- keep existing head, add Nav/Footer around slot -->
<body class="bg-background text-foreground font-sans antialiased">
  <Nav />
  <slot />
  <Footer />
</body>
```

**Step 4: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds. All existing pages now have Nav + Footer.

---

### Task 3: Section Component

**Files:**
- Create: `frontend/marketing/src/components/Section.astro`

**Step 1: Create Section component**

Create `frontend/marketing/src/components/Section.astro`:

```astro
---
export interface Props {
  title?: string
  subtitle?: string
  class?: string
  containerClass?: string
  id?: string
}

const { title, subtitle, class: extraClass = '', containerClass = '', id } = Astro.props
---

<section {id} class={`py-16 md:py-24 ${extraClass}`}>
  <div class={`max-w-6xl mx-auto px-6 ${containerClass}`}>
    {title && (
      <div class="text-center mb-12">
        <h2 class="font-serif font-semibold text-3xl md:text-4xl text-foreground mb-4">
          {title}
        </h2>
        {subtitle && (
          <p class="text-lg text-muted-foreground max-w-2xl mx-auto">
            {subtitle}
          </p>
        )}
      </div>
    )}
    <slot />
  </div>
</section>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds.

---

### Task 4: CTA Section Component

**Files:**
- Create: `frontend/marketing/src/components/CTASection.astro`

**Step 1: Create CTA Section**

Create `frontend/marketing/src/components/CTASection.astro`:

```astro
---
export interface Props {
  title?: string
  description?: string
  buttonText?: string
  buttonHref?: string
}

const {
  title = '准备好让你的简历脱颖而出了吗？',
  description = '免费开始使用，AI 帮你生成一份高质量的简历。',
  buttonText = '免费开始使用',
  buttonHref = '/app',
} = Astro.props
---

<section class="py-16 md:py-24 bg-gradient-to-b from-primary-50 to-background">
  <div class="max-w-2xl mx-auto px-6 text-center">
    <h2 class="font-serif font-semibold text-3xl md:text-4xl text-foreground mb-4">
      {title}
    </h2>
    <p class="text-lg text-muted-foreground mb-8">
      {description}
    </p>
    <a href={buttonHref}
      class="inline-flex items-center justify-center h-12 px-8 text-base font-medium text-white bg-primary-400 rounded-md hover:bg-primary-500 active:bg-primary-600 transition-colors duration-150 no-underline">
      {buttonText}
    </a>
  </div>
</section>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds.

---

### Task 5: FAQ Group Component

**Files:**
- Create: `frontend/marketing/src/components/FaqGroup.astro`

**Step 1: Create FaqGroup component**

Create `frontend/marketing/src/components/FaqGroup.astro`:

```astro
---
export interface Props {
  title: string
  id?: string
}

const { title, id } = Astro.props
---

<div {id} class="mb-12">
  <h3 class="font-serif font-semibold text-xl text-foreground mb-4">{title}</h3>
  <div class="space-y-2">
    <slot />
  </div>
</div>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds.

---

### Task 6: Hero Component (Homepage)

**Files:**
- Create: `frontend/marketing/src/components/Hero.astro`

**Step 1: Create Hero component**

Create `frontend/marketing/src/components/Hero.astro`:

```astro
---
const features = [
  {
    title: 'AI 写内容',
    description: '融合行业最佳实践，不是简单润色，而是重构结构与表达',
  },
  {
    title: '自由编辑',
    description: 'TipTap 所见即所得编辑器，像编辑 Word 一样无表单约束',
  },
  {
    title: '所见即所得',
    description: 'chromedp 服务端渲染，导出 PDF 与屏幕完全一致',
  },
]
---

<section class="pt-12 pb-16 md:pt-20 md:pb-24 bg-gradient-to-b from-primary-50 to-background">
  <div class="max-w-4xl mx-auto px-6 text-center">
    <h1 class="font-serif font-semibold text-4xl md:text-5xl lg:text-[3.5rem] leading-tight text-foreground mb-6">
      有简历？上传优化。<br class="md:hidden" />没简历？AI 帮你从零生成。
    </h1>
    <p class="text-lg md:text-xl text-muted-foreground max-w-3xl mx-auto mb-10 leading-relaxed">
      ResumeGenius 融合行业前辈的面试经验与简历最佳实践，AI 不再只是润色文字 ——
      它会重构结构、组织表达、主动搜集你的 GitHub / 文件 / 经历，直接生成可用简历。
    </p>
    <div class="flex flex-col sm:flex-row gap-4 justify-center mb-16">
      <a href="/app"
        class="inline-flex items-center justify-center h-12 px-8 text-base font-medium text-white bg-primary-400 rounded-md hover:bg-primary-500 active:bg-primary-600 transition-colors duration-150 no-underline">
        免费开始使用
      </a>
      <a href="#features"
        class="inline-flex items-center justify-center h-12 px-8 text-base font-medium text-primary-600 border border-primary-300 rounded-md hover:bg-primary-50 transition-colors duration-150 no-underline">
        了解更多
      </a>
    </div>

    <!-- Mini feature cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-3xl mx-auto">
      {features.map(f => (
        <div class="bg-card border border-border rounded-lg px-6 py-5 text-left">
          <h3 class="font-sans font-semibold text-foreground mb-2">{f.title}</h3>
          <p class="text-sm text-muted-foreground leading-relaxed">{f.description}</p>
        </div>
      ))}
    </div>
  </div>
</section>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds.

---

### Task 7: FeatureSection Component (Features page)

**Files:**
- Create: `frontend/marketing/src/components/FeatureSection.astro`

**Step 1: Create FeatureSection component**

Create `frontend/marketing/src/components/FeatureSection.astro`:

```astro
---
export interface Props {
  title: string
  subtitle: string
  imagePosition: 'left' | 'right'
  id?: string
}

const { title, subtitle, imagePosition, id } = Astro.props
const isLeft = imagePosition === 'left'
---

<div {id} class="grid grid-cols-1 md:grid-cols-2 gap-12 items-center py-8">
  <!-- Image placeholder -->
  <div class={`bg-muted border border-border rounded-lg h-64 flex items-center justify-center text-muted-foreground text-sm ${isLeft ? '' : 'md:order-2'}`}>
    示意图
  </div>

  <!-- Text -->
  <div class={isLeft ? 'md:order-2' : ''}>
    <h3 class="font-serif font-semibold text-2xl text-foreground mb-3">{title}</h3>
    <p class="text-base text-muted-foreground leading-relaxed">{subtitle}</p>
  </div>
</div>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds.

---

### Task 8: PricingCard Component

**Files:**
- Create: `frontend/marketing/src/components/PricingCard.astro`

**Step 1: Create PricingCard component**

Create `frontend/marketing/src/components/PricingCard.astro`:

```astro
---
export interface Props {
  name: string
  price: string
  period?: string
  features: string[]
  highlighted?: boolean
  ctaText?: string
  ctaHref?: string
}

const {
  name,
  price,
  period = '',
  features,
  highlighted = false,
  ctaText = '开始使用',
  ctaHref = '/app',
} = Astro.props
---

<div class={`bg-card border rounded-lg px-8 py-8 flex flex-col ${highlighted ? 'border-primary-400 ring-1 ring-primary-400/20' : 'border-border'}`}>
  <h3 class="font-sans font-semibold text-lg text-foreground mb-2">{name}</h3>
  <div class="flex items-baseline gap-1 mb-6">
    <span class="font-serif font-semibold text-4xl text-foreground">{price}</span>
    {period && <span class="text-sm text-muted-foreground">{period}</span>}
  </div>
  <ul class="flex-1 space-y-3 mb-8">
    {features.map(f => (
      <li class="flex items-start gap-2 text-sm text-muted-foreground">
        <svg class="w-4 h-4 mt-0.5 flex-shrink-0 text-primary-400" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        {f}
      </li>
    ))}
  </ul>
  <a href={ctaHref}
    class={`inline-flex items-center justify-center h-11 px-6 text-sm font-medium rounded-md transition-colors duration-150 no-underline text-center ${
      highlighted
        ? 'bg-primary-400 text-white hover:bg-primary-500 active:bg-primary-600'
        : 'border border-border text-foreground hover:bg-muted'
    }`}>
    {ctaText}
  </a>
</div>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Build succeeds.

---

### Task 9: Rebuild Homepage `/`

**Files:**
- Modify: `frontend/marketing/src/pages/index.astro`

**Step 1: Rewrite index.astro**

Replace `frontend/marketing/src/pages/index.astro`:

```astro
---
import BaseLayout from '../layouts/BaseLayout.astro'
import Hero from '../components/Hero.astro'
import Section from '../components/Section.astro'
import CTASection from '../components/CTASection.astro'
---

<BaseLayout title="AI 驱动的简历编辑器" description="ResumeGenius 融合行业最佳实践，AI 直接生成可用简历。上传优化或从零生成，所见即所得编辑器自由调整。">
  <Hero />

  <Section id="features" title="为什么选择 ResumeGenius？">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-8 mt-8">
      <div class="bg-card border border-border rounded-lg px-6 py-6">
        <div class="w-10 h-10 rounded-md bg-primary-100 flex items-center justify-center mb-4">
          <svg class="w-5 h-5 text-primary-600" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <h3 class="font-sans font-semibold text-foreground mb-2">AI 生成内容</h3>
        <p class="text-sm text-muted-foreground leading-relaxed">不只是在模板上填文字。AI 融合行业最佳实践，重构简历结构与表达方式。</p>
      </div>
      <div class="bg-card border border-border rounded-lg px-6 py-6">
        <div class="w-10 h-10 rounded-md bg-primary-100 flex items-center justify-center mb-4">
          <svg class="w-5 h-5 text-primary-600" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
        </div>
        <h3 class="font-sans font-semibold text-foreground mb-2">自由编辑</h3>
        <p class="text-sm text-muted-foreground leading-relaxed">TipTap 所见即所得编辑器，无表单约束，像编辑 Word 一样自由调整排版。</p>
      </div>
      <div class="bg-card border border-border rounded-lg px-6 py-6">
        <div class="w-10 h-10 rounded-md bg-primary-100 flex items-center justify-center mb-4">
          <svg class="w-5 h-5 text-primary-600" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 6 2 18 2 18 9"/><path d="M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6a2 2 0 0 0-2-2h-2"/><rect x="8" y="9" width="8" height="14" rx="1"/></svg>
        </div>
        <h3 class="font-sans font-semibold text-foreground mb-2">导出一致</h3>
        <p class="text-sm text-muted-foreground leading-relaxed">chromedp 服务端渲染 PDF，导出效果与编辑器预览完全一致。</p>
      </div>
    </div>
  </Section>

  <section class="py-16 md:py-24 bg-card border-t border-border">
    <div class="max-w-6xl mx-auto px-6">
      <h2 class="font-serif font-semibold text-3xl md:text-4xl text-foreground text-center mb-12">
        三步完成你的简历
      </h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
        <div class="text-center">
          <div class="w-12 h-12 rounded-full bg-primary-100 text-primary-600 font-semibold text-lg flex items-center justify-center mx-auto mb-4">1</div>
          <h3 class="font-sans font-semibold text-foreground mb-2">上传或搜集资料</h3>
          <p class="text-sm text-muted-foreground leading-relaxed">上传简历文件、接入 GitHub 仓库，或用对话引导 AI 搜集信息</p>
        </div>
        <div class="text-center">
          <div class="w-12 h-12 rounded-full bg-primary-100 text-primary-600 font-semibold text-lg flex items-center justify-center mx-auto mb-4">2</div>
          <h3 class="font-sans font-semibold text-foreground mb-2">AI 生成 + 自由编辑</h3>
          <p class="text-sm text-muted-foreground leading-relaxed">AI 结合最佳实践生成初稿，你可以在 TipTap 中自由调整</p>
        </div>
        <div class="text-center">
          <div class="w-12 h-12 rounded-full bg-primary-100 text-primary-600 font-semibold text-lg flex items-center justify-center mx-auto mb-4">3</div>
          <h3 class="font-sans font-semibold text-foreground mb-2">导出 PDF</h3>
          <p class="text-sm text-muted-foreground leading-relaxed">chromedp 服务端渲染，导出的 PDF 所见即所得</p>
        </div>
      </div>
    </div>
  </section>

  <CTASection />
</BaseLayout>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run dev` (check visually) and `bun run build`
Expected: Homepage renders with Hero, 3 feature cards, 3-step flow, CTA, Footer.

---

### Task 10: Features Page `/features`

**Files:**
- Modify: `frontend/marketing/src/pages/features.astro`

**Step 1: Rewrite features.astro**

Replace `frontend/marketing/src/pages/features.astro`:

```astro
---
import BaseLayout from '../layouts/BaseLayout.astro'
import Section from '../components/Section.astro'
import FeatureSection from '../components/FeatureSection.astro'
import CTASection from '../components/CTASection.astro'

const sections = [
  {
    title: '你不需要准备完整的简历材料',
    subtitle: 'Sub-Agent 自动从 Git 仓库提取项目信息、从上传文件中解析文本、通过对话引导关键信息补充。把散落在各处的资料交给 AI，零手工搬运。',
    imagePosition: 'right' as const,
  },
  {
    title: '不是简单的文本润色',
    subtitle: 'AI 融合行业前辈的面试经验与简历最佳实践，不只是改改措辞。它会重构简历的整体结构、优化内容组织、调整排版——直接生成完整的 HTML 简历。',
    imagePosition: 'left' as const,
  },
  {
    title: '像编辑 Word 一样编辑简历',
    subtitle: 'TipTap 所见即所得编辑器，没有预制表单的约束。拖拽排序、富文本格式、实时预览——你对排版的完全控制力。',
    imagePosition: 'right' as const,
  },
  {
    title: '用自然语言控制排版',
    subtitle: '告诉 AI 你想要的效果——"把工作经历放到教育经历前面"、"用更专业的方式描述这个项目"——AI 直接修改 HTML，一步完成内容与排版调整。',
    imagePosition: 'left' as const,
  },
  {
    title: '所见即所得，一字不差',
    subtitle: 'chromedp 服务端渲染 PDF。编辑器里看到的是什么，导出的 PDF 就是什么。不会出现模板平台常见的排版错乱或字体丢失。',
    imagePosition: 'right' as const,
  },
]
---

<BaseLayout title="功能介绍" description="ResumeGenius 不是模板平台，也不是表单工具——AI 直接生成简历 HTML，你可自由编辑后服务端渲染导出 PDF。">
  <Section title="不是模板，不是表单" subtitle="ResumeGenius 在五个关键环节上与传统方式不同">
    <div class="max-w-4xl mx-auto mt-8">
      {sections.map((s, i) => (
        <FeatureSection
          title={s.title}
          subtitle={s.subtitle}
          imagePosition={s.imagePosition}
          id={i > 0 ? undefined : undefined}
        />
      ))}
    </div>
  </Section>

  <CTASection
    title="体验不同于填表的方式"
    description="免费开始，AI 帮你从资料直接生成可用简历。"
  />
</BaseLayout>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Features page builds with 5 alternating sections.

---

### Task 11: Pricing Page `/pricing`

**Files:**
- Modify: `frontend/marketing/src/pages/pricing.astro`

**Step 1: Rewrite pricing.astro**

Replace `frontend/marketing/src/pages/pricing.astro`:

```astro
---
import BaseLayout from '../layouts/BaseLayout.astro'
import Section from '../components/Section.astro'
import PricingCard from '../components/PricingCard.astro'
import CTASection from '../components/CTASection.astro'

const freeFeatures = [
  '所有功能开放',
  '注册即送 100 积分（一次性）',
  'AI 对话消耗积分（1 积分 ≈ 1k token）',
  'PDF 导出 10 积分 / 次',
  '支持积分充值',
]

const proFeatures = [
  '所有功能开放',
  '每月赠送 1,900 积分',
  'AI 对话消耗积分',
  'PDF 导出无限次',
  '积分充值享 9 折',
]

const faqGroups = [
  {
    title: '积分相关',
    items: [
      { q: '积分消耗怎么算？', a: 'token 计算采用输入输出同价：1 积分 ≈ 1k token。一次简短的 AI 对话大约消耗 5-10 积分，一次完整的简历生成大约消耗 30-50 积分。' },
      { q: 'Free 积分用完怎么办？', a: '可以随时购买积分充值（1 人民币 = 100 积分），或升级到 Pro 获得每月 1,900 积分 + 无限 PDF 导出。' },
      { q: '积分会过期吗？', a: '不会。所有积分永不过期，随时可用。' },
    ],
  },
  {
    title: '升级与付费',
    items: [
      { q: '怎么升级到 Pro？', a: '在定价页选择 Pro 方案，点击"升级 Pro"即可。所有项目和数据无缝迁移。' },
      { q: '如何充值积分？', a: '在工作台中进入账号设置 → 积分充值，选择金额即可。Pro 用户自动享受 9 折。' },
      { q: '可以随时取消 Pro 订阅吗？', a: '可以。取消后当前月剩余时间仍可继续使用，下个周期不再续费。积分保留。' },
    ],
  },
]
---

<BaseLayout title="定价" description="简单积分制定价——免费开始，按需使用。Pro 享无限 PDF 导出与充值折扣。">
  <Section title="简单定价，按需使用">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-3xl mx-auto mt-8">
      <PricingCard
        name="Free"
        price="¥0"
        features={freeFeatures}
        ctaText="免费注册"
        ctaHref="/app"
      />
      <PricingCard
        name="Pro"
        price="¥19"
        period="/ 月"
        features={proFeatures}
        highlighted={true}
        ctaText="升级 Pro"
        ctaHref="/app"
      />
    </div>

    <!-- Points recharge -->
    <div class="max-w-md mx-auto mt-12 p-6 bg-card border border-border rounded-lg text-center">
      <h3 class="font-sans font-semibold text-foreground mb-2">积分充值</h3>
      <p class="text-sm text-muted-foreground mb-4">
        1 人民币 = 100 积分，随时充值，永不过期。<br />
        Pro 用户充值享 9 折优惠。
      </p>
    </div>
  </Section>

  <!-- FAQ -->
  <section class="py-16 bg-card border-t border-border">
    <div class="max-w-3xl mx-auto px-6">
      <h2 class="font-serif font-semibold text-3xl text-foreground text-center mb-12">常见问题</h2>
      {faqGroups.map(group => (
        <div class="mb-8">
          <h3 class="font-sans font-semibold text-lg text-foreground mb-4">{group.title}</h3>
          <div class="space-y-2">
            {group.items.map(item => (
              <details class="bg-background border border-border rounded-lg group">
                <summary class="px-5 py-4 cursor-pointer text-sm font-medium text-foreground hover:text-primary-600 transition-colors duration-150 list-none flex items-center justify-between">
                  {item.q}
                  <svg class="w-4 h-4 text-muted-foreground group-open:rotate-180 transition-transform duration-200" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
                </summary>
                <div class="px-5 pb-4 text-sm text-muted-foreground leading-relaxed">
                  {item.a}
                </div>
              </details>
            ))}
          </div>
        </div>
      ))}
    </div>
  </section>

  <CTASection
    title="免费注册，领 100 积分"
    description="所有功能开放，试试 AI 能帮你的简历做什么。"
    buttonText="免费开始使用"
  />
</BaseLayout>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Pricing page builds with 2 cards, recharge section, FAQ, CTA.

---

### Task 12: Help Page `/help`

**Files:**
- Modify: `frontend/marketing/src/pages/help.astro`

**Step 1: Rewrite help.astro**

Replace `frontend/marketing/src/pages/help.astro`:

```astro
---
import BaseLayout from '../layouts/BaseLayout.astro'
import Section from '../components/Section.astro'
import CTASection from '../components/CTASection.astro'

const groups = [
  {
    id: 'quickstart',
    title: '快速入门',
    items: [
      {
        q: '如何创建第一个项目？',
        a: '进入工作台后，点击"新建项目"，输入项目名称。你可以选择上传已有的简历文件（PDF/DOCX），或直接接入 GitHub 仓库让 AI 提取你的项目信息。',
      },
      {
        q: '如何上传简历并生成初稿？',
        a: '在项目中上传简历文件（支持 PDF 和 DOCX），系统会自动解析文本内容。AI 随即根据解析结果和行业最佳实践，生成一份完整的 HTML 简历初稿。',
      },
      {
        q: '如何与 AI 对话修改简历？',
        a: '在工作台右侧的 AI 对话面板中，用自然语言描述你想要的修改。AI 会流式返回修改后的 HTML。满意可以"应用到简历"，不满意可以继续对话调整。',
      },
      {
        q: '如何导出 PDF？',
        a: '在编辑器中满意后，点击顶部工具栏的"导出 PDF"。系统通过 chromedp 服务端渲染，生成与编辑器内完全一致的 PDF 文件。Free 用户每次导出消耗 10 积分。',
      },
    ],
  },
  {
    id: 'ai-chat',
    title: 'AI 对话',
    items: [
      {
        q: 'AI 能帮我做什么？',
        a: 'AI 可以：重构简历整体结构、优化措辞与表达、调整排版布局、根据目标职位定制内容重点、从 GitHub 等来源提取并组织你的项目经历。不是简单的文本润色，而是全链路操作 HTML。',
      },
      {
        q: '如何写出有效的 AI 指令？',
        a: '尽量具体。不好的指令："帮我把简历做好看点"。好的指令："把工作经历部分移到教育经历前面，用 STAR 原则重新描述最近两个项目，强调我的技术栈"。',
      },
      {
        q: '积分消耗与 token 换算',
        a: '1 积分 ≈ 1k token（输入输出同价）。一次简短的对话修正约 5-10 积分，一次完整的简历初稿生成约 30-50 积分。具体消耗取决于当前简历长度和修改复杂度。',
      },
      {
        q: 'AI 修改不满意怎么办？',
        a: 'AI 每次修改都返回独立的结果。你可以继续发送新的指令修正，也可以点击"拒绝"保留当前版本。所有修改都有版本快照，随时可以回退。',
      },
    ],
  },
  {
    id: 'edit-export',
    title: '编辑与导出',
    items: [
      {
        q: 'TipTap 编辑器有哪些功能？',
        a: '富文本格式（加粗、斜体、下划线）、字号与颜色调整、对齐方式、行距控制、有序/无序列表、Section 拖拽排序、原生 undo/redo。编辑器自动保存，无需手动操作。',
      },
      {
        q: '如何使用版本历史回退？',
        a: '点击顶部工具栏的"版本历史"，查看所有保存的快照（含时间与版本号）。点击任意历史版本可以预览，确认后可以回退——系统会基于回退的版本自动创建新快照。',
      },
      {
        q: '导出的 PDF 和编辑器内看到的一样吗？',
        a: '完全一致。ResumeGenius 使用 chromedp 在服务端以相同的 HTML 和 CSS 渲染 PDF，字体、排版、间距与编辑器内 A4 画布完全相同。',
      },
      {
        q: '支持哪些简历模板风格？',
        a: '目前 AI 会根据行业最佳实践自动选择适合的排版风格。你可以在 TipTap 编辑器中自由调整视觉风格（颜色、字号、间距、Section 顺序等）。后续版本将支持更多预设模板。',
      },
    ],
  },
  {
    id: 'account',
    title: '账号与付费',
    items: [
      {
        q: 'Free 和 Pro 有什么区别？',
        a: '两者所有功能完全开放。Free 注册送 100 积分（一次性），AI 对话和 PDF 导出均消耗积分。Pro（¥19/月）每月送 1,900 积分，PDF 导出无限次，积分充值享 9 折。',
      },
      {
        q: '如何充值积分？',
        a: '工作台中进入账号设置 → 积分充值，选择金额即可（1 元 = 100 积分）。Pro 用户自动享受充值 9 折。',
      },
      {
        q: '积分会过期吗？',
        a: '不会。所有积分永不过期。Pro 会员每月的赠送积分在当月有效期内不会清零，未用完的积分累积到下月。',
      },
    ],
  },
]
---

<BaseLayout title="帮助文档" description="ResumeGenius 使用指南——快速入门、AI 对话技巧、编辑与导出说明、账号与付费常见问题。">
  <Section title="帮助文档">
    <div class="flex flex-col md:flex-row gap-12 max-w-5xl mx-auto mt-8">
      <!-- Sidebar nav (desktop sticky) -->
      <aside class="md:w-48 flex-shrink-0 hidden md:block">
        <nav class="sticky top-24 space-y-1">
          {groups.map(g => (
            <a href={`#${g.id}`} class="block text-sm text-muted-foreground hover:text-primary-600 transition-colors duration-150 no-underline py-1.5">
              {g.title}
            </a>
          ))}
        </nav>
      </aside>

      <!-- FAQ content -->
      <div class="flex-1 min-w-0">
        {groups.map(group => (
          <div id={group.id} class="mb-10 scroll-mt-20">
            <h3 class="font-serif font-semibold text-xl text-foreground mb-4">{group.title}</h3>
            <div class="space-y-2">
              {group.items.map(item => (
                <details class="bg-card border border-border rounded-lg group">
                  <summary class="px-5 py-4 cursor-pointer text-sm font-medium text-foreground hover:text-primary-600 transition-colors duration-150 list-none flex items-center justify-between">
                    {item.q}
                    <svg class="w-4 h-4 flex-shrink-0 ml-2 text-muted-foreground group-open:rotate-180 transition-transform duration-200" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
                  </summary>
                  <div class="px-5 pb-4 text-sm text-muted-foreground leading-relaxed">
                    {item.a}
                  </div>
                </details>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  </Section>

  <CTASection
    title="还没开始？"
    description="免费注册，马上体验 AI 简历生成。"
    buttonText="免费开始使用"
  />
</BaseLayout>
```

**Step 2: Verify**

Run: `cd frontend/marketing && bun run build`
Expected: Help page builds with 4 groups, sidebar nav, FAQ list, CTA.

---

### Task 13: Final Verification — Build + Visual Check

**Step 1: Production build**

Run: `cd frontend/marketing && bun run build`
Expected: All 4 pages build successfully. No errors. Output in `dist/`.

**Step 2: Check output structure**

Run: `ls frontend/marketing/dist/`
Expected: `index.html`, `features/index.html`, `pricing/index.html`, `help/index.html`

**Step 3: Visual check (if dev server available)**

Run: `cd frontend/marketing && bun run dev`
Navigate to:
- `http://localhost:3000/` — Hero, features, steps, CTA
- `http://localhost:3000/features` — 5 alternating sections
- `http://localhost:3000/pricing` — 2 pricing cards, FAQ
- `http://localhost:3000/help` — 4 FAQ groups with sidebar

**Step 4: Mobile check**

Resize browser to 375px. Verify:
- Nav collapses to hamburger menu (click to toggle)
- Hero title wraps without overflow
- Feature cards stack vertically
- Pricing cards stack vertically
- Help sidebar disappears, FAQ takes full width

---

### Task 14: Commit

```bash
cd frontend/marketing
git add src/components/ src/styles/ src/layouts/ src/pages/ tailwind.config.mjs
git commit -m "feat: complete marketing site with Warm Editorial design"
```
