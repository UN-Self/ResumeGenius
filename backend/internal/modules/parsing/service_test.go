package parsing

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

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

type stubDraftGenerator struct {
	calledWith string
	result     string
	err        error
}

func (s *stubDraftGenerator) GenerateHTML(parsedText string) (string, error) {
	s.calledWith = parsedText
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
	if svc.projectExists == nil {
		t.Error("expected projectExists loader to be initialized")
	}
	if svc.listProjectAssets == nil {
		t.Error("expected listProjectAssets loader to be initialized")
	}
}

func TestNewParsingServiceWithGeneratorStoresGenerator(t *testing.T) {
	generator := &stubDraftGenerator{}

	svc := NewParsingServiceWithGenerator(nil, nil, nil, nil, generator)

	if svc.generator != generator {
		t.Error("expected draft generator to be stored")
	}
}

func TestNewParsingServiceWithGeneratorAndStorageStoresStorage(t *testing.T) {
	store := newTestStorage(t)

	svc := NewParsingServiceWithGeneratorAndStorage(nil, nil, nil, nil, nil, store)

	if svc.storage != store {
		t.Error("expected storage to be stored")
	}
}

func TestParseReturnsProjectNotFound(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)
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

func TestParseForUserReturnsProjectNotFoundWhenOwnershipCheckFails(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "owner-user",
		Title:  "Owned project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	svc := NewParsingService(db, nil, nil, nil)

	_, err := svc.ParseForUser("another-user", project.ID)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestParseReturnsNoUsableAssets(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)
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
	svc := NewParsingService(nil, pdfParser, docxParser, nil)
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
	if parsedContents[0].Label != "sample_resume.pdf" {
		t.Fatalf("expected pdf label sample_resume.pdf, got %q", parsedContents[0].Label)
	}
	expectedNoteText := "求职方向\n希望突出 React、TypeScript 和工程化经验"
	if parsedContents[1].AssetID != 12 || parsedContents[1].Type != AssetTypeNote || parsedContents[1].Text != expectedNoteText {
		t.Fatalf("unexpected note parsed content: %+v", parsedContents[1])
	}
	if parsedContents[1].Label != noteLabel {
		t.Fatalf("expected note label %q, got %q", noteLabel, parsedContents[1].Label)
	}
	if parsedContents[2].AssetID != 15 || parsedContents[2].Type != AssetTypeResumeDOCX || parsedContents[2].Text != "docx text" {
		t.Fatalf("unexpected docx parsed content: %+v", parsedContents[2])
	}
	if parsedContents[2].Label != "sample_resume.docx" {
		t.Fatalf("expected docx label sample_resume.docx, got %q", parsedContents[2].Label)
	}
}

func TestParseIncludesGitAssetsWhenExtractorConfigured(t *testing.T) {
	gitURL := "https://github.com/example/project"
	gitExtractor := &stubGitExtractor{
		result: &ParsedContent{Text: "Repository: project\n\nTech stack: Go, React"},
	}
	svc := NewParsingService(nil, nil, nil, gitExtractor)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 31, Type: AssetTypeGitRepo, URI: &gitURL},
		}, nil
	}

	parsedContents, err := svc.Parse(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitExtractor.calledWith != gitURL {
		t.Fatalf("expected git extractor url %s, got %s", gitURL, gitExtractor.calledWith)
	}
	if len(parsedContents) != 1 {
		t.Fatalf("expected 1 parsed content, got %d", len(parsedContents))
	}
	if parsedContents[0].Type != AssetTypeGitRepo {
		t.Fatalf("expected git repo parsed content, got %+v", parsedContents[0])
	}
	if parsedContents[0].Label != "project" {
		t.Fatalf("expected git label project, got %q", parsedContents[0].Label)
	}
	if !strings.Contains(parsedContents[0].Text, "Tech stack: Go, React") {
		t.Fatalf("expected git summary text, got %q", parsedContents[0].Text)
	}
}

func TestParsePropagatesAssetParsingErrors(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)
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

func TestParseSkipsRecoverableAssetErrorsWhenOtherAssetsAreUsable(t *testing.T) {
	pdfPath := "fixtures/sample_resume.pdf"
	gitURL := "https://github.com/example/project"
	pdfParser := &stubPdfParser{
		result: &ParsedContent{Text: "pdf text"},
	}
	gitExtractor := &stubGitExtractor{
		err: errors.New("clone failed"),
	}
	svc := NewParsingService(nil, pdfParser, nil, gitExtractor)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeNote},
			{ID: 2, Type: AssetTypeResumePDF, URI: &pdfPath},
			{ID: 3, Type: AssetTypeGitRepo, URI: &gitURL},
		}, nil
	}

	parsedContents, err := svc.Parse(21)
	if err != nil {
		t.Fatalf("expected parse to succeed with usable assets, got %v", err)
	}
	if len(parsedContents) != 1 {
		t.Fatalf("expected 1 usable parsed content, got %d", len(parsedContents))
	}
	if parsedContents[0].Type != AssetTypeResumePDF {
		t.Fatalf("expected surviving parsed content to be pdf, got %+v", parsedContents[0])
	}
	if gitExtractor.calledWith != gitURL {
		t.Fatalf("expected git extractor to be attempted with %s, got %s", gitURL, gitExtractor.calledWith)
	}
}

