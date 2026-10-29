package prompts

import (
	_ "embed"
	"encoding/json"
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

//go:embed effort_quick.txt
var _PROMPT_EFFORT_QUICK string

//go:embed effort_balanced.txt
var _PROMPT_EFFORT_BALANCED string

//go:embed effort_thorough.txt
var _PROMPT_EFFORT_THOROUGH string

var rxSkillFunction = regexp.MustCompile(`^/[-a-zA-Z0-9]+($|\s)`)

const EffortVariableName = "ttyphoonEffort"

func GetExplainCmd(agt *agent.Agent, userPrompt string) string {
	return os.Expand(_PROMPT_EXPLAIN_CMD, promptVars(agt, userPrompt))
}

func GetExplainCmdMessages(agt *agent.Agent, userPrompt string) []*schema.Message {
	return []*schema.Message{
		schema.SystemMessage(buildPromptSystem(agt, userPrompt)),
		schema.UserMessage(strings.TrimSpace(os.Expand(_PROMPT_EXPLAIN_CMD, promptVars(agt, userPrompt)))),
	}
}

func GetExplainCmdMessagesWithImages(agt *agent.Agent, userPrompt string, images []aitypes.ImageAttachment) []*schema.Message {
	messages := GetExplainCmdMessages(agt, userPrompt)
	messages[len(messages)-1] = buildUserMessageWithImages(messages[len(messages)-1].Content, images)
	return messages
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

	parts := make([]schema.MessageInputPart, 0, 2*len(images)+1)
	if text != "" {
		parts = append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeText, Text: text})
	}
	for index, img := range images {
		img := img
		if img.Path != "" {
			reference, _ := json.Marshal(struct {
				Attachment int    `json:"attachment"`
				InputImage string `json:"inputImage"`
			}{Attachment: index + 1, InputImage: img.Path})
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeText,
				Text: "Local file for the following attached image:\n" + string(reference) +
					"\nTo edit or transform this image, pass the exact inputImage path directly to generateImage. " +
					"The attachment is already saved in an allowed image location; no file search, bash command, or workspace copy is needed.",
			})
		}
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

func GetAskMessagesWithImages(agt *agent.Agent, userPrompt string, images []aitypes.ImageAttachment) []*schema.Message {
	messages := GetAskMessages(agt, userPrompt)
	messages[len(messages)-1] = buildUserMessageWithImages(messages[len(messages)-1].Content, images)
	return messages
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
	if agents != "" {
		if system == "" {
			system = agents
		} else {
			system += "\n\n" + agents
		}
	}
	return system + "\n\n# Effort and user feedback\n\n" + effortInstructions(agt)
}

func effortInstructions(agt *agent.Agent) string {
	effort := "quick"
	if agt != nil && agt.Meta != nil {
		if value, ok := agt.Meta.Variables[EffortVariableName]; ok {
			effort = strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		}
	}

	switch effort {
	case "quick":
		return strings.TrimSpace(_PROMPT_EFFORT_QUICK)
	case "thorough":
		return strings.TrimSpace(_PROMPT_EFFORT_THOROUGH)
	default:
		return strings.TrimSpace(_PROMPT_EFFORT_BALANCED)
	}
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
