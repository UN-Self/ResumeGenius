# Git 仓库延迟提取实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 git 仓库提取从上传时立即执行改为 AI 对话中有目的的延迟执行，支持 SSH 密钥管理和多次提取复用。

**Architecture:** 新增 SSHKey 模型独立管理密钥，Asset 表扩展 key_id/status 字段。AI Agent 新增 extract_git_repo 工具，通过便宜模型 subagent 探索仓库，探索报告存为新 Asset。前端改造 GitRepoDialog 支持密钥选择。

**Tech Stack:** Go + GORM + AES-GCM (golang.org/x/crypto), React + TypeScript

---

## 文件结构

### 新建文件
- `backend/internal/shared/models/ssh_key.go` — SSHKey 模型定义
- `backend/internal/shared/crypto/encrypt.go` — AES-GCM 加密/解密工具
- `backend/internal/shared/crypto/encrypt_test.go` — 加密工具测试
- `backend/internal/modules/intake/ssh_service.go` — SSH 密钥 CRUD 服务
- `backend/internal/modules/intake/ssh_service_test.go` — SSH 服务测试
- `backend/internal/modules/intake/ssh_handler.go` — SSH 密钥 API handler
- `backend/internal/modules/agent/git_extractor_tool.go` — extract_git_repo 工具实现
- `backend/internal/modules/agent/git_extractor_tool_test.go` — 工具测试
- `backend/internal/modules/agent/ai_provider.go` — AIProvider 实现（复用 parsing 模块 aiChat）
- `frontend/workbench/src/components/intake/SSHKeySelector.tsx` — SSH 密钥选择组件

### 修改文件
- `backend/internal/shared/models/models.go` — Asset 表新增 KeyID、Status 字段
- `backend/internal/shared/database/database.go` — AutoMigrate 添加 SSHKey
- `backend/internal/modules/intake/routes.go` — 新增 SSH 密钥路由
- `backend/internal/modules/intake/handler.go` — 修改 createGitReq 结构体
- `backend/internal/modules/intake/service.go` — 修改 CreateGitRepo 支持 key_id
- `backend/internal/modules/agent/tool_executor.go` — 注册 extract_git_repo 工具
- `backend/internal/modules/agent/prompt.go` — flow_rules 添加延迟提取引导
- `backend/internal/modules/agent/service.go` — 传递 sshService 给 toolExecutor
- `backend/cmd/server/main.go` — 初始化 sshService 并注入
- `frontend/workbench/src/components/intake/GitRepoDialog.tsx` — 集成 SSH 密钥选择
- `frontend/workbench/src/lib/api-client.ts` — 新增 SSH 密钥 API 调用

---

## Task 1: SSHKey 模型与数据库迁移

**Files:**
- Create: `backend/internal/shared/models/ssh_key.go`
- Modify: `backend/internal/shared/models/models.go:55-67` (Asset 添加字段)
- Modify: `backend/internal/shared/database/database.go:43-58` (AutoMigrate)

- [ ] **Step 1: 创建 SSHKey 模型**

```go
// backend/internal/shared/models/ssh_key.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type SSHKey struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       string         `gorm:"size:36;not null;index" json:"user_id"`
	Alias        string         `gorm:"size:100;not null" json:"alias"`
	EncryptedKey string         `gorm:"type:text;not null" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
```

- [ ] **Step 2: Asset 模型添加 KeyID 和 Status 字段**

修改 `backend/internal/shared/models/models.go` 的 Asset 结构体：

```go
type Asset struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProjectID uint           `gorm:"not null;index" json:"project_id"`
	Type      string         `gorm:"size:50;not null" json:"type"`
	URI       *string        `gorm:"type:text" json:"uri,omitempty"`
	Content   *string        `gorm:"type:text" json:"content,omitempty"`
	Label     *string        `gorm:"size:100" json:"label,omitempty"`
	FileHash  *string        `gorm:"size:64;index" json:"file_hash,omitempty"`
	Metadata  JSONB          `gorm:"type:jsonb" json:"metadata,omitempty"`
	KeyID     *uint          `gorm:"index" json:"key_id,omitempty"`
	Status    string         `gorm:"size:20;not null;default:'active'" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
```

- [ ] **Step 3: AutoMigrate 添加 SSHKey**

修改 `backend/internal/shared/database/database.go` 的 Migrate 函数：

```go
func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Asset{},
		&models.SSHKey{},
		// ... 其余不变
	)
	// ...
}
```

- [ ] **Step 4: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 5: Commit**

```bash
git add backend/internal/shared/models/ssh_key.go backend/internal/shared/models/models.go backend/internal/shared/database/database.go
git commit -m "feat: add SSHKey model and Asset key_id/status fields"
```

---

## Task 2: AES-GCM 加密工具

**Files:**
- Create: `backend/internal/shared/crypto/encrypt.go`
- Create: `backend/internal/shared/crypto/encrypt_test.go`

- [ ] **Step 1: 编写加密工具测试**

```go
// backend/internal/shared/crypto/encrypt_test.go
package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	plaintext := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-content\n-----END OPENSSH PRIVATE KEY-----"

	encrypted, err := Encrypt([]byte(plaintext), key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if string(encrypted) == plaintext {
		t.Fatal("encrypted text should not equal plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Fatalf("decrypted text %q does not match plaintext %q", string(decrypted), plaintext)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := []byte("0123456789abcdef0123456789abcdef")
	key2 := []byte("abcdef0123456789abcdef0123456789")

	encrypted, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd backend && go test ./internal/shared/crypto/... -v`
