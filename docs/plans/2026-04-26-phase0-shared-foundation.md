# 共享基石 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 搭建前后端项目骨架、共享模型、统一响应、数据库连接，为 5 个业务模块提供开发基础。

**Architecture:** Go 后端 Gin 路由分组注册各模块，GORM AutoMigrate 管理 6 张表。React 前端 Vite 脚手架 + shadcn/ui + react-router。docker-compose 提供 PostgreSQL。

**Tech Stack:** Go 1.22+ / Gin / GORM / PostgreSQL 15 / Vite / React 18 / TypeScript / Tailwind CSS / shadcn/ui

**Depends on:** 无（这是第一个计划）

---

### Task 1: Backend — go.mod + main.go

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`

**Step 1: 初始化 Go 模块**

```bash
cd backend
go mod init github.com/UN-Self/ResumeGenius/backend
```

**Step 2: 安装核心依赖**

```bash
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/postgres
```

**Step 3: 写 main.go 最小入口**

```go
package main

import (
	"log"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	log.Fatal(r.Run(":8080"))
}
```

**Step 4: 验证**

```bash
go run cmd/server/main.go &
curl http://localhost:8080/health
# Expected: {"status":"ok"}
kill %1
```

**Step 5: Commit**

```bash
git add backend/
git commit -m "feat: backend scaffold with Gin entry point"
```

---

### Task 2: Backend — 共享 GORM 模型

**Files:**
- Create: `backend/internal/shared/models/models.go`

**Step 1: 写模型测试**

Create: `backend/internal/shared/models/models_test.go`

```go
package models

import (
	"testing"
	"time"
)

func TestProjectFields(t *testing.T) {
	p := Project{
		Title:  "测试项目",
		Status: "active",
	}
	if p.Title != "测试项目" {
		t.Errorf("expected title 测试项目, got %s", p.Title)
	}
	if p.Status != "active" {
		t.Errorf("expected status active, got %s", p.Status)
	}
}

func TestAssetType(t *testing.T) {
	a := Asset{Type: "resume_pdf", ProjectID: 1}
	if a.Type != "resume_pdf" {
		t.Errorf("expected type resume_pdf, got %s", a.Type)
	}
}

func TestDraftHTMLContent(t *testing.T) {
	d := Draft{HTMLContent: "<html></html>", ProjectID: 1}
	if d.HTMLContent != "<html></html>" {
		t.Errorf("expected html content")
	}
}

func TestVersionSnapshot(t *testing.T) {
	now := time.Now()
	v := Version{HTMLSnapshot: "<html></html>", DraftID: 1, CreatedAt: now}
	if v.HTMLSnapshot == "" {
		t.Error("expected non-empty snapshot")
	}
}

func TestAIMessageRole(t *testing.T) {
	m := AIMessage{Role: "user", Content: "hello"}
	if m.Role != "user" {
		t.Errorf("expected role user, got %s", m.Role)
	}
}

