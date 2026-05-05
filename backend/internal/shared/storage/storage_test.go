package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSHA256Hex(t *testing.T) {
	assert.Equal(t,
		"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		SHA256Hex([]byte("hello world")),
	)
}

func TestNormalizeExt(t *testing.T) {
	assert.Equal(t, ".pdf", NormalizeExt("resume.PDF"))
	assert.Equal(t, ".docx", NormalizeExt(`..\unsafe\resume.docx`))
	assert.Equal(t, "", NormalizeExt("README"))
}

func TestLocalStorage_Save_ReturnsContentAddressedLogicalKey(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("hello world")
	hash := SHA256Hex(data)
	key, err := s.Save("user-1", hash, "resume.PDF", data)
	require.NoError(t, err)
	assert.Equal(t, "user-1/"+hash+".pdf", key)

	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	got, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, data, got)
	assert.Equal(t, "user-1", filepath.Base(filepath.Dir(fullPath)))
}

func TestLocalStorage_Save_ReusesExistingKeyForSameHash(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("same content")
	hash := SHA256Hex(data)
	key1, err := s.Save("user-1", hash, ".pdf", data)
	require.NoError(t, err)

	key2, err := s.Save("user-1", hash, "resume.pdf", data)
	require.NoError(t, err)

	assert.Equal(t, key1, key2)
	assert.True(t, strings.HasSuffix(key1, hash+".pdf"))
}

func TestLocalStorage_Save_DifferentUsersUseDifferentNamespaces(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("same content")
	hash := SHA256Hex(data)

	key1, err := s.Save("user-1", hash, ".pdf", data)
	require.NoError(t, err)
	key2, err := s.Save("user-2", hash, ".pdf", data)
	require.NoError(t, err)

	assert.NotEqual(t, key1, key2)
	assert.True(t, strings.HasPrefix(key1, "user-1/"))
	assert.True(t, strings.HasPrefix(key2, "user-2/"))
}

func TestLocalStorage_Save_RejectsInvalidInputs(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	_, err := s.Save("user/1", SHA256Hex([]byte("a")), ".pdf", []byte("a"))
	require.ErrorContains(t, err, "invalid user ID")

	_, err = s.Save("user-1", "not-a-hash", ".pdf", []byte("a"))
	require.ErrorContains(t, err, "invalid file hash")

	_, err = s.Save("user-1", SHA256Hex([]byte("a")), "", []byte("a"))
	require.ErrorContains(t, err, "empty file extension")
}

func TestLocalStorage_Resolve_ReturnsFullPath(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("data")
	key, err := s.Save("user-42", SHA256Hex(data), ".txt", data)
	require.NoError(t, err)

	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(fullPath))

	info, err := os.Stat(fullPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestLocalStorage_Delete_WorksByKey(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	data := []byte("to-delete")
	key, err := s.Save("user-1", SHA256Hex(data), ".pdf", data)
	require.NoError(t, err)

	fullPath, err := s.Resolve(key)
	require.NoError(t, err)
	require.FileExists(t, fullPath)

	require.NoError(t, s.Delete(key))
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Delete_NonExistentReturnsNil(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	assert.NoError(t, s.Delete("user-1/nonexistent.pdf"))
}