Expected: FAIL (package does not exist)

- [ ] **Step 3: 实现加密工具**

```go
// backend/internal/shared/crypto/encrypt.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd backend && go test ./internal/shared/crypto/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/shared/crypto/
git commit -m "feat: add AES-GCM encrypt/decrypt utility"
```

---

## Task 3: SSH 密钥服务层

**Files:**
- Create: `backend/internal/modules/intake/ssh_service.go`
- Create: `backend/internal/modules/intake/ssh_service_test.go`

- [ ] **Step 1: 编写 SSH 服务测试**

```go
// backend/internal/modules/intake/ssh_service_test.go
package intake

import (
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.SSHKey{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSSHKeyService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	key, err := svc.Create("user1", "my-deploy-key", "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if key.Alias != "my-deploy-key" {
		t.Fatalf("alias = %q, want %q", key.Alias, "my-deploy-key")
	}
}

func TestSSHKeyService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	svc.Create("user1", "key1", "key-content-1")
	svc.Create("user1", "key2", "key-content-2")
	svc.Create("user2", "key3", "key-content-3")

	keys, err := svc.List("user1")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len = %d, want 2", len(keys))
	}
}

func TestSSHKeyService_GetDecryptedKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	original := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----"
	created, _ := svc.Create("user1", "key1", original)

	decrypted, err := svc.GetDecryptedKey("user1", created.ID)
	if err != nil {
		t.Fatalf("GetDecryptedKey failed: %v", err)
	}
	if decrypted != original {
		t.Fatalf("decrypted = %q, want %q", decrypted, original)
	}
}

func TestSSHKeyService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	created, _ := svc.Create("user1", "key1", "key-content")

	if err := svc.Delete("user1", created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	keys, _ := svc.List("user1")
	if len(keys) != 0 {
		t.Fatalf("len = %d, want 0", len(keys))
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd backend && go test ./internal/modules/intake/... -run TestSSHKey -v`
Expected: FAIL

- [ ] **Step 3: 实现 SSH 服务**

```go
// backend/internal/modules/intake/ssh_service.go
package intake

import (
	"fmt"

	"encoding/base64"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/crypto"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type SSHKeyService struct {
	db     *gorm.DB
	aesKey []byte
}

func NewSSHKeyService(db *gorm.DB, aesKey string) *SSHKeyService {
	return &SSHKeyService{db: db, aesKey: []byte(aesKey)}
}

type SSHKeyResponse struct {
	ID        uint   `json:"id"`
	Alias     string `json:"alias"`
	CreatedAt string `json:"created_at"`
}

func (s *SSHKeyService) Create(userID, alias, privateKey string) (*SSHKeyResponse, error) {
	encrypted, err := crypto.Encrypt([]byte(privateKey), s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt key: %w", err)
	}

	key := models.SSHKey{
		UserID:       userID,
		Alias:        alias,
		EncryptedKey: base64.StdEncoding.EncodeToString(encrypted),
	}
	if err := s.db.Create(&key).Error; err != nil {
		return nil, fmt.Errorf("create ssh key: %w", err)
	}

	return &SSHKeyResponse{
		ID:        key.ID,
		Alias:     key.Alias,
		CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *SSHKeyService) List(userID string) ([]SSHKeyResponse, error) {
	var keys []models.SSHKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}

	result := make([]SSHKeyResponse, 0, len(keys))
	for _, k := range keys {
		result = append(result, SSHKeyResponse{
			ID:        k.ID,
			Alias:     k.Alias,
			CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return result, nil
}

func (s *SSHKeyService) GetDecryptedKey(userID string, keyID uint) (string, error) {
	var key models.SSHKey
	if err := s.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		return "", fmt.Errorf("ssh key not found: %w", err)
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(key.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("decode key: %w", err)
	}

	decrypted, err := crypto.Decrypt(encryptedBytes, s.aesKey)
	if err != nil {
		return "", fmt.Errorf("decrypt key: %w", err)
	}

	return string(decrypted), nil
}

func (s *SSHKeyService) Delete(userID string, keyID uint) error {
	// 检查是否有 Asset 引用此密钥
	var count int64
	s.db.Model(&models.Asset{}).Where("key_id = ?", keyID).Count(&count)
	if count > 0 {
		return fmt.Errorf("cannot delete: %d assets still reference this key", count)
	}

	result := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.SSHKey{})
	if result.Error != nil {
		return fmt.Errorf("delete ssh key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ssh key not found")
	}
	return nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd backend && go test ./internal/modules/intake/... -run TestSSHKey -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/intake/ssh_service.go backend/internal/modules/intake/ssh_service_test.go
git commit -m "feat: add SSH key service with encrypt/decrypt"
```

---

## Task 4: SSH 密钥 API Handler 与路由

**Files:**
- Create: `backend/internal/modules/intake/ssh_handler.go`
- Modify: `backend/internal/modules/intake/routes.go`

- [ ] **Step 1: 实现 SSH Handler**

```go
// backend/internal/modules/intake/ssh_handler.go
package intake

import (
	"strconv"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

const CodeSSHKeyNotFound = 1007 // SSH key not found or in use

type SSHHandler struct {
	sshSvc *SSHKeyService
}

func NewSSHHandler(sshSvc *SSHKeyService) *SSHHandler {
	return &SSHHandler{sshSvc: sshSvc}
}

type createSSHKeyReq struct {
	Alias      string `json:"alias" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
}

