package intake

import (
	"errors"
	"fmt"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type ProjectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) Create(userID, title string) (*models.Project, error) {
	proj := models.Project{
		UserID: userID,
		Title:  title,
		Status: "active",
	}
	if err := s.db.Create(&proj).Error; err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &proj, nil
}

func (s *ProjectService) List(userID string) ([]models.Project, error) {
	var projects []models.Project
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (s *ProjectService) GetByID(userID string, id uint) (*models.Project, error) {
	var proj models.Project
	if err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&proj).Error; err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &proj, nil
}

func (s *ProjectService) Delete(userID string, id uint) error {
	result := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Project{})
	if result.Error != nil {
		return fmt.Errorf("delete project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("project not found")
	}
	return nil
}
