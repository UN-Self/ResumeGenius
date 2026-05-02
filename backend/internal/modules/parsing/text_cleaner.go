package parsing

import (
	"regexp"
	"strings"
)

var (
	inlineWhitespacePattern   = regexp.MustCompile(`[ \t]+`)
	separatorLinePattern      = regexp.MustCompile(`^[\-=_~*•·.]{3,}$`)
	englishPageNumberPattern  = regexp.MustCompile(`(?i)^page\s+\d+(?:\s*(?:/|of)\s*\d+)?$`)
	fractionPageNumberPattern = regexp.MustCompile(`^\d+\s*/\s*\d+$`)
	chinesePageNumberPattern  = regexp.MustCompile(`^第\s*\d+\s*页(?:\s*/\s*共?\s*\d+\s*页)?$`)
)

// cleanParsedText removes low-signal formatting noise while preserving
// paragraphs and list structure for downstream AI consumption.
func cleanParsedText(raw string) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	cleaned := make([]string, 0, len(lines))
	blankCount := 0

	for _, line := range lines {
		cleanedLine, isExplicitBlank := cleanParsedLine(line)
		if cleanedLine == "" {
			if !isExplicitBlank || len(cleaned) == 0 || blankCount > 0 {
				continue
			}
			cleaned = append(cleaned, "")
			blankCount++
			continue
		}

		cleaned = append(cleaned, cleanedLine)
		blankCount = 0
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func cleanParsedLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", true
	}
	if isSeparatorLine(trimmed) || isSimplePageNumberLine(trimmed) {
		return "", false
	}

	trimmed = normalizeBulletLine(trimmed)
	trimmed = inlineWhitespacePattern.ReplaceAllString(trimmed, " ")
	return strings.TrimSpace(trimmed), false
}

func normalizeBulletLine(line string) string {
	for _, prefix := range []string{"•", "·", "●", "○", "▪", "◦", "‣", "*"} {
		if strings.HasPrefix(line, prefix) {
			body := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if body == "" {
				return ""
			}
			return "- " + body
		}
	}

	if strings.HasPrefix(line, "-") {
		body := strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if body == "" {
			return ""
		}
		return "- " + body
	}

	return line
}

func isSeparatorLine(line string) bool {
	return separatorLinePattern.MatchString(line)
}

func isSimplePageNumberLine(line string) bool {
	return englishPageNumberPattern.MatchString(line) ||
		fractionPageNumberPattern.MatchString(line) ||
		chinesePageNumberPattern.MatchString(line)
}

func truncateCleanedText(input string, limit int) string {
	cleaned := cleanParsedText(input)
	if cleaned == "" {
		return ""
	}
	if limit <= 0 {
		return cleaned
	}

	runes := []rune(cleaned)
	if len(runes) <= limit {
		return cleaned
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func cleanParsedContentText(parsed *ParsedContent) *ParsedContent {
	if parsed == nil {
		return nil
	}

	cleaned := *parsed
	cleaned.Text = cleanParsedText(cleaned.Text)
	return &cleaned
}
