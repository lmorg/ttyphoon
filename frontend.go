package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	godebug "runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/hotkeys"
	metamd "github.com/lmorg/ttyphoon/tools/meta-md"
	"github.com/lmorg/ttyphoon/types"
	globalhotkeys "github.com/lmorg/ttyphoon/utils/global_hotkeys"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	"github.com/lmorg/ttyphoon/utils/lsp"
	menuhyperlink "github.com/lmorg/ttyphoon/utils/menu_hyperlink"
	"github.com/lmorg/ttyphoon/utils/notes"
	"github.com/lmorg/ttyphoon/utils/swagger"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	mac "github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/clipboard"
)

//go:embed all:frontend/dist
var wailsAssets embed.FS

// App struct
type WApp struct {
	ctx           context.Context
	mdBaseDir     string
	projRoot      string
	groupName     string
	usrNotesDir   string
	homeDir       string
	globalNotes   string
	visible       bool
	notesKills    map[string]func()
	notesStickies map[string]types.Notification
	notesMu       sync.Mutex
	lspStartErrs  map[string]string
	lspManager    *lsp.Manager
	lspDocs       *lsp.DocumentStore
}

// NewApp creates a new App application struct
func NewWailsApp() *WApp {
	a := &WApp{
		visible:       true,
		notesKills:    map[string]func(){},
		notesStickies: map[string]types.Notification{},
		lspStartErrs:  map[string]string{},
		homeDir:       xdg.Home,
		projRoot:      notes.DirProjectRoot(""),
		globalNotes:   notes.DirGlobal(),
		lspManager:    lsp.NewManager(),
		lspDocs:       lsp.NewDocumentStore(),
	}

	return a
}

func (a *WApp) notifyLspStartError(languageID string, argv []string, err error) {
	if err == nil {
		return
	}

	log.Println(err)

	message := err.Error() //fmt.Sprintf("LSP (%s) failed to start: %v", languageID, err)
	/*if len(argv) > 0 {
		message += ": " + strings.Join(argv, " ")
	}*/

	key := fmt.Sprintf("%s|%s", a.projRoot, languageID)
	if prev, ok := a.lspStartErrs[key]; ok && prev == message {
		log.Printf("lsp: %s", message)
		return
	}
	a.lspStartErrs[key] = message

	if renderer, ok := renderwebkit.CurrentRenderer(); ok {
		renderer.DisplayNotification(types.NOTIFY_ERROR, message)
	} else {
		log.Printf("lsp: renderer unavailable for notification: %s", message)
	}

	log.Printf("lsp: %s", message)
}

func (a *WApp) clearLspStartError(languageID string) {
	key := fmt.Sprintf("%s|%s", a.projRoot, languageID)
	delete(a.lspStartErrs, key)
}

