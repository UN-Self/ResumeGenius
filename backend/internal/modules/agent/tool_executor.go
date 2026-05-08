package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/modules/designskill"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// ---------------------------------------------------------------------------
// Context keys
// ---------------------------------------------------------------------------

type contextKey int

const (
	draftIDKey contextKey = iota
	projectIDKey
)

// WithDraftID returns a context carrying the given draft ID.
func WithDraftID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, draftIDKey, id)
}

// WithProjectID returns a context carrying the given project ID.
func WithProjectID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, projectIDKey, id)
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

// ToolDef represents a tool definition for OpenAI function calling.
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolExecutor defines the interface for executing AI tool calls.
type ToolExecutor interface {
	// Tools returns the list of tool definitions.
	Tools() []ToolDef
	// Execute runs a tool by name with the given parameters and returns the result as a JSON string.
	Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error)
}

// AgentToolExecutor implements ToolExecutor using database queries.
type AgentToolExecutor struct {
	db          *gorm.DB
	skillLoader *SkillLoader
}

// NewAgentToolExecutor creates a new AgentToolExecutor.
func NewAgentToolExecutor(db *gorm.DB, skillLoader *SkillLoader) *AgentToolExecutor {
	return &AgentToolExecutor{db: db, skillLoader: skillLoader}
}

var resumeDesignConstraints = []string{
	"只设计常规招聘简历，不设计网页、落地页、仪表盘、海报或作品集。",
	"默认控制在一页 A4，优先压缩内容、字号、行距和间距，不通过夸张视觉扩张版面。",
	"使用白色或浅色纸面、深色正文、最多一个克制强调色，保持高对比和可打印。",
	"禁止 hero、bento/card grid、glassmorphism、aurora、3D、霓虹、复杂渐变、动画、发光、厚重阴影和大面积彩色背景。",
	"正文保持 13-15px 左右，姓名标题不超过 24px，分区标题 14-16px，技能列表必须可换行。",
}

const resumeDesignFallbackQuery = "professional business document resume A4 one page minimal swiss flat print readable"
const resumeDesignQuerySuffix = "professional business document resume A4 one page minimal swiss flat print readable conservative"

var resumeDesignBlockedResultTerms = []string{
	"aurora",
	"bento",
	"brutal",
	"card grid",
	"clay",
	"cyber",
	"dashboard",
	"glass",
	"hero",
	"hyperrealism",
	"immersive",
	"landing",
	"liquid",
	"motion",
	"neon",
	"portfolio",
	"retro",
	"skeuomorphism",
	"showcase",
	"social proof",
	"storytelling",
	"three",
	"vibrant",
	"video",
	"3d",
}

type resumeDesignSkillResponse struct {
	designskill.SkillSearchResult
	ResumeConstraints []string `json:"resume_constraints"`
	QueryUsed         string   `json:"query_used"`
	DomainUsed        string   `json:"domain_used,omitempty"`
	Note              string   `json:"note,omitempty"`
}

// Tools returns the AI-callable tool definitions.
func (e *AgentToolExecutor) Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "get_draft",
			Description: "读取当前简历 HTML。不带参数返回完整 HTML，带 selector 参数只返回匹配的片段（CSS 选择器）。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS 选择器，例如 '#experience'、'.skill-item'。不传则返回完整 HTML。",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "apply_edits",
			Description: "对简历 HTML 应用搜索替换编辑。提交一组操作，全部验证通过后原子执行。old_string 必须精确匹配当前 HTML。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ops": map[string]interface{}{
						"type":        "array",
						"description": "搜索替换操作数组",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"old_string":  map[string]interface{}{"type": "string", "description": "必须在当前 HTML 中精确匹配的文本"},
								"new_string":  map[string]interface{}{"type": "string", "description": "替换后的文本"},
								"description": map[string]interface{}{"type": "string", "description": "修改说明（可选）"},
							},
							"required": []string{"old_string", "new_string"},
						},
					},
				},
				"required": []string{"ops"},
			},
		},
		{
			Name:        "search_assets",
			Description: "搜索用户资料（旧简历、Git 摘要、笔记等）。可按关键词和类型过滤。长内容返回前 2000 字符。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "搜索关键词"},
					"type":  map[string]interface{}{"type": "string", "description": "资料类型：resume | git_summary | note"},
					"limit": map[string]interface{}{"type": "integer", "description": "返回数量上限，默认 5"},
				},
				"required": []string{},
			},
		},
		{
			Name:        "search_skills",
			Description: "搜索简历优化技能库（岗位面经、简历建议、A4 设计规范）。根据目标岗位或任务关键词查找建议；视觉/排版任务优先用 keyword='简历设计 A4 单页' category='design'。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type":        "string",
						"description": "搜索关键词，按岗位名、技能名或任务匹配，如'测试工程师'、'QA'、'自动化测试'、'简历设计 A4 单页'",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "技能分类名，如'test'、'tech'、'management'、'design'",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "返回数量上限，默认 3",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "search_design_skill",
			Description: "查询受简历约束的 ui-ux-pro-max 设计参考。仅用于常规 A4 简历的保守字体、配色、极简/瑞士风格辅助；不得用于 landing page、hero、bento、aurora、glass、3D、dashboard 等网页化设计。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":  map[string]interface{}{"type": "string", "description": "简历设计需求，例如 professional A4 resume layout、简历 排版 字体、简历 克制配色"},
					"domain": map[string]interface{}{"type": "string", "description": "可选且建议仅用：style | color | ux | typography。prompt/landing/product/chart 会被收敛到简历安全范围。"},
					"stack":  map[string]interface{}{"type": "string", "description": "兼容旧参数；简历设计中通常会忽略技术栈建议，避免生成网页 UI。"},
					"limit":  map[string]interface{}{"type": "integer", "description": "返回数量上限，默认 3"},
				},
				"required": []string{"query"},
			},
		},
	}
}

