package prompts

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
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

func GetExplainCmdMessages(agt *agent.Agent, userPrompt string) []*schema.Message {
	return []*schema.Message{
		schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
		schema.UserMessage(strings.TrimSpace(os.Expand(_PROMPT_EXPLAIN_CMD, promptVars(agt, userPrompt)))),
	}
}

func GetExplainDoc(agt *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_EXPLAIN_DOC, promptVars(agt, userPrompt))
}

func GetExplainDocMessages(agt *agent.Agent, userPrompt string) []*schema.Message {
	return []*schema.Message{
		schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
		schema.UserMessage(strings.TrimSpace(os.Expand(_PROMPT_EXPLAIN_DOC, promptVars(agt, userPrompt)))),
	}
}

func GetExplainDocMessagesWithImages(agt *agent.Agent, userPrompt string, images []aitypes.ImageAttachment) []*schema.Message {
	text := strings.TrimSpace(os.Expand(_PROMPT_EXPLAIN_DOC, promptVars(agt, userPrompt)))
	return []*schema.Message{
		schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
		buildUserMessageWithImages(text, images),
	}
}

// buildUserMessageWithImages attaches images as multimodal parts so the same message
// works unmodified across Claude, OpenAI, and Ollama chat model backends.
func buildUserMessageWithImages(text string, images []aitypes.ImageAttachment) *schema.Message {
	if len(images) == 0 {
		return schema.UserMessage(text)
	}

	parts := make([]schema.MessageInputPart, 0, len(images)+1)
	if text != "" {
		parts = append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeText, Text: text})
	}
	for _, img := range images {
		img := img
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeImageURL,
			Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &img.Base64,
					MIMEType:   img.MIMEType,
				},
			},
		})
	}

	return &schema.Message{Role: schema.User, UserInputMultiContent: parts}
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

func GetAskMessages(agt *agent.Agent, userPrompt string) []*schema.Message {
	fn := rxSkillFunction.FindString(userPrompt)
	if fn == "" {
		return []*schema.Message{
			schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
			schema.UserMessage(strings.TrimSpace(os.Expand(_PROMPT_ASK, promptVars(agt, userPrompt)))),
		}
	}

	agt.Meta.Function = strings.TrimRight(fn[1:], " ")
	skill := skills.ReadSkills().FromFunctionName(agt.Meta.Function)
	if skill == nil {
		return []*schema.Message{
			schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
			schema.UserMessage(strings.TrimSpace(os.Expand(_PROMPT_ASK, promptVars(agt, userPrompt)))),
		}
	}

	err := agt.StartTools(skill.Tools)
	if err != nil {
		agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}

	user := strings.TrimSpace(os.Expand(skill.Prompt+"\n# User Prompt\n\n$USER_PROMPT\n", promptVars(agt, userPrompt)))
	return []*schema.Message{
		schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
		schema.UserMessage(user),
	}
}

func GetTitle(agt *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_TITLE, promptVars(agt, userPrompt))
}

func GetTitleMessages(agt *agent.Agent, userPrompt string) []*schema.Message {
	return []*schema.Message{
		schema.UserMessage(strings.TrimSpace(os.Expand(_PROMPT_TITLE, promptVars(agt, userPrompt)))),
	}
}

func buildPromptSystem(agt *agent.Agent, userPrompt string) string {
	system := strings.TrimSpace(os.Expand(_PROMPT_SYSTEM, promptVars(agt, userPrompt)))
	agents := strings.TrimSpace(AgentsMd())
	if agents == "" {
		return system
	}
	if system == "" {
		return agents
	}
	return system + "\n\n" + agents
}

func promptVars(agt *agent.Agent, userPrompt string) func(string) string {
	return func(s string) string {
		switch s {
		case "SYSTEM_PROMPT":
			return buildPromptSystem(agt, userPrompt)
		case "MAX_ITERATIONS":
			return strconv.Itoa(agt.MaxIterations())
		case "HOST_OS":
			return runtime.GOOS
		case "HOST_CPU":
			return runtime.GOARCH
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
