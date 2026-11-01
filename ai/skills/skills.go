package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
)

type SkillT struct {
	Name                   string            `yaml:"name"`
	Description            string            `yaml:"description"`
	License                string            `yaml:"license"`
	Compatibility          string            `yaml:"compatibility"`
	Meta                   map[string]string `yaml:"metadata"`
	ToolsRaw               string            `yaml:"allowed-tools"`
	Tools                  []*ToolsT         `yaml:"-"`
	Prompt                 string            `yaml:"-"`
	Directory              string            `yaml:"-"`
	UserInvocable          *bool             `yaml:"user-invocable"`
	DisableModelInvocation bool              `yaml:"disable-model-invocation"`

	Variables []types.InputBoxWTVariables `yaml:"variables"`
}

type ToolsT struct {
	Name       string
	Parameters string
}

type Skills []*SkillT

func (skill *SkillT) IsUserInvocable() bool {
	return skill != nil && (skill.UserInvocable == nil || *skill.UserInvocable)
}

func (skill *SkillT) IsModelInvocable() bool {
	return skill != nil && !skill.DisableModelInvocation
}

func (skills Skills) UserInvocable() Skills {
	filtered := make(Skills, 0, len(skills))
	for _, skill := range skills {
		if skill.IsUserInvocable() {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

func (skills Skills) ModelInvocable() Skills {
	filtered := make(Skills, 0, len(skills))
	for _, skill := range skills {
		if skill.IsModelInvocable() {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

func (skills Skills) FromName(name string) *SkillT {
	for _, skill := range skills {
		if skill.Name == name {
			return skill
		}
	}
	return nil
}

func ParseSelection(response string, available Skills) (Skills, error) {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
			response = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var names []string
	if err := json.Unmarshal([]byte(response), &names); err != nil {
		return nil, fmt.Errorf("decode skill selection: %w", err)
	}

	selected := make(Skills, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if skill := available.FromName(name); skill != nil {
			selected = append(selected, skill)
		}
	}
	return selected, nil
}

func ReadSkills() (skills Skills) {
	files := file.GetConfigGlob("agent-skills/*/SKILL.md")
	files = append(files, file.GetConfigGlob("skills/*/SKILL.md")...)

	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Printf("[error] Cannot open skill file '%s': %v", files[i], err)
			continue
		}

		skill, err := parseSkill(f)
		_ = f.Close()
		if err != nil {
			log.Printf("[error] Cannot parse skill file '%s': %v", files[i], err)
			continue
		}
		skill.Directory = filepath.Dir(files[i])
		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return
}

func parseSkill(reader io.Reader) (*SkillT, error) {
	skill := new(SkillT)
	body, err := frontmatter.Parse(reader, skill)
	if err != nil {
		return nil, err
	}
	skill.Prompt = string(body)
	if err := parseSkillTools(skill); err != nil {
		return nil, err
	}
	return skill, nil
}

var rxSkillToolParams = regexp.MustCompile(`(([-a-zA-Z0-9]+)|([-a-zA-Z0-9]+)\((.*?)\))`)

func parseSkillTools(skill *SkillT) error {
	if skill.ToolsRaw == "" {
		return nil
	}

	tools, err := ParseTools(skill.ToolsRaw)
	if err != nil {
		return fmt.Errorf("error in skill %s: %w", skill.Name, err)
	}

	skill.Tools = tools
	return nil
}

func ParseTools(toolsRaw string) ([]*ToolsT, error) {
	var tools []*ToolsT

	toolsRaw = strings.TrimSpace(toolsRaw)
	if toolsRaw == "" {
		return tools, nil
	}

	split := strings.Split(toolsRaw, " ")

	for i := range split {
		match := rxSkillToolParams.FindAllStringSubmatch(split[i], -1)
		tool := new(ToolsT)
		switch len(match) {
		case 2:
			tool.Parameters = match[1][0]
			fallthrough
		case 1:
			tool.Name = match[0][0]
		default:
			//log.Printf("[error] Cannot parse tool '%s'", split[i])
			return nil, fmt.Errorf("cannot parse tool '%s'", split[i])
			//continue
		}
		tools = append(tools, tool)
	}

	return tools, nil
}
