package ai

import (
	"fmt"

	"github.com/lmorg/mxtty/debug"
)

type historyItemT struct {
	Title       string
	CmdLine     string
	OutputBlock string
	Response    string
}

type historyT []historyItemT

func (meta *AgentMeta) AddHistory(title string, response string) {
	meta.history = append(meta.history, historyItemT{
		Title:       title,
		CmdLine:     meta.CmdLine,
		OutputBlock: meta.OutputBlock,
		Response:    response,
	})
}

const _HISTORY_META = `
## Index:
%d
## Query summary:
%s
## Your response:
%s
---
`

func (meta *AgentMeta) History() string {
	if len(meta.history) == 0 {
		return ""
	}
	result := "---\n# Chat history:"
	for i := range meta.history {
		result += fmt.Sprintf(_HISTORY_META,
			i,
			meta.history[i].Title,
			meta.history[i].Response,
		)
	}
	debug.Log(result)
	return result
}
