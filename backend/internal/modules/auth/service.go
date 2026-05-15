package auth

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

var (
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrInvalidUsername               = errors.New("invalid username")
	ErrInvalidPassword               = errors.New("invalid password")
	ErrInvalidEmail                  = errors.New("invalid email")
	ErrUsernameTaken                 = errors.New("username already taken")
	ErrEmailTaken                    = errors.New("email already taken")
	ErrEmailNotFound                 = errors.New("email not found")
	ErrEmailAlreadyVerified          = errors.New("email already verified")
	ErrEmailNotVerified              = errors.New("email not verified")
	ErrAccountNeedsEmailRegistration = errors.New("account needs email registration")
	ErrInvalidVerificationCode       = errors.New("invalid verification code")
	ErrVerificationCodeExpired       = errors.New("verification code expired")
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
// Only accounts with a verified email are allowed to log in.
func (s *Service) Login(account, password string) (*models.User, error) {
	account = strings.TrimSpace(account)
	if len(account) < 3 {
		return nil, ErrInvalidUsername
	}
	if len(password) < 6 || len(password) > 128 {
		return nil, ErrInvalidPassword
	}

	var user models.User
	var err error
	if isEmailLogin(account) {
		account = strings.ToLower(account)
		err = s.db.Where("email = ?", account).First(&user).Error
	} else {
		if len(account) > 64 {
			return nil, ErrInvalidUsername
		}
		err = s.db.Where("username = ?", account).First(&user).Error
	}
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("query user: %w", err)
		}
		return nil, ErrInvalidCredentials
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); compareErr != nil {
		return nil, ErrInvalidCredentials
	}
	if user.Email == nil || strings.TrimSpace(*user.Email) == "" {
		return nil, ErrAccountNeedsEmailRegistration
	}
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}
	return &user, nil
}

func isEmailLogin(s string) bool {
	return strings.Contains(s, "@")
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

	var registered models.User
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// Email is the unique identity. Check it first.
		var byEmail models.User
		if err := tx.Where("email = ?", email).First(&byEmail).Error; err == nil {
			// Email already exists.
			if byEmail.EmailVerified {
				return ErrEmailTaken
			}
			if err := ensureUsernameAvailable(tx, username, byEmail.ID); err != nil {
				return err
			}
			// Unverified: overwrite this account.
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("hash password: %w", hashErr)
			}
			code, codeErr := GenerateCode()
			if codeErr != nil {
				return codeErr
			}
			expiry := time.Now().Add(15 * time.Minute)
			byEmail.Username = username
			byEmail.PasswordHash = string(hash)
			byEmail.VerificationCode = code
			byEmail.CodeExpiry = &expiry
			byEmail.EmailVerified = false
			if saveErr := tx.Save(&byEmail).Error; saveErr != nil {
				if mapped := mapUserConstraintError(saveErr); mapped != nil {
					return mapped
				}
				return fmt.Errorf("update user: %w", saveErr)
			}
			registered = byEmail
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("query user by email: %w", err)
		}

		// Email is new. Check username uniqueness (must be globally unique).
		if err := ensureUsernameAvailable(tx, username, ""); err != nil {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		code, err := GenerateCode()
		if err != nil {
			return err
		}
		now := time.Now()
		expiry := now.Add(15 * time.Minute)
		user := models.User{
			ID:               uuid.NewString(),
			Username:         username,
			Email:            &email,
			EmailVerified:    false,
			VerificationCode: code,
			CodeExpiry:       &expiry,
			PasswordHash:     string(hash),
			Plan:             "free",
			Points:           100,
			PlanStartedAt:    &now,
		}
		if createErr := tx.Create(&user).Error; createErr != nil {
			if mapped := mapUserConstraintError(createErr); mapped != nil {
				return mapped
			}
			return fmt.Errorf("create user: %w", createErr)
		}
		if err := tx.Create(&models.PointsRecord{
			UserID:  user.ID,
			Amount:  100,
			Balance: 100,
			Type:    "register_bonus",
			Note:    "首次注册赠送",
		}).Error; err != nil {
			return fmt.Errorf("create registration bonus: %w", err)
		}
		registered = user
		return nil
	}); err != nil {
		return nil, err
	}

	if sendErr := s.email.SendVerificationCode(email, registered.VerificationCode); sendErr != nil {
		return nil, fmt.Errorf("send verification code: %w", sendErr)
	}
	return &registered, nil
}

