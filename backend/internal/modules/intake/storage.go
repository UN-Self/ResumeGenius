package intake

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage defines the contract for file persistence.
type FileStorage interface {
	Save(projectID uint, filename string, data []byte) (string, error)
	Delete(path string) error
	Exists(path string) bool
}

// LocalStorage implements FileStorage using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new LocalStorage rooted at baseDir.
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

// Save writes data to {baseDir}/{projectID}/{uuid}_{filename} and returns the full path.
func (s *LocalStorage) Save(projectID uint, filename string, data []byte) (string, error) {
	projectDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", projectID))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("create project dir: %w", err)
	}

	uniqueName := fmt.Sprintf("%s_%s", uuid.New().String(), filename)
	fullPath := filepath.Join(projectDir, uniqueName)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fullPath, nil
}

// Delete removes the file at path. Returns nil if the file does not exist.
func (s *LocalStorage) Delete(path string) error {
	if !s.Exists(path) {
		return nil
	}
	return os.Remove(path)
}

// Exists reports whether the file at path exists on disk.
func (s *LocalStorage) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
