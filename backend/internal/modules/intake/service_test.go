package intake

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	sharedstorage "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
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

	var deleted models.Project
	require.NoError(t, db.Unscoped().First(&deleted, created.ID).Error)
	assert.True(t, deleted.DeletedAt.Valid)
}

func TestProjectService_Delete_CascadeClearsDraftAndRelations(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	proj, _ := svc.Create("user-1", "级联删除测试")

	// Create a draft linked to the project
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<p>hello</p>"}
	require.NoError(t, db.Create(&draft).Error)

	// Set current_draft_id (circular FK)
	require.NoError(t, db.Model(&proj).Update("current_draft_id", draft.ID).Error)

	// Create downstream: version, ai_session, ai_message, ai_tool_call
	version := models.Version{DraftID: draft.ID, HTMLSnapshot: "<p>v1</p>"}
	require.NoError(t, db.Create(&version).Error)

	session := models.AISession{DraftID: draft.ID}
	require.NoError(t, db.Create(&session).Error)

	toolCall := models.AIToolCall{
		SessionID: session.ID,
		ToolName:  "read_html",
		Params:    models.JSONB{"path": "resume.html"},
		Status:    "completed",
	}
	require.NoError(t, db.Create(&toolCall).Error)

	msg := models.AIMessage{SessionID: session.ID, Role: "user", Content: "hello", ToolCallID: &toolCall.ID}
	require.NoError(t, db.Create(&msg).Error)

	// Delete project
	err := svc.Delete("user-1", proj.ID)
	assert.NoError(t, err)

	var draftCount, versionCount, sessionCount, msgCount, toolCallCount int64
	require.NoError(t, db.Model(&models.Draft{}).Where("project_id = ?", proj.ID).Count(&draftCount).Error)
	require.NoError(t, db.Model(&models.Version{}).Where("draft_id = ?", draft.ID).Count(&versionCount).Error)
	require.NoError(t, db.Model(&models.AISession{}).Where("draft_id = ?", draft.ID).Count(&sessionCount).Error)
	require.NoError(t, db.Model(&models.AIMessage{}).Where("session_id = ?", session.ID).Count(&msgCount).Error)
	require.NoError(t, db.Model(&models.AIToolCall{}).Where("session_id = ?", session.ID).Count(&toolCallCount).Error)

	assert.Zero(t, draftCount, "active drafts should be hidden after soft delete")
	assert.Zero(t, versionCount, "active versions should be hidden after soft delete")
	assert.Zero(t, sessionCount, "active ai_sessions should be hidden after soft delete")
	assert.Zero(t, msgCount, "active ai_messages should be hidden after soft delete")
	assert.Zero(t, toolCallCount, "active ai_tool_calls should be hidden after soft delete")

	var deletedDraft models.Draft
	require.NoError(t, db.Unscoped().First(&deletedDraft, draft.ID).Error)
	assert.True(t, deletedDraft.DeletedAt.Valid)

	var deletedVersion models.Version
	require.NoError(t, db.Unscoped().First(&deletedVersion, version.ID).Error)
	assert.True(t, deletedVersion.DeletedAt.Valid)

	var deletedSession models.AISession
	require.NoError(t, db.Unscoped().First(&deletedSession, session.ID).Error)
	assert.True(t, deletedSession.DeletedAt.Valid)

	var deletedMessage models.AIMessage
	require.NoError(t, db.Unscoped().First(&deletedMessage, msg.ID).Error)
	assert.True(t, deletedMessage.DeletedAt.Valid)

	var deletedToolCall models.AIToolCall
	require.NoError(t, db.Unscoped().First(&deletedToolCall, toolCall.ID).Error)
	assert.True(t, deletedToolCall.DeletedAt.Valid)
}

