# Agent 模块调试日志设计

日期: 2026-05-09

## 背景

Agent 模块当前仅有 5 条日志，AI 对话全流程几乎不可观测。模型调用、工具执行、ReAct 循环、压缩触发等关键环节均为黑箱，调试困难。

## 目标

让 Agent 全链路可观测：每次请求从进入到结束，能看到模型调用了什么工具、传了什么参数、结果如何、耗时多少。

## 方案

### 基础设施：`debug.go`

新建 `backend/internal/modules/agent/debug.go`，提供统一的调试日志函数。

```go
package agent

import (
    "fmt"
    "log"
    "os"
    "sync"
)

var debugEnabled = sync.OnceValue(func() bool {
    return os.Getenv("AGENT_DEBUG_LOG") != "false"
})

func debugLog(component string, format string, args ...interface{}) {
    if debugEnabled() {
        log.Printf("[agent:%s] %s", component, fmt.Sprintf(format, args...))
    }
}
```

- 环境变量 `AGENT_DEBUG_LOG=false` 关闭所有调试日志
- 不设置时默认开启
- 日志前缀格式：`[agent:组件名]`
- 日志描述使用中文
- 原有 5 条日志保持不变

### 截断规则

避免日志刷屏，长内容统一截断：

| 内容类型 | 截断长度 |
|---|---|
| HTML 内容 | 前 200 字符 + `...` |
| 工具参数 JSON | 前 300 字符 + `...` |
| 工具返回结果 | 前 200 字符 + `...` |
| 模型原始 SSE 行 | 不记录（量太大） |

### provider.go 日志点

组件前缀：`agent:provider`

| 时机 | 日志内容 |
|---|---|
| API 调用前 | 正在调用模型 {model}，消息数 {n}，工具数 {n} |
| 响应状态码 | 模型响应，状态码 {code}，耗时 {duration} |
| 收到工具调用 | 模型请求调用 {n} 个工具：{tool_names} |
| 工具参数解析失败 | 工具 {name} 参数解析失败：{err} |
| 流读取异常 | SSE 流读取异常：{err}（保留原有 malformed line 日志） |
| 请求超时 | 模型调用超时，耗时 {duration} |

### service.go 日志点

组件前缀：`agent:service`

| 时机 | 日志内容 |
|---|---|
| 进入 StreamChatReAct | 收到请求，session={id}，draft={id}，project={id} |
| 用户消息 | 用户消息长度 {len} 字符 |
| 加载历史 | 历史消息 {n} 条，token 估算 {tokens} |
| 压缩触发 | 压缩触发，token 估算 {tokens}，压缩前 {before} 条消息 |
| 压缩完成 | 压缩完成，压缩后 {after} 条消息，耗时 {duration} |
| 压缩失败 | 压缩失败，使用原始消息：{err} |
| 资源预加载 | 预加载 {n} 个资源，总长度 {len} 字符 |
| 每轮迭代 | 第 {n} 轮迭代，消息数 {msg_count}，token 估算 {tokens} |
| stall 保护触发 | stall 保护触发，连续 {n} 轮无输出，已注入提醒 |
| 搜索过多提醒 | 搜索过多提醒触发，连续 {n} 轮未执行 apply_edits |
| 循环结束 | 循环结束，共 {n} 轮，总耗时 {duration} |
| 保存回复 | 保存助手回复，长度 {len} 字符 |

### tool_executor.go 日志点

组件前缀：`agent:tools`

| 时机 | 日志内容 |
|---|---|
| 工具调用开始 | 调用工具 {name}，参数摘要：{truncated_params} |
| apply_edits 操作数 | apply_edits 开始，共 {n} 个操作 |
| apply_edits 每条操作 | 操作 {i}/{n}：old={truncated_old} → new={truncated_new} |
| apply_edits 验证失败 | 操作 {i} 验证失败：{reason} |
| apply_edits 成功 | apply_edits 完成，成功 {n} 个编辑，新序列号 {seq}，耗时 {duration} |
| apply_edits 失败 | apply_edits 失败：{err}，耗时 {duration} |
| get_draft | get_draft，selector={selector}，返回 HTML 长度 {len} |
| search_assets | search_assets，查询={query}，结果 {n} 条 |
| load_skill | 加载技能 {name} |
| get_skill_reference | 获取技能参考文档 {skill}/{ref} |
| 工具执行完成 | 工具 {name} 执行{成功/失败}，耗时 {duration} |

### skill_loader.go 日志点

组件前缀：`agent:skills`

| 时机 | 日志内容 |
|---|---|
| 加载完成 | 技能加载完成，共 {n} 个技能：{skill_names} |
| 参考文档加载 | 技能 {name} 加载了 {n} 个参考文档 |

## 改动范围

| 文件 | 操作 | 改动量 |
|---|---|---|
| `debug.go` | 新建 | ~20 行 |
| `provider.go` | 修改 | +~6 条日志 |
| `service.go` | 修改 | +~10 条日志 |
| `tool_executor.go` | 修改 | +~8 条日志 |
| `skill_loader.go` | 修改 | +~2 条日志 |

不改动：`handler.go`（HTTP 层由 middleware.Logger 覆盖）、`thinking_recorder.go`（保留 AGENT_THINKING_LOG 机制）。
