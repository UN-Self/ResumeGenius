package render

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// envOrDefault returns the environment variable value or a fallback default.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// SetupTestDB connects to the database using environment variables, runs
// AutoMigrate inside a transaction, and registers a t.Cleanup rollback so
// every test gets a clean slate.
//
// Required ENV VARS (all have sensible defaults):
//
//	DB_HOST     (default: localhost)
//	DB_PORT     (default: 5432)
//	DB_USER     (default: postgres)
//	DB_PASSWORD (default: postgres)
//	DB_NAME     (default: resume_genius)
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "postgres"),
		envOrDefault("DB_PASSWORD", "postgres"),
		envOrDefault("DB_NAME", "resume_genius"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "failed to connect to database")

	// Wrap everything in a transaction so tests are isolated and rolled back.
	tx := db.Begin()
	require.NoError(t, tx.Error, "failed to begin transaction")

	t.Cleanup(func() {
		tx.Rollback()
	})

	err = tx.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Asset{},
		&models.Draft{},
		&models.Version{},
		&models.AISession{},
		&models.AIMessage{},
	)
	require.NoError(t, err, "failed to run AutoMigrate")

	return tx
}

// seedUser creates a minimal test User row and returns it.
func seedUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()

	user := models.User{
		ID:           uuid.New().String(),
		Username:     "testuser-" + uuid.New().String()[:8],
		PasswordHash: "$2a$10$placeholderhashfortest",
	}
	require.NoError(t, db.Create(&user).Error, "failed to seed user")
	return user
}

// seedDraft creates a User -> Project -> Draft chain and returns the Draft.
// It reuses the seedUser helper internally.
func seedDraft(t *testing.T, db *gorm.DB) models.Draft {
	t.Helper()

	user := seedUser(t, db)

	project := models.Project{
		UserID: user.ID,
		Title:  "Test Project",
		Status: "active",
	}
	require.NoError(t, db.Create(&project).Error, "failed to seed project")

	// Link current draft before creating the draft itself.
	project.CurrentDraftID = nil

	draft := models.Draft{
		ProjectID:   project.ID,
		HTMLContent: "<html><body><h1>Test Resume</h1></body></html>",
	}
	require.NoError(t, db.Create(&draft).Error, "failed to seed draft")

	// Back-fill the project's CurrentDraftID.
	project.CurrentDraftID = &draft.ID
	require.NoError(t, db.Save(&project).Error, "failed to update project current_draft_id")

	return draft
}
