# PR #27 网关修复 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all 7 confirmed gateway issues from PR #27 review: HTTP self-call anti-pattern, nginx config gaps, missing health checks, SSE disconnect detection, and outdated docs.

**Architecture:** The critical change replaces HTTP self-calls in agent's ToolExecutor with direct Go service calls via dependency injection. The agent package defines interfaces (`VersionCreator`, `TaskCreator`) to stay decoupled from the render package. main.go creates render services and injects them into both modules. Config/infra fixes (nginx, docker-compose) follow as separate tasks.

**Tech Stack:** Go (Gin, GORM), nginx, Docker Compose, React (TypeScript), TipTap

---

### Task 1: Refactor ToolExecutor — Replace HTTP Self-Call with Direct Service Injection

**Why:** `createVersion` and `exportPDF` make HTTP POST to `http://127.0.0.1:8080` — this fails inside Docker containers. Direct Go function calls are faster, simpler, and work everywhere.

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`
- Modify: `backend/internal/modules/agent/tool_executor_test.go`
- Modify: `backend/internal/modules/agent/routes.go`
- Modify: `backend/internal/modules/render/routes.go`
- Modify: `backend/cmd/server/main.go`

**Step 1: Write failing test for createVersion via interface**

In `backend/internal/modules/agent/tool_executor_test.go`, replace the HTTP-based `TestToolExecutor_CreateVersion` with a mock-interface test:

```go
// mockVersionSvc implements VersionCreator for testing.
type mockVersionSvc struct {
    createFn func(draftID uint, label string) (*models.Version, error)
}

func (m *mockVersionSvc) Create(draftID uint, label string) (*models.Version, error) {
    return m.createFn(draftID, label)
}

func TestToolExecutor_CreateVersion(t *testing.T) {
    executor := NewAgentToolExecutor(nil, &mockVersionSvc{
        createFn: func(draftID uint, label string) (*models.Version, error) {
            assert.Equal(t, uint(5), draftID)
            assert.Equal(t, "v1.0", label)
            return &models.Version{ID: 10, DraftID: 5, Label: "v1.0"}, nil
        },
    }, nil)

    result, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
        "draft_id": float64(5),
        "label":    "v1.0",
    })
    require.NoError(t, err)

    var data map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(result), &data))
    assert.Equal(t, float64(10), data["id"])
    assert.Equal(t, "v1.0", data["label"])
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/agent/... -run TestToolExecutor_CreateVersion -v`
Expected: FAIL — `NewAgentToolExecutor` still expects `(db, baseURL string)`, not interfaces.

**Step 3: Define interfaces and refactor AgentToolExecutor struct**

In `backend/internal/modules/agent/tool_executor.go`, replace the struct:

```go
// VersionCreator creates version snapshots (implemented by render.VersionService).
type VersionCreator interface {
    Create(draftID uint, label string) (*models.Version, error)
}

// TaskCreator triggers PDF export tasks (implemented by render.ExportService).
type TaskCreator interface {
    CreateTask(draftID uint, htmlContent string) (string, error)
}

// AgentToolExecutor implements ToolExecutor using database queries and direct service calls.
type AgentToolExecutor struct {
    db         *gorm.DB
    versionSvc VersionCreator
    exportSvc  TaskCreator
}

// NewAgentToolExecutor creates a new AgentToolExecutor.
func NewAgentToolExecutor(db *gorm.DB, versionSvc VersionCreator, exportSvc TaskCreator) *AgentToolExecutor {
    return &AgentToolExecutor{
        db:         db,
        versionSvc: versionSvc,
        exportSvc:  exportSvc,
    }
}
```

Remove: `httpClient *http.Client`, `baseURL string`, and the entire `httpPost` method.
Remove imports: `"bytes"`, `"io"`, `"net/http"`, `"time"` (no longer needed).

**Step 4: Rewrite createVersion and exportPDF**

```go
func (e *AgentToolExecutor) createVersion(ctx context.Context, params map[string]interface{}) (string, error) {
    draftID, err := getIntParam(params, "draft_id")
    if err != nil {
        return "", err
    }
    label, err := getStringParam(params, "label")
    if err != nil {
        return "", err
    }

    version, err := e.versionSvc.Create(draftID, label)
    if err != nil {
        return "", fmt.Errorf("create version: %w", err)
    }

    b, err := json.Marshal(version)
    if err != nil {
        return "", fmt.Errorf("marshal version: %w", err)
    }
    return string(b), nil
}

