package agent

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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
	db     *gorm.DB
	apiURL string
	apiKey string
	model  string
}

func NewChatService(db *gorm.DB) *ChatService {
	return &ChatService{
		db:     db,
		apiURL: os.Getenv("AI_API_URL"),
		apiKey: os.Getenv("AI_API_KEY"),
		model:  envOrDefault("AI_MODEL", "default"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (s *ChatService) StreamChat(sessionID uint, userMessage string, sendEvent func(string)) error {
	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return ErrSessionNotFound
	}

	s.db.Create(&models.AIMessage{SessionID: sessionID, Role: "user", Content: userMessage})

	if os.Getenv("USE_MOCK") == "true" {
		return s.mockStream(sessionID, sendEvent)
	}
	return s.callAI(sessionID, sendEvent)
}

func (s *ChatService) mockStream(sessionID uint, sendEvent func(string)) error {
	sendEvent(`{"type":"text","content":"好的，我来帮你优化简历。"}`)
	sendEvent(`{"type":"text","content":"\n<!--RESUME_HTML_START-->\n<html><body><h1>Mock优化简历</h1><p>这是AI生成的优化版本</p></body></html>\n<!--RESUME_HTML_END-->\n"}`)
	sendEvent(`{"type":"done"}`)

	s.db.Create(&models.AIMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   `好的，我来帮你优化简历。<!--RESUME_HTML_START--><html><body>Mock</body></html><!--RESUME_HTML_END-->`,
	})
	return nil
}

func (s *ChatService) callAI(sessionID uint, sendEvent func(string)) error {
	var messages []models.AIMessage
	if err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return fmt.Errorf("load messages: %w", err)
	}

	var session models.AISession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	var draft models.Draft
	if err := s.db.First(&draft, session.DraftID).Error; err != nil {
		return fmt.Errorf("load draft: %w", err)
	}

	var apiMessages []map[string]string
	apiMessages = append(apiMessages, map[string]string{
		"role": "system",
		"content": `你是简历优化助手。根据用户的要求修改简历内容。
当需要修改简历时，在回复中用 <!--RESUME_HTML_START--> 和 <!--RESUME_HTML_END--> 包裹完整的简历 HTML。

当前简历 HTML：
` + draft.HTMLContent,
	})
	for _, msg := range messages {
		apiMessages = append(apiMessages, map[string]string{"role": msg.Role, "content": msg.Content})
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":       s.model,
		"messages":    apiMessages,
		"temperature": 0.7,
		"stream":      true,
	})

	req, _ := http.NewRequest("POST", s.apiURL, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return fmt.Errorf("%w: %w", ErrModelTimeout, err)
		}
		return fmt.Errorf("model call failed: %w", err)
	}
	defer resp.Body.Close()

	return s.processSSEStream(resp.Body, sessionID, sendEvent)
}

func (s *ChatService) processSSEStream(body io.Reader, sessionID uint, sendEvent func(string)) error {
	scanner := bufio.NewScanner(body)
	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}
		fullContent.WriteString(content)

		event, _ := json.Marshal(map[string]string{"type": "text", "content": content})
		sendEvent(string(event))
	}

	s.db.Create(&models.AIMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   fullContent.String(),
	})

	sendEvent(`{"type":"done"}`)
	return scanner.Err()
}