func TestParsePersistsCleanedContentForFileAndGitAssets(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Persist parsed assets",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	pdfURI := "fixtures/sample_resume.pdf"
	docxURI := "fixtures/sample_resume.docx"
	gitURI := "https://github.com/example/project"
	noteContent := "原始备注正文"
	noteLabel := "备注标题"

	pdfAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumePDF, URI: &pdfURI}
	docxAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumeDOCX, URI: &docxURI}
	gitAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeGitRepo, URI: &gitURI}
	noteAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeNote, Content: &noteContent, Label: &noteLabel}
	if err := db.Create(&pdfAsset).Error; err != nil {
		t.Fatalf("create pdf asset: %v", err)
	}
	if err := db.Create(&docxAsset).Error; err != nil {
		t.Fatalf("create docx asset: %v", err)
	}
	if err := db.Create(&gitAsset).Error; err != nil {
		t.Fatalf("create git asset: %v", err)
	}
	if err := db.Create(&noteAsset).Error; err != nil {
		t.Fatalf("create note asset: %v", err)
	}

	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "Page 1 of 2\n• React   工程化",
			Images: []ParsedImage{
				{Description: "头像", DataBase64: "abc"},
			},
		},
	}
	docxParser := &stubDocxParser{
		result: &ParsedContent{Text: "\n工作经历\n\n\nABC   科技"},
	}
	gitExtractor := &stubGitExtractor{
		result: &ParsedContent{Text: "README:\n----\nVite   React"},
	}

	svc := NewParsingService(db, pdfParser, docxParser, gitExtractor)

	parsedContents, err := svc.Parse(project.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsedContents) != 4 {
		t.Fatalf("expected 4 parsed contents, got %d", len(parsedContents))
	}

	assertPersistedAssetContent(t, db, pdfAsset.ID, "- React 工程化", "success", true, 1)
	assertPersistedAssetContent(t, db, docxAsset.ID, "工作经历\n\nABC 科技", "success", true, 0)
	assertPersistedAssetContent(t, db, gitAsset.ID, "README:\nVite React", "success", true, 0)

	var storedNote models.Asset
	if err := db.First(&storedNote, noteAsset.ID).Error; err != nil {
		t.Fatalf("fetch note asset: %v", err)
	}
	if storedNote.Content == nil || *storedNote.Content != noteContent {
		t.Fatalf("expected note content to remain unchanged, got %+v", storedNote.Content)
	}
	assertParsingStatus(t, storedNote.Metadata, "success")
}

func TestParseMarksFailedAndSkippedAssetsInMetadata(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Parse statuses",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	brokenPDFURI := "fixtures/sample_resume.pdf"
	imageURI := "uploads/1/avatar.png"
	noteContent := "可用备注"

	brokenPDF := models.Asset{ProjectID: project.ID, Type: AssetTypeResumePDF, URI: &brokenPDFURI}
	imageAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumeImage, URI: &imageURI}
	noteAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeNote, Content: &noteContent}
	if err := db.Create(&brokenPDF).Error; err != nil {
		t.Fatalf("create broken pdf asset: %v", err)
	}
	if err := db.Create(&imageAsset).Error; err != nil {
		t.Fatalf("create image asset: %v", err)
	}
	if err := db.Create(&noteAsset).Error; err != nil {
		t.Fatalf("create note asset: %v", err)
	}

	pdfParser := &stubPdfParser{err: errors.New("broken pdf")}
	svc := NewParsingService(db, pdfParser, nil, nil)

	parsedContents, err := svc.Parse(project.ID)
	if err != nil {
		t.Fatalf("expected parse to succeed with surviving note asset, got %v", err)
	}
	if len(parsedContents) != 1 || parsedContents[0].Type != AssetTypeNote {
		t.Fatalf("expected only note parsed content to survive, got %+v", parsedContents)
	}

	var storedBrokenPDF models.Asset
	if err := db.First(&storedBrokenPDF, brokenPDF.ID).Error; err != nil {
		t.Fatalf("fetch broken pdf asset: %v", err)
	}
	assertParsingStatus(t, storedBrokenPDF.Metadata, "failed")

	var storedImage models.Asset
	if err := db.First(&storedImage, imageAsset.ID).Error; err != nil {
		t.Fatalf("fetch image asset: %v", err)
	}
	assertParsingStatus(t, storedImage.Metadata, "skipped")
}