func ensureUsernameAvailable(db *gorm.DB, username string, exceptID string) error {
	var user models.User
	query := db.Where("username = ?", username)
	if exceptID != "" {
		query = query.Where("id <> ?", exceptID)
	}
	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("query user by username: %w", err)
	}
	return ErrUsernameTaken
}

func mapUserConstraintError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if !errors.Is(err, gorm.ErrDuplicatedKey) &&
		!strings.Contains(msg, "duplicate") &&
		!strings.Contains(msg, "unique") {
		return nil
	}
	if strings.Contains(msg, "email") {
		return ErrEmailTaken
	}
	if strings.Contains(msg, "username") {
		return ErrUsernameTaken
	}
	return nil
}

// SendVerificationCode generates a new code and sends it to the given email.
// Returns the generated code so the caller can expose it in dev mode.
func (s *Service) SendVerificationCode(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrEmailNotFound
		}
		return "", fmt.Errorf("query user: %w", err)
	}
	if user.EmailVerified {
		return "", ErrEmailAlreadyVerified
	}

	code, err := GenerateCode()
	if err != nil {
		return "", err
	}

	expiry := time.Now().Add(15 * time.Minute)
	user.VerificationCode = code
	user.CodeExpiry = &expiry
	if updateErr := s.db.Save(&user).Error; updateErr != nil {
		return "", fmt.Errorf("update code: %w", updateErr)
	}

	if sendErr := s.email.SendVerificationCode(email, code); sendErr != nil {
		return code, fmt.Errorf("send verification code: %w", sendErr)
	}
	return code, nil
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

// UpdateProfile updates the user's display name (nickname).
func (s *Service) UpdateProfile(userID, nickname string) (*models.User, error) {
	nickname = strings.TrimSpace(nickname)
	if len(nickname) < 2 || len(nickname) > 32 {
		return nil, ErrInvalidUsername
	}
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	user.Username = nickname
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return &user, nil
}

// ChangePassword validates the old password and sets a new one.
func (s *Service) ChangePassword(userID, oldPassword, newPassword string) error {
	if len(newPassword) < 6 || len(newPassword) > 128 {
		return ErrInvalidPassword
	}
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("query user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	return s.db.Save(&user).Error
}

// UpdateAvatar compresses and saves an avatar image for the user.
// The image is resized to max 256x256 and saved as JPEG quality 80.
func (s *Service) UpdateAvatar(userID string, reader io.Reader, uploadDir string) (*models.User, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	// Resize to max 256x256 maintaining aspect ratio
	resized := resizeToFit(img, 256, 256)

	// Ensure avatar directory exists
	avatarDir := filepath.Join(uploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return nil, fmt.Errorf("create avatars dir: %w", err)
	}

	// Save as JPEG
	filePath := filepath.Join(avatarDir, userID+".jpg")
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: 80}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("write avatar: %w", err)
	}

	// Update user record
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	avatarURL := "/api/v1/auth/avatar/" + userID
	user.AvatarURL = &avatarURL
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return &user, nil
}

// GetAvatarPath returns the file path to the user's avatar, or empty string.
func (s *Service) GetAvatarPath(userID string, uploadDir string) string {
	filePath := filepath.Join(uploadDir, "avatars", userID+".jpg")
	if _, err := os.Stat(filePath); err == nil {
		return filePath
	}
	return ""
}

// GetPointsRecords returns the recent points transactions for a user.
func (s *Service) GetPointsRecords(userID string, limit int) ([]models.PointsRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var records []models.PointsRecord
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("query points records: %w", err)
	}
	return records, nil
}

