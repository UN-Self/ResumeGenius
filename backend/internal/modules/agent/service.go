package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrDraftNotFound   = errors.New("draft not found")
	ErrModelTimeout    = errors.New("model call timeout")
	ErrModelFormat     = errors.New("model returned invalid format")
	ErrMaxIterations   = errors.New("max tool-calling iterations exceeded")
)

// ---------------------------------------------------------------------------
// SessionService
// ---------------------------------------------------------------------------

type SessionService struct {
	db *gorm.DB
}

func NewSessionService(db *gorm.DB) *SessionService {
	return &SessionService{db: db}
}

func (s *SessionService) Create(draftID uint) (*models.AISession, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, fmt.Errorf("check draft: %w", err)
	}

	session := models.AISession{DraftID: draftID, ProjectID: &draft.ProjectID}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &session, nil
}

func (s *SessionService) GetByID(sessionID uint) (*models.AISession, error) {
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &session, nil
}

func (s *SessionService) ListByDraftID(draftID uint) ([]models.AISession, error) {
	var sessions []models.AISession
	if err := s.db.Where("draft_id = ?", draftID).Order("created_at DESC").Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return sessions, nil
}

func (s *SessionService) Delete(sessionID uint) error {
	if err := s.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{}).Error; err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	result := s.db.Delete(&models.AISession{}, sessionID)
	if result.Error != nil {
		return fmt.Errorf("delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *SessionService) SaveMessage(sessionID uint, role, content string) error {
	return s.db.Create(&models.AIMessage{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}).Error
}

func (s *SessionService) GetHistory(sessionID uint) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	if err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	return messages, nil
}

// ---------------------------------------------------------------------------
// ChatService
// ---------------------------------------------------------------------------

// systemPromptV2 is the fixed system prompt — no dynamic suffix.
const systemPromptV2 = `你是简历编辑专家。你可以像编辑代码一样精确编辑简历 HTML。

## 核心工具
- get_draft: 读取当前简历 HTML（可选 CSS selector 指定范围）
- apply_edits: 提交搜索替换操作修改简历（old_string 必须精确匹配）
- search_assets: 搜索用户资料库（旧简历、Git 摘要、笔记等）
- search_skills: 搜索简历优化技能库，包括岗位面经和简历设计规范
- search_design_skill: 查询受简历约束的 ui-ux-pro-max 设计参考，只能作为保守简历风格的辅助来源

## 工作流程
1. 先 get_draft 了解当前简历状态
2. 如需用户资料，用 search_assets 搜索
3. 当用户明确目标岗位时，先用 search_skills 搜索岗位关键词，参考面经和简历建议
4. 当用户要求调整视觉、排版、配色、模板、样式，先用 search_skills 搜索 keyword="简历设计 A4 单页" category="design"
5. 只有在仍需字体、配色或极简风格参考时，才调用 search_design_skill；不得把它当网页/产品 UI 灵感库
6. 用 apply_edits 提交精确修改
7. 修改后用 get_draft 验证结果
8. 完成后用自然语言总结修改内容

## 编辑原则
- apply_edits 是搜索替换，不是追加：old_string 必须匹配要被替换的已有内容，new_string 是替换后的内容
- 绝对禁止把整份简历作为 new_string 写入而不匹配任何 old_string，这会导致内容重复
- 每次只修改需要变化的部分，不要重写整个简历
- old_string 必须精确匹配，不匹配则修改会失败
- 失败时读取当前 HTML 找到正确内容后重试
- 保持 HTML 结构完整，确保渲染正确
- 内容简洁专业，突出关键信息

## A4 简历硬约束
- 当前产品编辑的是简历，不是网页、落地页、作品集、仪表盘或海报
- 默认目标是一页 A4：210mm x 297mm；如果内容过多，先压缩文案、字号、行距和间距，不要扩展成多页视觉稿
- 使用常见招聘简历样式：白色或浅色纸面、深色正文、最多一个克制强调色、清晰分区标题、紧凑项目符号、信息密度高但可读
- 正文字号保持在 13-15px 左右，姓名标题不超过 24px，分区标题 14-16px；不要使用超大 hero 字体
- 技能列表必须可换行、可读，禁止做成长串不换行的技能胶囊或大块色卡
- 禁止使用 landing page、hero、dashboard、bento/card grid、glassmorphism、aurora、3D、霓虹、复杂渐变、大面积紫蓝/粉色背景、纹理背景、动画、发光、厚重阴影、过度圆角和装饰图形
- 如果用户说"太花"、"太炫"、"过头"、"不像简历"，优先移除视觉特效，恢复常规专业简历样式

## 回复规范
- 不要使用任何 emoji 或特殊符号装饰

## 技能库（Skills）
你可以使用 search_skills 工具查找简历优化技能。
技能库包含岗位面经、面试官关注点、简历针对性修改建议，以及简历设计规范。

使用时机：当用户明确了目标岗位（如"测试工程师"、"前端开发"、"产品经理"等），
先调用 search_skills 获取该岗位的面经和建议，再基于建议修改简历。
`