func TestJSONBValue(t *testing.T) {
	j := JSONB{"key": "value"}
	val, err := j.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/shared/models/...
# Expected: FAIL (models.go not yet created)
```

**Step 3: 写 models.go 实现**

Create: `backend/internal/shared/models/models.go`

```go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

type Project struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Title          string    `gorm:"size:200;not null" json:"title"`
	Status         string    `gorm:"size:20;not null;default:'active'" json:"status"`
	CurrentDraftID *uint     `json:"current_draft_id"`
	CurrentDraft   *Draft    `gorm:"foreignKey:CurrentDraftID" json:"current_draft,omitempty"`
	Assets         []Asset   `gorm:"foreignKey:ProjectID" json:"assets,omitempty"`
	Drafts         []Draft   `gorm:"foreignKey:ProjectID" json:"drafts,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Asset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index" json:"project_id"`
	Type      string    `gorm:"size:50;not null" json:"type"`
	URI       *string   `gorm:"type:text" json:"uri,omitempty"`
	Content   *string   `gorm:"type:text" json:"content,omitempty"`
	Label     *string   `gorm:"size:100" json:"label,omitempty"`
	Metadata  JSONB     `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Draft struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	ProjectID   uint        `gorm:"not null;index" json:"project_id"`
	HTMLContent string      `gorm:"type:text;not null" json:"html_content"`
	Project     Project     `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Versions    []Version   `gorm:"foreignKey:DraftID" json:"versions,omitempty"`
	AISessions  []AISession `gorm:"foreignKey:DraftID" json:"ai_sessions,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Version struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DraftID      uint      `gorm:"not null;index" json:"draft_id"`
	HTMLSnapshot string    `gorm:"type:text;not null" json:"html_snapshot"`
	Label        *string   `gorm:"size:200" json:"label,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AISession struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	DraftID   uint        `gorm:"not null;index" json:"draft_id"`
	Draft     Draft       `gorm:"foreignKey:DraftID" json:"draft,omitempty"`
	Messages  []AIMessage `gorm:"foreignKey:SessionID" json:"messages,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

type AIMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID uint      `gorm:"not null;index" json:"session_id"`
	Session   AISession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	Role      string    `gorm:"size:20;not null" json:"role"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
```

**Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/shared/models/... -v
# Expected: PASS (6 tests)
```

**Step 5: Commit**

```bash
git add backend/internal/shared/models/
git commit -m "feat: add shared GORM models for all 6 tables"
```

---

### Task 3: Backend — 数据库连接 + AutoMigrate

**Files:**
- Create: `backend/internal/shared/database/database.go`

**Step 1: 写数据库连接测试**

Create: `backend/internal/shared/database/database_test.go`

```go
package database

import "testing"

func TestConnectDSNFormat(t *testing.T) {
	dsn := buildDSN("localhost", "5432", "testuser", "testpass", "testdb")
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if dsn != expected {
		t.Errorf("got %s, want %s", dsn, expected)
	}
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/shared/database/...
# Expected: FAIL
```

**Step 3: 写 database.go**

```go
package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func buildDSN(host, port, user, password, dbname string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

func Connect() *gorm.DB {
	dsn := buildDSN(
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "postgres"),
		envOrDefault("DB_PASSWORD", "postgres"),
		envOrDefault("DB_NAME", "resume_genius"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	return db
}

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.Project{},
		&models.Asset{},
		&models.Draft{},
		&models.Version{},
		&models.AISession{},
		&models.AIMessage{},
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	log.Println("database migrated successfully")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

**Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/shared/database/... -v
# Expected: PASS
```

**Step 5: Commit**

```bash
git add backend/internal/shared/database/
git commit -m "feat: add database connection with AutoMigrate"
```

---

### Task 4: Backend — 统一响应 + 中间件

**Files:**
- Create: `backend/internal/shared/response/response.go`
- Create: `backend/internal/shared/middleware/middleware.go`

**Step 1: 写响应测试**

Create: `backend/internal/shared/response/response_test.go`

```go
package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, gin.H{"id": 1})

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(0) {
		t.Errorf("expected code 0, got %v", body["code"])
	}
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 40001, "参数错误")

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(40001) {
		t.Errorf("expected code 40001, got %v", body["code"])
	}
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/shared/response/...
# Expected: FAIL
```

**Step 3: 写 response.go**

```go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Data:    data,
		Message: "ok",
	})
}

func Error(c *gin.Context, code int, message string) {
	status := httpStatusFromCode(code)
	c.JSON(status, APIResponse{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}

func httpStatusFromCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}
	if code >= 50000 {
		return http.StatusInternalServerError
	}
	if code >= 40400 {
		return http.StatusNotFound
	}
	if code >= 40300 {
		return http.StatusForbidden
	}
	if code >= 40100 {
		return http.StatusUnauthorized
	}
	return http.StatusBadRequest
}
```

**Step 4: 写 middleware.go**

```go
package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("[%s] %s %d %v", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}
```

**Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/shared/... -v
# Expected: PASS (all tests)
```

**Step 6: Commit**

```bash
git add backend/internal/shared/response/ backend/internal/shared/middleware/
git commit -m "feat: add unified API response and middleware (CORS, Logger)"
```

---

### Task 5: Backend — 模块路由注册框架

**Files:**
- Create: `backend/internal/modules/intake/routes.go`
- Create: `backend/internal/modules/parsing/routes.go`
- Create: `backend/internal/modules/agent/routes.go`
- Create: `backend/internal/modules/workbench/routes.go`
- Create: `backend/internal/modules/render/routes.go`
- Modify: `backend/cmd/server/main.go`

**Step 1: 每个模块创建最小 routes.go**

每个模块统一格式（以 workbench 为例）：

```go
package workbench

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/drafts/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "workbench", "status": "stub"})
	})
}
```

其余 4 个模块同理，替换模块名。

**Step 2: 更新 main.go 注册所有模块**

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/database"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	workbench "github.com/UN-Self/ResumeGenius/backend/internal/modules/workbench"
	render "github.com/UN-Self/ResumeGenius/backend/internal/modules/render"
	intake "github.com/UN-Self/ResumeGenius/backend/internal/modules/intake"
	parsing "github.com/UN-Self/ResumeGenius/backend/internal/modules/parsing"
	agent "github.com/UN-Self/ResumeGenius/backend/internal/modules/agent"
)

func main() {
	db := database.Connect()
	database.Migrate(db)

	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

	api := r.Group("/api/v1")
	intake.RegisterRoutes(api.Group("/intake"), db)
	parsing.RegisterRoutes(api.Group("/parsing"), db)
	agent.RegisterRoutes(api.Group("/ai"), db)
	workbench.RegisterRoutes(api.Group("/workbench"), db)
	render.RegisterRoutes(api.Group("/render"), db)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Fatal(r.Run(":8080"))
}
```

注意：各模块 `RegisterRoutes` 签名统一为 `func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB)`。

**Step 3: 验证所有 stub 端点**

