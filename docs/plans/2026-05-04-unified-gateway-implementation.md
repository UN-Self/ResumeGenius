# 统一网关 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增 gateway nginx 容器统一路由，工作台改用 `/app/` 前缀，实现营销站 CTA 到工作台端到端连通。

**Architecture:** 在 docker-compose 中新增 gateway 容器，proxy_pass 分发 `/` → marketing, `/app/*` → workbench, `/api/*` → backend。工作台 Vite base 和 React Router basename 改为 `/app/`。

**Tech Stack:** nginx:alpine, Vite base path, React Router basename, Docker Compose

**Design doc:** `docs/plans/2026-05-04-unified-gateway-design.md`

---

### Task 1: 工作台 Vite base 路径改为 /app/

**Files:**
- Modify: `frontend/workbench/vite.config.ts`

**Step 1: 修改 vite.config.ts**

在 `defineConfig` 中添加 `base: "/app/"`：

```ts
export default defineConfig({
  base: '/app/',
  plugins: [react(), tailwindcss()],
  // ... 其余不变
})
```

**Step 2: 验证构建产物**

Run: `cd frontend/workbench && bun run build`
Expected: `dist/` 下的 index.html 中引用路径变为 `/app/assets/...`

Run: `grep -r "app" dist/index.html | head -5`
Expected: 能看到 `href="/app/..."` 或 `src="/app/..."` 的资源引用

**Step 3: 确认测试不受影响**

Run: `cd frontend/workbench && bunx vitest run`
Expected: 全部 PASS（Vitest 不使用 base 路径）

**Step 4: Commit**

```bash
git add frontend/workbench/vite.config.ts
git commit -m "feat: 工作台 Vite base 路径改为 /app/"
```

---

### Task 2: 工作台 React Router basename 改为 /app

**Files:**
- Modify: `frontend/workbench/src/App.tsx`

**Step 1: 修改 BrowserRouter**

将 `<BrowserRouter>` 改为 `<BrowserRouter basename="/app">`：

```tsx
<BrowserRouter basename="/app">
```

路由路径（`/login`, `/`, `/projects/:projectId` 等）保持不变，React Router 会自动加上 `/app` 前缀。

所有 `<Navigate>` 的 `to` 路径也不需要改动——`basename` 会在运行时统一处理。

**Step 2: 确认测试通过**

Run: `cd frontend/workbench && bunx vitest run`
Expected: 全部 PASS

注意：现有测试 mock 了 `react-router-dom`（参见 `tests/ProjectList.test.tsx:23-30`），直接调用 `useNavigate` mock，不受 `basename` 影响。

**Step 3: Commit**

```bash
git add frontend/workbench/src/App.tsx
git commit -m "feat: 工作台 React Router basename 改为 /app"
```

---

### Task 3: 工作台 nginx.conf 适配 /app/ 前缀

**Files:**
- Modify: `frontend/workbench/nginx.conf`

**Step 1: 更新 nginx.conf**

将 SPA fallback location 从 `/` 改为 `/app/`，`try_files` 指向 `/app/index.html`：

```nginx
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    client_max_body_size 20m;

    # API requests proxy to backend (保留，作为兜底)
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE support for AI chat
    location /api/v1/ai/ {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    # SPA fallback for /app/*
    location /app/ {
        try_files $uri $uri/ /app/index.html;
    }

    # 根路径重定向到 /app/
    location = / {
        return 301 /app/;
    }
}
```

**Step 2: 验证 nginx 配置语法**

Run: `docker run --rm -v $(pwd)/frontend/workbench/nginx.conf:/etc/nginx/conf.d/default.conf docker.1ms.run/nginx:alpine nginx -t`
Expected: `syntax is ok` + `test is successful`

**Step 3: Commit**

```bash
git add frontend/workbench/nginx.conf
git commit -m "feat: 工作台 nginx 适配 /app/ 前缀"
```

---

### Task 4: 新增 gateway nginx 容器

**Files:**
- Create: `gateway/nginx.conf`
- Create: `gateway/Dockerfile`

**Step 1: 创建 gateway/nginx.conf**

```nginx
server {
    listen 80;
    server_name _;

    # 营销站
    location / {
        proxy_pass http://marketing:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # /app (无尾部斜杠) → 重定向
    location = /app {
        return 301 /app/;
    }

    # 工作台 SPA
    location /app/ {
        proxy_pass http://workbench:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # API (通用)
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE (AI 对话) — 最长前缀匹配优先于 /api/
    location /api/v1/ai/ {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}
```

