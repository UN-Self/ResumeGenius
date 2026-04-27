# 模块 parsing — 文件解析与 AI 初稿生成 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 PDF 文本提取和调用 AI API 生成 HTML 初稿，自动写入 drafts 表并创建版本快照。

**Architecture:** ParsingService 提取 PDF 文本，DraftGenerator 调用 AI API 生成 HTML。`USE_MOCK=true` 时跳过真实 API 调用，返回 fixture mock 数据。

**Tech Stack:** github.com/ledongthuc/pdf / net/http (OpenAI-compatible API) / GORM

**Depends on:** Phase 0 共享基石完成、模块 intake 的文件上传

**契约文档:** `docs/modules/parsing/contract.md`

---

### Task 1: 后端 — PDF 解析

**Files:**
- Create: `backend/internal/modules/parsing/parser_test.go`
- Create: `backend/internal/modules/parsing/parser.go`

**Step 1: 写失败测试**

```go
// parser_test.go
package parsing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTextFromPDF(t *testing.T) {
	// 创建一个最小合法 PDF（只含 header 的空 PDF）
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%EOF")

	tmpFile := filepath.Join(t.TempDir(), "test.pdf")
	os.WriteFile(tmpFile, minimalPDF, 0644)

	text, err := ExtractTextFromPDF(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 空 PDF 应返回空字符串或至少不 panic
	_ = text
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/parsing/... -v
# Expected: FAIL
```

**Step 3: 安装依赖 + 实现**

```bash
cd backend && go get github.com/ledongthuc/pdf
```

```go
// parser.go
package parsing

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

func ExtractTextFromPDF(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 PDF 失败: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	for pageNum := 1; pageNum <= r.NumPage(); pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(content)
	}

	return buf.String(), nil
}

func ReadFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

**Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/parsing/... -v -run TestExtractTextFromPDF
# Expected: PASS
```

**Step 5: Commit**

```bash
git add backend/internal/modules/parsing/
git commit -m "feat(module-b): implement PDF text extraction"
```

---

### Task 2: 后端 — AI 初稿生成（AI API + Mock 模式）

**Files:**
- Create: `backend/internal/modules/parsing/generator_test.go`
- Create: `backend/internal/modules/parsing/generator.go`

**Step 1: 写失败测试**

```go
// generator_test.go
package parsing

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMockGenerateDraft(t *testing.T) {
	os.Setenv("USE_MOCK", "true")

	html, err := GenerateDraftHTML("张三\n前端工程师\n3年经验")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html == "" {
		t.Error("expected non-empty HTML")
	}
	if !strings.Contains(html, "<html") {
		t.Errorf("expected HTML content, got: %s", html[:min(100, len(html))])
	}
}

func TestMockParseResponse(t *testing.T) {
	mockResp := `{"choices":[{"message":{"content":"<!DOCTYPE html><html><body>mock</body></html>"}}]}`
	html, err := parseAIResponse(mockResp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "mock") {
		t.Errorf("unexpected HTML: %s", html)
	}
}
```

**Step 2: 实现 generator.go**

```go
// generator.go
package parsing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/handy/resume-genius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type DraftGenerator struct {
	db       *gorm.DB
	apiURL   string
	apiKey   string
}

func NewDraftGenerator(db *gorm.DB) *DraftGenerator {
	return &DraftGenerator{
		db:     db,
		apiURL: os.Getenv("AI_API_URL"),
		apiKey: os.Getenv("AI_API_KEY"),
	}
}

func (g *DraftGenerator) Generate(projectID uint, parsedText string) (*models.Draft, error) {
	html, err := g.callAPI(parsedText)
	if err != nil {
		return nil, fmt.Errorf("AI 初稿生成失败: %w", err)
	}

	// 写入 drafts 表
	draft := models.Draft{
		ProjectID:   projectID,
		HTMLContent: html,
	}
	if err := g.db.Create(&draft).Error; err != nil {
		return nil, err
	}

	// 自动创建初始版本
	label := "AI 初始生成"
	g.db.Create(&models.Version{
		DraftID:      draft.ID,
		HTMLSnapshot: html,
		Label:        &label,
	})

	// 更新项目的 current_draft_id
	g.db.Model(&models.Project{}).Where("id = ?", projectID).Update("current_draft_id", draft.ID)

	return &draft, nil
}

func (g *DraftGenerator) callAPI(parsedText string) (string, error) {
	if os.Getenv("USE_MOCK") == "true" {
		return mockDraftHTML(), nil
	}

	systemPrompt := `你是一个简历生成助手。根据用户提供的原始简历文本，生成一份专业的 HTML 格式简历。
