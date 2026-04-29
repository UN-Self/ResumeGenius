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

	svc := NewParsingService(nil, pdfParser, docxParser, gitExtractor, nil)

	if svc.pdfParser != pdfParser {
		t.Error("expected pdf parser to be stored")
	}
	if svc.docxParser != docxParser {
		t.Error("expected docx parser to be stored")
	}
	if svc.gitExtractor != gitExtractor {
		t.Error("expected git extractor to be stored")
	}
	if svc.projectExists == nil {
		t.Error("expected projectExists loader to be initialized")
	}
	if svc.listProjectAssets == nil {
		t.Error("expected listProjectAssets loader to be initialized")
	}
}

func TestParseReturnsProjectNotFound(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)
	assetsListed := false
	svc.projectExists = func(projectID uint) (bool, error) {
		if projectID != 99 {
			t.Fatalf("expected project id 99, got %d", projectID)
		}
		return false, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		assetsListed = true
		return nil, nil
	}

	_, err := svc.Parse(99)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
	if assetsListed {
		t.Fatal("expected assets not to be loaded when project does not exist")
	}
}

func TestParseReturnsNoUsableAssets(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)
	gitURL := "https://github.com/example/project"
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeResumeImage},
			{ID: 2, Type: AssetTypeGitRepo, URI: &gitURL},
		}, nil
	}

	_, err := svc.Parse(1)
	if !errors.Is(err, ErrNoUsableAssets) {
		t.Fatalf("expected ErrNoUsableAssets, got %v", err)
	}
}

func TestParseAggregatesSupportedAssets(t *testing.T) {
	pdfPath := "fixtures/sample_resume.pdf"
	docxPath := "fixtures/sample_resume.docx"
	gitURL := "https://github.com/example/project"
	noteContent := "希望突出 React、TypeScript 和工程化经验"
	noteLabel := "求职方向"

	pdfParser := &stubPdfParser{
		result: &ParsedContent{Text: "pdf text"},
	}
	docxParser := &stubDocxParser{
		result: &ParsedContent{Text: "docx text"},
	}
	svc := NewParsingService(nil, pdfParser, docxParser, nil, nil)
	svc.projectExists = func(projectID uint) (bool, error) {
		if projectID != 7 {
			t.Fatalf("expected project id 7, got %d", projectID)
		}
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 11, Type: AssetTypeResumePDF, URI: &pdfPath},
			{ID: 12, Type: AssetTypeNote, Content: &noteContent, Label: &noteLabel},
			{ID: 13, Type: AssetTypeResumeImage},
			{ID: 14, Type: AssetTypeGitRepo, URI: &gitURL},
			{ID: 15, Type: AssetTypeResumeDOCX, URI: &docxPath},
		}, nil
	}

	parsedContents, err := svc.Parse(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsedContents) != 3 {
		t.Fatalf("expected 3 parsed contents, got %d", len(parsedContents))
	}
	if pdfParser.calledWith != pdfPath {
		t.Fatalf("expected pdf parser path %s, got %s", pdfPath, pdfParser.calledWith)
	}
	if docxParser.calledWith != docxPath {
		t.Fatalf("expected docx parser path %s, got %s", docxPath, docxParser.calledWith)
	}

	if parsedContents[0].AssetID != 11 || parsedContents[0].Type != AssetTypeResumePDF || parsedContents[0].Text != "pdf text" {
		t.Fatalf("unexpected pdf parsed content: %+v", parsedContents[0])
	}
	expectedNoteText := "求职方向\n希望突出 React、TypeScript 和工程化经验"
	if parsedContents[1].AssetID != 12 || parsedContents[1].Type != AssetTypeNote || parsedContents[1].Text != expectedNoteText {
		t.Fatalf("unexpected note parsed content: %+v", parsedContents[1])
	}
	if parsedContents[2].AssetID != 15 || parsedContents[2].Type != AssetTypeResumeDOCX || parsedContents[2].Text != "docx text" {
		t.Fatalf("unexpected docx parsed content: %+v", parsedContents[2])
	}
}

func TestParsePropagatesAssetParsingErrors(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 21, Type: AssetTypeNote},
		}, nil
	}

	_, err := svc.Parse(21)
	if !errors.Is(err, ErrAssetContentMissing) {
		t.Fatalf("expected ErrAssetContentMissing, got %v", err)
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
	svc := NewParsingService(nil, pdfParser, nil, nil, nil)

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
	svc := NewParsingService(nil, nil, docxParser, nil, nil)

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
	svc := NewParsingService(nil, nil, nil, gitExtractor, nil)

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
			svc:  NewParsingService(nil, nil, nil, nil, nil),
			want: ErrPDFParserNotConfigured,
		},
		{
			name: "missing docx parser",
			asset: models.Asset{Type: AssetTypeResumeDOCX, URI: &path},
			svc:  NewParsingService(nil, nil, nil, nil, nil),
			want: ErrDOCXParserNotConfigured,
		},
		{
			name: "missing git extractor",
			asset: models.Asset{Type: AssetTypeGitRepo, URI: &path},
			svc:  NewParsingService(nil, nil, nil, nil, nil),
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
	svc := NewParsingService(nil, pdfParser, docxParser, gitExtractor, nil)

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

func TestParseAssetBuildsParsedContentForNote(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)
	content := "希望突出 React、TypeScript 和工程化经验"
	label := "求职方向"

	parsed, err := svc.parseAsset(models.Asset{
		ID:      44,
		Type:    AssetTypeNote,
		Content: &content,
		Label:   &label,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed note content")
	}
	if parsed.AssetID != 44 {
		t.Fatalf("expected asset_id 44, got %d", parsed.AssetID)
	}
	if parsed.Type != AssetTypeNote {
		t.Fatalf("expected type %s, got %s", AssetTypeNote, parsed.Type)
	}
	expectedText := "求职方向\n希望突出 React、TypeScript 和工程化经验"
	if parsed.Text != expectedText {
		t.Fatalf("expected text %q, got %q", expectedText, parsed.Text)
	}
	if len(parsed.Images) != 0 {
		t.Fatalf("expected no images for note, got %d", len(parsed.Images))
	}
}

func TestParseAssetBuildsParsedContentForNoteWithoutLabel(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)
	content := "熟悉 React 组件设计和前端工程化。"

	parsed, err := svc.parseAsset(models.Asset{
		ID:      45,
		Type:    AssetTypeNote,
		Content: &content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Text != content {
		t.Fatalf("expected note content to be preserved, got %q", parsed.Text)
	}
}

func TestParseAssetRequiresContentForNote(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)
	blank := "   "

	tests := []models.Asset{
		{Type: AssetTypeNote},
		{Type: AssetTypeNote, Content: &blank},
	}

	for _, asset := range tests {
		_, err := svc.parseAsset(asset)
		if !errors.Is(err, ErrAssetContentMissing) {
			t.Fatalf("expected ErrAssetContentMissing for note, got %v", err)
		}
	}
}

func TestParseAssetReturnsUnsupportedType(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)

	_, err := svc.parseAsset(models.Asset{Type: "spreadsheet"})
	if !errors.Is(err, ErrUnsupportedAssetType) {
		t.Fatalf("expected ErrUnsupportedAssetType, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "spreadsheet") {
		t.Fatalf("expected unsupported type to appear in error, got %v", err)
	}
}

func TestParseAssetReturnsSkippedForResumeImage(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil, nil)

	_, err := svc.parseAsset(models.Asset{Type: AssetTypeResumeImage})
	if !errors.Is(err, ErrAssetTypeSkipped) {
		t.Fatalf("expected ErrAssetTypeSkipped, got %v", err)
	}
}