func (e *AgentToolExecutor) exportPDF(ctx context.Context, params map[string]interface{}) (string, error) {
    draftID, err := getIntParam(params, "draft_id")
    if err != nil {
        return "", err
    }
    htmlContent, err := getStringParam(params, "html_content")
    if err != nil {
        return "", err
    }

    taskID, err := e.exportSvc.CreateTask(draftID, htmlContent)
    if err != nil {
        return "", fmt.Errorf("create export task: %w", err)
    }

    result := map[string]interface{}{"task_id": taskID}
    b, err := json.Marshal(result)
    if err != nil {
        return "", fmt.Errorf("marshal result: %w", err)
    }
    return string(b), nil
}
```

**Step 5: Update all existing tests to use new constructor signature**

In `tool_executor_test.go`:
- All `NewAgentToolExecutor(nil, "")` → `NewAgentToolExecutor(nil, nil, nil)`
- All `NewAgentToolExecutor(db, "")` → `NewAgentToolExecutor(db, nil, nil)`
- Replace HTTP-based tests (`TestToolExecutor_CreateVersion`, `TestToolExecutor_ExportPDF`, `TestToolExecutor_ExportPDF_EmptyHTML`, `TestToolExecutor_HTTP_ServerError`, `TestToolExecutor_HTTP_NotFound`) with mock-interface versions.

Add mock types at the top of the test file:

```go
type mockVersionSvc struct {
    createFn func(draftID uint, label string) (*models.Version, error)
}

func (m *mockVersionSvc) Create(draftID uint, label string) (*models.Version, error) {
    return m.createFn(draftID, label)
}

type mockExportSvc struct {
    createTaskFn func(draftID uint, htmlContent string) (string, error)
}

func (m *mockExportSvc) CreateTask(draftID uint, htmlContent string) (string, error) {
    return m.createTaskFn(draftID, htmlContent)
}
```

Remove imports: `"net/http"`, `"net/http/httptest"`.

Full replacement tests:

```go
func TestToolExecutor_CreateVersion(t *testing.T) {
    executor := NewAgentToolExecutor(nil, &mockVersionSvc{
        createFn: func(draftID uint, label string) (*models.Version, error) {
            assert.Equal(t, uint(5), draftID)
            assert.Equal(t, "v1.0", label)
            return &models.Version{ID: 10, DraftID: 5, Label: "v1.0"}, nil
        },
    }, nil)

    result, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
        "draft_id": float64(5),
        "label":    "v1.0",
    })
    require.NoError(t, err)

    var data map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(result), &data))
    assert.Equal(t, float64(10), data["id"])
    assert.Equal(t, "v1.0", data["label"])
}

func TestToolExecutor_CreateVersion_ServiceError(t *testing.T) {
    executor := NewAgentToolExecutor(nil, &mockVersionSvc{
        createFn: func(draftID uint, label string) (*models.Version, error) {
            return nil, errors.New("draft not found")
        },
    }, nil)

    _, err := executor.Execute(context.Background(), "create_version", map[string]interface{}{
        "draft_id": float64(999),
        "label":    "test",
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "create version")
}

func TestToolExecutor_ExportPDF(t *testing.T) {
    executor := NewAgentToolExecutor(nil, nil, &mockExportSvc{
        createTaskFn: func(draftID uint, htmlContent string) (string, error) {
            assert.Equal(t, uint(3), draftID)
            assert.Equal(t, "<html>resume</html>", htmlContent)
            return "task_abc", nil
        },
    })

    result, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
        "draft_id":     float64(3),
        "html_content": "<html>resume</html>",
    })
    require.NoError(t, err)

    var data map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(result), &data))
    assert.Equal(t, "task_abc", data["task_id"])
}

