# 去掉模块字母前缀 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将所有模块的字母前缀（a_intake, b_parsing 等）统一去掉，目录名、Go 包名、文档引用、错误码体系全部对齐到新命名。

**Architecture:** 纯机械替换，无逻辑变更。先重命名 Go 目录和更新代码，确保编译通过后，再更新文档内容和重命名文档目录/文件。最后做全局搜索验证无残留。

**Tech Stack:** Go, Git, ripgrep

---

### Task 1: 重命名 Go 模块目录

**Files:**
- Rename: `backend/internal/modules/a_intake/` → `backend/internal/modules/intake/`
- Rename: `backend/internal/modules/b_parsing/` → `backend/internal/modules/parsing/`
- Rename: `backend/internal/modules/c_agent/` → `backend/internal/modules/agent/`
- Rename: `backend/internal/modules/d_workbench/` → `backend/internal/modules/workbench/`
- Rename: `backend/internal/modules/e_render/` → `backend/internal/modules/render/`

**Step 1: 用 git mv 重命名 5 个目录**

```bash
cd backend/internal/modules
git mv a_intake intake
git mv b_parsing parsing
git mv c_agent agent
git mv d_workbench workbench
git mv e_render render
```

**Step 2: 确认目录已重命名**

```bash
ls backend/internal/modules/
```

Expected: 输出 `agent/ intake/ parsing/ render/ workbench/`

---

### Task 2: 更新 Go 包声明和字符串字面量

**Files:**
- Modify: `backend/internal/modules/intake/routes.go`
- Modify: `backend/internal/modules/parsing/routes.go`
- Modify: `backend/internal/modules/agent/routes.go`
- Modify: `backend/internal/modules/workbench/routes.go`
- Modify: `backend/internal/modules/render/routes.go`

**Step 1: 更新 intake/routes.go**

```go
package intake

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
    rg.GET("/projects", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "intake", "status": "stub"})
    })
}
```

**Step 2: 更新 parsing/routes.go**

```go
package parsing

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
    rg.POST("/parsing/parse", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "parsing", "status": "stub"})
    })
}
```

**Step 3: 更新 agent/routes.go**

```go
package agent

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
    rg.POST("/ai/sessions", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "agent", "status": "stub"})
    })
}
```

**Step 4: 更新 workbench/routes.go**

```go
package workbench

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
    rg.GET("/drafts/:draft_id", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "workbench", "status": "stub"})
    })

    rg.PUT("/drafts/:draft_id", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "workbench", "status": "stub"})
    })
}
```

**Step 5: 更新 render/routes.go**

```go
package render

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
    rg.POST("/drafts/:draft_id/export", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "render", "status": "stub"})
    })

    rg.GET("/tasks/:task_id", func(c *gin.Context) {
        c.JSON(200, gin.H{"module": "render", "status": "stub"})
    })
}
```

---

### Task 3: 更新 main.go 的 import 和函数调用

**Files:**
- Modify: `backend/cmd/server/main.go`

**Step 1: 更新 import 路径和 RegisterRoutes 调用**

将第 9-13 行的 import：
```go
"github.com/handy/resume-genius/internal/modules/a_intake"
"github.com/handy/resume-genius/internal/modules/b_parsing"
"github.com/handy/resume-genius/internal/modules/c_agent"
"github.com/handy/resume-genius/internal/modules/d_workbench"
"github.com/handy/resume-genius/internal/modules/e_render"
```

改为：
```go
"github.com/handy/resume-genius/internal/modules/agent"
"github.com/handy/resume-genius/internal/modules/intake"
"github.com/handy/resume-genius/internal/modules/parsing"
"github.com/handy/resume-genius/internal/modules/render"
"github.com/handy/resume-genius/internal/modules/workbench"
```

将第 25-29 行的调用：
```go
a_intake.RegisterRoutes(v1, db)
b_parsing.RegisterRoutes(v1, db)
c_agent.RegisterRoutes(v1, db)
d_workbench.RegisterRoutes(v1, db)
e_render.RegisterRoutes(v1, db)
```

改为：
```go
intake.RegisterRoutes(v1, db)
parsing.RegisterRoutes(v1, db)
agent.RegisterRoutes(v1, db)
workbench.RegisterRoutes(v1, db)
render.RegisterRoutes(v1, db)
```

