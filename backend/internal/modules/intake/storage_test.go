package intake

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedstorage "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

func TestLocalStorage_Save(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("hello world")
	hash := sharedstorage.SHA256Hex(data)
	key, err := s.Save("user-1", hash, "resume.pdf", data)
	require.NoError(t, err)
	assert.Equal(t, "user-1/"+hash+".pdf", key)

	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	got, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, data, got)
	assert.Equal(t, "user-1", filepath.Base(filepath.Dir(fullPath)))
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("content")
	key, err := s.Save("user-1", sharedstorage.SHA256Hex(data), ".pdf", data)
	require.NoError(t, err)

	require.NoError(t, s.Delete(key))

	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}

func TestLocalStorage_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	require.NoError(t, s.Delete("user-1/nonexistent.pdf"))
}

func TestLocalStorage_Exists(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("content")
	key, err := s.Save("user-1", sharedstorage.SHA256Hex(data), ".pdf", data)
	require.NoError(t, err)

	assert.True(t, s.Exists(key))
	assert.False(t, s.Exists("user-1/nonexistent.pdf"))
}
