package intake

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

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
	var count int64
	result := s.db.Model(&models.Project{}).Where("id = ? AND user_id = ?", id, userID).Count(&count)
	if result.Error != nil {
		return fmt.Errorf("validate project ownership: %w", result.Error)
	}
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
		var draftIDs []uint
		if err := tx.Model(&models.Draft{}).Where("project_id = ?", id).Pluck("id", &draftIDs).Error; err != nil {
			return fmt.Errorf("load draft ids: %w", err)
		}

		if len(draftIDs) > 0 {
			if err := tx.Where("draft_id IN ?", draftIDs).Delete(&models.Version{}).Error; err != nil {
				return fmt.Errorf("delete versions: %w", err)
			}

			var sessionIDs []uint
			if err := tx.Model(&models.AISession{}).Where("draft_id IN ?", draftIDs).Pluck("id", &sessionIDs).Error; err != nil {
				return fmt.Errorf("load ai session ids: %w", err)
			}

			if len(sessionIDs) > 0 {
				if err := tx.Model(&models.AIMessage{}).Where("session_id IN ?", sessionIDs).Update("tool_call_id", nil).Error; err != nil {
					return fmt.Errorf("clear ai_message.tool_call_id: %w", err)
				}
				if err := tx.Where("session_id IN ?", sessionIDs).Delete(&models.AIMessage{}).Error; err != nil {
					return fmt.Errorf("delete ai_messages: %w", err)
				}
				if err := tx.Where("session_id IN ?", sessionIDs).Delete(&models.AIToolCall{}).Error; err != nil {
					return fmt.Errorf("delete ai_tool_calls: %w", err)
				}
			}

			if err := tx.Where("draft_id IN ?", draftIDs).Delete(&models.AISession{}).Error; err != nil {
				return fmt.Errorf("delete ai_sessions: %w", err)
			}
			if err := tx.Where("project_id = ?", id).Delete(&models.Draft{}).Error; err != nil {
				return fmt.Errorf("delete drafts: %w", err)
			}
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
	ErrUnsupportedFormat    = errors.New("unsupported file format")
	ErrFileTooLarge         = errors.New("file size exceeds 20MB limit")
	ErrInvalidGitURL        = errors.New("invalid git repository URL")
	ErrProjectNotFound      = errors.New("project not found")
	ErrAssetNotFound        = errors.New("asset not found")
	ErrAssetURIMissing      = errors.New("asset uri is required")
	ErrInvalidFolderName    = errors.New("folder name is required")
	ErrFolderDepthExceeded  = errors.New("folder depth cannot exceed 7 levels")
	ErrReplaceAssetMismatch = errors.New("replacement asset does not match uploaded filename")
)

var allowedExtensions = map[string]string{
	".pdf":  "resume_pdf",
	".docx": "resume_docx",
	".png":  "resume_image",
	".jpg":  "resume_image",
	".jpeg": "resume_image",
}

var maxFileSize = 20 * 1024 * 1024
var maxFolderDepth = 7

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
	result := s.db.Model(&models.Project{}).Where("id = ? AND user_id = ?", projectID, userID).Count(&count)
	if result.Error != nil {
		return fmt.Errorf("validate project: %w", result.Error)
	}
	if count == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *AssetService) UploadFile(userID string, projectID uint, filename string, data []byte, size int64) (*models.Asset, error) {
	return s.UploadFileWithReplacement(userID, projectID, filename, data, size, nil, nil)
}

func (s *AssetService) UploadFileWithReplacement(userID string, projectID uint, filename string, data []byte, size int64, replaceAssetID *uint, folderIDs ...*uint) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}
	var folderID *uint
	if len(folderIDs) > 0 {
		folderID = folderIDs[0]
	}
	if folderID != nil {
		if err := s.validateFolder(userID, projectID, *folderID); err != nil {
			return nil, err
		}
	}

	ext := strings.ToLower(filepath.Ext(filename))
	assetType, ok := allowedExtensions[ext]
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	if size > int64(maxFileSize) {
		return nil, ErrFileTooLarge
	}

	var replaceAsset *models.Asset
	var err error
	if replaceAssetID != nil {
		replaceAsset, err = s.loadReplaceableAsset(userID, projectID, filename, assetType, *replaceAssetID)
		if err != nil {
			return nil, err
		}
	}

	fileHash := storage.SHA256Hex(data)
	restoredAsset, err := s.restoreDeletedAssetByFileHash(userID, projectID, filename, assetType, fileHash, ext, data, replaceAsset, folderID)
	if err != nil {
		return nil, err
	}
	if restoredAsset != nil {
		return restoredAsset, nil
	}

	uri, err := s.storage.Save(userID, fileHash, ext, data)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      assetType,
		URI:       &uri,
		FileHash:  &fileHash,
		Metadata:  withAssetFolderMetadata(withAssetOriginalFilenameMetadata(nil, filename), folderID),
	}
	if err := s.db.Create(&asset).Error; err != nil {
		_ = s.storage.Delete(uri)
		return nil, fmt.Errorf("create asset record: %w", err)
	}

	if replaceAsset != nil {
		if err := s.DeleteAsset(userID, replaceAsset.ID); err != nil {
			_ = s.db.Delete(&models.Asset{}, asset.ID).Error
			_ = s.storage.Delete(uri)
			return nil, err
		}
	}

	return &asset, nil
}

