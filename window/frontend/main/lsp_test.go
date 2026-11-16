package mainwindow

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/lsp"
)

func TestLSPReferenceContextIncludesAdjacentLines(t *testing.T) {
	got := lspReferenceContext(2, "file:///main.go", func(string) (string, bool) {
		return "before\nmatch\nafter\nlast", true
	})
	want := []string{"match", "after", "last"}
	if len(got) != len(want) {
		t.Fatalf("context = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("context = %v, want %v", got, want)
		}
	}
}

func TestNotesLSPEnvironmentHelper(t *testing.T) {
	if os.Getenv("TTYPHOON_NOTES_LSP_HELPER") != "1" {
		return
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := lsp.ReadMessage(reader)
		if err != nil {
			os.Exit(1)
		}
		var result any
		switch msg.Method {
		case "initialize":
			result = map[string]any{}
		case "test/environment":
			result = os.Getenv("TTYPHOON_NOTES_LSP_SCOPE")
		case "shutdown":
			result = nil
		case "exit":
			os.Exit(0)
		default:
			continue
		}
		raw, err := json.Marshal(result)
		if err != nil || lsp.WriteMessage(os.Stdout, lsp.Message{JSONRPC: "2.0", ID: msg.ID, Result: raw}) != nil {
			os.Exit(2)
		}
	}
}

func TestNotesLanguageAndTyposLSPUseSeparateEnvironmentFiles(t *testing.T) {
	t.Setenv("TTYPHOON_NOTES_LSP_HELPER", "1")
	t.Setenv("TTYPHOON_NOTES_LSP_SCOPE", "host")
	previousLanguages, previousTypos := config.Config.Notes.Languages, config.Config.Notes.SpellCheck.TyposLsp
	t.Cleanup(func() {
		config.Config.Notes.Languages = previousLanguages
		config.Config.Notes.SpellCheck.TyposLsp = previousTypos
	})
	dir := t.TempDir()
	languageFile, typosFile := filepath.Join(dir, "language.vars"), filepath.Join(dir, "typos.vars")
	if err := os.WriteFile(languageFile, []byte("TTYPHOON_NOTES_LSP_SCOPE=language\n"), 0600); err != nil {
		t.Fatal(err)
	}
	argv := []string{os.Args[0], "-test.run=^TestNotesLSPEnvironmentHelper$"}
	config.Config.Notes.Languages = config.LanguagesT{
		"scopedtest": {LSP: config.LspT{Command: argv, EnvFiles: []string{languageFile}}},
	}
	config.Config.Notes.SpellCheck.TyposLsp = config.LspT{Command: argv, EnvFiles: []string{typosFile}}
	ctx, cancel := context.WithCancel(context.Background())
	a := NewWailsApp()
	a.ctx = ctx
	t.Cleanup(func() {
		a.lspManager.StopAll()
		cancel()
	})
	language := a.notesLspServerFor(filepath.Join(dir, "file.scopedtest"), "scopedtest")
	if language == nil {
		t.Fatal("language server did not start")
	}
	if a.notesTyposServer() != nil {
		t.Fatal("typos server started with a missing environment file")
	}
	if err := os.WriteFile(typosFile, []byte("TTYPHOON_NOTES_LSP_SCOPE=typos\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if a.notesTyposServer() != nil {
		t.Fatal("typos server unexpectedly retried before a configuration reload")
	}
	a.typosMu.Lock()
	a.typosUnavailRevision = 0
	a.typosMu.Unlock()
	typos := a.notesTyposServer()
	if typos == nil {
		t.Fatal("typos server did not retry after configuration changed")
	}
	for _, test := range []struct {
		server *lsp.ServerProcess
		want   string
	}{{language, "language"}, {typos, "typos"}} {
		response, err := test.server.Transport().Call(ctx, "test/environment", nil, 2*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		var value string
		if err := json.Unmarshal(response.Result, &value); err != nil || value != test.want {
			t.Fatal("language server received the wrong environment scope")
		}
	}
}
