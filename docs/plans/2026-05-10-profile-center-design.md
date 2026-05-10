# 个人中心设计日志

## 背景

为 ResumeGenius 工作台搭建完整的个人中心页面，涵盖个人信息管理、头像上传、积分可视化仪表盘、套餐商城四大模块。用户数据来自后端 `/auth/me` 接口，所有操作通过 API 实时同步。

## 页面布局

```
┌──────────────────────────────────────────────┐
│ ← 返回   个人中心                 [主题切换]  │
├──────────┬───────────────────────────────────┤
│          │                                   │
│ 头像卡片 │   Tab 内容区                       │
│ 套餐卡片 │   (个人信息/积分面板/套餐商城/密码) │
│          │                                   │
│ 导航 Tab │                                   │
│          │                                   │
│[管理后台]│                                   │
└──────────┴───────────────────────────────────┘
```

- 最大宽度 1400px，左侧栏固定 280px
- 响应式：移动端垂直堆叠

## 功能模块

### 1. 头像上传

- 点击头像触发文件选择器
- 客户端 Canvas 压缩：最大 256×256，JPEG Q80
- 服务端二次压缩：`golang.org/x/image/draw` CatmullRom 缩放
- 存储：`./uploads/avatars/{userID}.jpg`
- 无头像时显示渐变首字母
- Hover 显示相机图标覆盖层
- 上传中显示 loading spinner

### 2. 套餐计划卡片

- 展示当前套餐名称（Free / Pro）
- `font-serif italic` 衬线斜体大字
- Free：`animate-plan-title` 呼吸 + 字间距动效，hover 光晕缩放
- Pro：渐变色 `gradient-text`
- 显示开通时间和到期时间
- Free 套餐显示「永久有效」（绿色）

### 3. 个人信息编辑

- 昵称输入（2-32 字符）
- 邮箱只读展示
- 表单字段带浮动动画图标（UserRound / 信封）
- 保存成功/失败提示

### 4. 积分面板

可视化仪表盘，三区域布局：
- **统计卡片**（3 列）：可用积分（金币图标）、本月已用、累计获得
- **图表行**：
  - 面积图 — 近 30 天积分使用趋势（消耗橙色、获得绿色、渐变填充）
  - 环形图 — 消耗类别分布（AI 对话、PDF 导出等）
- **积分流水表**：时间、类型、数量、余额、备注

### 5. 套餐商城

两层布局：
- **会员方案**：Free（当前方案，按钮禁用） / Pro（¥19/月，升级入口）
- **积分充值**：4 档套餐卡片

| 套餐 | 价格 | 到账 | 赠送 | 折扣 |
|------|------|------|------|------|
| 尝鲜包 | ¥1 | 100 | — | 基准 |
| 进阶包 | ¥5 | 600 | +100 | 8.3折 |
| 专业包 | ¥10 | 1200 | +200 | 8.3折 |
| 旗舰包 | ¥29 | 3400 | +500 | 8.5折 |

- 专业包蓝色高亮边框 + 光晕阴影
- 旗舰包金色边框 + 光晕
- 点击购买先检查登录状态，未登录跳登录页（redirect 回定价页）
- 支付功能预留（提示「即将上线」）

### 6. 密码修改

- 原密码 + 新密码 + 确认密码
- 新密码最少 6 位
- 密码图标：KeyRound / LockKeyhole / ShieldCheck

## API 依赖

| 端点 | 用途 |
|------|------|
| `GET /auth/me` | 获取用户信息 |
| `PUT /auth/profile` | 修改昵称 |
| `POST /auth/avatar` | 上传头像 |
| `PUT /auth/password` | 修改密码 |
| `GET /auth/points/stats` | 积分统计 |
| `GET /auth/points/records` | 积分流水 |
| `GET /auth/points/dashboard` | 图表数据 |

## 前端技术

- **图标动画**：`animate-icon-float` 2.4s ease-in-out 上下浮动 + 微缩放
- **套餐标题动效**：`animate-plan-title` 呼吸透明度 + 字间距 + hover 光晕
- **图表**：recharts（AreaChart + PieChart）
- **金币图标**：独立 SVG 组件（PointsCoin），金色渐变 + 脉冲光晕 + 星光粒子

## 改动文件清单

| 文件 | 改动 |
|------|------|
| `frontend/workbench/src/pages/ProfilePage.tsx` | 个人中心完整页面 |
| `frontend/workbench/src/components/ui/PointsCoin.tsx` | 金币图标组件 |
| `frontend/workbench/src/components/ui/user-menu.tsx` | 导航栏头像显示真实图片 |
| `frontend/workbench/src/lib/api-client.ts` | API 类型和方法扩展 |
| `frontend/workbench/src/index.css` | 动画 keyframes（icon-float, plan-shimmer） |
| `backend/internal/modules/auth/service.go` | 头像上传/积分查询服务 |
| `backend/internal/modules/auth/handler.go` | userResp 扩展 |
| `docs/plans/2026-05-10-profile-center-design.md` | 本日志 |
