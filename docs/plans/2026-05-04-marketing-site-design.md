# ResumeGenius 营销站设计文档

日期：2026-05-04

## 1. 目标与受众

- **核心目标**：混合型——获客转化 + 帮助支持
- **受众**：求职者（有简历想优化 / 没简历想生成）
- **转化目标**：点击"开始使用"直接进入 `/app` 工作台
- **语言**：中文为主，组件设计预留 i18n（文案通过 props 驱动）

## 2. 技术方案

**方案 A：轻量纯 Astro**——只用 `.astro` 模板 + Tailwind CSS，手写组件，几乎零 JS。

| 项目 | 选型 |
|------|------|
| 框架 | Astro 5（纯静态输出 `output: 'static'`） |
| 样式 | Tailwind CSS 3（`theme.extend` 翻译 Warm Editorial Token） |
| 组件 | 手写 `.astro` 组件（~8 个） |
| 交互 | 原生 `<details>`（FAQ 折叠）、少量 vanilla JS（移动端汉堡菜单） |
| 字体 | Google Fonts：Playfair Display + Inter + JetBrains Mono |
| 图标 | Lucide SVG（内联 Astro 组件） |

## 3. 页面结构

```
/              ← Hero + 功能亮点 + CTA
/features      ← 功能详解（5 个 Section，对比叙事）
/pricing       ← Free/Pro 双卡 + 积分充值 + FAQ
/help          ← 帮助文档（4 组 FAQ，锚点导航）
/app           ← 外部链接到 Vite SPA 工作台
```

## 4. 设计系统

### 4.1 色彩：Warm Editorial 驼色

沿用 `docs/01-product/ui-design-system.md` 的完整 Token，翻译为 Tailwind v3 格式。

| Token | Hex | 营销站用途 |
|-------|-----|----------|
| `primary-400` | `#c4956a` | 主按钮、链接、高亮 |
| `primary-50` | `#faf6f2` | Hero 渐变背景起点 |
| `background` | `#faf8f5` | 页面基底 |
| `foreground` | `#1a1815` | 正文 |
| `card` | `#ffffff` | 卡片背景 |
| `muted-foreground` | `#8c8279` | 辅助文字 |
| `border` | `#e4ddd5` | 边框、分隔线 |

### 4.2 字体

| 角色 | 字体 | 营销站规格 |
|------|------|----------|
| Hero 标题 | Playfair Display 600 | 48-56px, leading 1.2 |
| Section 标题 | Playfair Display 500 | 28-32px |
| 正文 | Inter 400 | 16-18px, leading 1.6 |
| 辅助/标签 | Inter 400 | 14px |
| 代码 | JetBrains Mono 400 | 13px |

### 4.3 视觉增强（vs 工作台）

营销站在 Warm Editorial 基础上允许：
- Hero 区域微妙渐变背景（`primary-50 → background`）
- 更大字级（Hero 标题可达 56px vs 工作台 28px）
- Section 间大间距（48-64px）
- 交互 hover 动效（150-300ms ease）
- 不使用 box-shadow（一致原则）

## 5. 组件清单

### 布局级（2 个）
- `Nav.astro` — 固定顶部导航，移动端汉堡菜单（~15 行 vanilla JS）
- `Footer.astro` — 页脚，简版权 + 链接

### 内容级（6 个）
- `Hero.astro` — 首页大标题 + 副标题 + 3 迷你功能卡 + CTA
- `FeatureSection.astro` — 功能页 Section（左右交错布局，图示 + 文字）
- `PricingCard.astro` — 定价方案卡片（支持 highlight 推荐）
- `FaqGroup.astro` — FAQ 分组容器 + `<details>` 列表
- `CTASection.astro` — CTA 横幅（标题 + 描述 + 按钮）
- `Section.astro` — 通用区块容器（标题 + 插槽）

## 6. 各页详细设计

### 6.1 首页 `/`

Hero 文案（基于 README.md）：

- **主标题**："有简历？上传优化。没简历？AI 帮你从零生成。"
- **副标题**："ResumeGenius 融合行业前辈的面试经验与简历最佳实践，AI 不再只是润色文字 —— 它会重构结构、组织表达、主动搜集你的 GitHub / 文件 / 经历，直接生成可用简历。"
- **3 迷你卡**：AI 生成（结合最佳实践写内容）、自由编辑（TipTap 无表单约束）、所见即所得（chromedp 一致导出）
- **CTA**：[ 免费开始使用 ] + [ 了解更多 ↓ ]

