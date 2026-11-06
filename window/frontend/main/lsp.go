package mainwindow

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	"github.com/lmorg/ttyphoon/utils/lsp"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) notifyLspStartError(languageID string, err error) {
	if err == nil {
		return
	}

	log.Println(err)

	message := err.Error()

	key := fmt.Sprintf("%s|%s", a.GetCurrentProject(), languageID)
	a.lspStartErrMu.Lock()
	if prev, ok := a.lspStartErrs[key]; ok && prev == message {
		a.lspStartErrMu.Unlock()
		log.Printf("lsp: %s", message)
		return
	}
	a.lspStartErrs[key] = message
	a.lspStartErrMu.Unlock()

	if renderer, ok := renderwebkit.CurrentRenderer(); ok {
		renderer.DisplayNotification(types.NOTIFY_ERROR, message)
	} else {
		log.Printf("lsp: renderer unavailable for notification: %s", message)
	}

	log.Printf("lsp: %s", message)
}

func (a *WApp) clearLspStartError(languageID string) {
	key := fmt.Sprintf("%s|%s", a.GetCurrentProject(), languageID)
	a.lspStartErrMu.Lock()
	defer a.lspStartErrMu.Unlock()
	delete(a.lspStartErrs, key)
}

func (a *WApp) notifyLspWorkspaceFiles(send func(t *lsp.Transport) error) {
	if a == nil || a.lspManager == nil || send == nil {
		return
	}

	for _, sp := range a.lspManager.ServersForWorkspace(a.GetCurrentProject()) {
		if sp == nil {
			continue
		}
		t := sp.Transport()
		if t == nil {
			continue
		}
		if err := send(t); err != nil {
			log.Printf("lsp: workspace file notification failed: %v", err)
		}
	}
}

// ResolveNotesLspLanguage returns the canonical language id for a filename.
func (a *WApp) ResolveNotesLspLanguage(filename string) string {
	return lsp.ResolveLanguageIDForFile(filename)
}

// NotesLspAvailableForRuntime reports whether an LSP server is configured for a
// Jupyter code block's language runtime (description, alias, or extension). The
// runtime is mapped to a file extension via the jupyter language bindings, then
// resolved to candidate language ids whose configured LSP commands are checked.
func (a *WApp) NotesLspAvailableForRuntime(runtime string) bool {
	ext := jupyter.FileExtensionForLanguage(runtime)
	if ext == "" {
		return false
	}

	syntheticPath := "block." + ext
	for _, id := range lsp.ResolveLanguageIDsForFile(syntheticPath) {
		if settings, ok := config.Config.Notes.Languages[id]; ok && len(settings.LSP.Command) > 0 {
			return true
		}
	}

	return false
}

// notesLspServerFor resolves the language server for the file's language id,
// starting it if necessary. Wires the diagnostics listener on first start.
// Returns nil if no server is configured.
func (a *WApp) notesLspServerFor(absPath, languageID string) *lsp.ServerProcess {
	if languageID == "" {
		languageID = lsp.ResolveLanguageIDForFile(absPath)
	}

	var candidateIDs []string
	if languageID != "" {
		candidateIDs = append(candidateIDs, languageID)
	}
	for _, id := range lsp.ResolveLanguageIDsForFile(absPath) {
		if id != "" && !slices.Contains(candidateIDs, id) {
			candidateIDs = append(candidateIDs, id)
		}
	}

	if len(candidateIDs) == 0 {
		return nil
	}

	selectedLanguageID := ""
	var argv []string
	var initOptions map[string]any
	for _, id := range candidateIDs {
		if settings, ok := config.Config.Notes.Languages[id]; ok && len(settings.LSP.Command) > 0 {
			argv = settings.LSP.Command
			initOptions = settings.LSP.InitOptions
			selectedLanguageID = id
			break
		}
	}

	if len(argv) == 0 {
		return nil
	}

	alreadyRunning := a.lspManager.Has(a.GetCurrentProject(), selectedLanguageID)

	sp, err := a.lspManager.GetOrStart(a.ctx, a.GetCurrentProject(), selectedLanguageID, argv, initOptions)
	if err != nil {
		a.notifyLspStartError(selectedLanguageID, err)
		return nil
	}

	// Start notification listener only on the first time this server is created.
	if !alreadyRunning {
		go lsp.ListenForNotifications(a.ctx, sp, func(uri string) (string, bool) {
			doc := a.lspDocs.GetByURI(uri)
			if doc != nil {
				return doc.Content(), true
			}

			path, err := lsp.URIToFilePath(uri)
			if err != nil {
				return "", false
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return "", false
			}

			return string(b), true
		}, func(payload lsp.DiagnosticsPayload) {
			runtime.EventsEmit(a.ctx, "notesLspDiagnostics", payload)
		}, func(payload lsp.LspProgressPayload) {
			runtime.EventsEmit(a.ctx, "notesLspProgress", payload)
		}, func(payload lsp.LspLogPayload) {
			runtime.EventsEmit(a.ctx, "notesLspLog", payload)
		})
	}

	if err := sp.EnsureInitialized(a.ctx, a.GetCurrentProject()); err != nil {
		a.notifyLspStartError(selectedLanguageID, err)
		return nil
	}

	a.clearLspStartError(selectedLanguageID)

	return sp
}

