package intake

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

// --- ProjectService tests ---

func TestProjectService_Create(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	proj, err := svc.Create("user-1", "前端工程师简历")
	assert.NoError(t, err)
	assert.Equal(t, "前端工程师简历", proj.Title)
	assert.Equal(t, "user-1", proj.UserID)
	assert.Equal(t, "active", proj.Status)
	assert.Greater(t, proj.ID, uint(0))
}

func TestProjectService_List_FiltersByUserID(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	svc.Create("user-1", "项目A")
	svc.Create("user-2", "项目B")
	svc.Create("user-1", "项目C")

	projects, err := svc.List("user-1")
	assert.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestProjectService_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "测试项目")

	proj, err := svc.GetByID("user-1", created.ID)
	assert.NoError(t, err)
	assert.Equal(t, "测试项目", proj.Title)
}

func TestProjectService_GetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.GetByID("user-1", 9999)
	assert.Error(t, err)
}

func TestProjectService_GetByID_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	_, err := svc.GetByID("user-2", created.ID)
	assert.Error(t, err)
}

func TestProjectService_Delete(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "待删除")

	err := svc.Delete("user-1", created.ID)
	assert.NoError(t, err)

	_, err = svc.GetByID("user-1", created.ID)
	assert.Error(t, err)
}

func TestProjectService_Delete_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	err := svc.Delete("user-2", created.ID)
	assert.Error(t, err)
}

// --- AssetService tests ---

func TestAssetService_UploadFile(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	data := []byte("%PDF-1.4 fake pdf content")
	asset, err := svc.UploadFile("user-1", proj.ID, "resume.pdf", data, int64(len(data)))
	assert.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, "resume_pdf", asset.Type)
	assert.NotNil(t, asset.URI)
	assert.True(t, strings.HasSuffix(*asset.URI, "resume.pdf"))
}

func TestAssetService_UploadFile_UnsupportedFormat(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	_, err = svc.UploadFile("user-1", proj.ID, "malware.exe", []byte("exe content"), 10)
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestAssetService_UploadFile_ExceedsSizeLimit(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	oversized := make([]byte, 21*1024*1024)
	_, err = svc.UploadFile("user-1", proj.ID, "big.pdf", oversized, int64(len(oversized)))
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestAssetService_UploadFile_ProjectNotFound(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	_, err := svc.UploadFile("user-1", 9999, "resume.pdf", []byte("data"), 10)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestAssetService_CreateGitRepo(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	asset, err := svc.CreateGitRepo("user-1", proj.ID, "https://github.com/example/resume.git")
	assert.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, "git_repo", asset.Type)
	assert.Equal(t, "https://github.com/example/resume.git", *asset.URI)
}

func TestAssetService_CreateGitRepo_InvalidURL(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	_, err = svc.CreateGitRepo("user-1", proj.ID, "not-a-url")
	assert.ErrorIs(t, err, ErrInvalidGitURL)
}

func TestAssetService_CreateNote(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	asset, err := svc.CreateNote("user-1", proj.ID, "Candidate has 5 years of Go experience", "Background Notes")
	assert.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, "note", asset.Type)
	assert.Equal(t, "Candidate has 5 years of Go experience", *asset.Content)
	assert.Equal(t, "Background Notes", *asset.Label)
}

func TestAssetService_UpdateNote(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	created, err := svc.CreateNote("user-1", proj.ID, "Original note", "Label1")
	require.NoError(t, err)

	updated, err := svc.UpdateNote("user-1", created.ID, "Updated note content", "Label2")
	assert.NoError(t, err)
	assert.Equal(t, "Updated note content", *updated.Content)
	assert.Equal(t, "Label2", *updated.Label)
}

func TestAssetService_UpdateNote_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	created, err := svc.CreateNote("user-1", proj.ID, "Original note", "Label1")
	require.NoError(t, err)

	_, err = svc.UpdateNote("user-2", created.ID, "Hacked", "Hacked")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestAssetService_ListByProject(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj1, _ := projSvc.Create("user-1", "项目A")
	proj2, _ := projSvc.Create("user-1", "项目B")

	svc.CreateNote("user-1", proj1.ID, "Note for A", "A")
	svc.CreateNote("user-1", proj1.ID, "Another note for A", "A2")
	svc.CreateNote("user-1", proj2.ID, "Note for B", "B")

	assets, err := svc.ListByProject("user-1", proj1.ID)
	assert.NoError(t, err)
	assert.Len(t, assets, 2)
	for _, a := range assets {
		assert.Equal(t, proj1.ID, a.ProjectID)
	}
}

func TestAssetService_DeleteAsset(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	// Upload a file so it has a URI on disk
	asset, err := svc.UploadFile("user-1", proj.ID, "resume.pdf", []byte("%PDF fake"), 10)
	require.NoError(t, err)
	require.True(t, storage.Exists(*asset.URI))

	err = svc.DeleteAsset("user-1", asset.ID)
	assert.NoError(t, err)

	// File should be removed
	assert.False(t, storage.Exists(*asset.URI))

	// Asset record should be gone
	var found models.Asset
	result := db.First(&found, asset.ID)
	assert.Error(t, result.Error)
}

func TestAssetService_DeleteAsset_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	asset, err := svc.CreateNote("user-1", proj.ID, "Secret note", "Private")
	require.NoError(t, err)

	err = svc.DeleteAsset("user-2", asset.ID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestAssetService_DeleteProjectAssets(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	// Create multiple assets: file + note
	fileAsset, err := svc.UploadFile("user-1", proj.ID, "doc.pdf", []byte("%PDF fake"), 10)
	require.NoError(t, err)
	_, err = svc.CreateNote("user-1", proj.ID, "Some note", "Note")
	require.NoError(t, err)

	err = svc.DeleteProjectAssets("user-1", proj.ID)
	assert.NoError(t, err)

	// File should be removed
	assert.False(t, storage.Exists(*fileAsset.URI))

	// All assets for this project should be gone
	var assets []models.Asset
	db.Where("project_id = ?", proj.ID).Find(&assets)
	assert.Len(t, assets, 0)
}
