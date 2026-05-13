package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBestMatchNode_Found(t *testing.T) {
	html := `<html><body>
		<div class="experience">
			<h3>Google</h3>
			<p>Senior Engineer 2020-2024</p>
		</div>
		<div class="education">
			<h3>MIT</h3>
			<p>Computer Science</p>
		</div>
	</body></html>`

	node, score := FindBestMatchNode(html, "Senior Engineer")
	require.NotNil(t, node)
	assert.Greater(t, score, 0.0)
	assert.Contains(t, node.OuterHTML, "Senior Engineer")
}

func TestFindBestMatchNode_NotFound(t *testing.T) {
	html := `<html><body><p>Hello World</p></body></html>`
	node, score := FindBestMatchNode(html, "zzzzzzzzzzzz")
	assert.Nil(t, node)
	assert.Equal(t, 0.0, score)
}

func TestFindBestMatchNode_LongestCommonSubstring(t *testing.T) {
	html := `<html><body><p>Senior Software Engineer at Google</p></body></html>`
	node, score := FindBestMatchNode(html, "Sr. Software Engineer")
	require.NotNil(t, node)
	assert.Greater(t, score, 0.0)
	assert.Contains(t, node.OuterHTML, "Senior Software Engineer")
}

func TestTruncateAroundMatch(t *testing.T) {
	html := `<html><body>
		<div class="header"><h1>Resume</h1></div>
		<div class="experience">
			<div class="job">
				<h3>Google</h3>
				<p>Senior Engineer from 2020 to 2024, worked on search infrastructure</p>
			</div>
		</div>
		<div class="education"><h3>MIT</h3></div>
	</body></html>`

	snippet := TruncateAroundMatch(html, "Senior Engineer", 100)
	assert.Contains(t, snippet, "Senior Engineer")
	assert.LessOrEqual(t, len(snippet), 300)
}

func TestTruncateAroundMatch_MultipleOccurrences(t *testing.T) {
	html := `<html><body>
		<p>First occurrence of test</p>
		<p>Second occurrence of test here</p>
	</body></html>`

	snippet := TruncateAroundMatch(html, "test", 50)
	assert.Contains(t, snippet, "test")
}

func TestBuildStructureOverview(t *testing.T) {
	html := `<html>
	<head><style>body{font-size:14px} .header{color:red}</style></head>
	<body>
		<div class="header"><h1>John Doe</h1><p>Engineer</p></div>
		<div class="section experience">
			<h2>Experience</h2>
			<div class="item"><h3>Google</h3><p>Senior Engineer</p></div>
		</div>
		<div class="section education">
			<h2>Education</h2>
			<div class="item"><h3>MIT</h3></div>
		</div>
	</body></html>`

	overview := BuildStructureOverview(html)
	assert.Contains(t, overview, "<html>")
	assert.Contains(t, overview, "<body>")
	assert.Contains(t, overview, "header")
	assert.Contains(t, overview, "experience")
	assert.NotContains(t, overview, "John Doe")
	assert.NotContains(t, overview, "Senior Engineer")
}

func TestBuildStructureOverview_EmptyHTML(t *testing.T) {
	overview := BuildStructureOverview("")
	assert.Equal(t, "", overview)
}

func TestBuildStructureOverview_StyleTruncated(t *testing.T) {
	longCSS := "<style>" + strings.Repeat("body{color:red}", 100) + "</style>"
	html := "<html><head>" + longCSS + "</head><body><div>content</div></body></html>"

	overview := BuildStructureOverview(html)
	assert.Contains(t, overview, "<style>")
	assert.NotContains(t, overview, strings.Repeat("body{color:red}", 100))
}

func TestSearchInDraft_Found(t *testing.T) {
	html := `<html><body>
		<div class="experience">
			<h3>Google</h3>
			<p>Senior Engineer 2020-2024</p>
		</div>
		<div class="education">
			<h3>MIT</h3>
			<p>Computer Science</p>
		</div>
	</body></html>`

	result := searchInDraft(html, "Engineer")
	assert.Contains(t, result, "Senior Engineer")
	assert.Contains(t, result, "找到")
}

func TestSearchInDraft_NotFound(t *testing.T) {
	html := `<html><body><p>Hello World</p></body></html>`
	result := searchInDraft(html, "nonexistent")
	assert.Contains(t, result, "未找到")
}

func TestSearchInDraft_MaxResults(t *testing.T) {
	html := "<html><body>" + strings.Repeat("<p>keyword here</p>\n", 10) + "</body></html>"
	result := searchInDraft(html, "keyword")
	assert.Contains(t, result, "找到 5 处匹配")
}

func TestLongestCommonSubstring(t *testing.T) {
	tests := []struct {
		a, b, expected string
	}{
		{"abcde", "cdefg", "cde"},
		{"hello", "world", "l"},
		{"", "abc", ""},
		{"abc", "", ""},
		{"abc", "xyz", ""},
		{"abcde", "abcde", "abcde"},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := longestCommonSubstring(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	assert.Equal(t, "hello", truncateString("hello", 10))
	assert.Equal(t, "hello...", truncateString("hello world", 5))
	assert.Equal(t, "", truncateString("", 5))
}
