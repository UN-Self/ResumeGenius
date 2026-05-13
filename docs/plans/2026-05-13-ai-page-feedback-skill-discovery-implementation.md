# AI 页数反馈与 Skill 发现修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 AI 能感知简历页数，修复 Skill 发现断裂，增强 A4 设计参考

**Architecture:** 前端 TipTap 渲染后回传页数到后端 → get_draft 返回 page_count → AI 自行判断是否压缩。SkillLoader 传入 ChatService 构建技能清单注入系统提示词。新增 a4-template.yaml 参考。

**Tech Stack:** Go/Gin/GORM（后端）、React/TipTap（前端）、YAML（Skill 配置）

---

## 文件结构

| 操作 | 文件 | 职责 |
|------|------|------|
| 修改 | `backend/internal/shared/models/models.go:69-80` | Draft 增加 PageCount 字段 |
| 修改 | `backend/internal/modules/agent/skill_loader.go` | 新增 BuildSkillListing() 方法 |
| 修改 | `backend/internal/modules/agent/service.go:108-130` | ChatService 增加 skillLoader 字段 |
| 修改 | `backend/internal/modules/agent/routes.go:57-64` | 传 skillLoader 给 NewChatService |
| 修改 | `backend/internal/modules/agent/tool_executor.go:274-356` | get_draft 返回 page_count |
| 修改 | `backend/internal/modules/agent/service.go:238-276` | 构建 skillListing 注入系统提示词 |
| 新增 | `backend/internal/modules/agent/skills/resume-design/references/a4-template.yaml` | A4 精确尺寸与模板参考 |
| 修改 | `backend/internal/modules/agent/skills/resume-design/skill.yaml` | 注册新 reference |
| 新增 | `backend/internal/modules/workbench/handler.go` 或修改路由 | PATCH /drafts/:id/meta 端点 |
| 修改 | `frontend/workbench/src/pages/EditorPage.tsx` | TipTap onUpdate 回传页数 |
| 修改 | `backend/internal/modules/agent/tool_executor_test.go` | get_draft 测试覆盖 page_count |
| 修改 | `backend/internal/modules/agent/service_test.go` | ChatService 构造函数测试 |

---

### Task 1: Draft 模型增加 PageCount 字段

**Files:**
- Modify: `backend/internal/shared/models/models.go:69-80`
- Test: `backend/internal/shared/models/models_test.go`

- [ ] **Step 1: 在 Draft 结构体增加 PageCount 字段**

在 `models.go` 的 Draft 结构体中，在 `CurrentEditSequence` 之前添加 `PageCount` 字段：

```go
type Draft struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ProjectID   uint           `gorm:"not null;index" json:"project_id"`
	HTMLContent string         `gorm:"type:text;not null" json:"html_content"`
	Project     Project        `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Versions    []Version      `gorm:"foreignKey:DraftID" json:"versions,omitempty"`
	AISessions  []AISession    `gorm:"foreignKey:DraftID" json:"ai_sessions,omitempty"`
	PageCount             int            `gorm:"default:0" json:"page_count"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	CurrentEditSequence   int            `gorm:"default:0" json:"-"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}
```

- [ ] **Step 2: 更新测试中的 testutil AutoMigrate**

在 `backend/internal/modules/agent/testutil.go:54` 的 AutoMigrate 调用无需改动（已包含 `&models.Draft{}`），GORM AutoMigrate 会自动添加新字段。

- [ ] **Step 3: 运行现有测试验证不破坏**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/shared/models/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/shared/models/models.go
git commit -m "feat: Draft 模型增加 PageCount 字段"
```

---

### Task 2: SkillLoader 新增 BuildSkillListing 方法

**Files:**
- Modify: `backend/internal/modules/agent/skill_loader.go`
- Test: `backend/internal/modules/agent/skill_loader_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/internal/modules/agent/skill_loader_test.go` 中添加：

```go
package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSkillListing(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	listing := loader.BuildSkillListing()
	assert.NotEmpty(t, listing, "listing should not be empty when skills exist")
	assert.Contains(t, listing, "resume-design", "listing should mention resume-design skill")
	assert.Contains(t, listing, "load_skill", "listing should tell AI how to invoke")
}

func TestBuildSkillListing_Empty(t *testing.T) {
	loader := &SkillLoader{
		skills:     map[string]*SkillDescriptor{},
		references: map[string]map[string]*ReferenceContent{},
	}
	listing := loader.BuildSkillListing()
	assert.Empty(t, listing, "listing should be empty when no skills loaded")
}

func TestBuildSkillListing_ContainsTrigger(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	listing := loader.BuildSkillListing()
	assert.Contains(t, listing, "调整样式", "listing should include trigger info from skill.yaml")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestBuildSkillListing -v`
