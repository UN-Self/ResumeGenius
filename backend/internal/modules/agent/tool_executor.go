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

// HTMLRepository abstracts draft HTML operations for testability.
type HTMLRepository interface {
	GetDraft(ctx context.Context, draftID uint) (string, error)
	GetDraftStructure(ctx context.Context, draftID uint) (string, error)
	SearchInDraft(ctx context.Context, draftID uint, query string) (string, error)
	GetDraftSection(ctx context.Context, draftID uint, selector string) (string, error)
	ApplyEdits(ctx context.Context, draftID uint, ops []EditOp) (*ApplyEditsResult, error)
}

// AssetRepository abstracts asset search operations.
type AssetRepository interface {
	SearchAssets(ctx context.Context, projectID uint, query, assetType string, limit int) ([]AssetResult, error)
}

// EditOp represents a single edit operation.
type EditOp struct {
	OldString   string `json:"old_string"`
	NewString   string `json:"new_string"`
	Description string `json:"description,omitempty"`
}

// ApplyEditsResult is the result of applying edits.
type ApplyEditsResult struct {
	Applied     int `json:"applied"`
	Failed      int `json:"failed"`
	NewSequence int `json:"new_sequence"`
}

// AssetResult is a search result from assets.
type AssetResult struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// AgentToolExecutor implements ToolExecutor using database queries.
type AgentToolExecutor struct {
	db               *gorm.DB
	skillLoader      *SkillLoader
	getDraftCallCount sync.Map // sessionID -> int
	cachedTools      []ToolDef
}

// NewAgentToolExecutor creates a new AgentToolExecutor.
func NewAgentToolExecutor(db *gorm.DB, skillLoader *SkillLoader) *AgentToolExecutor {
	e := &AgentToolExecutor{db: db, skillLoader: skillLoader}
	e.cachedTools = e.buildTools()
	return e
}

// Tools returns the AI-callable tool definitions (cached).
func (e *AgentToolExecutor) Tools(_ context.Context) []ToolDef {
	return e.cachedTools
}

func (e *AgentToolExecutor) buildTools() []ToolDef {
	tools := []ToolDef{
		{
			Name:        "get_draft",
			Description: "获取简历 HTML 内容。支持 4 种模式：structure（结构概览，不含文本）、section（按 CSS selector 获取指定区域）、search（搜索包含关键词的片段）、full（完整 HTML）。最多调用 2 次（structure + full），之后必须用 apply_edits 编辑。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"mode": map[string]interface{}{
						"type":        "string",
						"description": "查询模式：structure（结构概览）、section（指定区域）、search（关键词搜索）、full（完整内容）。默认 full。",
						"enum":        []string{"structure", "section", "search", "full"},
					},
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS 选择器，mode=section 时使用。例如 .experience、#education",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "搜索关键词，mode=search 时使用。返回包含关键词的 HTML 片段（±200 字符上下文）。",
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

	if e.skillLoader != nil {
		tools = append(tools, ToolDef{
			Name:        "load_skill",
			Description: "加载技能参考内容。返回技能描述和全部参考文档。调用后按返回的 usage 指引操作。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"skill_name": map[string]interface{}{
						"type":        "string",
						"description": "技能名称，如 'resume-design'、'resume-interview'",
					},
				},
				"required": []string{"skill_name"},
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
	case "load_skill":
		result, err = e.loadSkill(ctx, params)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}

	if err != nil {
		debugLog("tools", "工具 %s 执行失败，耗时 %v: %v", toolName, time.Since(start), err)
	} else {
		debugLog("tools", "工具 %s 执行成功，耗时 %v", toolName, time.Since(start))
	}
	return result, err
}

// ---------------------------------------------------------------------------
// load_skill
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) loadSkill(ctx context.Context, params map[string]interface{}) (string, error) {
	if e.skillLoader == nil {
		return `{"error":"技能库未加载"}`, nil
	}

	skillName, _ := params["skill_name"].(string)
	if skillName == "" {
		return `{"error":"skill_name is required"}`, nil
	}

	debugLog("tools", "加载技能 %s（含全部参考文档）", skillName)

	result, err := e.skillLoader.LoadSkillWithReferences(skillName)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal skill: %w", err)
	}
	return string(b), nil
}

