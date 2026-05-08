package designskill

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed skill/ui-ux-pro-max/data/*.csv skill/ui-ux-pro-max/data/stacks/*.csv
var skillData embed.FS

type DomainConfig struct {
	File       string
	SearchCols []string
	OutputCols []string
}

type SkillSearchResult struct {
	Domain  string              `json:"domain,omitempty"`
	Stack   string              `json:"stack,omitempty"`
	Query   string              `json:"query"`
	File    string              `json:"file"`
	Count   int                 `json:"count"`
	Results []map[string]string `json:"results"`
}

var domainConfigs = map[string]DomainConfig{
	"style": {
		File:       "styles.csv",
		SearchCols: []string{"Style Category", "Keywords", "Best For", "Type"},
		OutputCols: []string{"Style Category", "Type", "Keywords", "Primary Colors", "Effects & Animation", "Best For", "Performance", "Accessibility", "Framework Compatibility", "Complexity"},
	},
	"prompt": {
		File:       "prompts.csv",
		SearchCols: []string{"Style Category", "AI Prompt Keywords (Copy-Paste Ready)", "CSS/Technical Keywords"},
		OutputCols: []string{"Style Category", "AI Prompt Keywords (Copy-Paste Ready)", "CSS/Technical Keywords", "Implementation Checklist"},
	},
	"color": {
		File:       "colors.csv",
		SearchCols: []string{"Product Type", "Keywords", "Notes"},
		OutputCols: []string{"Product Type", "Keywords", "Primary (Hex)", "Secondary (Hex)", "CTA (Hex)", "Background (Hex)", "Text (Hex)", "Border (Hex)", "Notes"},
	},
	"chart": {
		File:       "charts.csv",
		SearchCols: []string{"Data Type", "Keywords", "Best Chart Type", "Accessibility Notes"},
		OutputCols: []string{"Data Type", "Keywords", "Best Chart Type", "Secondary Options", "Color Guidance", "Accessibility Notes", "Library Recommendation", "Interactive Level"},
	},
	"landing": {
		File:       "landing.csv",
		SearchCols: []string{"Pattern Name", "Keywords", "Conversion Optimization", "Section Order"},
		OutputCols: []string{"Pattern Name", "Keywords", "Section Order", "Primary CTA Placement", "Color Strategy", "Conversion Optimization"},
	},
	"product": {
		File:       "products.csv",
		SearchCols: []string{"Product Type", "Keywords", "Primary Style Recommendation", "Key Considerations"},
		OutputCols: []string{"Product Type", "Keywords", "Primary Style Recommendation", "Secondary Styles", "Landing Page Pattern", "Dashboard Style (if applicable)", "Color Palette Focus"},
	},
	"ux": {
		File:       "ux-guidelines.csv",
		SearchCols: []string{"Category", "Issue", "Description", "Platform"},
		OutputCols: []string{"Category", "Issue", "Platform", "Description", "Do", "Don't", "Code Example Good", "Code Example Bad", "Severity"},
	},
	"typography": {
		File:       "typography.csv",
		SearchCols: []string{"Font Pairing Name", "Category", "Mood/Style Keywords", "Best For", "Heading Font", "Body Font"},
		OutputCols: []string{"Font Pairing Name", "Category", "Heading Font", "Body Font", "Mood/Style Keywords", "Best For", "Google Fonts URL", "CSS Import", "Tailwind Config", "Notes"},
	},
}

var stackConfigs = map[string]string{
	"html-tailwind": "stacks/html-tailwind.csv",
	"react":         "stacks/react.csv",
	"nextjs":        "stacks/nextjs.csv",
	"vue":           "stacks/vue.csv",
	"svelte":        "stacks/svelte.csv",
	"swiftui":       "stacks/swiftui.csv",
	"react-native":  "stacks/react-native.csv",
	"flutter":       "stacks/flutter.csv",
}

var tokenPattern = regexp.MustCompile(`[^\p{L}\p{N}_]+`)

func SearchSkill(query, domain, stack string, maxResults int) (SkillSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SkillSearchResult{}, fmt.Errorf("query is required")
	}
	if maxResults <= 0 {
		maxResults = 3
	}
	if stack != "" {
		return searchStack(stack, query, maxResults)
	}
	if domain == "" {
		domain = DetectDomain(query)
	}
	return searchDomain(domain, query, maxResults)
}

func DetectDomain(query string) string {
	lower := strings.ToLower(query)
	switch {
	case containsAny(lower, "color", "palette", "hex", "rgb", "配色", "颜色"):
		return "color"
	case containsAny(lower, "font", "typography", "字体", "排版"):
		return "typography"
	case containsAny(lower, "chart", "graph", "visualization", "图表"):
		return "chart"
	case containsAny(lower, "landing", "conversion", "hero"):
		return "landing"
	case containsAny(lower, "ux", "accessibility", "form", "empty", "loading", "交互", "可用性"):
		return "ux"
	case containsAny(lower, "react", "vue", "tailwind", "nextjs", "component"):
		return "product"
	default:
		return "style"
	}
}

func searchDomain(domain, query string, maxResults int) (SkillSearchResult, error) {
	cfg, ok := domainConfigs[domain]
	if !ok {
		return SkillSearchResult{}, fmt.Errorf("unknown domain %q", domain)
	}
	path := filepath.ToSlash(filepath.Join("skill", "ui-ux-pro-max", "data", cfg.File))
	rows, err := readCSV(path)
	if err != nil {
		return SkillSearchResult{}, err
	}
	results := rankRows(rows, cfg.SearchCols, cfg.OutputCols, query, maxResults)
	return SkillSearchResult{Domain: domain, Query: query, File: path, Count: len(results), Results: results}, nil
}

func searchStack(stack, query string, maxResults int) (SkillSearchResult, error) {
	file, ok := stackConfigs[stack]
	if !ok {
		return SkillSearchResult{}, fmt.Errorf("unknown stack %q", stack)
	}
	path := filepath.ToSlash(filepath.Join("skill", "ui-ux-pro-max", "data", file))
	rows, err := readCSV(path)
	if err != nil {
		return SkillSearchResult{}, err
	}
	results := rankRows(
		rows,
		[]string{"Category", "Guideline", "Description", "Do", "Don't"},
		[]string{"Category", "Guideline", "Description", "Do", "Don't", "Code Good", "Code Bad", "Severity", "Docs URL"},
		query,
		maxResults,
	)
	return SkillSearchResult{Stack: stack, Query: query, File: path, Count: len(results), Results: results}, nil
}

func readCSV(path string) ([]map[string]string, error) {
	file, err := skillData.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	headers := records[0]
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(record) {
				row[header] = record[i]
			} else {
				row[header] = ""
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type scoredRow struct {
	Index int
	Score float64
}

func rankRows(rows []map[string]string, searchCols, outputCols []string, query string, maxResults int) []map[string]string {
	docs := make([][]string, len(rows))
	docFreq := map[string]int{}
	totalLength := 0
	for i, row := range rows {
		doc := tokenize(joinColumns(row, searchCols))
		docs[i] = doc
		totalLength += len(doc)
		seen := map[string]struct{}{}
		for _, token := range doc {
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			docFreq[token]++
		}
	}
	queryTokens := tokenize(query)
	if len(rows) == 0 || len(queryTokens) == 0 {
		return nil
	}

	avgLength := float64(totalLength) / float64(len(rows))
	if avgLength == 0 {
		avgLength = 1
	}
	const k1 = 1.5
	const b = 0.75

	scored := make([]scoredRow, 0, len(rows))
	for i, doc := range docs {
		termFreq := map[string]int{}
		for _, token := range doc {
			termFreq[token]++
		}
		score := 0.0
		docLength := float64(len(doc))
		for _, token := range queryTokens {
			df := docFreq[token]
			if df == 0 {
				continue
			}
			tf := float64(termFreq[token])
			idf := math.Log((float64(len(rows))-float64(df)+0.5)/(float64(df)+0.5) + 1)
			score += idf * (tf * (k1 + 1)) / (tf + k1*(1-b+b*docLength/avgLength))
		}
		if score > 0 {
			scored = append(scored, scoredRow{Index: i, Score: score})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if maxResults <= 0 || maxResults > len(scored) {
		maxResults = len(scored)
	}
	out := make([]map[string]string, 0, maxResults)
	for _, item := range scored[:maxResults] {
		row := rows[item.Index]
		filtered := make(map[string]string, len(outputCols))
		for _, col := range outputCols {
			if value, ok := row[col]; ok {
				filtered[col] = value
			}
		}
		out = append(out, filtered)
	}
	return out
}

func joinColumns(row map[string]string, columns []string) string {
	values := make([]string, 0, len(columns))
	for _, col := range columns {
		values = append(values, row[col])
	}
	return strings.Join(values, " ")
}

func tokenize(value string) []string {
	lower := strings.ToLower(value)
	parts := strings.Fields(tokenPattern.ReplaceAllString(lower, " "))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if len([]rune(part)) > 1 {
			tokens = append(tokens, part)
		}
	}
	return tokens
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}
