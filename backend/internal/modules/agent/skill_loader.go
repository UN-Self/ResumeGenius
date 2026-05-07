package agent

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed skills/*.md skills/*/*.yaml
var skillsFS embed.FS

// SkillFile represents a single skill definition loaded from a YAML file.
type SkillFile struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Metadata    SkillMetadata    `yaml:"metadata"`
	References  []SkillReference `yaml:"references"`
	Content     string           `yaml:"content"`
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

// Search returns matching SkillFile objects based on keyword and/or category.
// When both keyword and category are empty, all skills are returned in full.
//
//   - keyword: case-insensitive contains match against Name, Description, and Keywords
//   - category: case-insensitive exact match against Metadata.Category
//   - limit: max results returned (0 = no limit)
func (l *SkillLoader) Search(keyword, category string, limit int) []SkillFile {
	if len(l.skills) == 0 {
		return nil
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

	return matched
}

// matchKeyword checks if the keyword matches any of the skill's keywords, name, or description.
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
