package parsing

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

type ParsingService struct {
	db           *gorm.DB
	pdfParser    PdfParser
	docxParser   DocxParser
	gitExtractor GitExtractor
	generator    DraftGeneratorInterface
	storage      storage.FileStorage

	projectExists     func(projectID uint) (bool, error)
	listProjectAssets func(projectID uint) ([]models.Asset, error)
}

func NewParsingService(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor) *ParsingService {
	return NewParsingServiceWithGeneratorAndStorage(db, pdfParser, docxParser, gitExtractor, nil, nil)
}

func NewParsingServiceWithGenerator(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor, generator DraftGeneratorInterface) *ParsingService {
	return NewParsingServiceWithGeneratorAndStorage(db, pdfParser, docxParser, gitExtractor, generator, nil)
}

func NewParsingServiceWithGeneratorAndStorage(
	db *gorm.DB,
	pdfParser PdfParser,
	docxParser DocxParser,
	gitExtractor GitExtractor,
	generator DraftGeneratorInterface,
	store storage.FileStorage,
) *ParsingService {
	svc := &ParsingService{
		db:           db,
		pdfParser:    pdfParser,
		docxParser:   docxParser,
		gitExtractor: gitExtractor,
		generator:    generator,
		storage:      store,
	}
	svc.projectExists = svc.defaultProjectExists
	svc.listProjectAssets = svc.defaultListProjectAssets
	return svc
}

// ParseForUser verifies project ownership before loading parsing results.
func (s *ParsingService) ParseForUser(userID string, projectID uint) ([]ParsedContent, error) {
	if err := s.ensureOwnedProject(userID, projectID); err != nil {
		return nil, err
	}
	return s.Parse(projectID)
}

// Parse loads a project's assets and aggregates supported parsing results.
func (s *ParsingService) Parse(projectID uint) ([]ParsedContent, error) {
	exists, err := s.projectExists(projectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	assets, err := s.listProjectAssets(projectID)
	if err != nil {
		return nil, err
	}

	parsedContents := make([]ParsedContent, 0, len(assets))
	var firstRecoverableErr error
	for _, asset := range assets {
		if s.shouldSkipAssetInParseFlow(asset) {
			if err := s.persistAssetParsingStatus(asset, "skipped"); err != nil {
				return nil, err
			}
			continue
		}

		parsed, err := s.parseAsset(asset)
		if err != nil {
			if persistErr := s.persistAssetParsingStatus(asset, "failed"); persistErr != nil {
				return nil, persistErr
			}
			if isRecoverableAssetError(err) {
				if firstRecoverableErr == nil {
					firstRecoverableErr = err
				}
				continue
			}
			return nil, err
		}
		if parsed != nil {
			if err := s.persistParsedAsset(asset, parsed); err != nil {
				return nil, err
			}
			parsedContents = append(parsedContents, *parsed)
		}
	}

	if len(parsedContents) == 0 {
		if firstRecoverableErr != nil {
			return nil, firstRecoverableErr
		}
		return nil, ErrNoUsableAssets
	}

	return parsedContents, nil
}

type GenerateResult struct {
	DraftID     uint   `json:"draft_id"`
	VersionID   uint   `json:"version_id"`
	HTMLContent string `json:"html_content"`
}

// GenerateForUser verifies project ownership before generating a draft.
func (s *ParsingService) GenerateForUser(userID string, projectID uint) (*GenerateResult, error) {
	if err := s.ensureOwnedProject(userID, projectID); err != nil {
		return nil, err
	}
	return s.Generate(projectID)
}

// Generate runs parse -> HTML generation -> drafts persistence.
func (s *ParsingService) Generate(projectID uint) (*GenerateResult, error) {
	if s.db == nil {
		return nil, ErrDatabaseNotConfigured
	}
	if s.generator == nil {
		return nil, ErrDraftGeneratorNotConfigured
	}

	parsedContents, err := s.Parse(projectID)
	if err != nil {
		return nil, err
	}

	aggregatedText := aggregateParsedContents(parsedContents)
	if aggregatedText == "" {
		return nil, ErrNoGeneratableText
	}
	htmlContent, err := s.generator.GenerateHTML(aggregatedText)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIGenerateFailed, err)
	}

	const initialGeneratedVersionLabel = "AI 初始生成"

	result := &GenerateResult{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		draft := models.Draft{
			ProjectID:   projectID,
			HTMLContent: htmlContent,
		}
		if err := tx.Create(&draft).Error; err != nil {
			return err
		}

		versionLabel := initialGeneratedVersionLabel
		version := models.Version{
			DraftID:      draft.ID,
			HTMLSnapshot: draft.HTMLContent,
			Label:        &versionLabel,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		update := tx.Model(&models.Project{}).
			Where("id = ?", projectID).
			Update("current_draft_id", draft.ID)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return ErrProjectNotFound
		}

		result.DraftID = draft.ID
		result.VersionID = version.ID
		result.HTMLContent = draft.HTMLContent
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *ParsingService) ensureOwnedProject(userID string, projectID uint) error {
	if s.db == nil {
		return ErrDatabaseNotConfigured
	}

	var project models.Project
	err := s.db.Select("id").
		Where("user_id = ? AND id = ?", userID, projectID).
		Take(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		return err
	}

	return nil
}