**Step 2: 编译验证**

```bash
cd backend && go build ./cmd/server/...
```

Expected: 编译成功，无错误

**Step 3: 运行测试**

```bash
cd backend && go test ./...
```

Expected: 全部 PASS

---

### Task 4: 重命名文档目录

**Files:**
- Rename: `docs/modules/a-intake/` → `docs/modules/intake/`
- Rename: `docs/modules/b-parsing/` → `docs/modules/parsing/`
- Rename: `docs/modules/c-agent/` → `docs/modules/agent/`
- Rename: `docs/modules/d-workbench/` → `docs/modules/workbench/`
- Rename: `docs/modules/e-render/` → `docs/modules/render/`

**Step 1: 用 git mv 重命名**

```bash
cd docs/modules
git mv a-intake intake
git mv b-parsing parsing
git mv c-agent agent
git mv d-workbench workbench
git mv e-render render
```

**Step 2: 确认目录已重命名**

```bash
ls docs/modules/
```

Expected: 输出 `agent/ intake/ parsing/ render/ workbench/`

---

### Task 5: 重命名历史计划文件

**Files:**
- Rename: `docs/plans/2026-04-26-module-a-intake.md` → `docs/plans/2026-04-26-module-intake.md`
- Rename: `docs/plans/2026-04-26-module-b-parsing.md` → `docs/plans/2026-04-26-module-parsing.md`
- Rename: `docs/plans/2026-04-26-module-c-agent.md` → `docs/plans/2026-04-26-module-agent.md`
- Rename: `docs/plans/2026-04-26-module-d-workbench.md` → `docs/plans/2026-04-26-module-workbench.md`
- Rename: `docs/plans/2026-04-26-module-e-render.md` → `docs/plans/2026-04-26-module-render.md`

**Step 1: 用 git mv 重命名**

```bash
cd docs/plans
git mv 2026-04-26-module-a-intake.md 2026-04-26-module-intake.md
git mv 2026-04-26-module-b-parsing.md 2026-04-26-module-parsing.md
git mv 2026-04-26-module-c-agent.md 2026-04-26-module-agent.md
git mv 2026-04-26-module-d-workbench.md 2026-04-26-module-workbench.md
git mv 2026-04-26-module-e-render.md 2026-04-26-module-render.md
```

---

### Task 6: 更新 CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: 更新架构图中的模块标签**

将第 72 行：
```
[A 项目管理] → 文件/资料 → [B 解析初稿] → HTML 初稿
```
改为：
```
[项目管理] → 文件/资料 → [解析初稿] → HTML 初稿
```

将第 76 行：
```
                      [C AI 对话]          [D TipTap 编辑]
```
改为：
```
                      [AI 对话]          [TipTap 编辑]
```

将第 81 行：
```
                      [E 版本管理 + PDF 导出]
```
改为：
```
                      [版本管理 + PDF 导出]
```

**Step 2: 更新路由映射表**

将第 92-98 行：
```
/api/v1/projects       → a_intake    （项目管理）
/api/v1/assets/*       → a_intake    （文件上传、Git 接入、补充文本）
/api/v1/parsing/*      → b_parsing   （解析、AI 初稿生成）
/api/v1/ai/*           → c_agent     （SSE 流式 AI 对话）
/api/v1/drafts/*       → d_workbench （TipTap 编辑、草稿 CRUD）
/api/v1/drafts/*       → e_render    （版本快照）
/api/v1/tasks/*        → e_render    （PDF 导出任务）
```
改为：
```
/api/v1/projects       → intake      （项目管理）
/api/v1/assets/*       → intake      （文件上传、Git 接入、补充文本）
/api/v1/parsing/*      → parsing     （解析、AI 初稿生成）
/api/v1/ai/*           → agent       （SSE 流式 AI 对话）
/api/v1/drafts/*       → workbench   （TipTap 编辑、草稿 CRUD）
/api/v1/drafts/*       → render      （版本快照）
/api/v1/tasks/*        → render      （PDF 导出任务）
```

**Step 3: 更新文档体系中的模块目录引用**

将第 149 行：
```
- **模块契约** `docs/modules/{a-intake,...,e-render}/`：各模块 contract.md + work-breakdown.md
```
改为：
```
- **模块契约** `docs/modules/{intake,...,render}/`：各模块 contract.md + work-breakdown.md
```