Expected: FAIL — `loader.BuildSkillListing undefined`

- [ ] **Step 3: 实现 BuildSkillListing**

在 `skill_loader.go` 末尾添加：

```go
// BuildSkillListing generates a skill listing string for the system prompt.
// Returns empty string if no skills are loaded.
func (l *SkillLoader) BuildSkillListing() string {
	if len(l.skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 可用技能\n")
	for _, desc := range l.skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", desc.Name, strings.TrimSpace(desc.Description)))
		if desc.Trigger != "" {
			sb.WriteString(fmt.Sprintf("  触发条件：%s\n", desc.Trigger))
		}
		sb.WriteString(fmt.Sprintf("  调用 load_skill(skill_name=\"%s\") 获取完整规范和参考。\n", desc.Name))
	}
	return sb.String()
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestBuildSkillListing -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/skill_loader.go backend/internal/modules/agent/skill_loader_test.go
git commit -m "feat: SkillLoader 新增 BuildSkillListing 方法"
```

---

### Task 3: ChatService 接收 SkillLoader 并构建技能清单

**Files:**
- Modify: `backend/internal/modules/agent/service.go:108-130` (ChatService 结构体和构造函数)
- Modify: `backend/internal/modules/agent/service.go:238-276` (系统提示词构建)
- Modify: `backend/internal/modules/agent/routes.go:57-64` (传 skillLoader)
- Test: `backend/internal/modules/agent/service_test.go`

- [ ] **Step 1: 写失败测试**

在 `service_test.go` 中添加：

```go
func TestNewChatService_WithSkillLoader(t *testing.T) {
	db := SetupTestDB(t)
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	mockProvider := &MockAdapter{}
	executor := NewAgentToolExecutor(db, loader)

	svc := NewChatService(db, mockProvider, executor, 3, loader)

	assert.NotNil(t, svc)
	assert.Equal(t, loader, svc.skillLoader)
}

func TestNewChatService_WithoutSkillLoader(t *testing.T) {
	db := SetupTestDB(t)
	mockProvider := &MockAdapter{}
	executor := NewAgentToolExecutor(db, nil)

	svc := NewChatService(db, mockProvider, executor, 3, nil)

	assert.NotNil(t, svc)
	assert.Nil(t, svc.skillLoader)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestNewChatService_With" -v`
Expected: FAIL — `too many arguments` 或 `skillLoader undefined`

- [ ] **Step 3: 修改 ChatService 结构体和构造函数**

在 `service.go` 中：

ChatService 结构体增加字段（约第 108-115 行）：

```go
type ChatService struct {
	db                *gorm.DB
	provider          ProviderAdapter
	toolExecutor      ToolExecutor
	recorder          *ThinkingRecorder
	maxIterations     int
	contextWindowSize int
	skillLoader       *SkillLoader
}
```

构造函数增加参数（约第 117-130 行）：

```go
func NewChatService(db *gorm.DB, provider ProviderAdapter, toolExecutor ToolExecutor, maxIterations int, skillLoader *SkillLoader) *ChatService {
	if maxIterations <= 0 {
		maxIterations = 3
	}
	windowSize := 128000
	return &ChatService{
		db:                db,
		provider:          provider,
		toolExecutor:      toolExecutor,
		maxIterations:     maxIterations,
		contextWindowSize: windowSize,
		skillLoader:       skillLoader,
	}
}
```

- [ ] **Step 4: 修改系统提示词构建**

在 `service.go` 的 `StreamChatReAct` 中（约第 275 行），将 `skillListing` 从 `""` 改为从 skillLoader 构建：

```go
// 5b. Pre-load project assets into system prompt so AI cannot ignore them
assetInfo := ""
if session.ProjectID != nil {
	assetInfo = s.preloadAssets(*session.ProjectID)
}

skillListing := ""
if s.skillLoader != nil {
	skillListing = s.skillLoader.BuildSkillListing()
}
sections = DefaultPromptSections(assetInfo, skillListing)
```

同样修改第 238 行的初始提示词构建：

```go
sections := DefaultPromptSections("", "")
// 初始构建不需要 skillListing，因为下面 5b 会重建。保持原样即可。
```

注意：第 238 行的初始构建仅用于 compaction 检查的 token 估算，275 行才是实际使用的提示词。所以只需改 275 行。