// NotesLspOpenDocument notifies the language server that a file was opened.
func (a *WApp) NotesLspOpenDocument(filePath, languageID, content string) {
	absPath := a.filePath(filePath)
	sp := a.notesLspServerFor(absPath, languageID)
	if sp == nil {
		return
	}
	t := sp.Transport()
	if t == nil {
		return
	}
	if err := a.lspDocs.DidOpen(a.ctx, t, absPath, languageID, content); err != nil {
		log.Printf("lsp: DidOpen %q: %v", absPath, err)
	}
}

// NotesLspChangeDocument notifies the language server of content changes.
func (a *WApp) NotesLspChangeDocument(filePath, content string) {
	absPath := a.filePath(filePath)
	if !a.lspDocs.IsOpen(absPath) {
		return
	}
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return
	}
	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return
	}
	t := sp.Transport()
	if t == nil {
		return
	}
	if err := a.lspDocs.DidChange(a.ctx, t, absPath, content); err != nil {
		log.Printf("lsp: DidChange %q: %v", absPath, err)
	}
}

// NotesLspSaveDocument notifies the language server that a file was saved.
func (a *WApp) NotesLspSaveDocument(filePath string) {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return
	}
	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return
	}
	t := sp.Transport()
	if t == nil {
		return
	}
	if err := a.lspDocs.DidSave(a.ctx, t, absPath); err != nil {
		log.Printf("lsp: DidSave %q: %v", absPath, err)
	}
}

// NotesLspCloseDocument notifies the language server that a file was closed.
func (a *WApp) NotesLspCloseDocument(filePath string) {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return
	}
	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		a.lspDocs.DidClose(a.ctx, nil, absPath) //nolint:errcheck // just clean up store
		return
	}
	t := sp.Transport()
	if err := a.lspDocs.DidClose(a.ctx, t, absPath); err != nil {
		log.Printf("lsp: DidClose %q: %v", absPath, err)
	}
}

const typosLspLanguageID = "__typos_lsp__"

// notesTyposServer resolves (starting if necessary) the workspace-wide
// typos-lsp spellcheck server. Returns nil when typos-lsp is not configured or
// failed to start, in which case callers fall back to aspell.
func (a *WApp) notesTyposServer() *lsp.ServerProcess {
	argv := config.Config.Notes.SpellCheck.TyposLsp.Command
	if len(argv) == 0 {
		return nil
	}

	a.typosMu.Lock()
	unavailable := a.typosUnavail[a.GetCurrentProject()]
	a.typosMu.Unlock()
	if unavailable {
		return nil
	}

	initOptions := config.Config.Notes.SpellCheck.TyposLsp.InitOptions
	alreadyRunning := a.lspManager.Has(a.GetCurrentProject(), typosLspLanguageID)

	sp, err := a.lspManager.GetOrStart(a.ctx, a.GetCurrentProject(), typosLspLanguageID, argv, initOptions)
	if err != nil {
		// typos-lsp is optional; do not surface a user-facing error. Record the
		// failure so a missing binary isn't repeatedly respawned.
		log.Printf("lsp: typos-lsp unavailable (%v); falling back to aspell", err)
		a.markTyposUnavailable()
		return nil
	}

	if !alreadyRunning {
		go lsp.ListenForNotifications(a.ctx, sp, func(uri string) (string, bool) {
			if doc := a.typosDocs.GetByURI(uri); doc != nil {
				return doc.Content(), true
			}
			return "", false
		}, func(payload lsp.DiagnosticsPayload) {
			runtime.EventsEmit(a.ctx, "notesTyposDiagnostics", payload)
		}, nil, nil)
	}

	if err := sp.EnsureInitialized(a.ctx, a.GetCurrentProject()); err != nil {
		log.Printf("lsp: typos-lsp init failed (%v); falling back to aspell", err)
		a.markTyposUnavailable()
		return nil
	}

	return sp
}

