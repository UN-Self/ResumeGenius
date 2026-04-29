# Storage Logical Key Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `FileStorage` 接口抽到 `shared/storage/` 层，`Save` 返回逻辑 key 而非物理路径，数据库存逻辑 key，解析时通过 storage 层 `Resolve` 回物理路径，为未来迁移 S3/MinIO 做准备。

**Architecture:** 把 `FileStorage` 接口从 `intake/storage.go` 移到 `backend/internal/shared/storage/`。intake 和 parsing 模块都依赖 shared 层，互不耦合。`Save` 返回逻辑 key（如 `1/uuid_filename.docx`），新增 `Resolve(key)` 方法在运行时拼接物理路径。

**Tech Stack:** Go 1.22+, GORM, Gin, testing (stdlib + testify)

---

### Task 1: 创建 shared/storage 层 + LocalStorage 实现

**Files:**
- Create: `backend/internal/shared/storage/storage.go`

**Step 1: 编写 failing test**

在 `backend/internal/shared/storage/storage_test.go` 中编写测试：

```go
package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
)

func TestLocalStorage_Save_ReturnsLogicalKey(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)

	key, err := s.Save(1, "resume.pdf", []byte("fake pdf"))
	require.NoError(t, err)

	// key 应该是相对路径，不含 baseDir
	assert.Equal(t, "1/resume.pdf", key)
	assert.False(t, filepath.IsAbs(key))
}

func TestLocalStorage_Save_CreatesUniqueFileName(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)

	key1, _ := s.Save(1, "resume.pdf", []byte("a"))
	key2, _ := s.Save(1, "resume.pdf", []byte("b"))

	assert.NotEqual(t, key1, key2)
	assert.True(t, strings.HasPrefix(key1, "1/"))
	assert.True(t, strings.HasPrefix(key2, "1/"))
}

func TestLocalStorage_Resolve_ReturnsFullPath(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)

	key, _ := s.Save(1, "test.docx", []byte("data"))
	fullPath, err := s.Resolve(key)
	require.NoError(t, err)

	assert.True(t, filepath.IsAbs(fullPath))
	assert.FileExists(t, fullPath)
}

func TestLocalStorage_Delete_WorksByKey(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)

	key, _ := s.Save(1, "to-delete.pdf", []byte("data"))
	require.True(t, s.Exists(key))

	err := s.Delete(key)
	require.NoError(t, err)
	assert.False(t, s.Exists(key))
}

func TestLocalStorage_Delete_NonExistentReturnsNil(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)

	err := s.Delete("1/nonexistent.pdf")
	assert.NoError(t, err)
}
```

**Step 2: 运行测试验证失败**

Run: `cd backend && go test ./internal/shared/storage/... -v`
Expected: FAIL — package 不存在

**Step 3: 实现 storage 接口和 LocalStorage**

创建 `backend/internal/shared/storage/storage.go`：

```go
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage defines the contract for file persistence.
// Implementations map logical keys to physical storage locations.
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

// Delete removes the file identified by the logical key.
func (s *LocalStorage) Delete(key string) error {
	fullPath, err := s.Resolve(key)
	if err != nil {
		return err
	}
	if !s.Exists(key) {
		return nil
	}
	return os.Remove(fullPath)
}

// Exists reports whether the file identified by the logical key exists on disk.
func (s *LocalStorage) Exists(key string) bool {
	fullPath, err := s.Resolve(key)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullPath)
	return err == nil
}

// Resolve maps a logical key to an absolute filesystem path.
func (s *LocalStorage) Resolve(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("storage: resolve empty key")
	}
	return filepath.Join(s.baseDir, key), nil
}
```

**Step 4: 运行测试验证通过**

Run: `cd backend && go test ./internal/shared/storage/... -v`
Expected: PASS

**Step 5: 提交**

```bash
git add backend/internal/shared/storage/
git commit -m "feat: extract FileStorage interface to shared/storage with logical key support"
```

---

### Task 2: intake 模块适配 shared/storage

**Files:**
- Modify: `backend/internal/modules/intake/storage.go` — 改为 re-export shared/storage
- Modify: `backend/internal/modules/intake/service.go` — `UploadFile` 存逻辑 key，`Delete`/`DeleteProjectAssets` 用逻辑 key
- Modify: `backend/internal/modules/intake/service_test.go` — 适配逻辑 key 断言

