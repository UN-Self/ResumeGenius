package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResourceRoutes_AreMountedOnApiV1(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("COOKIE_SECURE", "false")
	r, _ := setupRouter(nil)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/projects"},
		{http.MethodPost, "/api/v1/projects"},
		{http.MethodGet, "/api/v1/projects/1"},
		{http.MethodDelete, "/api/v1/projects/1"},
		{http.MethodPost, "/api/v1/assets/upload"},
		{http.MethodPost, "/api/v1/assets/git"},
		{http.MethodGet, "/api/v1/assets"},
		{http.MethodDelete, "/api/v1/assets/1"},
		{http.MethodPost, "/api/v1/assets/notes"},
		{http.MethodPut, "/api/v1/assets/notes/1"},
		{http.MethodPost, "/api/v1/parsing/parse"},
		{http.MethodPost, "/api/v1/parsing/generate"},
		{http.MethodPost, "/api/v1/ai/sessions"},
		{http.MethodGet, "/api/v1/drafts/1"},
		{http.MethodPost, "/api/v1/drafts/1/export"},
		{http.MethodGet, "/api/v1/tasks/task_1"},
		{http.MethodGet, "/api/v1/drafts/1/versions"},
		{http.MethodPost, "/api/v1/drafts/1/versions"},
		{http.MethodPost, "/api/v1/drafts/1/rollback"},
		{http.MethodGet, "/api/v1/tasks/task_1/file"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Fatalf("route not found: %s %s", tc.method, tc.path)
		}
	}
}

func TestLegacyRoutes_Return404(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("COOKIE_SECURE", "false")
	r, _ := setupRouter(nil)

	legacy := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/intake/projects"},
		{http.MethodGet, "/api/v1/workbench/drafts/1"},
		{http.MethodPost, "/api/v1/render/export"},
	}

	for _, tc := range legacy {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("legacy path should be 404: %s %s, got %d", tc.method, tc.path, w.Code)
		}
	}
}
