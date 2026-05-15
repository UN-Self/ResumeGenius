package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/env"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func buildDSN(host, port, user, password, dbname string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

func Connect() *gorm.DB {
	dsn := buildDSN(
		env.DefaultOr("DB_HOST", "localhost"),
		env.DefaultOr("DB_PORT", "5432"),
		env.DefaultOr("DB_USER", "postgres"),
		env.DefaultOr("DB_PASSWORD", "postgres"),
		env.DefaultOr("DB_NAME", "resume_genius"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	return db
}

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Asset{},
		&models.SSHKey{},
		&models.Draft{},
		&models.Version{},
		&models.AISession{},
		&models.AIMessage{},
		&models.AIToolCall{},
		&models.DraftEdit{},
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	log.Println("database migrated successfully")
}
