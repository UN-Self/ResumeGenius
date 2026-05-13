# 管理员后台完整设计方案

## 概述

ResumeGenius 管理员后台是一个**独立部署**的管理面板，通过 URL 路径 `/unself666` 访问，与工作台前端完全分离。三级权限体系（站长 / 管理员 / 用户）控制功能可见性。后台覆盖用户管理、积分运营、系统配置、数据分析、内容管理、审计日志六大领域。

## 部署方案

```
Nginx 路由规则：
/               → 营销站（Astro 静态）
/app/*          → 工作台（Vite React SPA）
/api/*          → Go 后端
/unself666      → 管理后台（Nginx 层鉴权，仅管理员可访问）
/api/v1/u666/*  → 管理后台 API（非标准路径，避免暴露 admin 关键字）
```

- 管理后台是**独立的前端项目**，不嵌入工作台
- **Nginx 层做鉴权**：普通用户访问 `/unself666` 时，Nginx 先请求后端校验 JWT + 角色，不通过直接返回 403，**前端代码不会下发**
- 后台有自己的登录页，独立于工作台的认证流程
- 即使有人知道 `/unself666` 这个路径，没有管理员权限连 HTML 都拿不到

**Nginx 配置（auth_request 方案）：**

```nginx
# 管理后台鉴权：先验证身份和角色，通过后才返回静态文件
location /unself666 {
    # 向后端发起内部鉴权请求
    auth_request /u666-auth-check;
    auth_request_set $auth_role $upstream_http_x_admin_role;

    # 鉴权通过后，转发角色信息给应用层
    proxy_set_header X-Admin-Role $auth_role;

    alias /var/www/admin;
    try_files $uri $uri/ /unself666/index.html;
}

# 内部鉴权接口（不对外暴露）
location = /u666-auth-check {
    internal;
    proxy_pass http://127.0.0.1:8080/api/v1/u666/auth-check;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header X-Original-URI $request_uri;
}
```

**后端鉴权接口** `GET /api/v1/u666/auth-check`：
- 从请求中提取 JWT cookie
- 验证 token 有效性 + 角色是否为 superadmin/admin
- 通过返回 `200` + `X-Admin-Role` header
- 失败返回 `403`，Nginx 直接拦截，前端代码不下发

**安全层级**：

| 层级 | 机制 | 效果 |
|------|------|------|
| Nginx 层 | auth_request | 普通用户连 HTML/JS 都拿不到 |
| 后端 API 层 | JWT + AdminRequired 中间件 | 即使绕过 Nginx，接口也打不通 |
| 前端层 | 路由守卫 + 角色检查 | 管理员操作的二次确认 |

## 权限体系

| 角色 | 标识 | 权限范围 |
|------|------|----------|
| 站长 | `superadmin` | 全部功能，可任命/撤换管理员 |
| 管理员 | `admin` | 用户查看、积分发放、系统设置（不含删除/恢复类高危操作） |
| 用户 | `user` | 无后台访问权限 |

- 首个注册用户自动成为站长
- 站长唯一且不可删除
- 管理员无权修改站长或自身权限
- 所有管理操作写入审计日志

---

## 功能模块全景

```
管理后台（/unself666）
├── 1. 仪表盘总览
├── 2. 用户管理
├── 3. 积分运营
├── 4. 套餐管理
├── 5. 系统设置
├── 6. 数据分析
├── 7. 审计日志
└── 8. 内容管理（营销站）
```

---

## 1. 仪表盘总览

### 状态：[ ] 已实现 [x] 待实现

运营人员登录后台看到的第一个页面，聚合核心指标。

**指标卡片：**

| 指标 | 说明 |
|------|------|
| 总注册用户 | 累计注册数 |
| 今日新增 | 当日注册数 |
| 活跃用户（7d） | 近 7 天有 AI 对话或导出操作的用户 |
| 付费转化率 | Pro 用户占比 |
| 积分发行总量 | 系统累计发放积分 |
| 积分消耗总量 | 用户累计消耗积分 |
| 本月收入 | 预估收入（积分充值 × 汇率） |
| API 调用量 | 近 24h AI 请求次数 |

**图表：**

- 日活趋势折线图（30d）
- 积分发行 vs 消耗对比柱状图（30d）
- 用户注册来源饼图（营销站 / 直接访问 / 邀请链接）
- Pro 转化漏斗（注册 → 活跃 → 首次充值 → Pro）

**数据来源：** `GET /api/v1/u666/dashboard`

---

## 2. 用户管理

### 状态：[x] 已实现基础版 [ ] 待完善

**基础功能（已实现）：**
- 用户列表表格：用户名、邮箱、验证状态、积分、套餐、角色
- 权限标签彩色展示（站长红 / 管理员琥珀 / 用户灰）
- 站长可下拉修改用户角色

**待扩展功能：**

