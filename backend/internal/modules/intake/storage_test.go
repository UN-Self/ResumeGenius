package intake

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorage_Save(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("hello world")
	key, err := s.Save(1, "resume.pdf", data)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Resolve the logical key to get the physical path
	fullPath, err := s.Resolve(key)
	require.NoError(t, err)

	got, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(got))
	}

	parentDir := filepath.Dir(fullPath)
	assert.Equal(t, "1", filepath.Base(parentDir), "expected project dir '1'")
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	key, err := s.Save(1, "resume.pdf", []byte("content"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = s.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// File should be gone (resolve to physical path to check)
	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestLocalStorage_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	err := s.Delete("/nonexistent/file.pdf")
	if err != nil {
		t.Fatalf("Delete nonexistent file should not error, got: %v", err)
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	path, err := s.Save(1, "resume.pdf", []byte("content"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !s.Exists(path) {
		t.Error("file should exist")
	}
	if s.Exists("/nonexistent/file.pdf") {
		t.Error("nonexistent file should not exist")
	}
}