func (a *WApp) markTyposUnavailable() {
	a.typosMu.Lock()
	a.typosUnavail[a.GetCurrentProject()] = true
	a.typosMu.Unlock()
}

// NotesTyposAvailable reports whether typos-lsp is configured for spellchecking.
func (a *WApp) NotesTyposAvailable() bool {
	return len(config.Config.Notes.SpellCheck.TyposLsp.Command) > 0
}

// NotesTyposOpenDocument opens a document with the typos-lsp spellcheck server.
// Returns true when the server is running and the document was opened (the
// frontend then renders typos diagnostics and suppresses aspell for that
// editor); false means the caller should keep using aspell.
func (a *WApp) NotesTyposOpenDocument(filePath, languageID, content string) bool {
	sp := a.notesTyposServer()
	if sp == nil {
		return false
	}
	t := sp.Transport()
	if t == nil {
		return false
	}

	absPath := a.filePath(filePath)
	if languageID == "" {
		languageID = "plaintext"
	}

	// Re-open cleanly if it was already tracked as open (e.g. on reload).
	if a.typosDocs.IsOpen(absPath) {
		a.typosDocs.DidClose(a.ctx, t, absPath) //nolint:errcheck // best-effort reset
	}

	if err := a.typosDocs.DidOpen(a.ctx, t, absPath, languageID, content); err != nil {
		log.Printf("lsp: typos DidOpen %q: %v", absPath, err)
		return false
	}
	return true
}

// NotesTyposChangeDocument notifies typos-lsp of content changes.
func (a *WApp) NotesTyposChangeDocument(filePath, content string) {
	absPath := a.filePath(filePath)
	if !a.typosDocs.IsOpen(absPath) {
		return
	}
	sp := a.notesTyposServer()
	if sp == nil {
		return
	}
	t := sp.Transport()
	if t == nil {
		return
	}
	if err := a.typosDocs.DidChange(a.ctx, t, absPath, content); err != nil {
		log.Printf("lsp: typos DidChange %q: %v", absPath, err)
	}
}

// NotesTyposCloseDocument notifies typos-lsp that a document was closed.
func (a *WApp) NotesTyposCloseDocument(filePath string) {
	absPath := a.filePath(filePath)
	if !a.typosDocs.IsOpen(absPath) {
		return
	}
	sp := a.notesTyposServer()
	if sp == nil {
		return
	}
	t := sp.Transport()
	if t == nil {
		return
	}
	if err := a.typosDocs.DidClose(a.ctx, t, absPath); err != nil {
		log.Printf("lsp: typos DidClose %q: %v", absPath, err)
	}
}

// NotesLspHover requests hover text at a document position.
func (a *WApp) NotesLspHover(filePath string, line, character int) string {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return ""
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return ""
	}

	t := sp.Transport()
	if t == nil {
		return ""
	}

	text, err := lsp.RequestHover(a.ctx, t, doc.URI, doc.Content(), line, character, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: Hover %q (%d,%d): %v", absPath, line, character, err)
		return ""
	}

	return text
}

// NotesLspSignatureHelp requests signature help at a document position.
func (a *WApp) NotesLspSignatureHelp(filePath string, line, character, triggerKind int, triggerChar string) string {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return ""
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return ""
	}

	t := sp.Transport()
	if t == nil {
		return ""
	}

	text, err := lsp.RequestSignatureHelp(a.ctx, t, doc.URI, doc.Content(), line, character, triggerKind, triggerChar, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: SignatureHelp %q (%d,%d): %v", absPath, line, character, err)
		return ""
	}

	return text
}

// NotesLspCompletion requests completion items at a document position.
func (a *WApp) NotesLspCompletion(filePath string, line, character, triggerKind int, triggerChar string) []lsp.CompletionItem {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	items, err := lsp.RequestCompletion(a.ctx, t, doc.URI, doc.Content(), line, character, triggerKind, triggerChar, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: Completion %q (%d,%d): %v", absPath, line, character, err)
		return nil
	}

	return items
}

// NotesLspDefinition requests definition locations at a document position.
func (a *WApp) NotesLspDefinition(filePath string, line, character int) []lsp.DefinitionLocation {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	locations, err := lsp.RequestDefinition(a.ctx, t, doc.URI, doc.Content(), line, character, sp.PositionEncoding(), func(uri string) (string, bool) {
		openDoc := a.lspDocs.GetByURI(uri)
		if openDoc != nil {
			return openDoc.Content(), true
		}

		path, err := lsp.URIToFilePath(uri)
		if err != nil {
			return "", false
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}

		return string(b), true
	})
	if err != nil {
		log.Printf("lsp: Definition %q (%d,%d): %v", absPath, line, character, err)
		return nil
	}

	return locations
}