| 功能 | 说明 |
|------|------|
| 搜索/筛选 | 按用户名、邮箱、角色、套餐筛选 |
| 分页 | 后端分页，避免大量数据前端卡顿 |
| 用户详情 | 点击展开：注册时间、最后登录、AI 使用次数、导出次数、积分流水 |
| 禁用/启用 | 临时封禁用户（禁止登录和 API 调用） |
| 删除用户 | 软删除（保留数据关联）或彻底删除 |
| 批量操作 | 选中多用户批量发放积分、修改套餐、导出 CSV |
| 邀请链追溯 | 查看用户由谁邀请注册 |

**API：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/u666/users` | 用户列表（分页+筛选） |
| GET | `/u666/users/:id` | 用户详情 |
| PUT | `/u666/users/:id/role` | 修改角色 |
| PUT | `/u666/users/:id/status` | 启用/禁用 |
| DELETE | `/u666/users/:id` | 删除用户 |
| POST | `/u666/users/batch` | 批量操作 |

---

## 3. 积分运营

### 状态：[ ] 已实现 [x] 待实现

独立的积分发放与回收工具，支持手动调节用户积分。

**功能：**

| 功能 | 说明 |
|------|------|
| 手动发放 | 选择用户 → 输入积分数量 → 填写备注原因 → 确认发放 |
| 批量发放 | 按角色/套餐筛选 → 统一发放（如「所有 Pro 用户赠送 500 积分」） |
| 积分回收 | 扣除用户积分（需填写原因，如刷分处理） |
| 发放模板 | 预设模板：「注册奖励」「活动赠送」「Bug 补偿」「人工充值」 |
| 发放记录 | 所有管理端积分操作列表，可按操作人、时间、类型筛选 |

**API：**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/u666/points/grant` | 手动发放积分 |
| POST | `/u666/points/batch-grant` | 批量发放 |
| POST | `/u666/points/revoke` | 回收积分 |
| GET | `/u666/points/records` | 管理操作积分记录 |

---

## 4. 套餐管理

### 状态：[ ] 已实现 [x] 待实现

管理用户的套餐订阅状态。

**功能：**

| 功能 | 说明 |
|------|------|
| 套餐修改 | 手动为用户升级/降级套餐（Free ↔ Pro） |
| 到期管理 | 查看即将到期用户，手动续期 |
| 套餐配置 | 修改 Free/Pro 的权益描述和定价（同步更新营销站定价页） |
| 套餐统计 | 各套餐用户数、占比、趋势 |

**API：**

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | `/u666/users/:id/plan` | 修改用户套餐 |
| PUT | `/u666/users/:id/plan-expiry` | 修改到期时间 |
| GET | `/u666/plans/stats` | 套餐统计数据 |
| PUT | `/u666/settings/plan-config` | 套餐配置 |

---

## 5. 系统设置

### 状态：[x] 已实现基础版 [ ] 待扩展

**已实现：**
- PDF 导出定价（无水印 / 带水印）
- AI Token 单价

**待扩展设置项：**

| 分类 | 设置项 | 说明 |
|------|--------|------|
| **积分定价** | 积分基准汇率 | 1 元 = X 积分 |
| | 套餐赠送比例 | 各套餐的赠送百分比 |
| **AI** | Token 单价 | 积分 / 1k token |
| | 最大对话轮次 | Agent 单次最大迭代数 |
| | 模型选择 | 可用模型列表 |
| **导出** | PDF 无水印价格 | 积分/次 |
| | PDF 带水印价格 | 积分/次 |
| | 水印文字 | 自定义水印内容 |
| **注册** | 开放注册 | 开启/关闭新用户注册 |
| | 注册验证 | 邮箱验证开启/关闭 |
| | 注册赠送积分 | 新用户默认赠送数量 |
| **安全** | 单日最大注册 IP | 防刷 |
| | 登录失败锁定 | N 次失败锁定 M 分钟 |
| **通知** | SMTP 配置 | 邮件服务器配置 |
| | 邮件模板 | 验证码/欢迎/密码重置邮件模板 |

**API：** `GET/PUT /u666/settings`（key-value 自由扩展）

---

## 6. 数据分析

### 状态：[ ] 已实现 [x] 待实现

独立的可视化分析页面，帮助运营决策。

**功能：**

| 模块 | 图表类型 | 说明 |
|------|----------|------|
| 用户增长 | 折线图 | 每日注册数（30d/90d/365d） |
| 用户活跃 | 热力图 | 各时段 AI 调用量分布 |
| 积分流转 | 桑基图 | 积分发行→消耗的流向 |
| AI 用量 | 柱状图 | 各模型 Token 消耗排名 |
| 导出统计 | 饼图 | 无水印/带水印导出比例 |
| 收入概览 | 面积图 | 日收入趋势 |
| 用户留存 | 表格 | D1/D7/D30 留存率 |

**数据来源：** 定时任务聚合统计表 + 实时查询

---

## 7. 审计日志

### 状态：[ ] 已实现 [x] 待实现

记录所有管理操作，满足安全审计需求。

**数据模型：**

```go
type AuditLog struct {
    ID         uint
    AdminID    string    // 操作管理员 ID
    AdminName  string    // 冗余存储，防止用户删除后无法追溯
    Action     string    // grant_points, change_role, update_setting...
    TargetType string    // user, setting...
    TargetID   string    // 操作对象 ID
    Detail     string    // JSON 详情（变更前后对比）
    IP         string
    CreatedAt  time.Time
}
```