- [ ] **Step 5: 修改 routes.go 传参**

在 `routes.go` 约第 64 行：

```go
chatSvc := NewChatService(db, provider, toolExecutor, maxIterations, skillLoader)
```

- [ ] **Step 6: 修复其他 NewChatService 调用点**

搜索所有 `NewChatService(` 调用，确保最后一个参数是 `skillLoader`（或 `nil`）。包括测试文件。

- [ ] **Step 7: 运行全部测试确认通过**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/... -v`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add backend/internal/modules/agent/service.go backend/internal/modules/agent/routes.go backend/internal/modules/agent/service_test.go
git commit -m "feat: ChatService 接收 SkillLoader 并构建技能清单注入系统提示词"
```

---

### Task 4: 新增 A4 模板参考文档

**Files:**
- Create: `backend/internal/modules/agent/skills/resume-design/references/a4-template.yaml`
- Modify: `backend/internal/modules/agent/skills/resume-design/skill.yaml`

- [ ] **Step 1: 创建 a4-template.yaml**

创建文件 `backend/internal/modules/agent/skills/resume-design/references/a4-template.yaml`：

```yaml
name: a4-template
content: |
  ## A4 简历精确尺寸与模板

  ### 关键尺寸（96dpi 下的像素值）
  - 画布宽高：794px x 1123px（对应 210mm x 297mm）
  - 页边距：上 68px、下 68px、左 76px、右 76px（对应 18mm / 20mm）
  - 有效内容区：642px x 987px（794-76-76 x 1123-68-68）
  - 分页间距：32px（仅前端显示，不影响内容计算）

  ### 内容容量参考
  - 正文字号 14px、行高 1.5 → 每行约 21px，有效区可容纳约 47 行
  - 每行约 45-50 个中文字符（642px 宽度下）
  - 单栏布局一页约 1500-2000 字纯文本
  - 双栏布局因列间距损耗，总容量约减少 10-15%

  ### HTML 模板骨架
  生成简历时应遵循以下结构，确保内容自然适配 A4：

  ```html
  <!-- 顶部联系信息区：不超过 60px 高度 -->
  <header style="padding-bottom: 8px; border-bottom: 1px solid #ddd;">
    <div style="font-size: 20px; font-weight: 600;">姓名</div>
    <div style="font-size: 12px; color: #666;">电话 | 邮箱 | 地址</div>
  </header>

  <!-- 内容分区：紧凑标题 + bullet 列表 -->
  <section style="margin-top: 8px;">
    <h2 style="font-size: 14px; border-bottom: 1px solid #eee; padding-bottom: 2px;">工作经历</h2>
    <div style="margin-top: 4px;">
      <div><strong>公司名</strong> | 职位 | 时间</div>
      <ul style="margin: 2px 0; padding-left: 16px; font-size: 13px;">
        <li>成果描述</li>
      </ul>
    </div>
  </section>
  ```

  ### 控制篇幅的具体操作
  1. 减少 bullet 条目（每个经历 2-3 条而非 4-5 条）
  2. 合并短条目为一段描述
  3. 技能用逗号分隔文本，不用大块色卡
  4. 减小段间距至 4-8px
  5. 缩短描述文案，删掉修饰性形容词
```

- [ ] **Step 2: 更新 skill.yaml 注册新 reference**

修改 `backend/internal/modules/agent/skills/resume-design/skill.yaml`：

```yaml
name: resume-design
description: |
  A4 单页简历设计规范。当用户要求调整样式、排版、配色、模板时使用。
trigger: 用户要求调整简历样式或需要设计参考时

usage: |
  1. 调用 get_skill_reference(skill_name="resume-design", reference_name="a4-guidelines") 获取完整规范
  2. 基于规范中的推荐样式和禁止样式修改简历 CSS

tools:
  - name: get_skill_reference
    description: 获取 A4 简历设计规范
    params:
      - name: skill_name
        type: string
        description: 固定传 "resume-design"
      - name: reference_name
        type: string
        description: 固定传 "a4-guidelines"

references:
  - name: a4-guidelines
    description: A4 单页简历设计规范，包含推荐样式、禁止样式和修改策略
  - name: a4-template
    description: A4 精确像素尺寸、内容容量参考和 HTML 模板骨架
```

- [ ] **Step 3: 写测试验证新 reference 能被加载**

在 `skill_loader_test.go` 中添加：