func (s *AssetService) CreateFolder(userID string, projectID uint, name string, parentFolderID *uint) (*models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}
	if parentFolderID != nil {
		if err := s.validateFolder(userID, projectID, *parentFolderID); err != nil {
			return nil, err
		}
		parentDepth, err := s.folderDepth(projectID, *parentFolderID)
		if err != nil {
			return nil, err
		}
		if parentDepth >= maxFolderDepth {
			return nil, ErrFolderDepthExceeded
		}
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidFolderName
	}

	asset := models.Asset{
		ProjectID: projectID,
		Type:      "folder",
		Label:     &trimmedName,
		Metadata:  withAssetFolderMetadata(models.JSONB{"folder": true}, parentFolderID),
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("create folder asset: %w", err)
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

// MoveAsset moves an asset to a different folder (or to root when targetFolderID is nil).
func (s *AssetService) MoveAsset(userID string, assetID uint, targetFolderID *uint) (*models.Asset, error) {
	var asset models.Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return nil, ErrAssetNotFound
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return nil, err
	}

	// Prevent moving a folder into itself
	if asset.Type == "folder" && targetFolderID != nil && *targetFolderID == assetID {
		return nil, fmt.Errorf("cannot move a folder into itself")
	}

	if targetFolderID != nil {
		if err := s.validateFolder(userID, asset.ProjectID, *targetFolderID); err != nil {
			return nil, err
		}

		if asset.Type == "folder" {
			if err := s.preventCircularMove(asset.ProjectID, assetID, *targetFolderID); err != nil {
				return nil, err
			}
		}

		// Check depth when moving a folder into another folder
		if asset.Type == "folder" {
			targetDepth, err := s.folderDepth(asset.ProjectID, *targetFolderID)
			if err != nil {
				return nil, err
			}
			if targetDepth+1 >= maxFolderDepth {
				return nil, ErrFolderDepthExceeded
			}
		}
	}

	asset.Metadata = withAssetFolderMetadata(asset.Metadata, targetFolderID)
	if err := s.db.Save(&asset).Error; err != nil {
		return nil, fmt.Errorf("move asset: %w", err)
	}

	return &asset, nil
}

// preventCircularMove ensures moving a folder into one of its descendants is rejected.
func (s *AssetService) preventCircularMove(projectID, folderID, targetID uint) error {
	currentID := targetID
	seen := make(map[uint]struct{})

	for currentID != 0 {
		if currentID == folderID {
			return fmt.Errorf("cannot move a folder into its own subfolder")
		}
		if _, ok := seen[currentID]; ok {
			return nil
		}
		seen[currentID] = struct{}{}

		var folder models.Asset
		if err := s.db.First(&folder, currentID).Error; err != nil {
			return nil
		}
		parentID, ok := assetFolderID(folder)
		if !ok {
			return nil
		}
		currentID = parentID
	}
	return nil
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

func (s *AssetService) ResolveAssetFile(userID string, assetID uint) (*models.Asset, string, error) {
	var asset models.Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return nil, "", ErrAssetNotFound
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return nil, "", err
	}

	if asset.URI == nil || strings.TrimSpace(*asset.URI) == "" {
		return nil, "", ErrAssetURIMissing
	}

	path, err := s.storage.Resolve(strings.TrimSpace(*asset.URI))
	if err != nil {
		return nil, "", err
	}
	if !s.storage.Exists(strings.TrimSpace(*asset.URI)) {
		return nil, "", ErrAssetNotFound
	}

	return &asset, path, nil
}

func (s *AssetService) DeleteAsset(userID string, assetID uint) error {
	var asset models.Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return ErrAssetNotFound
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.softDeleteAssetCascade(tx, asset); err != nil {
			return err
		}
		return nil
	})
}

