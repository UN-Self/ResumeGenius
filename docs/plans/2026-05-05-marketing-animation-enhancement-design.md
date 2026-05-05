# 营销站动画与深度感增强设计

**日期**: 2026-05-05
**目标**: 为营销站添加克制精致的动画/特效，减少扁平化，提升品质感

## 决策记录

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 视觉风格 | 克制精致 | 与品牌调性一致，不喧宾夺主 |
| 技术方案 | 纯 CSS + 原生 JS | 零依赖，SSR 友好，维护简单 |
| 实现策略 | 渐进增强层 | 改动最小，通过全局 CSS 类注入动画层 |

## 1. 基础动画系统

### 1.1 Tailwind 自定义 keyframes

在 `tailwind.config.mjs` 的 `extend` 中添加：

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

### 1.2 全局 CSS `.animate-on-scroll` 类

```css
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
```

支持 `--delay` CSS 变量实现交错入场：`style="--delay: 100ms"`

### 1.3 prefers-reduced-motion

```css
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

### 1.4 IntersectionObserver 脚本

在 `BaseLayout.astro` 的 `</body>` 前添加：

```js
const observer = new IntersectionObserver(
  (entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add('is-visible');
        observer.unobserve(entry.target);
      }
    });
  },
  { threshold: 0.1, rootMargin: '0px 0px -50px 0px' }
);
document.querySelectorAll('.animate-on-scroll').forEach((el) => observer.observe(el));
```

## 2. 组件级效果增强

### 2.1 卡片悬浮效果

应用于：Feature Cards、Pricing Cards、FAQ items

```css
/* 在 global.css 中添加 */
.card-hover {
  transition: transform 200ms ease-out, box-shadow 200ms ease-out;
}
.card-hover:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(42, 28, 16, 0.08);
}
```

使用铜色阴影 `rgba(42, 28, 16, 0.08)` 与品牌色一致。

### 2.2 按钮交互反馈

**主按钮** (`btn-primary`):
- hover: `translateY(-1px)` + `box-shadow(0 4px 12px rgba(196, 149, 106, 0.3))`
- active: `scale(0.98)` + 无阴影
- transition: 150ms

**次按钮** (`btn-secondary`):
- hover: 已有 `bg-muted`，增加 `translateY(-1px)`
- active: `scale(0.98)`

### 2.3 FAQ 展开/收起平滑过渡

使用 `grid` + `grid-template-rows: 0fr → 1fr` 技巧实现高度过渡：

```css
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
```

### 2.4 导航栏滚动增强

```css
.nav-scrolled {
  border-bottom: 1px solid rgba(228, 221, 213, 0.5);
}
```

JS 检测 `scrollY > 10` 时添加 `.nav-scrolled` 类，200ms 过渡。

### 2.5 Section 标题渐入

每个 `Section` 组件的标题和描述添加 `.animate-on-scroll`，描述带 `--delay: 100ms`。

## 3. 装饰层与深度感

### 3.1 Hero 装饰光斑（2 个）

| 位置 | 颜色 | 尺寸 | 动画 |
|------|------|------|------|
| 右上方 | `bg-primary-200/30` | ~400px | `animate-float 6s ease-in-out infinite` |
| 左下方 | `bg-accent/20` | ~300px | `animate-float-delayed 8s ease-in-out infinite` |

实现方式：`position: absolute` + `rounded-full` + `filter: blur(80px)` + `pointer-events: none` + `z-index: 0`

### 3.2 Feature Section 图片占位增强

- 添加内阴影 `shadow-[inset_0_2px_8px_rgba(0,0,0,0.06)]`
- 内部渐变叠加：`linear-gradient(135deg, rgba(250,246,242,0.5), transparent)`
- hover 时 `scale(1.02)` 300ms 过渡

### 3.3 Footer 渐变分隔线

```css
footer {
  border-image: linear-gradient(to right, transparent, #e4ddd5, transparent) 1;
}
```

### 3.4 背景纹理（可选，低优先级）

`body::before` 叠加极微弱的 SVG 噪点纹理（`opacity: 0.02`），增加"纸质感"减少"数字感"。

## 影响范围

| 文件 | 改动类型 |
|------|----------|
| `tailwind.config.mjs` | 添加 keyframes + animation utilities |
| `src/styles/global.css` | 添加 `.animate-on-scroll`、`.card-hover`、按钮类、reduced-motion |
| `src/layouts/BaseLayout.astro` | 添加 IntersectionObserver script + nav scroll JS |
| `src/components/Hero.astro` | 添加 2 个装饰光斑 div |
| `src/components/Section.astro` | 标题/描述添加 `.animate-on-scroll` |
| `src/components/FeatureSection.astro` | 图片占位增强 + 添加动画类 |
| `src/components/PricingCard.astro` | 添加 `.card-hover` 类 |
| `src/components/FaqGroup.astro` | 改为 grid 高度过渡 |
| `src/components/Nav.astro` | 添加 `.nav-scrolled` 逻辑 |
| `src/components/Footer.astro` | 渐变分隔线 |
| `src/pages/index.astro` | Feature cards + steps 添加动画类 |
| `src/pages/features.astro` | Feature sections 添加动画类 |
| `src/pages/pricing.astro` | Cards + FAQ 添加动画类 |
| `src/pages/help.astro` | FAQ groups 添加动画类 |