func TestParsePersistsDerivedImageAssetsForParsedImages(t *testing.T) {
	db := setupParsingTestDB(t)
	store := newTestStorage(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Persist derived images",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	pdfURI := "fixtures/sample_resume.pdf"
	pdfAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumePDF, URI: &pdfURI}
	if err := db.Create(&pdfAsset).Error; err != nil {
		t.Fatalf("create pdf asset: %v", err)
	}

	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "Page 1 of 1\n* React   engineering",
			Images: []ParsedImage{
				{
					Description: "avatar",
					DataBase64:  base64.StdEncoding.EncodeToString([]byte("avatar-bytes")),
				},
				{
					Description: "badge",
					DataBase64:  base64.StdEncoding.EncodeToString([]byte("badge-bytes")),
				},
			},
		},
	}

	svc := NewParsingServiceWithGeneratorAndStorage(db, pdfParser, nil, nil, nil, store)

	parsedContents, err := svc.Parse(project.ID)
	if err != nil {
		t.Fatalf("parse project: %v", err)
	}
	if len(parsedContents) != 1 {
		t.Fatalf("expected 1 parsed content, got %d", len(parsedContents))
	}

	var storedPDF models.Asset
	if err := db.First(&storedPDF, pdfAsset.ID).Error; err != nil {
		t.Fatalf("fetch source pdf asset: %v", err)
	}

	parsing := getParsingMetadataMap(t, storedPDF.Metadata)
	if got := parsing["images_persisted"]; got != true {
		t.Fatalf("expected images_persisted true, got %+v", got)
	}

	derivedIDs := derivedImageAssetIDsFromMetadata(storedPDF.Metadata)
	if len(derivedIDs) != 2 {
		t.Fatalf("expected 2 derived image ids, got %+v", derivedIDs)
	}

	var derivedAssets []models.Asset
	if err := db.Where("project_id = ? AND id IN ?", project.ID, derivedIDs).Order("id asc").Find(&derivedAssets).Error; err != nil {
		t.Fatalf("fetch derived image assets: %v", err)
	}
	if len(derivedAssets) != 2 {
		t.Fatalf("expected 2 derived image assets, got %d", len(derivedAssets))
	}

	for _, derivedAsset := range derivedAssets {
		if derivedAsset.Type != AssetTypeResumeImage {
			t.Fatalf("expected derived asset type %s, got %s", AssetTypeResumeImage, derivedAsset.Type)
		}
		if derivedAsset.URI == nil || strings.TrimSpace(*derivedAsset.URI) == "" {
			t.Fatalf("expected derived asset URI, got %+v", derivedAsset.URI)
		}
		if !store.Exists(*derivedAsset.URI) {
			t.Fatalf("expected persisted image file %q to exist", *derivedAsset.URI)
		}
		if derivedAsset.Label == nil || strings.TrimSpace(*derivedAsset.Label) == "" {
			t.Fatalf("expected derived asset label, got %+v", derivedAsset.Label)
		}

		derivedParsing := getParsingMetadataMap(t, derivedAsset.Metadata)
		if got := derivedParsing["derived"]; got != true {
			t.Fatalf("expected derived flag true, got %+v", got)
		}
		if got := derivedParsing["source_asset_type"]; got != AssetTypeResumePDF {
			t.Fatalf("expected source asset type %s, got %+v", AssetTypeResumePDF, got)
		}
		if got := derivedParsing["source_asset_id"]; got != float64(pdfAsset.ID) {
			t.Fatalf("expected source asset id %d, got %+v", pdfAsset.ID, got)
		}
	}

	assertPersistedAssetContent(t, db, pdfAsset.ID, "- React engineering", "success", true, 2)
}

