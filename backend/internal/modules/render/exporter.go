package render

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrTaskNotCompleted = errors.New("task not completed")
)

// PDFExporter defines the contract for HTML-to-PDF conversion.
type PDFExporter interface {
	ExportHTMLToPDF(htmlContent string) ([]byte, error)
}

// ExportTask represents an async PDF export job.
type ExportTask struct {
	ID          string    `json:"task_id"`
	DraftID     uint      `json:"draft_id"`
	Status      string    `json:"status"`   // pending | processing | completed | failed
	Progress    int       `json:"progress"` // 0-100
	DownloadURL *string   `json:"download_url"`
	Error       *string   `json:"error"`
	CreatedAt   time.Time `json:"created_at"`

	htmlContent string // unexported: HTML payload passed to the worker
	fileKey     string // unexported: logical key returned by store.Save
}

// ExportService manages async PDF export tasks.
type ExportService struct {
	exporter PDFExporter
	store    storage.FileStorage
	db       *gorm.DB
	tasks    sync.Map
	queue    chan *ExportTask
	closeCh  chan struct{}
	wg       sync.WaitGroup
}

// NewExportService creates a new ExportService with the given exporter and storage.
// It starts a background worker goroutine for processing tasks.
func NewExportService(exporter PDFExporter, store storage.FileStorage) *ExportService {
	s := &ExportService{
		exporter: exporter,
		store:    store,
		queue:    make(chan *ExportTask, 64),
		closeCh:  make(chan struct{}),
	}

	s.wg.Add(1)
	go s.worker()

	return s
}

// CreateTask validates the draft exists, reads its HTML from the DB,
// wraps it with the render template, and queues it for async processing.
func (s *ExportService) CreateTask(draftID uint) (string, error) {
	if s.db == nil {
		return "", errors.New("database connection required for export")
	}

	var draft models.Draft
	if err := s.db.First(&draft, draftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrDraftNotFound
		}
		return "", err
	}

	wrappedHTML := wrapWithTemplate(draft.HTMLContent)

	taskID := "task_" + uuid.New().String()
	task := &ExportTask{
		ID:          taskID,
		DraftID:     draftID,
		Status:      "pending",
		Progress:    0,
		CreatedAt:   time.Now().UTC(),
		htmlContent: wrappedHTML,
	}

	s.tasks.Store(taskID, task)
	s.queue <- task

	return taskID, nil
}

// GetTask returns the current state of an export task by its ID.
func (s *ExportService) GetTask(taskID string) (*ExportTask, error) {
	val, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, ErrTaskNotFound
	}
	return val.(*ExportTask), nil
}

// GetFile reads the PDF bytes for a completed task.
func (s *ExportService) GetFile(taskID string) ([]byte, error) {
	val, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, ErrTaskNotFound
	}

	task := val.(*ExportTask)
	if task.Status != "completed" {
		return nil, ErrTaskNotCompleted
	}

	// Resolve the logical key to an absolute path and read the file.
	absPath, err := s.store.Resolve(task.fileKey)
	if err != nil {
		return nil, fmt.Errorf("resolve file: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

// Close shuts down the worker goroutine and waits for it to finish.
func (s *ExportService) Close() {
	close(s.closeCh)
	s.wg.Wait()
}

// worker reads tasks from the queue and processes them.
func (s *ExportService) worker() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeCh:
			// Drain remaining tasks in queue before exiting.
			for {
				select {
				case task := <-s.queue:
					s.processTask(task)
				default:
					return
				}
			}
		case task := <-s.queue:
			s.processTask(task)
		}
	}
}