func (a *WApp) notifyLspWorkspaceFiles(send func(t *lsp.Transport) error) {
	if a == nil || a.lspManager == nil || send == nil {
		return
	}

	for _, sp := range a.lspManager.ServersForWorkspace(a.projRoot) {
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

type WindowStyleT struct {
	Colours          *ColoursT `json:"colors"`
	StatusBar        bool      `json:"statusBar"`
	FontFamily       string    `json:"fontFamily"`
	FontSize         int       `json:"fontSize"`
	AdjustCellWidth  int       `json:"adjustCellWidth"`
	AdjustCellHeight int       `json:"adjustCellHeight"`
}

type ColoursT struct {
	Fg            types.Colour `json:"fg"`
	Bg            types.Colour `json:"bg"`
	Black         types.Colour `json:"black"`
	Red           types.Colour `json:"red"`
	Green         types.Colour `json:"green"`
	Yellow        types.Colour `json:"yellow"`
	Blue          types.Colour `json:"blue"`
	Magenta       types.Colour `json:"magenta"`
	Cyan          types.Colour `json:"cyan"`
	White         types.Colour `json:"white"`
	BlackBright   types.Colour `json:"blackBright"`
	RedBright     types.Colour `json:"redBright"`
	GreenBright   types.Colour `json:"greenBright"`
	YellowBright  types.Colour `json:"yellowBright"`
	BlueBright    types.Colour `json:"blueBright"`
	MagentaBright types.Colour `json:"magentaBright"`
	CyanBright    types.Colour `json:"cyanBright"`
	WhiteBright   types.Colour `json:"whiteBright"`
	Accent        types.Colour `json:"accent"`
	Selection     types.Colour `json:"selection"`
	Link          types.Colour `json:"link"`
	Error         types.Colour `json:"error"`
}

func NewWindowStyle() *WindowStyleT {
	fontFamily := config.Config.TypeFace.FontName
	if fontFamily == "" {
		fontFamily = "Fira Code"
	}
	return &WindowStyleT{
		Colours: &ColoursT{
			Fg:            *types.SGR_DEFAULT.Fg,
			Bg:            *types.SGR_DEFAULT.Bg,
			Black:         *types.SGR_COLOR_BLACK,
			Red:           *types.SGR_COLOR_RED,
			Green:         *types.SGR_COLOR_GREEN,
			Yellow:        *types.SGR_COLOR_YELLOW,
			Blue:          *types.SGR_COLOR_BLUE,
			Magenta:       *types.SGR_COLOR_MAGENTA,
			Cyan:          *types.SGR_COLOR_CYAN,
			White:         *types.SGR_COLOR_WHITE,
			BlackBright:   *types.SGR_COLOR_BLACK_BRIGHT,
			RedBright:     *types.SGR_COLOR_RED_BRIGHT,
			GreenBright:   *types.SGR_COLOR_GREEN_BRIGHT,
			YellowBright:  *types.SGR_COLOR_YELLOW_BRIGHT,
			BlueBright:    *types.SGR_COLOR_BLUE_BRIGHT,
			MagentaBright: *types.SGR_COLOR_MAGENTA_BRIGHT,
			CyanBright:    *types.SGR_COLOR_CYAN_BRIGHT,
			WhiteBright:   *types.SGR_COLOR_WHITE_BRIGHT,
			Accent:        *types.SGR_COLOR_ACCENT,
			Selection:     *types.COLOR_SELECTION,
			Link:          *types.SGR_COLOR_BLUE,
			Error:         *types.COLOR_ERROR,
		},
		StatusBar:        config.Config.Window.StatusBar,
		FontFamily:       fmt.Sprintf(`"%s", monospace`, fontFamily),
		FontSize:         config.Config.TypeFace.FontSize,
		AdjustCellWidth:  config.Config.TypeFace.AdjustCellWidth,
		AdjustCellHeight: config.Config.TypeFace.AdjustCellHeight,
	}
}

func (a *WApp) GetWindowStyle() WindowStyleT {
	return *NewWindowStyle()
}

func (a *WApp) GetTerminalGlyphSize() *types.XY {
	renderer, ok := renderwebkit.CurrentRenderer()
	if ok {
		glyphSize := renderer.GetGlyphSize()
		if glyphSize != nil {
			return glyphSize
		}
	}

	return nil
}

func (a *WApp) WindowShow() {
	a.visible = true
	runtime.WindowShow(a.ctx)
}

func (a *WApp) WindowHide() {
	a.visible = false
	runtime.WindowHide(a.ctx)
}

func (a *WApp) WindowShowHide() {
	a.visible = !a.visible
	if a.visible {
		a.WindowShow()
	} else {
		a.WindowHide()
	}
}

func (a *WApp) TerminalMouseButton(cellX, cellY int32, button int, clicks int, pressed bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	state := types.BUTTON_RELEASED
	if pressed {
		state = types.BUTTON_PRESSED
	}

	renderer.HandleMouseButton(cellX, cellY, types.MouseButtonT(button), uint8(clicks), state)
}

func (a *WApp) TerminalMouseWheel(cellX, cellY, moveX, moveY int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleMouseWheel(cellX, cellY, moveX, moveY)
}

func (a *WApp) TerminalMouseMotion(cellX, cellY, relX, relY, state int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleMouseMotion(cellX, cellY, relX, relY, state)
}

func (a *WApp) TerminalSetFocus(focused bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	tile := renderer.ActiveTile()
	if tile == nil || tile.GetTerm() == nil {
		return
	}

	tile.GetTerm().SetFocus(focused)
	renderer.TriggerRedraw()
}

func (a *WApp) FocusTerminalPane() {
	runtime.EventsEmit(a.ctx, "focusTerminalPane")
}

func (a *WApp) TerminalRequestRedraw() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.TriggerRedraw()
}

func (a *WApp) CloseNotification(id int64) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.CloseNotification(id)
}

