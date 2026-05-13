package agent

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed skills/*.md skills/*/*.yaml skills/*/*/*.yaml
var skillsFS embed.FS

// SkillDescriptor represents a Layer 2 skill description document (skill.yaml).
type SkillDescriptor struct {
	Name        string              `yaml:"name"        json:"name"`
	Description string              `yaml:"description" json:"description"`
	Trigger     string              `yaml:"trigger"     json:"trigger,omitempty"`
	Usage       string              `yaml:"usage"       json:"usage"`
	Tools       []ToolDefinition    `yaml:"tools"       json:"tools"`
	References  []ReferenceMetadata `yaml:"references"  json:"references"`
}

// ToolDefinition describes a tool available within a skill.
type ToolDefinition struct {
	Name        string  `yaml:"name"        json:"name"`
	Description string  `yaml:"description" json:"description"`
	Params      []Param `yaml:"params"      json:"params"`
	Usage       string  `yaml:"usage"       json:"usage"`
}

// Param describes a tool parameter.
type Param struct {
	Name        string `yaml:"name"        json:"name"`
	Type        string `yaml:"type"        json:"type"`
	Description string `yaml:"description" json:"description"`
}

// ReferenceMetadata describes a reference's metadata within a skill descriptor.
type ReferenceMetadata struct {
	Name        string `yaml:"name"        json:"name"`
	Description string `yaml:"description" json:"description"`
}

// ReferenceContent represents a Layer 3 reference content file.
type ReferenceContent struct {
	Name    string `yaml:"name"    json:"name"`
	Content string `yaml:"content" json:"content"`
}

// SkillLoader loads and provides access to skill descriptors and references.
type SkillLoader struct {
	skills     map[string]*SkillDescriptor
	references map[string]map[string]*ReferenceContent // skillName -> refName -> content
}

// NewSkillLoader creates a SkillLoader by walking the embedded skills directory.
func NewSkillLoader() (*SkillLoader, error) {
	loader := &SkillLoader{
		skills:     make(map[string]*SkillDescriptor),
		references: make(map[string]map[string]*ReferenceContent),
	}

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

		// Distinguish skill descriptors from reference content by filename.
		if strings.HasSuffix(path, "/skill.yaml") {
			var desc SkillDescriptor
			if err := yaml.Unmarshal(data, &desc); err != nil {
				return fmt.Errorf("parse skill descriptor %s: %w", path, err)
			}
			loader.skills[desc.Name] = &desc
		} else if strings.Contains(path, "/references/") {
			var ref ReferenceContent
			if err := yaml.Unmarshal(data, &ref); err != nil {
				return fmt.Errorf("parse reference %s: %w", path, err)
			}
			// Extract skill name from path: skills/<skillName>/references/<file>.yaml
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				skillName := parts[1]
				if loader.references[skillName] == nil {
					loader.references[skillName] = make(map[string]*ReferenceContent)
				}
				loader.references[skillName][ref.Name] = &ref
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}

	// Log loaded skills summary
	names := make([]string, 0, len(loader.skills))
	for name := range loader.skills {
		names = append(names, name)
	}
	debugLog("skills", "技能加载完成，共 %d 个技能: %v", len(loader.skills), names)

	for name, refs := range loader.references {
		debugLog("skills", "技能 %s 加载了 %d 个参考文档", name, len(refs))
	}

	return loader, nil
}

// Skills returns all loaded skill descriptors.
func (l *SkillLoader) Skills() []*SkillDescriptor {
	result := make([]*SkillDescriptor, 0, len(l.skills))
	for _, s := range l.skills {
		result = append(result, s)
	}
	return result
}

// HasSkill reports whether a skill with the given name exists.
func (l *SkillLoader) HasSkill(name string) bool {
	_, ok := l.skills[name]
	return ok
}

// LoadSkill returns the descriptor for the named skill.
func (l *SkillLoader) LoadSkill(name string) (*SkillDescriptor, error) {
	desc, ok := l.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	return desc, nil
}

// GetReference returns the content of a specific reference within a skill.
func (l *SkillLoader) GetReference(skillName, refName string) (*ReferenceContent, error) {
	refs, ok := l.references[skillName]
	if !ok {
		// Check if the skill exists at all.
		if _, skillExists := l.skills[skillName]; !skillExists {
			return nil, fmt.Errorf("skill not found: %s", skillName)
		}
		return nil, fmt.Errorf("reference '%s' not found in skill '%s'", refName, skillName)
	}

	ref, ok := refs[refName]
	if !ok {
		// List available references for helpful error message.
		available := make([]string, 0, len(refs))
		for name := range refs {
			available = append(available, name)
		}
		return nil, fmt.Errorf("reference '%s' not found in skill '%s'. Available references: %v", refName, skillName, available)
	}

	return ref, nil
}

// SkillWithReferences is a skill descriptor with all reference content inlined.
type SkillWithReferences struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Trigger     string              `json:"trigger,omitempty"`
	Usage       string              `json:"usage"`
	References  []ReferenceWithName  `json:"references"`
}

// ReferenceWithName pairs a reference name with its full content.
type ReferenceWithName struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// LoadSkillWithReferences returns the skill descriptor with all reference content inlined.
func (l *SkillLoader) LoadSkillWithReferences(name string) (*SkillWithReferences, error) {
	desc, ok := l.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	refs := make([]ReferenceWithName, 0, len(desc.References))
	for _, refMeta := range desc.References {
		if refContent, ok := l.references[name][refMeta.Name]; ok {
			refs = append(refs, ReferenceWithName{
				Name:    refMeta.Name,
				Content: refContent.Content,
			})
		}
	}

	return &SkillWithReferences{
		Name:        desc.Name,
		Description: desc.Description,
		Trigger:     desc.Trigger,
		Usage:       desc.Usage,
		References:  refs,
	}, nil
}
