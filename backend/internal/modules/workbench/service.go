package workbench

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

var (
	// ErrDraftNotFound is returned when a draft doesn't exist.
	ErrDraftNotFound = errors.New("draft not found")
	// ErrHTMLContentEmpty is returned when HTML content is empty.
	ErrHTMLContentEmpty = errors.New("html content empty")
	// ErrProjectNotFound is returned when a project doesn't exist.
	ErrProjectNotFound = errors.New("project not found")
	// ErrProjectHasDraft is returned when a project already has a current draft.
	ErrProjectHasDraft = errors.New("project already has a draft")
)

// DraftService handles business logic for draft operations.
type DraftService struct {
	db *gorm.DB
}

// NewDraftService creates a new DraftService instance.
func NewDraftService(db *gorm.DB) *DraftService {
	return &DraftService{db: db}
}

// GetByID retrieves a draft by its ID.
// Returns ErrDraftNotFound if the draft doesn't exist.
func (s *DraftService) GetByID(id uint) (*models.Draft, error) {
	var draft models.Draft
	err := s.db.First(&draft, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return &draft, nil
}

// Create creates a new draft for a project with empty HTML content.
// Returns ErrProjectNotFound if the project doesn't exist.
// Returns ErrProjectHasDraft if the project already has a current draft.
func (s *DraftService) Create(projectID uint) (*models.Draft, error) {
	// Check if the project exists
	var project models.Project
	err := s.db.First(&project, projectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	// Check if the project already has a current draft
	if project.CurrentDraftID != nil {
		return nil, ErrProjectHasDraft
	}

	// Create the draft and link it to the project atomically
	var draft models.Draft
	err = s.db.Transaction(func(tx *gorm.DB) error {
		draft = models.Draft{
			ProjectID:   projectID,
			HTMLContent: "",
		}
		if err := tx.Create(&draft).Error; err != nil {
			return err
		}
		return tx.Model(&project).Update("current_draft_id", draft.ID).Error
	})
	if err != nil {
		return nil, err
	}

	return &draft, nil
}

// Update updates the HTML content of a draft.
// Returns ErrDraftNotFound if the draft doesn't exist.
// Returns ErrHTMLContentEmpty if the HTML content is empty.
func (s *DraftService) Update(id uint, htmlContent string, createVersion bool, versionLabel string) (*models.Draft, *uint, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return nil, nil, ErrHTMLContentEmpty
	}

	var draft models.Draft
	err := s.db.First(&draft, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrDraftNotFound
		}
		return nil, nil, err
	}

	if err := s.db.Model(&draft).Update("html_content", htmlContent).Error; err != nil {
		return nil, nil, err
	}

	if err := s.db.First(&draft, id).Error; err != nil {
		return nil, nil, err
	}

	var createdVersionID *uint
	if createVersion {
		label := strings.TrimSpace(versionLabel)
		if label == "" {
			label = "手动保存"
		}

		v := models.Version{
			DraftID:      draft.ID,
			HTMLSnapshot: htmlContent,
			Label:        &label,
		}
		if err := s.db.Create(&v).Error; err != nil {
			return nil, nil, err
		}
		createdVersionID = &v.ID
	}

	return &draft, createdVersionID, nil
}

// UpdateMeta updates metadata fields (e.g. page_count) for a draft.
// Returns ErrDraftNotFound if the draft doesn't exist.
func (s *DraftService) UpdateMeta(id uint, pageCount int) error {
	result := s.db.Model(&models.Draft{}).Where("id = ?", id).Update("page_count", pageCount)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDraftNotFound
	}
	return nil
}
