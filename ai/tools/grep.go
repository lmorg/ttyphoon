package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/utils/grep"
)

type Grep struct {
	agent   aitypes.Agent
	enabled bool
	cache   [][]*grep.Result
}

func init() {
	agent.ToolsAdd(&Grep{})
}

//go:embed grep_description.md
var grepDescription string

//If any context line is > 100 characters then that line is cropped to 99 and appended with "…"

func (t Grep) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &Grep{agent: agent, enabled: true}, nil
}

func (t *Grep) Enabled() bool { return t.enabled }
func (t *Grep) Toggle()       { t.enabled = !t.enabled }

func (t *Grep) Name() string        { return "grep" }
func (t *Grep) Path() string        { return "internal" }
func (t *Grep) Description() string { return grepDescription }

type grepInputT struct {
	Query   string       `json:"query"`
	Options grep.Options `json:"options"`
	Page    int          `json:"page"`
}

type grepReturnT struct {
	Results    []*grep.Result `json:"results"`
	PageCount  int            `json:"pagesCount"`
	PageNumber int            `json:"pageNumber"`
	Error      string         `json:"error"`
}

func (t *Grep) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	log.Printf("[debug] agent: tool=grep input=%s\n", input)

	inputT := new(grepInputT)
	err = json.Unmarshal([]byte(input), inputT)
	if err != nil {
		return err.Error(), nil
	}

	var returnT *grepReturnT

	if inputT.Query != "" {
		returnT = t.newSearch(inputT)
	} else {
		returnT = t.getPage(inputT)
	}

	b, err := json.Marshal(returnT)
	return string(b), err
}

func (t *Grep) newSearch(input *grepInputT) *grepReturnT {
	t.cache = [][]*grep.Result{}
	ch := make(chan []*grep.Result)
	go func() {
		for results := range ch {
			t.cache = append(t.cache, results)
		}
	}()

	mapper := func(s string) string { return s }
	err := grep.BatchedStreamResults(t.agent.GetMeta().Pwd, input.Query, input.Options, mapper, ch)
	if err != nil {
		return &grepReturnT{Error: err.Error()}
	}

	if len(t.cache) == 0 {
		return &grepReturnT{Results: []*grep.Result{}}
	}
	return &grepReturnT{PageCount: len(t.cache), Results: t.cache[0]}
}

func (t *Grep) getPage(input *grepInputT) *grepReturnT {
	if input.Page < 1 {
		return &grepReturnT{PageCount: len(t.cache), Error: "Page numbers cannot be 0 nor negative"}
	}
	if input.Page > len(t.cache) {
		return &grepReturnT{PageCount: len(t.cache), Error: "Page numbers cannot be greater than pageCount"}
	}

	return &grepReturnT{PageCount: len(t.cache), Results: t.cache[input.Page+1]}
}
