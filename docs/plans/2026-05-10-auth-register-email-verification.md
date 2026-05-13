# 注册认证系统设计日志

## 背景

ResumeGenius 需要完整的用户注册与认证体系，支持邮箱验证码注册、多种登录方式、JWT 鉴权。同时为后续积分系统和套餐体系提供用户基础数据支撑。

本轮目标：
- 实现邮箱验证码注册流程
- 支持用户名/邮箱双模式登录
- 搭建 JWT Cookie 鉴权体系
- 为积分、套餐、管理后台提供 User 模型扩展

## 数据模型

### User（扩展字段）

在基础认证字段之上新增：

| 字段 | 类型 | 说明 |
|------|------|------|
| `email` | `*string` | 邮箱（唯一身份标识） |
| `email_verified` | `bool` | 邮箱验证状态 |
| `verification_code` | `string` | 6 位验证码（JSON 隐藏） |
| `code_expiry` | `*time.Time` | 验证码过期时间 |
| `avatar_url` | `*string` | 头像 URL |
| `points` | `int` | 当前积分余额 |
| `plan` | `string` | 当前套餐（free/pro） |
| `plan_started_at` | `*time.Time` | 套餐开通时间 |
| `plan_expires_at` | `*time.Time` | 套餐到期时间 |
| `role` | `string` | 权限角色（superadmin/admin/user） |

### PointsRecord

```go
type PointsRecord struct {
    ID        uint
    UserID    string    // 关联用户
    Amount    int       // 正数=获得，负数=消耗
    Balance   int       // 操作后余额
    Type      string    // register_bonus, ai_usage, pdf_export...
    Note      string
    CreatedAt time.Time
}
```

## API 设计

### 公开路由

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/register` | 注册（发送验证码） |
| POST | `/auth/login` | 用户名/邮箱登录 |
| POST | `/auth/send-code` | 重发验证码 |
| POST | `/auth/verify-email` | 验证邮箱 |
| GET | `/auth/check-username` | 检查用户名可用性 |
| GET | `/auth/check-email` | 检查邮箱可用性 |
| POST | `/auth/logout` | 退出登录 |
| GET | `/auth/avatar/:user_id` | 获取头像图片 |

### 需登录路由

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/auth/me` | 获取当前用户信息 |
| PUT | `/auth/profile` | 修改昵称 |
| PUT | `/auth/password` | 修改密码 |
| POST | `/auth/avatar` | 上传头像（压缩存储） |
| GET | `/auth/points/records` | 积分流水 |
| GET | `/auth/points/stats` | 积分统计 |
| GET | `/auth/points/dashboard` | 积分仪表盘（含趋势图数据） |

### 响应格式

```json
{
  "id": "uuid",
  "username": "string",
  "email": "string",
  "email_verified": true,
  "avatar_url": "/api/v1/auth/avatar/uuid",
  "points": 100,
  "plan": "free",
  "plan_started_at": "2026-05-10T00:00:00Z",
  "role": "superadmin",
  "dev_code": "123456"
}
```

## 注册流程

1. 用户填写用户名 + 密码 + 邮箱
2. 后端校验：用户名 3-64 字符、密码 6-128 字符、邮箱格式
3. 检查邮箱是否已被验证账户占用
4. 未验证的同邮箱账户 → 覆盖更新（密码、验证码重新生成）
5. 新邮箱 → 创建账户，默认 `plan=free`、`role=user`
6. **首个注册账户自动设为 `role=superadmin`（站长）**
7. **新注册赠送 100 积分**，写入 PointsRecord（`register_bonus`）
8. 发送 6 位验证码邮件到注册邮箱
9. 用户输入验证码完成邮箱验证

## 邮箱验证

- 飞书企业邮箱 SMTP（smtp.feishu.cn:465）
- 隐式 TLS 连接（SMTPS），非 STARTTLS
- 验证码 15 分钟有效
- 开发模式：SMTP 未配置时验证码打印到 stdout
- 中文邮件内容：标题「ResumeGenius 邮箱验证」、正文含验证码和有效期提示

## 前端注册页面

- `/app/register` — 注册页
- 表单：用户名 + 邮箱 + 密码 + 验证码
- 发送验证码按钮，60 秒倒计时防重复
- 注册成功自动登录并跳转到首页
- 动效：网格涟漪背景 + 卡片入场动画

## 改动文件清单

| 文件 | 改动 |
|------|------|
| `backend/internal/shared/models/models.go` | User 字段扩展、PointsRecord 模型 |
| `backend/internal/modules/auth/service.go` | 注册/登录/验证/积分/头像服务 |
| `backend/internal/modules/auth/handler.go` | HTTP 处理器 |
| `backend/internal/modules/auth/routes.go` | 路由注册 |
| `backend/internal/modules/auth/email.go` | SMTP 邮件服务（465 SMTPS） |
| `frontend/workbench/src/pages/RegisterPage.tsx` | 注册页面 |
| `frontend/workbench/src/pages/LoginPage.tsx` | 登录页面 |
| `frontend/workbench/src/lib/api-client.ts` | AuthUser 类型、authApi 方法 |
| `docs/plans/2026-05-10-auth-register-email-verification.md` | 本日志 |