func (a *WApp) TerminalMenuHighlight(menuID, index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.MenuHighlight(menuID, index)
}

func (a *WApp) TerminalMenuSelect(menuID, index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.MenuSelect(menuID, index)
}

func (a *WApp) TerminalMenuCancel(menuID, index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.MenuCancel(menuID, index)
}

func (a *WApp) TerminalTextInput(text string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleTextInput(text)
}

func (a *WApp) TerminalResize(cols, rows int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.WindowResized(cols, rows)
	renderer.TriggerRedraw()
}

func (a *WApp) TerminalSetGlyphSize(width, height int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.SetGlyphSize(width, height)
}

func (a *WApp) TerminalCopyImageDataURL(dataURL string) error {
	if dataURL == "" {
		return fmt.Errorf("empty image data URL")
	}

	comma := strings.IndexByte(dataURL, ',')
	if comma <= 0 || comma >= len(dataURL)-1 {
		return fmt.Errorf("invalid image data URL")
	}

	meta := dataURL[:comma]
	if !strings.Contains(meta, ";base64") {
		return fmt.Errorf("image data URL is not base64 encoded")
	}

	png, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return fmt.Errorf("decode image data URL: %w", err)
	}

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer unavailable")
	}

	return renderer.CopyImageToClipboard(png)
}

func (a *WApp) RunAIAgentWithStream(tileID, prompt string) error {
	agt := agent.Get(tileID)
	if agt == nil {
		return fmt.Errorf("agent not found for tile %s", tileID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, err := agt.RunLLMWithStream(ctx, prompt, func(chunk string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "aiResponseStream", chunk)
		}
	})

	return err
}

func (a *WApp) TerminalGetTabs() []map[string]any {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return nil
	}

	tabs := renderer.GetWindowTabs()
	out := make([]map[string]any, 0, len(tabs))
	for i := range tabs {
		out = append(out, map[string]any{
			"id":     tabs[i].ID,
			"name":   tabs[i].Name,
			"index":  tabs[i].Index,
			"active": tabs[i].Active,
		})
	}

	return out
}

func (a *WApp) TerminalSelectWindow(windowID string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.SelectWindow(windowID)
}

func (a *WApp) TerminalKeyPress(key string, ctrl, alt, shift, meta bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleKeyPress(key, ctrl, alt, shift, meta)
}

func (a *WApp) NotesKeyPress(key string, ctrl, alt, shift, meta bool) map[string]bool {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return map[string]bool{
			"consume":      false,
			"prefixActive": false,
		}
	}

	consume, prefixActive := renderer.HandleNotesKeyPress(key, ctrl, alt, shift, meta)
	return map[string]bool{
		"consume":      consume,
		"prefixActive": prefixActive,
	}
}

func (a *WApp) TerminalInputBoxSubmit(id int64, value string, isOk bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.InputBoxSubmit(id, value, isOk)
}

func (a *WApp) startTerminalWindow() {
	err := startBackend(a)
	if err != nil {
		panic(err)
	}

	// Set app reference on renderer for hotkey handlers
	if wr, ok := renderwebkit.CurrentRenderer(); ok {
		wr.SetApp(a)
	}
}

