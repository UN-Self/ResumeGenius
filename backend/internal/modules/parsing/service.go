package parsing

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

type ParsingService struct {
	db           *gorm.DB
	pdfParser    PdfParser
	docxParser   DocxParser
	gitExtractor GitExtractor

	projectExists     func(projectID uint) (bool, error)
	listProjectAssets func(projectID uint) ([]models.Asset, error)
}

func NewParsingService(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor) *ParsingService {
	svc := &ParsingService{
		db:           db,
		pdfParser:    pdfParser,
		docxParser:   docxParser,
		gitExtractor: gitExtractor,
	}
	svc.projectExists = svc.defaultProjectExists
	svc.listProjectAssets = svc.defaultListProjectAssets
	return svc
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
	for _, asset := range assets {
		if s.shouldSkipAssetInParseFlow(asset) {
			continue
		}

		parsed, err := s.parseAsset(asset)
		if err != nil {
			return nil, err
		}
		if parsed != nil {
			parsedContents = append(parsedContents, *parsed)
		}
	}

	if len(parsedContents) == 0 {
		return nil, ErrNoUsableAssets
	}

	return parsedContents, nil
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
	path, err := requireAssetURI(asset)
	if err != nil {
		return nil, err
	}

	parsed, err := s.pdfParser.Parse(path)
	if err != nil {
		return nil, err
	}
	return attachAssetMetadata(asset, parsed), nil
}

func (s *ParsingService) parseDOCXAsset(asset models.Asset) (*ParsedContent, error) {
	if s.docxParser == nil {
		return nil, ErrDOCXParserNotConfigured
	}
	path, err := requireAssetURI(asset)
	if err != nil {
		return nil, err
	}

	parsed, err := s.docxParser.Parse(path)
	if err != nil {
		return nil, err
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
		return nil, err
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
	return &decorated
}
