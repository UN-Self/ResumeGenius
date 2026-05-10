# Agent 提示词与工具披露逻辑改造 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 Agent 无限循环问题 — 技能工具 description 不再泄露实际内容，system prompt 明确技能调用协议

**Architecture:** 精简 skill.yaml 的 description 为触发条件，system prompt 新增技能调用协议段落并精简工作流程。不改动 Go 逻辑代码。

**Tech Stack:** Go, YAML, 嵌入式文件系统 (embed.FS)

---

## 文件清单

| 文件 | 操作 | 职责 |
|---|---|---|
| `backend/internal/modules/agent/skills/resume-design/skill.yaml` | 修改 | description 精简为触发条件 |
| `backend/internal/modules/agent/skills/resume-interview/skill.yaml` | 修改 | description 精简为触发条件 |
| `backend/internal/modules/agent/service.go` | 修改 | system prompt 常量 |
| `backend/internal/modules/agent/tool_executor_test.go` | 修改 | 更新断言以匹配新 description |

---

### Task 1: 修改 resume-design/skill.yaml

**Files:**
- Modify: `backend/internal/modules/agent/skills/resume-design/skill.yaml`

- [ ] **Step 1: 精简 description，移除 CSS 摘要**

将 `resume-design/skill.yaml` 的 description 从包含完整 CSS 摘要的长文本改为简短触发条件：

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
```

- [ ] **Step 2: 验证 YAML 语法**

Run: `cd backend && go test ./internal/modules/agent/ -run TestSkillLoader_LoadSkill_Design -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/agent/skills/resume-design/skill.yaml
git commit -m "refactor: 精简 resume-design skill description 为触发条件，移除 CSS 摘要"
```

---

### Task 2: 修改 resume-interview/skill.yaml

**Files:**
- Modify: `backend/internal/modules/agent/skills/resume-interview/skill.yaml`

- [ ] **Step 1: 精简 description**

将 `resume-interview/skill.yaml` 的 description 改为简短触发条件：

```yaml
name: resume-interview
description: |
  目标岗位的面试官视角优化。当用户明确目标岗位时使用。
trigger: 用户提到目标岗位或需要面试相关建议时

usage: |
  1. 调用 get_skill_reference(skill_name="resume-interview", reference_name="<岗位名>") 获取面经
  2. 基于面经中的面试官关注点修改简历

tools:
  - name: get_skill_reference
    description: 获取指定岗位的面经内容
    params:
      - name: skill_name
        type: string
        description: 固定传 "resume-interview"
      - name: reference_name
        type: string
        description: 岗位名，如 "test-engineer"

references:
  - name: test-engineer
    description: 测试工程师岗位面经
