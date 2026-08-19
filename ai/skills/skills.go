package skills

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
)

type SkillT struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Meta          map[string]string `yaml:"metadata"`
	ToolsRaw      string            `yaml:"allowed-tools"`
	Tools         []*ToolsT         `yaml:"-"`
	Prompt        string            `yaml:"-"`
	FunctionName  string            `yaml:"function-name"`

	Variables []types.InputBoxWTVariables `yaml:"variables"`
}

type ToolsT struct {
	Name       string
	Parameters string
}

type Skills []*SkillT

func (skills Skills) FromFunctionName(fn string) *SkillT {
	for _, skill := range skills {
		if skill.FunctionName == fn {
			return skill
		}
	}
	return nil
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

		skill := new(SkillT)
		b, err := frontmatter.Parse(f, skill)
		if err != nil {
			log.Printf("[error] Cannot parse skill file '%s': %v", files[i], err)
			continue
		}
		skill.Prompt = string(b)
		if skill.FunctionName == "" {
			skill.FunctionName = skill.Meta["function-name"]
		}
		parseSkillTools(skill)
		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].FunctionName < skills[j].FunctionName
	})

	return
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
