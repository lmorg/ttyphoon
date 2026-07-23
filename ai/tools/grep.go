package tools

import (
	"context"
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
}

func init() {
	agent.ToolsAdd(&Grep{})
}

const grepDescription = `Searches project files for text containing a query phrase and returns a list of file names, line numbers and the lines preceding and following the matched query. Supports additional filters.
Input: json:
{
	"query": "search string",
	"options": {
		"caseSensitive": false, # true = return case sensitive matches only
		"regex": false,         # true = query string is a regex pattern
		"wholeWord": false,     # true = query string will not match substrings of words 
		"fileFilter": "",       # filter which files to query. Or omit to scan all files in project
	}
}

fileFilter options:
- default: must match all of the space-separated words, eg "readme md" for only files with BOTH "readme" and "md" in the filename or extension
- contains "*" or "?": glob match, eg "*.go" for files with "go" extensions
- "g " prefix: glob match, eg "g *.go" for files with "go" extensions
- "rx " prefix: files must match a regex pattern, eg "rx .*\.go$" for files with "go" extensions
- "or " prefix: files much match any of the space-separated words, eg "or md txt" for any files containing the word "md" or "txt" in the file name or extension
- "! " prefix: files much not match all of the space-separated words, eg "! md txt" for any files that DO NOT contain either the word "md" nor "txt" in the file name or extension
All fileFilter matches are case insensitive

Returns: json:
{
	"error": "error message, if applicable",
	"results": [
		"fileName": "exmaple.md",
		"path": "path/in/project/example.md",
		"line": 15,
		"Context": [
			"line before matched line",
			"this is the matched line",
			"line after matched line"
		]
	]
}
`

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
}

type grepReturnT struct {
	Results []*grep.Result `json:"results"`
	Error   string         `json:"error"`
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

	var inputT grepInputT
	err = json.Unmarshal([]byte(input), &inputT)
	if err != nil {
		return err.Error(), nil
	}

	ch := make(chan []*grep.Result)
	var results []*grep.Result
	go func() {
		for result := range ch {
			results = append(results, result...)
		}
	}()

	mapper := func(s string) string { return s }
	err = grep.BatchedStreamResults(t.agent.GetMeta().Pwd, inputT.Query, inputT.Options, mapper, ch)
	if err != nil {
		b, _ := json.Marshal(grepReturnT{Error: err.Error()})
		return string(b), nil
	}

	b, err := json.Marshal(grepReturnT{Results: results})
	return string(b), err
}