func (a *WApp) SendIpc(eventName string, parameters map[string]string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	switch eventName {
	case "terminal-extra-tab-state":
		tabID := strings.TrimSpace(parameters["id"])
		if tabID == "" {
			return
		}

		enabled := strings.EqualFold(parameters["enabled"], "true")
		if !enabled {
			renderer.SetTerminalPaneTabs(nil)
			return
		}

		tabName := strings.TrimSpace(parameters["name"])
		if tabName == "" {
			tabName = tabID
		}

		active := strings.EqualFold(parameters["active"], "true")
		renderer.SetTerminalPaneTabs([]types.TerminalPaneTab{{
			ID:     tabID,
			Name:   tabName,
			Active: active,
		}})

	case "terminal-notify":
		message := strings.TrimSpace(parameters["message"])
		if message == "" {
			return
		}

		switch strings.ToLower(strings.TrimSpace(parameters["level"])) {
		case "error":
			renderer.DisplayNotification(types.NOTIFY_ERROR, message)
		case "warn", "warning":
			renderer.DisplayNotification(types.NOTIFY_WARN, message)
		case "debug":
			renderer.DisplayNotification(types.NOTIFY_DEBUG, message)
		default:
			renderer.DisplayNotification(types.NOTIFY_INFO, message)
		}

	case "terminal-sticky-create":
		id := strings.TrimSpace(parameters["id"])
		message := strings.TrimSpace(parameters["message"])
		if id == "" || message == "" {
			return
		}
		var notifType types.NotificationType
		switch strings.ToLower(strings.TrimSpace(parameters["level"])) {
		case "error":
			notifType = types.NOTIFY_ERROR
		case "warn", "warning":
			notifType = types.NOTIFY_WARN
		default:
			notifType = types.NOTIFY_INFO
		}
		a.notesMu.Lock()
		if existing, ok := a.notesStickies[id]; ok {
			existing.Close()
			delete(a.notesStickies, id)
		}
		a.notesMu.Unlock()
		sticky := renderer.DisplaySticky(notifType, message, func() {})
		a.notesMu.Lock()
		a.notesStickies[id] = sticky
		a.notesMu.Unlock()

	case "terminal-sticky-update":
		id := strings.TrimSpace(parameters["id"])
		message := strings.TrimSpace(parameters["message"])
		if id == "" || message == "" {
			return
		}
		a.notesMu.Lock()
		sticky, ok := a.notesStickies[id]
		a.notesMu.Unlock()
		if ok {
			sticky.SetMessage(message)
		}

	case "terminal-sticky-close":
		id := strings.TrimSpace(parameters["id"])
		if id == "" {
			return
		}
		a.notesMu.Lock()
		if sticky, ok := a.notesStickies[id]; ok {
			sticky.Close()
			delete(a.notesStickies, id)
		}
		a.notesMu.Unlock()
	}
}

func (a *WApp) GetLanguageDescriptions(language string) []string {
	return jupyter.GetLanguageDescriptions(language)
}

func (a *WApp) GetAllLanguageDescriptions() []string {
	return jupyter.GetAllLanguageDescriptions()
}

func (a *WApp) RunNote(id string, code, language string) {
	ch := make(chan *jupyter.OutputT)

	ctx, kill := context.WithCancel(context.Background())
	a.notesMu.Lock()
	a.notesKills[id] = kill
	a.notesMu.Unlock()

	go jupyter.RunNote(ctx, id, a.projRoot, code, language, ch)

	go func() {
		for output := range ch {
			runtime.EventsEmit(a.ctx, "noteRun", map[string]string{
				"blockId": output.Id,
				"output":  output.Output,
				"isError": fmt.Sprintf("%v", output.IsErr),
			})
		}
		// Emit completion event when channel closes
		runtime.EventsEmit(a.ctx, "noteComplete", map[string]string{
			"blockId": id,
		})
		a.notesMu.Lock()
		delete(a.notesKills, id)
		a.notesMu.Unlock()
	}()
}

type RunFunctionReturnT struct {
	Output  string
	IsError bool
	CellId  string
}

