package syntaxcompletion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Request represents editor state before a key-trigger is applied.
type Request struct {
	Language       string
	Source         string
	Cursor         int
	SelectionStart int
	SelectionEnd   int
	Trigger        string
}

// Result describes a single editor edit and where the caret should land.
type Result struct {
	Applied bool
	Start   int
	End     int
	Text    string
	Cursor  int
}

type Engine struct {
	cfg        *Config
	classifier ContextClassifier
}

func NewDefaultEngine() (*Engine, error) {
	cfg, err := LoadDefaultConfig()
	if err != nil {
		return nil, err
	}
	return &Engine{cfg: cfg, classifier: newDefaultContextClassifier()}, nil
}

func NewEngineFromYAML(yml []byte) (*Engine, error) {
	cfg, err := LoadConfig(yml)
	if err != nil {
		return nil, err
	}
	return &Engine{cfg: cfg, classifier: newDefaultContextClassifier()}, nil
}

func (e *Engine) Complete(req Request) (Result, error) {
	if e == nil || e.cfg == nil {
		return Result{}, fmt.Errorf("syntaxcompletion engine is not initialized")
	}
	if req.Cursor < 0 || req.Cursor > len(req.Source) {
		return Result{}, fmt.Errorf("cursor out of range")
	}

	start, end := normalizeSelection(req.SelectionStart, req.SelectionEnd, req.Cursor)
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	cfg, ok := e.cfg.Languages[lang]
	if !ok {
		return Result{}, nil
	}

	if shouldBlockByContext(e.classifier, req, cfg.ContextGuard) {
		return Result{}, nil
	}

	if req.Trigger == "\n" && cfg.ListContinuation != nil && cfg.ListContinuation.Enabled {
		if out, ok := completeList(req, *cfg.ListContinuation); ok {
			return out, nil
		}
	}

	if req.Trigger == ">" && cfg.AutoCloseTags != nil && cfg.AutoCloseTags.Enabled {
		if out, ok := completeTag(req, *cfg.AutoCloseTags); ok {
			return out, nil
		}
	}

	for _, p := range cfg.Pairs {
		if p.Open == "" || p.Close == "" || p.Open != req.Trigger {
			continue
		}

		if p.SkipClose {
			if req.Cursor < len(req.Source) && req.Source[req.Cursor:req.Cursor+len(p.Close)] == p.Close {
				return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: "", Cursor: req.Cursor + len(p.Close)}, nil
			}
		}

		if start != end {
			selected := req.Source[start:end]
			text := p.Open + selected + p.Close
			return Result{Applied: true, Start: start, End: end, Text: text, Cursor: start + len(p.Open) + len(selected)}, nil
		}

		text := p.Open + p.Close
		return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: text, Cursor: req.Cursor + len(p.Open)}, nil
	}

	return Result{}, nil
}

func Apply(source string, res Result) (string, int, error) {
	if !res.Applied {
		return source, 0, nil
	}
	if res.Start < 0 || res.End < res.Start || res.End > len(source) {
		return "", 0, fmt.Errorf("invalid edit range")
	}
	out := source[:res.Start] + res.Text + source[res.End:]
	if res.Cursor < 0 || res.Cursor > len(out) {
		return "", 0, fmt.Errorf("invalid cursor position")
	}
	return out, res.Cursor, nil
}

func normalizeSelection(start, end, cursor int) (int, int) {
	if start < 0 || end < 0 {
		return cursor, cursor
	}
	if start > end {
		start, end = end, start
	}
	return start, end
}

var mdListRe = regexp.MustCompile(`^(\s*)(?:([*+-])\s+(\[(?: |x|X)\])?\s*(.*)|(\d+)([.)])\s+(.*))$`)

func completeList(req Request, cfg ListContinuationRule) (Result, bool) {
	lineStart := strings.LastIndex(req.Source[:req.Cursor], "\n") + 1
	lineEnd := req.Cursor
	if idx := strings.Index(req.Source[req.Cursor:], "\n"); idx >= 0 {
		lineEnd = req.Cursor + idx
	} else {
		lineEnd = len(req.Source)
	}

	lineBeforeCursor := req.Source[lineStart:req.Cursor]
	if strings.TrimSpace(lineBeforeCursor) == "" {
		return Result{}, false
	}
	// Keep the heuristic stable: only continue list when Enter happens at EOL.
	if req.Cursor != lineEnd {
		return Result{}, false
	}

	m := mdListRe.FindStringSubmatch(lineBeforeCursor)
	if len(m) == 0 {
		return Result{}, false
	}

	indent := m[1]
	if m[2] != "" { // unordered list
		marker := m[2]
		isChecklist := m[3] != ""
		itemBody := strings.TrimSpace(m[4])

		continuationMarker := marker
		if isChecklist {
			continuationMarker = marker + " [ ]"
		}

		if len(cfg.UnorderedMarkers) > 0 && !contains(cfg.UnorderedMarkers, continuationMarker) {
			if !contains(cfg.UnorderedMarkers, marker) {
				return Result{}, false
			}
		}

		if cfg.ExitOnEmptyItem && itemBody == "" {
			text := "\n" + indent
			return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: text, Cursor: req.Cursor + len(text)}, true
		}

		text := "\n" + indent + continuationMarker + " "
		return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: text, Cursor: req.Cursor + len(text)}, true
	}

	if !cfg.Ordered {
		return Result{}, false
	}

	n, err := strconv.Atoi(m[5])
	if err != nil {
		return Result{}, false
	}
	delim := m[6]
	itemBody := strings.TrimSpace(m[7])
	if cfg.ExitOnEmptyItem && itemBody == "" {
		text := "\n" + indent
		return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: text, Cursor: req.Cursor + len(text)}, true
	}
	text := fmt.Sprintf("\n%s%d%s ", indent, n+1, delim)
	return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: text, Cursor: req.Cursor + len(text)}, true
}

var htmlOpenTagRe = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9:-]*)[^>]*>$`)

func completeTag(req Request, cfg AutoCloseTagsRule) (Result, bool) {
	if req.Cursor == 0 {
		return Result{}, false
	}
	left := req.Source[:req.Cursor]
	m := htmlOpenTagRe.FindStringSubmatch(left)
	if len(m) < 2 {
		return Result{}, false
	}
	full := m[0]
	tag := strings.ToLower(m[1])
	if strings.HasPrefix(full, "</") || strings.HasSuffix(strings.TrimSpace(full), "/>") {
		return Result{}, false
	}
	if len(cfg.Allowed) > 0 && !containsFold(cfg.Allowed, tag) {
		return Result{}, false
	}
	text := "</" + tag + ">"
	return Result{Applied: true, Start: req.Cursor, End: req.Cursor, Text: text, Cursor: req.Cursor}, true
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsFold(items []string, want string) bool {
	for _, item := range items {
		if strings.EqualFold(item, want) {
			return true
		}
	}
	return false
}

func shouldBlockByContext(classifier ContextClassifier, req Request, cfg *ContextGuardRule) bool {
	if classifier == nil || cfg == nil || !cfg.Enabled {
		return false
	}

	state, err := classifier.Detect(req.Language, req.Source, req.Cursor)
	if err != nil {
		// Never block typing when classification fails.
		return false
	}

	for _, disallow := range cfg.DisallowIn {
		switch strings.ToLower(strings.TrimSpace(disallow)) {
		case "comment", "comments":
			if state.InComment {
				return true
			}
		case "string", "strings":
			if state.InString {
				return true
			}
		}
	}

	return false
}
