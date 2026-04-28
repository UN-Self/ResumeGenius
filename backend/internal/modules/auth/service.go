package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) LoginOrRegister(username, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 64 {
		return nil, ErrInvalidUsername
	}
	if len(password) < 6 || len(password) > 128 {
		return nil, ErrInvalidPassword
	}

	var user models.User
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("query user: %w", err)
		}

		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("hash password: %w", hashErr)
		}

		user = models.User{
			ID:           uuid.NewString(),
			Username:     username,
			PasswordHash: string(hash),
		}
		if createErr := s.db.Create(&user).Error; createErr != nil {
			return nil, fmt.Errorf("create user: %w", createErr)
		}
		return &user, nil
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); compareErr != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

func (s *Service) GetByID(id string) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return &user, nil
}