// NotesLspReferences requests references for a symbol at a document position.
func (a *WApp) NotesLspReferences(filePath string, line, character int) []lsp.ReferenceLocation {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil || sp.Transport() == nil {
		return nil
	}

	locations, err := lsp.RequestReferences(a.ctx, sp.Transport(), doc.URI, doc.Content(), line, character, sp.PositionEncoding(), func(uri string) (string, bool) {
		if openDoc := a.lspDocs.GetByURI(uri); openDoc != nil {
			return openDoc.Content(), true
		}
		path, pathErr := lsp.URIToFilePath(uri)
		if pathErr != nil {
			return "", false
		}
		b, readErr := os.ReadFile(path)
		return string(b), readErr == nil
	})
	if err != nil {
		log.Printf("lsp: References %q (%d,%d): %v", absPath, line, character, err)
		return nil
	}
	for i := range locations {
		locations[i].Context = lspReferenceContext(locations[i].Line, locations[i].URI, func(uri string) (string, bool) {
			if openDoc := a.lspDocs.GetByURI(uri); openDoc != nil {
				return openDoc.Content(), true
			}
			path, pathErr := lsp.URIToFilePath(uri)
			if pathErr != nil {
				return "", false
			}
			b, readErr := os.ReadFile(path)
			return string(b), readErr == nil
		})
	}
	return locations
}

func lspReferenceContext(line int, uri string, contentForURI func(string) (string, bool)) []string {
	content, ok := contentForURI(uri)
	if !ok {
		return nil
	}
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return nil
	}
	start := max(line-1, 0)
	end := min(line+2, len(lines))
	return append([]string(nil), lines[start:end]...)
}

// NotesLspDocumentSymbols requests symbols for the current document.
func (a *WApp) NotesLspDocumentSymbols(filePath string) []lsp.DocumentSymbolItem {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	items, err := lsp.RequestDocumentSymbols(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: DocumentSymbols %q: %v", absPath, err)
		return nil
	}

	return items
}

// NotesLspWorkspaceSymbols requests workspace symbols from the current file's language server.
func (a *WApp) NotesLspWorkspaceSymbols(filePath, query string) []lsp.WorkspaceSymbolItem {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	items, err := lsp.RequestWorkspaceSymbols(a.ctx, t, query, func(uri string) (string, bool) {
		openDoc := a.lspDocs.GetByURI(uri)
		if openDoc != nil {
			return openDoc.Content(), true
		}

		path, err := lsp.URIToFilePath(uri)
		if err != nil {
			return "", false
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}

		return string(contents), true
	}, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: WorkspaceSymbols %q (%q): %v", absPath, query, err)
		return nil
	}

	return items
}

// NotesLspInlayHints requests inlay hints for the current document.
func (a *WApp) NotesLspInlayHints(filePath string) []lsp.InlayHintItem {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	items, err := lsp.RequestInlayHints(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: InlayHints %q: %v", absPath, err)
		return nil
	}

	return items
}

// NotesLspSemanticTokens requests semantic tokens for the current document.
func (a *WApp) NotesLspSemanticTokens(filePath string) *lsp.SemanticTokensResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	result, err := lsp.RequestSemanticTokens(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding(), sp.SemanticTokensLegend())
	if err != nil {
		log.Printf("lsp: SemanticTokens %q: %v", absPath, err)
		return nil
	}

	return result
}

// NotesLspSemanticTokensDelta requests incremental semantic token updates for the
// current document using Monaco's previousResultId handoff when available.
func (a *WApp) NotesLspSemanticTokensDelta(filePath, previousResultID string) *lsp.SemanticTokensResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	result, err := lsp.RequestSemanticTokensDelta(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding(), sp.SemanticTokensLegend(), previousResultID)
	if err != nil {
		log.Printf("lsp: SemanticTokensDelta %q: %v", absPath, err)
		return nil
	}

	return result
}

// NotesLspCodeLens requests code lenses for the current document.
func (a *WApp) NotesLspCodeLens(filePath string) []lsp.CodeLensItem {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	items, err := lsp.RequestCodeLens(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: CodeLens %q: %v", absPath, err)
		return nil
	}

	return items
}

