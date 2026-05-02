package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

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

// AgentToolExecutor implements ToolExecutor using database queries and internal HTTP calls.
type AgentToolExecutor struct {
	db         *gorm.DB
	httpClient *http.Client
	baseURL    string
}

// NewAgentToolExecutor creates a new AgentToolExecutor.
// baseURL should point to the internal API server (e.g. "http://127.0.0.1:8080").
func NewAgentToolExecutor(db *gorm.DB, baseURL string) *AgentToolExecutor {
	return &AgentToolExecutor{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// Tools returns the six AI-callable tool definitions.
func (e *AgentToolExecutor) Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "get_project_assets",
			Description: "Get project's asset list",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "integer",
						"description": "The project ID to get assets from",
					},
				},
				"required": []interface{}{"project_id"},
			},
		},
		{
			Name:        "parse_project_assets",
			Description: "Parse project assets text content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "integer",
						"description": "The project ID to parse assets from",
					},
				},
				"required": []interface{}{"project_id"},
			},
		},
		{
			Name:        "get_draft",
			Description: "Get current draft HTML",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_id": map[string]interface{}{
						"type":        "integer",
						"description": "The draft ID to retrieve",
					},
				},
				"required": []interface{}{"draft_id"},
			},
		},
		{
			Name:        "save_draft",
			Description: "Save/update draft HTML",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_id": map[string]interface{}{
						"type":        "integer",
						"description": "The draft ID to update",
					},
					"html_content": map[string]interface{}{
						"type":        "string",
						"description": "The HTML content to save",
					},
				},
				"required": []interface{}{"draft_id", "html_content"},
			},
		},
		{
			Name:        "create_version",
			Description: "Create version snapshot",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_id": map[string]interface{}{
						"type":        "integer",
						"description": "The draft ID to create version for",
					},
					"label": map[string]interface{}{
						"type":        "string",
						"description": "The version label",
					},
				},
				"required": []interface{}{"draft_id", "label"},
			},
		},
		{
			Name:        "export_pdf",
			Description: "Trigger PDF export",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_id": map[string]interface{}{
						"type":        "integer",
						"description": "The draft ID to export",
					},
					"html_content": map[string]interface{}{
						"type":        "string",
						"description": "The HTML content to export",
					},
				},
				"required": []interface{}{"draft_id", "html_content"},
			},
		},
	}
}

// Execute dispatches to the correct tool implementation based on toolName.
func (e *AgentToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	switch toolName {
	case "get_project_assets":
		return e.getProjectAssets(ctx, params)
	case "parse_project_assets":
		return e.parseProjectAssets(ctx, params)
	case "get_draft":
		return e.getDraft(ctx, params)
	case "save_draft":
		return e.saveDraft(ctx, params)
	case "create_version":
		return e.createVersion(ctx, params)
	case "export_pdf":
		return e.exportPDF(ctx, params)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// getProjectAssets queries all assets belonging to a project.
func (e *AgentToolExecutor) getProjectAssets(ctx context.Context, params map[string]interface{}) (string, error) {
	projectID, err := getIntParam(params, "project_id")
	if err != nil {
		return "", err
	}

	var assets []models.Asset
	if err := e.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&assets).Error; err != nil {
		return "", fmt.Errorf("query assets: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(assets))
	for _, a := range assets {
		result = append(result, map[string]interface{}{
			"id":         a.ID,
			"project_id": a.ProjectID,
			"type":       a.Type,
			"uri":        a.URI,
			"content":    a.Content,
			"label":      a.Label,
			"created_at": a.CreatedAt,
		})
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
}

// parseProjectAssets calls the parsing module to extract text from project assets.
func (e *AgentToolExecutor) parseProjectAssets(ctx context.Context, params map[string]interface{}) (string, error) {
	projectID, err := getIntParam(params, "project_id")
	if err != nil {
		return "", err
	}

	body := map[string]interface{}{
		"project_id": projectID,
	}
	return e.httpPost(ctx, "/api/v1/parsing/parse", body)
}

// getDraft retrieves a draft by ID and returns its HTML content.
func (e *AgentToolExecutor) getDraft(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, err := getIntParam(params, "draft_id")
	if err != nil {
		return "", err
	}

	var draft models.Draft
	if err := e.db.WithContext(ctx).First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("query draft: %w", err)
	}

	result := map[string]interface{}{
		"draft_id":     draft.ID,
		"html_content": draft.HTMLContent,
		"updated_at":   draft.UpdatedAt,
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
}

// saveDraft updates a draft's HTML content in the database.
func (e *AgentToolExecutor) saveDraft(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, err := getIntParam(params, "draft_id")
	if err != nil {
		return "", err
	}

	htmlContent, err := getStringParam(params, "html_content")
	if err != nil {
		return "", err
	}

	result := e.db.WithContext(ctx).Model(&models.Draft{}).Where("id = ?", draftID).Update("html_content", htmlContent)
	if result.Error != nil {
		return "", fmt.Errorf("update draft: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("draft not found: %d", draftID)
	}

	out := map[string]interface{}{
		"draft_id": draftID,
		"status":   "saved",
	}

	b, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
}

// createVersion creates a version snapshot for the given draft via HTTP.
func (e *AgentToolExecutor) createVersion(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, err := getIntParam(params, "draft_id")
	if err != nil {
		return "", err
	}

	label, err := getStringParam(params, "label")
	if err != nil {
		return "", err
	}

	body := map[string]interface{}{
		"label": label,
	}
	return e.httpPost(ctx, fmt.Sprintf("/api/v1/drafts/%d/versions", draftID), body)
}

// exportPDF triggers a PDF export for the given draft via HTTP.
func (e *AgentToolExecutor) exportPDF(ctx context.Context, params map[string]interface{}) (string, error) {
	draftID, err := getIntParam(params, "draft_id")
	if err != nil {
		return "", err
	}

	htmlContent, err := getStringParam(params, "html_content")
	if err != nil {
		return "", err
	}

	body := map[string]interface{}{
		"html_content": htmlContent,
	}
	return e.httpPost(ctx, fmt.Sprintf("/api/v1/drafts/%d/export", draftID), body)
}

// httpPost sends an HTTP POST with a JSON body and extracts the data field from the
// standard API response envelope.
func (e *AgentToolExecutor) httpPost(ctx context.Context, path string, body map[string]interface{}) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Code    int              `json:"code"`
		Data    *json.RawMessage `json:"data"`
		Message string           `json:"message"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if apiResp.Code != 0 {
		return "", fmt.Errorf("api error code %d: %s", apiResp.Code, apiResp.Message)
	}

	if apiResp.Data == nil {
		return "{}", nil
	}

	return string(*apiResp.Data), nil
}

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

// getStringParam extracts a string value from params.
func getStringParam(params map[string]interface{}, name string) (string, error) {
	v, ok := params[name]
	if !ok {
		return "", fmt.Errorf("missing required parameter: %s", name)
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string, got %T", name, v)
	}
	return s, nil
}