func (s *AssetService) DeleteProjectAssets(userID string, projectID uint) error {
	if err := s.validateProject(userID, projectID); err != nil {
		return err
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

func withAssetOriginalFilenameMetadata(existing models.JSONB, filename string) models.JSONB {
	metadata := cloneAssetJSONB(existing)
	parsing := cloneAssetJSONMap(metadata["parsing"])
	originalFilename := strings.TrimSpace(filepath.Base(filename))
	if originalFilename != "" {
		parsing["original_filename"] = originalFilename
	}
	metadata["parsing"] = parsing
	return metadata
}

func withAssetFolderMetadata(existing models.JSONB, folderID *uint) models.JSONB {
	metadata := cloneAssetJSONB(existing)
	if folderID == nil {
		delete(metadata, "folder_id")
		return metadata
	}
	metadata["folder_id"] = *folderID
	return metadata
}

func (s *AssetService) validateFolder(userID string, projectID uint, folderID uint) error {
	var folder models.Asset
	if err := s.db.First(&folder, folderID).Error; err != nil {
		return ErrAssetNotFound
	}
	if folder.ProjectID != projectID || folder.Type != "folder" {
		return ErrAssetNotFound
	}
	return s.validateProject(userID, folder.ProjectID)
}

func (s *AssetService) folderDepth(projectID uint, folderID uint) (int, error) {
	depth := 0
	currentID := folderID
	seen := make(map[uint]struct{})

	for currentID != 0 {
		if _, ok := seen[currentID]; ok {
			return depth, nil
		}
		seen[currentID] = struct{}{}

		var folder models.Asset
		if err := s.db.First(&folder, currentID).Error; err != nil {
			return 0, ErrAssetNotFound
		}
		if folder.ProjectID != projectID || folder.Type != "folder" {
			return 0, ErrAssetNotFound
		}

		depth++
		parentID, ok := assetFolderID(folder)
		if !ok {
			break
		}
		currentID = parentID
	}

	return depth, nil
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

func (s *AssetService) loadReplaceableAsset(userID string, projectID uint, filename, assetType string, replaceAssetID uint) (*models.Asset, error) {
	var asset models.Asset
	if err := s.db.First(&asset, replaceAssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, fmt.Errorf("find replacement asset: %w", err)
	}

	if asset.ProjectID != projectID {
		return nil, ErrReplaceAssetMismatch
	}

	if err := s.validateProject(userID, asset.ProjectID); err != nil {
		return nil, err
	}

	if asset.Type != assetType {
		return nil, ErrReplaceAssetMismatch
	}

	existingName := assetOriginalFilename(asset)
	if existingName == "" || !sameAssetFilename(existingName, filename) {
		return nil, ErrReplaceAssetMismatch
	}

	return &asset, nil
}

func (s *AssetService) findDeletedAssetByFileHash(userID, fileHash string) (*models.Asset, error) {
	var asset models.Asset
	err := s.db.Unscoped().
		Joins("JOIN projects ON projects.id = assets.project_id").
		Where("projects.user_id = ? AND assets.file_hash = ? AND assets.deleted_at IS NOT NULL", userID, fileHash).
		Order("assets.deleted_at DESC").
		First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find deleted asset by file hash: %w", err)
	}
	return &asset, nil
}

func (s *AssetService) restoreDeletedAssetByFileHash(
	userID string,
	projectID uint,
	filename, assetType, fileHash, ext string,
	data []byte,
	replaceAsset *models.Asset,
	folderID *uint,
) (*models.Asset, error) {
	deletedAsset, err := s.findDeletedAssetByFileHash(userID, fileHash)
	if err != nil || deletedAsset == nil {
		return deletedAsset, err
	}

	restoredMetadata := withAssetFolderMetadata(withAssetOriginalFilenameMetadata(deletedAsset.Metadata, filename), folderID)
	restoredURI, err := s.storage.Save(userID, fileHash, ext, data)
	if err != nil {
		return nil, fmt.Errorf("restore asset file: %w", err)
	}
	derivedIDs := derivedImageAssetIDsFromAssetMetadata(deletedAsset.Metadata)
	restoredAssetID := deletedAsset.ID

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if replaceAsset != nil {
			if err := s.softDeleteAssetCascade(tx, *replaceAsset); err != nil {
				return err
			}
		}

		updates := map[string]interface{}{
			"project_id": projectID,
			"type":       assetType,
			"uri":        restoredURI,
			"label":      nil,
			"file_hash":  fileHash,
			"metadata":   restoredMetadata,
			"deleted_at": nil,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}
		if err := tx.Unscoped().Model(&models.Asset{}).Where("id = ?", restoredAssetID).Updates(updates).Error; err != nil {
			return fmt.Errorf("restore deleted asset: %w", err)
		}

		if len(derivedIDs) == 0 {
			return nil
		}

		if err := tx.Unscoped().
			Model(&models.Asset{}).
			Where("type = ? AND id IN ?", "resume_image", derivedIDs).
			Updates(map[string]interface{}{
				"project_id": projectID,
				"deleted_at": nil,
				"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
			return fmt.Errorf("restore derived image assets: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var restoredAsset models.Asset
	if err := s.db.First(&restoredAsset, restoredAssetID).Error; err != nil {
		return nil, fmt.Errorf("reload restored asset: %w", err)
	}
	return &restoredAsset, nil
}

func (s *AssetService) softDeleteAssetCascade(tx *gorm.DB, asset models.Asset) error {
	assetsToDelete, err := s.collectAssetsForDelete(tx, asset)
	if err != nil {
		return err
	}

	ids := assetIDsForDelete(assetsToDelete)
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("id IN ?", ids).Delete(&models.Asset{}).Error; err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}
	return nil
}

func (s *AssetService) collectAssetsForDelete(db *gorm.DB, asset models.Asset) ([]models.Asset, error) {
	assets := []models.Asset{asset}
	if asset.Type == "folder" {
		descendants, err := s.collectFolderDescendantsForDelete(db, asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, descendants...)
	}

	derivedIDs := derivedImageAssetIDsForAssets(assets)
	if len(derivedIDs) == 0 {
		return assets, nil
	}

	var derivedAssets []models.Asset
	if err := db.
		Where("project_id = ? AND type = ? AND id IN ?", asset.ProjectID, "resume_image", derivedIDs).
		Find(&derivedAssets).Error; err != nil {
		return nil, fmt.Errorf("find derived image assets: %w", err)
	}

	return append(assets, derivedAssets...), nil
}

func derivedImageAssetIDsForAssets(assets []models.Asset) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0)
	for _, asset := range assets {
		for _, id := range derivedImageAssetIDsFromAssetMetadata(asset.Metadata) {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *AssetService) collectFolderDescendantsForDelete(db *gorm.DB, folder models.Asset) ([]models.Asset, error) {
	var projectAssets []models.Asset
	if err := db.Where("project_id = ?", folder.ProjectID).Find(&projectAssets).Error; err != nil {
		return nil, fmt.Errorf("find folder descendants: %w", err)
	}

	descendants := make([]models.Asset, 0)
	visitedFolders := map[uint]struct{}{folder.ID: {}}
	queue := []uint{folder.ID}

	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]

		for _, candidate := range projectAssets {
			candidateFolderID, ok := assetFolderID(candidate)
			if !ok || candidateFolderID != parentID || candidate.ID == folder.ID {
				continue
			}

			descendants = append(descendants, candidate)
			if candidate.Type != "folder" {
				continue
			}
			if _, seen := visitedFolders[candidate.ID]; seen {
				continue
			}
			visitedFolders[candidate.ID] = struct{}{}
			queue = append(queue, candidate.ID)
		}
	}

	return descendants, nil
}

func assetFolderID(asset models.Asset) (uint, bool) {
	raw, ok := asset.Metadata["folder_id"]
	if !ok {
		return 0, false
	}

	switch typed := raw.(type) {
	case float64:
		if typed <= 0 {
			return 0, false
		}
		return uint(typed), true
	case int:
		if typed <= 0 {
			return 0, false
		}
		return uint(typed), true
	case uint:
		if typed == 0 {
			return 0, false
		}
		return typed, true
	default:
		return 0, false
	}
}

func derivedImageAssetIDsFromAssetMetadata(metadata models.JSONB) []uint {
	parsing := cloneAssetJSONMap(metadata["parsing"])
	rawIDs, ok := parsing["derived_image_asset_ids"]
	if !ok {
		return nil
	}

	items, ok := rawIDs.([]interface{})
	if !ok {
		return nil
	}

	ids := make([]uint, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case float64:
			ids = append(ids, uint(typed))
		case int:
			ids = append(ids, uint(typed))
		case uint:
			ids = append(ids, typed)
		}
	}
	return ids
}

func assetIDsForDelete(assets []models.Asset) []uint {
	ids := make([]uint, 0, len(assets))
	seen := make(map[uint]struct{}, len(assets))
	for _, asset := range assets {
		if asset.ID == 0 {
			continue
		}
		if _, ok := seen[asset.ID]; ok {
			continue
		}
		seen[asset.ID] = struct{}{}
		ids = append(ids, asset.ID)
	}
	return ids
}

var storedFilePrefixPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}_`)

func assetOriginalFilename(asset models.Asset) string {
	parsing := cloneAssetJSONMap(asset.Metadata["parsing"])
	if originalFilename, ok := parsing["original_filename"].(string); ok {
		trimmed := strings.TrimSpace(originalFilename)
		if trimmed != "" {
			return trimmed
		}
	}

	if asset.URI == nil || *asset.URI == "" {
		return ""
	}

	base := path.Base(*asset.URI)
	base = storedFilePrefixPattern.ReplaceAllString(base, "")
	return strings.TrimSpace(base)
}

func sameAssetFilename(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}