要求：
1. 使用简洁的 HTML+CSS，适合 A4 打印
2. 包含：个人信息、工作经历、项目经历、教育经历、专业技能
3. 使用中文
4. 只返回 HTML 代码，不要其他解释`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      "default",
		"messages":   []map[string]string{{"role": "system", "content": systemPrompt}, {"role": "user", "content": parsedText}},
		"temperature": 0.7,
	})

	req, _ := http.NewRequest("POST", g.apiURL, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return parseAIResponse(string(body))
}

func parseAIResponse(body string) (string, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("AI 响应为空")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func mockDraftHTML() string {
	return `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"/><style>
@page{size:A4;margin:0}*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Noto Sans SC',sans-serif;font-size:10.5pt;line-height:1.4;color:#333}
.resume{width:210mm;min-height:297mm;padding:18mm 20mm}
.section{margin-bottom:10pt}.section h2{font-size:12pt;font-weight:600;border-bottom:1pt solid #ddd;padding-bottom:3pt;margin-bottom:6pt}
.item{margin-bottom:6pt}.item-header{display:flex;justify-content:space-between}
.item h3{font-size:10.5pt;font-weight:600}.item .date{font-size:9pt;color:#888}
</style></head><body><div class="resume"><header><h1>Mock Resume</h1><p>这是 AI 生成的简历初稿</p></header><section class="section"><h2>工作经历</h2><div class="item"><p>（内容由 AI 生成，请根据实际情况修改）</p></div></section></div></body></html>`
}

// 公开的 GenerateDraftHTML 函数（供测试用）
func GenerateDraftHTML(text string) (string, error) {
	gen := &DraftGenerator{}
	return gen.callAPI(text)
}
```

**Step 3: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/parsing/... -v
# Expected: PASS
```

**Step 4: Commit**

```bash
git add backend/internal/modules/parsing/
git commit -m "feat(module-b): implement AI draft generation with AI API + mock mode"
```

---

### Task 3: 后端 — Parse + Generate API 端点

**Files:**
- Create: `backend/internal/modules/parsing/handler_test.go`
- Create: `backend/internal/modules/parsing/handler.go`
- Modify: `backend/internal/modules/parsing/routes.go`

**Step 1: 写失败测试**

```go
// handler_test.go
package parsing

import (
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/handy/resume-genius/backend/internal/shared/models"
)

func TestFullParsingFlow(t *testing.T) {
	os.Setenv("USE_MOCK", "true")

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Project{}, &models.Asset{}, &models.Draft{}, &models.Version{})

	// 创建测试数据
	db.Create(&models.Project{Title: "test", Status: "active"})
	tmpDir := t.TempDir()
	uri := tmpDir + "/resume.pdf"
	os.WriteFile(uri, []byte("%PDF-1.0\n%%EOF"), 0644)
	db.Create(&models.Asset{ProjectID: 1, Type: "resume_pdf", URI: &uri})

	generator := NewDraftGenerator(db)
	parsingSvc := NewParsingService(db, tmpDir, generator)

	result, err := parsingSvc.ParseAndGenerate(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DraftID == 0 {
		t.Error("expected non-zero draft_id")
	}

	var draft models.Draft
	db.First(&draft, result.DraftID)
	if draft.HTMLContent == "" {
		t.Error("expected non-empty html_content")
	}

	var versionCount int64
	db.Model(&models.Version{}).Count(&versionCount)
	if versionCount != 1 {
		t.Errorf("expected 1 version, got %d", versionCount)
	}
}
```

**Step 2: 实现 ParsingService + Handler**

```go
// 追加到 service.go (或新建)
type ParsingService struct {
	db         *gorm.DB
	uploadDir  string
	generator  *DraftGenerator
}

func NewParsingService(db *gorm.DB, uploadDir string, generator *DraftGenerator) *ParsingService {
	return &ParsingService{db: db, uploadDir: uploadDir, generator: generator}
}

type ParseResult struct {
	DraftID    uint   `json:"draft_id"`
	VersionID  uint   `json:"version_id"`
	HTMLContent string `json:"html_content"`
}

func (s *ParsingService) Parse(projectID uint) ([]map[string]interface{}, error) {
	var assets []models.Asset
	s.db.Where("project_id = ?", projectID).Find(&assets)

	if len(assets) == 0 {
		return nil, fmt.Errorf("项目无可用资产")
	}

	var results []map[string]interface{}
	for _, asset := range assets {
		if asset.URI == nil {
			continue
		}
		text, err := ExtractTextFromPDF(*asset.URI)
		if err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"asset_id": asset.ID,
			"type":     asset.Type,
			"text":     text,
		})
	}
	return results, nil
}

