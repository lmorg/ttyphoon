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
	cache   [][]*grep.Result
}

func init() {
	agent.ToolsAdd(&Grep{})
}

const grepDescription = `Searches project files for text containing a query phrase and returns a list of file names, line numbers and the lines preceding and following the matched query. Supports additional filters.
Input: json:
{
	"query": "search string",   # omit if requesting a new page from the previous search 
	"options": {
		"caseSensitive": false, # true = return case sensitive matches only
		"regex": false,         # true = query string is a regex pattern
		"wholeWord": false,     # true = query string will not match substrings of words 
		"fileFilter": "",       # filter which files to query. Or omit to scan all files in project
	},
	"page": 0 # which page of results to return. Omit this if new search. Page count is included in return
}

fileFilter syntax:
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
	"pageCount": 5,  # number of pages of results. Each return is a single page
	"pageNumber": 1, # the index of this page in the page count. Indexes start from 1
	"results": [ # max 50 items per page
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
If pageCount is greater than 1, then the return only contains partial results and you'll need to make another request with {"page":2} to get the next page...and so on until pageNumber matches pageCount. However if pageCount is > 1 then that might be an indicator that a refined search is required. Multiple pages can result in running out of tokens
`

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
