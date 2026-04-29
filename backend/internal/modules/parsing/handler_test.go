package parsing

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
)

type stubParseService struct {
	calledWithUser string
	calledWith     uint
	parseResult    []ParsedContent
	parseErr       error
	generateResult *GenerateResult
	generateErr    error
}

func (s *stubParseService) ParseForUser(userID string, projectID uint) ([]ParsedContent, error) {
	s.calledWithUser = userID
	s.calledWith = projectID
	return s.parseResult, s.parseErr
}

func (s *stubParseService) GenerateForUser(userID string, projectID uint) (*GenerateResult, error) {
	s.calledWithUser = userID
	s.calledWith = projectID
	return s.generateResult, s.generateErr
}

func TestParse_SucceedsAndReturnsParsedContents(t *testing.T) {
	service := &stubParseService{
		parseResult: []ParsedContent{
			{AssetID: 1, Type: AssetTypeResumePDF, Label: "sample_resume.pdf", Text: "pdf text"},
			{AssetID: 2, Type: AssetTypeNote, Label: "补充说明", Text: "note text"},
		},
	}

	router := newParseTestRouter(service)
	w := performParseRequest(router, `{"project_id": 7}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if service.calledWith != 7 {
		t.Fatalf("expected service to be called with project_id 7, got %d", service.calledWith)
	}
	if service.calledWithUser != "user-1" {
		t.Fatalf("expected service to be called with user user-1, got %q", service.calledWithUser)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	parsedContents := data["parsed_contents"].([]interface{})
	if len(parsedContents) != 2 {
		t.Fatalf("expected 2 parsed contents, got %d", len(parsedContents))
	}
	first := parsedContents[0].(map[string]interface{})
	if first["label"] != "sample_resume.pdf" {
		t.Fatalf("expected first parsed label sample_resume.pdf, got %v", first["label"])
	}
	second := parsedContents[1].(map[string]interface{})
	if second["label"] != "补充说明" {
		t.Fatalf("expected second parsed label 补充说明, got %v", second["label"])
	}
}

func TestParse_Returns401WhenUnauthorized(t *testing.T) {
	service := &stubParseService{}
	router := newUnauthorizedParseTestRouter(service)
	w := performParseRequest(router, `{"project_id": 7}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != 40100 {
		t.Fatalf("expected code 40100, got %v", resp["code"])
	}
}

func TestParse_Returns400WhenRequestBodyInvalid(t *testing.T) {
	service := &stubParseService{}
	router := newParseTestRouter(service)

	tests := []string{
		`invalid json`,
		`{}`,
		`{"project_id": 0}`,
	}

	for _, body := range tests {
		w := performParseRequest(router, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for body %s, got %d", body, w.Code)
		}

		resp := decodeHandlerResponse(t, w)
		if resp["code"].(float64) != 40000 {
			t.Fatalf("expected code 40000 for body %s, got %v", body, resp["code"])
		}
	}
}

func TestParse_Returns404WhenProjectNotFound(t *testing.T) {
	service := &stubParseService{parseErr: ErrProjectNotFound}
	router := newParseTestRouter(service)
	w := performParseRequest(router, `{"project_id": 9}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != CodeProjectNotFound {
		t.Fatalf("expected code %d, got %v", CodeProjectNotFound, resp["code"])
	}
}

func TestParse_Returns400WhenNoUsableAssets(t *testing.T) {
	service := &stubParseService{parseErr: ErrNoUsableAssets}
	router := newParseTestRouter(service)
	w := performParseRequest(router, `{"project_id": 9}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != CodeNoUsableAssets {
		t.Fatalf("expected code %d, got %v", CodeNoUsableAssets, resp["code"])
	}
}

func TestParse_Returns400ForFormatSpecificParseErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStatus int
	}{
		{
			name:       "pdf parse error",
			err:        ErrPDFParseFailed,
			wantCode:   CodeParsePDFFailed,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "docx parse error",
			err:        ErrDOCXParseFailed,
			wantCode:   CodeParseDOCXFailed,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &stubParseService{parseErr: tc.err}
			router := newParseTestRouter(service)
			w := performParseRequest(router, `{"project_id": 9}`)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}

			resp := decodeHandlerResponse(t, w)
			if resp["code"].(float64) != float64(tc.wantCode) {
				t.Fatalf("expected code %d, got %v", tc.wantCode, resp["code"])
			}
		})
	}
}

func TestParse_Returns400ForInvalidAssetDataErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{name: "missing asset uri", err: ErrAssetURIMissing, wantMsg: "project contains invalid asset data"},
		{name: "missing asset content", err: ErrAssetContentMissing, wantMsg: "project contains invalid asset data"},
		{name: "no generatable text", err: ErrNoGeneratableText, wantMsg: "project has no usable text content"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &stubParseService{parseErr: tc.err}
			router := newParseTestRouter(service)
			w := performParseRequest(router, `{"project_id": 9}`)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}

			resp := decodeHandlerResponse(t, w)
			if resp["code"].(float64) != float64(CodeInvalidAssetData) {
				t.Fatalf("expected code %d, got %v", CodeInvalidAssetData, resp["code"])
			}
			if resp["message"] != tc.wantMsg {
				t.Fatalf("expected message %q, got %v", tc.wantMsg, resp["message"])
			}
		})
	}
}