**Step 1: 修改 `intake/storage.go` 为 re-export**

将 `intake/storage.go` 改为从 shared 层 re-export：

```go
package intake

import "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"

// FileStorage is re-exported from shared/storage for backward compatibility.
type FileStorage = storage.FileStorage

// NewLocalStorage delegates to shared/storage.NewLocalStorage.
func NewLocalStorage(baseDir string) *storage.LocalStorage {
	return storage.NewLocalStorage(baseDir)
}
```

**Step 2: 运行现有测试确认编译通过（此时行为不变，但 Save 返回逻辑 key）**

Run: `cd backend && go build ./internal/modules/intake/...`
Expected: 编译通过（注意 `Save` 现在返回逻辑 key，`Exists` 和 `Delete` 也接受逻辑 key）

**Step 3: 运行 intake 测试**

Run: `cd backend && go test ./internal/modules/intake/... -v`

预期变化：
- `TestAssetService_UploadFile` 中 `strings.HasSuffix(*asset.URI, "resume.pdf")` 仍通过（逻辑 key 末尾包含文件名）
- `TestAssetService_DeleteAsset` 中 `storage.Exists(*asset.URI)` 现在用逻辑 key 查找，应通过
- `TestAssetService_DeleteProjectAssets` 同理

Expected: PASS

**Step 4: 提交**

```bash
git add backend/internal/modules/intake/storage.go backend/internal/modules/intake/service.go backend/internal/modules/intake/service_test.go
git commit -m "refactor: intake module uses shared/storage with logical keys"
```

---

### Task 3: parsing 模块注入 storage，解析前 Resolve

**Files:**
- Modify: `backend/internal/modules/parsing/service.go` — 注入 `storage.FileStorage`，PDF/DOCX 解析前 Resolve
- Modify: `backend/internal/modules/parsing/service_test.go` — 适配
- Modify: `backend/internal/modules/parsing/routes.go` — 传入 storage

**Step 1: 编写 failing test — 解析 DOCX 前需要 Resolve**

在 `service_test.go` 中新增测试：

```go
func TestParseDOCXAsset_ResolveLogicalKey(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir)

	// 写入真实文件
	logicalKey, err := store.Save(1, "resume.docx", []byte("fake docx"))
	require.NoError(t, err)

	// Asset.URI 存的是逻辑 key
	docxParser := &stubDocxParser{
		result: &ParsedContent{Text: "docx text"},
	}
	svc := NewParsingService(nil, nil, docxParser, nil, store)

	parsed, err := svc.parseAsset(models.Asset{
		ID:   22,
		Type: AssetTypeResumeDOCX,
		URI:  &logicalKey,
	})
	require.NoError(t, err)

	// docxParser 应该收到解析后的物理路径
	expectedPath, _ := store.Resolve(logicalKey)
	assert.Equal(t, expectedPath, docxParser.calledWith)
	assert.Equal(t, "docx text", parsed.Text)
}
```

**Step 2: 运行测试验证失败**

Run: `cd backend && go test ./internal/modules/parsing/... -run TestParseDOCXAsset_ResolveLogicalKey -v`
Expected: FAIL — `NewParsingService` 签名不匹配（参数不够）或 `parseDOCXAsset` 未 Resolve

**Step 3: 修改 parsing/service.go**

a) `ParsingService` 新增 `storage` 字段：

```go
import (
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"
	// ...
)

type ParsingService struct {
	db           *gorm.DB
	pdfParser    PdfParser
	docxParser   DocxParser
	gitExtractor GitExtractor
	storage      storage.FileStorage

	projectExists     func(projectID uint) (bool, error)
	listProjectAssets func(projectID uint) ([]models.Asset, error)
}
```

b) `NewParsingService` 新增 `storage` 参数：

```go
func NewParsingService(db *gorm.DB, pdfParser PdfParser, docxParser DocxParser, gitExtractor GitExtractor, store storage.FileStorage) *ParsingService {
	// ...
	svc.storage = store
	return svc
}
```

c) `parsePDFAsset` 和 `parseDOCXAsset` 解析前 Resolve：

