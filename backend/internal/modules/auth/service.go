package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidUsername         = errors.New("invalid username")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrInvalidEmail            = errors.New("invalid email")
	ErrUsernameTaken           = errors.New("username already taken")
	ErrEmailTaken              = errors.New("email already taken")
	ErrEmailNotFound           = errors.New("email not found")
	ErrEmailAlreadyVerified    = errors.New("email already verified")
	ErrEmailNotVerified        = errors.New("email not verified")
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	ErrVerificationCodeExpired = errors.New("verification code expired")
)

type Service struct {
	db    *gorm.DB
	email *EmailService
}

func NewService(db *gorm.DB, email *EmailService) *Service {
	return &Service{db: db, email: email}
}

// Login authenticates a user by username and password.
// It does NOT auto-register — returns ErrInvalidCredentials if user not found.
// Returns ErrEmailNotVerified if the user exists but email is not yet verified
// (only for accounts with a non-empty email — legacy accounts are exempt).
func (s *Service) Login(username, password string) (*models.User, error) {
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
		return nil, ErrInvalidCredentials
	}

	// Legacy accounts (nil email) skip email verification check.
	if user.Email != nil && *user.Email != "" && !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); compareErr != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

// Register creates a new user with EmailVerified=false and sends a verification code.
// If an unverified account with the same username or email already exists,
// it is overwritten (password updated + new code sent) instead of returning an error.
func (s *Service) Register(username, password, email string) (*models.User, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 64 {
		return nil, ErrInvalidUsername
	}
	if len(password) < 6 || len(password) > 128 {
		return nil, ErrInvalidPassword
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}

	// Look for an existing account by username or email.
	var existing models.User
	foundByUser := s.db.Where("username = ?", username).First(&existing)
	foundByEmail := s.db.Where("email = ?", email).First(&existing)

	if foundByUser.Error == nil || foundByEmail.Error == nil {
		// Found an existing record — only allow re-register if unverified.
		if existing.EmailVerified {
			if existing.Username == username {
				return nil, ErrUsernameTaken
			}
			return nil, ErrEmailTaken
		}
		// Unverified: overwrite password and resend code.
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("hash password: %w", hashErr)
		}
		code, codeErr := GenerateCode()
		if codeErr != nil {
			return nil, codeErr
		}
		expiry := time.Now().Add(15 * time.Minute)
		existing.Username = username
		existing.Email = &email
		existing.PasswordHash = string(hash)
		existing.VerificationCode = code
		existing.CodeExpiry = &expiry
		existing.EmailVerified = false
		if saveErr := s.db.Save(&existing).Error; saveErr != nil {
			return nil, fmt.Errorf("update user: %w", saveErr)
		}
		if sendErr := s.email.SendVerificationCode(email, code); sendErr != nil {
			return nil, fmt.Errorf("send verification code: %w", sendErr)
		}
		return &existing, nil
	}

	// No existing account — create new.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	code, err := GenerateCode()
	if err != nil {
		return nil, err
	}
	expiry := time.Now().Add(15 * time.Minute)
	user := models.User{
		ID:               uuid.NewString(),
		Username:         username,
		Email:            &email,
		EmailVerified:    false,
		VerificationCode: code,
		CodeExpiry:       &expiry,
		PasswordHash:     string(hash),
	}
	if createErr := s.db.Create(&user).Error; createErr != nil {
		return nil, fmt.Errorf("create user: %w", createErr)
	}
	if sendErr := s.email.SendVerificationCode(email, code); sendErr != nil {
		return nil, fmt.Errorf("send verification code: %w", sendErr)
	}
	return &user, nil
}

// SendVerificationCode generates a new code and sends it to the given email.
// The user must exist and not already be verified.
func (s *Service) SendVerificationCode(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmailNotFound
		}
		return fmt.Errorf("query user: %w", err)
	}
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	code, err := GenerateCode()
	if err != nil {
		return err
	}

	expiry := time.Now().Add(15 * time.Minute)
	user.VerificationCode = code
	user.CodeExpiry = &expiry
	if updateErr := s.db.Save(&user).Error; updateErr != nil {
		return fmt.Errorf("update code: %w", updateErr)
	}

	if sendErr := s.email.SendVerificationCode(email, code); sendErr != nil {
		return fmt.Errorf("send verification code: %w", sendErr)
	}
	return nil
}

// VerifyEmail validates the verification code and marks the email as verified.
func (s *Service) VerifyEmail(email, code string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmailNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	if user.EmailVerified {
		return nil, ErrEmailAlreadyVerified
	}
	if user.VerificationCode == "" || user.CodeExpiry == nil {
		return nil, ErrInvalidVerificationCode
	}
	if time.Now().After(*user.CodeExpiry) {
		return nil, ErrVerificationCodeExpired
	}
	if user.VerificationCode != code {
		return nil, ErrInvalidVerificationCode
	}

	user.EmailVerified = true
	user.VerificationCode = ""
	user.CodeExpiry = nil
	if updateErr := s.db.Save(&user).Error; updateErr != nil {
		return nil, fmt.Errorf("verify email: %w", updateErr)
	}
	return &user, nil
}

// CheckUsername returns true if the username is available.
func (s *Service) CheckUsername(username string) (bool, error) {
	username = strings.TrimSpace(username)
	var count int64
	if err := s.db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check username: %w", err)
	}
	return count == 0, nil
}

// CheckEmail returns true if the email is available.
func (s *Service) CheckEmail(email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var count int64
	if err := s.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check email: %w", err)
	}
	return count == 0, nil
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

func isValidEmail(email string) bool {
	return len(email) >= 5 && len(email) <= 254 &&
		strings.Contains(email, "@") &&
		strings.Contains(email, ".")
}