func TestParse_Returns500ForUnexpectedErrors(t *testing.T) {
	service := &stubParseService{parseErr: errors.New("boom")}
	router := newParseTestRouter(service)
	w := performParseRequest(router, `{"project_id": 9}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != 50000 {
		t.Fatalf("expected code 50000, got %v", resp["code"])
	}
}

func TestGenerate_SucceedsAndReturnsDraftData(t *testing.T) {
	service := &stubParseService{
		generateResult: &GenerateResult{
			DraftID:     5,
			VersionID:   8,
			HTMLContent: "<!DOCTYPE html><html><body>mock</body></html>",
		},
	}

	router := newGenerateTestRouter(service)
	w := performGenerateRequest(router, `{"project_id": 7}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if service.calledWith != 7 {
		t.Fatalf("expected service to be called with project_id 7, got %d", service.calledWith)
	}
	if service.calledWithUser != "user-1" {
		t.Fatalf("expected service to be called with user user-1, got %q", service.calledWithUser)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["draft_id"].(float64) != 5 {
		t.Fatalf("expected draft_id 5, got %v", data["draft_id"])
	}
	if data["version_id"].(float64) != 8 {
		t.Fatalf("expected version_id 8, got %v", data["version_id"])
	}
	if data["html_content"].(string) != "<!DOCTYPE html><html><body>mock</body></html>" {
		t.Fatalf("unexpected html_content: %v", data["html_content"])
	}
}

func TestGenerate_Returns401WhenUnauthorized(t *testing.T) {
	service := &stubParseService{}
	router := newUnauthorizedGenerateTestRouter(service)
	w := performGenerateRequest(router, `{"project_id": 7}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != 40100 {
		t.Fatalf("expected code 40100, got %v", resp["code"])
	}
}

func TestGenerate_Returns400WhenRequestBodyInvalid(t *testing.T) {
	service := &stubParseService{}
	router := newGenerateTestRouter(service)

	tests := []string{
		`invalid json`,
		`{}`,
		`{"project_id": 0}`,
	}

	for _, body := range tests {
		w := performGenerateRequest(router, body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for body %s, got %d", body, w.Code)
		}

		resp := decodeHandlerResponse(t, w)
		if resp["code"].(float64) != 40000 {
			t.Fatalf("expected code 40000 for body %s, got %v", body, resp["code"])
		}
	}
}

func TestGenerate_ReturnsMappedErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "project not found", err: ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: CodeProjectNotFound},
		{name: "no usable assets", err: ErrNoUsableAssets, wantStatus: http.StatusBadRequest, wantCode: CodeNoUsableAssets},
		{name: "ai generation failed", err: ErrAIGenerateFailed, wantStatus: http.StatusInternalServerError, wantCode: CodeAIGenerateFailed},
		{name: "invalid asset data", err: ErrAssetContentMissing, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAssetData},
		{name: "no generatable text", err: ErrNoGeneratableText, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAssetData},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &stubParseService{generateErr: tc.err}
			router := newGenerateTestRouter(service)
			w := performGenerateRequest(router, `{"project_id": 9}`)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}

			resp := decodeHandlerResponse(t, w)
			if resp["code"].(float64) != float64(tc.wantCode) {
				t.Fatalf("expected code %d, got %v", tc.wantCode, resp["code"])
			}
		})
	}
}

func TestGenerate_Returns500ForUnexpectedErrors(t *testing.T) {
	service := &stubParseService{generateErr: errors.New("boom")}
	router := newGenerateTestRouter(service)
	w := performGenerateRequest(router, `{"project_id": 9}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	resp := decodeHandlerResponse(t, w)
	if resp["code"].(float64) != 50000 {
		t.Fatalf("expected code 50000, got %v", resp["code"])
	}
}

func TestRoutePaths_CorrectlyMounted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterRoutes(v1, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parsing/parse", strings.NewReader(`{"project_id": 1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Fatal("resource path should be mounted")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/parsing/generate", strings.NewReader(`{"project_id": 1}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Fatal("generate route should be mounted in B8")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/b_parsing/parse", strings.NewReader(`{"project_id": 1}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("legacy path should not be mounted, got %d", w.Code)
	}
}

func newParseTestRouter(service parseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "user-1")
		c.Next()
	})
	handler := NewHandler(service)
	router.POST("/parsing/parse", handler.Parse)
	return router
}

func newGenerateTestRouter(service parseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "user-1")
		c.Next()
	})
	handler := NewHandler(service)
	router.POST("/parsing/generate", handler.Generate)
	return router
}

func newUnauthorizedParseTestRouter(service parseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(service)
	router.POST("/parsing/parse", handler.Parse)
	return router
}

func newUnauthorizedGenerateTestRouter(service parseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(service)
	router.POST("/parsing/generate", handler.Generate)
	return router
}

func performParseRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/parsing/parse", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func performGenerateRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/parsing/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeHandlerResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}