func (s *ParsingService) defaultProjectExists(projectID uint) (bool, error) {
	if s.db == nil {
		return false, ErrDatabaseNotConfigured
	}

	var project models.Project
	err := s.db.Select("id").Take(&project, projectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *ParsingService) defaultListProjectAssets(projectID uint) ([]models.Asset, error) {
	if s.db == nil {
		return nil, ErrDatabaseNotConfigured
	}

	var assets []models.Asset
	if err := s.db.
		Where("project_id = ?", projectID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&assets).Error; err != nil {
		return nil, err
	}

	return assets, nil
}

func (s *ParsingService) shouldSkipAssetInParseFlow(asset models.Asset) bool {
	switch asset.Type {
	case AssetTypeResumeImage:
		return true
	case AssetTypeGitRepo:
		// B5 keeps the core flow on PDF/DOCX/note; Git joins once an extractor is wired.
		return s.gitExtractor == nil
	default:
		return false
	}
}

// parseAsset is the central asset-type dispatcher for module parsing.
func (s *ParsingService) parseAsset(asset models.Asset) (*ParsedContent, error) {
	switch asset.Type {
	case AssetTypeResumePDF:
		return s.parsePDFAsset(asset)
	case AssetTypeResumeDOCX:
		return s.parseDOCXAsset(asset)
	case AssetTypeResumeImage:
		return nil, ErrAssetTypeSkipped
	case AssetTypeGitRepo:
		return s.parseGitAsset(asset)
	case AssetTypeNote:
		return s.parseNoteAsset(asset)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAssetType, asset.Type)
	}
}

func (s *ParsingService) parsePDFAsset(asset models.Asset) (*ParsedContent, error) {
	if s.pdfParser == nil {
		return nil, ErrPDFParserNotConfigured
	}
	path, err := s.resolveAssetPath(asset)
	if err != nil {
		return nil, err
	}

	parsed, err := s.pdfParser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPDFParseFailed, err)
	}
	return cleanParsedContentText(attachAssetMetadata(asset, parsed)), nil
}

func (s *ParsingService) parseDOCXAsset(asset models.Asset) (*ParsedContent, error) {
	if s.docxParser == nil {
		return nil, ErrDOCXParserNotConfigured
	}
	path, err := s.resolveAssetPath(asset)
	if err != nil {
		return nil, err
	}

	parsed, err := s.docxParser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDOCXParseFailed, err)
	}
	return cleanParsedContentText(attachAssetMetadata(asset, parsed)), nil
}

func (s *ParsingService) parseGitAsset(asset models.Asset) (*ParsedContent, error) {
	if s.gitExtractor == nil {
		return nil, ErrGitExtractorNotConfigured
	}
	repoURL, err := requireAssetURI(asset)
	if err != nil {
		return nil, err
	}

	parsed, err := s.gitExtractor.Extract(repoURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitExtractFailed, err)
	}
	return cleanParsedContentText(attachAssetMetadata(asset, parsed)), nil
}

func (s *ParsingService) parseNoteAsset(asset models.Asset) (*ParsedContent, error) {
	text, err := requireNoteContent(asset)
	if err != nil {
		return nil, err
	}

	return cleanParsedContentText(attachAssetMetadata(asset, &ParsedContent{
		Text:   text,
		Images: nil,
	})), nil
}

func (s *ParsingService) resolveAssetPath(asset models.Asset) (string, error) {
	if s.storage == nil {
		return requireAssetURI(asset)
	}

	key, err := requireAssetURI(asset)
	if err != nil {
		return "", err
	}
	return s.storage.Resolve(key)
}

func requireAssetURI(asset models.Asset) (string, error) {
	if asset.URI == nil || *asset.URI == "" {
		return "", ErrAssetURIMissing
	}
	return *asset.URI, nil
}