func (a *WApp) RunFunction(cellId, code string, parameters []string, language string) RunFunctionReturnT {
	output, err := jupyter.RunFunction(context.Background(), a.projRoot, code, parameters, language)
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

func (a *WApp) StopNote(id string) {
	a.notesMu.Lock()
	fn, ok := a.notesKills[id]
	a.notesMu.Unlock()
	if !ok {
		log.Printf("cannot stop note %s because no kill function exists", id)
		return
	}

	fn()

	runtime.EventsEmit(a.ctx, "noteRun", map[string]string{
		"blockId": id,
		"output":  "[process killed]",
		"isError": fmt.Sprintf("%v", true),
	})
}

type GetFileReturnT struct {
	Contents string `json:"contents"`
	Binary   bool   `json:"binary"`
	Error    string `json:"error"`
}

func (a *WApp) GetFile(filename string) GetFileReturnT {
	requestedFilename := strings.TrimSpace(filename)
	filename = a.filePath(filename)

	stat, err := os.Stat(filename)
	if err != nil {
		//log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}
	if stat.Size() > config.Config.Notes.MaxFileSize*1024*1024 {
		return GetFileReturnT{Error: "File too large to open"}
	}

	f, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}

	defer f.Close()
	defer godebug.FreeOSMemory()

	b, err := io.ReadAll(f)
	if err != nil {
		//log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}

	a.mdBaseDir = filepath.Dir(filename)
	err = notes.RecentListAdd(a.projRoot, requestedFilename)
	if err != nil {
		//log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}

	if bytes.Contains(b[:min(1024, len(b))], []byte{0}) {
		return GetFileReturnT{Contents: base64.StdEncoding.EncodeToString(b), Binary: true}
	}

	return GetFileReturnT{Contents: string(b), Binary: false}
}

var rxExtension = regexp.MustCompile(`.[a-zA-Z0-9]+$`)

func (a *WApp) GetImage(path string) string {
	if len(path) == 0 {
		return "error: empty string"
	}

	ext := strings.ToLower(rxExtension.FindString(path))
	if len(ext) == 0 {
		return "error: extension not found"
	}

	resolvedPath := path
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = string(filepath.Separator) + strings.TrimLeft(resolvedPath, string(filepath.Separator))
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		resolvedPath = filepath.Join(a.mdBaseDir, strings.TrimLeft(path, "/\\"))
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	/*if recentFile := a.notesFileForAbsolutePath(resolvedPath); recentFile != "" {
		notes.RecentListAdd(a.projRoot, recentFile)
	}*/

	base64 := base64.StdEncoding.EncodeToString(b)

	return fmt.Sprintf("data:%s;base64,%s", imageMime(ext), base64)
}

func imageMime(ext string) string {
	if ext == ".svg" {
		return "image/svg+xml"
	}
	return "image/" + ext[1:]
}

func (a *WApp) ListFiles() []string {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return []string{}
	}

	ulf := notes.ListFiles(renderer)

	a.globalNotes = ulf.PathGlobalNotes
	a.usrNotesDir = ulf.PathUserNotes
	a.projRoot = ulf.PathProjectRoot
	a.groupName = ulf.GroupName

	return ulf.Files
}

func (a *WApp) expandMappingFuncWithProject(s, projectPath string) string {
	switch s {
	case "PROJECT":
		if projectPath != "" {
			return projectPath
		}
		return a.projRoot
	case "NOTES":
		return a.usrNotesDir
	case "HOME":
		return a.homeDir
	case "GLOBAL":
		return a.globalNotes
	//case "HISTORY":
	//	return a.historyDir
	default:
		return "error"
	}
}

func (a WApp) filePath(filename string) string {
	filename = os.Expand(filename, func(s string) string {
		return a.expandMappingFuncWithProject(s, "")
	})
	if filepath.IsLocal(filename) {
		filename = a.usrNotesDir + string(filepath.Separator) + filename
	}
	return filename
}

// filePathWithProject returns the expanded file path using a specific project root.
// This uses the same expansion logic as filePath but allows overriding $PROJECT.
func (a WApp) filePathWithProject(filename string, projectPath string) string {
	filename = os.Expand(filename, func(s string) string {
		return a.expandMappingFuncWithProject(s, projectPath)
	})
	if filepath.IsLocal(filename) {
		filename = a.usrNotesDir + string(filepath.Separator) + filename
	}
	return filename
}

func (a *WApp) ResolveFilePath(filename string) string {
	return a.filePath(filename)
}

// ResolveNotesLspLanguage returns the canonical language id for a filename.
func (a *WApp) ResolveNotesLspLanguage(filename string) string {
	return lsp.ResolveLanguageIDForFile(filename)
}

// ----------------------------------------------------------------------------
// LSP document lifecycle bridge (called from JS)
// ----------------------------------------------------------------------------

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
	for _, id := range candidateIDs {
		argv = lsp.LookupArgv(config.Config.Notes.LSP, id)
		if len(argv) > 0 {
			selectedLanguageID = id
			break
		}
	}

	if len(argv) == 0 {
		return nil
	}

	alreadyRunning := a.lspManager.Has(a.projRoot, selectedLanguageID)

	sp, err := a.lspManager.GetOrStart(a.ctx, a.projRoot, selectedLanguageID, argv)
	if err != nil {
		a.notifyLspStartError(selectedLanguageID, argv, err)
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

	if err := sp.EnsureInitialized(a.ctx, a.projRoot); err != nil {
		a.notifyLspStartError(selectedLanguageID, argv, err)
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
func (a *WApp) NotesLspSemanticTokens(filePath string) []lsp.SemanticTokenItem {
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

	items, err := lsp.RequestSemanticTokens(a.ctx, t, doc.URI, doc.Content(), sp.PositionEncoding())
	if err != nil {
		log.Printf("lsp: SemanticTokens %q: %v", absPath, err)
		return nil
	}

	return items
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

// GetCurrentProject returns the absolute path of the current project root.
// This is used by the frontend to track which project a file is associated with,
// preventing issues where a file opened in one project could be overwritten
// if the user switches projects before autosave completes.
func (a *WApp) GetCurrentProject() string {
	return a.projRoot
}

func (a *WApp) NotesRecentFiles() []string {
	return notes.GetRecentList(a.projRoot)
}

func (a *WApp) GetProjectCache() *notes.ProjectCacheT {
	return notes.GetProjectCache(a.groupName)
}

func (a *WApp) SetProjectCache(ptr *notes.ProjectCacheT) {
	notes.SetProjectCache(a.groupName, ptr)
}

func (a *WApp) GetDocumentCache(filename string) *notes.DocumentCacheT {
	return notes.GetDocumentCache(a.projRoot, filename)
}

func (a *WApp) SetDocumentCache(filename string, ptr *notes.DocumentCacheT) {
	notes.SetDocumentCache(a.projRoot, filename, ptr)
}

func (a *WApp) hyperlinkMenuItems(url, text string) []types.MenuItem {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return nil
	}

	return menuhyperlink.MenuItems(renderer, url, text)
}

func (a *WApp) GetHyperlinkMenuActions(url, text string) []map[string]any {
	menuItems := a.hyperlinkMenuItems(url, text)
	out := make([]map[string]any, 0, len(menuItems))

	for i := range menuItems {
		out = append(out, map[string]any{
			"title":  menuItems[i].Title,
			"icon":   menuItems[i].Icon,
			"action": strconv.Itoa(i),
		})
	}

	return out
}

func (a *WApp) RunHyperlinkMenuAction(url, text, action string) {
	menuItems := a.hyperlinkMenuItems(url, text)
	if len(menuItems) == 0 {
		return
	}

	index, err := strconv.Atoi(strings.TrimSpace(action))
	if err != nil || index < 0 || index >= len(menuItems) {
		return
	}

	// Execute the menu item callback if it exists
	if menuItems[index].Fn != nil {
		menuItems[index].Fn()
	}
}

func (a *WApp) DisplayHyperlinkMenu(url, text string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	menu := renderer.NewContextMenu()
	menu.Append(a.hyperlinkMenuItems(url, text)...)
	menu.DisplayMenu("Hyperlink action", true)
}

// SaveFile saves a file. If projectPath is empty, it uses the current $PROJECT.
// If projectPath is set, $PROJECT in filename is expanded against projectPath,
// which keeps autosave bound to the project where the file was opened.
func (a *WApp) SaveFile(filename, contents, projectPath string) error {
	var absPath string
	if projectPath == "" {
		absPath = a.filePath(filename)
	} else {
		absPath = a.filePathWithProject(filename, projectPath)
	}

	_, statErr := os.Stat(absPath)
	created := os.IsNotExist(statErr)

	if err := os.WriteFile(absPath, []byte(contents), 0644); err != nil {
		return err
	}

	if created {
		a.notifyLspWorkspaceFiles(func(t *lsp.Transport) error {
			return lsp.NotifyDidCreateFiles(t, []string{absPath})
		})
	}

	return nil
}

func (a *WApp) SaveBinaryFile(filename, base64Contents string) error {
	filename = a.filePath(filename)

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("create directory for binary file: %w", err)
	}

	b, err := base64.StdEncoding.DecodeString(base64Contents)
	if err != nil {
		return fmt.Errorf("decode base64 file contents: %w", err)
	}

	return os.WriteFile(filename, b, 0644)
}

func (a *WApp) SaveImageDialog(defaultFilename string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Image",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Images",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.svg",
			},
		},
	})
	if err != nil {
		return "", err
	}

	return path, nil
}

