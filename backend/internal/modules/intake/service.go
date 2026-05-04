package intake

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
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
	err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&proj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &proj, nil
}

func (s *ProjectService) Delete(userID string, id uint) error {
	// Validate ownership first
	var count int64
	s.db.Where("id = ? AND user_id = ?", id, userID).Model(&models.Project{}).Count(&count)
	if count == 0 {
		return ErrProjectNotFound
	}

	// Cascade delete in dependency order to respect FK constraints:
	// clear current_draft_id → ai_messages → ai_sessions → versions → drafts → assets → project
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Clear current_draft_id first — projects.current_draft_id is a FK to drafts (circular ref)
		if err := tx.Model(&models.Project{}).Where("id = ?", id).Update("current_draft_id", nil).Error; err != nil {
			return fmt.Errorf("clear current_draft_id: %w", err)
		}

		// Delete AI messages for all drafts of this project
		if err := tx.Exec(`DELETE FROM ai_messages WHERE session_id IN (SELECT id FROM ai_sessions WHERE draft_id IN (SELECT id FROM drafts WHERE project_id = ?))`, id).Error; err != nil {
			return fmt.Errorf("delete ai_messages: %w", err)
		}

		// Delete AI sessions for all drafts of this project
		if err := tx.Exec(`DELETE FROM ai_sessions WHERE draft_id IN (SELECT id FROM drafts WHERE project_id = ?)`, id).Error; err != nil {
			return fmt.Errorf("delete ai_sessions: %w", err)
		}

		// Delete versions for all drafts of this project
		if err := tx.Exec(`DELETE FROM versions WHERE draft_id IN (SELECT id FROM drafts WHERE project_id = ?)`, id).Error; err != nil {
			return fmt.Errorf("delete versions: %w", err)
		}

		// Delete all drafts
		if err := tx.Where("project_id = ?", id).Delete(&models.Draft{}).Error; err != nil {
			return fmt.Errorf("delete drafts: %w", err)
		}

		// Delete all assets
		if err := tx.Where("project_id = ?", id).Delete(&models.Asset{}).Error; err != nil {
			return fmt.Errorf("delete assets: %w", err)
		}

		// Finally delete the project itself
		if err := tx.Where("id = ?", id).Delete(&models.Project{}).Error; err != nil {
			return fmt.Errorf("delete project: %w", err)
		}

		return nil
	})
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
	storage storage.FileStorage
}

func NewAssetService(db *gorm.DB, store storage.FileStorage) *AssetService {
	return &AssetService{db: db, storage: store}
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
	return s.UpdateAsset(userID, noteID, &content, &label)
}

func (s *AssetService) UpdateAsset(userID string, assetID uint, content, label *string) (*models.Asset, error) {
	var asset models.Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return nil, ErrAssetNotFound
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return nil, err
	}

	if content != nil {
		originalContent := cloneOptionalString(asset.Content)
		asset.Content = cloneOptionalString(content)
		if shouldTrackManualAssetContentEdit(asset.Type) && assetContentChanged(originalContent, content) {
			asset.Metadata = withAssetUserEditMetadata(asset.Metadata)
		}
	}
	if label != nil {
		asset.Label = cloneOptionalLabel(label)
	}
	if err := s.db.Save(&asset).Error; err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
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

func cloneOptionalString(input *string) *string {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneOptionalLabel(input *string) *string {
	if input == nil {
		return nil
	}
	if *input == "" {
		return nil
	}
	value := *input
	return &value
}

func shouldTrackManualAssetContentEdit(assetType string) bool {
	switch assetType {
	case "resume_pdf", "resume_docx", "git_repo":
		return true
	default:
		return false
	}
}

func assetContentChanged(existing, incoming *string) bool {
	if incoming == nil {
		return false
	}
	if existing == nil {
		return true
	}
	return *existing != *incoming
}

func withAssetUserEditMetadata(existing models.JSONB) models.JSONB {
	metadata := cloneAssetJSONB(existing)
	parsing := cloneAssetJSONMap(metadata["parsing"])
	parsing["updated_by_user"] = true
	metadata["parsing"] = parsing
	return metadata
}

func cloneAssetJSONB(input models.JSONB) models.JSONB {
	if input == nil {
		return models.JSONB{}
	}

	cloned := make(models.JSONB, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneAssetJSONMap(input interface{}) map[string]interface{} {
	switch typed := input.(type) {
	case map[string]interface{}:
		cloned := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			cloned[key] = value
		}
		return cloned
	case models.JSONB:
		cloned := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			cloned[key] = value
		}
		return cloned
	default:
		return map[string]interface{}{}
	}
}