func (h *SSHHandler) CreateSSHKey(c *gin.Context) {
	var req createSSHKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "alias and private_key are required")
		return
	}

	key, err := h.sshSvc.Create(userID(c), req.Alias, req.PrivateKey)
	if err != nil {
		response.Error(c, CodeInternalError, "failed to create SSH key")
		return
	}

	response.Success(c, key)
}

func (h *SSHHandler) ListSSHKeys(c *gin.Context) {
	keys, err := h.sshSvc.List(userID(c))
	if err != nil {
		response.Error(c, CodeInternalError, "failed to list SSH keys")
		return
	}

	response.Success(c, keys)
}

func (h *SSHHandler) DeleteSSHKey(c *gin.Context) {
	keyID, err := strconv.ParseUint(c.Param("key_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid key_id")
		return
	}

	if err := h.sshSvc.Delete(userID(c), uint(keyID)); err != nil {
		response.Error(c, CodeSSHKeyNotFound, "SSH key not found or in use")
		return
	}

	response.Success(c, nil)
}
```

- [ ] **Step 2: 添加路由**

修改 `backend/internal/modules/intake/routes.go`：

```go
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, uploadDir string, aesKey string) {
	store := storage.NewLocalStorage(uploadDir)
	projectSvc := NewProjectService(db)
	assetSvc := NewAssetService(db, store)
	h := NewHandler(projectSvc, assetSvc)

	// SSH Key management
	sshSvc := NewSSHKeyService(db, aesKey)
	sshH := NewSSHHandler(sshSvc)

	// Project CRUD
	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects", h.ListProjects)
	rg.GET("/projects/:project_id", h.GetProject)
	rg.DELETE("/projects/:project_id", h.DeleteProject)

	// SSH Key management
	rg.POST("/ssh-keys", sshH.CreateSSHKey)
	rg.GET("/ssh-keys", sshH.ListSSHKeys)
	rg.DELETE("/ssh-keys/:key_id", sshH.DeleteSSHKey)

	// Asset management
	rg.POST("/assets/upload", h.UploadFile)
	rg.POST("/assets/folders", h.CreateFolder)
	rg.POST("/assets/git", h.CreateGitRepo)
	rg.GET("/assets", h.ListAssets)
	rg.GET("/assets/:asset_id/file", h.GetAssetFile)
	rg.DELETE("/assets/:asset_id", h.DeleteAsset)
	rg.PATCH("/assets/:asset_id", h.UpdateAsset)
	rg.PATCH("/assets/:asset_id/move", h.MoveAsset)

	// Notes
	rg.POST("/assets/notes", h.CreateNote)
	rg.PUT("/assets/notes/:note_id", h.UpdateNote)
}
```

同时修改 `backend/cmd/server/main.go` 中的调用：

```go
// 在 main.go 中添加本地辅助函数（database.envOrDefault 未导出，不可跨包调用）
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// 在 intake 路由注册处：
aesKey := os.Getenv("SSH_KEY_AES_KEY")
if aesKey == "" {
    log.Fatal("SSH_KEY_AES_KEY environment variable is required")
}
intake.RegisterRoutes(authed, db, uploadDir, aesKey)
```

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 4: Commit**

```bash
git add backend/internal/modules/intake/ssh_handler.go backend/internal/modules/intake/routes.go
git commit -m "feat: add SSH key management API (POST/GET/DELETE /ssh-keys)"
```

---

## Task 5: 修改 CreateGitRepo 支持 key_id

**Files:**
- Modify: `backend/internal/modules/intake/handler.go:43-47`
- Modify: `backend/internal/modules/intake/service.go:283-334`

- [ ] **Step 1: 修改请求结构体**

修改 `backend/internal/modules/intake/handler.go` 中的 `createGitReq`：

```go
type createGitReq struct {
	ProjectID uint     `json:"project_id"`
	RepoURL   string   `json:"repo_url"`
	RepoURLs  []string `json:"repo_urls"`
	KeyID     *uint    `json:"key_id"`
}
```

修改 `CreateGitRepo` handler 方法，将 `keyID` 传递给 service：

```go
func (h *Handler) CreateGitRepo(c *gin.Context) {
	var req createGitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "invalid request body")
		return
	}

	urls := req.RepoURLs
	if req.RepoURL != "" {
		urls = append(urls, req.RepoURL)
	}

	assets, err := h.assetSvc.CreateGitRepo(userID(c), req.ProjectID, urls, req.KeyID)
	// ... 错误处理不变
}
```

- [ ] **Step 2: 修改 service 层**

修改 `backend/internal/modules/intake/service.go` 中 `CreateGitRepo` 方法签名和实现：

```go
func (s *AssetService) CreateGitRepo(userID string, projectID uint, repoURLs []string, keyID *uint) ([]models.Asset, error) {
	if err := s.validateProject(userID, projectID); err != nil {
		return nil, err
	}

	// 验证 key_id 如果提供
	if keyID != nil {
		var key models.SSHKey
		if err := s.db.Where("id = ? AND user_id = ?", *keyID, userID).First(&key).Error; err != nil {
			return nil, fmt.Errorf("SSH key not found: %w", err)
		}
	}

	// ... 去重和校验逻辑不变 ...

	assets := make([]models.Asset, 0, len(cleaned))
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, repoURL := range cleaned {
			urlCopy := repoURL
			asset := models.Asset{
				ProjectID: projectID,
				Type:      "git_repo",
				URI:       &urlCopy,
				KeyID:     keyID,
				Status:    "pending",
			}
			if err := tx.Create(&asset).Error; err != nil {
				return fmt.Errorf("create git repo asset: %w", err)
			}
			assets = append(assets, asset)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return assets, nil
}
```

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 4: Commit**

```bash
git add backend/internal/modules/intake/handler.go backend/internal/modules/intake/service.go
git commit -m "feat: CreateGitRepo accepts key_id for SSH key association"
```

---

## Task 6: extract_git_repo 工具实现

**Files:**
- Create: `backend/internal/modules/agent/git_extractor_tool.go`
- Create: `backend/internal/modules/agent/git_extractor_tool_test.go`
- Modify: `backend/internal/modules/agent/tool_executor.go:127-208`

- [ ] **Step 1: 编写工具测试**

```go
// backend/internal/modules/agent/git_extractor_tool_test.go
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupToolTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Asset{}, &models.SSHKey{}, &models.Project{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestExtractGitRepoTool_AssetNotFound(t *testing.T) {
	db := setupToolTestDB(t)
	tool := NewExtractGitRepoTool(db, nil, nil, "", 50)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"repo_asset_id": float64(999),
		"user_context":  "test",
	})
	if err == nil {
		t.Fatal("expected error for non-existent asset")
	}
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
	if err == nil {
		t.Fatal("expected error for non-git-repo asset")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd backend && go test ./internal/modules/agent/... -run TestExtractGitRepo -v`
Expected: FAIL

- [ ] **Step 3: 实现 extract_git_repo 工具**

```go
// backend/internal/modules/agent/git_extractor_tool.go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type SSHKeyProvider interface {
	GetDecryptedKey(userID string, keyID uint) (string, error)
}

type ExtractGitRepoTool struct {
	db           *gorm.DB
	sshProvider  SSHKeyProvider
	aiProvider   AIProvider
	extractModel string
	sizeLimitMB  int
}

func NewExtractGitRepoTool(db *gorm.DB, sshProvider SSHKeyProvider, aiProvider AIProvider, extractModel string, sizeLimitMB int) *ExtractGitRepoTool {
	return &ExtractGitRepoTool{
		db:           db,
		sshProvider:  sshProvider,
		aiProvider:   aiProvider,
		extractModel: extractModel,
		sizeLimitMB:  sizeLimitMB,
	}
}

func (t *ExtractGitRepoTool) Name() string {
	return "extract_git_repo"
}

func (t *ExtractGitRepoTool) Description() string {
	return "提取 Git 仓库内容并生成简历导向的分析报告。需要提供仓库 Asset ID 和用户岗位上下文。"
}

func (t *ExtractGitRepoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repo_asset_id": map[string]interface{}{
				"type":        "integer",
				"description": "git_repo 类型的 Asset ID",
			},
			"user_context": map[string]interface{}{
				"type":        "string",
				"description": "用户的岗位目标和重点（从 interview 技能获取）",
			},
		},
		"required": []string{"repo_asset_id", "user_context"},
	}
}

