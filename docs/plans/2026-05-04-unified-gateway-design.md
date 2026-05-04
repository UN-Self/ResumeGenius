# 营销站与工作台统一网关设计

日期: 2026-05-04

## 问题

文档规划了统一路由（`/` → 营销站, `/app/*` → 工作台, `/api/*` → Go 后端），但实际部署中三个服务各自独立运行，没有统一网关。营销站 CTA 按钮（"开始使用"、"免费开始使用"等）链接到 `/app`，但没有任何服务监听该路径。

## 决策

1. **新增 gateway nginx 容器**作为统一入口，proxy_pass 分发到各服务
2. **工作台采用 `/app/` 前缀**：Vite `base` + React Router `basename` 统一改为 `/app/`
3. **本地开发保持独立**：各服务单独跑（bun run dev / go run），不用 Docker
4. **Docker Compose 只用于生产部署**

## 架构

```
docker compose up
        │
   gateway (:80)          ← 新增，统一入口
    ┌────┼────┐
    │    │    │
    ▼    ▼    ▼
marketing  workbench  backend
(:80)      (:80)      (:8080)
(Astro)    (React)    (Go)
```

### 路由规则

| 路径 | 目标 | 说明 |
|------|------|------|
| `/` | `http://marketing:80` | 营销站静态页面 |
| `/features`, `/pricing`, `/help` | `http://marketing:80` | 营销站子页面 |
| `/app` | 301 → `/app/` | 重定向 |
| `/app/`, `/app/*` | `http://workbench:80` | SPA fallback |
| `/api/*` | `http://backend:8080` | Go API |
| `/api/v1/ai/*` | `http://backend:8080` | SSE 流式（proxy_buffering off） |

Gateway 只做路由分发，不做静态文件服务、gzip、缓存。

## 改动范围

### 1. 新增 gateway nginx

文件: `gateway/nginx.conf`

```nginx
server {
    listen 80;

    location / {
        proxy_pass http://marketing:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /app/ {
        proxy_pass http://workbench:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location = /app {
        return 301 /app/;
    }

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/v1/ai/ {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

文件: `gateway/Dockerfile`

```dockerfile
FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### 2. 工作台 `/app/` 前缀改造

**vite.config.ts**: 添加 `base: "/app/"`

**App.tsx**: BrowserRouter 添加 `basename="/app"`

**workbench/nginx.conf**: SPA fallback 改为 `try_files $uri $uri/ /app/index.html`

### 3. Docker Compose 改造

新增 `gateway` 服务，依赖 `marketing`、`workbench`、`backend`。`gateway` 暴露主机端口 80（或 8080 避免冲突）。`marketing` 和 `workbench` 不再暴露主机端口（可选保留供调试）。

### 4. 不改动的部分

- 后端代码（路由、CORS 已就绪）
- 营销站内容和组件
- 不加 HTTPS（后续部署阶段）
- 不加认证/登录跳转

## 本地开发

各服务独立运行，不用 Docker：

- workbench: `bun run dev` → `:3000`（Vite 自动代理 `/api`）
- marketing: `bun run dev` → `:4321`
- backend: `go run` → `:8080`

CORS 已配置（echo origin + credentials），跨端口开发无问题。营销站 CTA 链接 `/app` 在本地不生效（无 gateway），属预期行为。

## 方案选择记录

| 方案 | 描述 | 选择 | 原因 |
|------|------|------|------|
| A | Gateway proxy 分发到各服务 nginx | ✅ | 职责清晰，改动最小 |
| B | Gateway 直接服务静态文件 | ❌ | nginx.conf 过于复杂 |
| C | Go 后端做统一入口 | ❌ | 违反关注点分离 |