func TestToolExecutor_ExportPDF_ServiceError(t *testing.T) {
    executor := NewAgentToolExecutor(nil, nil, &mockExportSvc{
        createTaskFn: func(draftID uint, htmlContent string) (string, error) {
            return "", errors.New("export failed")
        },
    })

    _, err := executor.Execute(context.Background(), "export_pdf", map[string]interface{}{
        "draft_id":     float64(1),
        "html_content": "<html></html>",
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "create export task")
}
```

**Step 6: Run all agent tests to verify they pass**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: ALL PASS

**Step 7: Update agent routes.go**

In `backend/internal/modules/agent/routes.go`, change the RegisterRoutes signature:

```go
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, versionSvc VersionCreator, exportSvc TaskCreator) {
    sessionSvc := NewSessionService(db)

    var provider ProviderAdapter
    if os.Getenv("USE_MOCK") == "true" {
        provider = &MockAdapter{}
    } else {
        provider = NewOpenAIAdapter(
            os.Getenv("AI_API_URL"),
            os.Getenv("AI_API_KEY"),
            envOrDefault("AI_MODEL", "default"),
        )
    }

    toolExecutor := NewAgentToolExecutor(db, versionSvc, exportSvc)
    maxIterations := 3
    if v := os.Getenv("AGENT_MAX_ITERATIONS"); v != "" {
        if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
            maxIterations = parsed
        }
    }
    chatSvc := NewChatService(db, provider, toolExecutor, maxIterations)
    h := NewHandler(sessionSvc, chatSvc)

    rg.POST("/ai/sessions", h.CreateSession)
    rg.GET("/ai/sessions", h.ListSessions)
    rg.GET("/ai/sessions/:session_id", h.GetSession)
    rg.DELETE("/ai/sessions/:session_id", h.DeleteSession)
    rg.POST("/ai/sessions/:session_id/chat", h.Chat)
    rg.GET("/ai/sessions/:session_id/history", h.GetHistory)
}
```

**Step 8: Update render routes.go**

In `backend/internal/modules/render/routes.go`, change RegisterRoutes to accept pre-created services:

```go
// RegisterRoutes registers all render module endpoints.
// Accepts pre-created services from main.go (lifecycle managed there).
func RegisterRoutes(rg *gin.RouterGroup, versionSvc *VersionService, exportSvc *ExportService) {
    h := NewHandler(versionSvc, exportSvc)

    // Version management
    rg.GET("/drafts/:draft_id/versions", h.ListVersions)
    rg.POST("/drafts/:draft_id/versions", h.CreateVersion)
    rg.POST("/drafts/:draft_id/rollback", h.Rollback)

    // PDF export
    rg.POST("/drafts/:draft_id/export", h.CreateExport)
    rg.GET("/tasks/:task_id", h.GetTask)
    rg.GET("/tasks/:task_id/file", h.DownloadFile)
}
```

Remove the import of `"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"` — no longer needed.

**Step 9: Update main.go wiring**

In `backend/cmd/server/main.go`, update `setupRouter`:

```go
func setupRouter(db *gorm.DB) (*gin.Engine, func()) {
    r := gin.Default()
    r.Use(middleware.CORS(), middleware.Logger())

    uploadDir := os.Getenv("UPLOAD_DIR")
    if uploadDir == "" {
        uploadDir = "./uploads"
    }
    os.MkdirAll(uploadDir, 0755)

    store := storage.NewLocalStorage(uploadDir)

    // Create render services upfront (shared with agent module)
    versionSvc := render.NewVersionService(db)
    chromeExporter := render.NewChromeExporter()
    exportSvc := render.NewExportService(chromeExporter, store)
    exportSvc.db = db

    v1 := r.Group("/api/v1")
    secret, err := jwtSecret()
    if err != nil {
        log.Fatalf("invalid auth config: %v", err)
    }
    ttl := jwtTTL()
    secure := cookieSecure()

    authed := v1.Group("")
    authed.Use(middleware.AuthRequired(secret))
    auth.RegisterRoutes(v1, authed, db, secret, ttl, secure)
    intake.RegisterRoutes(authed, db, uploadDir)
    parsing.RegisterRoutes(authed, db, store)
    agent.RegisterRoutes(authed, db, versionSvc, exportSvc)
    workbench.RegisterRoutes(authed, db)
    render.RegisterRoutes(authed, versionSvc, exportSvc)

    cleanup := func() {
        chromeExporter.Close()
        exportSvc.Close()
    }
    return r, cleanup
}
```

**Step 10: Run full backend test suite + compile check**

Run: `cd backend && go build ./cmd/server/... && go test ./...`
Expected: BUILD OK, ALL TESTS PASS

**Step 11: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go \
        backend/internal/modules/agent/tool_executor_test.go \
        backend/internal/modules/agent/routes.go \
        backend/internal/modules/render/routes.go \
        backend/cmd/server/main.go
git commit -m "refactor: replace HTTP self-call with direct service injection in ToolExecutor"
```