```go
func TestSkillLoader_A4TemplateReference(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	result, err := loader.LoadSkillWithReferences("resume-design")
	require.NoError(t, err)

	refNames := make([]string, len(result.References))
	for i, ref := range result.References {
		refNames[i] = ref.Name
	}
	assert.Contains(t, refNames, "a4-guidelines", "should have a4-guidelines reference")
	assert.Contains(t, refNames, "a4-template", "should have a4-template reference")

	// Verify a4-template has content
	for _, ref := range result.References {
		if ref.Name == "a4-template" {
			assert.Contains(t, ref.Content, "794px", "should contain canvas width")
			assert.Contains(t, ref.Content, "987px", "should contain content area height")
			assert.Contains(t, ref.Content, "1500-2000", "should contain capacity estimate")
			return
		}
	}
	t.Fatal("a4-template reference not found")
}
```

- [ ] **Step 4: 运行测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestSkillLoader_A4TemplateReference -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/agent/skills/resume-design/references/a4-template.yaml backend/internal/modules/agent/skills/resume-design/skill.yaml backend/internal/modules/agent/skill_loader_test.go
git commit -m "feat: 新增 a4-template 简历设计参考文档"
```

---

### Task 5: get_draft 返回 page_count

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go:274-356`
- Test: `backend/internal/modules/agent/tool_executor_test.go`

- [ ] **Step 1: 写失败测试**

在 `tool_executor_test.go` 中添加：

```go
func TestGetDraft_Full_ReturnsPageCount(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body><p>hello</p></body></html>", PageCount: 2}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(2), data["page_count"], "should return page_count from DB")
	assert.Contains(t, data, "html", "should still contain html")
}

func TestGetDraft_Full_PageCountZero(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	draft := models.Draft{ProjectID: proj.ID, HTMLContent: "<html><body><p>hello</p></body></html>"}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, float64(0), data["page_count"], "default page_count should be 0")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run "TestGetDraft_Full_ReturnsPageCount|TestGetDraft_Full_PageCountZero" -v`
Expected: FAIL — 现在返回纯 HTML 字符串，不是 JSON

- [ ] **Step 3: 修改 getDraft 函数**

修改 `tool_executor.go` 的 `getDraft` 方法。核心改动：

1. 查询 Draft 时增加 `page_count` 字段到 Select：

```go
if err := e.db.WithContext(ctx).Select("id", "html_content", "page_count").First(&draft, draftID).Error; err != nil {
```

2. 每个返回纯 HTML 的 case 改为返回 JSON（`{ html, page_count }`）。只改 `full` 模式（AI 主要用这个），其他模式保持原样：

```go
case "full":
	runes := []rune(draft.HTMLContent)
	if len(runes) == 0 {
		return "当前简历 HTML 为空，请直接使用 apply_edits 创建完整简历内容。", nil
	}
	debugLog("tools", "get_draft mode=full len=%d page_count=%d", len(draft.HTMLContent), draft.PageCount)
	resultData := map[string]interface{}{
		"html":       draft.HTMLContent,
		"page_count": draft.PageCount,
	}
	b, err := json.Marshal(resultData)
	if err != nil {
		return "", fmt.Errorf("marshal get_draft result: %w", err)
	}
	return string(b), nil
```

3. `structure` 和 `section` 和 `search` 模式保持返回纯文本/HTML 不变（这些是辅助查询，不需要 page_count）。

- [ ] **Step 4: 更新现有 TestGetDraft_Full 测试**

现有 `TestGetDraft_Full` 断言 `result == html`，现在返回 JSON，需要更新：

```go
func TestGetDraft_Full(t *testing.T) {
	db := SetupTestDB(t)
	executor := NewAgentToolExecutor(db, nil)

	proj := models.Project{Title: "Test", Status: "active"}
	require.NoError(t, db.Create(&proj).Error)

	html := `<html><body><h1>Hello</h1><p>World</p></body></html>`
	draft := models.Draft{ProjectID: proj.ID, HTMLContent: html}
	require.NoError(t, db.Create(&draft).Error)

	ctx := WithDraftID(context.Background(), draft.ID)
	result, err := executor.Execute(ctx, "get_draft", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	assert.Equal(t, html, data["html"])
	assert.Contains(t, data, "page_count")
}
```

- [ ] **Step 5: 运行全部 get_draft 测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/ -run TestGetDraft -v`
Expected: ALL PASS

- [ ] **Step 6: 运行全部 agent 测试确认不破坏**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/agent/... -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go
git commit -m "feat: get_draft 返回 page_count 供 AI 感知页数"
```

---

### Task 6: PATCH /drafts/:id/meta 端点

**Files:**
- Modify: `backend/internal/modules/workbench/routes.go`（或 handler.go，视现有路由注册位置而定）
- Test: 对应的测试文件

