package agent

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// MatchResult represents a matched HTML node with context.
type MatchResult struct {
	OuterHTML string  // 匹配节点的 OuterHTML
	Score     float64 // 相似度分数 (0-1)
	ParentTag string  // 父元素标签名
	TagName   string  // 匹配节点标签名
}

// FindBestMatchNode searches all text nodes in html for the one most similar to query.
// Returns nil if no node contains any common substring.
func FindBestMatchNode(html, query string) (*MatchResult, float64) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, 0
	}

	var bestResult *MatchResult
	bestScore := 0.0

	doc.Find("*").Each(func(_ int, sel *goquery.Selection) {
		text := sel.Text()
		if len(strings.TrimSpace(text)) == 0 {
			return
		}

		lcs := longestCommonSubstring(strings.ToLower(text), strings.ToLower(query))
		if len(lcs) == 0 {
			return
		}

		score := float64(len(lcs)) / float64(len(query))
		if score <= bestScore || score < 0.1 {
			return
		}

		outerHTML, _ := sel.Html()
		tagName := goquery.NodeName(sel)
		parentTag := ""
		if parent := sel.Parent(); parent != nil {
			parentTag = goquery.NodeName(parent)
		}

		bestScore = score
		bestResult = &MatchResult{
			OuterHTML: outerHTML,
			Score:     score,
			ParentTag: parentTag,
			TagName:   tagName,
		}
	})

	return bestResult, bestScore
}

// longestCommonSubstring returns the longest common substring of a and b.
func longestCommonSubstring(a, b string) string {
	if len(a) == 0 || len(b) == 0 {
		return ""
	}

	prev := make([]int, len(b)+1)
	maxLen := 0
	endIdx := 0

	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
				if curr[j] > maxLen {
					maxLen = curr[j]
					endIdx = i
				}
			}
		}
		prev = curr
	}

	if maxLen == 0 {
		return ""
	}
	return a[endIdx-maxLen : endIdx]
}

// TruncateAroundMatch finds the first occurrence of query in html and returns
// a snippet with contextRadius characters of context around it.
func TruncateAroundMatch(html, query string, contextRadius int) string {
	idx := strings.Index(strings.ToLower(html), strings.ToLower(query))
	if idx < 0 {
		result, _ := FindBestMatchNode(html, query)
		if result != nil {
			return truncateString(result.OuterHTML, contextRadius*2+len(query))
		}
		return truncateString(html, contextRadius*2)
	}

	start := idx - contextRadius
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + contextRadius
	if end > len(html) {
		end = len(html)
	}

	snippet := html[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(html) {
		snippet = snippet + "..."
	}
	return snippet
}

func truncateString(s string, maxLen int) string {
	return truncateWithNotice(s, maxLen, "...")
}

// BuildStructureOverview returns a compact HTML tag tree showing structure
// without text content. Style tags are truncated to 200 chars.
func BuildStructureOverview(html string) string {
	if len(html) == 0 {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return truncateString(html, 500)
	}

	htmlEl := doc.Find("html").First()
	if htmlEl.Length() == 0 {
		return truncateString(html, 500)
	}

	var b strings.Builder
	b.WriteString("<html>\n")
	buildNodeOverview(htmlEl, &b, 1)
	return b.String()
}

func buildNodeOverview(sel *goquery.Selection, b *strings.Builder, depth int) {
	if depth > 6 {
		return
	}

	sel.Contents().Each(func(_ int, child *goquery.Selection) {
		if child.Is("text") || child.Is("comment") {
			return
		}

		tag := goquery.NodeName(child)

		class, _ := child.Attr("class")
		id, _ := child.Attr("id")

		b.WriteString(strings.Repeat("  ", depth))
		b.WriteString("<" + tag)
		if id != "" {
			b.WriteString(` id="` + id + `"`)
		}
		if class != "" {
			b.WriteString(` class="` + class + `"`)
		}
		b.WriteString(">")

		if tag == "style" {
			styleText := child.Text()
			if len(styleText) > 200 {
				b.WriteString(truncateString(styleText, 200))
			} else {
				b.WriteString(styleText)
			}
			b.WriteString("</style>")
			return
		}

		b.WriteString("\n")
		buildNodeOverview(child, b, depth+1)
	})
}

// searchInDraft finds up to 5 snippets in html that contain query,
// each with ±200 characters of context.
func searchInDraft(html, query string) string {
	lowerHTML := strings.ToLower(html)
	lowerQuery := strings.ToLower(query)

	var results []string
	searchFrom := 0
	maxResults := 5
	radius := 200

	for len(results) < maxResults {
		idx := strings.Index(lowerHTML[searchFrom:], lowerQuery)
		if idx < 0 {
			break
		}
		absIdx := searchFrom + idx

		start := absIdx - radius
		if start < 0 {
			start = 0
		}
		end := absIdx + len(query) + radius
		if end > len(html) {
			end = len(html)
		}

		snippet := html[start:end]
		if start > 0 {
			snippet = "..." + snippet
		}
		if end < len(html) {
			snippet = snippet + "..."
		}

		results = append(results, snippet)
		searchFrom = absIdx + len(query)
	}

	if len(results) == 0 {
		return fmt.Sprintf("未找到包含 %q 的内容", query)
	}

	return fmt.Sprintf("找到 %d 处匹配:\n\n%s", len(results), strings.Join(results, "\n\n---\n\n"))
}