// Execute dispatches to the correct tool implementation based on toolName.
func (e *AgentToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	switch toolName {
	case "get_draft":
		return e.getDraft(ctx, params)
	case "apply_edits":
		return e.applyEdits(ctx, params)
	case "search_assets":
		return e.searchAssets(ctx, params)
	case "search_skills":
		return e.searchSkills(ctx, params)
	case "search_design_skill":
		return e.searchDesignSkill(ctx, params)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// ---------------------------------------------------------------------------
// design skill tool
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) searchDesignSkill(ctx context.Context, params map[string]interface{}) (string, error) {
	query, _ := params["query"].(string)
	domain, _ := params["domain"].(string)
	stack, _ := params["stack"].(string)
	limit := 3
	if _, ok := params["limit"]; ok {
		if n, err := getIntParam(params, "limit"); err == nil && n > 0 {
			limit = n
		}
	}

	query, domain, stack, note := normalizeResumeDesignSearch(query, domain, stack)
	result, err := designskill.SearchSkill(query, domain, stack, limit)
	if err != nil {
		return "", err
	}
	result = filterResumeDesignResult(result)
	if result.Count == 0 && (domain != "style" || query != resumeDesignFallbackQuery) {
		fallback, fallbackErr := designskill.SearchSkill(resumeDesignFallbackQuery, "style", "", limit)
		if fallbackErr == nil {
			result = filterResumeDesignResult(fallback)
			if note != "" {
				note += " "
			}
			note += "已回退到保守 A4 简历风格查询。"
		}
	}

	resp := resumeDesignSkillResponse{
		SkillSearchResult: result,
		ResumeConstraints: append([]string(nil), resumeDesignConstraints...),
		QueryUsed:         query,
		DomainUsed:        result.Domain,
		Note:              strings.TrimSpace(note),
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
}

func normalizeResumeDesignSearch(query, domain, stack string) (string, string, string, string) {
	query = strings.TrimSpace(query)
	domain = strings.ToLower(strings.TrimSpace(domain))
	stack = strings.ToLower(strings.TrimSpace(stack))

	var notes []string
	if query == "" {
		query = resumeDesignFallbackQuery
		notes = append(notes, "空查询已替换为保守 A4 简历设计查询。")
	} else if !strings.Contains(strings.ToLower(query), "resume") && !strings.Contains(query, "简历") {
		query += " resume"
	}

	switch domain {
	case "", "style", "color", "ux", "typography":
	case "prompt", "landing", "product", "chart":
		domain = "style"
		notes = append(notes, "网页/产品/图表类 domain 已收敛为 style，避免生成非简历 UI。")
	default:
		domain = "style"
		notes = append(notes, "未知 domain 已收敛为 style。")
	}

	if stack != "" {
		stack = ""
		notes = append(notes, "技术栈建议已忽略，避免把简历改成前端应用界面。")
	}

	if !strings.Contains(strings.ToLower(query), "a4") {
		query += " " + resumeDesignQuerySuffix
	}
	return strings.TrimSpace(query), domain, stack, strings.Join(notes, " ")
}

func filterResumeDesignResult(result designskill.SkillSearchResult) designskill.SkillSearchResult {
	if len(result.Results) == 0 {
		return result
	}

	filtered := make([]map[string]string, 0, len(result.Results))
	for _, row := range result.Results {
		if isResumeDesignResultAllowed(row) {
			filtered = append(filtered, row)
		}
	}
	result.Results = filtered
	result.Count = len(filtered)
	return result
}

func isResumeDesignResultAllowed(row map[string]string) bool {
	labels := []string{
		row["Style Category"],
		row["Pattern Name"],
		row["Product Type"],
		row["Best Chart Type"],
	}
	joined := strings.ToLower(strings.Join(labels, " "))
	for _, term := range resumeDesignBlockedResultTerms {
		if strings.Contains(joined, term) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// get_draft
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) getDraft(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok {
		return "", errors.New("draft_id not found in context")
	}

	var draft models.Draft
	if err := e.db.WithContext(ctx).Select("html_content").First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}

	selector, _ := params["selector"].(string)
	if selector == "" {
		return draft.HTMLContent, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(draft.HTMLContent))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}
	html, err := doc.Find(selector).Html()
	if err != nil {
		return "", fmt.Errorf("extract selector: %w", err)
	}
	return html, nil
}

// ---------------------------------------------------------------------------
// apply_edits
// ---------------------------------------------------------------------------

type editOp struct {
	OldString   string `json:"old_string"`
	NewString   string `json:"new_string"`
	Description string `json:"description"`
}

func (e *AgentToolExecutor) applyEdits(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok {
		return "", errors.New("draft_id not found in context")
	}

	// Parse ops from params
	opsRaw, ok := params["ops"]
	if !ok {
		return "", errors.New("missing required parameter: ops")
	}

	opsSlice, ok := opsRaw.([]interface{})
	if !ok {
		return "", fmt.Errorf("ops must be an array, got %T", opsRaw)
	}

	var ops []editOp
	for i, opRaw := range opsSlice {
		opMap, ok := opRaw.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("ops[%d] must be an object", i)
		}
		oldStr, ok := opMap["old_string"].(string)
		if !ok {
			return "", fmt.Errorf("ops[%d].old_string must be a string", i)
		}
		newStr, ok := opMap["new_string"].(string)
		if !ok {
			return "", fmt.Errorf("ops[%d].new_string must be a string", i)
		}
		if err := validateResumeEditFragment(newStr); err != nil {
			return "", fmt.Errorf("ops[%d].new_string violates resume design constraints: %w", i, err)
		}
		desc, _ := opMap["description"].(string)
		ops = append(ops, editOp{OldString: oldStr, NewString: newStr, Description: desc})
	}

	if len(ops) == 0 {
		return "", errors.New("ops must not be empty")
	}

	// Use a closure variable to capture the result from within the transaction.
	var result string
	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Get current draft
		var draft models.Draft
		if err := tx.Select("id, html_content, current_edit_sequence").First(&draft, draftID).Error; err != nil {
			return fmt.Errorf("get draft: %w", err)
		}

		html := draft.HTMLContent

		// 2. Ensure base snapshot exists (sequence 0)
		var baseCount int64
		tx.Model(&models.DraftEdit{}).Where("draft_id = ? AND sequence = 0", draftID).Count(&baseCount)
		if baseCount == 0 {
			baseEdit := models.DraftEdit{
				DraftID:      draftID,
				Sequence:     0,
				OpType:       "base_snapshot",
				HtmlSnapshot: html,
			}
			if err := tx.Create(&baseEdit).Error; err != nil {
				return fmt.Errorf("create base snapshot: %w", err)
			}
		}

		// 3. Validate all ops first (dry run)
		for _, op := range ops {
			if !strings.Contains(html, op.OldString) {
				return fmt.Errorf("old_string not found: %q", op.OldString)
			}
		}

		// 4. Apply all ops for real
		nextSeq := draft.CurrentEditSequence + 1
		applied := 0
		for _, op := range ops {
			html = strings.ReplaceAll(html, op.OldString, op.NewString)

			// 5. Record each op as DraftEdit with HtmlSnapshot
			edit := models.DraftEdit{
				DraftID:      draftID,
				Sequence:     nextSeq,
				OpType:       "replace",
				OldString:    op.OldString,
				NewString:    op.NewString,
				Description:  op.Description,
				HtmlSnapshot: html,
			}
			if err := tx.Create(&edit).Error; err != nil {
				return fmt.Errorf("record edit: %w", err)
			}
			nextSeq++
			applied++
		}

		// 6. Update draft's HTMLContent and CurrentEditSequence
		if err := tx.Model(&models.Draft{}).Where("id = ?", draftID).Updates(map[string]interface{}{
			"html_content":          html,
			"current_edit_sequence": nextSeq - 1,
		}).Error; err != nil {
			return fmt.Errorf("update draft: %w", err)
		}

		// 7. Build result
		resultData := map[string]interface{}{
			"applied":      applied,
			"new_sequence": nextSeq - 1,
		}
		b, err := json.Marshal(resultData)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		result = string(b)
		return nil
	})

	return result, err
}