type PointsStats struct {
	Balance     int   `json:"balance"`
	MonthUsed   int64 `json:"month_used"`
	TotalEarned int64 `json:"total_earned"`
}

type DailyUsage struct {
	Date   string `json:"date"`
	Used   int64  `json:"used"`
	Earned int64  `json:"earned"`
}

type CategoryUsage struct {
	Type  string `json:"type"`
	Total int64  `json:"total"`
}

type PointsDashboard struct {
	Balance     int             `json:"balance"`
	MonthUsed   int64           `json:"month_used"`
	TotalEarned int64           `json:"total_earned"`
	DailyUsage  []DailyUsage    `json:"daily_usage"`
	Categories  []CategoryUsage `json:"categories"`
}

// GetPointsStats returns aggregated points stats for a user.
func (s *Service) GetPointsStats(userID string) (*PointsStats, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var monthUsed int64
	s.db.Model(&models.PointsRecord{}).
		Where("user_id = ? AND amount < 0 AND created_at >= ?", userID, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&monthUsed)

	var totalEarned int64
	s.db.Model(&models.PointsRecord{}).
		Where("user_id = ? AND amount > 0", userID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalEarned)

	return &PointsStats{
		Balance:     user.Points,
		MonthUsed:   -monthUsed,
		TotalEarned: totalEarned,
	}, nil
}

// GetDashboard returns full dashboard data: stats + daily usage + category breakdown.
func (s *Service) GetDashboard(userID string) (*PointsDashboard, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var monthUsed int64
	s.db.Model(&models.PointsRecord{}).
		Where("user_id = ? AND amount < 0 AND created_at >= ?", userID, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&monthUsed)

	var totalEarned int64
	s.db.Model(&models.PointsRecord{}).
		Where("user_id = ? AND amount > 0", userID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalEarned)

	// Daily usage for last 30 days
	dailyUsage := make([]DailyUsage, 30)
	daysAgo := now.AddDate(0, 0, -29)
	for i := 0; i < 30; i++ {
		day := daysAgo.AddDate(0, 0, i)
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, now.Location())
		dayEnd := dayStart.Add(24 * time.Hour)

		var used int64
		s.db.Model(&models.PointsRecord{}).
			Where("user_id = ? AND amount < 0 AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).
			Select("COALESCE(SUM(amount), 0)").Scan(&used)

		var earned int64
		s.db.Model(&models.PointsRecord{}).
			Where("user_id = ? AND amount > 0 AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).
			Select("COALESCE(SUM(amount), 0)").Scan(&earned)

		dailyUsage[i] = DailyUsage{
			Date:   dayStart.Format("01-02"),
			Used:   -used,
			Earned: earned,
		}
	}

	// Category breakdown (all time)
	var catRows []struct {
		Type  string
		Total int64
	}
	s.db.Model(&models.PointsRecord{}).
		Where("user_id = ? AND amount < 0", userID).
		Select("type, COALESCE(SUM(ABS(amount)), 0) as total").
		Group("type").Order("total DESC").Scan(&catRows)

	categories := make([]CategoryUsage, len(catRows))
	for i, r := range catRows {
		categories[i] = CategoryUsage{Type: r.Type, Total: r.Total}
	}

	return &PointsDashboard{
		Balance:     user.Points,
		MonthUsed:   -monthUsed,
		TotalEarned: totalEarned,
		DailyUsage:  dailyUsage,
		Categories:  categories,
	}, nil
}

func resizeToFit(img image.Image, maxW, maxH int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxW && h <= maxH {
		return img
	}

	// Maintain aspect ratio
	ratio := float64(w) / float64(h)
	var newW, newH int
	if ratio > float64(maxW)/float64(maxH) {
		newW = maxW
		newH = int(float64(maxW) / ratio)
	} else {
		newH = maxH
		newW = int(float64(maxH) * ratio)
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

func isValidEmail(email string) bool {
	return len(email) >= 5 && len(email) <= 254 &&
		strings.Contains(email, "@") &&
		strings.Contains(email, ".")
}