---

### Task 2: Harden gateway/nginx.conf

**Why:** Missing `client_max_body_size` causes 413 on file upload, no `server_tokens off` leaks version info, no error intercept returns nginx HTML instead of JSON on upstream failure.

**Files:**
- Modify: `gateway/nginx.conf`

**Step 1: Rewrite gateway/nginx.conf**

```nginx
server {
    listen 80;
    server_name _;

    client_max_body_size 20m;
    server_tokens off;

    # JSON error response for upstream failures
    location = /50x.json {
        default_type application/json;
        return 503 '{"code":59999,"data":null,"message":"服务暂时不可用，请稍后重试"}';
    }

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
        proxy_intercept_errors on;
        error_page 502 503 504 = /50x.json;
    }

    # SSE (AI 对话) — 最长前缀匹配优先于 /api/
    location /api/v1/ai/ {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_read_timeout 86400;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Step 2: Verify nginx config syntax (if nginx available locally)**

Run: `docker run --rm -v $(pwd)/gateway/nginx.conf:/etc/nginx/conf.d/default.conf:ro nginx:alpine nginx -t`
Expected: `syntax is ok / test is successful`

**Step 3: Commit**

```bash
git add gateway/nginx.conf
git commit -m "fix: harden gateway nginx — add body limit, hide version, intercept 50x"
```

---

### Task 3: Fix workbench nginx SSE proxy headers

**Why:** SSE location block missing `X-Forwarded-For` and `X-Forwarded-Proto`, causing log/protocol issues when gateway proxies through.

**Files:**
- Modify: `frontend/workbench/nginx.conf`

**Step 1: Add missing headers to SSE block**

Change the `/api/v1/ai/` location block in `frontend/workbench/nginx.conf`:

```nginx
    # SSE support for AI chat
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
```

**Step 2: Verify syntax**

Run: `docker run --rm -v $(pwd)/frontend/workbench/nginx.conf:/etc/nginx/conf.d/default.conf:ro nginx:alpine nginx -t`
Expected: `syntax is ok / test is successful`

**Step 3: Commit**

```bash
git add frontend/workbench/nginx.conf
git commit -m "fix: add X-Forwarded headers to workbench SSE proxy block"
```

---

### Task 4: Add health checks to docker-compose.yml

**Why:** Gateway starts proxying before dependencies are ready, causing 502 on startup. `service_started` condition is insufficient.

**Files:**
- Modify: `docker-compose.yml`

**Step 1: Add health checks and update depends_on**

```yaml
  backend:
    # ... existing build/environment ...
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8080/api/v1/auth/me || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    # ... existing ports/depends_on/restart ...

  gateway:
    # ... existing build ...
    depends_on:
      marketing:
        condition: service_healthy
      workbench:
        condition: service_healthy
      backend:
        condition: service_healthy
    # ... existing restart ...

  workbench:
    # ... existing build ...
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:80/ || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s
    # ... existing depends_on/restart ...

  marketing:
    # ... existing build ...
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:80/ || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s
    # ... existing restart ...
```

**Step 2: Validate compose file**

Run: `docker compose config --quiet`
Expected: No errors

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "fix: add health checks to all services, gateway waits for healthy deps"
```

---

