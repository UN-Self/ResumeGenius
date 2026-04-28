package parsing

import (
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
}

func NewParsingService(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor) *ParsingService {
	return &ParsingService{
		db:           db,
		pdfParser:    pdfParser,
		docxParser:   docxParser,
		gitExtractor: gitExtractor,
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
