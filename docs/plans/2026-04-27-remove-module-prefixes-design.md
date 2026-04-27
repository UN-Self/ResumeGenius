# 去掉模块字母前缀

**日期**: 2026-04-27
**状态**: 已批准

## 背景

模块目录和包名使用字母前缀（a_intake, b_parsing 等）最初用于排序和快速识别模块。随着项目成熟，前缀增加了认知负担且无实际价值，决定统一去掉。

## 命名映射

| 旧名 | 新名 | 错误码旧前缀 | 错误码新前缀 |
|---|---|---|---|
| `a_intake` / `a-intake` | `intake` | A=01xxx | intake=01xxx |
| `b_parsing` / `b-parsing` | `parsing` | B=02xxx | parsing=02xxx |
| `c_agent` / `c-agent` | `agent` | C=03xxx | agent=03xxx |
| `d_workbench` / `d-workbench` | `workbench` | D=04xxx | workbench=04xxx |
| `e_render` / `e-render` | `render` | E=05xxx | render=05xxx |

## 变更清单

### 目录重命名（10 个）

- `backend/internal/modules/{a_intake,...,e_render}/` → `{intake,...,render}/`
- `docs/modules/{a-intake,...,e-render}/` → `{intake,...,render}/`

### 文件重命名（5 个）

- `docs/plans/2026-04-26-module-{a-intake,...,e-render}.md` → `module-{intake,...,render}.md`

### Go 代码修改（6 个文件）

- 5 个 `routes.go`：`package` 声明 + `"module"` 字符串字面量
- `main.go`：import 路径 + `RegisterRoutes` 调用

### 文档内容修改（12 个文件）

- `CLAUDE.md`
- `docs/README.md`
- `docs/01-product/api-conventions.md`
- `docs/01-product/dev-work-breakdown.md`
- `docs/superpowers/specs/2026-04-23-architecture-v2-design.md`
- `docs/plans/2026-04-27-routing-alignment-implementation.md`
- `docs/plans/2026-04-26-phase0-shared-foundation.md`
- `docs/plans/2026-04-26-module-intake.md`
- `docs/plans/2026-04-26-module-parsing.md`
- `docs/plans/2026-04-26-module-agent.md`
- `docs/plans/2026-04-26-module-workbench.md`
- `docs/plans/2026-04-26-module-render.md`

## 不变更项

- `go.mod`（模块路径不含前缀）
- `docker-compose.yml`
- 前端代码（无硬编码模块名引用）
- API URL 路径（`/api/v1/projects` 等不变）
- 数据库表名（不变）

## 验证

- `go build ./...` 编译通过
- `go test ./...` 测试通过
- 全局搜索确认无残留的 `a_intake`、`b_parsing`、`c_agent`、`d_workbench`、`e_render`

## 提交

单次 commit：`refactor: 去掉模块字母前缀，统一为 intake/parsing/agent/workbench/render`
