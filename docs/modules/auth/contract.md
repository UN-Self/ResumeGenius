# 模块 auth 契约：登录与身份识别

更新时间：2026-04-28

## 1. 角色定义

**负责**：

- 用户登录（首次登录可自动创建账号）
- JWT 签发与身份识别
- 当前登录用户查询
- 退出登录（清理 cookie）

**不负责**：

- 用户资料管理
- 权限分级（RBAC）
- 第三方 OAuth 登录

## 2. API 端点

遵循 [api-conventions.md](../../01-product/api-conventions.md)。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/auth/login` | 登录（不存在则注册） |
| GET | `/api/v1/auth/me` | 获取当前用户 |
| POST | `/api/v1/auth/logout` | 退出登录 |

### 2.1 POST /api/v1/auth/login

```json
{
  "username": "alice",
  "password": "pass123456"
}
```

行为：
- 若 `username` 不存在：创建用户并登录
- 若存在且密码正确：直接登录
- 若存在但密码错误：返回 `40100`

响应：

```json
{
  "code": 0,
  "data": {
    "id": "2d98cb4f-0f7c-4f8d-a6e0-8d724e9ef0a2",
    "username": "alice"
  },
  "message": "ok"
}
```

同时设置 `HttpOnly` cookie：`rg_access_token=<jwt>`

### 2.2 GET /api/v1/auth/me

请求：携带 `rg_access_token` cookie。

响应：

```json
{
  "code": 0,
  "data": {
    "id": "2d98cb4f-0f7c-4f8d-a6e0-8d724e9ef0a2",
    "username": "alice"
  },
  "message": "ok"
}
```

未登录或 token 无效：

```json
{
  "code": 40100,
  "data": null,
  "message": "unauthorized"
}
```

### 2.3 POST /api/v1/auth/logout

认证要求：`无需认证`（public 路由）。
请求：可带或不带 cookie（幂等）。

响应：

```json
{
  "code": 0,
  "data": null,
  "message": "ok"
}
```

## 3. JWT 约定

- 算法：HS256
- Claims：`sub`（user_id）、`username`、`iat`、`exp`
- 默认有效期：365 天（`JWT_TTL_HOURS=8760`）
- token 存储位置：`HttpOnly` cookie `rg_access_token`

## 4. 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 40000 | 400 | 参数错误（用户名/密码长度不合法、JSON 结构错误） |
| 40100 | 401 | 未认证、token 无效、用户名密码不匹配 |
| 50000 | 500 | 服务内部错误 |
