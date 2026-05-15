package intake

import (
	"encoding/base64"
	"fmt"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/crypto"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type SSHKeyService struct {
	db     *gorm.DB
	aesKey []byte
}

func NewSSHKeyService(db *gorm.DB, aesKey string) *SSHKeyService {
	return &SSHKeyService{db: db, aesKey: []byte(aesKey)}
}

type SSHKeyResponse struct {
	ID        uint   `json:"id"`
	Alias     string `json:"alias"`
	CreatedAt string `json:"created_at"`
}

func (s *SSHKeyService) Create(userID, alias, privateKey string) (*SSHKeyResponse, error) {
	encrypted, err := crypto.Encrypt([]byte(privateKey), s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt key: %w", err)
	}

	key := models.SSHKey{
		UserID:       userID,
		Alias:        alias,
		EncryptedKey: base64.StdEncoding.EncodeToString(encrypted),
	}
	if err := s.db.Create(&key).Error; err != nil {
		return nil, fmt.Errorf("create ssh key: %w", err)
	}

	return &SSHKeyResponse{
		ID:        key.ID,
		Alias:     key.Alias,
		CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *SSHKeyService) List(userID string) ([]SSHKeyResponse, error) {
	var keys []models.SSHKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}

	result := make([]SSHKeyResponse, 0, len(keys))
	for _, k := range keys {
		result = append(result, SSHKeyResponse{
			ID:        k.ID,
			Alias:     k.Alias,
			CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return result, nil
}

func (s *SSHKeyService) GetDecryptedKey(userID string, keyID uint) (string, error) {
	var key models.SSHKey
	if err := s.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		return "", fmt.Errorf("ssh key not found: %w", err)
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(key.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("decode key: %w", err)
	}

	decrypted, err := crypto.Decrypt(encryptedBytes, s.aesKey)
	if err != nil {
		return "", fmt.Errorf("decrypt key: %w", err)
	}

	return string(decrypted), nil
}

func (s *SSHKeyService) Delete(userID string, keyID uint) error {
	var count int64
	s.db.Model(&models.Asset{}).
		Joins("JOIN projects ON projects.id = assets.project_id").
		Where("assets.key_id = ? AND projects.user_id = ?", keyID, userID).
		Count(&count)
	if count > 0 {
		return fmt.Errorf("assets still reference this key")
	}

	result := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.SSHKey{})
	if result.Error != nil {
		return fmt.Errorf("delete ssh key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ssh key not found")
	}
	return nil
}