**Step 2: 创建 gateway/Dockerfile**

```dockerfile
FROM docker.1ms.run/nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

**Step 3: 验证 nginx 配置语法**

Run: `docker run --rm -v $(pwd)/gateway/nginx.conf:/etc/nginx/conf.d/default.conf docker.1ms.run/nginx:alpine nginx -t`
Expected: `syntax is ok` + `test is successful`

**Step 4: Commit**

```bash
git add gateway/nginx.conf gateway/Dockerfile
git commit -m "feat: 新增 gateway nginx 统一网关"
```

---

### Task 5: Docker Compose 集成 gateway

**Files:**
- Modify: `docker-compose.yml`

**Step 1: 在 docker-compose.yml 中添加 gateway 服务**

在 `services:` 下添加 `gateway`，同时调整 `workbench` 和 `marketing` 的端口暴露：

```yaml
  gateway:
    build:
      context: ./gateway
      dockerfile: Dockerfile
    ports:
      - "${HOST_GATEWAY_PORT:-80}:80"
    depends_on:
      marketing:
        condition: service_started
      workbench:
        condition: service_started
      backend:
        condition: service_started
    restart: always
```

同时修改 `workbench` 和 `marketing` 的端口——注释掉直接的端口映射（但仍可通过环境变量开启）：

```yaml
  workbench:
    build:
      context: ./frontend/workbench
      dockerfile: Dockerfile
    ports:
      - "${HOST_WORKBENCH_PORT:-}:80"   # 默认不暴露，通过 gateway 访问
    depends_on:
      backend:
        condition: service_started
    restart: always

  marketing:
    build:
      context: ./frontend/marketing
      dockerfile: Dockerfile
    ports:
      - "${HOST_MARKETING_PORT:-}:80"   # 默认不暴露，通过 gateway 访问
    restart: always
```

**Step 2: 验证 docker-compose 配置语法**

Run: `docker compose config --quiet`
Expected: 无报错输出

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: docker-compose 新增 gateway 服务"
```

---

### Task 6: 端到端验证

**Files:** 无代码改动，纯验证

**Step 1: 构建并启动所有服务**

Run: `docker compose up -d --build`
Expected: 所有 5 个服务（postgres, backend, workbench, marketing, gateway）启动成功

**Step 2: 验证营销站**

Run: `curl -s -o /dev/null -w "%{http_code}" http://localhost/`
Expected: `200`

Run: `curl -s http://localhost/ | head -5`
Expected: 能看到营销站 HTML 内容

**Step 3: 验证工作台 SPA**

Run: `curl -s -o /dev/null -w "%{http_code}" http://localhost/app/`
Expected: `200`

Run: `curl -s http://localhost/app/ | grep -o 'app/assets/[^"]*' | head -3`
Expected: 能看到 `/app/assets/...` 路径的资源引用

**Step 4: 验证 /app 重定向**

Run: `curl -s -o /dev/null -w "%{http_code}" http://localhost/app`
Expected: `301`

Run: `curl -sI http://localhost/app | grep Location`
Expected: `Location: /app/`

**Step 5: 验证 API 代理**

Run: `curl -s -o /dev/null -w "%{http_code}" http://localhost/api/v1/projects`
Expected: `401`（未认证，但说明代理到后端了）而非 `404` 或 `502`

**Step 6: 验证工作台内部路由**

Run: `curl -s -o /dev/null -w "%{http_code}" http://localhost/app/login`
Expected: `200`（SPA fallback 返回 index.html）

**Step 7: 停止服务**

Run: `docker compose down`
Expected: 所有容器停止

---

### 任务依赖图

```
Task 1 (Vite base)  ─┐
Task 2 (Router basename) ─┼─→ Task 3 (nginx.conf) ─→ Task 5 (docker-compose) ─→ Task 6 (验证)
                          │
Task 4 (gateway) ──────────────────────────────────────→ Task 5 (docker-compose) ─→ Task 6 (验证)
```

Task 1、2、4 互相独立，可以并行。Task 3 依赖 Task 1（需要确认构建产物路径）。Task 5 依赖 Task 3 和 4。Task 6 依赖 Task 5。
