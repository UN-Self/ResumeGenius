# Skill 系统三层渐进式披露实施设计

更新时间：2026-05-09
Issue: #45
状态: 设计已完成，待实施

## 1. 设计目标

将现有的单工具 `search_skills` 模式重构为三层渐进式披露架构：
- **Layer 1**（System Prompt）：技能索引（名称 + 触发条件），固定不变
- **Layer 2**（`load_skill`）：技能描述文档（用途、工具、reference 列表）
- **Layer 3**（`get_skill_reference`）：具体 reference 内容

## 2. 实施方案

选择 **方案 B：一次性完整迁移**

- 删除旧的 `search_skills` 工具和 `Search()` 方法
- 新增 `load_skill` 和 `get_skill_reference` 工具
- 迁移所有现有 skill 到新目录结构
- 不创建空占位符 reference（只声明实际存在的）

## 3. 目录结构

### 3.1 目标结构

```
backend/internal/modules/agent/skills/
├── resume-interview/
│   ├── skill.yaml                  # Layer 2: 描述文档
│   └── references/
│       └── test-engineer.yaml      # Layer 3: 测试工程师面经
├── resume-design/
│   ├── skill.yaml                  # Layer 2: 描述文档
│   └── references/
│       └── a4-guidelines.yaml      # Layer 3: A4 设计规范
└── README.md                       # 更新说明
```

### 3.2 删除文件

- `test/test-resume.yaml`
- `design/resume-design.yaml`

## 4. 核心组件设计

### 4.1 SkillLoader 重构

#### 新增结构体

```go
// SkillDescriptor 表示 Layer 2 描述文档
type SkillDescriptor struct {
    Name        string              `yaml:"name"`
    Description string              `yaml:"description"`
    Trigger     string              `yaml:"trigger"`
    Usage       string              `yaml:"usage"`
    Tools       []ToolDefinition    `yaml:"tools"`
    References  []ReferenceMetadata `yaml:"references"`
}

// ToolDefinition 描述二级工具
type ToolDefinition struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Params      []Param `yaml:"params"`
    Usage       string `yaml:"usage"`
}

// Param 描述工具参数
type Param struct {
    Name        string `yaml:"name"`
    Type        string `yaml:"type"`
    Description string `yaml:"description"`
}

// ReferenceMetadata 描述 reference 元信息
type ReferenceMetadata struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
}

// ReferenceContent 表示 Layer 3 内容
type ReferenceContent struct {
    Name    string `yaml:"name"`
    Content string `yaml:"content"`
}
```

#### 方法变更

```go
// 新增方法
func (l *SkillLoader) LoadSkill(name string) (*SkillDescriptor, error)
func (l *SkillLoader) GetReference(skillName, refName string) (*ReferenceContent, error)

// 删除方法
// func (l *SkillLoader) Search(keyword, category string, limit int) []SkillFile
```

#### embed.FS 模式更新

```go
//go:embed skills/*/*.yaml skills/*/*/*.yaml
var skillsFS embed.FS
```

### 4.2 会话状态追踪（顺序校验）

```go
type AgentToolExecutor struct {
    db          *gorm.DB
    skillLoader *SkillLoader
    // 会话级别状态：key=sessionID, value=已加载的 skill 集合
    loadedSkills sync.Map
}

// 辅助方法
func (e *AgentToolExecutor) markSkillLoaded(sessionID uint, skillName string)
func (e *AgentToolExecutor) isSkillLoaded(sessionID uint, skillName string) bool
func (e *AgentToolExecutor) clearSessionSkills(sessionID uint)
```

### 4.3 工具定义

#### 删除

```go
// search_skills 工具及其执行方法
```

#### 新增

