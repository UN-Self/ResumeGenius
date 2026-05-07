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
	// When keyword and category are both empty, returns SkillSummary
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
