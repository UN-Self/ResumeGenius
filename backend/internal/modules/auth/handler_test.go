package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type authAPIResponse struct {
	Code    int                    `json:"code"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
}

func setupAuthRouter(t *testing.T) (*gin.Engine, *authFixture) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)
	emailSvc := &EmailService{devMode: true}
	svc := NewService(db, emailSvc)
	h := NewHandler(svc, "test-secret", time.Hour, false, "", t.TempDir())

	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	return r, &authFixture{db: db}
}

type authFixture struct {
	db *gorm.DB
}

func doAuthJSON(t *testing.T, r *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseAuthResponse(t *testing.T, w *httptest.ResponseRecorder) authAPIResponse {
	t.Helper()

	var resp authAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

func TestHandlerRegisterReturnsDevCode(t *testing.T) {
	r, _ := setupAuthRouter(t)

	w := doAuthJSON(t, r, http.MethodPost, "/auth/register", `{
		"username": "handler_user",
		"password": "secret123",
		"email": "handler@example.com"
	}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
	resp := parseAuthResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d message=%s", resp.Code, resp.Message)
	}
	if resp.Data["dev_code"] == "" {
		t.Fatalf("expected dev_code in response, got %#v", resp.Data)
	}
}

func TestHandlerLoginRejectsUnverifiedEmail(t *testing.T) {
	r, fx := setupAuthRouter(t)
	createAuthUser(t, fx.db, models.User{
		ID:            "handler-unverified",
		Username:      "handler_unverified",
		Email:         strPtr("handler_unverified@example.com"),
		EmailVerified: false,
	}, "secret123")

	w := doAuthJSON(t, r, http.MethodPost, "/auth/login", `{
		"username": "handler_unverified",
		"password": "secret123"
	}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
	resp := parseAuthResponse(t, w)
	if resp.Code != 40300 {
		t.Fatalf("expected code 40300, got %d", resp.Code)
	}
}
