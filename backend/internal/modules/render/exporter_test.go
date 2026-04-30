package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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

func TestCreateTask_ReturnsTaskID(t *testing.T) {
	db := SetupTestDB(t)
	draft := seedDraft(t, db)
	svc, _ := newTestExportServiceWithDB(t, db)

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")
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

	_, err := svc.CreateTask(99999, "<html><body>Resume</body></html>")
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

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")
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

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")
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

	taskID, err := svc.CreateTask(draft.ID, "<html><body>Resume</body></html>")
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

func TestGetFile_TaskNotCompleted(t *testing.T) {
	svc, _ := newTestExportService(t)
	mock := &MockExporter{
		PDFBytes: samplePDF(t),
		Err:      fmt.Errorf("intentional slow failure"),
	}
	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir)
	svc = NewExportService(mock, store)
	t.Cleanup(func() { svc.Close() })

	// Create task directly without DB validation (db is nil, skip validation)
	svc.db = nil
	taskID, err := svc.CreateTask(1, "<html><body>Test</body></html>")
	require.NoError(t, err)

	// Try to get file immediately — task should still be pending/processing
	_, err = svc.GetFile(taskID)
	require.ErrorIs(t, err, ErrTaskNotCompleted)
}
