package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

type User struct {
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	Username     string    `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Project struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         string    `gorm:"size:36;not null;index" json:"user_id"`
	Title          string    `gorm:"size:200;not null" json:"title"`
	Status         string    `gorm:"size:20;not null;default:'active'" json:"status"`
	CurrentDraftID *uint     `json:"current_draft_id"`
	CurrentDraft   *Draft    `gorm:"foreignKey:CurrentDraftID" json:"current_draft,omitempty"`
	Assets         []Asset   `gorm:"foreignKey:ProjectID" json:"assets,omitempty"`
	Drafts         []Draft   `gorm:"foreignKey:ProjectID" json:"drafts,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Asset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index" json:"project_id"`
	Type      string    `gorm:"size:50;not null" json:"type"`
	URI       *string   `gorm:"type:text" json:"uri,omitempty"`
	Content   *string   `gorm:"type:text" json:"content,omitempty"`
	Label     *string   `gorm:"size:100" json:"label,omitempty"`
	Metadata  JSONB     `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Draft struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	ProjectID   uint        `gorm:"not null;index" json:"project_id"`
	HTMLContent string      `gorm:"type:text;not null" json:"html_content"`
	Project     Project     `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Versions    []Version   `gorm:"foreignKey:DraftID" json:"versions,omitempty"`
	AISessions  []AISession `gorm:"foreignKey:DraftID" json:"ai_sessions,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Version struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DraftID      uint      `gorm:"not null;index" json:"draft_id"`
	HTMLSnapshot string    `gorm:"type:text;not null" json:"html_snapshot"`
	Label        *string   `gorm:"size:200" json:"label,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AISession struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	DraftID   uint        `gorm:"not null;index" json:"draft_id"`
	ProjectID *uint       `gorm:"index" json:"project_id"`
	Draft     Draft       `gorm:"foreignKey:DraftID" json:"draft,omitempty"`
	Status    string      `gorm:"size:20;not null;default:'active'" json:"status"`
	Messages  []AIMessage `gorm:"foreignKey:SessionID" json:"messages,omitempty"`
	ToolCalls []AIToolCall `gorm:"foreignKey:SessionID" json:"tool_calls,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (s *AISession) BeforeDelete(tx *gorm.DB) error {
	return tx.Where("session_id = ?", s.ID).Delete(&AIToolCall{}).Error
}

type AIMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SessionID  uint      `gorm:"not null;index" json:"session_id"`
	Session    AISession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	Role       string    `gorm:"size:20;not null" json:"role"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	Thinking   *string   `gorm:"type:text" json:"thinking,omitempty"`
	ToolCallID *uint     `gorm:"index" json:"tool_call_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AIToolCall struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	SessionID   uint       `gorm:"not null;index" json:"session_id"`
	ToolName    string     `gorm:"size:100;not null" json:"tool_name"`
	Params      JSONB      `gorm:"type:jsonb;not null" json:"params"`
	Result      *JSONB     `gorm:"type:jsonb" json:"result,omitempty"`
	Status      string     `gorm:"size:20;not null;default:'pending'" json:"status"`
	Error       *string    `gorm:"type:text" json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
