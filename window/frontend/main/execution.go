package mainwindow

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/utils/jupyter"
	"github.com/lmorg/ttyphoon/utils/syntaxcompletion"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

func (a *WApp) GetLanguageDescriptions(language string) []string {
	return jupyter.GetLanguageDescriptions(language)
}

func (a *WApp) GetAllLanguageDescriptions() []string {
	return jupyter.GetAllLanguageDescriptions()
}

// RunNote runs a code block. Block IDs are only unique within a document, so
// the frontend also passes a runID that is unique across open documents. The
// runID keys the kill function and is echoed in noteRun and noteComplete.
func (a *WApp) RunNote(docPath, id, runID string, code, language string) {
	ch := make(chan *jupyter.OutputT)

	bookID := jupyter.BookId(a.GetCurrentProject(), a.filePath(docPath))

	ctx, kill := context.WithCancel(context.Background())
	a.notesMu.Lock()
	a.notesKills[runID] = kill
	a.notesMu.Unlock()

	go jupyter.RunNote(ctx, bookID, id, a.GetCurrentProject(), code, language, ch)

	go func() {
		for output := range ch {
			runtime.EventsEmit(a.ctx, "noteRun", map[string]string{
				"runId":   runID,
				"blockId": id,
				"output":  output.Output,
				"isError": fmt.Sprintf("%v", output.IsErr),
			})
		}
		// Emit completion event when channel closes
		runtime.EventsEmit(a.ctx, "noteComplete", map[string]string{
			"runId":   runID,
			"blockId": id,
		})
		a.notesMu.Lock()
		delete(a.notesKills, runID)
		a.notesMu.Unlock()
	}()
}

type RunFunctionReturnT struct {
	Output  string
	IsError bool
	CellId  string
}

type CompleteSyntaxReturnT struct {
	Applied bool   `json:"applied"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Text    string `json:"text"`
	Cursor  int    `json:"cursor"`
	Error   string `json:"error"`
}

func (a *WApp) CompleteSyntax(docPath, language, source string, cursor, selectionStart, selectionEnd int, trigger string) CompleteSyntaxReturnT {
	if a.syntaxEngine == nil {
		engine, err := syntaxcompletion.NewDefaultEngine()
		if err != nil {
			return CompleteSyntaxReturnT{Error: err.Error()}
		}
		a.syntaxEngine = engine
	}

	resolvedLanguage := jupyter.ResolveLanguageAlias(language, docPath)
	if resolvedLanguage == "" && docPath != "" {
		ext := strings.TrimPrefix(filepath.Ext(a.filePath(docPath)), ".")
		resolvedLanguage = jupyter.ResolveLanguageAlias(ext, docPath)
	}
	if resolvedLanguage == "" {
		resolvedLanguage = strings.TrimSpace(strings.ToLower(language))
	}

	result, err := a.syntaxEngine.Complete(syntaxcompletion.Request{
		Language:       resolvedLanguage,
		Source:         source,
		Cursor:         cursor,
		SelectionStart: selectionStart,
		SelectionEnd:   selectionEnd,
		Trigger:        trigger,
	})
	if err != nil {
		return CompleteSyntaxReturnT{Error: err.Error()}
	}

	return CompleteSyntaxReturnT{
		Applied: result.Applied,
		Start:   result.Start,
		End:     result.End,
		Text:    result.Text,
		Cursor:  result.Cursor,
	}
}

func (a *WApp) RunFunction(docPath, functionName, cellId, code string, parameters []string, language string) RunFunctionReturnT {
	bookID := jupyter.BookId(a.GetCurrentProject(), a.filePath(docPath))

	output, err := jupyter.RunFunction(context.Background(), a.GetCurrentProject(), bookID, functionName, code, parameters, language)
	if err != nil {
		return RunFunctionReturnT{
			Output:  err.Error(),
			IsError: true,
			CellId:  cellId,
		}
	}

	return RunFunctionReturnT{
		Output:  output,
		IsError: false,
		CellId:  cellId,
	}
}

func (a *WApp) FormatCodeBlock(docPath, codeId, code, language string) *jupyter.FormatCodeReturnT {
	bookId := jupyter.BookId(a.GetCurrentProject(), a.filePath(docPath))
	return jupyter.FormatCode(context.Background(), bookId, "lsp_"+codeId, a.GetCurrentProject(), code, language)
}

// FormatNotesContent formats a whole document's content (no snippet template
// wrapping) using the configured FormatCommand for the language. When no
// FormatCommand is configured, HasFormatter is false and the frontend falls
// back to LSP formatting.
func (a *WApp) FormatNotesContent(docPath, content, language string) *jupyter.FormatCodeReturnT {
	bookId := jupyter.BookId(a.GetCurrentProject(), a.filePath(docPath))
	return jupyter.FormatNotesContent(context.Background(), bookId, "editor", a.GetCurrentProject(), content, language)
}

func (a *WApp) FormatCodeFile(docPath, language string) {
	filePath := a.filePathWithProject(docPath, a.GetCurrentProject())
	log.Printf(`[debug] FormatCodeFile: pwd="%s" docPath="%s" language="%s"`, a.GetCurrentProject(), filePath, language)
	jupyter.FormatFile(context.Background(), a.GetCurrentProject(), filePath, language)
}

func (a *WApp) StopNote(runID string) {
	a.notesMu.Lock()
	fn, ok := a.notesKills[runID]
	a.notesMu.Unlock()
	if !ok {
		log.Printf("[warn] notes: cannot stop run %s because no kill function exists", runID)
		return
	}

	fn()

	runtime.EventsEmit(a.ctx, "noteRun", map[string]string{
		"runId":   runID,
		"output":  "[process killed]",
		"isError": fmt.Sprintf("%v", true),
	})
}