func (a *WApp) WindowPrint() {
	runtime.WindowPrint(a.ctx)
}

type ClipboardData struct {
	Text  string `json:"text"`
	Image string `json:"image"`
}

func (a *WApp) GetFileMetaMarkdown(filename string) string {
	return metamd.DocumentForPath(a.filePath(filename))
}

// GetClipboardData returns clipboard data as either text or a base64-encoded PNG image.
func (a *WApp) GetClipboardData() ClipboardData {
	b := clipboard.Read(clipboard.FmtImage)
	if len(b) != 0 {
		return ClipboardData{Image: base64.StdEncoding.EncodeToString(b)}
	}

	return ClipboardData{Text: string(clipboard.Read(clipboard.FmtText))}
}

func (a *WApp) RenameFile(oldPath, newPath string) error {
	oldAbsPath := a.filePath(oldPath)
	newAbsPath := a.filePath(newPath)
	err := os.Rename(oldAbsPath, newAbsPath)
	if err != nil {
		return err
	}

	err = notes.RecentListRename(a.projRoot, oldPath, newPath)
	if err != nil {
		return err
	}

	a.notifyLspWorkspaceFiles(func(t *lsp.Transport) error {
		return lsp.NotifyDidRenameFiles(t, [][2]string{{oldAbsPath, newAbsPath}})
	})

	return nil
}