func (e *AgentToolExecutor) ClearSessionState(sessionID uint) {
	e.getDraftCallCount.Delete(sessionID)
}

// ---------------------------------------------------------------------------
// get_draft
// ---------------------------------------------------------------------------

func (e *AgentToolExecutor) getDraft(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, ok := ctx.Value(draftIDKey).(uint)
	if !ok {
		return "", errors.New("draft_id not found in context")
	}

	// Track and limit get_draft calls per session
	sessionID, _ := ctx.Value(sessionIDKey).(uint)
	const maxGetDraftCalls = 2
	if sessionID > 0 {
		countVal, _ := e.getDraftCallCount.LoadOrStore(sessionID, new(int))
		count := countVal.(*int)
		*count++
		if *count > maxGetDraftCalls {
			debugLog("tools", "get_draft 调用被拒绝，session=%d 已调用 %d 次", sessionID, *count)
			return fmt.Sprintf("你已经读取了简历 %d 次，内容没有变化。请直接使用 apply_edits 编辑简历，不要再调用 get_draft。", *count), nil
		}
	}

	var draft models.Draft
	if err := e.db.WithContext(ctx).Select("id", "html_content").First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}

	// Parse mode parameter, default "full" (backward compatible)
	mode := "full"
	if m, ok := params["mode"].(string); ok && m != "" {
		mode = m
	}

	// Backward compatible: selector without mode → section
	selector, _ := params["selector"].(string)
	if selector != "" && mode == "full" {
		mode = "section"
	}

	switch mode {
	case "structure":
		debugLog("tools", "get_draft mode=structure")
		structure := BuildStructureOverview(draft.HTMLContent)
		if structure == "" {
			return "当前简历 HTML 为空，请直接使用 apply_edits 创建完整简历内容。", nil
		}
		return structure, nil

	case "section":
		if selector == "" {
			return "", fmt.Errorf("mode=section 需要 selector 参数")
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(draft.HTMLContent))
		if err != nil {
			return "", fmt.Errorf("parse HTML: %w", err)
		}
		selection := doc.Find(selector)
		if selection.Length() == 0 {
			return "", fmt.Errorf("selector %q 未匹配到任何元素", selector)
		}
		html, err := selection.Html()
		if err != nil {
			return "", fmt.Errorf("extract selector: %w", err)
		}
		debugLog("tools", "get_draft mode=section selector=%s len=%d", selector, len(html))
		return html, nil

	case "search":
		query, _ := params["query"].(string)
		if query == "" {
			return "", fmt.Errorf("mode=search 需要 query 参数")
		}
		debugLog("tools", "get_draft mode=search query=%s", query)
		return searchInDraft(draft.HTMLContent, query), nil

	case "full":
		runes := []rune(draft.HTMLContent)
		if len(runes) == 0 {
			return "当前简历 HTML 为空，请直接使用 apply_edits 创建完整简历内容。", nil
		}
		debugLog("tools", "get_draft mode=full len=%d", len(draft.HTMLContent))
		return draft.HTMLContent, nil

	default:
		return "", fmt.Errorf("未知 mode: %s，支持 structure/section/search/full", mode)
	}
}

// ---------------------------------------------------------------------------
// apply_edits
// ---------------------------------------------------------------------------

