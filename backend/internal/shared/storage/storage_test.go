package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorage_Save_ReturnsLogicalKey(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	key, err := s.Save(1, "resume.pdf", []byte("hello world"))
	require.NoError(t, err)
	require.NotEmpty(t, key)

	// Logical key should NOT contain the base directory
	assert.False(t, strings.HasPrefix(key, dir),
		"key should be logical, not a full path. got: %s", key)

	// Logical key should start with project ID
	assert.True(t, strings.HasPrefix(key, "1/"),
		"key should start with project ID prefix. got: %s", key)

	// Logical key should end with filename
	assert.True(t, strings.HasSuffix(key, "resume.pdf"),
		"key should end with filename. got: %s", key)
}

func TestLocalStorage_Save_CreatesUniqueFileName(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	key1, err := s.Save(1, "resume.pdf", []byte("v1"))
	require.NoError(t, err)

	key2, err := s.Save(1, "resume.pdf", []byte("v2"))
	require.NoError(t, err)

	// Two saves of the same file should produce different keys
	assert.NotEqual(t, key1, key2,
		"saving the same filename twice should produce different keys")
}

func TestLocalStorage_Resolve_ReturnsFullPath(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	key, err := s.Save(42, "test.txt", []byte("data"))
	require.NoError(t, err)

	fullPath, err := s.Resolve(key)
	require.NoError(t, err)

	// Resolved path must be absolute
	assert.True(t, filepath.IsAbs(fullPath),
		"resolved path should be absolute. got: %s", fullPath)

	// The file must actually exist on disk
	info, err := os.Stat(fullPath)
	require.NoError(t, err, "resolved path should point to an existing file")
	assert.False(t, info.IsDir(), "resolved path should be a file, not a directory")
}

func TestLocalStorage_Delete_WorksByKey(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	key, err := s.Save(1, "resume.pdf", []byte("to-delete"))
	require.NoError(t, err)

	// Verify file exists before deletion
	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	_, err = os.Stat(fullPath)
	require.NoError(t, err, "file should exist before deletion")

	// Delete by logical key
	err = s.Delete(key)
	require.NoError(t, err)

	// File should no longer exist
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err), "file should not exist after deletion")
}

func TestLocalStorage_Delete_NonExistentReturnsNil(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	err := s.Delete("9999/nonexistent_file.pdf")
	assert.NoError(t, err,
		"deleting a non-existent key should return nil error")
}
