package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

// ---------------------------------------------------------------------------
// MockExporter
// ---------------------------------------------------------------------------

// MockExporter implements PDFExporter for testing.
type MockExporter struct {
	PDFBytes []byte
	Err      error
}

// ExportHTMLToPDF returns the configured PDF bytes or error.
func (m *MockExporter) ExportHTMLToPDF(_ string) ([]byte, error) {
	return m.PDFBytes, m.Err
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// samplePDF reads the fixture PDF from the project root.
func samplePDF(t *testing.T) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "..", "..", "fixtures", "sample_resume.pdf")
	data, err := os.ReadFile(p)
	require.NoError(t, err, "failed to read fixture PDF: %s", p)
	return data
}

// newTestExportService creates an ExportService with MockExporter + LocalStorage
// (temp dir) and registers Close cleanup.
func newTestExportService(t *testing.T) (*ExportService, *MockExporter) {
	t.Helper()

	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir)
	mock := &MockExporter{
		PDFBytes: samplePDF(t),
	}

	svc := NewExportService(mock, store)

	t.Cleanup(func() {
		svc.Close()
	})

	return svc, mock
}

// newTestExportServiceWithDB creates an ExportService with DB injected for draft validation.
func newTestExportServiceWithDB(t *testing.T, db *gorm.DB) (*ExportService, *MockExporter) {
	t.Helper()
	svc, mock := newTestExportService(t)
	svc.db = db
	return svc, mock
}

// waitForTask polls a task until it reaches a terminal state or timeout.
func waitForTask(t *testing.T, svc *ExportService, taskID string, timeout time.Duration) *ExportTask {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task, err := svc.GetTask(taskID)
		require.NoError(t, err)
		if task.Status == "completed" || task.Status == "failed" {
			return task
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach terminal state within %v", taskID, timeout)
	return nil
}

// ---------------------------------------------------------------------------
// CreateTask
// ---------------------------------------------------------------------------

func TestCreateTask_ReadsHTMLFromDB(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, mock := newTestExportServiceWithDB(t, db)

	_ = mock // mock exporter is used internally

	// Verify seedDraft HTML is in the DB
	var fetched models.Draft
	require.NoError(t, db.First(&fetched, draft.ID).Error)
	require.Equal(t, "<html><body><h1>Test Resume</h1></body></html>", fetched.HTMLContent)

	taskID, err := svc.CreateTask(draft.ID)
	require.NoError(t, err)

	task := waitForTask(t, svc, taskID, 3*time.Second)
	assert.Equal(t, "completed", task.Status)

	// Verify the mock exporter received HTML wrapped with the template
	stored, _ := svc.tasks.Load(taskID)
	assert.Contains(t, stored.(*ExportTask).htmlContent, "<h1>Test Resume</h1>")
	assert.Contains(t, stored.(*ExportTask).htmlContent, ".resume-page")
}

func TestCreateTask_ReturnsTaskID(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportServiceWithDB(t, db)

	taskID, err := svc.CreateTask(draft.ID)
	require.NoError(t, err)
	require.NotNil(t, taskID)
	assert.True(t, strings.HasPrefix(taskID, "task_"), "task ID should start with 'task_', got: %s", taskID)

	// Verify task is retrievable
	task, err := svc.GetTask(taskID)
	require.NoError(t, err)
	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, draft.ID, task.DraftID)
	assert.Equal(t, "pending", task.Status)
	assert.Equal(t, 0, task.Progress)
}

func TestCreateTask_DraftNotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc, _ := newTestExportServiceWithDB(t, db)

	_, err := svc.CreateTask(99999)
	require.ErrorIs(t, err, ErrDraftNotFound)
}

// ---------------------------------------------------------------------------
// GetTask
// ---------------------------------------------------------------------------

func TestGetTask_NotFound(t *testing.T) {
	svc, _ := newTestExportService(t)

	_, err := svc.GetTask("task_nonexistent")
	require.ErrorIs(t, err, ErrTaskNotFound)
}

