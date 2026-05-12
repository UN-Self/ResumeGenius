# Fix Asset Restore Scope: Limit to Same Project

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent `findDeletedAssetByFileHash` from restoring soft-deleted assets across projects — restore should only happen within the same project.

**Architecture:** Add `projectID` filter to the SQL query in `findDeletedAssetByFileHash`, ensuring only assets belonging to the target project are considered for restoration. Cross-project uploads with the same file hash will create fresh assets and go through full re-parsing.

**Tech Stack:** Go, GORM (SQLite for tests), existing test infrastructure in `intake/testutil.go`

---

### Task 1: Write failing test for cross-project restore rejection

**Files:**
- Modify: `backend/internal/modules/intake/service_test.go`

- [ ] **Step 1: Write the failing test**

Add a new test after `TestAssetService_UploadFile_RestoresSoftDeletedSameFileAndDerivedAssets` (after line 398):

```go
func TestAssetService_UploadFile_DoesNotRestoreSoftDeletedAssetFromOtherProject(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	projA, err := projSvc.Create("user-1", "Project A")
	require.NoError(t, err)
	projB, err := projSvc.Create("user-1", "Project B")
	require.NoError(t, err)

	fileBytes := []byte("%PDF test content")
	assetA, err := svc.UploadFile("user-1", projA.ID, "resume.pdf", fileBytes, int64(len(fileBytes)))
	require.NoError(t, err)

	parsedContent := "Old parsed text from project A"
	require.NoError(t, db.Model(&models.Asset{}).Where("id = ?", assetA.ID).Updates(map[string]interface{}{
		"content": parsedContent,
	}).Error)

	require.NoError(t, svc.DeleteAsset("user-1", assetA.ID))
	requireSoftDeletedAsset(t, db, assetA.ID)

	restoredAsset, err := svc.UploadFile("user-1", projB.ID, "resume.pdf", fileBytes, int64(len(fileBytes)))
	require.NoError(t, err)
	require.NotNil(t, restoredAsset)

	assert.NotEqual(t, assetA.ID, restoredAsset.ID, "should create new asset, not restore from another project")
	assert.Equal(t, projB.ID, restoredAsset.ProjectID)
	assert.Nil(t, restoredAsset.Content, "new asset should have no content until parsed")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/intake/ -run TestAssetService_UploadFile_DoesNotRestoreSoftDeletedAssetFromOtherProject -v`
Expected: FAIL — `assert.NotEqual` will fail because the current code restores the cross-project asset (same ID).

---

### Task 2: Implement the project-scoped restore

**Files:**
- Modify: `backend/internal/modules/intake/service.go:695-709`

- [ ] **Step 1: Write minimal implementation**

Change `findDeletedAssetByFileHash` to accept and filter by `projectID`:

```go
func (s *AssetService) findDeletedAssetByFileHash(userID string, projectID uint, fileHash string) (*models.Asset, error) {
	var asset models.Asset
	err := s.db.Unscoped().
		Joins("JOIN projects ON projects.id = assets.project_id").
		Where("projects.user_id = ? AND assets.project_id = ? AND assets.file_hash = ? AND assets.deleted_at IS NOT NULL", userID, projectID, fileHash).
		Order("assets.deleted_at DESC").
		First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find deleted asset by file hash: %w", err)
	}
	return &asset, nil
}
```

Update the single call site in `restoreDeletedAssetByFileHash` (line 719):

```go
	deletedAsset, err := s.findDeletedAssetByFileHash(userID, projectID, fileHash)
```

- [ ] **Step 2: Run all intake tests**

Run: `cd backend && go test ./internal/modules/intake/ -v`
Expected: Some tests pass, but `TestAssetService_UploadFile_RestoresSoftDeletedSameFileAndDerivedAssets` will FAIL because it uploads to `projB` but expects restoration of an asset from `projA`.

---

### Task 3: Fix existing cross-project restore test

**Files:**
- Modify: `backend/internal/modules/intake/service_test.go:323-398`

- [ ] **Step 1: Update `TestAssetService_UploadFile_RestoresSoftDeletedSameFileAndDerivedAssets` to use same project**

The test currently uploads to `projA`, deletes, then re-uploads to `projB` (cross-project). Change it to re-upload to `projA` (same project). The key changes:

1. Replace `projB` with `projA` in the second `UploadFile` call (line 375)
2. Update all assertions referencing `projB.ID` to use `projA.ID`

The updated test:

```go
func TestAssetService_UploadFile_RestoresSoftDeletedSameFileAndDerivedAssets(t *testing.T) {
	db := SetupTestDB(t)
	storage := NewLocalStorage(t.TempDir())
	svc := NewAssetService(db, storage)

	projSvc := NewProjectService(db)
	projA, err := projSvc.Create("user-1", "Project A")
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

	require.NoError(t, svc.DeleteAsset("user-1", oldAsset.ID))
	requireSoftDeletedAsset(t, db, oldAsset.ID)
	requireSoftDeletedAsset(t, db, derivedAsset.ID)

	restoredAsset, err := svc.UploadFile("user-1", projA.ID, "resume.pdf", originalBytes, int64(len(originalBytes)))
	require.NoError(t, err)
	require.NotNil(t, restoredAsset)
	assert.Equal(t, oldAsset.ID, restoredAsset.ID)
	assert.Equal(t, projA.ID, restoredAsset.ProjectID)
	require.NotNil(t, restoredAsset.FileHash)
	require.NotNil(t, oldAsset.FileHash)
	assert.Equal(t, *oldAsset.FileHash, *restoredAsset.FileHash)
	assert.Nil(t, restoredAsset.Content, "content should be cleared to force re-parse")
	require.NotNil(t, restoredAsset.URI)
	assert.True(t, storage.Exists(*restoredAsset.URI))

	var activeAssets []models.Asset
	require.NoError(t, db.Where("project_id = ?", projA.ID).Order("id asc").Find(&activeAssets).Error)
	require.Len(t, activeAssets, 2)
	assert.Equal(t, restoredAsset.ID, activeAssets[0].ID)
	assert.Equal(t, derivedAsset.ID, activeAssets[1].ID)

	var restoredDerived models.Asset
	require.NoError(t, db.First(&restoredDerived, derivedAsset.ID).Error)
	assert.Equal(t, projA.ID, restoredDerived.ProjectID)
	assert.True(t, storage.Exists(derivedKey))
}
```

- [ ] **Step 2: Run all intake tests**

Run: `cd backend && go test ./internal/modules/intake/ -v`
Expected: ALL tests PASS.

---

### Task 4: Run full test suite and commit

**Files:** None (verification only)

- [ ] **Step 1: Run full backend test suite**

Run: `cd backend && go test ./...`
Expected: ALL tests PASS.

- [ ] **Step 2: Commit**

```bash
cd backend
git add internal/modules/intake/service.go internal/modules/intake/service_test.go
git commit -m "fix: 限制 asset 恢复范围为同项目，避免跨项目复用旧解析结果"
```
