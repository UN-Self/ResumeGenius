package models

import (
	"time"

	"gorm.io/gorm"
)

type SSHKey struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       string         `gorm:"size:36;not null;index" json:"user_id"`
	Alias        string         `gorm:"size:100;not null" json:"alias"`
	EncryptedKey string         `gorm:"type:text;not null" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