// ---------------------------------------------------------------------------
// TaskFlows
// ---------------------------------------------------------------------------

func TestTaskFlows_ToCompleted(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportServiceWithDB(t, db)

	taskID, err := svc.CreateTask(draft.ID)
	require.NoError(t, err)

	task := waitForTask(t, svc, taskID, 3*time.Second)

	assert.Equal(t, "completed", task.Status)
	assert.Equal(t, 100, task.Progress)
	require.NotNil(t, task.DownloadURL)
	assert.Contains(t, *task.DownloadURL, fmt.Sprintf("/api/v1/tasks/%s/file", taskID))
}

func TestTaskFlows_ToFailed(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, mock := newTestExportServiceWithDB(t, db)

	// Configure mock to return an error
	mock.Err = fmt.Errorf("chromedp: headless shell crashed")

	taskID, err := svc.CreateTask(draft.ID)
	require.NoError(t, err)

	task := waitForTask(t, svc, taskID, 3*time.Second)

	assert.Equal(t, "failed", task.Status)
	require.NotNil(t, task.Error)
	assert.Contains(t, *task.Error, "chromedp: headless shell crashed")
}

// ---------------------------------------------------------------------------
// GetFile
// ---------------------------------------------------------------------------

func TestGetFile_CompletedTask(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportServiceWithDB(t, db)

	taskID, err := svc.CreateTask(draft.ID)
	require.NoError(t, err)

	waitForTask(t, svc, taskID, 3*time.Second)

	pdfBytes, err := svc.GetFile(taskID)
	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)
	// Should start with PDF magic bytes: %PDF
	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF"),
		"returned file should be a PDF (starts with %%PDF)")
}

func TestGetFile_TaskNotFound(t *testing.T) {
	svc, _ := newTestExportService(t)

	_, err := svc.GetFile("task_nonexistent")
	require.ErrorIs(t, err, ErrTaskNotFound)
}

func TestNewChromeExporter_RespectsChromeBinEnv(t *testing.T) {
	t.Setenv("CHROME_BIN", "/usr/bin/my-custom-chrome")

	exporter := NewChromeExporter()
	defer exporter.Close()

	assert.NotNil(t, exporter)
	assert.NotNil(t, exporter.allocCtx)
}

