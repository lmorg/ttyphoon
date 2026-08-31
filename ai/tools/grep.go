package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"sync"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/utils/grep"
)

type Grep struct {
	agent   aitypes.Agent
	enabled bool
	cacheMu sync.RWMutex
	cache   [][]*grep.Result
}

func init() {
	agent.ToolsAdd(&Grep{})
}

//go:embed grep_description.md
var grepDescription string

//If any context line is > 100 characters then that line is cropped to 99 and appended with "…"

func (t *Grep) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &Grep{agent: agent, enabled: true}, nil
}

func (t *Grep) Enabled() bool { return t.enabled }
func (t *Grep) Toggle()       { t.enabled = !t.enabled }

func (t *Grep) Name() string        { return "grep" }
func (t *Grep) Path() string        { return "internal" }
func (t *Grep) Description() string { return grepDescription }
func (t *Grep) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}

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
		returnT = t.newSearch(ctx, inputT)
	} else {
		returnT = t.getPage(inputT)
	}

	b, err := json.Marshal(returnT)
	return string(b), err
}

func (t *Grep) newSearch(ctx context.Context, input *grepInputT) *grepReturnT {
	cache := [][]*grep.Result{}
	ch := make(chan []*grep.Result)
	done := make(chan struct{})
	go func() {
		for results := range ch {
			cache = append(cache, results)
		}
		close(done)
	}()

	mapper := func(s string) string { return s }
	err := grep.BatchedStreamResults(ctx, t.agent.ProjectRoot(), input.Query, input.Options, mapper, ch)
	<-done
	if err != nil {
		return &grepReturnT{Error: err.Error()}
	}

	t.cacheMu.Lock()
	t.cache = cache
	t.cacheMu.Unlock()
	if len(cache) == 0 {
		return &grepReturnT{Results: []*grep.Result{}}
	}
	return &grepReturnT{PageCount: len(cache), Results: cache[0]}
}

func (t *Grep) getPage(input *grepInputT) *grepReturnT {
	t.cacheMu.RLock()
	defer t.cacheMu.RUnlock()
	if input.Page < 1 {
		return &grepReturnT{PageCount: len(t.cache), Error: "Page numbers cannot be 0 nor negative"}
	}
	if input.Page > len(t.cache) {
		return &grepReturnT{PageCount: len(t.cache), Error: "Page numbers cannot be greater than pageCount"}
	}

	return &grepReturnT{PageCount: len(t.cache), Results: t.cache[input.Page+1]}
}