func strPtr(s string) *string { return &s }

func (t *ExtractGitRepoTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	assetID, err := getIntParam(params, "repo_asset_id") // getIntParam 定义在 tool_executor.go（同包）
	if err != nil {
		return "", fmt.Errorf("repo_asset_id is required")
	}
	userContext, _ := params["user_context"].(string)

	// 1. 查询 Asset
	var asset models.Asset
	if err := t.db.WithContext(ctx).First(&asset, assetID).Error; err != nil {
		return "", fmt.Errorf("asset not found: %w", err)
	}
	if asset.Type != "git_repo" {
		return "", fmt.Errorf("asset type is %q, not git_repo", asset.Type)
	}
	if asset.URI == nil {
		return "", fmt.Errorf("asset has no URI")
	}

	// 2. 获取 SSH 密钥
	if asset.KeyID == nil {
		return "", fmt.Errorf("asset has no SSH key associated")
	}

	// 通过 Project 获取 user_id
	var project models.Project
	if err := t.db.WithContext(ctx).Select("user_id").First(&project, asset.ProjectID).Error; err != nil {
		return "", fmt.Errorf("project not found: %w", err)
	}

	// 3. 克隆仓库
	tempDir, cleanup, err := t.cloneRepo(ctx, *asset.URI, asset.KeyID, project.UserID)
	if err != nil {
		return "", fmt.Errorf("clone failed: %w", err)
	}
	defer cleanup()

	// 4. 检查大小
	sizeMB, err := dirSizeMB(tempDir)
	if err != nil {
		return "", fmt.Errorf("check size: %w", err)
	}
	if sizeMB > t.sizeLimitMB {
		return "", fmt.Errorf("仓库大小 %.1fMB 超过限制 %dMB", sizeMB, t.sizeLimitMB)
	}

	// 5. 探索仓库
	report, err := t.exploreRepo(ctx, tempDir, userContext)
	if err != nil {
		return "", fmt.Errorf("explore failed: %w", err)
	}

	// 6. 存储报告
	analysisAsset := models.Asset{
		ProjectID: asset.ProjectID,
		Type:      "git_analysis",
		Content:   &report,
		Label:     strPtr(fmt.Sprintf("Git Analysis: %s", *asset.URI)),
	}
	if err := t.db.WithContext(ctx).Create(&analysisAsset).Error; err != nil {
		return "", fmt.Errorf("save analysis: %w", err)
	}

	// 7. 更新原 Asset 状态
	t.db.WithContext(ctx).Model(&asset).Update("status", "analyzed")

	result := map[string]interface{}{
		"analysis_asset_id": analysisAsset.ID,
		"repo_url":          *asset.URI,
		"size_mb":           sizeMB,
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd backend && go test ./internal/modules/agent/... -run TestExtractGitRepo -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/git_extractor_tool.go backend/internal/modules/agent/git_extractor_tool_test.go
git commit -m "feat: add extract_git_repo tool for lazy git extraction"
```

---

## Task 7: SSH 克隆与 subagent 探索逻辑

**Files:**
- Modify: `backend/internal/modules/agent/git_extractor_tool.go`
- Create: `backend/internal/modules/agent/ai_provider.go`

- [ ] **Step 1: 实现 AIProvider**

```go
// backend/internal/modules/agent/ai_provider.go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AIProvider defines the interface for AI model calls.
type AIProvider interface {
	Call(ctx context.Context, model, systemPrompt, userMessage string) (string, error)
}

// OpenAIProvider implements AIProvider using OpenAI-compatible API.
type OpenAIProvider struct {
	apiURL string
	apiKey string
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		apiURL: strings.TrimSpace(os.Getenv("AI_API_URL")),
		apiKey: strings.TrimSpace(os.Getenv("AI_API_KEY")),
	}
}

func (p *OpenAIProvider) Call(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	if p.apiURL == "" || p.apiKey == "" {
		return "", fmt.Errorf("AI_API_URL or AI_API_KEY not configured")
	}

	endpoint := normalizeChatURL(p.apiURL)

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.3,
		"stream":      false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("response has no choices")
	}

	return result.Choices[0].Message.Content, nil
}

func normalizeChatURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/chat/completions") {
		return raw
	}
	if strings.HasSuffix(raw, "/v1") || strings.HasSuffix(raw, "/api/paas/v4") {
		return raw + "/chat/completions"
	}
	return raw + "/v1/chat/completions"
}
```

- [ ] **Step 2: 实现 cloneRepo 方法**

在 `git_extractor_tool.go` 中添加：

```go
func (t *ExtractGitRepoTool) cloneRepo(ctx context.Context, repoURL string, keyID *uint, userID string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "resume-genius-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() { os.RemoveAll(tempDir) }

	sshKeyPath := ""

	// 获取并写入 SSH 密钥
	if t.sshProvider != nil && keyID != nil {
		keyContent, err := t.sshProvider.GetDecryptedKey(userID, *keyID)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("get SSH key: %w", err)
		}

		keyPath := filepath.Join(tempDir, "id_rsa")
		if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("write SSH key: %w", err)
		}

		sshKeyPath = keyPath
	}

	repoDir := filepath.Join(tempDir, "repo")
	cloneCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cloneCtx, "git", "clone", "--depth", "1", repoURL, repoDir)
	cmd.Env = os.Environ()
	if sshKeyPath != "" {
		// 安全说明：StrictHostKeyChecking=no 仅适用于服务器端自动化场景，
		// 存在 MITM 攻击风险。生产环境应确保网络环境可信或使用 known_hosts 预置。
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", sshKeyPath)
		cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+sshCmd)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		cleanup()
		if cloneCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("git clone timed out after 60s")
		}
		return "", nil, fmt.Errorf("git clone failed: %s", stderr.String())
	}

	return repoDir, cleanup, nil
}
```

- [ ] **Step 2: 实现 dirSizeMB 辅助函数**

```go
func dirSizeMB(path string) (float64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return float64(size) / (1024 * 1024), err
}
```

- [ ] **Step 3: 实现 exploreRepo 方法（subagent 调用）**

```go
func (t *ExtractGitRepoTool) exploreRepo(ctx context.Context, repoDir, userContext string) (string, error) {
	// 收集仓库上下文
	var contextBuilder strings.Builder

	// 文件树
	contextBuilder.WriteString("## 仓库文件树\n")
	filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(repoDir, path)
		if strings.HasPrefix(relPath, ".git") || strings.HasPrefix(relPath, "node_modules") {
			return nil
		}
		contextBuilder.WriteString(relPath + "\n")
		return nil
	})

	// 关键文件内容
	contextBuilder.WriteString("\n## 关键文件\n")
	readKeyFiles(repoDir, &contextBuilder)

	// 用户上下文
	if userContext != "" {
		contextBuilder.WriteString("\n## 用户岗位重点\n")
		contextBuilder.WriteString(userContext)
	}

	// 调用 AI 生成报告
	systemPrompt := buildGitExploreSystemPrompt()
	userMessage := contextBuilder.String()

	report, err := t.aiProvider.Call(ctx, t.extractModel, systemPrompt, userMessage)
	if err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}

	return report, nil
}
```

- [ ] **Step 4: 实现 readKeyFiles 和 callAI 辅助函数**

```go
func readKeyFiles(repoDir string, sb *strings.Builder) {
	maxFiles := 10
	maxBytesPerFile := 5000
	count := 0

	filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || count >= maxFiles {
			return nil
		}
		relPath, _ := filepath.Rel(repoDir, path)
		if strings.HasPrefix(relPath, ".git") {
			return nil
		}

		name := info.Name()
		isKey := name == "README.md" || name == "package.json" || name == "go.mod" ||
			name == "Cargo.toml" || name == "requirements.txt" || name == "Dockerfile" ||
			name == "Makefile" || name == "main.go" || name == "index.ts" || name == "app.py"
		if !isKey {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if len(data) > maxBytesPerFile {
			data = data[:maxBytesPerFile]
		}

		sb.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", relPath, string(data)))
		count++
		return nil
	})
}