func (a *WApp) DeleteFile(filename string) error {
	absPath := a.filePath(filename)
	err := os.Remove(absPath)
	if err != nil {
		return err
	}

	err = notes.RecentListDelete(a.projRoot, filename)
	if err != nil {
		return err
	}

	a.notifyLspWorkspaceFiles(func(t *lsp.Transport) error {
		return lsp.NotifyDidDeleteFiles(t, []string{absPath})
	})

	return nil
}

func (a *WApp) GetCustomRegexp() []map[string]string {
	var result []map[string]string
	for _, custom := range config.Config.Terminal.Widgets.AutoHyperlink.CustomRegexp {
		if custom.Rx == nil {
			continue
		}
		result = append(result, map[string]string{
			"pattern": custom.Rx.String(),
			"link":    custom.Link,
		})
	}
	return result
}

func (a *WApp) SendToTerminal(content string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.ActiveTile().GetTerm().Reply([]byte(content))
}

// SwaggerRequest executes an HTTP request for a Swagger/OpenAPI endpoint.
// All request logic lives in utils/swagger; this method is a thin binding.
func (a *WApp) SwaggerRequest(req swagger.RequestT) swagger.ResponseT {
	return swagger.Execute(a.ctx, req)
}

// ViewFileInNotes displays a popup menu (in Go) to select a file to view in the Notes pane.
// On selection it emits:
//  1. "viewFileInNotesOpen" — tells the frontend to load the chosen file.
//  2. "terminalActivateAuxTab" with id "notes" — switches to the Notes tab if it is
//     currently registered as an auxiliary terminal pane tab.
func (a *WApp) ViewFileInNotes() {
	files := a.ListFiles()
	if len(files) == 0 || a.ctx == nil {
		return
	}

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	onSelect := func(i int) {
		if i < 0 || i >= len(files) {
			return
		}
		filename := files[i]

		// If Notes is registered as an auxiliary tab, activate it.
		/*for _, tab := range renderer.TerminalPaneTabs() {
			if tab.ID == "notes" {
				renderer.ActivateTerminalPaneTab("notes")
				break
			}
		}*/

		// Tell the frontend to open the file in the Notes pane.
		runtime.EventsEmit(a.ctx, "viewFileInNotesOpen", filename)
	}

	renderer.DisplayMenu("Select file to view in Notes", files, nil, onSelect, nil)
}