// buildEditMatchError constructs a structured error when old_string is not found in HTML.
func buildEditMatchError(oldString, html string) string {
	var b strings.Builder
	b.WriteString("未找到匹配内容\n\n")

	b.WriteString("搜索内容: ")
	b.WriteString(truncateString(oldString, 100))
	b.WriteString("\n\n")

	matchResult, score := FindBestMatchNode(html, oldString)

	b.WriteString("可能原因: ")
	if matchResult != nil && score > 0.3 {
		b.WriteString("缩进/换行不一致或片段跨越标签边界")
	} else if matchResult != nil && score > 0.1 {
		b.WriteString("HTML 已被修改，内容部分匹配")
	} else {
		b.WriteString("HTML 中不存在相似内容，可能已被完全替换")
	}
	b.WriteString("\n\n")

	b.WriteString("附近 HTML 片段:\n")
	if matchResult != nil {
		snippet := TruncateAroundMatch(html, oldString, 200)
		b.WriteString(snippet)
	} else {
		overview := BuildStructureOverview(html)
		if len(overview) > 0 {
			b.WriteString("当前 HTML 结构:\n")
			b.WriteString(truncateString(overview, 500))
		} else {
			b.WriteString(truncateString(html, 500))
		}
	}
	b.WriteString("\n\n")

	b.WriteString("建议: 使用更短的唯一片段重新搜索，确保文本精确匹配（包括空格和换行）。不要重新调用 get_draft，直接用更短的 old_string 重试 apply_edits。")

	return b.String()
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

	var ops []EditOp
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
		ops = append(ops, EditOp{OldString: oldStr, NewString: newStr, Description: desc})
	}

	if len(ops) == 0 {
		return "", errors.New("ops must not be empty")
	}

	debugLog("tools", "apply_edits 开始，共 %d 个操作", len(ops))
	start := time.Now()

	// Step-by-step execution (non-atomic: successful ops preserved on partial failure)
	var resultData struct {
		Applied     int `json:"applied"`
		Failed      int `json:"failed"`
		NewSequence int `json:"new_sequence"`
	}
	var lastErr error

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

		nextSeq := draft.CurrentEditSequence + 1

		// 3. Apply ops step by step
		for i, op := range ops {
			if !strings.Contains(html, op.OldString) {
				lastErr = fmt.Errorf("op #%d 匹配失败:\n%s", i+1, buildEditMatchError(op.OldString, html))
				resultData.Failed++
				continue
			}

			debugLog("tools", "操作 %d/%d: old=%s → new=%s", i+1, len(ops), truncateHTML(op.OldString), truncateHTML(op.NewString))
			html = strings.ReplaceAll(html, op.OldString, op.NewString)

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
			resultData.Applied++
		}

		// 4. Update draft (even with partial failure, preserve successful ops)
		if resultData.Applied > 0 {
			if err := tx.Model(&models.Draft{}).Where("id = ?", draftID).Updates(map[string]interface{}{
				"html_content":          html,
				"current_edit_sequence": nextSeq - 1,
			}).Error; err != nil {
				return fmt.Errorf("update draft: %w", err)
			}
		}

		resultData.NewSequence = nextSeq - 1
		return nil
	})

	if err != nil {
		debugLog("tools", "apply_edits 失败，耗时 %v: %v", time.Since(start), err)
		opsJSON, _ := json.Marshal(ops)
		debugLogFull("tools", "apply_edits 失败 - 完整 ops", string(opsJSON))
		return "", err
	}

	b, _ := json.Marshal(resultData)
	result := string(b)

	if lastErr != nil {
		debugLog("tools", "apply_edits 部分成功 (%d/%d applied, %d failed), 耗时 %v",
			resultData.Applied, len(ops), resultData.Failed, time.Since(start))
		return result, fmt.Errorf("部分成功 (%d/%d applied, %d failed): %w",
			resultData.Applied, len(ops), resultData.Failed, lastErr)
	}

	debugLog("tools", "apply_edits 完成，applied=%d new_sequence=%d, 耗时 %v",
		resultData.Applied, resultData.NewSequence, time.Since(start))
	return result, nil
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
			content = truncateWithNotice(content, 2000, "...(truncated)")
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
