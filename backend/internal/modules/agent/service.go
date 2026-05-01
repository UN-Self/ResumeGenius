package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrDraftNotFound   = errors.New("draft not found")
	ErrModelTimeout    = errors.New("model call timeout")
	ErrModelFormat     = errors.New("model returned invalid format")
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
	db       *gorm.DB
	provider ProviderAdapter
}

func NewChatService(db *gorm.DB, provider ProviderAdapter) *ChatService {
	return &ChatService{db: db, provider: provider}
}

func buildSystemPrompt(draftHTML string) string {
	return "你是简历优化助手。根据用户的要求修改简历内容。\n当需要修改简历时，在回复中用 <!--RESUME_HTML_START--> 和 <!--RESUME_HTML_END--> 包裹完整的简历 HTML。\n\n当前简历 HTML：\n" + draftHTML
}

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

func (s *ChatService) loadMessages(sessionID uint) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	if err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	return messages, nil
}