func TestProjectService_Delete_CascadeClearsAssets(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)
	storage := NewLocalStorage(t.TempDir())
	assetSvc := NewAssetService(db, storage)

	proj, _ := svc.Create("user-1", "资产删除测试")

	// Create file and note assets
	fileAsset, err := assetSvc.UploadFile("user-1", proj.ID, "resume.pdf", []byte("%PDF fake"), 10)
	require.NoError(t, err)
	noteAsset, err := assetSvc.CreateNote("user-1", proj.ID, "note content", "Label")
	require.NoError(t, err)

	err = svc.Delete("user-1", proj.ID)
	assert.NoError(t, err)

	assert.True(t, storage.Exists(*fileAsset.URI))

	var assetCount int64
	require.NoError(t, db.Model(&models.Asset{}).Where("project_id = ?", proj.ID).Count(&assetCount).Error)
	assert.Zero(t, assetCount)

	requireSoftDeletedAsset(t, db, fileAsset.ID)
	requireSoftDeletedAsset(t, db, noteAsset.ID)
}

func TestProjectService_Delete_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	err := svc.Delete("user-2", created.ID)
	assert.Error(t, err)
}

func saveTestStoredFile(t *testing.T, store sharedstorage.FileStorage, userID string, filename string, data []byte) string {
	t.Helper()

	key, err := store.Save(userID, sharedstorage.SHA256Hex(data), filename, data)
	require.NoError(t, err)
	return key
}

func requireSoftDeletedAsset(t *testing.T, db *gorm.DB, assetID uint) models.Asset {
	t.Helper()

	var asset models.Asset
	require.NoError(t, db.Unscoped().First(&asset, assetID).Error)
	require.True(t, asset.DeletedAt.Valid)
	return asset
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
	assert.NotNil(t, asset.FileHash)
	assert.Equal(t, sharedstorage.SHA256Hex(data), *asset.FileHash)
	assert.Equal(t, "user-1/"+*asset.FileHash+".pdf", *asset.URI)
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

func TestAssetService_UploadFileWithReplacement_ReplacesSameNameAssetInProject(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	oldAsset, err := svc.UploadFile("user-1", proj.ID, "resume.pdf", []byte("%PDF old"), 8)
	require.NoError(t, err)

	derivedBytes := []byte("img")
	derivedKey := saveTestStoredFile(t, storage, proj.UserID, "derived.png", derivedBytes)
	derivedLabel := "Derived image"
	derivedAsset := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_image",
		URI:       &derivedKey,
		Label:     &derivedLabel,
		Metadata: models.JSONB{
			"parsing": map[string]interface{}{
				"derived":         true,
				"source_asset_id": oldAsset.ID,
			},
		},
	}
	require.NoError(t, db.Create(&derivedAsset).Error)
	require.NoError(t, db.Model(&models.Asset{}).Where("id = ?", oldAsset.ID).Update("metadata", models.JSONB{
		"parsing": map[string]interface{}{
			"original_filename":       "resume.pdf",
			"derived_image_asset_ids": []interface{}{derivedAsset.ID},
		},
	}).Error)

	replaceID := oldAsset.ID
	replacedAsset, err := svc.UploadFileWithReplacement("user-1", proj.ID, "resume.pdf", []byte("%PDF new"), 8, &replaceID)
	require.NoError(t, err)
	require.NotNil(t, replacedAsset)
	assert.NotEqual(t, oldAsset.ID, replacedAsset.ID)
	require.NotNil(t, replacedAsset.URI)
	assert.True(t, storage.Exists(*replacedAsset.URI))
	assert.True(t, storage.Exists(*oldAsset.URI))
	assert.True(t, storage.Exists(derivedKey))

	requireSoftDeletedAsset(t, db, oldAsset.ID)
	requireSoftDeletedAsset(t, db, derivedAsset.ID)

	var keptAssets []models.Asset
	require.NoError(t, db.Where("project_id = ?", proj.ID).Find(&keptAssets).Error)
	require.Len(t, keptAssets, 1)
	assert.Equal(t, replacedAsset.ID, keptAssets[0].ID)
}