```

- [ ] **Step 2: 验证 YAML 语法**

Run: `cd backend && go test ./internal/modules/agent/ -run TestSkillLoader_LoadSkill_Existing -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/agent/skills/resume-interview/skill.yaml
git commit -m "refactor: 精简 resume-interview skill description 为触发条件"
```

---

### Task 3: 修改 system prompt

**Files:**
- Modify: `backend/internal/modules/agent/service.go:106-154`

- [ ] **Step 1: 替换 systemPromptV2 常量**

将 `service.go` 中的 `systemPromptV2` 常量替换为：

```go
const systemPromptV2 = `你是简历编辑专家。你可以像编辑代码一样精确编辑简历 HTML。

## 核心工具
- get_draft: 读取当前简历 HTML（可选 CSS selector 指定范围）
- apply_edits: 提交搜索替换操作修改简历（old_string 必须精确匹配）
- search_assets: 搜索用户资料库（旧简历、Git 摘要、笔记等）

## 核心铁律
所有简历内容必须以用户上传的资料为唯一事实来源。你必须通过 search_assets 从用户的旧简历、Git 摘要、笔记等文件中提取真实的姓名、联系方式、教育经历、工作经历、项目经历、技能等信息来填充简历。
只有在反复搜索后确实找不到某项关键信息时，才可以在最终回复中列出缺失项，提醒用户上传相关文件或手动补充。禁止在任何情况下凭空编造个人身份信息或职业经历。

## 工作流程
1. get_draft 查看当前简历
2. search_assets 搜索用户资料，提取真实信息
3. 根据用户需求调用对应技能工具，按返回的 usage 指引操作
4. 用 apply_edits 提交修改
5. 完成后总结修改内容

## 技能调用协议
调用技能工具后，按返回的 usage 指引操作。不要跳过指引中的步骤。

## 编辑原则
- apply_edits 是搜索替换，不是追加：old_string 必须匹配要被替换的已有内容，new_string 是替换后的内容
- 绝对禁止把整份简历作为 new_string 写入而不匹配任何 old_string，这会导致内容重复
- 每次只修改需要变化的部分，不要重写整个简历
- old_string 必须精确匹配，不匹配则修改会失败
- 失败时读取当前 HTML 找到正确内容后重试
- 保持 HTML 结构完整，确保渲染正确
- 内容简洁专业，突出关键信息

## A4 简历硬约束
- 当前产品编辑的是简历，不是网页、落地页、作品集、仪表盘或海报
- 默认目标是一页 A4：210mm x 297mm；如果内容过多，先压缩文案、字号、行距和间距，不要扩展成多页视觉稿
- 使用常见招聘简历样式：白色或浅色纸面、深色正文、最多一个克制强调色、清晰分区标题、紧凑项目符号、信息密度高但可读
- 正文字号保持在 13-15px 左右，姓名标题不超过 24px，分区标题 14-16px；不要使用超大 hero 字体
- 字体必须支持中文渲染；禁止使用仅含拉丁字符的字体（如 Inter、Roboto 单独指定）；中文内容必须落在含有 "Noto Sans CJK SC"、"Microsoft YaHei"、"PingFang SC" 或系统 sans-serif 回退的字体栈中
- 技能列表必须可换行、可读，禁止做成长串不换行的技能胶囊或大块色卡
- 禁止使用 landing page、hero、dashboard、bento/card grid、glassmorphism、aurora、3D、霓虹、复杂渐变、大面积紫蓝/粉色背景、纹理背景、动画、发光、厚重阴影、过度圆角和装饰图形
- 如果用户说"太花"、"太炫"、"过头"、"不像简历"，优先移除视觉特效，恢复常规专业简历样式

## 回复规范
- 不要使用任何 emoji 或特殊符号装饰
`
```

- [ ] **Step 2: 编译检查**

Run: `cd backend && go build ./cmd/server/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/agent/service.go
git commit -m "refactor: 精简 system prompt 工作流程，新增技能调用协议"
```

---

### Task 4: 更新测试断言

**Files:**
- Modify: `backend/internal/modules/agent/tool_executor_test.go:178-195`

- [ ] **Step 1: 更新 TestSkillTool_ResumeDesign_DescriptionContainsCSSGuidelines**

当前测试断言 description 包含 CSS 内容（linear-gradient、backdrop-filter、Noto Sans CJK SC），改造后 description 不再包含这些内容。将测试改为验证 description 是简短触发条件，CSS 内容在 reference 中：

```go
func TestSkillTool_ResumeDesign_DescriptionIsConcise(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	executor := NewAgentToolExecutor(nil, loader)

	ctx := WithSessionID(context.Background(), 300)
	result, err := executor.Execute(ctx, "resume-design", nil)
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &data))
	desc, ok := data["description"].(string)
	require.True(t, ok)

	// description 应该是简短触发条件，不包含 CSS 摘要
	assert.NotContains(t, desc, "linear-gradient", "description should not contain CSS guidelines")
	assert.NotContains(t, desc, "backdrop-filter", "description should not contain CSS guidelines")
	assert.NotContains(t, desc, "Noto Sans CJK SC", "description should not contain CSS guidelines")
	assert.Less(t, len(desc), 100, "description should be concise (< 100 chars)")

	// CSS 内容应在 reference 中
	ref, err := loader.GetReference("resume-design", "a4-guidelines")
	require.NoError(t, err)
	assert.Contains(t, ref.Content, "linear-gradient", "reference should contain CSS guidelines")
	assert.Contains(t, ref.Content, "Noto Sans CJK SC", "reference should contain CSS guidelines")
}
```

- [ ] **Step 2: 运行测试**

Run: `cd backend && go test ./internal/modules/agent/ -v`
Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/modules/agent/tool_executor_test.go
git commit -m "test: 更新 skill description 断言以匹配精简后的触发条件"
```

---

### Task 5: 端到端验证

- [ ] **Step 1: 运行全部 agent 测试**

Run: `cd backend && go test ./internal/modules/agent/... -v`
Expected: 全部 PASS，无失败

- [ ] **Step 2: 启动服务手动验证**

Run: `cd backend && go run cmd/server/main.go`
Expected: 服务启动正常，无 panic

- [ ] **Step 3: 构建检查**

Run: `cd backend && go build ./cmd/server/...`
Expected: 无错误