func requireNoteContent(asset models.Asset) (string, error) {
	if asset.Content == nil {
		return "", ErrAssetContentMissing
	}

	content := strings.TrimSpace(*asset.Content)
	if content == "" {
		return "", ErrAssetContentMissing
	}

	if asset.Label == nil {
		return content, nil
	}

	label := strings.TrimSpace(*asset.Label)
	if label == "" {
		return content, nil
	}

	return label + "\n" + content, nil
}

func attachAssetMetadata(asset models.Asset, parsed *ParsedContent) *ParsedContent {
	if parsed == nil {
		return nil
	}

	decorated := *parsed
	decorated.AssetID = asset.ID
	decorated.Type = asset.Type
	decorated.Label = AssetLabel(asset.Label, asset.URI)
	return &decorated
}

func (s *ParsingService) persistParsedAsset(asset models.Asset, parsed *ParsedContent) error {
	if s.db == nil || asset.ID == 0 {
		return nil
	}

	var oldDerivedAssets []models.Asset
	if s.storage != nil {
		oldDerivedAssets = loadDerivedImageAssetsByIDs(s.db, asset.ProjectID, derivedImageAssetIDsFromMetadata(asset.Metadata))
	}
	savedKeys := make([]string, 0, len(parsed.Images))
	newDerivedImageAssetIDs := make([]uint, 0, len(parsed.Images))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if s.storage != nil {
			for index, image := range parsed.Images {
				key, createdID, err := s.createDerivedImageAsset(tx, asset, image, index)
				if err != nil {
					return err
				}
				savedKeys = append(savedKeys, key)
				newDerivedImageAssetIDs = append(newDerivedImageAssetIDs, createdID)
			}
		}

		derivedImageIDsForMetadata := []uint(nil)
		if s.storage != nil {
			derivedImageIDsForMetadata = newDerivedImageAssetIDs
		}
		imagesPersisted := s.storage != nil && len(parsed.Images) > 0

		updates := map[string]interface{}{
			"metadata": withAssetParsingMetadata(
				asset.Metadata,
				"success",
				parsed,
				shouldPersistParsedText(asset),
				derivedImageIDsForMetadata,
				imagesPersisted,
			),
		}
		if content, ok := parsedAssetContentForPersistence(asset, parsed); ok {
			updates["content"] = content
		}
		if err := tx.Model(&models.Asset{}).Where("id = ?", asset.ID).Updates(updates).Error; err != nil {
			return err
		}

		oldIDs := assetIDs(oldDerivedAssets)
		if len(oldIDs) > 0 {
			if err := tx.Where("project_id = ? AND id IN ?", asset.ProjectID, oldIDs).Delete(&models.Asset{}).Error; err != nil {
				return fmt.Errorf("delete stale derived image assets: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		for _, key := range savedKeys {
			_ = s.storage.Delete(key)
		}
		return err
	}

	for _, oldAsset := range oldDerivedAssets {
		if oldAsset.URI != nil {
			_ = s.storage.Delete(*oldAsset.URI)
		}
	}

	return nil
}

func (s *ParsingService) persistAssetParsingStatus(asset models.Asset, status string) error {
	if s.db == nil || asset.ID == 0 {
		return nil
	}

	result := s.db.Model(&models.Asset{}).
		Where("id = ?", asset.ID).
		Update("metadata", withAssetParsingMetadata(asset.Metadata, status, nil, false, nil, false))
	return result.Error
}

func parsedAssetContentForPersistence(asset models.Asset, parsed *ParsedContent) (string, bool) {
	if parsed == nil || !shouldPersistParsedText(asset) {
		return "", false
	}
	return strings.TrimSpace(parsed.Text), true
}

func shouldPersistParsedText(asset models.Asset) bool {
	switch asset.Type {
	case AssetTypeResumePDF, AssetTypeResumeDOCX, AssetTypeGitRepo:
		return true
	default:
		return false
	}
}

func withAssetParsingMetadata(
	existing models.JSONB,
	status string,
	parsed *ParsedContent,
	contentPersisted bool,
	derivedImageAssetIDs []uint,
	imagesPersisted bool,
) models.JSONB {
	metadata := cloneJSONB(existing)
	parsing := cloneJSONMap(metadata["parsing"])
	parsing["status"] = status

	if status == "success" {
		parsing["cleaned"] = true
		parsing["image_count"] = parsedImageCount(parsed)
		if contentPersisted {
			parsing["content_persisted"] = true
		}
	}
	if derivedImageAssetIDs != nil {
		parsing["derived_image_asset_ids"] = uintSliceToInterfaceSlice(derivedImageAssetIDs)
	}
	if imagesPersisted {
		parsing["images_persisted"] = true
	}

	metadata["parsing"] = parsing
	return metadata
}

func cloneJSONB(input models.JSONB) models.JSONB {
	if input == nil {
		return models.JSONB{}
	}

	cloned := make(models.JSONB, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneJSONMap(input interface{}) map[string]interface{} {
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

func parsedImageCount(parsed *ParsedContent) int {
	if parsed == nil {
		return 0
	}
	return len(parsed.Images)
}

func (s *ParsingService) createDerivedImageAsset(tx *gorm.DB, sourceAsset models.Asset, image ParsedImage, index int) (string, uint, error) {
	if s.storage == nil {
		return "", 0, ErrDatabaseNotConfigured
	}

	imageBytes, err := base64.StdEncoding.DecodeString(image.DataBase64)
	if err != nil {
		return "", 0, fmt.Errorf("decode parsed image base64: %w", err)
	}

	filename := fmt.Sprintf("asset-%d-image-%d.png", sourceAsset.ID, index+1)
	key, err := s.storage.Save(sourceAsset.ProjectID, filename, imageBytes)
	if err != nil {
		return "", 0, fmt.Errorf("save parsed image: %w", err)
	}

	label := derivedPersistedImageLabel(sourceAsset, index)
	metadata := models.JSONB{
		"parsing": map[string]interface{}{
			"derived":           true,
			"source_asset_id":   sourceAsset.ID,
			"source_asset_type": sourceAsset.Type,
		},
	}
	if strings.TrimSpace(image.Description) != "" {
		metadata["image_description"] = strings.TrimSpace(image.Description)
	}

	derivedAsset := models.Asset{
		ProjectID: sourceAsset.ProjectID,
		Type:      AssetTypeResumeImage,
		URI:       &key,
		Label:     &label,
		Metadata:  metadata,
	}
	if err := tx.Create(&derivedAsset).Error; err != nil {
		_ = s.storage.Delete(key)
		return "", 0, fmt.Errorf("create derived image asset: %w", err)
	}

	return key, derivedAsset.ID, nil
}

func derivedImageLabel(sourceAsset models.Asset, index int) string {
	base := strings.TrimSpace(AssetLabel(sourceAsset.Label, sourceAsset.URI))
	if base == "" {
		base = fmt.Sprintf("素材 %d", sourceAsset.ID)
	}
	return fmt.Sprintf("%s 图片 %d", base, index+1)
}

func derivedImageAssetIDsFromMetadata(metadata models.JSONB) []uint {
	parsing := cloneJSONMap(metadata["parsing"])
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

func loadDerivedImageKeysForCleanup(db *gorm.DB, projectID uint, ids []uint) []models.Asset {
	if db == nil || len(ids) == 0 {
		return nil
	}

	var assets []models.Asset
	if err := db.Where("project_id = ? AND id IN ?", projectID, ids).Find(&assets).Error; err != nil {
		return nil
	}
	return assets
}

func uintSliceToInterfaceSlice(ids []uint) []interface{} {
	values := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		values = append(values, id)
	}
	return values
}

func derivedPersistedImageLabel(sourceAsset models.Asset, index int) string {
	base := strings.TrimSpace(AssetLabel(sourceAsset.Label, sourceAsset.URI))
	if base == "" {
		base = fmt.Sprintf("Asset %d", sourceAsset.ID)
	}
	return fmt.Sprintf("%s Image %d", base, index+1)
}

func loadDerivedImageAssetsByIDs(db *gorm.DB, projectID uint, ids []uint) []models.Asset {
	return loadDerivedImageKeysForCleanup(db, projectID, ids)
}

func assetIDs(assets []models.Asset) []uint {
	ids := make([]uint, 0, len(assets))
	for _, asset := range assets {
		ids = append(ids, asset.ID)
	}
	return ids
}

func isRecoverableAssetError(err error) bool {
	switch {
	case errors.Is(err, ErrAssetURIMissing):
		return true
	case errors.Is(err, ErrAssetContentMissing):
		return true
	case errors.Is(err, ErrAssetTypeSkipped):
		return true
	case errors.Is(err, ErrUnsupportedAssetType):
		return true
	case errors.Is(err, ErrPDFParseFailed):
		return true
	case errors.Is(err, ErrDOCXParseFailed):
		return true
	case errors.Is(err, ErrGitExtractFailed):
		return true
	default:
		return false
	}
}

func aggregateParsedContents(parsedContents []ParsedContent) string {
	parts := make([]string, 0, len(parsedContents))
	for _, parsed := range parsedContents {
		text := strings.TrimSpace(parsed.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n\n")
}
