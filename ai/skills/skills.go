package skills

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/lmorg/ttyphoon/utils/file"
)

type SkillT struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Meta          map[string]string `yaml:"metadata"`
	ToolsRaw      string            `yaml:"allowed-tools"`
	Tools         []*skillToolT     `yaml:"-"`
	Prompt        string            `yaml:"-"`
	FunctionName  string            `yaml:"function-name"`
}

type skillToolT struct {
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

func ReadSkills() Skills {
	var (
		files  = file.GetConfigGlob("agent-skills/*/SKILL.md")
		skills []*SkillT
	)

	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Printf("Cannot open skill file '%s': %v", files[i], err)
			continue
		}

		skill := new(SkillT)
		b, err := frontmatter.Parse(f, skill)
		if err != nil {
			log.Printf("Cannot parse skill file '%s': %v", files[i], err)
			continue
		}
		skill.Prompt = string(b)
		if skill.FunctionName == "" {
			skill.FunctionName = skill.Meta["function-name"]
		}
		parseSkillTools(skill)
		skills = append(skills, skill)
	}

	return skills
}

var rxSkillToolParams = regexp.MustCompile(`(([-a-zA-Z0-9]+)|([-a-zA-Z0-9]+)\((.*?)\))`)

func parseSkillTools(skill *SkillT) {
	if skill.ToolsRaw == "" {
		return
	}

	tools := strings.Split(skill.ToolsRaw, " ")
	for i := range tools {

		match := rxSkillToolParams.FindAllStringSubmatch(tools[i], -1)
		tool := new(skillToolT)
		switch len(match) {
		case 2:
			tool.Parameters = match[1][0]
			fallthrough
		case 1:
			tool.Name = match[0][0]
		default:
			log.Printf("Cannot parse skill '%s' tool '%s'", skill.Name, tools[i])
			continue
		}
		skill.Tools = append(skill.Tools, tool)
	}
}
