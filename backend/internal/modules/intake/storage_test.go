package intake

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorage_Save(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("hello world")
	path, err := s.Save(1, "resume.pdf", data)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(got))
	}

	parentDir := filepath.Dir(path)
	if filepath.Base(parentDir) != "1" {
		t.Errorf("expected project dir '1', got '%s'", filepath.Base(parentDir))
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	path, err := s.Save(1, "resume.pdf", []byte("content"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = s.Delete(path)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
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