func TestAssetService_UploadFile_RestoresSoftDeletedSameFileAndDerivedAssets(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	projA, err := projSvc.Create("user-1", "Project A")
	require.NoError(t, err)
	projB, err := projSvc.Create("user-1", "Project B")
	require.NoError(t, err)

	originalBytes := []byte("%PDF restore me")
	oldAsset, err := svc.UploadFile("user-1", projA.ID, "resume.pdf", originalBytes, int64(len(originalBytes)))
	require.NoError(t, err)

	derivedBytes := []byte("derived-image")
	derivedHash := sharedstorage.SHA256Hex(derivedBytes)
	derivedKey, err := storage.Save(projA.UserID, derivedHash, ".png", derivedBytes)
	require.NoError(t, err)
	derivedLabel := "Derived image"
	derivedAsset := models.Asset{
		ProjectID: projA.ID,
		Type:      "resume_image",
		URI:       &derivedKey,
		Label:     &derivedLabel,
		FileHash:  &derivedHash,
		Metadata: models.JSONB{
			"parsing": map[string]interface{}{
				"derived":         true,
				"source_asset_id": oldAsset.ID,
			},
		},
	}
	require.NoError(t, db.Create(&derivedAsset).Error)

	parsedContent := "Persisted parsed text"
	require.NoError(t, db.Model(&models.Asset{}).Where("id = ?", oldAsset.ID).Updates(map[string]interface{}{
		"content": parsedContent,
		"metadata": models.JSONB{
			"parsing": map[string]interface{}{
				"original_filename":       "resume.pdf",
				"source_deleted":          true,
				"derived_image_asset_ids": []interface{}{derivedAsset.ID},
			},
		},
		"uri": nil,
	}).Error)

	require.NoError(t, svc.DeleteAsset("user-1", oldAsset.ID))
	requireSoftDeletedAsset(t, db, oldAsset.ID)
	requireSoftDeletedAsset(t, db, derivedAsset.ID)

	restoredAsset, err := svc.UploadFile("user-1", projB.ID, "resume.pdf", originalBytes, int64(len(originalBytes)))
	require.NoError(t, err)
	require.NotNil(t, restoredAsset)
	assert.Equal(t, oldAsset.ID, restoredAsset.ID)
	assert.Equal(t, projB.ID, restoredAsset.ProjectID)
	require.NotNil(t, restoredAsset.FileHash)
	require.NotNil(t, oldAsset.FileHash)
	assert.Equal(t, *oldAsset.FileHash, *restoredAsset.FileHash)
	require.NotNil(t, restoredAsset.Content)
	assert.Equal(t, parsedContent, *restoredAsset.Content)
	require.NotNil(t, restoredAsset.URI)
	assert.True(t, storage.Exists(*restoredAsset.URI))

	var activeAssets []models.Asset
	require.NoError(t, db.Where("project_id = ?", projB.ID).Order("id asc").Find(&activeAssets).Error)
	require.Len(t, activeAssets, 2)
	assert.Equal(t, restoredAsset.ID, activeAssets[0].ID)
	assert.Equal(t, derivedAsset.ID, activeAssets[1].ID)

	var restoredDerived models.Asset
	require.NoError(t, db.First(&restoredDerived, derivedAsset.ID).Error)
	assert.Equal(t, projB.ID, restoredDerived.ProjectID)
	assert.True(t, storage.Exists(derivedKey))
}

func TestAssetService_UploadFileWithReplacement_RejectsCrossProjectReplace(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	projA, err := projSvc.Create("user-1", "Project A")
	require.NoError(t, err)
	projB, err := projSvc.Create("user-1", "Project B")
	require.NoError(t, err)

	oldAsset, err := svc.UploadFile("user-1", projA.ID, "resume.pdf", []byte("%PDF old"), 8)
	require.NoError(t, err)

	replaceID := oldAsset.ID
	_, err = svc.UploadFileWithReplacement("user-1", projB.ID, "resume.pdf", []byte("%PDF new"), 8, &replaceID)
	assert.ErrorIs(t, err, ErrReplaceAssetMismatch)

	var assetsA []models.Asset
	require.NoError(t, db.Where("project_id = ?", projA.ID).Find(&assetsA).Error)
	assert.Len(t, assetsA, 1)

	var assetsB []models.Asset
	require.NoError(t, db.Where("project_id = ?", projB.ID).Find(&assetsB).Error)
	assert.Len(t, assetsB, 0)
}

func TestAssetService_CreateGitRepo(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	assets, err := svc.CreateGitRepo("user-1", proj.ID, []string{"https://github.com/example/resume.git"})
	assert.NoError(t, err)
	assert.Len(t, assets, 1)
	assert.Equal(t, "git_repo", assets[0].Type)
	assert.Equal(t, "https://github.com/example/resume.git", *assets[0].URI)
}

