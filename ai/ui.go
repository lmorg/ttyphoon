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
)

func Explain(agent *agent.Agent, promptDialogue bool) {
	if !promptDialogue {
		askAI(agent, prompts.GetExplain(agent, ""), fmt.Sprintf("```\n%s\n```", agent.Meta.CmdLine), agent.Meta.CmdLine)
		return
	}

	fn := func(userPrompt string) {
		askAI(agent, prompts.GetExplain(agent, userPrompt), "> "+userPrompt, userPrompt)
	}

	agent.Renderer().DisplayInputBox("(Optional) Add to prompt", "", fn, nil)
}

const _STICKY_MESSAGE = "Asking %s...."

func AskAI(agent *agent.Agent, prompt string) {
	go func() {
		askAI(agent, prompts.GetAsk(agent, prompt), "> "+prompt, prompt)
	}()
}

func askAI(agent *agent.Agent, prompt string, title string, query string) {
	sticky := agent.Renderer().DisplaySticky(
		types.NOTIFY_INFO,
		fmt.Sprintf(_STICKY_MESSAGE, agent.ServiceName()),
		func() {},
	)

	go func() {
		startTime := time.Now()
		noteTime := startTime
		filenameCh := make(chan string, 1)

		// Generate the note filename in parallel so it is ready when output is rendered.
		go func() {
			filenameCh <- buildAINoteFilename(agent, query, noteTime)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		sticky.UpdateCanceller(cancel)
		defer cancel()

		result, err := agent.RunLLMWithStream(ctx, prompt, func(chunk string) {
			if chunk == "" {
				return
			}
			agent.Renderer().EmitAIResponseChunk(chunk)
		})
		sticky.Close()
		if err != nil {
			agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result = err.Error()

		} else {
			agent.AddHistory(title, result)
		}

		endTime := time.Now()
		data := &historymd.TemplateFieldsT{
			AppName:      app.Name(),
			GroupName:    agent.Term().Tile().GroupName(),
			TileName:     agent.Term().Tile().Name(),
			TimeStart:    startTime.Format(historymd.FMT_DATE),
			TimeEnd:      endTime.Format(historymd.FMT_DATE),
			TimeDuration: endTime.Sub(startTime).String(),
			Pwd:          agent.Meta.Pwd,
			Agent:        agent.ServiceName(),
			Query:        query,
			FullPrompt:   prompt,
			Output:       result,
		}

		var b []byte
		buf := bytes.NewBuffer(b)
		historymd.Ai(data, func(tmpl *template.Template, data *historymd.TemplateFieldsT) error {
			return tmpl.Execute(buf, data)
		})

		filename := <-filenameCh
		agent.Renderer().NotesCreateAndOpen(filename, buf.String())
	}()
}
