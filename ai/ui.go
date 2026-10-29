package ai

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/ai/prompts"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	historymd "github.com/lmorg/ttyphoon/utils/history_md"
	"github.com/lmorg/ttyphoon/utils/imageattachment"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func ExplainCmdOutput(agt *agent.Agent) {
	params := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           "Explain output",
			DraftKey:        AIWorkspaceDraftKey(agt),
			Placeholder:     "Optional",
			Multiline:       true,
			AllowImagePaste: true,
			Variables: []types.InputBoxWTVariables{
				agt.ListModelsInputVariable(),
				EffortInputVariable(),
				SaveMarkdownToggle(false),
			},
		},

		OkFunc: func(v *types.InputBoxCallbackResultT) {
			if agt.Meta == nil {
				agt.Meta = &aitypes.Meta{}
			}
			agt.Meta.Variables = v.Variables

			agt.SetModelFromInputVariable(v.Variables)

			query := fmt.Sprintf("%s\n\n```\n%s\n```", v.String(), agt.Meta.CmdLine)
			query = strings.TrimSpace(query)
			images, err := prepareInputboxImages(agt, v.Images)
			if err != nil {
				agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
				return
			}
			askAI(agt, prompts.GetExplainCmdMessagesWithImages(agt, query, images), query)
		},
	}
	agt.Renderer().DisplayInputBoxW(params)
}

func ExplainDoc(agt *agent.Agent, filename, contents string, images []aitypes.ImageAttachment) {
	tile := agt.Renderer().ActiveTile()
	if tile == nil {
		return
	}

	agt.Meta = &aitypes.Meta{
		Pwd:         tile.Pwd(),
		CmdLine:     filename,
		OutputBlock: contents,
	}
	params := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           "Ask AI about " + filename,
			DraftKey:        AIWorkspaceDraftKey(agt),
			Placeholder:     "Optional",
			Multiline:       true,
			AllowImagePaste: true,
			Variables: []types.InputBoxWTVariables{
				agt.ListModelsInputVariable(),
				EffortInputVariable(),
				SaveMarkdownToggle(false),
			},
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			agt.Meta.Variables = v.Variables
			agt.SetModelFromInputVariable(v.Variables)
			pastedImages, err := prepareInputboxImages(agt, v.Images)
			if err != nil {
				agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
				return
			}
			attachments := append(append([]aitypes.ImageAttachment(nil), images...), pastedImages...)
			askAI(agt, prompts.GetExplainDocMessagesWithImages(agt, v.String(), attachments), v.String())
		},
	}
	agt.Renderer().DisplayInputBoxW(params)
}

const _STICKY_MESSAGE = "Asking %s...."

func AskAI(agt *agent.Agent, prompt string) {
	AskAIWithImages(agt, prompt, nil)
}

func AskAIWithImages(agt *agent.Agent, prompt string, images []aitypes.ImageAttachment) {
	go func() {
		attachments, err := prepareInputboxImages(agt, images)
		if err != nil {
			agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}
		askAI(agt, prompts.GetAskMessagesWithImages(agt, prompt, attachments), prompt)
	}()
}

func prepareInputboxImages(agt *agent.Agent, images []aitypes.ImageAttachment) ([]aitypes.ImageAttachment, error) {
	stored, err := imageattachment.Store(images)
	if err != nil || len(stored) == 0 {
		return stored, err
	}
	if agt.Meta == nil {
		agt.Meta = &aitypes.Meta{}
	}
	for _, attachment := range stored {
		path := (&url.URL{Path: attachment.Path}).EscapedPath()
		agt.Meta.OutputBlock += fmt.Sprintf("\n\n![attached image](%s)", path)
	}
	return stored, nil
}

const SAVE_RESPONSE = "saveResponse"

const EFFORT_LEVEL = prompts.EffortVariableName

func EffortInputVariable() types.InputBoxWTVariables {
	return types.InputBoxWTVariables{
		Name:        EFFORT_LEVEL,
		Label:       "Effort",
		Description: "Quick returns a focused pass; Thorough investigates alternatives and edge cases. This guides the agent but is not a time limit.",
		Default:     "Quick",
		Options:     []string{"Quick", "Balanced", "Thorough"},
		Type:        "list",
	}
}

func AIWorkspaceDraftKey(agt *agent.Agent) string {
	if agt == nil || agt.Workspace() == "" {
		return ""
	}
	return "ai-workspace:" + agt.Workspace()
}

func joinPromptMessages(messages []*schema.Message) string {
	if len(messages) == 0 {
		return ""
	}

	parts := make([]string, 0, len(messages))
	for i := range messages {
		if messages[i] == nil {
			continue
		}
		content := strings.TrimSpace(messages[i].Content)
		if content == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s]\n%s", messages[i].Role, content))
	}

	return strings.Join(parts, "\n\n")
}

