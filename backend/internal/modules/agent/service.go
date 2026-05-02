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

	session := models.AISession{DraftID: draftID}
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

func (s *SessionService) GetDraftContent(draftID uint) (string, error) {
	var draft models.Draft
	if err := s.db.Select("html_content").First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrDraftNotFound
		}
		return "", fmt.Errorf("get draft content: %w", err)
	}
	return draft.HTMLContent, nil
}

// --- ChatService ---

type ChatService struct {
	db            *gorm.DB
	provider      ProviderAdapter
	toolExecutor  ToolExecutor
	recorder      *ThinkingRecorder
	maxIterations int
}

func NewChatService(db *gorm.DB, provider ProviderAdapter, toolExecutor ToolExecutor, maxIterations int) *ChatService {
	if maxIterations <= 0 {
		maxIterations = 3
	}
	return &ChatService{
		db:            db,
		provider:      provider,
		toolExecutor:  toolExecutor,
		maxIterations: maxIterations,
	}
}

func buildSystemPrompt(draftHTML string) string {
	return "你是简历优化助手。根据用户的要求修改简历内容。\n当需要修改简历时，在回复中用 <!--RESUME_HTML_START--> 和 <!--RESUME_HTML_END--> 包裹完整的简历 HTML。\n\n当前简历 HTML：\n" + draftHTML
}

// systemPromptReAct is the system prompt used by the ReAct loop.
var systemPromptReAct = `你是一个专业的简历助手。你的任务是根据用户提供的资料和要求，生成一份完整的、可直接渲染的HTML简历。

## 工作流程
1. 如果用户提到了项目/文件，首先调用 get_project_assets 获取资料内容
2. 如果需要查看当前草稿，调用 get_draft 获取最新的简历 HTML
3. 分析资料内容，按照用户要求的格式和内容生成完整的简历 HTML
4. 生成后调用 save_draft 保存到草稿
5. 完成后，报告用户你做了哪些修改

## 轮次限制（极其重要）
你最多只有 **3 轮** 思考和工具调用机会：
- 第 1-2 轮：获取资料、分析内容、构建简历结构
- 第 3 轮：**无论如何必须产出第一版完整简历 HTML**，即使信息不完整
如果第 3 轮结束时你还没有调用 save_draft 保存简历，本次对话将失败。
不要追求完美——先用已有信息生成一版可用的简历，用户可以后续让你修改。
信息不足时，用合理的推断填充，并在回复中说明哪些部分是推断的。

## 输出格式
- 生成的 HTML 必须是完整的、独立的 HTML 文档（含 <!DOCTYPE html>、CSS 样式）
- 页面尺寸为 A4（210mm × 297mm），使用 @page { size: A4; margin: 0; }
- 使用语义化 HTML 标签（header、section、h1-h3、ul/li）
- CSS 内联在 <style> 标签中，不引用外部资源
- 字体：font-family: 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif
- 简历内容应简洁、专业，突出关键信息
- 仅在用户明确要求时才创建版本快照或导出 PDF

## 重要规则
- 生成 HTML 前必须先获取资料（get_project_assets）或查看当前草稿（get_draft）
- 生成 HTML 后必须调用 save_draft 保存
- 不要编造用户没有提供的信息（信息不足时合理推断并标注）
- 如果资料不足以生成完整简历，在第 3 轮直接生成最佳可用版本
`

func (s *ChatService) StreamChat(sessionID uint, userMessage string, sendEvent func(string)) error {
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return ErrSessionNotFound
	}

	s.db.Create(&models.AIMessage{SessionID: sessionID, Role: "user", Content: userMessage})

	var draft models.Draft
	if err := s.db.First(&draft, session.DraftID).Error; err != nil {
		return fmt.Errorf("load draft: %w", err)
	}

	messages, err := s.loadMessages(sessionID)
	if err != nil {
		return err
	}

	apiMessages := []Message{
		{Role: "system", Content: buildSystemPrompt(draft.HTMLContent)},
	}
	for _, m := range messages {
		apiMessages = append(apiMessages, Message{Role: m.Role, Content: m.Content})
	}

	var fullContent strings.Builder
	sendChunk := func(chunk string) error {
		fullContent.WriteString(chunk)
		event, _ := json.Marshal(map[string]string{"type": "text", "content": chunk})
		sendEvent(string(event))
		return nil
	}

	if err := s.provider.StreamChat(context.Background(), apiMessages, sendChunk); err != nil {
		return err
	}

	s.db.Create(&models.AIMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   fullContent.String(),
	})

	sendEvent(`{"type":"done"}`)
	return nil
}