// ChatService orchestrates the ReAct reasoning loop.
type ChatService struct {
	db                *gorm.DB
	provider          ProviderAdapter
	toolExecutor      ToolExecutor
	recorder          *ThinkingRecorder
	maxIterations     int
	contextWindowSize int
}

// NewChatService creates a ChatService. contextWindowSize defaults to 128000.
func NewChatService(db *gorm.DB, provider ProviderAdapter, toolExecutor ToolExecutor, maxIterations int) *ChatService {
	if maxIterations <= 0 {
		maxIterations = 3
	}
	windowSize := 128000
	return &ChatService{
		db:                db,
		provider:          provider,
		toolExecutor:      toolExecutor,
		maxIterations:     maxIterations,
		contextWindowSize: windowSize,
	}
}

// ---------------------------------------------------------------------------
// Compaction
// ---------------------------------------------------------------------------

// estimateTokens gives a rough token count using the "4 chars ~ 1 token" heuristic.
func (s *ChatService) estimateTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) + len(m.Name) + len(m.ToolCallID)
	}
	return total / 2
}

// needsCompaction returns true when the estimated token count exceeds 80% of the window.
func (s *ChatService) needsCompaction(messages []Message) bool {
	threshold := int(float64(s.contextWindowSize) * 0.8)
	return s.estimateTokens(messages) > threshold
}

// compactMessages summarizes old messages and retains the most recent 4.
func (s *ChatService) compactMessages(ctx context.Context, messages []models.AIMessage) ([]models.AIMessage, error) {
	if len(messages) <= 4 {
		return messages, nil
	}

	splitIdx := len(messages) - 4
	oldMessages := messages[:splitIdx]
	retained := messages[splitIdx:]

	var sb strings.Builder
	sb.WriteString("请将以下对话历史压缩为简洁摘要，保留：讨论了什么需求、做了哪些修改、当前简历状态。\n\n")
	for _, m := range oldMessages {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}

	var summary strings.Builder
	err := s.provider.StreamChat(ctx, []Message{
		{Role: "system", Content: "你是对话历史摘要助手。压缩为简洁摘要，保留关键信息。"},
		{Role: "user", Content: sb.String()},
	}, func(chunk string) error {
		summary.WriteString(chunk)
		return nil
	})
	if err != nil {
		return messages, fmt.Errorf("compaction failed: %w", err)
	}

	result := []models.AIMessage{{
		Role:      "system",
		Content:   "[对话摘要] " + summary.String(),
		CreatedAt: oldMessages[0].CreatedAt,
	}}
	return append(result, retained...), nil
}

// ---------------------------------------------------------------------------
// StreamChatReAct
// ---------------------------------------------------------------------------