```go
{
    Name:        "load_skill",
    Description: "加载指定技能的描述文档，了解技能的用途、可用工具和参考资源",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "skill_name": map[string]interface{}{
                "type":        "string",
                "description": "技能名称，如 'resume-interview'、'resume-design'",
            },
        },
        "required": []string{"skill_name"},
    },
},
{
    Name:        "get_skill_reference",
    Description: "获取技能库中指定岗位的面经内容或设计规范。必须在 load_skill 之后使用。",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "skill_name": map[string]interface{}{
                "type":        "string",
                "description": "技能名称，必须先通过 load_skill 加载该技能",
            },
            "reference_name": map[string]interface{}{
                "type":        "string",
                "description": "reference 名称，如 'test-engineer'、'a4-guidelines'",
            },
        },
        "required": []string{"skill_name", "reference_name"},
    },
}
```

### 4.4 System Prompt 更新

#### 删除

```text
## 技能库（Skills）
你可以使用 search_skills 工具查找简历优化技能。
技能库包含岗位面经、面试官关注点、简历针对性修改建议，以及简历设计规范。

使用时机：当用户明确了目标岗位（如"测试工程师"、"前端开发"、"产品经理"等），
先调用 search_skills 获取该岗位的面经和建议，再基于建议修改简历。
```

#### 替换为

```text
## 技能库（Skills）
- resume-interview: 根据目标岗位的面试官视角优化简历，提供岗位面经、面试官关注点和简历针对性修改建议。当用户明确了目标岗位（如"测试工程师"、"前端开发"）时使用。
- resume-design: 提供 A4 单页简历设计规范和保守风格参考，帮助用户调整视觉、排版、配色。当用户要求调整简历样式或需要设计参考时使用。
```

**关键**：格式为 `name: description + 触发条件`，不提及工具名。

### 4.5 Context Key 扩展

```go
const (
    draftIDKey contextKey = iota
    projectIDKey
    sessionIDKey  // 新增：用于会话状态追踪
)

func WithSessionID(ctx context.Context, id uint) context.Context {
    return context.WithValue(ctx, sessionIDKey, id)
}
```

## 5. 数据流

```
用户: "我要应聘测试工程师，帮我改简历"

1. AI 读 system prompt → 判断命中 resume-interview
2. AI 调用 load_skill("resume-interview")
   → 返回描述文档（usage + tools + references 列表）
3. AI 读描述文档 → 知道该用 get_skill_reference
4. AI 调用 get_skill_reference("resume-interview", "test-engineer")
   → 校验：resume-interview 是否已加载 ✓
   → 返回测试工程师面经全文
5. AI 基于面经内容修改简历
```

## 6. 错误处理

| 场景 | 错误消息 | 目的 |
|---|---|---|
| skill 不存在 | `{"error": "skill not found: xxx"}` | 明确告知 skill 不存在 |
| skill 未加载 | `{"error": "skill 'resume-interview' not loaded: call load_skill('resume-interview') first"}` | 引导 AI 正确调用顺序 |
| reference 不存在 | `{"error": "reference 'frontend-developer' not found in skill 'resume-interview'. Available references: [test-engineer]"}` | 告知可用选项 |
| 参数缺失 | `{"error": "skill_name is required: you must call load_skill first"}` | 明确参数要求 |
| YAML 解析失败 | `{"error": "failed to parse skill file: xxx"}` | 内部错误，需修复 |

## 7. YAML 文件示例

### 7.1 resume-interview/skill.yaml

```yaml
name: resume-interview
description: 根据目标岗位的面试官视角优化简历，提供岗位面经、面试官关注点和简历针对性修改建议
trigger: 用户提到目标岗位或需要面试相关建议时

usage: |
  1. 确认用户的目标岗位
  2. 使用 get_skill_reference 获取该岗位的面经
  3. 基于面经中的面试官关注点修改简历

tools:
  - name: get_skill_reference
    description: 获取指定岗位的面经内容（面试题、参考答案、面试官关注点、简历修改建议）
    params:
      - name: skill_name
        type: string
        description: 固定传 "resume-interview"
      - name: reference_name
        type: string
        description: 岗位名，如 "test-engineer"
    usage: 调用后会返回该岗位的完整面经，确保已确认用户岗位后再调用

references:
  - name: test-engineer
    description: 测试工程师岗位面经，含 18 道面试题及面试官建议
```

