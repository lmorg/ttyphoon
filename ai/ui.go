package ai

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/prompts"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	historymd "github.com/lmorg/ttyphoon/utils/history_md"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func ExplainCmdOutput(agt *agent.Agent) {
	fn := func(v *types.InputBoxCallbackResultT) {
		if agt.Meta == nil {
			agt.Meta = &aitypes.Meta{}
		}
		agt.Meta.Variables = v.Variables

		query := fmt.Sprintf("%s\n\n```\n%s\n```", v.String(), agt.Meta.CmdLine)
		query = strings.TrimSpace(query)
		askAI(agt, prompts.GetExplainCmd(agt, query), query)
	}

	params := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:       "Explain output",
			Placeholder: "Optional",
			Multiline:   true,
			Variables:   []types.InputBoxWTVariables{SaveMarkdownToggle(false)},
		},
		OkFunc: fn,
	}
	agt.Renderer().DisplayInputBoxW(params)

	//agt.Renderer().DisplayInputBox("(Optional) Add to prompt", "", fn, nil)
}

func ExplainDoc(agt *agent.Agent, filename, contents string) {
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
			Title:       "Ask AI about " + filename,
			Placeholder: "Optional",
			Multiline:   true,
			Variables:   []types.InputBoxWTVariables{SaveMarkdownToggle(false)},
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			agt.Meta.Variables = v.Variables
			askAI(agt, prompts.GetExplainDoc(agt, v.String()), v.String())
		},
	}
	agt.Renderer().DisplayInputBoxW(params)
}

const _STICKY_MESSAGE = "Asking %s...."

func AskAI(agt *agent.Agent, prompt string) {
	go func() {
		askAI(agt, prompts.GetAsk(agt, prompt), prompt)
	}()
}

const SAVE_RESPONSE = "saveResponse"

func askAI(agt *agent.Agent, prompt string, query string) {
	prompt += prompts.AgentsMd()
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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		sticky.UpdateCanceller(cancel)
		defer cancel()

		startAIJob(agt, query)

		result, err := agt.RunLLMWithStream(ctx, prompt, func(chunk string) {
			if chunk == "" {
				return
			}
			emitAIResponseChunk(agt, chunk)
		})
		sticky.Close()
		finishAIJob(agt)
		if err != nil {
			agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result = err.Error()

		} else {
			agt.AddHistory(query, result)
		}

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
			FullPrompt:   prompt,
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
	runtime.EventsEmit(agt.Renderer().GetWindowContext(), "aiJobStart", title)
}

func emitAIResponseChunk(agt *agent.Agent, chunk string) {
	runtime.EventsEmit(agt.Renderer().GetWindowContext(), "aiResponseStream", chunk)
}

func finishAIJob(agt *agent.Agent) {
	runtime.EventsEmit(agt.Renderer().GetWindowContext(), "aiJobFinish")
}

func UriPrompt(agt *agent.Agent, prompt, tools string) {
	agt.Meta = &aitypes.Meta{}
	toolOptions := []string{""}
	if tools == "" {
		toolOptions[0] = "eg mcp(atlassian)"
	}
	agt.Renderer().DisplayInputBoxW(&types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:     "Invoke prompt from URI?",
			Multiline: true,
			Prefill:   prompt,
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
				SaveMarkdownToggle(false),
			},
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			if agt.Meta == nil {
				agt.Meta = &aitypes.Meta{}
			}
			agt.Meta.Variables = v.Variables

			if raw, ok := v.Variables["model"]; ok {
				selectedModel := strings.TrimSpace(fmt.Sprint(raw))
				if selectedModel != "" {
					err := agt.SetServiceModelFromSelection(selectedModel)
					if err != nil {
						agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
						return
					}
				}
			}

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
			AskAI(agt, v.String())
		},
	})
}