func (s *ParsingService) ParseAndGenerate(projectID uint) (*ParseResult, error) {
	parsed, err := s.Parse(projectID)
	if err != nil {
		return nil, err
	}

	// 合并所有解析文本
	var allText string
	for _, p := range parsed {
		allText += p["text"].(string) + "\n"
	}

	draft, err := s.generator.Generate(projectID, allText)
	if err != nil {
		return nil, err
	}

	var version models.Version
	s.db.Where("draft_id = ?", draft.ID).First(&version)

	return &ParseResult{
		DraftID:     draft.ID,
		VersionID:   version.ID,
		HTMLContent: draft.HTMLContent,
	}, nil
}
```

```go
// handler.go
package parsing

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/handy/resume-genius/backend/internal/shared/response"
)

type Handler struct {
	svc *ParsingService
}

func NewHandler(svc *ParsingService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Parse(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	if projectID == 0 {
		response.Error(c, 40000, "project_id is required")
		return
	}
	results, err := h.svc.Parse(uint(projectID))
	if err != nil {
		response.Error(c, 2004, err.Error())
		return
	}
	response.Success(c, gin.H{"parsed_contents": results})
}

func (h *Handler) Generate(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	if projectID == 0 {
		response.Error(c, 40000, "project_id is required")
		return
	}
	result, err := h.svc.ParseAndGenerate(uint(projectID))
	if err != nil {
		response.Error(c, 2005, err.Error())
		return
	}
	response.Success(c, result)
}
```

**Step 3: 更新 routes.go**

```go
package parsing

import (
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	generator := NewDraftGenerator(db)
	svc := NewParsingService(db, uploadDir, generator)
	h := NewHandler(svc)

	rg.GET("/parse", h.Parse)
	rg.POST("/generate", h.Generate)
}
```

**Step 4: 运行测试确认通过**

```bash
USE_MOCK=true go test ./internal/modules/parsing/... -v
# Expected: PASS
```

**Step 5: Commit**

```bash
git add backend/internal/modules/parsing/
git commit -m "feat(module-b): implement parse + generate API endpoints"
```

---

## 验证清单

- [ ] `go test ./internal/modules/parsing/... -v` 全部通过
- [ ] `USE_MOCK=true` 启动后端
- [ ] `curl "localhost:8080/api/v1/parsing/parse?project_id=1"` 返回解析结果
- [ ] `curl -X POST "localhost:8080/api/v1/parsing/generate?project_id=1"` 返回 draft_id + html_content
- [ ] drafts 表有新记录，versions 表有自动快照
- [ ] `USE_MOCK=false` + 配置 AI API 环境变量后能调用真实 API