func buildGitExploreSystemPrompt() string {
	return `你是一位技术简历顾问和代码库分析专家。分析提供的仓库上下文，生成简历导向的分析报告。

必须包含：
1. 项目概述（解决什么问题、目标用户、规模）
2. 技术栈全景（语言、框架、数据库、基础设施、前端、AI/ML）
3. 架构要点（3-5 个面试官感兴趣的技术话题）
4. 技术亮点（3-5 个可写进简历的 bullet point，含技术细节）

禁止编造信息。输出中文。`
}
```

- [ ] **Step 5: 添加 exploreRepo 测试（mock AIProvider）**

在 `git_extractor_tool_test.go` 中添加：

```go
type mockAIProvider struct {
	response string
	err      error
}

func (m *mockAIProvider) Call(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	return m.response, m.err
}

type mockSSHProvider struct {
	keys map[uint]string
}

func (m *mockSSHProvider) GetDecryptedKey(userID string, keyID uint) (string, error) {
	if k, ok := m.keys[keyID]; ok {
		return k, nil
	}
	return "", fmt.Errorf("key not found")
}

func TestExtractGitRepoTool_ExploreRepo(t *testing.T) {
	db := setupToolTestDB(t)
	mockAI := &mockAIProvider{response: "# Test Analysis\n\n## 项目概述\nMock analysis result"}
	tool := NewExtractGitRepoTool(db, nil, mockAI, "haiku", 50)

	// 创建一个临时目录模拟仓库
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Project"), 0644)

	report, err := tool.exploreRepo(context.Background(), tmpDir, "目标岗位：测试工程师")
	if err != nil {
		t.Fatalf("exploreRepo failed: %v", err)
	}
	if !strings.Contains(report, "Mock analysis result") {
		t.Fatalf("report = %q, expected mock response", report)
	}
}
```

- [ ] **Step 6: 运行测试**

Run: `cd backend && go test ./internal/modules/agent/... -run TestExtractGitRepo -v`
Expected: PASS

- [ ] **Step 7: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 8: Commit**

```bash
git add backend/internal/modules/agent/git_extractor_tool.go backend/internal/modules/agent/git_extractor_tool_test.go backend/internal/modules/agent/ai_provider.go
git commit -m "feat: implement SSH clone and subagent exploration logic"
```

---

## Task 8: 注册工具到 AgentToolExecutor

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:127-208`
- Modify: `backend/internal/modules/agent/service.go:118-119`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 修改 AgentToolExecutor 构造函数**

修改 `tool_executor.go` 中的 `AgentToolExecutor` 结构体：

```go
type AgentToolExecutor struct {
	db               *gorm.DB
	skillLoader      *SkillLoader
	extractGitTool   *ExtractGitRepoTool
	getDraftCallCount sync.Map
	cachedTools      []ToolDef
}
```

修改构造函数：

```go
func NewAgentToolExecutor(db *gorm.DB, skillLoader *SkillLoader, extractGitTool *ExtractGitRepoTool) *AgentToolExecutor {
	e := &AgentToolExecutor{
		db:             db,
		skillLoader:    skillLoader,
		extractGitTool: extractGitTool,
	}
	e.cachedTools = e.buildTools()
	return e
}
```

- [ ] **Step 2: 在 buildTools 中注册新工具**

在 `buildTools()` 方法末尾、`load_skill` 之前添加：

```go
if e.extractGitTool != nil {
	tools = append(tools, ToolDef{
		Name:        e.extractGitTool.Name(),
		Description: e.extractGitTool.Description(),
		Parameters:  e.extractGitTool.Parameters(),
	})
}
```

- [ ] **Step 3: 在 Execute 中添加分发**

在 `Execute` 方法的 switch 中添加：

```go
case "extract_git_repo":
	if e.extractGitTool != nil {
		result, err = e.extractGitTool.Execute(ctx, params)
	} else {
		return "", fmt.Errorf("extract_git_repo tool not available")
	}
```

- [ ] **Step 4: 修改 main.go 初始化**

修改 `backend/cmd/server/main.go`：

```go
// 在 intake 模块初始化后
sshSvc := intake.NewSSHKeyService(db, os.Getenv("SSH_KEY_AES_KEY"))
extractModel := envOrDefault("GIT_EXTRACT_MODEL", "haiku")
sizeLimitMB := 50 // 从 GIT_REPO_SIZE_LIMIT_MB 读取
aiProvider := agent.NewOpenAIProvider()
extractTool := agent.NewExtractGitRepoTool(db, sshSvc, aiProvider, extractModel, sizeLimitMB)

// agent 模块初始化时传入
toolExecutor := agent.NewAgentToolExecutor(db, skillLoader, extractTool)
```

