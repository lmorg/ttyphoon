package prompts

import (
	_ "embed"
	"fmt"
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

//go:embed explain_cmd.md
var _PROMPT_EXPLAIN_CMD string

//go:embed explain_doc.md
var _PROMPT_EXPLAIN_DOC string

//go:embed ask.md
var _PROMPT_ASK string

//go:embed title.md
var _PROMPT_TITLE string

var rxSkillFunction = regexp.MustCompile(`^/[-a-zA-Z0-9]+($|\s)`)

func GetExplainCmd(agt *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_EXPLAIN_CMD, promptVars(agt, userPrompt))
}

func GetExplainDoc(agt *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_EXPLAIN_DOC, promptVars(agt, userPrompt))
}

func GetAsk(agt *agent.Agent, userPrompt string) string {
	fn := rxSkillFunction.FindString(userPrompt)
	if fn == "" {
		return os.Expand(_PROMPT_ASK, promptVars(agt, userPrompt))
	}

	agt.Meta.Function = strings.TrimRight(fn[1:], " ")
	skill := skills.ReadSkills().FromFunctionName(agt.Meta.Function)
	if skill == nil {
		return os.Expand(_PROMPT_ASK, promptVars(agt, userPrompt))
	}

	err := agt.StartTools(skill.Tools)
	if err != nil {
		agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
	return os.Expand(skill.Prompt+"\n$SYSTEM_PROMPT\n# User Prompt\n\n$USER_PROMPT\n", promptVars(agt, userPrompt))
}

func GetTitle(agt *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_TITLE, promptVars(agt, userPrompt))
}

func promptVars(agt *agent.Agent, userPrompt string) func(string) string {
	return func(s string) string {
		switch s {
		case "SYSTEM_PROMPT":
			return os.Expand(_PROMPT_SYSTEM, promptVars(agt, userPrompt))
		case "MAX_ITERATIONS":
			return strconv.Itoa(agt.MaxIterations())
		case "HOST_OS":
			return runtime.GOOS
		case "HOST_CPU":
			return runtime.GOARCH
		case "HISTORY":
			return agt.History.String()
		case "USER_PROMPT":
			return userPrompt
		case "COMMAND_LINE":
			return agt.Meta.CmdLine
		case "COMMAND_OUTPUT":
			return agt.Meta.OutputBlock
		default:
			if len(agt.Meta.Variables) == 0 {
				return "$" + s
			}
			val, ok := agt.Meta.Variables[s]
			if !ok {
				return "$" + s
			}
			return fmt.Sprintf("%v", val)
		}
	}
}