// ---------------------------------------------------------------------------
// search_assets
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) searchAssets(ctx context.Context, params map[string]interface{}) (string, error) {
	projectID, ok := ctx.Value(projectIDKey).(uint)
	if !ok {
		return "", errors.New("project_id not found in context")
	}

	query := e.db.WithContext(ctx)
	query = query.Model(&models.Asset{}).Where("project_id = ?", projectID)

	// Optional type filter
	if typeVal, ok := params["type"].(string); ok && typeVal != "" {
		query = query.Where("type = ?", typeVal)
	}

	// Optional keyword filter (search in content)
	if keyword, ok := params["query"].(string); ok && keyword != "" {
		query = query.Where("content ILIKE ?", "%"+keyword+"%")
	}

	// Limit
	limit := 5
	if _, ok := params["limit"]; ok {
		if n, err := getIntParam(params, "limit"); err == nil && n > 0 {
			limit = n
		}
	}
	query = query.Limit(limit)

	var assets []models.Asset
	if err := query.Find(&assets).Error; err != nil {
		return "", fmt.Errorf("search assets: %w", err)
	}

	type assetResult struct {
		ID        uint   `json:"id"`
		Type      string `json:"type"`
		Label     string `json:"label,omitempty"`
		Content   string `json:"content,omitempty"`
		CreatedAt string `json:"created_at"`
	}

	results := make([]assetResult, 0, len(assets))
	for _, a := range assets {
		ar := assetResult{
			ID:        a.ID,
			Type:      a.Type,
			CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if a.Label != nil {
			ar.Label = *a.Label
		}
		if a.Content != nil {
			content := *a.Content
			if len(content) > 2000 {
				content = content[:2000] + "...(truncated)"
			}
			ar.Content = content
		}
		results = append(results, ar)
	}

	resultData := map[string]interface{}{
		"results": results,
	}
	b, err := json.Marshal(resultData)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// search_skills
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) searchSkills(ctx context.Context, params map[string]interface{}) (string, error) {
	if e.skillLoader == nil {
		return `{"skills":[],"message":"技能库未加载"}`, nil
	}

	keyword, _ := params["keyword"].(string)
	category, _ := params["category"].(string)
	limit := 3
	if _, ok := params["limit"]; ok {
		if n, err := getIntParam(params, "limit"); err == nil && n > 0 {
			limit = n
		}
	}

	results := e.skillLoader.Search(keyword, category, limit)

	resp := map[string]interface{}{
		"skills": results,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("marshal search results: %w", err)
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// resume edit guardrails
// ---------------------------------------------------------------------------

var resumeEditRejectPatterns = []struct {
	Pattern string
	Reason  string
}{
	{"linear-gradient(", "简历禁止复杂渐变背景"},
	{"radial-gradient(", "简历禁止复杂渐变背景"},
	{"conic-gradient(", "简历禁止复杂渐变背景"},
	{"backdrop-filter:", "简历禁止玻璃拟态模糊效果"},
	{"-webkit-backdrop-filter:", "简历禁止玻璃拟态模糊效果"},
	{"filter:blur(", "简历禁止模糊滤镜"},
	{"@keyframes", "简历禁止动画"},
	{"animation:", "简历禁止动画"},
	{"text-shadow:", "简历禁止发光或阴影文字"},
	{"box-shadow:", "简历禁止厚重卡片阴影"},
	{"position:fixed", "简历禁止固定定位布局"},
	{"position:absolute", "简历禁止绝对定位布局"},
	{"height:100vh", "简历禁止网页视口高度布局"},
	{"min-height:100vh", "简历禁止网页视口高度布局"},
}

func validateResumeEditFragment(fragment string) error {
	compact := strings.ToLower(fragment)
	compact = strings.ReplaceAll(compact, " ", "")
	compact = strings.ReplaceAll(compact, "\n", "")
	compact = strings.ReplaceAll(compact, "\t", "")
	compact = strings.ReplaceAll(compact, "\r", "")

	for _, item := range resumeEditRejectPatterns {
		if strings.Contains(compact, item.Pattern) {
			return errors.New(item.Reason)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getIntParam extracts an integer from params, supporting float64 (from JSON
// unmarshaling), int, int64, and json.Number types.
func getIntParam(params map[string]interface{}, name string) (int, error) {
	v, ok := params[name]
	if !ok {
		return 0, fmt.Errorf("missing required parameter: %s", name)
	}

	switch val := v.(type) {
	case float64:
		return int(val), nil
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case json.Number:
		n, err := strconv.Atoi(val.String())
		if err != nil {
			return 0, fmt.Errorf("invalid integer parameter %s: %w", name, err)
		}
		return n, nil
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("invalid integer parameter %s: %w", name, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("parameter %s must be a number, got %T", name, v)
	}
}