- [ ] **Step 5: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/cmd/server/main.go
git commit -m "feat: register extract_git_repo tool in AgentToolExecutor"
```

---

## Task 9: System Prompt 延迟提取引导

**Files:**
- Modify: `backend/internal/modules/agent/prompt.go:93-97`

- [ ] **Step 1: 修改 flowRulesSection**

```go
const flowRulesSection = `## 循环控制规则
- get_draft 最多调用 2 次（structure + full），之后必须直接用 apply_edits 编辑
- 重复读取不会获得新信息，只会浪费步骤
- apply_edits 失败时，用更短的唯一片段重试，不要重新读取整个简历
- 如果步骤即将耗尽，优先输出当前最佳结果，不要继续搜索

## Git 仓库延迟提取
- 当用户提到目标岗位且项目中有 status="pending" 的 git_repo 资产时，应主动触发 git 分析流程
- 流程：先 load_skill("resume-interview") 获取岗位重点，再调用 extract_git_repo 工具
- extract_git_repo 的 user_context 参数应包含从 interview 技能获取的岗位关注点
- 分析完成后通过 search_assets 搜索 git_analysis 类型的报告来参考`
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/agent/prompt.go
git commit -m "feat: add lazy git extraction guidance to system prompt"
```

---

## Task 10: 前端 SSH 密钥选择组件

**Files:**
- Create: `frontend/workbench/src/components/intake/SSHKeySelector.tsx`
- Modify: `frontend/workbench/src/components/intake/GitRepoDialog.tsx`
- Modify: `frontend/workbench/src/lib/api-client.ts`

- [ ] **Step 1: 添加 API 调用并更新 Asset 接口**

修改 `frontend/workbench/src/lib/api-client.ts`：

更新 Asset 接口（如存在）：

```typescript
interface Asset {
  id: number
  project_id: number
  type: string
  uri?: string
  content?: string
  label?: string
  key_id?: number
  status: string
  created_at: string
  updated_at: string
}
```

在 `createGitRepo` 之前添加 SSH Key API：

```typescript
// SSH Key management
listSSHKeys: () =>
  request<Array<{ id: number; alias: string; created_at: string }>>('/ssh-keys'),

createSSHKey: (alias: string, privateKey: string) =>
  request<{ id: number; alias: string; created_at: string }>('/ssh-keys', {
    method: 'POST',
    body: JSON.stringify({ alias, private_key: privateKey }),
  }),

deleteSSHKey: (id: number) =>
  request<null>(`/ssh-keys/${id}`, { method: 'DELETE' }),
```

修改 `createGitRepo` 签名：

```typescript
createGitRepo: (projectId: number, repoUrls: string[], keyId?: number) =>
  request<Asset[]>('/assets/git', {
    method: 'POST',
    body: JSON.stringify({ project_id: projectId, repo_urls: repoUrls, key_id: keyId }),
  }),
```

- [ ] **Step 2: 创建 SSHKeySelector 组件**

```tsx
// frontend/workbench/src/components/intake/SSHKeySelector.tsx
import { useState, useEffect } from 'react'
import { api } from '@/lib/api-client'

interface SSHKey {
  id: number
  alias: string
  created_at: string
}

interface SSHKeySelectorProps {
  value: number | null
  onChange: (keyId: number | null) => void
}

