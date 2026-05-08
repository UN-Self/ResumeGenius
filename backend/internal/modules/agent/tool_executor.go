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
			Description: "搜索简历优化技能库（面经）。根据用户目标岗位的关键词（如'测试工程师'、'QA'）或分类查找匹配的面经和简历修改建议。不传参数返回全部技能摘要。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type":        "string",
						"description": "搜索关键词，按岗位名或技能名匹配，如'测试工程师'、'QA'、'自动化测试'",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "技能分类名，如'test'、'tech'、'management'",
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
			Description: "查询 ui-ux-pro-max 设计知识库，获取风格、配色、字体、图表、UX、技术栈建议，用于优化简历视觉表达。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":  map[string]interface{}{"type": "string", "description": "设计需求，例如 modern engineer resume layout 或 简历 技能矩阵 配色"},
					"domain": map[string]interface{}{"type": "string", "description": "可选：style | prompt | color | chart | landing | product | ux | typography"},
					"stack":  map[string]interface{}{"type": "string", "description": "可选：react | nextjs | vue | html-tailwind | svelte | swiftui | react-native | flutter"},
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

	result, err := designskill.SearchSkill(query, domain, stack, limit)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
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
