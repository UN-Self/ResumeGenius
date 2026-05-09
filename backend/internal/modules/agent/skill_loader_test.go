package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LoadSkill tests ---

func TestSkillLoader_LoadSkill_Existing(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	desc, err := loader.LoadSkill("resume-interview")
	require.NoError(t, err)
	assert.Equal(t, "resume-interview", desc.Name)
	assert.NotEmpty(t, desc.Description)
	assert.NotEmpty(t, desc.Usage)
	assert.NotEmpty(t, desc.Tools)
	assert.NotEmpty(t, desc.References)
}

func TestSkillLoader_LoadSkill_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	_, err = loader.LoadSkill("nonexistent-skill")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill not found")
}

func TestSkillLoader_LoadSkill_Design(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	desc, err := loader.LoadSkill("resume-design")
	require.NoError(t, err)
	assert.Equal(t, "resume-design", desc.Name)
	assert.NotEmpty(t, desc.References)
	assert.Equal(t, "a4-guidelines", desc.References[0].Name)
}

// --- HasSkill tests ---

func TestSkillLoader_HasSkill_Existing(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	assert.True(t, loader.HasSkill("resume-interview"))
	assert.True(t, loader.HasSkill("resume-design"))
}

func TestSkillLoader_HasSkill_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)
	assert.False(t, loader.HasSkill("nonexistent"))
}

// --- GetReference tests ---

func TestSkillLoader_GetReference_Existing(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	ref, err := loader.GetReference("resume-interview", "test-engineer")
	require.NoError(t, err)
	assert.Equal(t, "test-engineer", ref.Name)
	assert.NotEmpty(t, ref.Content)
	assert.Contains(t, ref.Content, "测试工程师")
}

func TestSkillLoader_GetReference_NotFound(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	_, err = loader.GetReference("resume-interview", "nonexistent-ref")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "test-engineer") // should list available references
}

func TestSkillLoader_GetReference_WrongSkill(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	_, err = loader.GetReference("nonexistent-skill", "test-engineer")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill not found")
}

func TestSkillLoader_GetReference_DesignSkill(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	ref, err := loader.GetReference("resume-design", "a4-guidelines")
	require.NoError(t, err)
	assert.Equal(t, "a4-guidelines", ref.Name)
	assert.Contains(t, ref.Content, "A4 单页简历设计规范")
}

// --- Skills listing ---

func TestSkillLoader_Skills_ReturnsAllDescriptors(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	skills := loader.Skills()
	require.Len(t, skills, 2)

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
		assert.NotEmpty(t, s.Description)
		assert.NotEmpty(t, s.References)
	}
	assert.True(t, names["resume-interview"])
	assert.True(t, names["resume-design"])
}