**Step 4: 更新错误码段**

将第 157 行：
```
- 错误码段：A=01xxx, B=02xxx, C=03xxx, D=04xxx, E=05xxx
```
改为：
```
- 错误码段：intake=01xxx, parsing=02xxx, agent=03xxx, workbench=04xxx, render=05xxx
```

---

### Task 7: 更新 docs/README.md

**Files:**
- Modify: `docs/README.md`

**Step 1: 更新目录结构中的模块目录**

将第 29-33 行：
```
    a-intake/                      # A 资料接入
    b-parsing/                     # B 解析初稿
    c-agent/                       # C AI 对话
    d-workbench/                   # D 可视化编辑
    e-render/                      # E 版本导出
```
改为：
```
    intake/                        # 资料接入
    parsing/                       # 解析初稿
    agent/                         # AI 对话
    workbench/                     # 可视化编辑
    render/                        # 版本导出
```

**Step 2: 更新架构总览图**

将第 41 行：
```
[A 资料接入] → 文件/资料 → [B 解析初稿] → HTML 初稿
```
改为：
```
[资料接入] → 文件/资料 → [解析初稿] → HTML 初稿
```

将第 45 行：
```
                      [C AI 对话]          [D 可视化编辑]
```
改为：
```
                      [AI 对话]          [可视化编辑]
```

将第 50 行：
```
                    [E 版本管理 + PDF 导出]
```
改为：
```
                    [版本管理 + PDF 导出]
```

**Step 3: 更新模块表格**

将第 58-62 行：
```
| A 资料接入 | 项目 CRUD、文件上传、Git 接入、补充文本 | [contract.md](./modules/a-intake/contract.md) |
| B 解析初稿 | 解析文件提取文本 → AI 生成简历 HTML | [contract.md](./modules/b-parsing/contract.md) |
| C AI 对话 | 多轮 SSE 对话，AI 返回修改后的 HTML | [contract.md](./modules/c-agent/contract.md) |
| D 可视化编辑 | TipTap 所见即所得编辑，A4 画布 | [contract.md](./modules/d-workbench/contract.md) |
| E 版本导出 | HTML 快照版本管理 + chromedp PDF 导出 | [contract.md](./modules/e-render/contract.md) |
```
改为：
```
| 资料接入 | 项目 CRUD、文件上传、Git 接入、补充文本 | [contract.md](./modules/intake/contract.md) |
| 解析初稿 | 解析文件提取文本 → AI 生成简历 HTML | [contract.md](./modules/parsing/contract.md) |
| AI 对话 | 多轮 SSE 对话，AI 返回修改后的 HTML | [contract.md](./modules/agent/contract.md) |
| 可视化编辑 | TipTap 所见即所得编辑，A4 画布 | [contract.md](./modules/workbench/contract.md) |
| 版本导出 | HTML 快照版本管理 + chromedp PDF 导出 | [contract.md](./modules/render/contract.md) |
```

---

### Task 8: 更新 api-conventions.md

**Files:**
- Modify: `docs/01-product/api-conventions.md`

**Step 1: 更新模块资源表格**

将第 38-42 行：
```
| A 资料接入 | `/api/v1/projects/`、`/api/v1/assets/` | `POST /api/v1/assets/upload` |
| B 解析初稿 | `/api/v1/parsing/` | `POST /api/v1/parsing/parse` |
| C AI 对话 | `/api/v1/ai/` | `POST /api/v1/ai/sessions` |
| D 可视化编辑 | `/api/v1/drafts/` | `PUT /api/v1/drafts/{draft_id}` |
| E 版本导出 | `/api/v1/drafts/{draft_id}/`、`/api/v1/tasks/` | `POST /api/v1/drafts/{draft_id}/export` |
```
改为：
```
| 资料接入 | `/api/v1/projects/`、`/api/v1/assets/` | `POST /api/v1/assets/upload` |
| 解析初稿 | `/api/v1/parsing/` | `POST /api/v1/parsing/parse` |
| AI 对话 | `/api/v1/ai/` | `POST /api/v1/ai/sessions` |
| 可视化编辑 | `/api/v1/drafts/` | `PUT /api/v1/drafts/{draft_id}` |
| 版本导出 | `/api/v1/drafts/{draft_id}/`、`/api/v1/tasks/` | `POST /api/v1/drafts/{draft_id}/export` |
```