func TestParseRetainsDerivedImageAssetsAfterSourceDeletionFallback(t *testing.T) {
	db := setupParsingTestDB(t)
	store := newTestStorage(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Replace derived images",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	pdfURI := "fixtures/sample_resume.pdf"
	pdfAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumePDF, URI: &pdfURI}
	if err := db.Create(&pdfAsset).Error; err != nil {
		t.Fatalf("create pdf asset: %v", err)
	}

	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "first version",
			Images: []ParsedImage{
				{Description: "old", DataBase64: base64.StdEncoding.EncodeToString([]byte("old-image"))},
			},
		},
	}
	svc := NewParsingServiceWithGeneratorAndStorage(db, pdfParser, nil, nil, nil, store)

	if _, err := svc.Parse(project.ID); err != nil {
		t.Fatalf("first parse: %v", err)
	}

	var firstStoredPDF models.Asset
	if err := db.First(&firstStoredPDF, pdfAsset.ID).Error; err != nil {
		t.Fatalf("fetch source asset after first parse: %v", err)
	}

	firstIDs := derivedImageAssetIDsFromMetadata(firstStoredPDF.Metadata)
	if len(firstIDs) != 1 {
		t.Fatalf("expected 1 first derived image id, got %+v", firstIDs)
	}

	var firstDerived models.Asset
	if err := db.First(&firstDerived, firstIDs[0]).Error; err != nil {
		t.Fatalf("fetch first derived asset: %v", err)
	}
	if firstDerived.URI == nil {
		t.Fatal("expected first derived asset URI")
	}
	firstKey := *firstDerived.URI
	if !store.Exists(firstKey) {
		t.Fatalf("expected first derived image file %q to exist", firstKey)
	}

	pdfParser.result = &ParsedContent{
		Text: "second version",
		Images: []ParsedImage{
			{Description: "new", DataBase64: base64.StdEncoding.EncodeToString([]byte("new-image"))},
		},
	}

	if _, err := svc.Parse(project.ID); err != nil {
		t.Fatalf("second parse: %v", err)
	}

	var secondStoredPDF models.Asset
	if err := db.First(&secondStoredPDF, pdfAsset.ID).Error; err != nil {
		t.Fatalf("fetch source asset after second parse: %v", err)
	}

	secondIDs := derivedImageAssetIDsFromMetadata(secondStoredPDF.Metadata)
	if len(secondIDs) != 1 {
		t.Fatalf("expected 1 second derived image id, got %+v", secondIDs)
	}
	if secondIDs[0] != firstIDs[0] {
		t.Fatalf("expected derived image id to stay stable after persisted-text fallback, first=%d second=%d", firstIDs[0], secondIDs[0])
	}

	var derivedCount int64
	if err := db.Model(&models.Asset{}).Where("project_id = ? AND type = ?", project.ID, AssetTypeResumeImage).Count(&derivedCount).Error; err != nil {
		t.Fatalf("count derived assets: %v", err)
	}
	if derivedCount != 1 {
		t.Fatalf("expected 1 derived image asset after fallback parse, got %d", derivedCount)
	}

	var secondDerived models.Asset
	if err := db.First(&secondDerived, secondIDs[0]).Error; err != nil {
		t.Fatalf("fetch second derived asset: %v", err)
	}
	if secondDerived.URI == nil || !store.Exists(*secondDerived.URI) {
		t.Fatalf("expected existing derived image file to remain available, got %+v", secondDerived.URI)
	}
	if secondDerived.URI == nil || *secondDerived.URI != firstKey {
		t.Fatalf("expected derived image file key to stay stable, first=%q second=%+v", firstKey, secondDerived.URI)
	}
}

func TestParseDeletesOriginalSourceFileAfterPersistence(t *testing.T) {
	db := setupParsingTestDB(t)
	store := newTestStorage(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Delete original source",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	originalKey, err := store.Save(project.ID, "resume.pdf", []byte("pdf-bytes"))
	if err != nil {
		t.Fatalf("save original source file: %v", err)
	}

	pdfAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumePDF, URI: &originalKey}
	if err := db.Create(&pdfAsset).Error; err != nil {
		t.Fatalf("create pdf asset: %v", err)
	}

	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "Page 1 of 1\n* React   engineering",
			Images: []ParsedImage{
				{Description: "avatar", DataBase64: base64.StdEncoding.EncodeToString([]byte("avatar-bytes"))},
			},
		},
	}

	svc := NewParsingServiceWithGeneratorAndStorage(db, pdfParser, nil, nil, nil, store)

	parsedContents, err := svc.Parse(project.ID)
	if err != nil {
		t.Fatalf("parse project: %v", err)
	}
	if len(parsedContents) != 1 {
		t.Fatalf("expected 1 parsed content, got %d", len(parsedContents))
	}
	if store.Exists(originalKey) {
		t.Fatalf("expected original source file %q to be deleted", originalKey)
	}

	var storedPDF models.Asset
	if err := db.First(&storedPDF, pdfAsset.ID).Error; err != nil {
		t.Fatalf("fetch stored pdf asset: %v", err)
	}
	if storedPDF.URI != nil {
		t.Fatalf("expected source asset uri to be cleared after deletion, got %+v", storedPDF.URI)
	}

	parsing := getParsingMetadataMap(t, storedPDF.Metadata)
	if got := parsing["source_deleted"]; got != true {
		t.Fatalf("expected source_deleted true, got %+v", got)
	}
	if got := parsing["original_asset_type"]; got != AssetTypeResumePDF {
		t.Fatalf("expected original_asset_type %s, got %+v", AssetTypeResumePDF, got)
	}
	if got := parsing["original_filename"]; got != "resume.pdf" {
		t.Fatalf("expected original_filename resume.pdf, got %+v", got)
	}
	if got := parsing["original_uri"]; got != originalKey {
		t.Fatalf("expected original_uri %q, got %+v", originalKey, got)
	}

	derivedIDs := derivedImageAssetIDsFromMetadata(storedPDF.Metadata)
	if len(derivedIDs) != 1 {
		t.Fatalf("expected 1 derived image asset id, got %+v", derivedIDs)
	}

	var derivedAsset models.Asset
	if err := db.First(&derivedAsset, derivedIDs[0]).Error; err != nil {
		t.Fatalf("fetch derived image asset: %v", err)
	}
	if derivedAsset.URI == nil || !store.Exists(*derivedAsset.URI) {
		t.Fatalf("expected derived image file to remain available, got %+v", derivedAsset.URI)
	}

	pdfParser.calledWith = ""
	parsedContents, err = svc.Parse(project.ID)
	if err != nil {
		t.Fatalf("parse deleted-source project via persisted fallback: %v", err)
	}
	if len(parsedContents) != 1 {
		t.Fatalf("expected 1 parsed content after fallback, got %d", len(parsedContents))
	}
	if pdfParser.calledWith != "" {
		t.Fatalf("expected parser not to be called after source deletion, got %q", pdfParser.calledWith)
	}
	if parsedContents[0].Text != "- React engineering" {
		t.Fatalf("expected persisted cleaned text, got %q", parsedContents[0].Text)
	}
}

