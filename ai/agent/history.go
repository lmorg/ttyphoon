package agent

import (
	"fmt"

	"github.com/lmorg/mxtty/debug"
)

type HistoryItemT struct {
	Title       string
	CmdLine     string
	OutputBlock string
	Response    string
}

type HistoryT []HistoryItemT

func (meta *Meta) AddHistory(title string, response string) {
	meta.History = append(meta.History, HistoryItemT{
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