```go
func (s *ParsingService) parsePDFAsset(asset models.Asset) (*ParsedContent, error) {
	if s.pdfParser == nil {
		return nil, ErrPDFParserNotConfigured
	}
	path, err := s.resolveAssetPath(asset)
	if err != nil {
		return nil, err
	}
	parsed, err := s.pdfParser.Parse(path)
	if err != nil {
		return nil, err
	}
	return attachAssetMetadata(asset, parsed), nil
}

func (s *ParsingService) parseDOCXAsset(asset models.Asset) (*ParsedContent, error) {
	if s.docxParser == nil {
		return nil, ErrDOCXParserNotConfigured
	}
	path, err := s.resolveAssetPath(asset)
	if err != nil {
		return nil, err
	}
	parsed, err := s.docxParser.Parse(path)
	if err != nil {
		return nil, err
	}
	return attachAssetMetadata(asset, parsed), nil
}

func (s *ParsingService) resolveAssetPath(asset models.Asset) (string, error) {
	if s.storage == nil {
		return requireAssetURI(asset)
	}
	key, err := requireAssetURI(asset)
	if err != nil {
		return "", err
	}
	return s.storage.Resolve(key)
}
```

注意：`parseGitAsset` 不改（Git URI 是 URL，不是文件路径），`parseNoteAsset` 也不改。

**Step 4: 更新 `parsing/routes.go`**

```go
import "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, store storage.FileStorage) {
	rg.POST("/parsing/parse", func(c *gin.Context) {
		c.JSON(200, gin.H{"module": "parsing", "status": "stub"})
	})
}
```

（routes.go 中 parsing 还是 stub，但签名要改。后续接入真实 handler 时再注入 `ParsingService`。）

**Step 5: 更新现有 parsing 测试**

所有 `NewParsingService(nil, ...)` 调用新增 `nil` 参数（storage 为 nil 时 fallback 到原始行为）：

```go
// 之前：
NewParsingService(nil, pdfParser, docxParser, nil)
// 之后：
NewParsingService(nil, pdfParser, docxParser, nil, nil)
```

**Step 6: 运行全部 parsing 测试**

Run: `cd backend && go test ./internal/modules/parsing/... -v`
Expected: PASS

**Step 7: 提交**

```bash
git add backend/internal/modules/parsing/
git commit -m "feat: parsing module resolves logical keys via shared/storage"
```

---

### Task 4: main.go 创建共享 storage 实例

**Files:**
- Modify: `backend/cmd/server/main.go`

**Step 1: 修改 setupRouter**

```go
import "github.com/UN-Self/ResumeGenius/backend/internal/shared/storage"

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.Logger())

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(uploadDir, 0755)

	store := storage.NewLocalStorage(uploadDir)

	// ... auth setup ...

	intake.RegisterRoutes(authed, db, uploadDir)
	parsing.RegisterRoutes(authed, db, store)
	// ...
}
```

注意：`intake.RegisterRoutes` 仍然接收 `uploadDir` 字符串（内部自己创建 `LocalStorage`），后续可以把 intake 也改为直接接收 `storage.FileStorage`，但那是另一步优化，本次不改。

**Step 2: 编译检查**

Run: `cd backend && go build ./cmd/server/...`
Expected: PASS

**Step 3: 运行全部测试**

Run: `cd backend && go test ./... -v`
Expected: ALL PASS

**Step 4: 提交**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire shared storage into parsing module in main.go"
```

---

### Task 5: 清理旧 intake/storage.go

**Files:**
- Delete: `backend/internal/modules/intake/storage.go`（所有代码已迁移到 shared/storage，只剩下 re-export）

**Step 1: 将 re-export 移到 `intake/routes.go` 或直接在各处引用 shared/storage**

选项 A（推荐）：删除 `intake/storage.go`，在 `intake/service.go` 和 `intake/routes.go` 中直接引用 `shared/storage`：

- `service.go` 中 `FileStorage` → `storage.FileStorage`
- `routes.go` 中 `NewLocalStorage` → `storage.NewLocalStorage`

**Step 2: 编译 + 测试**

Run: `cd backend && go build ./... && go test ./... -v`
Expected: ALL PASS

**Step 3: 提交**

```bash
git add -A
git commit -m "refactor: remove intake/storage.go re-export, use shared/storage directly"
```
