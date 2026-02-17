package prompts

import (
	_ "embed"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/types"
)

//go:embed system.md
var _PROMPT_SYSTEM string

//go:embed explain.md
var _PROMPT_EXPLAIN string

//go:embed ask.md
var _PROMPT_ASK string

var rxSkillFunction = regexp.MustCompile(`^/[-a-zA-Z0-9]+($|\s)`)

func GetExplain(meta *agent.Meta, userPrompt string) string {
	return os.Expand(_PROMPT_EXPLAIN, promptVars(meta, userPrompt))
}

func GetAsk(meta *agent.Meta, userPrompt string) string {
	fn := rxSkillFunction.FindString(userPrompt)
	if fn == "" {
		return os.Expand(_PROMPT_ASK, promptVars(meta, userPrompt))
	}

	fn = strings.TrimRight(fn[1:], " ")
	skill := skills.ReadSkills().FromFunctionName(fn)
	if skill == nil {
		return os.Expand(_PROMPT_ASK, promptVars(meta, userPrompt))
	}

	err := meta.SkillStartTools(skill)
	if err != nil {
		meta.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
	return os.Expand(skill.Prompt+"\n$SYSTEM_PROMPT\n# User Prompt\n\n$USER_PROMPT\n", promptVars(meta, userPrompt))
}

func promptVars(meta *agent.Meta, userPrompt string) func(string) string {
	return func(s string) string {
		switch s {
		case "SYSTEM_PROMPT":
			return os.Expand(_PROMPT_SYSTEM, promptVars(meta, userPrompt))
		case "MAX_ITERATIONS":
			return strconv.Itoa(meta.MaxIterations())
		case "HOST_OS":
			return runtime.GOOS
		case "HOST_CPU":
			return runtime.GOARCH
		case "HISTORY":
			return meta.History.String()
		case "USER_PROMPT":
			return userPrompt
		case "COMMAND_LINE":
			return meta.CmdLine
		case "COMMAND_OUTPUT":
			return meta.OutputBlock
		default:
			return "$" + s
		}
	}
}
