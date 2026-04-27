# Routing Contract Alignment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将后端路由入口与模块注册方式调整为资源路径模式，并将 CLAUDE.md 明确对齐各模块 contract.md 的路径约定。

**Architecture:** 入口层仅保留 `/api/v1` 基础分组，模块自行在各自 `routes.go` 中注册资源子路径。通过新增 `setupRouter` 使路由可测试化，先写契约路径失败测试，再做最小改动让测试通过，最后同步文档口径。

**Tech Stack:** Go 1.24, Gin, GORM, Go testing, ripgrep

---

### Task 1: 建立路由契约失败测试骨架

**Files:**
- Create: `backend/cmd/server/main_test.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/cmd/server/main_test.go`

**Step 1: Write the failing test**

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestResourceRoutes_AreMountedOnApiV1(t *testing.T) {
    r := setupRouter(nil)

    cases := []struct {
        method string
        path   string
    }{
        {http.MethodGet, "/api/v1/projects"},
        {http.MethodPost, "/api/v1/parsing/parse"},
        {http.MethodPost, "/api/v1/ai/sessions"},
        {http.MethodGet, "/api/v1/drafts/1"},
        {http.MethodPost, "/api/v1/drafts/1/export"},
        {http.MethodGet, "/api/v1/tasks/task_1"},
    }

    for _, tc := range cases {
        req := httptest.NewRequest(tc.method, tc.path, nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code == http.StatusNotFound {
            t.Fatalf("route not found: %s %s", tc.method, tc.path)
        }
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: FAIL（通常会先报 `setupRouter` 未定义，或路由 404）

**Step 3: Write minimal implementation**

```go
func setupRouter(db *gorm.DB) *gin.Engine {
    r := gin.Default()
    r.Use(middleware.CORS(), middleware.Logger())

    v1 := r.Group("/api/v1")
    intake.RegisterRoutes(v1, db)
    parsing.RegisterRoutes(v1, db)
    agent.RegisterRoutes(v1, db)
    workbench.RegisterRoutes(v1, db)
    render.RegisterRoutes(v1, db)

    return r
}
```

并在 `main()` 中复用：

```go
r := setupRouter(db)
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: PASS（如果仍失败，进入 Task 2/3 的模块路径修正）

**Step 5: Commit**

```bash
git add backend/cmd/server/main.go backend/cmd/server/main_test.go
git commit -m "test: add router contract harness for resource paths"
```

### Task 2: 修正 B/C 模块到资源子路径

**Files:**
- Modify: `backend/internal/modules/parsing/routes.go`
- Modify: `backend/internal/modules/agent/routes.go`
- Test: `backend/cmd/server/main_test.go`

**Step 1: Write the failing test**

在 `main_test.go` 中细化断言（可增补 table case 名称），确保以下路径可达：

```go
{http.MethodPost, "/api/v1/parsing/parse"},
{http.MethodPost, "/api/v1/ai/sessions"},
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: FAIL（旧实现通常会出现 404）

**Step 3: Write minimal implementation**

`parsing/routes.go`:

```go
rg.POST("/parsing/parse", func(c *gin.Context) {
    c.JSON(200, gin.H{"module": "parsing", "status": "stub"})
})
```

`agent/routes.go`:

```go
rg.POST("/ai/sessions", func(c *gin.Context) {
    c.JSON(200, gin.H{"module": "agent", "status": "stub"})
})
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: PASS（B/C 路径相关 case）

**Step 5: Commit**

```bash
git add backend/internal/modules/parsing/routes.go backend/internal/modules/agent/routes.go backend/cmd/server/main_test.go
git commit -m "fix: align parsing and ai route prefixes to contracts"
```

### Task 3: 修正 D/E 模块并支持 E 多资源路径

**Files:**
- Modify: `backend/internal/modules/workbench/routes.go`
- Modify: `backend/internal/modules/render/routes.go`
- Test: `backend/cmd/server/main_test.go`

**Step 1: Write the failing test**

确保以下契约路径可达：

```go
{http.MethodGet, "/api/v1/drafts/1"},
{http.MethodPut, "/api/v1/drafts/1"},
{http.MethodPost, "/api/v1/drafts/1/export"},
{http.MethodGet, "/api/v1/tasks/task_1"},
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: FAIL（至少 E 的 tasks 或 drafts/export 404）

**Step 3: Write minimal implementation**

`workbench/routes.go`:

```go
rg.GET("/drafts/:draft_id", func(c *gin.Context) {
    c.JSON(200, gin.H{"module": "workbench", "status": "stub"})
})

rg.PUT("/drafts/:draft_id", func(c *gin.Context) {
    c.JSON(200, gin.H{"module": "workbench", "status": "stub"})
})
```

`render/routes.go`:

```go
rg.POST("/drafts/:draft_id/export", func(c *gin.Context) {
    c.JSON(200, gin.H{"module": "render", "status": "stub"})
})

rg.GET("/tasks/:task_id", func(c *gin.Context) {
    c.JSON(200, gin.H{"module": "render", "status": "stub"})
})
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/modules/workbench/routes.go backend/internal/modules/render/routes.go backend/cmd/server/main_test.go
git commit -m "fix: align drafts and tasks resource routes"
```

### Task 4: 更新 CLAUDE.md 为 contract 资源路径口径

**Files:**
- Modify: `CLAUDE.md`
- Reference: `docs/01-product/api-conventions.md`
- Reference: `docs/modules/intake/contract.md`
- Reference: `docs/modules/parsing/contract.md`
- Reference: `docs/modules/agent/contract.md`
- Reference: `docs/modules/workbench/contract.md`
- Reference: `docs/modules/render/contract.md`

**Step 1: Write the failing test**

使用文本校验作为文档契约检查：

```bash
rg -n "/api/v1/intake|/api/v1/workbench|/api/v1/render" CLAUDE.md
```

**Step 2: Run test to verify it fails**

Run: `rg -n "/api/v1/intake|/api/v1/workbench|/api/v1/render" CLAUDE.md`
Expected: 有匹配结果（表示仍是旧口径）

**Step 3: Write minimal implementation**

将路由映射段改为资源路径示意（例如 `/projects`、`/assets`、`/parsing/*`、`/ai/*`、`/drafts/*`、`/tasks/*`），并新增一句：

```md
路由与端点定义以 docs/modules/*/contract.md 为唯一契约来源。
```

**Step 4: Run test to verify it passes**

Run:

```bash
rg -n "/api/v1/intake|/api/v1/workbench|/api/v1/render" CLAUDE.md
```

Expected: 无匹配

**Step 5: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: align route mapping with module contracts"
```

### Task 5: 全量验证与收尾

**Files:**
- Verify: `backend/cmd/server/main.go`
- Verify: `backend/cmd/server/main_test.go`
- Verify: `backend/internal/modules/*/routes.go`
- Verify: `CLAUDE.md`

**Step 1: Write the failing test**

补充旧前缀不可达断言：

```go
legacy := []struct {
    method string
    path   string
}{
    {http.MethodGet, "/api/v1/intake/projects"},
    {http.MethodGet, "/api/v1/workbench/drafts/1"},
    {http.MethodPost, "/api/v1/render/export"},
}

for _, tc := range legacy {
    req := httptest.NewRequest(tc.method, tc.path, nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusNotFound {
        t.Fatalf("legacy path should be 404: %s %s, got %d", tc.method, tc.path, w.Code)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./cmd/server -run TestResourceRoutes_AreMountedOnApiV1 -v`
Expected: 在彻底移除旧分组前会失败

**Step 3: Write minimal implementation**

确保 `main.go` 不再使用 `v1.Group("/intake")`、`v1.Group("/workbench")`、`v1.Group("/render")`。

**Step 4: Run test to verify it passes**

Run:

```bash
cd backend && go test ./cmd/server -v
cd backend && go test ./... 
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/server/main.go backend/cmd/server/main_test.go backend/internal/modules/intake/routes.go backend/internal/modules/parsing/routes.go backend/internal/modules/agent/routes.go backend/internal/modules/workbench/routes.go backend/internal/modules/render/routes.go CLAUDE.md
git commit -m "feat: align API routing with resource-based contracts"
```
