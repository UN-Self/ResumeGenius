package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_ContainsAllSections(t *testing.T) {
	sections := DefaultPromptSections("", "")
	prompt := BuildSystemPrompt(sections)

	assert.Contains(t, prompt, "简历编辑专家")
	assert.Contains(t, prompt, "核心工具")
	assert.Contains(t, prompt, "核心铁律")
	assert.Contains(t, prompt, "编辑原则")
	assert.Contains(t, prompt, "A4 简历硬约束")
	assert.Contains(t, prompt, "循环控制规则")
	assert.Contains(t, prompt, "回复规范")
}

func TestBuildSystemPrompt_IncludesAssets(t *testing.T) {
	assetInfo := "\n## 用户已上传 2 个文件\n"
	sections := DefaultPromptSections(assetInfo, "")
	prompt := BuildSystemPrompt(sections)

	assert.Contains(t, prompt, "用户已上传 2 个文件")
}

func TestBuildSystemPrompt_IncludesSkills(t *testing.T) {
	skillInfo := "- resume-design: A4 简历设计规范\n"
	sections := DefaultPromptSections("", skillInfo)
	prompt := BuildSystemPrompt(sections)

	assert.Contains(t, prompt, "resume-design")
}

func TestBuildSystemPrompt_NoAntiLoopInstruction(t *testing.T) {
	sections := DefaultPromptSections("", "")
	prompt := BuildSystemPrompt(sections)

	assert.NotContains(t, prompt, "失败时读取当前 HTML 找到正确内容后重试")
	assert.Contains(t, prompt, "更短的唯一片段")
}

func TestBuildSystemPrompt_FlowRulesIncludeCallLimit(t *testing.T) {
	sections := DefaultPromptSections("", "")
	prompt := BuildSystemPrompt(sections)

	assert.Contains(t, prompt, "最多调用 2 次")
	assert.Contains(t, prompt, "apply_edits")
}

func TestBuildSystemPrompt_A4ConstraintsProtectSystemShell(t *testing.T) {
	sections := DefaultPromptSections("", "")
	prompt := BuildSystemPrompt(sections)

	assert.Contains(t, prompt, "系统已经提供 A4 纸张")
	assert.Contains(t, prompt, ".resume-document")
	assert.Contains(t, prompt, "不要写 width:210mm")
	assert.Contains(t, prompt, "不能给整页或根容器加约束边框")
}
