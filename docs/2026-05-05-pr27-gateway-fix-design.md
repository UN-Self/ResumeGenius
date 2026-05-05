# PR #27 网关修复设计

日期: 2026-05-05

## 背景

PR #27 引入了统一 nginx 网关（gateway），将营销站、工作台、后端 API 统一路由。review 发现 11 个问题，需要修复后才能合并。

## 修复原则

- 外部只暴露 gateway 一个端口，内部服务走 Docker 网络
- 内部服务互信，减少内部鉴权
- Tool Executor 去掉 HTTP 自调用，改直接调 service 方法

## 改动清单

### 1. gateway/nginx.conf 重写

- 加 `client_max_body_size 20m;`（修复文件上传 413）
- 加 `server_tokens off;`（隐藏版本号）
- `/api/` 和 `/api/v1/ai/` 加 `proxy_intercept_errors on;` + `error_page 502 503 504`，返回 JSON 错误体
- 新增 `location = /50x.json`，返回 `{"code":59999,"data":null,"message":"服务暂时不可用，请稍后重试"}`

### 2. frontend/workbench/nginx.conf 补头

- `/api/v1/ai/` 块补 `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`
- `/api/v1/ai/` 块补 `proxy_set_header X-Forwarded-Proto $scheme;`

### 3. docker-compose.yml 健康检查 + 收端口

- backend 加 healthcheck：`wget -qO- http://localhost:8080/api/v1/auth/me`
- workbench 加 healthcheck：`wget -qO- http://localhost:80/`
- marketing 加 healthcheck：`wget -qO- http://localhost:80/`
- gateway 的 depends_on 全部改为 `condition: service_healthy`
- workbench/marketing ports 保持 `${HOST_XXX_PORT:-}:80` 格式（默认不暴露）

### 4. ChatPanel.tsx SSE 断连检测

在 `while` 循环结束后：
- 用一个布尔变量 `gotDone` 追踪是否收到 `done` 事件
- `case 'done'` 时设 `gotDone = true`
- 循环结束后若 `!gotDone`，调 `setError('连接中断，AI 回复可能不完整')`

### 5. Tool Executor 去掉 HTTP 自调用

**tool_executor.go：**
- 删除 `httpClient`、`baseURL` 字段和 `httpPost` 方法
- 新增 `versionSvc *render.VersionService` 和 `exportSvc *render.ExportService` 字段
- `createVersion` 改为直接调 `versionSvc.Create(draftID, label)`，返回结果的 JSON
- `exportPDF` 改为直接调 `exportSvc.CreateTask(draftID, htmlContent)`，返回 task ID 的 JSON
- 构造函数改为 `NewAgentToolExecutor(db, versionSvc, exportSvc)`

**routes.go：**
- 在 agent 模块 RegisterRoutes 中接收 render 模块的 versionSvc 和 exportSvc
- 传入 `NewAgentToolExecutor(db, versionSvc, exportSvc)`

**render/routes.go：**
- RegisterRoutes 返回 service 实例或通过回调暴露给 agent 模块
- 或者改为在 main.go 中统一初始化 service 再传入各模块（推荐，减少模块间耦合）

**main.go：**
- 调整启动顺序：先创建 render 的 versionSvc 和 exportSvc，再传给 agent 模块

### 6. CLAUDE.md 改三行

- 第 138 行：错误码 `5 位 SSCCC（SS=模块 01-05）` → `4 位（模块分段：intake=1xxx, parsing=2xxx, agent=3xxx, workbench=4xxx, render=5xxx）`
- 第 140 行：事件类型 `text, html_start, html_chunk, html_end, done` → `text, thinking, tool_call, tool_result, error, done`
- 第 142 行：`v1 无认证` → `v1 认证：JWT（HttpOnly cookie），详见 api-conventions`

### 7. .env.example 补全

```env
# Gateway（统一入口）
HOST_GATEWAY_PORT=80

# 前端 Workbench — 可选，直接访问容器调试用
HOST_WORKBENCH_PORT=3000

# 营销站 Marketing — 可选，直接访问容器调试用
HOST_MARKETING_PORT=4321
```

## 不改动的部分

- 营销站代码和 nginx 配置
- 后端 SSE handler 逻辑
- 前端路由配置（basename、vite base 已正确）
- gateway Dockerfile
