package parsing

import (
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
	return NewParsingServiceWithGenerator(db, pdfParser, docxParser, gitExtractor, nil)
}

func NewParsingServiceWithGenerator(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor, generator DraftGeneratorInterface) *ParsingService {
	svc := &ParsingService{
		db:           db,
		pdfParser:    pdfParser,
		docxParser:   docxParser,
		gitExtractor: gitExtractor,
		generator:    generator,
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
			continue
		}

		parsed, err := s.parseAsset(asset)
		if err != nil {
			if isRecoverableAssetError(err) {
				if firstRecoverableErr == nil {
					firstRecoverableErr = err
				}
				continue
			}
			return nil, err
		}
		if parsed != nil {
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
	return attachAssetMetadata(asset, parsed), nil
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
	return attachAssetMetadata(asset, parsed), nil
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
	return attachAssetMetadata(asset, parsed), nil
}

func (s *ParsingService) parseNoteAsset(asset models.Asset) (*ParsedContent, error) {
	text, err := requireNoteContent(asset)
	if err != nil {
		return nil, err
	}

	return attachAssetMetadata(asset, &ParsedContent{
		Text:   text,
		Images: nil,
	}), nil
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