export default function SSHKeySelector({ value, onChange }: SSHKeySelectorProps) {
  const [keys, setKeys] = useState<SSHKey[]>([])
  const [showNew, setShowNew] = useState(false)
  const [alias, setAlias] = useState('')
  const [privateKey, setPrivateKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.listSSHKeys().then(setKeys).catch(() => {})
  }, [])

  const handleCreate = async () => {
    if (!alias.trim() || !privateKey.trim()) return
    try {
      setLoading(true)
      setError('')
      const newKey = await api.createSSHKey(alias.trim(), privateKey.trim())
      setKeys(prev => [newKey, ...prev])
      onChange(newKey.id)
      setShowNew(false)
      setAlias('')
      setPrivateKey('')
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">SSH 密钥</label>

      {!showNew ? (
        <div className="flex gap-2">
          <select
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
            value={value ?? ''}
            onChange={(e) => onChange(e.target.value ? Number(e.target.value) : null)}
          >
            <option value="">选择已有密钥...</option>
            {keys.map(k => (
              <option key={k.id} value={k.id}>{k.alias}</option>
            ))}
          </select>
          <button
            type="button"
            className="text-sm text-primary hover:underline whitespace-nowrap"
            onClick={() => setShowNew(true)}
          >
            上传新密钥
          </button>
        </div>
      ) : (
        <div className="space-y-2 border rounded-md p-3">
          <input
            type="text"
            placeholder="密钥别名（如：github-deploy-key）"
            value={alias}
            onChange={(e) => setAlias(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          />
          <textarea
            placeholder="粘贴 SSH 私钥内容（-----BEGIN OPENSSH PRIVATE KEY----- ...）"
            value={privateKey}
            onChange={(e) => setPrivateKey(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm min-h-[100px] font-mono text-xs"
          />
          <p className="text-xs text-amber-600">
            安全提示：请使用专用只读密钥，切勿使用生产环境密钥或有写权限的密钥。
          </p>
          <div className="flex gap-2">
            <button
              type="button"
              className="text-sm text-muted-foreground hover:underline"
              onClick={() => { setShowNew(false); setAlias(''); setPrivateKey(''); }}
            >
              取消
            </button>
            <button
              type="button"
              className="text-sm bg-primary text-primary-foreground px-3 py-1 rounded-md disabled:opacity-50"
              onClick={handleCreate}
              disabled={!alias.trim() || !privateKey.trim() || loading}
            >
              {loading ? '保存中...' : '保存密钥'}
            </button>
          </div>
          {error && <p className="text-xs text-destructive">{error}</p>}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: 修改 GitRepoDialog 集成 SSHKeySelector**

```tsx
// frontend/workbench/src/components/intake/GitRepoDialog.tsx
import { useState } from 'react'
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import SSHKeySelector from './SSHKeySelector'

interface GitRepoDialogProps {
  open: boolean
  onClose: () => void
  onSubmit: (repoUrls: string[], keyId?: number) => Promise<void>
}

export default function GitRepoDialog({ open, onClose, onSubmit }: GitRepoDialogProps) {
  const [repoUrlsText, setRepoUrlsText] = useState('')
  const [selectedKeyId, setSelectedKeyId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    const urls = repoUrlsText
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line !== '')

    if (urls.length === 0) {
      setError('请输入至少一个 Git 仓库地址')
      return
    }
    if (!selectedKeyId) {
      setError('请选择或上传 SSH 密钥')
      return
    }
    try {
      setSubmitting(true)
      setError('')
      await onSubmit(urls, selectedKeyId)
      setRepoUrlsText('')
      setSelectedKeyId(null)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleClose = () => {
    setRepoUrlsText('')
    setSelectedKeyId(null)
    setError('')
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>接入 Git 仓库</ModalHeader>
      <p className="text-xs text-muted-foreground mt-1">
        输入 Git 仓库地址（SSH 格式），每行一个
      </p>

      <textarea
        value={repoUrlsText}
        onChange={(e) => setRepoUrlsText(e.target.value)}
        placeholder={'git@github.com:user/repo1.git\ngit@github.com:user/repo2.git'}
        className="mt-4 w-full rounded-md border border-input bg-background px-3 py-2 text-sm min-h-[80px] resize-y"
        rows={4}
      />

      <div className="mt-3">
        <SSHKeySelector value={selectedKeyId} onChange={setSelectedKeyId} />
      </div>

      {error && (
        <p className="text-xs text-destructive mt-2">{error}</p>
      )}

      <ModalFooter>
        <Button variant="secondary" onClick={handleClose}>
          取消
        </Button>
        <Button onClick={handleSubmit} disabled={!repoUrlsText.trim() || !selectedKeyId || submitting}>
          {submitting ? '接入中...' : '接入'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
```

- [ ] **Step 4: 更新调用方适配新签名**

修改 `AssetWorkspace.tsx` 中调用 `GitRepoDialog` 的地方，适配新的 `onSubmit` 签名 `(repoUrls: string[], keyId?: number) => Promise<void>`。

- [ ] **Step 5: 前端编译验证**

Run: `cd frontend/workbench && bun run build`
Expected: 编译通过

- [ ] **Step 6: Commit**

```bash
git add frontend/workbench/src/components/intake/ frontend/workbench/src/lib/api-client.ts
git commit -m "feat: add SSH key selector to Git repo upload dialog"
```

---

## Task 11: 环境变量配置与文档

**Files:**
- Modify: `backend/internal/shared/database/database.go` (envOrDefault 已存在)
- Modify: `.env.example` 或相关配置文档

- [ ] **Step 1: 确保环境变量可配置**

在 `backend/cmd/server/main.go` 中添加环境变量读取：

```go
extractModel := envOrDefault("GIT_EXTRACT_MODEL", "haiku")
sizeLimitStr := envOrDefault("GIT_REPO_SIZE_LIMIT_MB", "50")
sizeLimitMB, _ := strconv.Atoi(sizeLimitStr)
if sizeLimitMB <= 0 {
    sizeLimitMB = 50
}
```

- [ ] **Step 2: 更新 CLAUDE.md 环境变量表**

在 `CLAUDE.md` 的环境变量表中添加：

```markdown
| `SSH_KEY_AES_KEY` | — | SSH 私钥加密密钥（必填，缺失时启动失败） |
| `GIT_EXTRACT_MODEL` | haiku | Git 探索 subagent 模型 |
| `GIT_REPO_SIZE_LIMIT_MB` | 50 | Git 仓库大小限制（MB） |
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md backend/cmd/server/main.go
git commit -m "docs: add env vars for git extraction and SSH key encryption"
```

---

## Task 12: 清理旧的 git 提取逻辑

**Files:**
- Review: `backend/internal/modules/parsing/git_extractor.go`
- Review: `backend/internal/modules/parsing/git_ai_extractor.go`
- Review: `backend/internal/modules/intake/service.go`

- [ ] **Step 1: 确认旧逻辑无其他调用方**

Run: `cd backend && grep -rn "GitExtractor\|AIGitExtractor\|GitRepositoryExtractor" --include="*.go" | grep -v "_test.go"`
Expected: 仅在 parsing 包内部和可能的 parsing routes 中出现

- [ ] **Step 2: 移除 intake 中对 parsing 的 git 提取调用**

如果 `intake/service.go` 或 `intake/handler.go` 中有调用 parsing 模块进行 git 提取的代码，移除这些调用。`CreateGitRepo` 应仅入库，不再触发提取。

- [ ] **Step 3: 标记 parsing 模块的 git 提取为废弃**

在 `git_extractor.go` 和 `git_ai_extractor.go` 文件顶部添加注释：

```go
// DEPRECATED: Git extraction logic has moved to agent/git_extractor_tool.go.
// These files are retained for reference but should not be used in new code.
```

保留文件不删除，因为 `extract_git_repo` 工具中的 `readKeyFiles` 和 `buildGitExploreSystemPrompt` 参考了这些文件的逻辑。

- [ ] **Step 4: 编译验证**

Run: `cd backend && go build ./...`
Expected: 编译通过

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/parsing/ backend/internal/modules/intake/
git commit -m "refactor: deprecate old git extraction, moved to agent tool"
```

---

## 最终验证

- [ ] **全量编译**

```bash
cd backend && go build ./...
cd frontend/workbench && bun run build
```

- [ ] **运行后端测试**

```bash
cd backend && go test ./...
```

- [ ] **端到端手动测试**

1. 启动后端和前端
2. 创建项目
3. 上传 SSH 密钥（POST /ssh-keys）
4. 提交 git 仓库链接（POST /assets/git with key_id）
5. 在 AI 对话中表达目标岗位
6. 验证 AI 自动触发 extract_git_repo
7. 验证分析报告存为新 Asset
