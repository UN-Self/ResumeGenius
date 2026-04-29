package intake

import "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"

// NewLocalStorage delegates to shared/storage.NewLocalStorage.
func NewLocalStorage(baseDir string) *storage.LocalStorage {
	return storage.NewLocalStorage(baseDir)
}
