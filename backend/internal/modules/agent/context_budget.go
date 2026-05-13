package agent

// TruncateToolResult applies tool-specific truncation strategies.
func TruncateToolResult(toolName, result, mode string) string {
	switch toolName {
	case "get_draft":
		switch mode {
		case "structure", "section":
			return result
		case "search":
			return truncateByLines(result, 50)
		default:
			return truncateWithNotice(result, 5000, "已截断，请使用 mode=section 或 mode=search 获取具体内容")
		}
	case "apply_edits":
		return truncateWithNotice(result, 1000, "...")
	case "search_assets":
		return truncateWithNotice(result, 3000, "...")
	case "error":
		return truncateWithNotice(result, 2000, "...")
	default:
		return truncateWithNotice(result, 1600, "...")
	}
}

func truncateWithNotice(s string, maxLen int, notice string) string {
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + notice
}

func truncateByLines(s string, maxLines int) string {
	lines := 0
	for i, c := range s {
		if c == '\n' {
			lines++
			if lines >= maxLines {
				return s[:i] + "\n...(已截断)"
			}
		}
	}
	return s
}
