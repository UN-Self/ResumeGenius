package agent

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "172.29.26.38"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "45432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "unself"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "unself"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "resume_genius"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}

	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	tx.AutoMigrate(&models.Project{}, &models.Draft{}, &models.AISession{}, &models.AIMessage{})

	return tx
}
