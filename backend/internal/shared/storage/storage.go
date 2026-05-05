package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileStorage defines the contract for file persistence using logical keys.
// Save returns a logical key (e.g. "user-1/<sha256>.pdf") rather than a full path.
// Resolve converts a logical key into an absolute filesystem path.
type FileStorage interface {
	Save(userID string, fileHash string, ext string, data []byte) (key string, err error)
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

var (
	storageNamespacePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	fileHashPattern         = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// SHA256Hex returns the lowercase SHA-256 hex digest for the provided bytes.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// NormalizeExt keeps only the trailing extension and lowercases it.
func NormalizeExt(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.Ext(filepath.Base(trimmed)))
}

func normalizeStorageNamespace(userID string) (string, error) {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return "", errors.New("storage: empty user ID")
	}
	if !storageNamespacePattern.MatchString(trimmed) {
		return "", fmt.Errorf("storage: invalid user ID %q", userID)
	}
	return trimmed, nil
}

func normalizeFileHash(fileHash string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(fileHash))
	if !fileHashPattern.MatchString(trimmed) {
		return "", fmt.Errorf("storage: invalid file hash %q", fileHash)
	}
	return trimmed, nil
}

func buildLogicalKey(userID string, fileHash string, ext string) (string, error) {
	namespace, err := normalizeStorageNamespace(userID)
	if err != nil {
		return "", err
	}
	normalizedHash, err := normalizeFileHash(fileHash)
	if err != nil {
		return "", err
	}
	normalizedExt := NormalizeExt(ext)
	if normalizedExt == "" {
		return "", errors.New("storage: empty file extension")
	}
	return filepath.ToSlash(filepath.Join(namespace, normalizedHash+normalizedExt)), nil
}

// Save writes data to {baseDir}/{userID}/{sha256}.{ext} and returns the logical key.
// If the file already exists, Save returns the existing key without rewriting it.
func (s *LocalStorage) Save(userID string, fileHash string, ext string, data []byte) (string, error) {
	logicalKey, err := buildLogicalKey(userID, fileHash, ext)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.baseDir, logicalKey)
	if s.Exists(logicalKey) {
		return logicalKey, nil
	}

	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("create storage dir: %w", err)
	}

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