func askAI(agt *agent.Agent, promptMessages []*schema.Message, query string) {
	sticky := agt.Renderer().DisplaySticky(
		types.NOTIFY_INFO,
		fmt.Sprintf(_STICKY_MESSAGE, agt.ServiceName()),
		func() {},
	)

	go func() {
		startTime := time.Now()
		noteTime := startTime
		filenameCh := make(chan string, 1)
		saveResponse := true

		if len(agt.Meta.Variables) > 0 {
			if val, ok := agt.Meta.Variables[SAVE_RESPONSE]; ok {
				if b, ok := val.(bool); ok {
					saveResponse = b
				}
			}
		}

		if saveResponse {
			// Generate the note filename in parallel so it is ready when output is rendered.
			go func() {
				filenameCh <- buildAINoteFilename(agt, query, noteTime)
			}()
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.Config.Ai.RequestTimeoutDuration())
		sticky.UpdateCanceller(cancel)
		defer cancel()

		startAIJob(agt, query)

		result, err := agt.RunLLMWithMessageStream(ctx, promptMessages, nil)
		sticky.Close()
		if err != nil {
			agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result = err.Error()
		}

		_, promptID, storeErr := sessiondb.AppendActiveSessionEntry(agt.Workspace(), query, agt.Meta.CmdLine, agt.Meta.OutputBlock, result)
		if storeErr != nil {
			agt.Renderer().DisplayNotification(types.NOTIFY_WARN, storeErr.Error())
		}

		emitAIFinalResponse(agt, result, promptID)
		finishAIJob(agt)

		endTime := time.Now()
		data := &historymd.TemplateFieldsT{
			AppName:      app.Name(),
			GroupName:    agt.Term().Tile().GroupName(),
			TileName:     agt.Term().Tile().Name(),
			TimeStart:    startTime.Format(historymd.FMT_DATE),
			TimeEnd:      endTime.Format(historymd.FMT_DATE),
			TimeDuration: endTime.Sub(startTime).String(),
			Pwd:          agt.Meta.Pwd,
			Agent:        agt.ServiceName(),
			Query:        query,
			FullPrompt:   joinPromptMessages(promptMessages),
			Output:       result,
		}

		var b []byte
		buf := bytes.NewBuffer(b)
		historymd.Ai(data, func(tmpl *template.Template, data *historymd.TemplateFieldsT) error {
			return tmpl.Execute(buf, data)
		})

		if saveResponse {
			filename := <-filenameCh
			agt.Renderer().NotesCreateAndOpen(filename, buf.String())
		}
	}()
}

func SaveMarkdownToggle(Default bool) types.InputBoxWTVariables {
	return types.InputBoxWTVariables{
		Name:        SAVE_RESPONSE,
		Label:       "Save response",
		Description: "Write output to disk?",
		Type:        "boolean",
		Default:     fmt.Sprintf("%v", Default),
	}
}

func startAIJob(agt *agent.Agent, title string) {
	sessiondb.WriteToSessionLog(sessiondb.SessionLogContext{
		Workspace:       agt.Workspace(),
		Query:           title,
		CommandLine:     agt.Meta.CmdLine,
		OutputBlock:     agt.Meta.OutputBlock,
		WorkspaceActive: agt.IsWorkspaceActive(),
		Emit: func(event string, payload any) {
			runtime.EventsEmit(agt.Renderer().GetWindowContext(), event, payload)
		},
	}, sessiondb.SESSION_LOG_START_JOB, title)
}

func emitAIFinalResponse(agt *agent.Agent, output string, promptID int64) {
	sessiondb.WriteToSessionLog(sessiondb.SessionLogContext{
		Workspace:       agt.Workspace(),
		CommandLine:     agt.Meta.CmdLine,
		OutputBlock:     agt.Meta.OutputBlock,
		PromptID:        promptID,
		WorkspaceActive: agt.IsWorkspaceActive(),
		Emit: func(event string, payload any) {
			runtime.EventsEmit(agt.Renderer().GetWindowContext(), event, payload)
		},
	}, sessiondb.SESSION_LOG_FINALIZE_JOB, output)
}

func finishAIJob(agt *agent.Agent) {
	sessiondb.WriteToSessionLog(sessiondb.SessionLogContext{
		Workspace:       agt.Workspace(),
		WorkspaceActive: agt.IsWorkspaceActive(),
		Emit: func(event string, payload any) {
			runtime.EventsEmit(agt.Renderer().GetWindowContext(), event, payload)
		},
	}, sessiondb.SESSION_LOG_FINISH_JOB, "")
}

func UriPrompt(agt *agent.Agent, prompt, tools string) {
	agt.Meta = &aitypes.Meta{}
	toolOptions := []string{""}
	if tools == "" {
		toolOptions[0] = "eg mcp(atlassian)"
	}
	agt.Renderer().DisplayInputBoxW(&types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           "Invoke prompt from URI?",
			DraftKey:        AIWorkspaceDraftKey(agt),
			Multiline:       true,
			AllowImagePaste: true,
			Prefill:         prompt,
			Variables: []types.InputBoxWTVariables{
				{
					Name:        "tools",
					Label:       "Tools",
					Description: "Tools and MCP servers to use:\n",
					Default:     tools,
					Type:        "string",
					Options:     toolOptions,
				},
				agt.ListModelsInputVariable(),
				EffortInputVariable(),
				SaveMarkdownToggle(false),
			},
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			if agt.Meta == nil {
				agt.Meta = &aitypes.Meta{}
			}
			agt.Meta.Variables = v.Variables

			agt.SetModelFromInputVariable(v.Variables)

			var selectedTools string
			raw, ok := v.Variables["tools"]
			if ok {
				selectedTools, _ = raw.(string)
			}

			parsedTools, err := skills.ParseTools(selectedTools)
			if err != nil {
				agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
				return
			}
			err = agt.StartTools(parsedTools)
			if err != nil {
				agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
				return
			}
			AskAIWithImages(agt, v.String(), v.Images)
		},
	})
}
