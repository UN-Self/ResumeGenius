# Agent Skill 系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Skill 系统让 AI 在对话中按岗位查找面经（面试题 + 参考答案 + 面试官关注点），基于面经针对性修改简历。

**Architecture:** Skill 文件存为 YAML，Go 1.16+ `embed.FS` 编译时打包进二进制。SkillLoader 启动时加载全部 skill 并解析为结构化对象，`search_skills` 工具让 AI 按关键词/分类按需检索返回完整 skill 内容。渐进式披露：system prompt 仅声明 skill 库存在，不加载具体内容。

**Tech Stack:** Go 1.25, `gopkg.in/yaml.v3`, `embed.FS`

---

### Task 1: 安装 yaml 依赖

**Files:**
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`

- [ ] **Step 1: 确保 yaml.v3 为直接依赖**

```bash
cd backend && go get gopkg.in/yaml.v3
```

- [ ] **Step 2: 验证**

```bash
go build ./...
```

Expected: Build succeeds (可能已有该依赖，仅升级版本)。

---

### Task 2: 创建 Skill 文件

**Files:**
- Create: `backend/internal/modules/agent/skills/README.md`
- Create: `backend/internal/modules/agent/skills/test/test-resume.yaml`

- [ ] **Step 1: 创建 `skills/README.md`**

```markdown
# Agent Skill 库

本目录按岗位分类存储面经（面试题 + 参考答案 + 面试官关注点）和简历针对性修改建议。

## 目录结构

```
skills/
├── README.md          # 本文件
├── test/              # 测试/QA 岗位
├── tech/              # 技术岗位（待扩展）
├── management/        # 管理岗位（待扩展）
└── creative/          # 创意岗位（待扩展）
```

## 使用方式

System prompt 告知 AI 技能库的存在。AI 在用户明确目标岗位后，调用 `search_skills` 工具按关键词或分类检索匹配的 skill，获取面经内容后应用于简历修改。
```

- [ ] **Step 2: 创建 `skills/test/test-resume.yaml`**

```yaml
name: test-resume
description: >-
  面经和简历优化建议，适用于测试工程师岗位，
  强调测试方法论、自动化能力和质量意识
metadata:
  category: test
  keywords:
    - 测试工程师
    - QA
    - 质量保障
    - 自动化测试
  seniority:
    - junior
    - mid
  industries:
    - 互联网
    - 软件
references:
  - title: ISTQB Foundation Level Syllabus
    source: ISTQB
  - title: Google Testing Blog - Just Say No to More End-to-End Tests
    source: https://testing.googleblog.com
  - title: 测试架构师成长路径
    source: 极客时间专栏
content: |
  ## 面试经验：测试工程师

  ### 常见面试题
  1. **如何设计一个完整的测试用例？**
     回答要点：从等价类划分、边界值分析、场景法、错误推测法四个维度入手。区分功能测试、性能测试、安全测试、兼容性测试。结合需求文档和用户故事，确保正向用例和反向用例覆盖完整。

  2. **自动化测试如何落地？**
     回答要点：先评估 ROI，选择高频回归场景。分层策略：单元测试（50%）> 接口测试（30%）> UI 自动化（20%）。工具选型结合团队技术栈。关键是持续集成流水线中的稳定性维护。

  3. **上线后发现 bug 怎么处理？**
     回答要点：立即评估严重等级和影响范围。紧急修复走 hotfix 流程。复盘：漏测原因分析（用例缺失？环境差异？数据问题？），补充测试用例，改进测试流程。

  4. **接口测试和 UI 测试的取舍？**
     回答要点：接口测试覆盖业务逻辑，运行快、维护成本低；UI 测试覆盖用户操作流程，运行慢、易碎。推荐重心放在接口测试，UI 测试只覆盖核心流程。

  ### 面试官高频关注点
  - 自动化测试框架从零到一的搭建经验（不仅是使用）
  - CI/CD 流水线中的测试集成实践
  - 质量度量和改进闭环（缺陷分析、覆盖率追踪）
  - 测试左移：需求评审阶段的测试设计能力
  - 测试环境的治理和数据构造能力

  ### 简历针对性要点
  - 项目经历中强调**主导**而非"参与"，突出测试框架选型和技术决策
  - 每个项目都要有量化指标：覆盖率从 x% 提升到 y%，自动化率 z%，减少线上故障 n 起
  - 工具链集中展示：Selenium/Cypress/Playwright、JMeter/Locust、Postman、Docker
  - 测试方法论单独成段：TDD/BDD、探索式测试、基于风险的测试
  - 突出跨团队协作和推动质量改进的能力
  - 有 ISTQB 等证书放在显眼位置