func (a *WApp) GetAppTitle() string { return appTitle() }

// ShowCommandPalette opens the command palette and sends all options to the
// frontend in one payload. Filtering is done in JS.
func (a *WApp) ShowCommandPalette() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}
	renderer.ShowCommandPalette()
}

// CommandPaletteSelect executes the chosen item via the renderer.
func (a *WApp) CommandPaletteSelect(index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}
	renderer.CommandPaletteSelect(index)
}

func (a *WApp) AskAI(callerType, filename, contents string) {
	switch callerType {
	case "notesDocument":
		renderer, ok := renderwebkit.CurrentRenderer()
		if !ok {
			return
		}
		tile := renderer.ActiveTile()
		if tile == nil {
			return
		}
		agt := agent.Get(tile.Id())
		ai.ExplainDoc(agt, filename, contents)
	default:
		return
	}
}

// --------------------

func (a *WApp) startup(ctx context.Context) {
	a.ctx = ctx
	hotkeys.SetTerminalFocusFn(a.FocusTerminalPane)

	globalhotkeys.Register(func(key string) {
		switch key {
		case "F12":
			a.WindowShowHide()
		}
	})
}

func (a *WApp) domReady(ctx context.Context) {
	go a.startTerminalWindow()
}

func appTitle() string {
	return fmt.Sprintf("%s: %s", app.Name(), app.TagLine())
}

//go:embed build/appicon.png
var appIcon []byte

func startWails() {
	wapp := NewWailsApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             appTitle(),
		AlwaysOnTop:       config.Config.Window.AlwaysOnTop,
		HideWindowOnClose: true,
		WindowStartState:  options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: wailsAssets,
		},
		BackgroundColour: &options.RGBA{
			R: types.SGR_DEFAULT.Bg.Red,
			G: types.SGR_DEFAULT.Bg.Green,
			B: types.SGR_DEFAULT.Bg.Blue,
			A: 255,
		},
		OnStartup:  wapp.startup,
		OnDomReady: wapp.domReady,
		Bind:       []any{wapp},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
			About: &mac.AboutInfo{
				Title:   app.Name(),
				Message: fmt.Sprintf("%s\n\nVersion: %s (%s)\nBuild Date: %s\n\nCopyright: %s\nSoftware License: %s", app.TagLine(), app.Version(), app.Branch(), app.BuildDate(), app.Copyright(), app.License()),
				Icon:    appIcon,
			},
		},
		Linux: &linux.Options{
			Icon:        appIcon,
			ProgramName: app.Name(),
		},
		Windows: &windows.Options{
			WindowClassName: app.Name(),
		},

		//BindingsAllowedOrigins: "*",
	})

	if err != nil {
		//closeHotkeys()
		panic(err)
	}
}