// processTask performs the actual HTML-to-PDF conversion for a single task.
func (s *ExportService) processTask(task *ExportTask) {
	// Transition to processing.
	task.Status = "processing"
	task.Progress = 30

	// Call the exporter.
	pdfBytes, err := s.exporter.ExportHTMLToPDF(task.htmlContent)
	if err != nil {
		errMsg := err.Error()
		task.Status = "failed"
		task.Error = &errMsg
		task.Progress = 0
		return
	}

	// Save the PDF file using content-addressed storage under the exports namespace.
	fileKey, err := s.store.Save("exports", storage.SHA256Hex(pdfBytes), ".pdf", pdfBytes)
	if err != nil {
		errMsg := fmt.Sprintf("save file: %s", err.Error())
		task.Status = "failed"
		task.Error = &errMsg
		task.Progress = 0
		return
	}

	task.fileKey = fileKey
	downloadURL := fmt.Sprintf("/api/v1/tasks/%s/file", task.ID)
	task.DownloadURL = &downloadURL
	task.Status = "completed"
	task.Progress = 100
}

// ChromeExporter uses chromedp to render HTML to PDF.
type ChromeExporter struct {
	allocCtx context.Context
	cancel   context.CancelFunc
}

// NewChromeExporter creates a long-lived Chrome process via chromedp.
// Caller must call Close() to release resources.
func NewChromeExporter() *ChromeExporter {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	// Allow overriding the Chrome executable path via environment variable.
	// In Alpine Docker, the binary is at /usr/bin/chromium.
	if chromeBin := os.Getenv("CHROME_BIN"); chromeBin != "" {
		opts = append(opts, chromedp.ExecPath(chromeBin))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return &ChromeExporter{allocCtx: allocCtx, cancel: cancel}
}

// ExportHTMLToPDF renders the given HTML content to PDF bytes.
// The HTML is expected to be a complete document captured from the editor DOM.
// Each call creates a new tab (context) but reuses the same Chrome process.
func (c *ChromeExporter) ExportHTMLToPDF(htmlContent string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(c.allocCtx)
	defer cancel()

	renderHTML := injectFontCSS(htmlContent)

	var buf []byte
	err := chromedp.Run(ctx,
		// Force screen media to prevent @media print rules from affecting PDF layout
		emulation.SetEmulatedMedia().WithMedia("screen"),
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, renderHTML).Do(ctx)
		}),
		// Wait for fonts to finish loading before rendering.
		chromedp.Evaluate(`document.fonts.ready`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4
				WithPaperHeight(11.69). // A4
				Do(ctx)
			return err
		}),
	)
	return buf, err
}

// Close releases the Chrome allocator resources.
func (c *ChromeExporter) Close() {
	c.cancel()
}

//go:embed render-template.html
var renderTemplate string

//go:embed fonts/inter-regular.woff2
var interRegular []byte

//go:embed fonts/inter-medium.woff2
var interMedium []byte

//go:embed fonts/inter-semibold.woff2
var interSemiBold []byte

// interFontFaceCSS generates @font-face declarations with embedded base64 WOFF2 data.
func interFontFaceCSS() string {
	return fmt.Sprintf(`@font-face { font-family: "Inter"; font-style: normal; font-weight: 400; font-display: swap; src: url(data:font/woff2;base64,%s) format("woff2"); }
@font-face { font-family: "Inter"; font-style: normal; font-weight: 500; font-display: swap; src: url(data:font/woff2;base64,%s) format("woff2"); }
@font-face { font-family: "Inter"; font-style: normal; font-weight: 600; font-display: swap; src: url(data:font/woff2;base64,%s) format("woff2"); }`,
		interRegular, interMedium, interSemiBold)
}

// injectFontCSS inserts the embedded Inter font @font-face CSS into the HTML document.
func injectFontCSS(html string) string {
	fontStyle := "<style>" + interFontFaceCSS() + "</style>"
	return strings.Replace(html, "</head>", fontStyle+"</head>", 1)
}

// wrapWithTemplate inserts the HTML fragment into the render template,
// replacing the {{CONTENT}} placeholder.
func wrapWithTemplate(htmlFragment string) string {
	return strings.Replace(renderTemplate, "{{CONTENT}}", htmlFragment, 1)
}