```

---

### Task 3: 实现 SkillLoader

**Files:**
- Create: `backend/internal/modules/agent/skill_loader.go`

- [ ] **Step 1: 创建 skill_loader.go，实现结构和 embed 加载**

```go
package agent

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed skills/*.md skills/*/*.yaml
var skillsFS embed.FS

// SkillFile represents a single skill definition loaded from a YAML file.
type SkillFile struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Metadata    SkillMetadata   `yaml:"metadata"`
	References  []SkillReference `yaml:"references"`
	Content     string          `yaml:"content"`
}

// SkillMetadata holds structured metadata for skill search.
type SkillMetadata struct {
	Category   string   `yaml:"category"`
	Keywords   []string `yaml:"keywords"`
	Seniority  []string `yaml:"seniority"`
	Industries []string `yaml:"industries"`
}

// SkillReference holds a reference source for the skill content.
type SkillReference struct {
	Title  string `yaml:"title"`
	Source string `yaml:"source"`
}

// SkillSummary is a lightweight representation returned when no search filter is given.
type SkillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// SkillLoader loads and provides search over skill files.
type SkillLoader struct {
	skills []SkillFile
}

// NewSkillLoader creates a SkillLoader by walking the embedded skills directory
// and parsing all .yaml files.
func NewSkillLoader() (*SkillLoader, error) {
	loader := &SkillLoader{}
	err := fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		data, err := skillsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read skill file %s: %w", path, err)
		}
		var skill SkillFile
		if err := yaml.Unmarshal(data, &skill); err != nil {
			return fmt.Errorf("parse skill file %s: %w", path, err)
		}
		loader.skills = append(loader.skills, skill)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	return loader, nil
}

// Skills returns the full list of loaded skills.
func (l *SkillLoader) Skills() []SkillFile {
	return l.skills
}

// Search returns skill summaries (name + description + category) when keyword
// and category are empty, or full SkillFile objects when a filter is provided.
//
//   - keyword: matched against SkillFile.Metadata.Keywords (case-insensitive contains)
//   - category: matched against SkillFile.Metadata.Category (case-insensitive exact match)
//   - limit: max results returned (0 = no limit)
//   - summaryOnly: when true, returns lightweight summaries instead of full skill content
func (l *SkillLoader) Search(keyword, category string, limit int, summaryOnly bool) []interface{} {
	if len(l.skills) == 0 {
		return []interface{}{}
	}

	var matched []SkillFile
	for _, s := range l.skills {
		if keyword != "" {
			if !matchKeyword(s, keyword) {
				continue
			}
		}
		if category != "" {
			if !strings.EqualFold(s.Metadata.Category, category) {
				continue
			}
		}
		matched = append(matched, s)
	}

	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	if summaryOnly || (keyword == "" && category == "") {
		result := make([]interface{}, len(matched))
		for i, s := range matched {
			result[i] = SkillSummary{
				Name:        s.Name,
				Description: s.Description,
				Category:    s.Metadata.Category,
			}
		}
		return result
	}

	result := make([]interface{}, len(matched))
	for i, s := range matched {
		result[i] = s
	}
	return result
}

// matchKeyword checks if the keyword matches any of the skill's keywords.
// Matching is case-insensitive and uses contains semantics.
func matchKeyword(s SkillFile, keyword string) bool {
	kw := strings.ToLower(keyword)
	for _, k := range s.Metadata.Keywords {
		if strings.Contains(strings.ToLower(k), kw) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(s.Name), kw) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Description), kw) {
		return true
	}
	return false
}
```

---

### Task 4: 修改 tool_executor.go — 添加 search_skills 工具

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor.go`

- [ ] **Step 1: 修改 `AgentToolExecutor` 结构体，新增 `skillLoader` 字段**

```go
// AgentToolExecutor implements ToolExecutor using database queries.
type AgentToolExecutor struct {
	db          *gorm.DB
	skillLoader *SkillLoader
}

// NewAgentToolExecutor creates a new AgentToolExecutor.
func NewAgentToolExecutor(db *gorm.DB, skillLoader *SkillLoader) *AgentToolExecutor {
	return &AgentToolExecutor{db: db, skillLoader: skillLoader}
}
```

- [ ] **Step 2: 在 `Tools()` 方法中添加 `search_skills` 工具定义**

