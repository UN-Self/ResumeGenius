package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// setupToolTestDB creates an in-memory SQLite DB for tool unit tests.
func setupToolTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	err = db.AutoMigrate(&models.Asset{}, &models.SSHKey{}, &models.Project{})
	require.NoError(t, err, "auto migrate failed")
	return db
}

// ---------------------------------------------------------------------------
// Mock providers
// ---------------------------------------------------------------------------

type mockAIProvider struct {
	response string
	err      error
}

func (m *mockAIProvider) Call(_ context.Context, _, _, _ string) (string, error) {
	return m.response, m.err
}

type mockSSHProvider struct {
	keys map[uint]string
}

func (m *mockSSHProvider) GetDecryptedKey(_ string, keyID uint) (string, error) {
	if k, ok := m.keys[keyID]; ok {
		return k, nil
	}
	return "", fmt.Errorf("key not found for id %d", keyID)
}

// ---------------------------------------------------------------------------
// Execute tests
// ---------------------------------------------------------------------------

func TestExtractGitRepoTool_AssetNotFound(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"repo_asset_id": float64(999),
		"user_context":  "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset not found")
}

func TestExtractGitRepoTool_AssetNotGitRepo(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	content := "some content"
	db.Create(&models.Asset{
		ProjectID: 1,
		Type:      "note",
		Content:   &content,
	})

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"repo_asset_id": float64(1),
		"user_context":  "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not git_repo")
}

func TestExtractGitRepoTool_MissingURI(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	keyID := uint(1)
	db.Create(&models.Asset{
		ProjectID: 1,
		Type:      "git_repo",
		KeyID:     &keyID,
	})

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"repo_asset_id": float64(1),
		"user_context":  "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no URI")
}

func TestExtractGitRepoTool_MissingKeyID(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	uri := "git@github.com:test/repo.git"
	db.Create(&models.Asset{
		ProjectID: 1,
		Type:      "git_repo",
		URI:       &uri,
	})

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"repo_asset_id": float64(1),
		"user_context":  "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SSH key associated")
}

func TestExtractGitRepoTool_MissingRepoAssetID(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"user_context": "test",
	})
	require.Error(t, err)
}

func TestExtractGitRepoTool_NameAndParameters(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	assert.Equal(t, "extract_git_repo", tool.Name())

	params := tool.Parameters()
	assert.Equal(t, "object", params["type"])
	props := params["properties"].(map[string]interface{})
	assert.Contains(t, props, "repo_asset_id")
	assert.Contains(t, props, "user_context")
	required := params["required"].([]string)
	assert.Contains(t, required, "repo_asset_id")
	assert.Contains(t, required, "user_context")
}

// ---------------------------------------------------------------------------
// exploreRepo tests
// ---------------------------------------------------------------------------

func TestExploreRepo_BasicAnalysis(t *testing.T) {
	db := setupToolTestDB(t)
	mockAI := &mockAIProvider{response: "# Test Analysis\n\n## 项目概述\nMock analysis result"}
	tool := NewExtractGitRepoTool(db, nil, mockAI, "haiku", 50)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Project\n\nA test repo."), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/test\ngo 1.21"), 0644)
	require.NoError(t, err)

	report, err := tool.exploreRepo(context.Background(), tmpDir, "目标岗位：后端工程师")
	require.NoError(t, err)
	assert.Contains(t, report, "Mock analysis result")
}

func TestExploreRepo_WithUserContext(t *testing.T) {
	db := setupToolTestDB(t)

	mockAI := &mockAIProvider{
		response: "# Analysis",
	}
	wrappedAI := &captureAIProvider{inner: mockAI}
	tool := NewExtractGitRepoTool(db, nil, wrappedAI, "haiku", 50)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)

	_, err := tool.exploreRepo(context.Background(), tmpDir, "目标岗位：Go 后端工程师")
	require.NoError(t, err)
	assert.Contains(t, wrappedAI.lastUserMsg, "目标岗位：Go 后端工程师")
}

func TestExploreRepo_AIError(t *testing.T) {
	db := setupToolTestDB(t)
	mockAI := &mockAIProvider{err: fmt.Errorf("API unavailable")}
	tool := NewExtractGitRepoTool(db, nil, mockAI, "haiku", 50)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)

	_, err := tool.exploreRepo(context.Background(), tmpDir, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AI analysis failed")
}

// ---------------------------------------------------------------------------
// Helper: dirSizeMB
// ---------------------------------------------------------------------------

func TestDirSizeMB(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("hello world"), 0644)

	sizeMB, err := dirSizeMB(tmpDir)
	require.NoError(t, err)
	// "hello world" = 11 bytes, very small in MB
	assert.True(t, sizeMB < 0.01, "expected tiny size, got %f MB", sizeMB)
}

func TestDirSizeMB_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	sizeMB, err := dirSizeMB(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 0.0, sizeMB)
}

// ---------------------------------------------------------------------------
// Helper: normalizeChatURL
// ---------------------------------------------------------------------------

func TestNormalizeChatURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://api.example.com/v1", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/v1/", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/v1/chat/completions", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/api/paas/v4", "https://api.example.com/api/paas/v4/chat/completions"},
		{"https://api.example.com/custom", "https://api.example.com/custom/v1/chat/completions"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeAIURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Helper: captureAIProvider (captures last user message for assertions)
// ---------------------------------------------------------------------------

type captureAIProvider struct {
	inner          *mockAIProvider
	lastUserMsg    string
	lastModel      string
	lastSystemMsg  string
}

func (c *captureAIProvider) Call(_ context.Context, model, systemPrompt, userMessage string) (string, error) {
	c.lastUserMsg = userMessage
	c.lastModel = model
	c.lastSystemMsg = systemPrompt
	return c.inner.Call(context.TODO(), model, systemPrompt, userMessage)
}

func TestReadKeyFiles_SkipDotGit(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("git config"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Project"), 0644)

	var sb strings.Builder
	readKeyFiles(tmpDir, &sb)
	assert.NotContains(t, sb.String(), ".git/config")
	assert.Contains(t, sb.String(), "README.md")
}

func TestReadKeyFiles_SkipNodeModules(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "node_modules", "lodash"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "node_modules", "lodash", "index.js"), []byte("module.exports"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)

	var sb strings.Builder
	readKeyFiles(tmpDir, &sb)
	assert.NotContains(t, sb.String(), "node_modules")
	assert.Contains(t, sb.String(), "package.json")
}

func TestDirSizeMB_SkipDotGit(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "big"), make([]byte, 5000), 0644)
	os.WriteFile(filepath.Join(tmpDir, "real.txt"), []byte("data"), 0644)

	sizeMB, err := dirSizeMB(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 4.0/1024/1024, sizeMB) // only real.txt counted
}