- [ ] **Step 1: 找到 workbench 模块的路由和 handler 文件**

查看 `backend/internal/modules/workbench/` 下的路由注册位置，确定 drafts 相关端点在哪里定义。

- [ ] **Step 2: 写 handler 方法**

在 workbench handler 中添加 `UpdateDraftMeta` 方法：

```go
func (h *Handler) UpdateDraftMeta(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("draft_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid draft_id")
		return
	}

	var req struct {
		PageCount int `json:"page_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result := h.db.Model(&models.Draft{}).Where("id = ?", draftID).Update("page_count", req.PageCount)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "update failed")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "draft not found")
		return
	}

	response.Success(c, nil)
}
```

- [ ] **Step 3: 注册路由**

在 workbench 路由注册中添加：

```go
drafts := rg.Group("/drafts")
// ... existing routes ...
drafts.PATCH("/:draft_id/meta", h.UpdateDraftMeta)
```

- [ ] **Step 4: 写测试**

```go
func TestUpdateDraftMeta(t *testing.T) {
	db := SetupTestDB(t)
	// 创建 test draft
	// 发送 PATCH 请求
	// 验证 page_count 更新
	// 验证 404 情况
}
```

- [ ] **Step 5: 运行测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./internal/modules/workbench/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/modules/workbench/
git commit -m "feat: PATCH /drafts/:id/meta 端点，前端回传页数"
```

---

### Task 7: 前端 TipTap onUpdate 回传页数

**Files:**
- Modify: `frontend/workbench/src/pages/EditorPage.tsx`
- Modify: `frontend/workbench/src/lib/api-client.ts`（如需新增 API 调用函数）

- [ ] **Step 1: 在 api-client 中新增 updateDraftMeta 函数**

```typescript
export async function updateDraftMeta(draftId: number, meta: { page_count: number }): Promise<void> {
  await apiClient.patch(`/drafts/${draftId}/meta`, meta)
}
```

- [ ] **Step 2: 在 EditorPage 中添加页数回传逻辑**

在 `EditorPage.tsx` 的 editor 创建后，添加 useEffect 监听 editor 的 update 事件：

```typescript
// Page count feedback
const lastPageCountRef = useRef(0)

useEffect(() => {
  if (!editor) return

  const handler = () => {
    // 从 PaginationPlus 获取页数
    const pages = editor.view.dom.closest('.resume-document')
      ?.querySelectorAll('.rm-page-break .page')
      ?.length || 1

    if (pages !== lastPageCountRef.current) {
      lastPageCountRef.current = pages
      updateDraftMeta(draftId, { page_count: pages }).catch(() => {
        // 静默失败，不影响编辑体验
      })
    }
  }

  editor.on('update', handler)
  return () => { editor.off('update', handler) }
}, [editor, draftId])
```

注意：需要添加 debounce（约 500ms），避免频繁请求。可以用简单的 setTimeout 实现。

- [ ] **Step 3: 手动测试**

1. 启动后端：`cd backend && go run cmd/server/main.go`
2. 启动前端：`cd frontend/workbench && bun run dev`
3. 打开编辑器，编辑简历内容触发分页变化
4. 检查浏览器 Network 面板确认 PATCH 请求发送
5. 检查数据库确认 page_count 更新

- [ ] **Step 4: Commit**

```bash
git add frontend/workbench/src/pages/EditorPage.tsx frontend/workbench/src/lib/api-client.ts
git commit -m "feat: 前端 TipTap 渲染后回传页数到后端"
```

---

### Task 8: 集成验证

**Files:** 无新增

- [ ] **Step 1: 运行后端全部测试**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: 编译检查**

Run: `cd /home/handy/Projects/ResumeGenius/backend && go build ./cmd/server/...`
Expected: 无错误

- [ ] **Step 3: 运行前端测试**

Run: `cd /home/handy/Projects/ResumeGenius/frontend/workbench && bunx vitest run`
Expected: ALL PASS

- [ ] **Step 4: 端到端验证**

1. 启动服务
2. 创建项目、上传简历、开始 AI 对话
3. 验证系统提示词中包含技能清单（查看后端日志 debugLog 输出）
4. AI 调用 get_draft 后返回 JSON 包含 page_count
5. AI 调用 load_skill("resume-design") 后返回 a4-template 参考
6. 前端编辑后 page_count 正确回传

- [ ] **Step 5: Final commit (如有修复)**

```bash
git add -A
git commit -m "fix: 集成验证修复"
```