func TestParseKeepsOriginalSourceFileWhenImagePersistenceFails(t *testing.T) {
	db := setupParsingTestDB(t)
	store := newTestStorage(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Keep original source on image persistence failure",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	originalKey, err := store.Save(project.ID, "resume.pdf", []byte("pdf-bytes"))
	if err != nil {
		t.Fatalf("save original source file: %v", err)
	}

	pdfAsset := models.Asset{ProjectID: project.ID, Type: AssetTypeResumePDF, URI: &originalKey}
	if err := db.Create(&pdfAsset).Error; err != nil {
		t.Fatalf("create pdf asset: %v", err)
	}

	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "React engineering",
			Images: []ParsedImage{
				{Description: "broken", DataBase64: "not-base64"},
			},
		},
	}

	svc := NewParsingServiceWithGeneratorAndStorage(db, pdfParser, nil, nil, nil, store)

	_, err = svc.Parse(project.ID)
	if err == nil {
		t.Fatal("expected parse to fail when derived image persistence fails")
	}
	if !store.Exists(originalKey) {
		t.Fatalf("expected original source file %q to remain when persistence fails", originalKey)
	}

	var storedPDF models.Asset
	if err := db.First(&storedPDF, pdfAsset.ID).Error; err != nil {
		t.Fatalf("fetch stored pdf asset: %v", err)
	}
	if storedPDF.URI == nil || *storedPDF.URI != originalKey {
		t.Fatalf("expected source asset uri to remain %q, got %+v", originalKey, storedPDF.URI)
	}
	if storedPDF.Content != nil {
		t.Fatalf("expected content not to be persisted on failure, got %+v", storedPDF.Content)
	}
	if storedPDF.Metadata != nil {
		t.Fatalf("expected metadata to remain unchanged on failure, got %+v", storedPDF.Metadata)
	}
}

func TestGenerateCreatesDraftVersionAndUpdatesCurrentDraftID(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Test project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	noteContent := "Focus on React and TypeScript experience"
	noteLabel := "Target role"
	generator := &stubDraftGenerator{
		result: "<!DOCTYPE html><html><body>mock resume</body></html>",
	}
	svc := NewParsingServiceWithGenerator(db, nil, nil, nil, generator)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeNote, Content: &noteContent, Label: &noteLabel},
		}, nil
	}

	result, err := svc.Generate(project.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected generate result")
	}
	if result.DraftID == 0 {
		t.Fatal("expected non-zero draft id")
	}
	if result.VersionID == 0 {
		t.Fatal("expected non-zero version id")
	}
	if result.HTMLContent != generator.result {
		t.Fatalf("expected html content to match generator output, got %q", result.HTMLContent)
	}
	if !strings.Contains(generator.calledWith, "Target role\nFocus on React and TypeScript experience") {
		t.Fatalf("expected aggregated parsed text to be passed to generator, got %q", generator.calledWith)
	}

	var draft models.Draft
	if err := db.First(&draft, result.DraftID).Error; err != nil {
		t.Fatalf("fetch draft: %v", err)
	}
	if draft.HTMLContent != generator.result {
		t.Fatalf("expected stored draft html to match generator output, got %q", draft.HTMLContent)
	}

	var version models.Version
	if err := db.First(&version, result.VersionID).Error; err != nil {
		t.Fatalf("fetch version: %v", err)
	}
	if version.DraftID != draft.ID {
		t.Fatalf("expected version draft_id %d, got %d", draft.ID, version.DraftID)
	}
	if version.HTMLSnapshot != generator.result {
		t.Fatalf("expected version snapshot to match generator output, got %q", version.HTMLSnapshot)
	}
	if version.Label == nil || *version.Label != "AI 初始生成" {
		t.Fatalf("expected version label %q, got %+v", "AI 初始生成", version.Label)
	}

	var updatedProject models.Project
	if err := db.First(&updatedProject, project.ID).Error; err != nil {
		t.Fatalf("fetch project: %v", err)
	}
	if updatedProject.CurrentDraftID == nil || *updatedProject.CurrentDraftID != draft.ID {
		t.Fatalf("expected current_draft_id to point at draft %d, got %+v", draft.ID, updatedProject.CurrentDraftID)
	}
}