**Step 2: 更新错误码体系中的模块标签**

将第 95-99 行：
```
- 模块 A（资料接入）：1001–1999
- 模块 B（解析初稿）：2001–2999
- 模块 C（AI 对话）：3001–3999
- 模块 D（可视化编辑）：4001–4999
- 模块 E（版本导出）：5001–5999
```
改为：
```
- 模块 intake（资料接入）：1001–1999
- 模块 parsing（解析初稿）：2001–2999
- 模块 agent（AI 对话）：3001–3999
- 模块 workbench（可视化编辑）：4001–4999
- 模块 render（版本导出）：5001–5999
```

**Step 3: 更新模块错误码表格**

将第 120-124 行：
```
| A 资料接入 | 1001–1999 | 1001 = 文件格式不支持 |
| B 解析初稿 | 2001–2999 | 2001 = PDF 解析失败 |
| C AI 对话 | 3001–3999 | 3001 = 模型调用超时 |
| D 可视化编辑 | 4001–4999 | 4001 = 草稿不存在 |
| E 版本导出 | 5001–5999 | 5001 = PDF 导出失败 |
```
改为：
```
| 资料接入 | 1001–1999 | 1001 = 文件格式不支持 |
| 解析初稿 | 2001–2999 | 2001 = PDF 解析失败 |
| AI 对话 | 3001–3999 | 3001 = 模型调用超时 |
| 可视化编辑 | 4001–4999 | 4001 = 草稿不存在 |
| 版本导出 | 5001–5999 | 5001 = PDF 导出失败 |
```

---

### Task 9: 更新 dev-work-breakdown.md

**Files:**
- Modify: `docs/01-product/dev-work-breakdown.md`

**Step 1: 更新项目结构中的模块目录**

将第 24-28 行：
```
        a_intake/
        b_parsing/
        c_agent/
        d_workbench/
        e_render/
```
改为：
```
        intake/
        parsing/
        agent/
        workbench/
        render/
```

**Step 2: 更新模块交付物表格**

将第 87-91 行：
```
| A 资料接入 | — | 3 页 | 10 个 | projects + assets 表操作 |
| B 解析初稿 | — | 2 页 | 2 个 | 文本提取 → AI 生成 HTML |
| C AI 对话 | — | 集成在工作台 | 3 个 | SSE 流式对话 |
| D 可视化编辑 | — | 工作台主体 | 2 个 | TipTap 集成 + 自动保存 |
| E 版本导出 | — | 弹窗/抽屉 | 5 个 | HTML 快照 + chromedp PDF |
```
改为：
```
| 资料接入 | — | 3 页 | 10 个 | projects + assets 表操作 |
| 解析初稿 | — | 2 页 | 2 个 | 文本提取 → AI 生成 HTML |
| AI 对话 | — | 集成在工作台 | 3 个 | SSE 流式对话 |
| 可视化编辑 | — | 工作台主体 | 2 个 | TipTap 集成 + 自动保存 |
| 版本导出 | — | 弹窗/抽屉 | 5 个 | HTML 快照 + chromedp PDF |
```

**Step 3: 更新错误码协作规则**

将第 125 行：
```
- **错误码不冲突**：A=1001–1999, B=2001–2999, C=3001–3999, D=4001–4999, E=5001–5999
```
改为：
```
- **错误码不冲突**：intake=1001–1999, parsing=2001–2999, agent=3001–3999, workbench=4001–4999, render=5001–5999
```

**Step 4: 更新模块工作明细链接表**

