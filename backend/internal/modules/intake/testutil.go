package intake

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func SetupTestDB(t *testing.T) *gorm.DB {
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
		&models.AISession{},
		&models.AIMessage{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return tx
}

type failingDeleteStorage struct {
	deleteErr error
	saved     map[string][]byte
	saveSeq   int
}

func newFailingDeleteStorage(deleteErr error) *failingDeleteStorage {
	return &failingDeleteStorage{
		deleteErr: deleteErr,
		saved:     map[string][]byte{},
	}
}

func (s *failingDeleteStorage) Save(projectID uint, filename string, data []byte) (string, error) {
	s.saveSeq++
	key := fmt.Sprintf("%d/test-%d-%s", projectID, s.saveSeq, filename)
	s.saved[key] = append([]byte(nil), data...)
	return key, nil
}

func (s *failingDeleteStorage) Delete(key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.saved, key)
	return nil
}

func (s *failingDeleteStorage) Exists(key string) bool {
	_, ok := s.saved[key]
	return ok
}

func (s *failingDeleteStorage) Resolve(key string) (string, error) {
	return key, nil
}
