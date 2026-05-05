# PR #28 修复与存储模型重构设计

日期: 2026-05-05

## 背景

PR #28 实现了素材正文持久化和编辑页素材管理。review 发现了 12 个问题，同时在讨论中确定了存储模型的重构方向：内容寻址存储 + 软删除 + 哈希去重。

## 设计原则

- 文件按内容哈希存储，同一用户只存一份
- 删除全部改为软删除，数据可恢复
- 同用户再次上传相同文件时，恢复旧资产记录，复用文件和解析结果
- 先改 DB 再处理文件，保证数据一致性

## 改动清单

### 1. 所有模型加软删除

给以下模型加 `DeletedAt gorm.DeletedAt` 字段：

- `Project`
- `Asset`
- `Draft`
- `Version`
- `AISession`
- `AIMessage`
- `AIToolCall`

GORM 的 `DeletedAt` 会自动让 `.Delete()` 变成软删除（设置 `deleted_at`），`.Find()` 等查询自动过滤软删除记录。需要查询包含软删除记录时用 `.Unscoped()`。

### 2. 文件存储改为内容寻址

**storage.go 改造：**

- `Save` 方法签名改为 `Save(userID uint, fileHash string, ext string, data []byte) (string, error)`
- 存储路径从 `{projectID}/{uuid}_{filename}` 改为 `{userID}/{fileHash}.{ext}`
- `ext` 用 `filepath.Base` + `filepath.Ext` 清理，只保留扩展名部分（如 `.pdf`）
- `fileHash` 是 SHA-256 十六进制字符串，不存在路径遍历风险
- 如果文件已存在（相同路径），直接返回 key，不重复写入

**intake/service.go 上传流程改造：**

```
上传文件 → 算 SHA-256 → 查同用户软删除资产(file_hash匹配)
  ├── 命中 → UNDELETE（清空 deleted_at，更新 project_id/label 等）→ 返回
  └── 未命中 → 正常存文件 + 创建资产记录 + 解析
```

### 3. Asset 模型加 file_hash 字段

`models.Asset` 新增 `FileHash *string` 字段（`gorm:"size:64;index"`），存储文件的 SHA-256 哈希。

- 文件类资产（resume_pdf、resume_docx）：上传时计算并存储
- 图片资产（resume_image）：从原始文件提取时计算并存储
- note/git_repo 类型：不需要 file_hash，为 nil

用于同用户去重查询（需 JOIN projects 获取 user_id）：
```go
s.db.Unscoped().
  Joins("JOIN projects ON projects.id = assets.project_id").
  Where("projects.user_id = ? AND assets.file_hash = ? AND assets.deleted_at IS NOT NULL", userID, fileHash).
  First(&asset)
```

### 4. 删除逻辑适配软删除

**intake/service.go：**

- `DeleteAsset`：改为 `s.db.Delete(&models.Asset{}, ...)` 软删除。主素材和派生图片都是软删除，不删物理文件。
- `DeleteProjectAssets`：同理，批量软删除。
- `ProjectService.Delete`：软删除项目，级联软删除关联资产。
- 图片物理文件的清理后续可以做定期 GC（扫描无引用的文件），不在本次范围内。

**parsing/service.go：**

- `deleteOriginalSourceAfterPersistence`：先更新 DB 标记 `source_deleted`，再删物理文件。软删除不涉及此处，但顺序修正仍然有意义。
- `loadDerivedImageAssetsForCleanup`：返回 error 而不是吞掉。

### 5. 前端错误处理

**AssetSidebar.tsx：**

`handleUpload`、`handleCreateNote`、`handleUpdateAsset` 加 try/catch：

```typescript
const handleUpload = async (file: File, replaceAssetId?: number) => {
  setError('')
  try {
    await intakeApi.uploadFile(projectId, file, replaceAssetId)
    await parsingApi.parseProject(projectId)
  } catch (err) {
    setError(err instanceof Error ? err.message : '上传或解析失败')
  } finally {
    await refreshAssets()
  }
}
```

`handleCreateNote`、`handleUpdateAsset` 同理。

**ProjectDetail.tsx：**

`handleUpload`、`handleCreateGit`、`handleCreateNote`、`handleUpdateNote`、`handleDeleteAsset` 同样加 try/catch。

### 6. EditorPage 闭包过期修复

用 `useRef` 持有 `draftId`，避免 `useAutoSave` 的 save 回调闭包捕获过期值。

### 7. 错误码修正

`intake/handler.go`：
- `CodeInternalError`：`50001` → `1999`
- `CodeParamInvalid`：`10001` → `1001`

### 8. validateProject 错误处理

`intake/service.go` 的 `validateProject` 和 `ProjectService.Delete`：
检查 `Count().Error`，DB 错误不再返回 `ErrProjectNotFound`。

### 9. PDF 图片静默丢弃加日志

`parsing/pdf_parser.go`：`extractPageImages` 中 `!ok` 的分支加 `log.Printf` 记录跳过的图片信息。

### 10. 补测试

- AssetSidebar 组件测试
- api-client 中 `updateAsset` 和 `parsingApi` 测试
- 软删除去重流程测试（上传相同文件 → 命中软删除 → 恢复）
- file_hash 存储和查询测试

## 不改动的部分

- AI 对话模块（agent）
- 营销站
- Docker/网关配置（PR #27 范围）
- 前端路由和页面结构

## 迁移注意

- `AutoMigrate` 会自动加 `deleted_at` 列，默认为 NULL，不影响现有数据
- `file_hash` 列新增后，现有资产该字段为 NULL，不影响已有功能
- 存储路径变更后，新上传的文件按新路径存，现有文件的旧路径 URI 仍然有效