func TestGenerateForUserReturnsProjectNotFoundWhenOwnershipCheckFails(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "owner-user",
		Title:  "Owned project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	generator := &stubDraftGenerator{
		result: "<!DOCTYPE html><html><body>mock resume</body></html>",
	}
	svc := NewParsingServiceWithGenerator(db, nil, nil, nil, generator)

	_, err := svc.GenerateForUser("another-user", project.ID)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestGenerateAggregatesGitAssetTextWhenExtractorConfigured(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Test project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	gitURL := "https://github.com/example/project"
	gitExtractor := &stubGitExtractor{
		result: &ParsedContent{Text: "Repository: project\n\nREADME:\nResume generator"},
	}
	generator := &stubDraftGenerator{
		result: "<!DOCTYPE html><html><body>mock resume</body></html>",
	}
	svc := NewParsingServiceWithGenerator(db, nil, nil, gitExtractor, generator)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeGitRepo, URI: &gitURL},
		}, nil
	}

	result, err := svc.Generate(project.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.DraftID == 0 {
		t.Fatalf("expected generated draft result, got %+v", result)
	}
	if gitExtractor.calledWith != gitURL {
		t.Fatalf("expected git extractor url %s, got %s", gitURL, gitExtractor.calledWith)
	}
	if !strings.Contains(generator.calledWith, "Repository: project") {
		t.Fatalf("expected generator input to include git summary, got %q", generator.calledWith)
	}
	if !strings.Contains(generator.calledWith, "README:\nResume generator") {
		t.Fatalf("expected generator input to include git readme summary, got %q", generator.calledWith)
	}
}

func TestGenerateRollsBackWhenVersionCreationFails(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Test project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	noteContent := "Focus on React and TypeScript experience"
	generator := &stubDraftGenerator{
		result: "<!DOCTYPE html><html><body>mock resume</body></html>",
	}
	svc := NewParsingServiceWithGenerator(db, nil, nil, nil, generator)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeNote, Content: &noteContent},
		}, nil
	}

	const callbackName = "parsing:test_fail_version_create"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil {
			return
		}
		if tx.Statement.Schema.Name == "Version" {
			tx.AddError(errors.New("forced version insert failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(callbackName)
	})

	_, err := svc.Generate(project.ID)
	if err == nil || !strings.Contains(err.Error(), "forced version insert failure") {
		t.Fatalf("expected forced version insert failure, got %v", err)
	}

	var draftCount int64
	if err := db.Model(&models.Draft{}).Where("project_id = ?", project.ID).Count(&draftCount).Error; err != nil {
		t.Fatalf("count drafts: %v", err)
	}
	if draftCount != 0 {
		t.Fatalf("expected no drafts to remain after rollback, got %d", draftCount)
	}

	var versionCount int64
	if err := db.Model(&models.Version{}).Count(&versionCount).Error; err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if versionCount != 0 {
		t.Fatalf("expected no versions to remain after rollback, got %d", versionCount)
	}

	var updatedProject models.Project
	if err := db.First(&updatedProject, project.ID).Error; err != nil {
		t.Fatalf("fetch project: %v", err)
	}
	if updatedProject.CurrentDraftID != nil {
		t.Fatalf("expected current_draft_id to remain nil, got %+v", updatedProject.CurrentDraftID)
	}
}

func TestGenerateDoesNotCreateDirtyDraftWhenGeneratorFails(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Test project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	noteContent := "Focus on React and TypeScript experience"
	generator := &stubDraftGenerator{err: errors.New("mock failed")}
	svc := NewParsingServiceWithGenerator(db, nil, nil, nil, generator)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeNote, Content: &noteContent},
		}, nil
	}

	_, err := svc.Generate(project.ID)
	if !errors.Is(err, ErrAIGenerateFailed) {
		t.Fatalf("expected ErrAIGenerateFailed, got %v", err)
	}

	var draftCount int64
	if err := db.Model(&models.Draft{}).Where("project_id = ?", project.ID).Count(&draftCount).Error; err != nil {
		t.Fatalf("count drafts: %v", err)
	}
	if draftCount != 0 {
		t.Fatalf("expected no drafts to be created, got %d", draftCount)
	}

	var updatedProject models.Project
	if err := db.First(&updatedProject, project.ID).Error; err != nil {
		t.Fatalf("fetch project: %v", err)
	}
	if updatedProject.CurrentDraftID != nil {
		t.Fatalf("expected current_draft_id to remain nil, got %+v", updatedProject.CurrentDraftID)
	}
}

func TestGenerateReturnsDatabaseNotConfiguredWhenDBMissing(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)

	_, err := svc.Generate(1)
	if !errors.Is(err, ErrDatabaseNotConfigured) {
		t.Fatalf("expected ErrDatabaseNotConfigured before DB-backed generate flow, got %v", err)
	}
}

