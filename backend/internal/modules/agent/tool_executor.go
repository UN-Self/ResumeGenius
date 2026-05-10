package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// ---------------------------------------------------------------------------
// Context keys
// ---------------------------------------------------------------------------

type contextKey int

const (
	draftIDKey   contextKey = iota
	projectIDKey
	sessionIDKey
)

// WithDraftID returns a context carrying the given draft ID.
func WithDraftID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, draftIDKey, id)
}

// WithProjectID returns a context carrying the given project ID.
func WithProjectID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, projectIDKey, id)
}

// WithSessionID returns a context carrying the given session ID.
func WithSessionID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, sessionIDKey, id)
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
	Tools(ctx context.Context) []ToolDef
	// Execute runs a tool by name with the given parameters and returns the result as a JSON string.
	Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error)
	// ClearSessionState releases any per-session cached state (e.g. loaded skill tracking).
	ClearSessionState(sessionID uint)
}

// AgentToolExecutor implements ToolExecutor using database queries.
type AgentToolExecutor struct {
	db           *gorm.DB
	skillLoader  *SkillLoader
	loadedSkills sync.Map // sessionID -> map[string]bool (loaded skill names)
}

// NewAgentToolExecutor creates a new AgentToolExecutor.
func NewAgentToolExecutor(db *gorm.DB, skillLoader *SkillLoader) *AgentToolExecutor {
	return &AgentToolExecutor{db: db, skillLoader: skillLoader}
}

// Tools returns the AI-callable tool definitions.
func (e *AgentToolExecutor) Tools(ctx context.Context) []ToolDef {
	// 1. Base tools (fixed)
	tools := []ToolDef{
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
	}

	// 2. Skill tools (auto-generated from SkillLoader, no parameters)
	if e.skillLoader != nil {
		for _, skill := range e.skillLoader.Skills() {
			tools = append(tools, ToolDef{
				Name:        skill.Name,
				Description: skill.Description,
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			})
		}
	}

	// 3. Sub-tools (dynamically injected based on loaded skills)
	sessionID, _ := ctx.Value(sessionIDKey).(uint)
	if e.hasLoadedSkills(sessionID) {
		tools = append(tools, ToolDef{
			Name:        "get_skill_reference",
			Description: "获取技能库中指定岗位的面经内容或设计规范。必须先调用对应技能工具。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"skill_name": map[string]interface{}{
						"type":        "string",
						"description": "技能名称，如 'resume-interview'、'resume-design'",
					},
					"reference_name": map[string]interface{}{
						"type":        "string",
						"description": "reference 名称，如 'test-engineer'、'a4-guidelines'",
					},
				},
				"required": []string{"skill_name", "reference_name"},
			},
		})
	}

	return tools
}

// Execute dispatches to the correct tool implementation based on toolName.
func (e *AgentToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	start := time.Now()

	paramsJSON, _ := json.Marshal(params)
	debugLog("tools", "调用工具 %s，参数摘要: %s", toolName, truncateParams(string(paramsJSON)))

	var result string
	var err error

	switch toolName {
	case "get_draft":
		result, err = e.getDraft(ctx, params)
	case "apply_edits":
		result, err = e.applyEdits(ctx, params)
	case "search_assets":
		result, err = e.searchAssets(ctx, params)
	case "get_skill_reference":
		result, err = e.getSkillReference(ctx, params)
	default:
		// Check if it's a skill tool
		if e.skillLoader != nil && e.skillLoader.HasSkill(toolName) {
			result, err = e.executeSkillTool(ctx, toolName)
		} else {
			return "", fmt.Errorf("unknown tool: %s", toolName)
		}
	}

	if err != nil {
		debugLog("tools", "工具 %s 执行失败，耗时 %v: %v", toolName, time.Since(start), err)
	} else {
		debugLog("tools", "工具 %s 执行成功，耗时 %v", toolName, time.Since(start))
	}
	return result, err
}