func TestAssetService_CreateGitRepo_InvalidURL(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	_, err = svc.CreateGitRepo("user-1", proj.ID, []string{"not-a-url"})
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

func TestAssetService_UpdateAsset_UpdatesPersistedFileContentAndLabel(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	content := "Original parsed text"
	label := "Original label"
	metadata := models.JSONB{
		"parsing": map[string]interface{}{
			"status": "success",
		},
	}
	asset := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_pdf",
		Content:   &content,
		Label:     &label,
		Metadata:  metadata,
	}
	require.NoError(t, db.Create(&asset).Error)

	newContent := "Updated parsed text"
	newLabel := "Updated label"
	updated, err := svc.UpdateAsset("user-1", asset.ID, &newContent, &newLabel)
	assert.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated parsed text", *updated.Content)
	assert.Equal(t, "Updated label", *updated.Label)
	assert.Equal(t, "resume_pdf", updated.Type)

	parsing := getAssetParsingMetadata(t, updated.Metadata)
	assert.Equal(t, "success", parsing["status"])
	assert.Equal(t, true, parsing["updated_by_user"])
}

func TestAssetService_UpdateAsset_PartialUpdateKeepsExistingFields(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	content := "Keep this content"
	label := "Original label"
	asset, err := svc.CreateNote("user-1", proj.ID, content, label)
	require.NoError(t, err)

	newLabel := "Only label changed"
	updated, err := svc.UpdateAsset("user-1", asset.ID, nil, &newLabel)
	assert.NoError(t, err)
	require.NotNil(t, updated)
	assert.NotNil(t, updated.Content)
	assert.Equal(t, content, *updated.Content)
	assert.NotNil(t, updated.Label)
	assert.Equal(t, newLabel, *updated.Label)
	assert.Nil(t, updated.Metadata)
}

func TestAssetService_UpdateAsset_LeavesParsingMetadataCleanWhenOnlyLabelChanges(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	content := "Original parsed text"
	label := "Original label"
	metadata := models.JSONB{
		"parsing": map[string]interface{}{
			"status": "success",
		},
	}
	asset := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_pdf",
		Content:   &content,
		Label:     &label,
		Metadata:  metadata,
	}
	require.NoError(t, db.Create(&asset).Error)

	newLabel := "Only label changed"
	updated, err := svc.UpdateAsset("user-1", asset.ID, nil, &newLabel)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, metadata, updated.Metadata)
}

func TestAssetService_UpdateAsset_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	asset, err := svc.CreateNote("user-1", proj.ID, "Original note", "Label1")
	require.NoError(t, err)

	newContent := "Hacked"
	_, err = svc.UpdateAsset("user-2", asset.ID, &newContent, nil)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func getAssetParsingMetadata(t *testing.T, metadata models.JSONB) map[string]interface{} {
	t.Helper()

	require.NotNil(t, metadata)
	rawParsing, ok := metadata["parsing"]
	require.True(t, ok)

	parsing, ok := rawParsing.(map[string]interface{})
	require.True(t, ok)
	return parsing
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

	assert.True(t, storage.Exists(*asset.URI))

	var found models.Asset
	result := db.First(&found, asset.ID)
	assert.Error(t, result.Error)

	deleted := requireSoftDeletedAsset(t, db, asset.ID)
	assert.Equal(t, asset.ID, deleted.ID)
}