// StreamChatReAct implements the core ReAct reasoning loop.
// It streams thinking events, tool calls, tool results, and the final text response
// via the sendEvent callback. The loop runs for at most s.maxIterations rounds.
func (s *ChatService) StreamChatReAct(sessionID uint, userMessage string, sendEvent func(string)) error {
	// 1. Load session, verify existence
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return ErrSessionNotFound
	}

	// 2. Save user message to DB
	if err := s.db.Create(&models.AIMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   userMessage,
	}).Error; err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	// 3. Load full message history for this session
	history, err := s.loadMessages(sessionID)
	if err != nil {
		return err
	}

	// 4. Start ReAct loop
	toolResults := make([]Message, 0)

	for iteration := 0; iteration < s.maxIterations; iteration++ {
		// a. Build messages array: system + history + pending tool results
		sysPrompt := systemPromptReAct + fmt.Sprintf("\n\n## 当前会话\n- draft_id: %d\n- project_id: %d\n请在所有工具调用中使用这些 ID。", session.DraftID, session.ProjectID)
			apiMessages := []Message{{Role: "system", Content: sysPrompt}}
		for _, m := range history {
			apiMessages = append(apiMessages, Message{Role: m.Role, Content: m.Content})
		}
		apiMessages = append(apiMessages, toolResults...)

		var fullText strings.Builder
		var thinkingAccum strings.Builder
		hadText := false

		// b. Call provider with streaming callbacks
		err := s.provider.StreamChatReAct(
			context.Background(),
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
				if err := s.db.Create(&toolCall).Error; err != nil {
					return fmt.Errorf("save tool call: %w", err)
				}

				// Send tool_call SSE
				callData, _ := json.Marshal(map[string]interface{}{
					"type":   "tool_call",
					"name":   call.Name,
					"params": call.Params,
				})
				sendEvent(string(callData))

				// Execute tool
				result, execErr := s.toolExecutor.Execute(context.Background(), call.Name, call.Params)

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
					failData, _ := json.Marshal(map[string]string{
						"type": "tool_result", "name": call.Name, "status": "failed",
					})
					sendEvent(string(failData))
					// Add error result to pending messages for next iteration
					toolResults = append(toolResults, Message{
						Role: "tool", Content: fmt.Sprintf(`{"error":"%s"}`, execErr.Error()),
					})
				} else {
					// Save completed tool result
					var parsed map[string]interface{}
					var resultJSON *models.JSONB
					if err := json.Unmarshal([]byte(result), &parsed); err == nil {
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
					okData, _ := json.Marshal(map[string]string{
						"type": "tool_result", "name": call.Name, "status": "completed",
					})
					sendEvent(string(okData))
					// Add result to pending messages for next iteration
					toolResults = append(toolResults, Message{
						Role: "tool", Content: result,
					})
				}
				return nil
			},
			// onText: accumulate final text response
			func(chunk string) error {
				hadText = true
				fullText.WriteString(chunk)
				textData, _ := json.Marshal(map[string]string{"type": "text", "content": chunk})
				sendEvent(string(textData))
				return nil
			},
		)
		if err != nil {
			return err
		}

		// c. If onText was called (AI produced final text response)
		if hadText {
			thinkingStr := thinkingAccum.String()
			var thinkingPtr *string
			if thinkingStr != "" {
				thinkingPtr = &thinkingStr
			}
			if err := s.db.Create(&models.AIMessage{
				SessionID: sessionID,
				Role:      "assistant",
				Content:   fullText.String(),
				Thinking:  thinkingPtr,
			}).Error; err != nil {
				return fmt.Errorf("save assistant message: %w", err)
			}
			sendEvent(`{"type":"done"}`)
			return nil
		}

		// d. If only tool_calls were made (no final text), continue loop
		//    toolResults already contain the executed tool results
	}

	// 5. Loop exceeded maxIterations
	return ErrMaxIterations
}

func (s *ChatService) loadMessages(sessionID uint) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	if err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	return messages, nil
}
