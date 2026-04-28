package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIssueAndParseToken(t *testing.T) {
	token, err := IssueToken("user-1", "alice", "secret", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	claims, err := ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.Subject != "user-1" {
		t.Fatalf("expected subject user-1, got %s", claims.Subject)
	}
	if claims.Username != "alice" {
		t.Fatalf("expected username alice, got %s", claims.Username)
	}
}

func TestAuthRequired_UnauthorizedWithoutCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired("secret"))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_AuthorizedWithValidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired("secret"))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": UserIDFromContext(c)})
	})

	token, err := IssueToken("user-1", "alice", "secret", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: AccessTokenCookieName, Value: token})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