找到 `Tools()` 的返回数组，在 `search_assets` 后面添加：

```go
{
    Name:        "search_skills",
    Description: "搜索简历优化技能库（面经）。根据用户目标岗位的关键词（如'测试工程师'、'QA'）或分类查找匹配的面经和简历修改建议。不传参数返回全部技能摘要。",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "keyword": map[string]interface{}{
                "type":        "string",
                "description": "搜索关键词，按岗位名或技能名匹配，如'测试工程师'、'QA'、'自动化测试'",
            },
            "category": map[string]interface{}{
                "type":        "string",
                "description": "技能分类名，如'test'、'tech'、'management'",
            },
            "limit": map[string]interface{}{
                "type":        "integer",
                "description": "返回数量上限，默认 3",
            },
        },
        "required": []string{},
    },
},
```

- [ ] **Step 3: 在 `Execute()` 方法的 `switch` 中添加 `search_skills` case**

```go
case "search_skills":
    return e.searchSkills(ctx, params)
```

- [ ] **Step 4: 新增 `searchSkills` 方法**

```go
func (e *AgentToolExecutor) searchSkills(ctx context.Context, params map[string]interface{}) (string, error) {
	if e.skillLoader == nil {
		return `{"skills":[],"message":"技能库未加载"}`, nil
	}

	keyword, _ := params["keyword"].(string)
	category, _ := params["category"].(string)
	limit := 3
	if _, ok := params["limit"]; ok {
		if n, err := getIntParam(params, "limit"); err == nil && n > 0 {
			limit = n
		}
	}

	summaryOnly := keyword == "" && category == ""
	results := e.skillLoader.Search(keyword, category, limit, summaryOnly)

	resp := map[string]interface{}{
		"skills": results,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("marshal search results: %w", err)
	}
	return string(b), nil
}
```

---

### Task 5: 修改 service.go — 更新 system prompt

**Files:**
- Modify: `backend/internal/modules/agent/service.go`

- [ ] **Step 1: 在 `systemPromptV2` 末尾追加 skill 使用声明**

找到 `## 回复规范` 后面的结束位置，在 const 结束前追加：

```go
## 技能库（Skills）
你可以使用 search_skills 工具查找简历优化技能。
技能库按岗位分类存放，每个技能包含对应岗位的常见面试题、
面试官关注点和简历针对性修改建议。

使用时机：当用户明确了目标岗位（如"测试工程师"、"前端开发"、"产品经理"等），
先调用 search_skills 获取该岗位的面经和建议，再基于建议修改简历。
```

---

### Task 6: 修改 routes.go — 初始化 SkillLoader

**Files:**
- Modify: `backend/internal/modules/agent/routes.go`

- [ ] **Step 1: 在 `RegisterRoutes` 开头创建 SkillLoader**

```go
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	skillLoader, err := NewSkillLoader()
	if err != nil {
		log.Printf("agent: failed to load skills (non-fatal): %v", err)
		skillLoader = nil
	}

	sessionSvc := NewSessionService(db)
	// ...
	toolExecutor := NewAgentToolExecutor(db, skillLoader)
	// ... (rest unchanged)
```

注意：需要在文件顶部添加 `log` 包的 import。

---

