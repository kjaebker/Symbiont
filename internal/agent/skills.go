package agent

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed skills
var skillsFS embed.FS

// SkillMeta holds the parsed YAML frontmatter of a SKILL.md file.
type SkillMeta struct {
	Name        string `yaml:"name"        json:"name"`
	Description string `yaml:"description" json:"description"`
}

// Skill is a fully loaded skill: metadata plus the body text.
type Skill struct {
	SkillMeta
	Body string `json:"body"`
}

// SkillFile returns the complete file content (frontmatter + body) suitable
// for writing to ~/.claude/skills/symbiont/<name>.md.
func (s Skill) SkillFile() string {
	return "---\nname: " + s.Name + "\ndescription: " + s.Description + "\n---\n\n" + s.Body + "\n"
}

// LoadSkills reads all embedded SKILL.md files and returns them in directory
// order (alphabetical by skill folder name).
func LoadSkills() ([]Skill, error) {
	entries, err := skillsFS.ReadDir("skills")
	if err != nil {
		return nil, fmt.Errorf("reading skills dir: %w", err)
	}
	skills := make([]Skill, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := "skills/" + e.Name() + "/SKILL.md"
		data, err := fs.ReadFile(skillsFS, path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		skill, err := parseSkill(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// GetSkill returns the named skill or an error if not found.
func GetSkill(name string) (*Skill, error) {
	skills, err := LoadSkills()
	if err != nil {
		return nil, err
	}
	for i := range skills {
		if skills[i].Name == name {
			return &skills[i], nil
		}
	}
	return nil, fmt.Errorf("skill %q not found", name)
}

// parseSkill splits the YAML frontmatter block from body text.
// Expected format:
//
//	---
//	name: …
//	description: …
//	---
//	<body>
func parseSkill(data []byte) (Skill, error) {
	const fence = "---"
	s := string(data)
	if !strings.HasPrefix(s, fence+"\n") {
		return Skill{}, fmt.Errorf("missing opening frontmatter fence")
	}
	rest := s[len(fence)+1:]
	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return Skill{}, fmt.Errorf("missing closing frontmatter fence")
	}
	fm := rest[:end]
	body := strings.TrimSpace(rest[end+1+len(fence):])

	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return Skill{}, fmt.Errorf("parsing frontmatter yaml: %w", err)
	}
	if meta.Name == "" {
		return Skill{}, fmt.Errorf("frontmatter missing required field: name")
	}
	return Skill{SkillMeta: meta, Body: body}, nil
}