// NotesLspExecuteCodeLens executes one code lens command by index.
func (a *WApp) NotesLspExecuteCodeLens(filePath string, index int) bool {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return false
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return false
	}

	t := sp.Transport()
	if t == nil {
		return false
	}

	applied, err := lsp.ApplyCodeLens(a.ctx, t, doc.URI, index)
	if err != nil {
		log.Printf("lsp: ExecuteCodeLens %q #%d: %v", absPath, index, err)
		return false
	}

	return applied
}

// NotesLspFormat requests whole-document formatting and returns updated content.
func (a *WApp) NotesLspFormat(filePath string) lsp.FormatResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return lsp.FormatResult{}
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return lsp.FormatResult{}
	}

	t := sp.Transport()
	if t == nil {
		return lsp.FormatResult{}
	}

	result, err := lsp.RequestFormatting(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: Format %q: %v", absPath, err)
		return lsp.FormatResult{}
	}

	return result
}

// NotesLspFormatRange requests range formatting and returns updated content.
func (a *WApp) NotesLspFormatRange(filePath string, startLine, startCharacter, endLine, endCharacter int) lsp.FormatResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return lsp.FormatResult{}
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return lsp.FormatResult{}
	}

	t := sp.Transport()
	if t == nil {
		return lsp.FormatResult{}
	}

	result, err := lsp.RequestRangeFormatting(
		a.ctx,
		t,
		doc.URI,
		doc.Content(),
		startLine,
		startCharacter,
		endLine,
		endCharacter,
		sp.PositionEncoding(),
	)
	if err != nil {
		log.Printf("lsp: FormatRange %q (%d,%d)-(%d,%d): %v", absPath, startLine, startCharacter, endLine, endCharacter, err)
		return lsp.FormatResult{}
	}

	return result
}

// NotesLspCodeActions requests code actions for a cursor position.
func (a *WApp) NotesLspCodeActions(filePath string, line, character int, diagnostics []lsp.Diagnostic) []lsp.CodeActionItem {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return nil
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return nil
	}

	t := sp.Transport()
	if t == nil {
		return nil
	}

	items, err := lsp.RequestCodeActions(a.ctx, t, doc.URI, doc.Content(), line, character, diagnostics, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: CodeActions %q (%d,%d): %v", absPath, line, character, err)
		return nil
	}

	return items
}

// NotesLspApplyCodeAction applies a selected code action and returns updated content.
func (a *WApp) NotesLspApplyCodeAction(filePath string, line, character, index int, diagnostics []lsp.Diagnostic) lsp.ApplyCodeActionResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return lsp.ApplyCodeActionResult{}
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return lsp.ApplyCodeActionResult{}
	}

	t := sp.Transport()
	if t == nil {
		return lsp.ApplyCodeActionResult{}
	}

	result, err := lsp.ApplyCodeAction(a.ctx, t, doc.URI, doc.Content(), line, character, diagnostics, index, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: ApplyCodeAction %q (%d,%d) #%d: %v", absPath, line, character, index, err)
		return lsp.ApplyCodeActionResult{}
	}

	return result
}

// NotesLspPrepareRename checks if symbol rename is valid at a cursor position.
func (a *WApp) NotesLspPrepareRename(filePath string, line, character int) lsp.PrepareRenameResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return lsp.PrepareRenameResult{}
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return lsp.PrepareRenameResult{}
	}

	t := sp.Transport()
	if t == nil {
		return lsp.PrepareRenameResult{}
	}

	result, err := lsp.RequestPrepareRename(a.ctx, t, doc.URI, doc.Content(), line, character, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: PrepareRename %q (%d,%d): %v", absPath, line, character, err)
		return lsp.PrepareRenameResult{}
	}

	return result
}

// NotesLspRename applies LSP rename edits at a cursor position.
func (a *WApp) NotesLspRename(filePath string, line, character int, newName string) lsp.RenameResult {
	absPath := a.filePath(filePath)
	doc := a.lspDocs.Get(absPath)
	if doc == nil {
		return lsp.RenameResult{}
	}

	sp := a.notesLspServerFor(absPath, doc.LanguageID)
	if sp == nil {
		return lsp.RenameResult{}
	}

	t := sp.Transport()
	if t == nil {
		return lsp.RenameResult{}
	}

	result, err := lsp.RequestRename(a.ctx, t, doc.URI, doc.Content(), line, character, newName, sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: Rename %q (%d,%d) -> %q: %v", absPath, line, character, newName, err)
		return lsp.RenameResult{}
	}

	return result
}

// NotesLspStopAll shuts down all running language servers (called on app exit).
func (a *WApp) NotesLspStopAll() {
	a.lspManager.StopAll()
}
