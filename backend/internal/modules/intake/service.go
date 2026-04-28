package intake

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"

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

// --- Error definitions ---

var (
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrFileTooLarge      = errors.New("file size exceeds 20MB limit")
	ErrInvalidGitURL     = errors.New("invalid git repository URL")
	ErrProjectNotFound   = errors.New("project not found")
	ErrAssetNotFound     = errors.New("asset not found")
)

var allowedExtensions = map[string]string{
	".pdf":  "resume_pdf",
	".docx": "resume_docx",
	".png":  "resume_image",
	".jpg":  "resume_image",
	".jpeg": "resume_image",
}

var maxFileSize = 20 * 1024 * 1024

var gitURLPattern = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

// --- AssetService ---

type AssetService struct {
	db      *gorm.DB
	storage FileStorage
}

func NewAssetService(db *gorm.DB, storage FileStorage) *AssetService {
	return &AssetService{db: db, storage: storage}
}

func (s *AssetService) validateProject(userID string, projectID uint) error {
	var count int64
	s.db.Where("id = ? AND user_id = ?", projectID, userID).Model(&models.Project{}).Count(&count)
	if count == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *AssetService) UploadFile(userID string, projectID uint, filename string, data []byte, size int64) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	ext := filepath.Ext(filename)
	assetType, ok := allowedExtensions[ext]
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	if size > int64(maxFileSize) {
		return nil, ErrFileTooLarge
	}

	uri, err := s.storage.Save(projectID, filename, data)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      assetType,
		URI:       &uri,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		// Rollback: delete the file already saved
		_ = s.storage.Delete(uri)
		return nil, fmt.Errorf("create asset record: %w", err)
	}

	return &asset, nil
}

func (s *AssetService) CreateGitRepo(userID string, projectID uint, repoURL string) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	if !gitURLPattern.MatchString(repoURL) {
		return nil, ErrInvalidGitURL
	}
	if _, err := url.Parse(repoURL); err != nil {
		return nil, ErrInvalidGitURL
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      "git_repo",
		URI:       &repoURL,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("create git repo asset: %w", err)
	}

	return &asset, nil
}

func (s *AssetService) CreateNote(userID string, projectID uint, content, label string) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      "note",
		Content:   &content,
		Label:     &label,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("create note asset: %w", err)
	}

	return &asset, nil
}

func (s *AssetService) UpdateNote(userID string, noteID uint, content, label string) (*models.Asset, error) {
	var asset models.Asset
	if err := s.db.First(&asset, noteID).Error; err != nil {
		return nil, ErrAssetNotFound
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return nil, err
	}

	asset.Content = &content
	asset.Label = &label
	if err := s.db.Save(&asset).Error; err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}

	return &asset, nil
}

func (s *AssetService) ListByProject(userID string, projectID uint) ([]models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	var assets []models.Asset
	if err := s.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&assets).Error; err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	return assets, nil
}

func (s *AssetService) DeleteAsset(userID string, assetID uint) error {
	var asset models.Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return ErrAssetNotFound
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return err
	}

	if asset.URI != nil && *asset.URI != "" {
		_ = s.storage.Delete(*asset.URI)
	}

	if err := s.db.Delete(&asset).Error; err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}

	return nil
}

func (s *AssetService) DeleteProjectAssets(userID string, projectID uint) error {
	if err := s.validateProject(userID, projectID); err != nil {
		return err
	}

	var assets []models.Asset
	if err := s.db.Where("project_id = ?", projectID).Find(&assets).Error; err != nil {
		return fmt.Errorf("find project assets: %w", err)
	}

	for _, asset := range assets {
		if asset.URI != nil && *asset.URI != "" {
			_ = s.storage.Delete(*asset.URI)
		}
	}

	if err := s.db.Where("project_id = ?", projectID).Delete(&models.Asset{}).Error; err != nil {
		return fmt.Errorf("delete project assets: %w", err)
	}

	return nil
}