func TestGetFile_TaskNotCompleted(t *testing.T) {
	svc, _ := newTestExportService(t)

	// db is nil, CreateTask will return error since DB is required
	_, err := svc.CreateTask(1)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// wrapWithTemplate
// ---------------------------------------------------------------------------

func TestWrapWithTemplate_ReplacesPlaceholder(t *testing.T) {
	result := wrapWithTemplate("<h1>Hello</h1><p>World</p>")

	assert.Contains(t, result, "<h1>Hello</h1><p>World</p>")
	assert.Contains(t, result, `<div class="resume-page resume-document">`)
	assert.Contains(t, result, ".resume-page h1")
	assert.Contains(t, result, "@page")
	assert.Contains(t, result, "box-sizing: border-box")
	assert.Contains(t, result, "white-space: pre-wrap")
	assert.NotContains(t, result, "{{CONTENT}}")
}

func TestWrapWithTemplate_PageMarginInsteadOfDivPadding(t *testing.T) {
	result := wrapWithTemplate("<h1>Hello</h1>")

	// @page rule should carry the margin (not the .resume-page div)
	assert.Contains(t, result, "@page { size: A4; margin: 18mm 20mm; }",
		"@page should specify margin: 18mm 20mm so that every PDF page (including page 2+) gets top margin")
	assert.NotContains(t, result, "padding: 18mm 20mm",
		".resume-page should NOT have padding: 18mm 20mm; margin belongs on @page instead")
	// .resume-page should NOT set explicit width/min-height that overflows @page content area
	assert.NotContains(t, result, "width: 210mm",
		".resume-page should not set width: 210mm — with @page margin the content area is only 170mm wide")
	assert.NotContains(t, result, "min-height: 297mm",
		".resume-page should not set min-height: 297mm — with @page margin the content area is only 261mm tall")
}

func TestWrapWithTemplate_ParagraphMinHeight(t *testing.T) {
	result := wrapWithTemplate("<p></p>")

	assert.Contains(t, result, "min-height: 1.5em",
		".resume-page p should have min-height matching line-height so empty paragraphs match editor canvas height")
}

func TestWrapWithTemplate_MarginResetMatchesTailwindPreflight(t *testing.T) {
	result := wrapWithTemplate("<h1>Hello</h1><p>World</p>")

	// Editor uses Tailwind Preflight which resets margins; PDF template must match
	assert.Contains(t, result, ".resume-page h1,",
		"margin reset rule should include h1")
	assert.Contains(t, result, ".resume-page p,",
		"margin reset rule should include p")
	assert.Contains(t, result, "{ margin: 0; }",
		"block element margins must be reset to 0 to match Tailwind Preflight")
}

func TestWrapWithTemplate_EmptyContent(t *testing.T) {
	result := wrapWithTemplate("")

	assert.Contains(t, result, `<div class="resume-page resume-document"></div>`)
	assert.Contains(t, result, ".resume-page")
}

func TestWrapWithTemplate_ExtractsFullDocumentBodyAndStyles(t *testing.T) {
	input := `<!DOCTYPE html><html><head><style>.resume-document .name{font-size:22px}</style></head><body><section class="name">陈子俊</section></body></html>`

	result := wrapWithTemplate(input)

	assert.Contains(t, result, `<style>.resume-document .name{font-size:22px}</style>`)
	assert.Contains(t, result, `<section class="name">陈子俊</section>`)
	assert.NotContains(t, result, "<body><section")
	assert.NotContains(t, result, "<html><head>")
	assert.NotContains(t, result, `<div class="resume-page resume-document"><!DOCTYPE html>`)

	// Verify user CSS appears after template </style> (takes priority over template defaults)
	assert.Regexp(t, `</style>\s*<style>\.resume-document \.name\{font-size:22px\}</style>`, result,
		"user CSS must appear after template </style> to take priority")
}

func TestExtractRenderableHTML_RemovesStyleFromBody(t *testing.T) {
	body, styles := extractRenderableHTML(`<style>.x{color:red}</style><div class="x">hello</div>`)

	assert.Contains(t, styles, ".x{color:red}")
	assert.Contains(t, body, `<div class="x">hello</div>`)
	assert.NotContains(t, body, "<style>")
}

// ---------------------------------------------------------------------------
// injectFontCSS
// ---------------------------------------------------------------------------

func TestInjectFontCSS_InsertsBeforeHeadClose(t *testing.T) {
	input := `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><p>test</p></body></html>`
	result := injectFontCSS(input)

	assert.Contains(t, result, `@font-face`)
	assert.Contains(t, result, `font-family: "Inter"`)
	assert.Contains(t, result, "font-weight: 400")
	assert.Contains(t, result, "font-weight: 600")
	assert.Contains(t, result, "base64,")
	// Font style should be inserted before </head>
	assert.Contains(t, result, "<style>")
	assert.Contains(t, result, "</style></head>")
}

func TestInjectFontCSS_PreservesOriginalContent(t *testing.T) {
	input := `<!DOCTYPE html><html><head><style>body{color:red}</style></head><body><p>hello</p></body></html>`
	result := injectFontCSS(input)

	assert.Contains(t, result, "body{color:red}")
	assert.Contains(t, result, "<p>hello</p>")
}

func TestInjectFontCSS_NoHeadTag(t *testing.T) {
	input := `<p>no head tag</p>`
	result := injectFontCSS(input)
	// Should return unchanged when no </head> found
	assert.Equal(t, input, result)
}

func TestRegisterRoutes_ReturnsCleanup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")

	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir)

	cleanup := RegisterRoutes(v1, nil, store)
	assert.NotNil(t, cleanup)

	assert.NotPanics(t, func() {
		cleanup()
	})
}
