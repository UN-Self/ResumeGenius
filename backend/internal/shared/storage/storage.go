package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage defines the contract for file persistence using logical keys.
// Save returns a logical key (e.g. "1/uuid_filename.pdf") rather than a full path.
// Resolve converts a logical key into an absolute filesystem path.
type FileStorage interface {
	Save(projectID uint, filename string, data []byte) (key string, err error)
	Delete(key string) error
	Exists(key string) bool
	Resolve(key string) (string, error)
}

// LocalStorage implements FileStorage using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new LocalStorage rooted at baseDir.
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

// Save writes data to {baseDir}/{projectID}/{uuid}_{filename} and returns the logical key.
func (s *LocalStorage) Save(projectID uint, filename string, data []byte) (string, error) {
	projectDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", projectID))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("create project dir: %w", err)
	}

	uniqueName := fmt.Sprintf("%s_%s", uuid.New().String(), filename)
	logicalKey := fmt.Sprintf("%d/%s", projectID, uniqueName)
	fullPath := filepath.Join(s.baseDir, logicalKey)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return logicalKey, nil
}

// Delete removes the file identified by the logical key. Returns nil if the file does not exist.
func (s *LocalStorage) Delete(key string) error {
	if key == "" {
		return nil
	}

	fullPath := filepath.Join(s.baseDir, key)
	if !s.Exists(key) {
		return nil
	}
	return os.Remove(fullPath)
}

// Exists reports whether the file identified by the logical key exists on disk.
func (s *LocalStorage) Exists(key string) bool {
	if key == "" {
		return false
	}
	fullPath := filepath.Join(s.baseDir, key)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Resolve converts a logical key to an absolute filesystem path.
// Returns an error if the key is empty.
func (s *LocalStorage) Resolve(key string) (string, error) {
	if key == "" {
		return "", errors.New("storage: cannot resolve empty key")
	}
	return filepath.Join(s.baseDir, key), nil
}