**记录的 action 类型：**
- `grant_points` / `revoke_points` — 积分操作
- `change_role` — 修改角色
- `change_plan` — 修改套餐
- `update_setting` — 修改系统设置
- `disable_user` / `enable_user` — 用户封禁
- `delete_user` — 删除用户

**API：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/u666/audit-logs` | 审计日志列表（分页+筛选） |

---

## 8. 内容管理（营销站）

### 状态：[ ] 已实现 [x] 待实现

允许运营人员在不改代码的情况下更新营销站内容。

**功能：**

| 功能 | 说明 |
|------|------|
| Hero 文案 | 主标题、副标题、CTA 按钮文字 |
| 功能亮点 | 功能卡片标题和描述 |
| 定价页 | 套餐名称、价格、权益列表、FAQ |
| 页脚 | 联系方式、社交媒体链接 |
| SEO | 页面标题、描述、OG 标签 |

**数据模型：** 复用 SystemSetting 表，以命名空间区分：
- `content.hero.title`、`content.hero.subtitle`、`content.hero.cta`
- `content.features.0.title`、`content.features.0.desc`
- `content.pricing.free.features`（JSON 数组）
- `content.faq.0.q`、`content.faq.0.a`

营销站页面通过 API 动态读取内容，fallback 到代码默认值。

---

## 技术实现路线

### Phase 1（已完成 — 当前分支）

- [x] 用户管理基础版（列表 + 角色修改）
- [x] 系统设置基础版（PDF 价格 + AI Token 价格）
- [x] 权限体系（站长/管理员/用户）
- [x] SystemSetting 数据模型
- [x] 管理后台独立前端项目骨架

### Phase 2（短期）

- [ ] 仪表盘总览（核心指标 + 图表）
- [ ] 用户管理完善（搜索、分页、详情、禁用）
- [ ] 积分运营（手动发放/批量发放/回收）
- [ ] Nginx 路由配置（`/unself666`）

### Phase 3（中期）

- [ ] 套餐管理（手动升级/统计/配置）
- [ ] 数据分析页面
- [ ] 系统设置扩展（注册控制、安全配置）
- [ ] 审计日志

### Phase 4（长期）

- [ ] 内容管理（营销站动态配置）
- [ ] 邮件模板编辑器
- [ ] 用户行为分析（留存、转化漏斗）
- [ ] 多语言管理界面

---

## 数据库扩展

```
现有表：
  users (已扩展 role, plan, points)
  points_records (积分流水)
  system_settings (key-value 配置)

新增表：
  audit_logs (审计日志)
  analytics_daily (每日聚合统计，定时任务写入)
```

---

## 安全设计

**三层防御：**

| 层级 | 机制 | 防什么 |
|------|------|--------|
| Nginx 层 | auth_request 鉴权 | 普通用户连管理后台的 HTML/JS 都拿不到，逆向也看不到代码 |
| 后端 API 层 | JWT + AdminRequired 中间件 | 即使绕过 Nginx 直接调接口，角色不对也返回 403 |
| 前端层 | 路由守卫 + 角色检查 | 管理员操作的二次确认，防止误操作 |

**其他安全措施：**
- 站长专属操作（修改角色、删除用户）额外 SuperadminRequired 检查
- 所有积分/权限修改操作写入审计日志
- 敏感设置变更（SMTP 密码等）不返回明文
- IP 频率限制：管理 API 单 IP 60 次/分钟
- 管理员会话超时：30 分钟无操作需重新验证

---

## 改动文件清单

### 已实现（当前分支）

| 文件 | 改动 |
|------|------|
| `backend/internal/shared/models/models.go` | User.Role、SystemSetting |
| `backend/internal/shared/database/database.go` | AutoMigrate |
| `backend/internal/modules/admin/service.go` | 用户管理/设置服务 |
| `backend/internal/modules/admin/handler.go` | 处理器 + 权限中间件 |
| `backend/internal/modules/admin/routes.go` | 路由注册 |
| `backend/cmd/server/main.go` | 模块注册 |
| `backend/internal/modules/auth/service.go` | 首个用户 superadmin |
| `backend/internal/modules/auth/handler.go` | userResp 扩展 |

### 待实现

| 文件 | 改动 |
|------|------|
| `backend/internal/shared/models/models.go` | AuditLog、AnalyticsDaily |
| `backend/internal/modules/admin/handler.go` | 积分运营、数据统计 |
| `backend/internal/modules/admin/service.go` | 对应服务层 |
| `backend/internal/modules/admin/middleware.go` | 频率限制中间件 |
| `backend/internal/modules/admin/audit.go` | 审计日志写入工具 |
| `frontend/admin/` | **独立管理后台前端项目** |
| `frontend/admin/src/pages/` | 仪表盘、积分运营、用户管理 UI |
| `nginx.conf` | 添加 `/unself666` 路由规则 |

---

*文档版本：v3.0 — 2026-05-12*
