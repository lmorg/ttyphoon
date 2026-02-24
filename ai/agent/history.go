package agent

import (
	"fmt"

	"github.com/lmorg/ttyphoon/debug"
)

type HistoryItemT struct {
	Title       string
	CmdLine     string
	OutputBlock string
	Response    string
}

type HistoryT []HistoryItemT

func (agent *Agent) AddHistory(title string, response string) {
	agent.History = append(agent.History, HistoryItemT{
		Title:       title,
		CmdLine:     agent.Meta.CmdLine,
		OutputBlock: agent.Meta.OutputBlock,
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

func (h HistoryT) String() string {
	if len(h) == 0 {
		return ""
	}
	result := "---\n# Chat history:"
	for i := range h {
		result += fmt.Sprintf(_HISTORY_META,
			i,
			h[i].Title,
			h[i].Response,
		)
	}
	debug.Log(result)
	return result
}