// StreamChatReAct implements the core ReAct reasoning loop.
// It streams thinking events, tool calls, tool results, edit events, and the
// final text response via the sendEvent callback. The loop runs for at most
// s.maxIterations stall rounds (iterations with no tool calls or text).
func (s *ChatService) StreamChatReAct(sessionID uint, userMessage string, sendEvent func(string)) error {
	// 1. Load session, verify existence
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return ErrSessionNotFound
	}

	// 2. Save user message to DB
	s.db.Create(&models.AIMessage{SessionID: sessionID, Role: "user", Content: userMessage})
	startData, _ := json.Marshal(map[string]string{
		"type":    "thinking",
		"content": "已收到请求，正在加载简历上下文...\n",
	})
	sendEvent(string(startData))

	// 3. Load full message history for this session
	history, err := s.loadMessages(sessionID)
	if err != nil {
		return err
	}

	// 4. Compaction check
	apiHistory := make([]Message, len(history))
	for i, m := range history {
		apiHistory[i] = Message{Role: m.Role, Content: m.Content}
	}
	allMsgs := append([]Message{{Role: "system", Content: systemPromptV2}}, apiHistory...)
	if s.needsCompaction(allMsgs) {
		compacted, compactErr := s.compactMessages(context.Background(), history)
		if compactErr == nil {
			history = compacted
			s.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{})
			for _, m := range history {
				m.ID = 0
				m.SessionID = sessionID
				s.db.Create(&m)
			}
		}
	}

	// 5. Build context with draftID and projectID
	ctx := WithDraftID(context.Background(), session.DraftID)
	if session.ProjectID != nil {
		ctx = WithProjectID(ctx, *session.ProjectID)
	}

	// 6. Start ReAct loop
	toolResults := make([]Message, 0)
	var allThinking strings.Builder
	stallCount := 0

	for totalIter := 0; totalIter < s.maxIterations*2+1; totalIter++ {
		// a. Build messages array: system + history + pending tool results
		apiMessages := []Message{{Role: "system", Content: systemPromptV2}}
		for _, m := range history {
			apiMessages = append(apiMessages, Message{Role: m.Role, Content: m.Content})
		}
		apiMessages = append(apiMessages, toolResults...)

		var fullText, thinkingAccum strings.Builder
		var iterationToolCalls []ToolCallRequest
		hadText, hadToolCalls := false, false

		// b. Call provider with streaming callbacks
		stepData, _ := json.Marshal(map[string]string{
			"type":    "thinking",
			"content": fmt.Sprintf("第 %d 步：正在请求模型生成下一步操作...\n", totalIter+1),
		})
		sendEvent(string(stepData))
		err := s.provider.StreamChatReAct(
			ctx,
			apiMessages,
			s.toolExecutor.Tools(),
			// onReasoning: stream thinking chunks
			func(chunk string) error {
				thinkingAccum.WriteString(chunk)
				data, _ := json.Marshal(map[string]string{"type": "thinking", "content": chunk})
				sendEvent(string(data))
				if s.recorder != nil {
					s.recorder.Write(chunk)
				}
				return nil
			},
			// onToolCall: save, execute, and stream tool call + result
			func(call ToolCallRequest) error {
				now := time.Now()
				toolCall := models.AIToolCall{
					SessionID: sessionID,
					ToolName:  call.Name,
					Params:    models.JSONB(call.Params),
					Status:    "running",
					StartedAt: &now,
				}
				s.db.Create(&toolCall)

				// Send tool_call SSE
				callData, _ := json.Marshal(map[string]interface{}{
					"type":   "tool_call",
					"name":   call.Name,
					"params": call.Params,
				})
				sendEvent(string(callData))
				hadToolCalls = true
				iterationToolCalls = append(iterationToolCalls, call)

				// Execute tool (pass context so tool executor can read draftID/projectID)
				result, execErr := s.toolExecutor.Execute(ctx, call.Name, call.Params)
				completedAt := time.Now()

				if execErr != nil {
					// Save failed tool result
					errMsg := execErr.Error()
					s.db.Model(&toolCall).Updates(map[string]interface{}{
						"status":       "failed",
						"error":        errMsg,
						"completed_at": completedAt,
					})
					// Send tool_result (failed)
					failData, _ := json.Marshal(map[string]interface{}{
						"type":   "tool_result",
						"name":   call.Name,
						"status": "failed",
						"result": toolErrorForClient(errMsg),
					})
					sendEvent(string(failData))
					// Add error result to pending messages for next iteration
					toolResults = append(toolResults, Message{
						Role:       "tool",
						Content:    fmt.Sprintf(`{"error":"%s"}`, errMsg),
						ToolCallID: call.ID,
						Name:       call.Name,
					})
				} else {
					// Emit edit SSE event for apply_edits
					if call.Name == "apply_edits" {
						editData, _ := json.Marshal(map[string]interface{}{
							"type":   "edit",
							"name":   call.Name,
							"params": call.Params,
							"result": result,
						})
						sendEvent(string(editData))
					}

					// Save completed tool result
					var parsed map[string]interface{}
					var resultJSON *models.JSONB
					if json.Unmarshal([]byte(result), &parsed) == nil {
						j := models.JSONB(parsed)
						resultJSON = &j
					}
					updates := map[string]interface{}{
						"status":       "completed",
						"completed_at": completedAt,
					}
					if resultJSON != nil {
						updates["result"] = resultJSON
					}
					s.db.Model(&toolCall).Updates(updates)
					// Send tool_result (completed)
					okData, _ := json.Marshal(map[string]interface{}{
						"type":   "tool_result",
						"name":   call.Name,
						"status": "completed",
						"result": toolResultForClient(call.Name, result),
					})
					sendEvent(string(okData))
					// Add result to pending messages for next iteration
					toolResults = append(toolResults, Message{
						Role:       "tool",
						Content:    result,
						ToolCallID: call.ID,
						Name:       call.Name,
					})
				}
				return nil
			},
			// onText: accumulate final text response
			func(chunk string) error {
				hadText = true
				fullText.WriteString(chunk)
				data, _ := json.Marshal(map[string]string{"type": "text", "content": chunk})
				sendEvent(string(data))
				return nil
			},
		)
		if err != nil {
			return err
		}

		// c. If tool calls were made, build assistant tool_calls message and continue loop
		if hadToolCalls {
			allThinking.WriteString(thinkingAccum.String())
			tcMsgs := make([]ToolCallMessage, len(iterationToolCalls))
			for i, tc := range iterationToolCalls {
				argsJSON, _ := json.Marshal(tc.Params)
				tcMsgs[i] = ToolCallMessage{
					ID:   tc.ID,
					Type: "function",
					Function: ToolCallFunction{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
			assistantTC := Message{
				Role:      "assistant",
				Content:   fullText.String(),
				ToolCalls: tcMsgs,
			}
			toolResults = append([]Message{assistantTC}, toolResults...)
			stallCount = 0
			continue
		}

		// d. No tool calls — if had text, save and finish
		if hadText {
			allThinking.WriteString(thinkingAccum.String())
			thinkingStr := allThinking.String()
			var thinkingPtr *string
			if thinkingStr != "" {
				thinkingPtr = &thinkingStr
			}
			s.db.Create(&models.AIMessage{
				SessionID: sessionID,
				Role:      "assistant",
				Content:   fullText.String(),
				Thinking:  thinkingPtr,
			})
			sendEvent(`{"type":"done"}`)
			return nil
		}

		// e. No output at all — count as stall
		stallCount++
		if stallCount >= s.maxIterations {
			return ErrMaxIterations
		}
	}

	return ErrMaxIterations
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *ChatService) loadMessages(sessionID uint) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	if err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	return messages, nil
}

const clientToolResultPreviewRunes = 1600

func toolResultForClient(toolName, result string) string {
	switch toolName {
	case "get_draft":
		return marshalClientToolResult(map[string]interface{}{
			"summary": "已读取当前简历 HTML，日志仅展示截断预览，完整内容已发送给模型。",
			"chars":   len([]rune(result)),
			"preview": truncateRunes(result, clientToolResultPreviewRunes),
		})
	case "apply_edits":
		return result
	default:
		payload := map[string]interface{}{
			"summary": "工具结果预览，完整内容已发送给模型。",
			"chars":   len([]rune(result)),
			"preview": truncateRunes(result, clientToolResultPreviewRunes),
		}
		if len([]rune(result)) <= clientToolResultPreviewRunes {
			var parsed interface{}
			if json.Unmarshal([]byte(result), &parsed) == nil {
				payload["preview"] = parsed
			}
		}
		return marshalClientToolResult(payload)
	}
}

func toolErrorForClient(message string) string {
	return marshalClientToolResult(map[string]interface{}{
		"error": truncateRunes(message, clientToolResultPreviewRunes),
	})
}

func marshalClientToolResult(payload map[string]interface{}) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"error":"failed to marshal tool result preview"}`
	}
	return string(b)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "...(truncated)"
}

// ---------------------------------------------------------------------------
// EditService (Undo/Redo)
// ---------------------------------------------------------------------------

// EditService provides undo/redo operations backed by DraftEdit snapshots.
type EditService struct {
	db *gorm.DB
}

func NewEditService(db *gorm.DB) *EditService {
	return &EditService{db: db}
}

func (s *EditService) Undo(draftID uint) (string, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}
	if draft.CurrentEditSequence <= 0 {
		return "", errors.New("no more edits to undo")
	}

	targetSeq := draft.CurrentEditSequence - 1
	var edit models.DraftEdit
	if err := s.db.Where("draft_id = ? AND sequence = ?", draftID, targetSeq).First(&edit).Error; err != nil {
		return "", fmt.Errorf("get snapshot: %w", err)
	}

	s.db.Model(&draft).Updates(map[string]interface{}{
		"html_content": edit.HtmlSnapshot, "current_edit_sequence": targetSeq,
	})
	return edit.HtmlSnapshot, nil
}

func (s *EditService) Redo(draftID uint) (string, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		return "", fmt.Errorf("get draft: %w", err)
	}

	nextSeq := draft.CurrentEditSequence + 1
	var edit models.DraftEdit
	if err := s.db.Where("draft_id = ? AND sequence = ?", draftID, nextSeq).First(&edit).Error; err != nil {
		return "", errors.New("no more edits to redo")
	}

	s.db.Model(&draft).Updates(map[string]interface{}{
		"html_content": edit.HtmlSnapshot, "current_edit_sequence": nextSeq,
	})
	return edit.HtmlSnapshot, nil
}
