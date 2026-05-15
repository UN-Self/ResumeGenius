package auth

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.PointsRecord{}); err != nil {
		t.Fatalf("migrate auth tables: %v", err)
	}
	return db
}

func newAuthServiceForTest(db *gorm.DB) *Service {
	return NewService(db, &EmailService{devMode: true})
}

func createAuthUser(t *testing.T, db *gorm.DB, user models.User, password string) models.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user.PasswordHash = string(hash)
	if user.Plan == "" {
		user.Plan = "free"
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func strPtr(s string) *string {
	return &s
}

func TestLoginRequiresVerifiedEmail(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthServiceForTest(db)

	createAuthUser(t, db, models.User{
		ID:            "verified-user",
		Username:      "verified",
		Email:         strPtr("verified@example.com"),
		EmailVerified: true,
	}, "secret123")
	createAuthUser(t, db, models.User{
		ID:            "unverified-user",
		Username:      "unverified",
		Email:         strPtr("unverified@example.com"),
		EmailVerified: false,
	}, "secret123")
	createAuthUser(t, db, models.User{
		ID:       "legacy-user",
		Username: "legacy",
	}, "secret123")

	user, err := svc.Login("verified", "secret123")
	if err != nil {
		t.Fatalf("verified user login failed: %v", err)
	}
	if user.ID != "verified-user" {
		t.Fatalf("expected verified user, got %s", user.ID)
	}

	if _, err := svc.Login("unverified", "secret123"); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
	if _, err := svc.Login("legacy", "secret123"); !errors.Is(err, ErrAccountNeedsEmailRegistration) {
		t.Fatalf("expected ErrAccountNeedsEmailRegistration, got %v", err)
	}
}

func TestRegisterCreatesUnverifiedUserAndPointsRecord(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthServiceForTest(db)

	user, err := svc.Register("alice", "secret123", "Alice@Example.com")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.Email == nil || *user.Email != "alice@example.com" {
		t.Fatalf("expected normalized email, got %#v", user.Email)
	}
	if user.EmailVerified {
		t.Fatal("new user should be unverified")
	}
	if user.VerificationCode == "" || user.CodeExpiry == nil {
		t.Fatal("expected verification code and expiry")
	}
	if user.Points != 100 || user.Plan != "free" {
		t.Fatalf("expected free plan with 100 points, got plan=%s points=%d", user.Plan, user.Points)
	}

	var record models.PointsRecord
	if err := db.Where("user_id = ? AND type = ?", user.ID, "register_bonus").First(&record).Error; err != nil {
		t.Fatalf("registration bonus record missing: %v", err)
	}
	if record.Amount != 100 || record.Balance != 100 {
		t.Fatalf("unexpected points record: amount=%d balance=%d", record.Amount, record.Balance)
	}
}

func TestRegisterRejectsVerifiedEmailAndUsernameConflicts(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthServiceForTest(db)

	createAuthUser(t, db, models.User{
		ID:            "existing-email",
		Username:      "taken_email",
		Email:         strPtr("taken@example.com"),
		EmailVerified: true,
	}, "secret123")
	createAuthUser(t, db, models.User{
		ID:            "existing-username",
		Username:      "taken_user",
		Email:         strPtr("other@example.com"),
		EmailVerified: true,
	}, "secret123")

	if _, err := svc.Register("new_user", "secret123", "taken@example.com"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
	if _, err := svc.Register("taken_user", "secret123", "new@example.com"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestSendVerificationCodeRefreshesCodeAndExpiry(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthServiceForTest(db)

	oldExpiry := time.Now().Add(-time.Hour)
	user := createAuthUser(t, db, models.User{
		ID:               "unverified-refresh",
		Username:         "refresh",
		Email:            strPtr("refresh@example.com"),
		EmailVerified:    false,
		VerificationCode: "111111",
		CodeExpiry:       &oldExpiry,
	}, "secret123")

	code, err := svc.SendVerificationCode("refresh@example.com")
	if err != nil {
		t.Fatalf("send code failed: %v", err)
	}
	if code == "" {
		t.Fatalf("expected refreshed code, got %q", code)
	}

	var updated models.User
	if err := db.First(&updated, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if updated.VerificationCode != code {
		t.Fatalf("expected stored code %q, got %q", code, updated.VerificationCode)
	}
	if updated.CodeExpiry == nil || !updated.CodeExpiry.After(time.Now()) {
		t.Fatalf("expected future expiry, got %v", updated.CodeExpiry)
	}
}

func TestVerifyEmailValidatesCodeAndExpiry(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthServiceForTest(db)

	expiredAt := time.Now().Add(-time.Minute)
	createAuthUser(t, db, models.User{
		ID:               "expired-code-user",
		Username:         "expired_code",
		Email:            strPtr("expired@example.com"),
		EmailVerified:    false,
		VerificationCode: "123456",
		CodeExpiry:       &expiredAt,
	}, "secret123")
	if _, err := svc.VerifyEmail("expired@example.com", "123456"); !errors.Is(err, ErrVerificationCodeExpired) {
		t.Fatalf("expected ErrVerificationCodeExpired, got %v", err)
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	user := createAuthUser(t, db, models.User{
		ID:               "verify-code-user",
		Username:         "verify_code",
		Email:            strPtr("verify@example.com"),
		EmailVerified:    false,
		VerificationCode: "654321",
		CodeExpiry:       &expiresAt,
	}, "secret123")
	if _, err := svc.VerifyEmail("verify@example.com", "000000"); !errors.Is(err, ErrInvalidVerificationCode) {
		t.Fatalf("expected ErrInvalidVerificationCode, got %v", err)
	}

	verified, err := svc.VerifyEmail("verify@example.com", "654321")
	if err != nil {
		t.Fatalf("verify email failed: %v", err)
	}
	if !verified.EmailVerified || verified.VerificationCode != "" || verified.CodeExpiry != nil {
		t.Fatalf("expected verified user with cleared code, got verified=%v code=%q expiry=%v", verified.EmailVerified, verified.VerificationCode, verified.CodeExpiry)
	}

	var stored models.User
	if err := db.First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("query stored user: %v", err)
	}
	if !stored.EmailVerified || stored.VerificationCode != "" || stored.CodeExpiry != nil {
		t.Fatalf("expected stored verified user with cleared code, got verified=%v code=%q expiry=%v", stored.EmailVerified, stored.VerificationCode, stored.CodeExpiry)
	}
}

func TestToUserRespOnlyIncludesDevCodeInDevMode(t *testing.T) {
	user := &models.User{
		ID:               "dev-code-user",
		Username:         "dev_code",
		Email:            strPtr("dev@example.com"),
		EmailVerified:    false,
		VerificationCode: "123456",
		Plan:             "free",
	}

	if resp := toUserResp(user, true); resp.DevCode != "123456" {
		t.Fatalf("expected dev code in dev mode, got %q", resp.DevCode)
	}
	if resp := toUserResp(user, false); resp.DevCode != "" {
		t.Fatalf("expected no dev code in production mode, got %q", resp.DevCode)
	}
}
