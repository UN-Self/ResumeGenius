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

	results := loader.Search("测试工程师", "", 0)
	require.NotEmpty(t, results, "should find test-resume skill")
	assert.Equal(t, "test-resume", results[0].Name)
	assert.NotEmpty(t, results[0].Content, "should include full content")
}

func TestSkillLoader_SearchByKeyword_WithLimit(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("测试工程师", "", 1)
	require.Len(t, results, 1)
}

func TestSkillLoader_SearchByCategory(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("", "test", 0)
	require.NotEmpty(t, results, "should find skills in test category")
}

func TestSkillLoader_SearchNoMatch(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("nonexistent_keyword_xyz", "", 0)
	assert.Empty(t, results)
}

func TestSkillLoader_SearchAllReturnsFullContent(t *testing.T) {
	loader, err := NewSkillLoader()
	require.NoError(t, err)

	results := loader.Search("", "", 0)
	require.NotEmpty(t, results)
	assert.NotEmpty(t, results[0].Content, "should include full content when no filter")
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
