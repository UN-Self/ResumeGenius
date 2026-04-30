package parsing

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	sharedstorage "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

func setupParsingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}

	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	if err := tx.AutoMigrate(
		&models.Project{},
		&models.Asset{},
		&models.Draft{},
		&models.Version{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return tx
}

func newTestStorage(t *testing.T) sharedstorage.FileStorage {
	t.Helper()
	return sharedstorage.NewLocalStorage(t.TempDir())
}
