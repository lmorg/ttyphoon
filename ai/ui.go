package ai

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/prompts"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	historymd "github.com/lmorg/ttyphoon/utils/history_md"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func Explain(agent *agent.Agent, promptDialogue bool) {
	if !promptDialogue {
		askAI(agent, prompts.GetExplainCmd(agent, ""), fmt.Sprintf("```\n%s\n```", agent.Meta.CmdLine), agent.Meta.CmdLine)
		return
	}

	fn := func(v *types.InputBoxCallbackResultT) {
		askAI(agent, prompts.GetExplainCmd(agent, v.String()), "> "+v.String(), v.String())
	}

	agent.Renderer().DisplayInputBox("(Optional) Add to prompt", "", fn, nil)
}

func ExplainDoc(agt *agent.Agent, filename, contents string) {
	tile := agt.Renderer().ActiveTile()
	if tile == nil {
		return
	}

	agt.Meta = &agent.Meta{
		Pwd:         tile.Pwd(),
		CmdLine:     filename,
		OutputBlock: contents,
	}
	params := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:       "Ask AI about " + filename,
			Placeholder: "Optional",
			Multiline:   true,
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			askAI(agt, prompts.GetExplainDoc(agt, v.String()), "> "+v.String(), v.String())
		},
	}
	agt.Renderer().DisplayInputBoxW(params)
}

const _STICKY_MESSAGE = "Asking %s...."

func AskAI(agt *agent.Agent, prompt string) {
	go func() {
		askAI(agt, prompts.GetAsk(agt, prompt), "> "+prompt, prompt)
	}()
}

const SAVE_RESPONSE = "saveResponse"

func askAI(agt *agent.Agent, prompt string, title string, query string) {
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

		startAIJob(agt)

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
			agt.AddHistory(title, result)
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

func startAIJob(agt *agent.Agent) {
	runtime.EventsEmit(agt.Renderer().GetWindowContext(), "aiJobStart")
}

func emitAIResponseChunk(agt *agent.Agent, chunk string) {
	runtime.EventsEmit(agt.Renderer().GetWindowContext(), "aiResponseStream", chunk)
}

func finishAIJob(agt *agent.Agent) {
	runtime.EventsEmit(agt.Renderer().GetWindowContext(), "aiJobFinish")
}
