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

// --- BuildSkillListing tests ---

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

// --- A4 template reference ---

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

	for _, ref := range result.References {
		if ref.Name == "a4-template" {
			assert.Contains(t, ref.Content, "794px", "should contain canvas width")
			assert.Contains(t, ref.Content, "1013px", "should contain content area height")
			assert.Contains(t, ref.Content, "1500-2000", "should contain capacity estimate")
			return
		}
	}
	t.Fatal("a4-template reference not found")
}