func TestGenerateReturnsDraftGeneratorNotConfiguredWhenMissing(t *testing.T) {
	db := setupParsingTestDB(t)
	svc := NewParsingService(db, nil, nil, nil)

	_, err := svc.Generate(1)
	if !errors.Is(err, ErrDraftGeneratorNotConfigured) {
		t.Fatalf("expected ErrDraftGeneratorNotConfigured, got %v", err)
	}
}

func TestGenerateReturnsNoGeneratableTextWhenParsedContentsHaveNoText(t *testing.T) {
	db := setupParsingTestDB(t)
	project := models.Project{
		UserID: "test-user-1",
		Title:  "Empty text project",
		Status: "active",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	docxPath := "fixtures/sample_resume.docx"
	docxParser := &stubDocxParser{
		result: &ParsedContent{Text: "   "},
	}
	generator := &stubDraftGenerator{
		result: "<!DOCTYPE html><html><body>mock resume</body></html>",
	}
	svc := NewParsingServiceWithGenerator(db, nil, docxParser, nil, generator)
	svc.projectExists = func(projectID uint) (bool, error) {
		return true, nil
	}
	svc.listProjectAssets = func(projectID uint) ([]models.Asset, error) {
		return []models.Asset{
			{ID: 1, Type: AssetTypeResumeDOCX, URI: &docxPath},
		}, nil
	}

	_, err := svc.Generate(project.ID)
	if !errors.Is(err, ErrNoGeneratableText) {
		t.Fatalf("expected ErrNoGeneratableText, got %v", err)
	}
	if generator.calledWith != "" {
		t.Fatalf("expected generator not to be called when no text is available, got %q", generator.calledWith)
	}

	var draftCount int64
	if err := db.Model(&models.Draft{}).Where("project_id = ?", project.ID).Count(&draftCount).Error; err != nil {
		t.Fatalf("count drafts: %v", err)
	}
	if draftCount != 0 {
		t.Fatalf("expected no drafts to be created, got %d", draftCount)
	}
}

func TestAggregateParsedContents(t *testing.T) {
	parsedContents := []ParsedContent{
		{Text: " first "},
		{Text: ""},
		{Text: "second"},
	}

	aggregated := aggregateParsedContents(parsedContents)
	expected := "first\n\nsecond"
	if aggregated != expected {
		t.Fatalf("expected %q, got %q", expected, aggregated)
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
	if parsed.Label != "sample_resume.pdf" {
		t.Fatalf("expected label sample_resume.pdf, got %q", parsed.Label)
	}
	if parsed.Text != "pdf text" {
		t.Fatalf("expected text to be preserved")
	}
	if len(parsed.Images) != 1 || parsed.Images[0].Description != "头像" {
		t.Fatalf("expected image metadata to be preserved")
	}
}

func TestParseAssetWrapsPDFParserErrors(t *testing.T) {
	path := "fixtures/sample_resume.pdf"
	expected := errors.New("broken pdf")
	svc := NewParsingService(nil, &stubPdfParser{err: expected}, nil, nil)

	_, err := svc.parseAsset(models.Asset{
		Type: AssetTypeResumePDF,
		URI:  &path,
	})
	if !errors.Is(err, ErrPDFParseFailed) {
		t.Fatalf("expected ErrPDFParseFailed, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), expected.Error()) {
		t.Fatalf("expected wrapped error to include original message, got %v", err)
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
	if parsed.Label != "sample_resume.docx" {
		t.Fatalf("expected label sample_resume.docx, got %q", parsed.Label)
	}
	if parsed.Text != "docx text" {
		t.Fatalf("expected text to be preserved")
	}
}

func TestParseAssetWrapsDOCXParserErrors(t *testing.T) {
	path := "fixtures/sample_resume.docx"
	expected := errors.New("broken docx")
	svc := NewParsingService(nil, nil, &stubDocxParser{err: expected}, nil)

	_, err := svc.parseAsset(models.Asset{
		Type: AssetTypeResumeDOCX,
		URI:  &path,
	})
	if !errors.Is(err, ErrDOCXParseFailed) {
		t.Fatalf("expected ErrDOCXParseFailed, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), expected.Error()) {
		t.Fatalf("expected wrapped error to include original message, got %v", err)
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
	if parsed.Label != "project" {
		t.Fatalf("expected label project, got %q", parsed.Label)
	}
	if parsed.Text != "git summary" {
		t.Fatalf("expected text to be preserved")
	}
}

func TestParseAssetWrapsGitExtractorErrors(t *testing.T) {
	repoURL := "https://github.com/example/project"
	expected := errors.New("clone failed")
	svc := NewParsingService(nil, nil, nil, &stubGitExtractor{err: expected})

	_, err := svc.parseAsset(models.Asset{
		Type: AssetTypeGitRepo,
		URI:  &repoURL,
	})
	if !errors.Is(err, ErrGitExtractFailed) {
		t.Fatalf("expected ErrGitExtractFailed, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), expected.Error()) {
		t.Fatalf("expected wrapped error to include original message, got %v", err)
	}
}

func TestParseAssetReturnsParserConfigurationErrors(t *testing.T) {
	path := "fixtures/sample_resume.pdf"
	tests := []struct {
		name  string
		asset models.Asset
		svc   *ParsingService
		want  error
	}{
		{
			name:  "missing pdf parser",
			asset: models.Asset{Type: AssetTypeResumePDF, URI: &path},
			svc:   NewParsingService(nil, nil, nil, nil),
			want:  ErrPDFParserNotConfigured,
		},
		{
			name:  "missing docx parser",
			asset: models.Asset{Type: AssetTypeResumeDOCX, URI: &path},
			svc:   NewParsingService(nil, nil, nil, nil),
			want:  ErrDOCXParserNotConfigured,
		},
		{
			name:  "missing git extractor",
			asset: models.Asset{Type: AssetTypeGitRepo, URI: &path},
			svc:   NewParsingService(nil, nil, nil, nil),
			want:  ErrGitExtractorNotConfigured,
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

func TestParseAssetBuildsParsedContentForNote(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)
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
	if parsed.Label != label {
		t.Fatalf("expected label %q, got %q", label, parsed.Label)
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
	svc := NewParsingService(nil, nil, nil, nil)
	content := "熟悉 React 组件设计和前端工程化。"

	parsed, err := svc.parseAsset(models.Asset{
		ID:      45,
		Type:    AssetTypeNote,
		Content: &content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Label != "" {
		t.Fatalf("expected empty label, got %q", parsed.Label)
	}
	if parsed.Text != content {
		t.Fatalf("expected note content to be preserved, got %q", parsed.Text)
	}
}

func TestParseAssetCleansNoteContent(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)
	content := "\n• React   工程化\n----\n第 2 页 / 共 3 页\n\nTypeScript  组件设计\n"
	label := "求职方向"

	parsed, err := svc.parseAsset(models.Asset{
		ID:      46,
		Type:    AssetTypeNote,
		Content: &content,
		Label:   &label,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedText := "求职方向\n- React 工程化\n\nTypeScript 组件设计"
	if parsed.Text != expectedText {
		t.Fatalf("expected cleaned note text %q, got %q", expectedText, parsed.Text)
	}
}

func TestParseAssetCleansParserOutputText(t *testing.T) {
	path := "fixtures/sample_resume.pdf"
	pdfParser := &stubPdfParser{
		result: &ParsedContent{
			Text: "Page 1 of 2\n• React   工程化\n----\n量化结果",
		},
	}
	svc := NewParsingService(nil, pdfParser, nil, nil)

	parsed, err := svc.parseAsset(models.Asset{
		ID:   47,
		Type: AssetTypeResumePDF,
		URI:  &path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedText := "- React 工程化\n量化结果"
	if parsed.Text != expectedText {
		t.Fatalf("expected cleaned parser text %q, got %q", expectedText, parsed.Text)
	}
}

func TestParseAssetRequiresContentForNote(t *testing.T) {
	svc := NewParsingService(nil, nil, nil, nil)
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

func assertPersistedAssetContent(t *testing.T, db *gorm.DB, assetID uint, wantContent, wantStatus string, wantPersisted bool, wantImageCount int) {
	t.Helper()

	var asset models.Asset
	if err := db.First(&asset, assetID).Error; err != nil {
		t.Fatalf("fetch asset %d: %v", assetID, err)
	}
	if asset.Content == nil || *asset.Content != wantContent {
		t.Fatalf("expected asset %d content %q, got %+v", assetID, wantContent, asset.Content)
	}

	parsing := getParsingMetadataMap(t, asset.Metadata)
	if got := parsing["status"]; got != wantStatus {
		t.Fatalf("expected asset %d parsing.status %q, got %+v", assetID, wantStatus, got)
	}
	if got := parsing["content_persisted"]; got != wantPersisted {
		t.Fatalf("expected asset %d content_persisted %v, got %+v", assetID, wantPersisted, got)
	}
	if got := parsing["image_count"]; got != float64(wantImageCount) {
		t.Fatalf("expected asset %d image_count %d, got %+v", assetID, wantImageCount, got)
	}
}

func assertParsingStatus(t *testing.T, metadata models.JSONB, want string) {
	t.Helper()

	parsing := getParsingMetadataMap(t, metadata)
	if got := parsing["status"]; got != want {
		t.Fatalf("expected parsing.status %q, got %+v", want, got)
	}
}

func getParsingMetadataMap(t *testing.T, metadata models.JSONB) map[string]interface{} {
	t.Helper()

	if metadata == nil {
		t.Fatal("expected metadata to be present")
	}

	rawParsing, ok := metadata["parsing"]
	if !ok {
		t.Fatalf("expected metadata.parsing, got %+v", metadata)
	}

	parsing, ok := rawParsing.(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata.parsing to be a map, got %T", rawParsing)
	}
	return parsing
}
