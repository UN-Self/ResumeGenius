# Issue #30 网关配置修复与基础设施加固 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复网关配置问题，加固基础设施（健康检查、错误页、SSE 断连检测、去掉 HTTP 自调用）。

**Architecture:** 7 项独立修复，按风险从低到高排列：配置文件 → 前端组件 → 后端重构。

**Tech Stack:** Nginx, Docker Compose, React/TypeScript (TipTap), Go/Gin

---

### Task 1: ~~gateway/nginx.conf 重写~~ → 已删除 gateway 服务

**Status: DONE — gateway 服务整体移除，路由合并到 marketing**

原 gateway 的功能（反向代理、错误页、SSE 支持）已合并到 `frontend/marketing/nginx.conf`。具体包括：

- client_max_body_size 50m、server_tokens off
- /app/ → proxy_pass workbench:80
- /api/ → proxy_pass backend:8080（含 502/503/504 错误拦截）
- /api/v1/ai/ → SSE 长连接支持（proxy_buffering off, proxy_read_timeout 86400）
- 50x.json 错误页
- CSP header 仅在营销站静态资源生效，/app/ 和 /api/ 不受限
- 静态资源缓存正则排除 /app/ 和 /api/ 路径（避免拦截 workbench 资源）

**Files:**
- ~~Modify: `gateway/nginx.conf`~~ → 已删除
- Modify: `frontend/marketing/nginx.conf`（合并所有路由规则）

---

### Task 2: workbench nginx.conf 补头

**Status: DONE**

**Files:**
- Modify: `frontend/workbench/nginx.conf`

- [x] **Step 1: /api/v1/ai/ 块补两个 proxy_set_header**

在 SSE location 中添加了 X-Forwarded-For 和 X-Forwarded-Proto 头。

- [x] **Step 2: root 路径修正**

将 root 从 `/usr/share/nginx/html` 改为 `/usr/share/nginx/html/app`，匹配 Dockerfile 的 COPY 目标路径，解决 workbench 容器返回 nginx 默认页的问题。

---

### Task 3: docker-compose.yml 健康检查

**Status: DONE**

**Files:**
- Modify: `docker-compose.yml`

变更内容：
- backend 加 healthcheck（curl -sS http://localhost:8080/api/v1/auth/me）
- workbench 加 healthcheck（curl -f http://localhost/）
- marketing 加 healthcheck（curl -f http://localhost/）+ depends_on workbench/backend
- **删除 gateway 服务**，marketing 接管统一入口职责
- marketing 端口保持 44321（`"${HOST_MARKETING_PORT:-44321}:80"`）

健康检查从 wget 改为 curl（Alpine 镜像无 wget）。

---

### Task 4: ChatPanel.tsx SSE 断连检测

**Status: DONE**

**Files:**
- Modify: `frontend/workbench/src/components/chat/ChatPanel.tsx`

- [x] 添加 gotDone 标志位
- [x] switch 中加 done case
- [x] while 循环结束后检测断连，未收到 done 则 setError

---

### Task 5: Tool Executor 去掉 HTTP 自调用

**Status: DONE**

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`
- Modify: `backend/internal/modules/agent/tool_executor_test.go`
- Modify: `backend/internal/modules/agent/service_test.go`

- [x] 移除 HTTP 客户端（httpClient、baseURL、httpPost）
- [x] 添加 VersionService 和 ExportService 接口依赖注入
- [x] createVersion/exportPDF 直接调 service 而非 HTTP
- [x] 测试验证不再发起 HTTP 请求

---

### Task 6: 更新 routes.go 和 main.go 初始化链路

**Status: DONE**

**Files:**
- Modify: `backend/internal/modules/agent/routes.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/modules/render/routes.go`

- [x] render/routes.go 导出 NewServices() 工厂函数
- [x] agent/routes.go 接收 versionSvc 和 exportSvc 参数
- [x] main.go 先创建 render services 再传给 agent 模块

---

### Task 7: CLAUDE.md 更新

**Status: DONE**

**Files:**
- Modify: `CLAUDE.md`

- [x] 错误码格式：通用错误 5 位 SSCCC，模块错误 4 位 MCCC
- [x] SSE 事件类型：text, thinking, tool_call, tool_result, error, done
- [x] 认证描述：JWT（HttpOnly cookie）

---

### Task 8: .env.example 补全

**Status: DONE**

**Files:**
- Modify: `.env.example`

- [x] 移除 HOST_GATEWAY_PORT（gateway 已删除）
- [x] HOST_MARKETING_PORT=44321 保持不变

---

### 额外变更（实施过程中发现并修复）

1. **backend/Dockerfile** — 添加 curl（健康检查需要）
2. **frontend/marketing/Dockerfile** — 添加 curl（健康检查需要）
3. **docker-compose.yml gateway 孤立容器** — 可通过 `docker compose down --remove-orphans` 清理

---

### 验证清单

- [x] `docker compose up -d` 全部容器正常启动
- [x] `http://localhost:44321/` 营销站正常
- [x] `http://localhost:44321/app/` 工作台正常加载（SPA + 静态资源）
- [x] `http://localhost:44321/app/` 登录功能正常（API 请求不被 CSP 阻止）
- [ ] OpenResty 反向代理场景未验证（用户后续自行测试）