### Task 5: Add SSE disconnect detection to ChatPanel

**Why:** If the SSE stream drops mid-response (network issue, server crash), the user sees no error — the incomplete reply appears as if complete. Need to track whether a `done` event was received before stream closure.

**Files:**
- Modify: `frontend/workbench/src/components/chat/ChatPanel.tsx`

**Step 1: Add done tracking in handleSend**

In the SSE while loop area (around line 130), add a `gotDone` tracker, a `case 'done':` handler, and post-loop check:

After `let inHTML = false` (line 130), add:

```typescript
      let gotDone = false
```

In the switch statement (around line 146), add a new case after `case 'error'`:

```typescript
              case 'done':
                gotDone = true
                break
```

After the `while (true)` loop ends (after line 200), before the closing `}` of the try block, add:

```typescript
      if (inHTML && currentHTML) {
        setHtmlPreview(currentHTML)
      }
      if (!gotDone) {
        setError('连接中断，AI 回复可能不完整')
      }
```

**Step 2: Verify TypeScript compiles**

Run: `cd frontend/workbench && bunx tsc --noEmit`
Expected: No errors

**Step 3: Run existing tests**

Run: `cd frontend/workbench && bunx vitest run`
Expected: ALL PASS (existing tests should still pass — this is additive behavior)

**Step 4: Commit**

```bash
git add frontend/workbench/src/components/chat/ChatPanel.tsx
git commit -m "fix: add SSE disconnect detection and partial HTML flush in ChatPanel"
```

---

### Task 6: Update CLAUDE.md and .env.example

**Why:** Three lines in CLAUDE.md describe outdated API conventions. .env.example is missing the gateway port variable.

**Files:**
- Modify: `CLAUDE.md` (lines 138, 140, 142)
- Modify: `.env.example`

**Step 1: Fix CLAUDE.md line 138**

Change:
```
- 响应格式 `{code: 0, data: {...}, message: "ok"}`，错误码 5 位 `SSCCC`（SS=模块 01-05）
```
To:
```
- 响应格式 `{code: 0, data: {...}, message: "ok"}`，错误码 4 位（模块分段：intake=1xxx, parsing=2xxx, agent=3xxx, workbench=4xxx, render=5xxx）
```

**Step 2: Fix CLAUDE.md line 140**

Change:
```
- AI 对话用 SSE 流式响应（Content-Type: text/event-stream，event types: text, html_start, html_chunk, html_end, done）
```
To:
```
- AI 对话用 SSE 流式响应（Content-Type: text/event-stream，event types: text, thinking, tool_call, tool_result, error, done）
```

**Step 3: Fix CLAUDE.md line 142**

Change:
```
- v1 无认证
```
To:
```
- v1 认证：JWT（HttpOnly cookie），详见 api-conventions
```

**Step 4: Add HOST_GATEWAY_PORT to .env.example**

After the `HOST_BACKEND_PORT=8080` line, add:

```env
# Gateway（统一入口）
HOST_GATEWAY_PORT=80
```

**Step 5: Commit**

```bash
git add CLAUDE.md .env.example
git commit -m "docs: update CLAUDE.md error codes, SSE events, auth info; add HOST_GATEWAY_PORT"
```

---

## Dependency Graph

```
Task 1 (ToolExecutor refactor) ── must be first, blocks backend compilation
Task 2 (gateway nginx) ── independent
Task 3 (workbench nginx) ── independent
Task 4 (docker-compose) ── independent
Task 5 (ChatPanel SSE) ── independent
Task 6 (docs) ── independent

Tasks 2-6 can run in any order after Task 1.
```

## Risk Assessment

| Task | Risk | Mitigation |
|------|------|------------|
| Task 1 | Interface mismatch, test breakage | Compile check + full test suite after each step |
| Task 2 | nginx syntax error | `nginx -t` via Docker |
| Task 3 | nginx syntax error | `nginx -t` via Docker |
| Task 4 | wget not in Alpine images | Backend uses Alpine + wget; frontend images need checking |
| Task 5 | TypeScript error | `tsc --noEmit` before commit |
| Task 6 | None | Text-only changes |
