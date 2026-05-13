# apply_edits CSS 规范化匹配方案

## 问题

`apply_edits` 使用 `strings.Contains(html, oldString)` 做精确子串匹配。AI 模型（如 glm-4.7）生成 CSS `old_string` 时经常改变格式（多行→单行），导致匹配失败。

典型失败场景：
```
存储格式（多行缩进）:
.resume-document .header {
  color: rgb(255, 255, 255);
  position: relative;
}

模型生成（单行）:
.resume-document .header { color: rgb(255, 255, 255); position: relative; }
```

日志分析显示，session 429 中约 60% 的 `apply_edits` 失败都是此原因。

## 方案

在 `applyEdits` 中增加 **CSS 空白规范化回退匹配**：

1. 先尝试精确匹配 `strings.Contains`（保持现有行为不变）
2. 失败后，对 `old_string` 和 HTML 做空白规范化，再匹配
3. 规范化匹配成功后，通过位置映射回原始 HTML 做替换

## 改动范围

仅修改 `backend/internal/modules/agent/tool_executor.go`，新增/改动：

### 1. 新增 `normalizeCSSWhitespace(s string) string`

将连续空白（空格、tab、换行）折叠为单个空格，并 trim 首尾。

```go
var multiWhitespace = regexp.MustCompile(`\s+`)

func normalizeCSSWhitespace(s string) string {
    return strings.TrimSpace(multiWhitespace.ReplaceAllString(s, " "))
}
```

### 2. 新增 `findWithCSSNormalization(html, oldString string) (int, int)`

在规范化后的 HTML 中查找规范化后的 oldString，返回原始 HTML 中的起止位置。

```go
func findWithCSSNormalization(html, oldString string) (start, end int) {
    normalizedOld := normalizeCSSWhitespace(oldString)
    if normalizedOld == "" {
        return -1, -1
    }

    // 规范化 HTML，保留每个字符到原始位置的映射
    var normalized strings.Builder
    positionMap := make([]int, 0, len(html))
    lastWasSpace := false

    for i, r := range html {
        if unicode.IsSpace(r) {
            if !lastWasSpace {
                normalized.WriteRune(' ')
                positionMap = append(positionMap, i)
                lastWasSpace = true
            }
        } else {
            normalized.WriteRune(r)
            positionMap = append(positionMap, i)
            lastWasSpace = false
        }
    }

    normalizedHTML := normalized.String()
    idx := strings.Index(normalizedHTML, normalizedOld)
    if idx == -1 {
        return -1, -1
    }

    start = positionMap[idx]
    endPos := idx + len(normalizedOld) - 1
    if endPos < len(positionMap) {
        end = positionMap[endPos] + 1
    } else {
        end = len(html)
    }
    return start, end
}
```

### 3. 修改 `applyEdits` 中的匹配逻辑（~line 482）

```go
// 原代码:
if !strings.Contains(html, op.OldString) {
    lastErr = fmt.Errorf(...)
    resultData.Failed++
    continue
}
html = strings.ReplaceAll(html, op.OldString, op.NewString)

// 改为:
if strings.Contains(html, op.OldString) {
    // 精确匹配路径（不变）
    html = strings.ReplaceAll(html, op.OldString, op.NewString)
} else if start, end := findWithCSSNormalization(html, op.OldString); start >= 0 {
    // 规范化匹配路径：用原始 HTML 中的实际内容替换
    html = html[:start] + op.NewString + html[end:]
    debugLog("tools", "操作 %d/%d: 通过 CSS 规范化匹配成功", i+1, len(ops))
} else {
    lastErr = fmt.Errorf("op #%d 匹配失败:\n%s", i+1, buildEditMatchError(op.OldString, html))
    resultData.Failed++
    continue
}
```

注意：规范化匹配路径用 `html[:start] + op.NewString + html[end:]` 替换，而不是 `strings.ReplaceAll`，因为规范化匹配定位的是原始 HTML 中的实际位置。

## 不做的事

- **不做模糊匹配/Levenshtein**：可能导致错误位置替换，风险太高
- **不做 CSS AST 解析**：过度工程化，空白折叠已覆盖主要场景
- **不改 prompt**：glm-4.7 的 CSS 格式化行为是模型层面的，prompt 难以精确控制
- **不改 `get_draft` 限制或 stall 保护逻辑**：这些是独立问题，本次不涉及

## 验证

1. 单元测试：构造多行 CSS 的 HTML，用单行 CSS 作为 old_string 验证匹配
2. 集成测试：用 docker compose 启动服务，复现日志中的编辑场景
3. 回归测试：确保精确匹配路径不受影响（`go test ./...`）