将第 132-136 行：
```
| A 资料接入 | [modules/a-intake/work-breakdown.md](../modules/a-intake/work-breakdown.md) |
| B 解析初稿 | [modules/b-parsing/work-breakdown.md](../modules/b-parsing/work-breakdown.md) |
| C AI 对话 | [modules/c-agent/work-breakdown.md](../modules/c-agent/work-breakdown.md) |
| D 可视化编辑 | [modules/d-workbench/work-breakdown.md](../modules/d-workbench/work-breakdown.md) |
| E 版本导出 | [modules/e-render/work-breakdown.md](../modules/e-render/work-breakdown.md) |
```
改为：
```
| 资料接入 | [modules/intake/work-breakdown.md](../modules/intake/work-breakdown.md) |
| 解析初稿 | [modules/parsing/work-breakdown.md](../modules/parsing/work-breakdown.md) |
| AI 对话 | [modules/agent/work-breakdown.md](../modules/agent/work-breakdown.md) |
| 可视化编辑 | [modules/workbench/work-breakdown.md](../modules/workbench/work-breakdown.md) |
| 版本导出 | [modules/render/work-breakdown.md](../modules/render/work-breakdown.md) |
```

---

### Task 10: 更新 architecture-v2-design.md

**Files:**
- Modify: `docs/superpowers/specs/2026-04-23-architecture-v2-design.md`

对所有出现的模式执行替换（用 Edit 工具 replace_all）：

- `模块 A` → `模块 intake`（标题上下文如 "模块 A：" 或 "模块 A（"）
- `模块 B` → `模块 parsing`
- `模块 C` → `模块 agent`
- `模块 D` → `模块 workbench`
- `模块 E` → `模块 render`
- `a-intake` → `intake`
- `6.2 模块 B` → `6.2 模块 parsing`
- `6.3 模块 C` → `6.3 模块 agent`
- `6.4 模块 D` → `6.4 模块 workbench`
- `6.5 模块 E` → `6.5 模块 render`

注意：保留 ASCII 流程图中的 `(A)`、`(B)`、`(C)`、`(D)`、`(E)` 等简写标注不变，因为它们在图表中用于空间紧凑表示。

---

### Task 11: 更新 routing-alignment-implementation.md

**Files:**
- Modify: `docs/plans/2026-04-27-routing-alignment-implementation.md`

对所有出现的模式执行替换：

- `a_intake` → `intake`
- `b_parsing` → `parsing`
- `c_agent` → `agent`
- `d_workbench` → `workbench`
- `e_render` → `render`
- `a-intake` → `intake`
- `b-parsing` → `parsing`
- `c-agent` → `agent`
- `d-workbench` → `workbench`
- `e-render` → `render`

---

### Task 12: 更新 phase0-shared-foundation.md

**Files:**
- Modify: `docs/plans/2026-04-26-phase0-shared-foundation.md`

对所有出现的模式执行替换：

- `a_intake` → `intake`
- `b_parsing` → `parsing`
- `c_agent` → `agent`
- `d_workbench` → `workbench`
- `e_render` → `render`

---

### Task 13: 更新 5 个模块计划文件

**Files:**
- Modify: `docs/plans/2026-04-26-module-intake.md`
- Modify: `docs/plans/2026-04-26-module-parsing.md`
- Modify: `docs/plans/2026-04-26-module-agent.md`
- Modify: `docs/plans/2026-04-26-module-workbench.md`
- Modify: `docs/plans/2026-04-26-module-render.md`

每个文件执行对应替换：

**module-intake.md:** `a_intake` → `intake`, `a-intake` → `intake`, `模块 A` → `模块 intake`
**module-parsing.md:** `b_parsing` → `parsing`, `b-parsing` → `parsing`, `模块 B` → `模块 parsing`
**module-agent.md:** `c_agent` → `agent`, `c-agent` → `agent`, `模块 C` → `模块 agent`
**module-workbench.md:** `d_workbench` → `workbench`, `d-workbench` → `workbench`, `模块 D` → `模块 workbench`
**module-render.md:** `e_render` → `render`, `e-render` → `render`, `模块 E` → `模块 render`

---

### Task 14: 全局验证

**Step 1: 搜索旧前缀残留**

```bash
rg -n "a_intake|b_parsing|c_agent|d_workbench|e_render" --type-not binary
```

Expected: 无匹配（仅可能在 git 历史中出现，不在此范围）

```bash
rg -n "a-intake|b-parsing|c-agent|d-workbench|e-render" --type-not binary
```

Expected: 无匹配

**Step 2: Go 编译和测试**

```bash
cd backend && go build ./...
cd backend && go test ./...
```

Expected: 全部通过

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor: 去掉模块字母前缀，统一为 intake/parsing/agent/workbench/render"
```