### Task 7: 适配测试文件

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor_test.go`
- Modify: `backend/internal/modules/agent/testutil.go`

- [ ] **Step 1: 修改 `tool_executor_test.go` 中所有 `NewAgentToolExecutor` 调用**

所有 `NewAgentToolExecutor(db)` 改为 `NewAgentToolExecutor(db, nil)`。
`NewAgentToolExecutor(nil)` 改为 `NewAgentToolExecutor(nil, nil)`。

具体位置：
- TestToolExecutor_Tools_ThreeDefinitions: L20 → `NewAgentToolExecutor(nil, nil)`
- TestToolExecutor_Tools_NamesAreCorrect: L39 → `NewAgentToolExecutor(nil, nil)`
- TestToolExecutor_Tools_ParameterSchemas: L56 → `NewAgentToolExecutor(nil, nil)`
- TestGetDraft_Full: L108 → `NewAgentToolExecutor(db, nil)`
- TestGetDraft_Selector: L125 → `NewAgentToolExecutor(db, nil)`
- TestGetDraft_ContextMissing: L144 → `NewAgentToolExecutor(db, nil)`
- TestApplyEdits_Success: L157 → `NewAgentToolExecutor(db, nil)`
- TestApplyEdits_OldStringNotFound: L229 → `NewAgentToolExecutor(db, nil)`
- TestApplyEdits_BaseSnapshot: L264 → `NewAgentToolExecutor(db, nil)`
- TestSearchAssets: L298 → `NewAgentToolExecutor(db, nil)`
- TestSearchAssets_Empty: L342 → `NewAgentToolExecutor(db, nil)`
- TestExecute_UnknownTool: L365 → `NewAgentToolExecutor(nil, nil)`

- [ ] **Step 2: 修改 `TestToolExecutor_Tools_ThreeDefinitions` 断言**

```go
require.Len(t, tools, 4)   // was 3
```

- [ ] **Step 3: 修改 `TestToolExecutor_Tools_NamesAreCorrect` 期望列表**

```go
expected := []string{
    "get_draft",
    "apply_edits",
    "search_assets",
    "search_skills",
}
```

- [ ] **Step 4: 修改 `TestToolExecutor_Tools_ParameterSchemas` 添加 search_skills 断言**

在末尾 `}` 前添加：

```go
// search_skills: all optional, required = []
{
    tool := toolByName["search_skills"]
    props := tool.Parameters["properties"].(map[string]interface{})
    assert.Contains(t, props, "keyword")
    assert.Contains(t, props, "category")
    assert.Contains(t, props, "limit")
    req := tool.Parameters["required"].([]string)
    assert.Empty(t, req)
}
```

---

### Task 8: 编写 SkillLoader 单元测试

**Files:**
- Create: `backend/internal/modules/agent/skill_loader_test.go`

- [ ] **Step 1: 创建测试文件**

```go
package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillLoader_LoadAll(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	assert.NotEmpty(t, loader.Skills(), "should load at least one skill")
}

func TestSkillLoader_SearchByKeyword(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("测试工程师", "", 0, false)
	require.NotEmpty(t, results, "should find test-resume skill")

	skill, ok := results[0].(SkillFile)
	require.True(t, ok, "should return full SkillFile when summaryOnly=false")
	assert.Equal(t, "test-resume", skill.Name)
}

func TestSkillLoader_SearchByKeyword_WithLimit(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("测试工程师", "", 1, false)
	require.Len(t, results, 1)
}

func TestSkillLoader_SearchByCategory(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("", "test", 0, false)
	require.NotEmpty(t, results, "should find skills in test category")
}

func TestSkillLoader_SearchNoMatch(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("nonexistent_keyword_xyz", "", 0, false)
	assert.Empty(t, results)
}

func TestSkillLoader_SearchAllSummary(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("", "", 0, false)
	// 当 keyword 和 category 都为空时，返回 SkillSummary
	require.NotEmpty(t, results)
	summary, ok := results[0].(SkillSummary)
	require.True(t, ok, "should return SkillSummary when no filter")
	assert.NotEmpty(t, summary.Name)
	assert.NotEmpty(t, summary.Category)
}

func TestSkillLoader_ContentNotEmpty(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	for _, s := range loader.Skills() {
		assert.NotEmpty(t, s.Content, "skill %s must have content", s.Name)
		assert.NotEmpty(t, s.Metadata.Keywords, "skill %s must have keywords", s.Name)
		assert.NotEmpty(t, s.Metadata.Category, "skill %s must have category", s.Name)
	}
}
```

---

### Task 9: 编译和验证

- [ ] **Step 1: 编译验证**

```bash
cd backend && go build ./cmd/server/...
```
Expected: Build succeeds with no errors.

- [ ] **Step 2: 运行所有 agent 模块测试**

```bash
cd backend && go test ./internal/modules/agent/... -v
```
Expected: All tests pass (至少包括原有的 47 个后端测试 + 新增 skill_loader 测试)。

---

### Task 10: Git commit

- [ ] **Step 1: 提交**

```bash
git add backend/internal/modules/agent/skills/ backend/internal/modules/agent/skill_loader.go backend/internal/modules/agent/skill_loader_test.go backend/internal/modules/agent/tool_executor.go backend/internal/modules/agent/tool_executor_test.go backend/internal/modules/agent/service.go backend/internal/modules/agent/routes.go backend/go.mod backend/go.sum
git commit -m "feat(agent): 添加 skill 系统和 search_skills 工具

Skill 系统按岗位分类存储面经（面试题、参考答案、面试官关注点），
通过 search_skills 工具让 AI 按需检索，实现渐进式披露。
首发 test 分类下的测试工程师 skill。"
```
