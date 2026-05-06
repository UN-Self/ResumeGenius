package render

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

var (
	ErrDraftNotFound   = errors.New("draft not found")
	ErrVersionNotFound = errors.New("version not found")
)

// RollbackResult contains the outcome of a version rollback operation.
type RollbackResult struct {
	DraftID             uint   `json:"draft_id"`
	HTMLContent         string `json:"-"`
	UpdatedAt           string `json:"updated_at"`
	NewVersionID        uint   `json:"new_version_id"`
	NewVersionLabel     string `json:"new_version_label"`
	NewVersionCreatedAt string `json:"new_version_created_at"`
}

// VersionService provides version CRUD and rollback operations.
type VersionService struct {
	db *gorm.DB
}

// NewVersionService creates a new VersionService.
func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

// ListByDraft returns all versions for the given draft, ordered by created_at DESC.
func (s *VersionService) ListByDraft(draftID uint) ([]models.Version, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}

	var versions []models.Version
	err := s.db.Where("draft_id = ?", draftID).
		Order("created_at DESC").
		Find(&versions).Error
	return versions, err
}

// GetByID returns a single version by ID, scoped to the given draft.
func (s *VersionService) GetByID(draftID, versionID uint) (*models.Version, error) {
	var version models.Version
	err := s.db.Where("id = ? AND draft_id = ?", versionID, draftID).First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	return &version, nil
}

// Create snapshots the current draft HTML and creates a new version record.
// If label is empty, it defaults to "手动保存".
func (s *VersionService) Create(draftID uint, label string) (*models.Version, error) {
	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}

	if label == "" {
		label = "手动保存"
	}

	ver := models.Version{
		DraftID:      draftID,
		HTMLSnapshot: draft.HTMLContent,
		Label:        &label,
	}

	if err := s.db.Create(&ver).Error; err != nil {
		return nil, err
	}

	return &ver, nil
}

// Rollback restores the draft's HTML to the target version's snapshot.
// It creates an auto-snapshot of the current HTML before overwriting.
func (s *VersionService) Rollback(draftID, versionID uint) (*RollbackResult, error) {
	var result RollbackResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.rollbackInTx(tx, draftID, versionID, &result)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// rollbackInTx performs the actual rollback logic inside a transaction.
func (s *VersionService) rollbackInTx(tx *gorm.DB, draftID, versionID uint, result *RollbackResult) error {
	// Validate the version exists AND belongs to the given draft.
	var version models.Version
	if err := tx.Where("id = ? AND draft_id = ?", versionID, draftID).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVersionNotFound
		}
		return err
	}

	// Fetch the current draft state.
	var draft models.Draft
	if err := tx.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDraftNotFound
		}
		return err
	}

	// Auto-snapshot the current HTML before overwriting.
	autoLabel := fmt.Sprintf("回退到版本 %d", versionID)
	autoVersion := models.Version{
		DraftID:      draftID,
		HTMLSnapshot: draft.HTMLContent,
		Label:        &autoLabel,
	}
	if err := tx.Create(&autoVersion).Error; err != nil {
		return err
	}

	// Restore the target version's HTML.
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err := tx.Model(&draft).Updates(map[string]interface{}{
		"html_content": version.HTMLSnapshot,
		"updated_at":   now,
	}).Error; err != nil {
		return err
	}

	result.DraftID = draftID
	result.HTMLContent = version.HTMLSnapshot
	result.UpdatedAt = now
	result.NewVersionID = autoVersion.ID
	result.NewVersionLabel = autoLabel
	result.NewVersionCreatedAt = autoVersion.CreatedAt.Format("2006-01-02T15:04:05Z")

	return nil
}

// UpdateDraftHTML is a package-level helper for tests to change a draft's HTML content.
// It is NOT a method on VersionService.
func UpdateDraftHTML(db *gorm.DB, draftID uint, html string) {
	db.Model(&models.Draft{}).Where("id = ?", draftID).
		Update("html_content", html)
}
