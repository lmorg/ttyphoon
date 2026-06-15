package syntaxcompletion

import "testing"

func TestLoadDefaultConfig(t *testing.T) {
	cfg, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("LoadDefaultConfig() error = %v", err)
	}
	if len(cfg.Languages) == 0 {
		t.Fatalf("expected default languages to be loaded")
	}
	if _, ok := cfg.Languages["go"]; !ok {
		t.Fatalf("expected go language config")
	}
}

func TestGoBracketPair(t *testing.T) {
	eng := mustDefaultEngine(t)
	res, err := eng.Complete(Request{Language: "go", Source: "arr", Cursor: 3, Trigger: "["})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}

	out, cursor, err := Apply("arr", res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "arr[]" {
		t.Fatalf("output mismatch: got %q", out)
	}
	if cursor != 4 {
		t.Fatalf("cursor mismatch: got %d, want 4", cursor)
	}
}

func TestJSONBracePair(t *testing.T) {
	eng := mustDefaultEngine(t)
	res, err := eng.Complete(Request{Language: "json", Source: "", Cursor: 0, Trigger: "{"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, cursor, err := Apply("", res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "{}" {
		t.Fatalf("output mismatch: got %q", out)
	}
	if cursor != 1 {
		t.Fatalf("cursor mismatch: got %d, want 1", cursor)
	}
}

func TestWrapSelection(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "hello"
	res, err := eng.Complete(Request{
		Language:       "go",
		Source:         src,
		Cursor:         5,
		SelectionStart: 1,
		SelectionEnd:   4,
		Trigger:        "[",
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, cursor, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "h[ell]o" {
		t.Fatalf("output mismatch: got %q", out)
	}
	if cursor != 5 {
		t.Fatalf("cursor mismatch: got %d, want 5", cursor)
	}
}

func TestSkipExistingCloseQuote(t *testing.T) {
	eng := mustDefaultEngine(t)
	eng.classifier = nil
	src := "\"\""
	res, err := eng.Complete(Request{Language: "json", Source: src, Cursor: 1, Trigger: "\""})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, cursor, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != src {
		t.Fatalf("output mismatch: got %q, want unchanged", out)
	}
	if cursor != 2 {
		t.Fatalf("cursor mismatch: got %d, want 2", cursor)
	}
}

func TestMarkdownUnorderedListContinuation(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "- item"
	res, err := eng.Complete(Request{Language: "markdown", Source: src, Cursor: len(src), Trigger: "\n"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, cursor, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "- item\n- " {
		t.Fatalf("output mismatch: got %q", out)
	}
	if cursor != len(out) {
		t.Fatalf("cursor mismatch: got %d, want %d", cursor, len(out))
	}
}

func TestMarkdownChecklistContinuation(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "- [ ] item"
	res, err := eng.Complete(Request{Language: "markdown", Source: src, Cursor: len(src), Trigger: "\n"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, _, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "- [ ] item\n- [ ] " {
		t.Fatalf("output mismatch: got %q", out)
	}
}

func TestMarkdownOrderedListContinuation(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "9. thing"
	res, err := eng.Complete(Request{Language: "markdown", Source: src, Cursor: len(src), Trigger: "\n"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, _, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "9. thing\n10. " {
		t.Fatalf("output mismatch: got %q", out)
	}
}

func TestMarkdownExitOnEmptyItem(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "  - "
	res, err := eng.Complete(Request{Language: "markdown", Source: src, Cursor: len(src), Trigger: "\n"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, _, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "  - \n  " {
		t.Fatalf("output mismatch: got %q", out)
	}
}

func TestMarkdownExitOnEmptyChecklistItem(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "  - [ ] "
	res, err := eng.Complete(Request{Language: "markdown", Source: src, Cursor: len(src), Trigger: "\n"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, _, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "  - [ ] \n  " {
		t.Fatalf("output mismatch: got %q", out)
	}
}

func TestHTMLAutoCloseTag(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "<p>"
	res, err := eng.Complete(Request{Language: "html", Source: src, Cursor: len(src), Trigger: ">"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion to apply")
	}
	out, cursor, err := Apply(src, res)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if out != "<p></p>" {
		t.Fatalf("output mismatch: got %q", out)
	}
	if cursor != len("<p>") {
		t.Fatalf("cursor mismatch: got %d, want %d", cursor, len("<p>"))
	}
}

func TestHTMLDoesNotCloseSelfClosingTag(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "<br/>"
	res, err := eng.Complete(Request{Language: "html", Source: src, Cursor: len(src), Trigger: ">"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if res.Applied {
		t.Fatalf("did not expect completion for self-closing tag")
	}
}

func TestUnknownLanguageNoCompletion(t *testing.T) {
	eng := mustDefaultEngine(t)
	res, err := eng.Complete(Request{Language: "brainfuck", Source: "", Cursor: 0, Trigger: "{"})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if res.Applied {
		t.Fatalf("did not expect completion")
	}
}

func TestCursorOutOfRange(t *testing.T) {
	eng := mustDefaultEngine(t)
	_, err := eng.Complete(Request{Language: "go", Source: "abc", Cursor: 10, Trigger: "["})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestContextGuardBlocksWhenClassifierSaysString(t *testing.T) {
	eng := mustDefaultEngine(t)
	eng.classifier = stubClassifier{state: ContextState{InString: true}}

	res, err := eng.Complete(Request{Language: "go", Source: "x", Cursor: 1, Trigger: "["})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if res.Applied {
		t.Fatalf("did not expect completion when context guard blocks")
	}
}

func TestContextGuardFailOpenOnClassifierError(t *testing.T) {
	eng := mustDefaultEngine(t)
	eng.classifier = stubClassifier{err: errStubClassifier}

	res, err := eng.Complete(Request{Language: "go", Source: "x", Cursor: 1, Trigger: "["})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion when classifier fails (fail-open)")
	}
}

func TestContextGuardNoClassifierDoesNotBlock(t *testing.T) {
	eng := mustDefaultEngine(t)
	eng.classifier = nil

	res, err := eng.Complete(Request{Language: "go", Source: "x", Cursor: 1, Trigger: "["})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected completion when classifier is nil")
	}
}

type stubClassifier struct {
	state ContextState
	err   error
}

func (s stubClassifier) Detect(_ string, _ string, _ int) (ContextState, error) {
	if s.err != nil {
		return ContextState{}, s.err
	}
	return s.state, nil
}

var errStubClassifier = &stubError{"classifier error"}

type stubError struct {
	msg string
}

func (e *stubError) Error() string {
	return e.msg
}

func mustDefaultEngine(t *testing.T) *Engine {
	t.Helper()
	eng, err := NewDefaultEngine()
	if err != nil {
		t.Fatalf("NewDefaultEngine() error = %v", err)
	}
	return eng
}
