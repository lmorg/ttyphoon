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

func GetExplain(agent *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_EXPLAIN, promptVars(agent, userPrompt))
}

func GetAsk(agent *agent.Agent, userPrompt string) string {
	fn := rxSkillFunction.FindString(userPrompt)
	if fn == "" {
		return os.Expand(_PROMPT_ASK, promptVars(agent, userPrompt))
	}

	fn = strings.TrimRight(fn[1:], " ")
	skill := skills.ReadSkills().FromFunctionName(fn)
	if skill == nil {
		return os.Expand(_PROMPT_ASK, promptVars(agent, userPrompt))
	}

	err := agent.SkillStartTools(skill)
	if err != nil {
		agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
	return os.Expand(skill.Prompt+"\n$SYSTEM_PROMPT\n# User Prompt\n\n$USER_PROMPT\n", promptVars(agent, userPrompt))
}

func promptVars(agent *agent.Agent, userPrompt string) func(string) string {
	return func(s string) string {
		switch s {
		case "SYSTEM_PROMPT":
			return os.Expand(_PROMPT_SYSTEM, promptVars(agent, userPrompt))
		case "MAX_ITERATIONS":
			return strconv.Itoa(agent.MaxIterations())
		case "HOST_OS":
			return runtime.GOOS
		case "HOST_CPU":
			return runtime.GOARCH
		case "HISTORY":
			return agent.History.String()
		case "USER_PROMPT":
			return userPrompt
		case "COMMAND_LINE":
			return agent.Meta.CmdLine
		case "COMMAND_OUTPUT":
			return agent.Meta.OutputBlock
		default:
			return "$" + s
		}
	}
}