func TestAssetService_DeleteAsset_RemovesDerivedImageAssets(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "测试项目")
	require.NoError(t, err)

	sourceAsset, err := svc.UploadFile("user-1", proj.ID, "resume.pdf", []byte("%PDF fake"), 10)
	require.NoError(t, err)

	derivedKey1 := saveTestStoredFile(t, storage, proj.UserID, "derived-1.png", []byte("img-1"))
	derivedLabel1 := "简历图片 1"
	derivedAsset1 := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_image",
		URI:       &derivedKey1,
		Label:     &derivedLabel1,
		Metadata: models.JSONB{
			"parsing": map[string]interface{}{
				"derived":         true,
				"source_asset_id": sourceAsset.ID,
			},
		},
	}
	require.NoError(t, db.Create(&derivedAsset1).Error)

	derivedKey2 := saveTestStoredFile(t, storage, proj.UserID, "derived-2.png", []byte("img-2"))
	derivedLabel2 := "简历图片 2"
	derivedAsset2 := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_image",
		URI:       &derivedKey2,
		Label:     &derivedLabel2,
		Metadata: models.JSONB{
			"parsing": map[string]interface{}{
				"derived":         true,
				"source_asset_id": sourceAsset.ID,
			},
		},
	}
	require.NoError(t, db.Create(&derivedAsset2).Error)

	unrelatedKey := saveTestStoredFile(t, storage, proj.UserID, "standalone.png", []byte("standalone"))
	unrelatedLabel := "独立图片"
	unrelatedAsset := models.Asset{
		ProjectID: proj.ID,
		Type:      "resume_image",
		URI:       &unrelatedKey,
		Label:     &unrelatedLabel,
	}
	require.NoError(t, db.Create(&unrelatedAsset).Error)

	require.NoError(t, db.Model(&models.Asset{}).Where("id = ?", sourceAsset.ID).Update("metadata", models.JSONB{
		"parsing": map[string]interface{}{
			"derived_image_asset_ids": []interface{}{derivedAsset1.ID, derivedAsset2.ID},
		},
	}).Error)

	require.True(t, storage.Exists(*sourceAsset.URI))
	require.True(t, storage.Exists(derivedKey1))
	require.True(t, storage.Exists(derivedKey2))
	require.True(t, storage.Exists(unrelatedKey))

	err = svc.DeleteAsset("user-1", sourceAsset.ID)
	assert.NoError(t, err)

	assert.True(t, storage.Exists(*sourceAsset.URI))
	assert.True(t, storage.Exists(derivedKey1))
	assert.True(t, storage.Exists(derivedKey2))
	assert.True(t, storage.Exists(unrelatedKey))

	requireSoftDeletedAsset(t, db, sourceAsset.ID)
	requireSoftDeletedAsset(t, db, derivedAsset1.ID)
	requireSoftDeletedAsset(t, db, derivedAsset2.ID)

	var kept models.Asset
	require.NoError(t, db.First(&kept, unrelatedAsset.ID).Error)
	assert.Equal(t, unrelatedAsset.ID, kept.ID)
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

func TestAssetService_DeleteAsset_IgnoresStorageDeleteFailuresBecauseFilesAreRetained(t *testing.T) {
	db := SetupTestDB(t)
	storage := newFailingDeleteStorage(errors.New("disk busy"))
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	asset, err := svc.UploadFile("user-1", proj.ID, "resume.pdf", []byte("%PDF fake"), 10)
	require.NoError(t, err)

	err = svc.DeleteAsset("user-1", asset.ID)
	require.NoError(t, err)

	requireSoftDeletedAsset(t, db, asset.ID)
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

	assert.True(t, storage.Exists(*fileAsset.URI))

	var assets []models.Asset
	require.NoError(t, db.Where("project_id = ?", proj.ID).Find(&assets).Error)
	assert.Len(t, assets, 0)

	requireSoftDeletedAsset(t, db, fileAsset.ID)
}

func TestAssetService_DeleteProjectAssets_IgnoresStorageDeleteFailuresBecauseFilesAreRetained(t *testing.T) {
	db := SetupTestDB(t)
	storage := newFailingDeleteStorage(errors.New("disk busy"))
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user-1", "Test Project")
	require.NoError(t, err)

	fileAsset, err := svc.UploadFile("user-1", proj.ID, "doc.pdf", []byte("%PDF fake"), 10)
	require.NoError(t, err)
	_, err = svc.CreateNote("user-1", proj.ID, "Some note", "Note")
	require.NoError(t, err)

	err = svc.DeleteProjectAssets("user-1", proj.ID)
	require.NoError(t, err)

	var keptAssets []models.Asset
	require.NoError(t, db.Where("project_id = ?", proj.ID).Find(&keptAssets).Error)
	assert.Len(t, keptAssets, 0)

	requireSoftDeletedAsset(t, db, fileAsset.ID)
}
