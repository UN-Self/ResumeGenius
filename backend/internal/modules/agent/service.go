package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

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

// ChatService orchestrates the ReAct reasoning loop.
type ChatService struct {
	db                *gorm.DB
	provider          ProviderAdapter
	toolExecutor      ToolExecutor
	recorder          *ThinkingRecorder
	maxIterations     int
	contextWindowSize int
	skillLoader       *SkillLoader
}

// NewChatService creates a ChatService. contextWindowSize defaults to 128000.
func NewChatService(db *gorm.DB, provider ProviderAdapter, toolExecutor ToolExecutor, maxIterations int, skillLoader *SkillLoader) *ChatService {
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
		skillLoader:       skillLoader,
	}
}

// ---------------------------------------------------------------------------
// Compaction
// ---------------------------------------------------------------------------

// estimateTokens gives a rough token count using the "4 chars ~ 1 token" heuristic.
func (s *ChatService) estimateTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += utf8.RuneCountInString(m.Content) + utf8.RuneCountInString(m.Name) + utf8.RuneCountInString(m.ToolCallID)
	}
	return total * 2 / 3
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

	// Structured summary prompt
	var sb strings.Builder
	sb.WriteString("请将以下对话压缩为结构化摘要。使用以下格式：\n\n")
	sb.WriteString("[对话摘要]\n")
	sb.WriteString("- 用户意图：（用户最初想要做什么）\n")
	sb.WriteString("- 已完成：（已经完成了哪些修改）\n")
	sb.WriteString("- 关键结论：（重要的决策或发现）\n")
	sb.WriteString("- 待继续：（还有什么未完成的工作）\n\n")
	sb.WriteString("对话内容：\n")
	for _, m := range oldMessages {
		content := m.Content
		runes := []rune(content)
		if len(runes) > 500 {
			content = string(runes[:500]) + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n", m.Role, content))
	}

	var summary strings.Builder
	err := s.provider.StreamChat(ctx, []Message{
		{Role: "system", Content: "你是对话历史摘要助手。按指定格式压缩对话。"},
		{Role: "user", Content: sb.String()},
	}, func(chunk string) error {
		summary.WriteString(chunk)
		return nil
	})
	if err != nil {
		// Fallback: keep original messages, log warning
		log.Printf("[agent:compact] WARN 压缩失败，保留原始消息: %v", err)
		return messages, nil
	}

	result := []models.AIMessage{{
		Role:      "system",
		Content:   "[对话摘要]\n" + summary.String(),
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
func (s *ChatService) StreamChatReAct(ctx context.Context, sessionID uint, userMessage string, sendEvent func(string) error) error {
	// 1. Load session, verify existence
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return ErrSessionNotFound
	}
	debugLog("service", "收到请求，session=%d, draft=%d", sessionID, session.DraftID)

	logger := slog.Default().With("sessionID", sessionID, "draftID", session.DraftID)
	logger.Info("react_loop_start", "messageLen", len(userMessage))

	// 2. Save user message to DB
	s.db.Create(&models.AIMessage{SessionID: sessionID, Role: "user", Content: userMessage})
	debugLog("service", "用户消息长度 %d 字符", len(userMessage))
	startData, _ := json.Marshal(map[string]string{
		"type":    "thinking",
		"content": "已收到请求，正在加载简历上下文...\n",
	})
	if err := sendEvent(string(startData)); err != nil {
		return fmt.Errorf("send start event: %w", err)
	}

	// 3. Load full message history for this session
	history, err := s.loadMessages(sessionID)
	if err != nil {
		return err
	}

	// 4. Build system prompt first, then use it for compaction check
	sections := DefaultPromptSections("", "")
	augmentedPrompt := BuildSystemPrompt(sections)

	apiHistory := make([]Message, len(history))
	for i, m := range history {
		apiHistory[i] = Message{Role: m.Role, Content: m.Content}
	}
	allMsgs := append([]Message{{Role: "system", Content: augmentedPrompt}}, apiHistory...)
	debugLog("service", "历史消息 %d 条，token 估算 %d", len(history), s.estimateTokens(allMsgs))
	if s.needsCompaction(allMsgs) {
		compacted, compactErr := s.compactMessages(context.Background(), history)
		debugLog("service", "压缩触发，token 估算 %d，压缩前 %d 条消息", s.estimateTokens(allMsgs), len(history))
		if compactErr == nil {
			history = compacted
			s.db.Where("session_id = ?", sessionID).Delete(&models.AIMessage{})
			for _, m := range history {
				m.ID = 0
				m.SessionID = sessionID
				s.db.Create(&m)
			}
			debugLog("service", "压缩完成，压缩后 %d 条消息", len(compacted))
		} else {
			debugLog("service", "压缩失败，使用原始消息: %v", compactErr)
		}
	}

	// 5. Build context with draftID and projectID
	ctx = WithDraftID(ctx, session.DraftID)
	if session.ProjectID != nil {
		ctx = WithProjectID(ctx, *session.ProjectID)
	}

	// 5b. Pre-load project assets into system prompt so AI cannot ignore them
	assetInfo := ""
	if session.ProjectID != nil {
		assetInfo = s.preloadAssets(*session.ProjectID)
	}
	skillListing := ""
	if s.skillLoader != nil {
		skillListing = s.skillLoader.BuildSkillListing()
	}
	sections = DefaultPromptSections(assetInfo, skillListing)
	augmentedPrompt = BuildSystemPrompt(sections)
	debugLog("service", "资源预加载完成，system prompt 长度 %d 字符", len(augmentedPrompt))

	// 6. Start ReAct loop
	loopStart := time.Now()
	toolResults := make([]Message, 0)
	var allThinking strings.Builder
	stallCount := 0
	searchOnlyCount := 0

	for totalIter := 0; totalIter < s.maxIterations*2+1; totalIter++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			debugLog("service", "context cancelled, exiting loop")
			s.toolExecutor.ClearSessionState(sessionID)
			return ctx.Err()
		default:
		}

		// a. Build messages array: system + history + pending tool results
		apiMessages := []Message{{Role: "system", Content: augmentedPrompt}}
		for _, m := range history {
			apiMessages = append(apiMessages, Message{Role: m.Role, Content: m.Content})
		}
		apiMessages = append(apiMessages, toolResults...)

		var fullText, thinkingAccum strings.Builder
		var iterationToolCalls []ToolCallRequest
		var iterToolResults []Message
		hadText, hadToolCalls := false, false

		// b. Call provider with streaming callbacks
		stepData, _ := json.Marshal(map[string]string{
			"type":    "thinking",
			"content": fmt.Sprintf("第 %d 步：正在请求模型生成下一步操作...\n", totalIter+1),
		})
		if err := sendEvent(string(stepData)); err != nil {
			return fmt.Errorf("send step event: %w", err)
		}
		debugLog("service", "第 %d 轮迭代，消息数 %d，token 估算 %d", totalIter+1, len(apiMessages), s.estimateTokens(apiMessages))
		log.Printf("agent: iteration %d calling model with %d messages and %d tools", totalIter, len(apiMessages), len(s.toolExecutor.Tools(ctx)))
		err := s.provider.StreamChatReAct(
			ctx,
			apiMessages,
			s.toolExecutor.Tools(ctx),
			// onReasoning: stream thinking chunks
			func(chunk string) error {
				thinkingAccum.WriteString(chunk)
				data, _ := json.Marshal(map[string]string{"type": "thinking", "content": chunk})
				if err := sendEvent(string(data)); err != nil {
					return fmt.Errorf("send thinking event: %w", err)
				}
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
				if err := sendEvent(string(callData)); err != nil {
					return fmt.Errorf("send tool_call event: %w", err)
				}
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
					if err := sendEvent(string(failData)); err != nil {
						return fmt.Errorf("send tool_result (failed) event: %w", err)
					}
					// Add error result to pending messages for next iteration
					iterToolResults = append(iterToolResults, Message{
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
						if err := sendEvent(string(editData)); err != nil {
							return fmt.Errorf("send edit event: %w", err)
						}
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
					if err := sendEvent(string(okData)); err != nil {
						return fmt.Errorf("send tool_result (completed) event: %w", err)
					}
					// Add result to pending messages for next iteration
					iterToolResults = append(iterToolResults, Message{
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
				if err := sendEvent(string(data)); err != nil {
					return fmt.Errorf("send text event: %w", err)
				}
				return nil
			},
		)
		if err != nil {
			log.Printf("agent: iteration %d model call failed: %v", totalIter, err)
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
				Role:             "assistant",
				Content:          fullText.String(),
				ReasoningContent: thinkingAccum.String(),
				ToolCalls:        tcMsgs,
			}
			toolResults = append([]Message{assistantTC}, iterToolResults...)
			stallCount = 0

			// Track search-only iterations with progressive, escalating reminders.
			hasApply := false
			for _, tc := range iterationToolCalls {
				if tc.Name == "apply_edits" {
					hasApply = true
					break
				}
			}
			if hasApply {
				searchOnlyCount = 0
			} else {
				searchOnlyCount++
				reminder := ""
				remaining := (s.maxIterations*2+1) - totalIter
				switch {
				case remaining <= 2:
					reminder = "[系统指令] 最后机会。必须立刻调用 apply_edits，否则任务失败。"
				case searchOnlyCount >= 4:
					reminder = "[系统指令] 禁止再调用 get_draft。必须立刻调用 apply_edits。"
				case searchOnlyCount == 3:
					reminder = "[系统提醒] 停止搜索，立即调用 apply_edits 编辑简历。"
				case searchOnlyCount == 2:
					reminder = "[系统提醒] 你已读取了简历结构，现在应该开始编辑了。"
				}
				if reminder != "" {
					toolResults = append(toolResults, Message{Role: "system", Content: reminder})
					debugLog("service", "搜索过多提醒触发，连续 %d 轮未执行 apply_edits", searchOnlyCount)
					debugLogFull("service", "提醒消息内容", reminder)
				}
			}
			continue
		}

		// d. No tool calls — if had text, save and finish
		if hadText {
			allThinking.WriteString(thinkingAccum.String())
			debugLog("service", "模型文本输出，长度 %d 字符，内容: %s", fullText.Len(), truncateDebug(fullText.String(), 500))
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
			debugLog("service", "保存助手回复，长度 %d 字符", len(fullText.String()))
			debugLog("service", "循环结束，共 %d 轮，总耗时 %v", totalIter+1, time.Since(loopStart))
			s.toolExecutor.ClearSessionState(sessionID)
			if err := sendEvent(`{"type":"done"}`); err != nil {
				return fmt.Errorf("send done event: %w", err)
			}
			return nil
		}

		// e. No output at all — count as stall
		stallCount++
		debugLog("service", "stall 保护触发，连续 %d 轮无输出", stallCount)
		if stallCount >= s.maxIterations {
			return ErrMaxIterations
		}
	}

	debugLog("service", "迭代汇总: 总轮次 %d，总耗时 %v", s.maxIterations*2+1, time.Since(loopStart))
	logger.Warn("react_loop_max_iterations", "maxIterations", s.maxIterations*2+1, "duration", time.Since(loopStart))
	s.toolExecutor.ClearSessionState(sessionID)
	return ErrMaxIterations
}

// preloadAssets queries the project's non-deleted, non-folder assets and returns
// a system-prompt appendix listing available files and containing the actual
// content of the first contentful asset so the AI cannot ignore it.
func (s *ChatService) preloadAssets(projectID uint) string {
	var assets []models.Asset
	if err := s.db.Where("project_id = ? AND type != ?", projectID, "folder").
		Order("created_at DESC").Limit(20).Find(&assets).Error; err != nil || len(assets) == 0 {
		return "\n## 当前项目状态\n用户尚未上传任何文件。如果用户要求写简历但没有提供资料，请在第一次回复中提醒用户上传旧简历、Git 仓库链接或笔记补充说明。"
	}

	typeCount := make(map[string]int)
	var labels []string
	for _, a := range assets {
		typeCount[a.Type]++
		l := "(未命名)"
		if a.Label != nil && *a.Label != "" {
			l = *a.Label
		}
		labels = append(labels, l)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n## 用户已上传 %d 个文件\n", len(assets)))
	for t, n := range typeCount {
		sb.WriteString(fmt.Sprintf("- %s：%d 个\n", t, n))
	}
	sb.WriteString("文件列表：" + strings.Join(labels, "、") + "\n")
	sb.WriteString("所有简历信息必须从这些文件中提取。找不到的信息请列出缺失项提醒用户补充。禁止凭空编造。\n")

	// Include the first asset's content directly in the system prompt
	for _, a := range assets {
		if a.Content == nil || len(strings.TrimSpace(*a.Content)) == 0 {
			continue
		}
		label := a.Type
		if a.Label != nil && *a.Label != "" {
			label = *a.Label
		}
		sb.WriteString(fmt.Sprintf("\n**以下是「%s」的解析内容：**\n```\n%s\n```\n", label, *a.Content))
		break
	}

	return sb.String()
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
	return truncateWithNotice(value, limit, "...(truncated)")
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