### 7.2 resume-interview/references/test-engineer.yaml

```yaml
name: test-engineer
content: |
  ## 面试经验：测试工程师

  ### 常见面试题
  1. **黑盒测试有哪些方法？**
     回答要点：...
  ...（完整面经内容）
```

### 7.3 resume-design/skill.yaml

```yaml
name: resume-design
description: 提供 A4 单页简历设计规范和保守风格参考，包含推荐样式、禁止样式和修改策略
trigger: 用户要求调整视觉、排版、配色、模板、样式时

usage: |
  1. 使用 get_skill_reference 获取 A4 设计规范
  2. 基于规范指导简历样式修改

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
    usage: 返回完整的 A4 单页简历设计规范

references:
  - name: a4-guidelines
    description: A4 单页简历设计规范，包含推荐样式、禁止样式和修改策略
```

## 8. 测试策略

### 8.1 单元测试

新增以下测试用例：

```go
// SkillLoader 测试
func TestSkillLoader_LoadSkill_Existing()
func TestSkillLoader_LoadSkill_NotFound()
func TestSkillLoader_GetReference_Existing()
func TestSkillLoader_GetReference_NotFound()
func TestSkillLoader_GetReference_WrongSkill()

// ToolExecutor 测试
func TestToolExecutor_LoadSkill_Valid()
func TestToolExecutor_LoadSkill_InvalidName()
func TestToolExecutor_GetReference_Valid()
func TestToolExecutor_GetReference_SkillNotLoaded()  // 顺序校验
func TestToolExecutor_GetReference_ReferenceNotFound()
```

删除以下测试用例：

```go
func TestSkillLoader_SearchByKeyword()
func TestSkillLoader_SearchByCategory()
func TestSkillLoader_SearchAllReturnsFullContent()
```

### 8.2 集成测试

```go
func TestToolExecutor_CompleteFlow() {
    // 1. load_skill("resume-interview")
    // 2. get_skill_reference("resume-interview", "test-engineer")
    // 验证完整流程
}

func TestService_SystemPrompt_CacheFriendly() {
    // 验证 system prompt 不包含动态内容
}
```

## 9. 实施检查清单

- [ ] 重构 `skills/` 目录结构
- [ ] 创建 `resume-interview/skill.yaml`
- [ ] 创建 `resume-interview/references/test-engineer.yaml`
- [ ] 创建 `resume-design/skill.yaml`
- [ ] 创建 `resume-design/references/a4-guidelines.yaml`
- [ ] 删除 `test/test-resume.yaml`
- [ ] 删除 `design/resume-design.yaml`
- [ ] 更新 `README.md`
- [ ] 重写 `SkillLoader`（新增 LoadSkill/GetReference，删除 Search）
- [ ] 更新 `embed.FS` 模式
- [ ] 替换 `tool_executor.go` 中的工具定义
- [ ] 新增会话状态追踪（loadedSkills sync.Map）
- [ ] 实现 `load_skill` 执行方法
- [ ] 实现 `get_skill_reference` 执行方法（含顺序校验）
- [ ] 删除 `search_skills` 执行方法
- [ ] 更新 `service.go` system prompt
- [ ] 新增 `sessionIDKey` 和 `WithSessionID` 函数
- [ ] 更新 `StreamChatReAct` 中的 context 设置
- [ ] 适配所有测试文件
- [ ] 新增 SkillLoader 单元测试
- [ ] 新增 ToolExecutor 顺序校验测试
- [ ] 运行完整测试套件验证

## 10. 依赖关系

- **依赖**：无
- **被依赖**：PR #47 Phase A（设计知识库重构）依赖本 Issue 完成

## 11. 后续工作

本 Issue 完成后，PR #47 Phase A 可以基于本架构将 `designskill` 模块纳入 `resume-design` skill。