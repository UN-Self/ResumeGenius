package workbench

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func buildTestDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "postgres"),
		envOrDefault("DB_PASSWORD", "postgres"),
		envOrDefault("DB_NAME", "resume_genius"),
	)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustOpenTestDB opens a test database connection using environment variables
// or defaults. It migrates the schema to ensure tables exist.
func mustOpenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := buildTestDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}

	// Auto-migrate to ensure tables exist
	if err := db.AutoMigrate(
		&models.Project{},
		&models.Asset{},
		&models.Draft{},
		&models.Version{},
		&models.AISession{},
		&models.AIMessage{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// rollbackTestDB begins a transaction and returns a cleanup function
// that rolls back the transaction. Use with defer to ensure test isolation.
//
// Usage:
//
//	db := mustOpenTestDB(t)
//	tx := rollbackTestDB(t, db)
//	// Use tx instead of db for all operations
func rollbackTestDB(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()

	// Clean up any existing data before the test
	db.Exec("TRUNCATE TABLE ai_messages, ai_sessions, versions, drafts, assets, projects CASCADE")

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}

	t.Cleanup(func() {
		// Try to rollback, ignore error if already committed/rolled back
		_ = tx.Rollback().Error
	})

	return tx
}
