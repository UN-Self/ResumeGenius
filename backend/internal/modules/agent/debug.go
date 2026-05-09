package agent

import (
	"log"
	"os"
	"sync"
)

var debugEnabled = sync.OnceValue(func() bool {
	return os.Getenv("AGENT_DEBUG_LOG") != "false"
})

func debugLog(component string, format string, args ...any) {
	if debugEnabled() {
		log.Printf("[agent:"+component+"] "+format, args...)
	}
}

// truncateDebug truncates s to maxLen runes, appending "..." if truncated.
func truncateDebug(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// truncateHTML truncates HTML content to 200 runes.
func truncateHTML(s string) string { return truncateDebug(s, 200) }

// truncateParams truncates tool parameters JSON to 300 runes.
func truncateParams(s string) string { return truncateDebug(s, 300) }

// truncateResult truncates tool result to 200 runes.
func truncateResult(s string) string { return truncateDebug(s, 200) }