// ---------------------------------------------------------------------------
// skill tool execution
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) executeSkillTool(ctx context.Context, skillName string) (string, error) {
	if e.skillLoader == nil {
		return `{"error":"技能库未加载"}`, nil
	}

	debugLog("tools", "加载技能 %s", skillName)

	desc, err := e.skillLoader.LoadSkill(skillName)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}

	// Mark skill as loaded for this session.
	if sessionID, ok := ctx.Value(sessionIDKey).(uint); ok {
		e.markSkillLoaded(sessionID, skillName)
	}

	b, err := json.Marshal(desc)
	if err != nil {
		return "", fmt.Errorf("marshal skill descriptor: %w", err)
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// get_skill_reference
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) getSkillReference(ctx context.Context, params map[string]interface{}) (string, error) {
	if e.skillLoader == nil {
		return `{"error":"技能库未加载"}`, nil
	}

	skillName, _ := params["skill_name"].(string)
	refName, _ := params["reference_name"].(string)

	if skillName == "" {
		return `{"error":"skill_name is required"}`, nil
	}
	if refName == "" {
		return `{"error":"reference_name is required"}`, nil
	}

	// Check that the skill has been loaded first (order enforcement).
	if sessionID, ok := ctx.Value(sessionIDKey).(uint); ok {
		if !e.isSkillLoaded(sessionID, skillName) {
			return fmt.Sprintf(`{"error":"skill '%s' not loaded: call '%s' tool first"}`, skillName, skillName), nil
		}
	}

	debugLog("tools", "获取技能参考文档 %s/%s", skillName, refName)

	ref, err := e.skillLoader.GetReference(skillName, refName)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}

	b, err := json.Marshal(ref)
	if err != nil {
		return "", fmt.Errorf("marshal reference: %w", err)
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// session skill tracking helpers
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) markSkillLoaded(sessionID uint, skillName string) {
	val, _ := e.loadedSkills.LoadOrStore(sessionID, &sync.Map{})
	m := val.(*sync.Map)
	m.Store(skillName, true)
}

func (e *AgentToolExecutor) isSkillLoaded(sessionID uint, skillName string) bool {
	val, ok := e.loadedSkills.Load(sessionID)
	if !ok {
		return false
	}
	m := val.(*sync.Map)
	_, loaded := m.Load(skillName)
	return loaded
}

func (e *AgentToolExecutor) ClearSessionState(sessionID uint) {
	e.loadedSkills.Delete(sessionID)
}

func (e *AgentToolExecutor) hasLoadedSkills(sessionID uint) bool {
	val, ok := e.loadedSkills.Load(sessionID)
	if !ok {
		return false
	}
	m := val.(*sync.Map)
	has := false
	m.Range(func(_, _ interface{}) bool {
		has = true
		return false
	})
	return has
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
		debugLog("tools", "get_draft，返回完整 HTML 长度 %d", len(draft.HTMLContent))
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
	debugLog("tools", "get_draft，selector=%s，返回 HTML 长度 %d", selector, len(html))
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

	debugLog("tools", "apply_edits 开始，共 %d 个操作", len(ops))
	start := time.Now()

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
		for i, op := range ops {
			if !strings.Contains(html, op.OldString) {
				oldPreview := truncateDebug(op.OldString, 100)
				htmlPreview := truncateDebug(html, 200)
				debugLog("tools", "操作 %d 验证失败: old_string 未找到: %s", i, truncateHTML(op.OldString))
				debugLogFull("tools", fmt.Sprintf("apply_edits 操作 %d 失败 - 完整 new_string", i), op.NewString)
				return fmt.Errorf("old_string not found in current draft. old_string前100字符: %q | 当前HTML前200字符: %q", oldPreview, htmlPreview)
			}
		}

		// 4. Apply all ops for real
		nextSeq := draft.CurrentEditSequence + 1
		applied := 0
		for i, op := range ops {
			debugLog("tools", "操作 %d/%d: old=%s → new=%s", i+1, len(ops), truncateHTML(op.OldString), truncateHTML(op.NewString))
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

	if err != nil {
		debugLog("tools", "apply_edits 失败，耗时 %v: %v", time.Since(start), err)
		opsJSON, _ := json.Marshal(ops)
		debugLogFull("tools", "apply_edits 失败 - 完整 ops", string(opsJSON))
	} else {
		debugLog("tools", "apply_edits 完成，耗时 %v", time.Since(start))
	}
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
	keyword, _ := params["query"].(string)
	if keyword != "" {
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

	debugLog("tools", "search_assets，查询=%s，结果 %d 条", keyword, len(assets))

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
