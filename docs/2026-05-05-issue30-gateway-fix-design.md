# Issue #30 — 网关配置修复与基础设施加固 设计文档

## 概述

PR #27 review 发现的 11 个问题中除营销站以外的所有修复，共 7 项改动。

## 1. gateway/nginx.conf 重写

- 加 `client_max_body_size 50m;`（默认 1MB 太小，文件上传会 413）
- 加 `server_tokens off;`（隐藏 nginx 版本号）
- `/api/` 和 `/api/v1/ai/` 加 `proxy_intercept_errors on;` + `error_page 502 503 504 /50x.json`
- 新增 `location = /50x.json` 返回 `{"code":59999,"data":null,"message":"服务暂时不可用，请稍后重试"}`

## 2. frontend/workbench/nginx.conf 补头

`/api/v1/ai/` 块补两个 header（nginx location 之间不继承 proxy_set_header）：
- `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`
- `proxy_set_header X-Forwarded-Proto $scheme;`

## 3. docker-compose.yml 健康检查

三个服务加 healthcheck：
- backend: `wget -qO- http://localhost:8080/api/v1/auth/me`
- workbench: `wget -qO- http://localhost:80/`
- marketing: `wget -qO- http://localhost:80/`

gateway 的 `depends_on` 全部改为 `condition: service_healthy`。

## 4. ChatPanel.tsx SSE 断连检测

- 在 while 循环前加 `let gotDone = false`
- switch 中加 `case 'done': gotDone = true; break`
- 循环结束后若 `!gotDone`，调用 `setError('连接中断，AI 回复可能不完整')`

## 5. Tool Executor 去掉 HTTP 自调用

- 删除 `httpClient`、`baseURL` 字段和整个 `httpPost` 方法
- 新增 `versionSvc` 和 `exportSvc` 字段（直接引用 render 模块的 service）
- `createVersion` 改为调用 `versionSvc.Create(draftID, label)`
- `exportPDF` 改为调用 `exportSvc.CreateTask(draftID, htmlContent)`
- 构造函数改为 `NewAgentToolExecutor(db, versionSvc, exportSvc)`
- routes.go 和 main.go 同步调整初始化链路

## 6. CLAUDE.md 改三行

| 行 | 旧 | 新 |
|----|-----|-----|
| 错误码 | 5 位 SSCCC（SS=模块 01-05） | 通用错误 5 位（SSCCC），模块错误 4 位（MCCC） |
| SSE 事件 | text, html_start, html_chunk, html_end, done | text, thinking, tool_call, tool_result, error, done |
| 认证 | v1 无认证 | v1 认证：JWT（HttpOnly cookie） |

同时第 158 行错误码段保持 5 位格式不变（与通用错误一致）。

## 7. .env.example 补全

加 Gateway section：
```
# Gateway（统一入口）
HOST_GATEWAY_PORT=80
```