```bash
go run cmd/server/main.go &
curl http://localhost:8080/api/v1/workbench/drafts/1
# Expected: {"module":"workbench","status":"stub"}
curl http://localhost:8080/health
# Expected: {"status":"ok"}
kill %1
```

**Step 4: Commit**

```bash
git add backend/
git commit -m "feat: register all 5 module route stubs"
```

---

### Task 6: Frontend — Vite + React 脚手架

**Files:**
- Create: `frontend/workbench/package.json`
- Create: `frontend/workbench/vite.config.ts`
- Create: `frontend/workbench/tsconfig.json`
- Create: `frontend/workbench/index.html`
- Create: `frontend/workbench/src/main.tsx`
- Create: `frontend/workbench/src/App.tsx`

**Step 1: 初始化项目**

```bash
cd frontend/workbench
npm create vite@latest . -- --template react-ts
```

**Step 2: 安装依赖**

```bash
npm install react-router-dom
npm install -D tailwindcss @tailwindcss/vite
```

**Step 3: 配置 Tailwind**

在 `vite.config.ts` 中添加 tailwindcss 插件：

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
```

在 `src/index.css` 中：

```css
@import "tailwindcss";
```

**Step 4: 写 App.tsx 最小路由**

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<div>ResumeGenius Workbench</div>} />
        <Route path="/editor/:projectId" element={<div>Editor</div>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
```

**Step 5: 验证**

```bash
npm run dev
# 浏览器打开 http://localhost:3000，应显示 "ResumeGenius Workbench"
```

**Step 6: Commit**

```bash
git add frontend/workbench/
git commit -m "feat: frontend workbench scaffold with Vite + React + Tailwind"
```

---

### Task 7: Frontend — API Client + shadcn/ui 初始化

**Files:**
- Create: `frontend/workbench/src/lib/api-client.ts`
- Setup: `frontend/workbench/src/components/ui/` (shadcn/ui)

**Step 1: 写 API client 测试**

Create: `frontend/workbench/src/lib/api-client.test.ts`

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './api-client'

describe('apiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('calls GET with correct path', async () => {
    const mock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ code: 0, data: { id: 1 } }) })
    vi.stubGlobal('fetch', mock)

    const result = await apiClient.get('/drafts/1')
    expect(mock).toHaveBeenCalledWith('/api/v1/drafts/1', expect.objectContaining({ method: 'GET' }))
  })
})
```

**Step 2: 运行测试确认失败**

```bash
cd frontend/workbench && npx vitest run src/lib/api-client.test.ts
# Expected: FAIL
```

**Step 3: 写 api-client.ts**

```ts
const BASE = '/api/v1'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const json = await res.json()
  if (json.code !== 0) {
    throw new ApiError(json.code, json.message)
  }
  return json.data as T
}

export class ApiError extends Error {
  constructor(public code: number, message: string) {
    super(message)
  }
}

export const apiClient = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
```

**Step 4: 运行测试确认通过**

```bash
cd frontend/workbench && npx vitest run src/lib/api-client.test.ts
# Expected: PASS
```

**Step 5: 初始化 shadcn/ui**

```bash
cd frontend/workbench
npx shadcn@latest init
# 选择: New York style, Zinc color, CSS variables: yes
```

**Step 6: Commit**

```bash
git add frontend/workbench/
git commit -m "feat: add API client and initialize shadcn/ui"
```

---

### Task 8: Fixtures + Docker Compose

**Files:**
- Create: `fixtures/sample_draft.html`
- Create: `fixtures/sample_ai_response.json`
- Create: `docker-compose.yml`

**Step 1: 创建 fixtures**

从 `docs/02-data-models/mock-fixtures.md` 复制 `sample_draft.html` 和 `sample_ai_response.json` 内容到 `fixtures/`。

**Step 2: 写 docker-compose.yml**

```yaml
version: "3.8"
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: resume_genius
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

**Step 3: 验证数据库启动**

```bash
docker-compose up -d postgres
docker-compose exec postgres pg_isready
# Expected: accepting connections
```

**Step 4: 验证后端完整启动**

```bash
cd backend
DB_HOST=localhost go run cmd/server/main.go &
curl http://localhost:8080/health
# Expected: {"status":"ok"}
kill %1
```

**Step 5: Commit**

```bash
git add fixtures/ docker-compose.yml
git commit -m "feat: add fixtures and docker-compose for PostgreSQL"
```

---

## 验证清单

完成所有 Task 后，确认：

- [ ] `docker-compose up -d` 能启动 PostgreSQL
- [ ] `go run cmd/server/main.go` 能连接数据库并启动 Gin
- [ ] `curl localhost:8080/health` 返回 `{"status":"ok"}`
- [ ] 所有 5 个模块 stub 端点返回 200
- [ ] `go test ./...` 全部通过
- [ ] `npm run dev` 能启动前端
- [ ] 浏览器访问 `localhost:3000` 看到页面
- [ ] `fixtures/sample_draft.html` 存在且内容完整