Section 顺序：
1. Hero（含迷你功能卡）
2. 核心功能亮点（3 列 FeatureCard）
3. 使用流程（3 步骤：上传 → 编辑 → 导出）
4. CTA 横幅
5. Footer

### 6.2 功能介绍页 `/features`

对比叙事，5 个 Section：

1. **资料搜集**："你不需要准备完整的简历材料" — Sub-Agent 自动提取
2. **AI 生成**："不是简单的文本润色" — 融合最佳实践，一步到位
3. **自由编辑**："像编辑 Word 一样编辑简历" — TipTap 无表单约束
4. **AI 对话修改**："用自然语言控制排版" — AI 直接操作 HTML
5. **PDF 导出**："所见即所得，一字不差" — chromedp 服务端渲染

布局：左右交错（偶数 Section 图示左、文字右；奇数反之）。

### 6.3 定价页 `/pricing`

积分制：

| 维度 | Free | Pro（¥19/月） |
|------|------|-------------|
| 积分 | 注册送 100（一次性） | 每月送 1,900 |
| 功能 | 全部开放 | 全部开放 |
| AI 对话 | 消耗积分 | 消耗积分 |
| PDF 导出 | 10 积分/次 | 无限 |
| 充值折扣 | 无 | 9 折 |

- 积分充值：1 人民币 = 100 积分（约 100k token）
- Pro 权益重点：无限导出 + 充值折扣
- FAQ：积分换算（1 积分 ≈ 1k token，输入输出同价）、导出消耗、充值方式、升级路径

### 6.4 帮助文档页 `/help`

4 组 FAQ，左侧 sticky 目录导航（桌面端），`<details>` 原生实现：

1. **快速入门**（4 问）：创建项目、上传生成初稿、AI 对话修改、导出 PDF
2. **AI 对话**（4 问）：AI 能做什么、有效指令技巧、积分与 token、不满意怎么办
3. **编辑与导出**（4 问）：编辑器技巧、版本回退、导出一致性、模板风格
4. **账号与付费**（3 问）：Free vs Pro、充值方式、积分有效期

## 7. Tailwind 配置要点

```js
// tailwind.config.mjs — 关键扩展
module.exports = {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#faf6f2', 100: '#f2e8e0', 200: '#e5d2c2',
          300: '#d4b99a', 400: '#c4956a', 500: '#b3804d',
          600: '#9c6b3a', 700: '#7d5530', 800: '#5e4025',
          900: '#3f2b1a', 950: '#2a1c10',
        },
        background: '#faf8f5',
        foreground: '#1a1815',
        card: { DEFAULT: '#ffffff', foreground: '#1a1815' },
        muted: { DEFAULT: '#f5f1ed', foreground: '#8c8279' },
        secondary: { DEFAULT: '#e8d5c4', foreground: '#5c4a3a' },
        accent: { DEFAULT: '#d4a574', foreground: '#4a3020' },
        destructive: { DEFAULT: '#d64545', foreground: '#ffffff' },
        border: '#e4ddd5',
        input: '#e4ddd5',
        ring: '#c4956a',
      },
      fontFamily: {
        serif: ['Playfair Display', 'serif'],
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      fontSize: {
        'hero': ['3rem', { lineHeight: '1.15', fontWeight: '600' }],
        'hero-md': ['3.5rem', { lineHeight: '1.15', fontWeight: '600' }],
      },
    },
  },
}
```

## 8. 非目标（本阶段不做）

- 不做移动端深度适配（后续迭代）
- 不做博客/内容营销
- 不做多语言（i18n 结构预留但不实现翻译）
- 不做用户认证集成（营销站本身无登录）
- 不做暗色模式（后续迭代）

## 9. 文件结构

```
frontend/marketing/src/
├── layouts/
│   └── BaseLayout.astro    ← 重构：加入 Nav + Footer
├── pages/
│   ├── index.astro         ← 重写
│   ├── features.astro      ← 重写
│   ├── pricing.astro       ← 重写
│   ├── help.astro          ← 重写
│   └── app/                ← 如果需代理到 Vite SPA（视部署方案）
├── components/
│   ├── Nav.astro
│   ├── Footer.astro
│   ├── Hero.astro
│   ├── FeatureSection.astro
│   ├── PricingCard.astro
│   ├── FaqGroup.astro
│   ├── CTASection.astro
│   └── Section.astro
├── icons/                   ← Lucide SVG 内联组件
└── styles/
    └── global.css           ← Tailwind @tailwind directives
```
