package parsing

import (
	"errors"
	"strings"
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

type stubPdfParser struct {
	calledWith string
	result     *ParsedContent
	err        error
}

func (s *stubPdfParser) Parse(path string) (*ParsedContent, error) {
	s.calledWith = path
	return s.result, s.err
}

type stubDocxParser struct {
	calledWith string
	result     *ParsedContent
	err        error
}

func (s *stubDocxParser) Parse(path string) (*ParsedContent, error) {
	s.calledWith = path
	return s.result, s.err
}

type stubGitExtractor struct {
	calledWith string
	result     *ParsedContent
	err        error
}

func (s *stubGitExtractor) Extract(repoURL string) (*ParsedContent, error) {
	s.calledWith = repoURL
	return s.result, s.err
}

func TestNewParsingServiceStoresDependencies(t *testing.T) {
	pdfParser := &stubPdfParser{}
	docxParser := &stubDocxParser{}
	gitExtractor := &stubGitExtractor{}

	svc := NewParsingService(nil, pdfParser, docxParser, gitExtractor)

	if svc.pdfParser != pdfParser {
		t.Error("expected pdf parser to be stored")
	}
	if svc.docxParser != docxParser {
		t.Error("expected docx parser to be stored")
	}
	if svc.gitExtractor != gitExtractor {
		t.Error("expected git extractor to be stored")
	}
}

func TestParseAssetDispatchesPDFParser(t *testing.T) {
	path := "fixtures/sample_resume.pdf"
	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "pdf text",
			Images: []ParsedImage{
				{Description: "头像", DataBase64: "abc"},
			},
		},
	}
	svc := NewParsingService(nil, pdfParser, nil, nil)

	parsed, err := svc.parseAsset(models.Asset{
		ID:   11,
		Type: AssetTypeResumePDF,
		URI:  &path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pdfParser.calledWith != path {
		t.Fatalf("expected pdf parser path %s, got %s", path, pdfParser.calledWith)
	}
	if parsed.AssetID != 11 {
		t.Fatalf("expected asset_id 11, got %d", parsed.AssetID)
	}
	if parsed.Type != AssetTypeResumePDF {
		t.Fatalf("expected type %s, got %s", AssetTypeResumePDF, parsed.Type)
	}
	if parsed.Text != "pdf text" {
		t.Fatalf("expected text to be preserved")
	}
	if len(parsed.Images) != 1 || parsed.Images[0].Description != "头像" {
		t.Fatalf("expected image metadata to be preserved")
	}
}

func TestParseAssetDispatchesDOCXParser(t *testing.T) {
	path := "fixtures/sample_resume.docx"
	docxParser := &stubDocxParser{
		result: &ParsedContent{Text: "docx text"},
	}
	svc := NewParsingService(nil, nil, docxParser, nil)

	parsed, err := svc.parseAsset(models.Asset{
		ID:   22,
		Type: AssetTypeResumeDOCX,
		URI:  &path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docxParser.calledWith != path {
		t.Fatalf("expected docx parser path %s, got %s", path, docxParser.calledWith)
	}
	if parsed.AssetID != 22 {
		t.Fatalf("expected asset_id 22, got %d", parsed.AssetID)
	}
	if parsed.Type != AssetTypeResumeDOCX {
		t.Fatalf("expected type %s, got %s", AssetTypeResumeDOCX, parsed.Type)
	}
	if parsed.Text != "docx text" {
		t.Fatalf("expected text to be preserved")
	}
}

func TestParseAssetDispatchesGitExtractor(t *testing.T) {
	repoURL := "https://github.com/example/project"
	gitExtractor := &stubGitExtractor{
		result: &ParsedContent{Text: "git summary"},
	}
	svc := NewParsingService(nil, nil, nil, gitExtractor)

	parsed, err := svc.parseAsset(models.Asset{
		ID:   33,
		Type: AssetTypeGitRepo,
		URI:  &repoURL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitExtractor.calledWith != repoURL {
		t.Fatalf("expected git extractor url %s, got %s", repoURL, gitExtractor.calledWith)
	}
	if parsed.AssetID != 33 {
		t.Fatalf("expected asset_id 33, got %d", parsed.AssetID)
	}
	if parsed.Type != AssetTypeGitRepo {
		t.Fatalf("expected type %s, got %s", AssetTypeGitRepo, parsed.Type)
	}
	if parsed.Text != "git summary" {
		t.Fatalf("expected text to be preserved")
	}
}

func TestParseAssetReturnsParserConfigurationErrors(t *testing.T) {
	path := "fixtures/sample_resume.pdf"
	tests := []struct {
		name string
		asset models.Asset
		svc  *ParsingService
		want error
	}{
		{
			name: "missing pdf parser",
			asset: models.Asset{Type: AssetTypeResumePDF, URI: &path},
			svc:  NewParsingService(nil, nil, nil, nil),
			want: ErrPDFParserNotConfigured,
		},
		{
			name: "missing docx parser",
			asset: models.Asset{Type: AssetTypeResumeDOCX, URI: &path},
			svc:  NewParsingService(nil, nil, nil, nil),
			want: ErrDOCXParserNotConfigured,
		},
		{
			name: "missing git extractor",
			asset: models.Asset{Type: AssetTypeGitRepo, URI: &path},
			svc:  NewParsingService(nil, nil, nil, nil),
			want: ErrGitExtractorNotConfigured,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.svc.parseAsset(tc.asset)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected error %v, got %v", tc.want, err)
			}
		})
	}
}

func TestParseAssetRequiresURIForFileBackedAssets(t *testing.T) {
	pdfParser := &stubPdfParser{}
	docxParser := &stubDocxParser{}
	gitExtractor := &stubGitExtractor{}
	svc := NewParsingService(nil, pdfParser, docxParser, gitExtractor)

	tests := []models.Asset{
		{Type: AssetTypeResumePDF},
		{Type: AssetTypeResumeDOCX},
		{Type: AssetTypeGitRepo},
	}

	for _, asset := range tests {
		_, err := svc.parseAsset(asset)
		if !errors.Is(err, ErrAssetURIMissing) {
			t.Fatalf("expected ErrAssetURIMissing for type %s, got %v", asset.Type, err)
		}
	}
}

func TestParseAssetReturnsNoteNotImplemented(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)

	_, err := svc.parseAsset(models.Asset{Type: AssetTypeNote})
	if !errors.Is(err, ErrNoteParsingNotImplemented) {
		t.Fatalf("expected ErrNoteParsingNotImplemented, got %v", err)
	}
}

func TestParseAssetReturnsUnsupportedType(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)

	_, err := svc.parseAsset(models.Asset{Type: "spreadsheet"})
	if !errors.Is(err, ErrUnsupportedAssetType) {
		t.Fatalf("expected ErrUnsupportedAssetType, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "spreadsheet") {
		t.Fatalf("expected unsupported type to appear in error, got %v", err)
	}
}

func TestParseAssetReturnsSkippedForResumeImage(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)

	_, err := svc.parseAsset(models.Asset{Type: AssetTypeResumeImage})
	if !errors.Is(err, ErrAssetTypeSkipped) {
		t.Fatalf("expected ErrAssetTypeSkipped, got %v", err)
	}
}
