package main

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	ttyphoon "github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	"github.com/lmorg/ttyphoon/window/backend"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var wailsAssets embed.FS

type pStructT struct {
	inputBox *dispatcher.PInputBoxT
	markdown *dispatcher.PMarkdownT
	preview  *dispatcher.PPreviewT
	notes    *dispatcher.PNotesT
}

// App struct
type WApp struct {
	ctx         context.Context
	payload     *dispatcher.PayloadT
	window      dispatcher.WindowTypeT
	mdBaseDir   string
	projRoot    string
	usrNotesDir string
	homeDir     string
	globalNotes string
	historyDir  string
	visible     bool
	ipc         *dispatcher.IpcT
	msgPipe     chan *dispatcher.IpcMessageT
	ps          *pStructT
	notesKills  map[string]func()
}

var WWindowTsBindings = []struct {
	Value  dispatcher.WindowTypeT
	TSName string
}{
	{dispatcher.WindowTerminal, "terminal"},
	{dispatcher.WindowInputBox, "inputBox"},
	{dispatcher.WindowMarkdown, "markdown"},
	{dispatcher.WindowPreview, "preview"},
	{dispatcher.WindowNotes, "notes"},
}

// NewApp creates a new App application struct
func NewWailsApp(window dispatcher.WindowTypeT, payload *dispatcher.PayloadT, ps *pStructT) *WApp {
	a := &WApp{
		window:     window,
		payload:    payload,
		msgPipe:    make(chan *dispatcher.IpcMessageT),
		ps:         ps,
		visible:    true,
		notesKills: map[string]func(){},
	}

	a.homeDir, _ = os.UserHomeDir()

	switch window {
	case dispatcher.WindowNotes:
		a.projRoot = ps.notes.ProjectRoot
		if a.projRoot == "" {
			a.projRoot, _ = os.Getwd()
		}

		sep := string(filepath.Separator)
		a.usrNotesDir = ps.notes.UserNotes
		a.globalNotes = filepath.Clean(fmt.Sprintf("%s%s..%s", a.usrNotesDir, sep, sep)) + sep
		a.historyDir = a.globalNotes + "history"
	}

	return a
}

func (a *WApp) GetWindowType() string {
	return string(a.window)
}

func (a *WApp) GetWindowStyle() dispatcher.WindowStyleT {
	return a.payload.Window
}

func (a *WApp) GetParameters() any {
	return a.payload.Parameters
}

/*func (a *WApp) GetTerminalDrawOps() []renderwebkit.DrawCommand {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return []renderwebkit.DrawCommand{}
	}

	return renderer.PopDrawCommands()
}*/

func (a *WApp) GetTerminalGlyphSize() *types.XY {
	renderer, ok := renderwebkit.CurrentRenderer()
	if ok {
		glyphSize := renderer.GetGlyphSize()
		if glyphSize != nil {
			return glyphSize
		}
	}

	return renderwebkit.GetConfiguredGlyphSize()
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

func (a *WApp) TerminalRequestRedraw() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.TriggerRedraw()
}

func (a *WApp) startTerminalWindow() {
	renderer, size := backend.Initialise()

	tmuxClient, err := tmux.NewStartSession(renderer, size, tmux.START_ATTACH_SESSION)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "no sessions") {
			log.Println(err)
			return
		}

		tmuxClient, err = tmux.NewStartSession(renderer, size, tmux.START_NEW_SESSION)
		if err != nil {
			log.Println(err)
			return
		}
	}

	backend.Start(renderer, tmuxClient.GetTermTiles(), tmuxClient, a.ctx)
}

func (a *WApp) SendIpc(eventName string, parameters map[string]string) {
	err := a.ipc.Send(&dispatcher.IpcMessageT{
		EventName:  eventName,
		Parameters: parameters,
	})
	if err != nil {
		log.Println(err)
	}
}

func (a *WApp) WindowShow() {
	a.visible = true
	runtime.WindowSetPosition(a.ctx, 0, 0)
	runtime.WindowShow(a.ctx)
	runtime.WindowSetPosition(a.ctx, 0, 0)
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

func (a *WApp) SendVisualInputBox(value string, notesCheckbox bool) {
	err := a.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "ok",
		Parameters: map[string]string{
			"value":        value,
			"notesDisplay": fmt.Sprintf("%v", notesCheckbox),
		},
	})
	if err != nil {
		log.Println(err.Error())
	}

	runtime.Quit(a.ctx)
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
	a.notesKills[id] = kill

	go jupyter.RunNote(ctx, id, code, language, ch)

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
		delete(a.notesKills, id)
	}()
}

func (a *WApp) StopNote(id string) {
	fn, ok := a.notesKills[id]
	if !ok {
		log.Printf("cannot stop note %s because no kill function exists", id)
	}

	fn()

	runtime.EventsEmit(a.ctx, "noteRun", map[string]string{
		"blockId": id,
		"output":  "[process killed]",
		"isError": fmt.Sprintf("%v", true),
	})
}

func (a *WApp) GetMarkdown(filename string) string {
	filename = a.filePath(filename)

	f, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return err.Error()
	}

	b, err := io.ReadAll(f)
	if err != nil {
		log.Println(err)
		return err.Error()
	}

	a.mdBaseDir = filepath.Dir(filename)

	return string(b)
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

	if path[0] != '/' {
		// TODO: this isn't Windows compatible
		path = a.mdBaseDir + "/" + path
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

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
	var files []string

	cache.Read(cache.NS_NOTESW_FILES, a.usrNotesDir, &files)
	cache.Write(cache.NS_NOTESW_FILES, a.usrNotesDir, &files, cache.Days(365))

	files = append(files, listFiles(a.globalNotes, "GLOBAL")...)
	files = append(files, listFiles(a.usrNotesDir, "NOTES")...)
	files = append(files, listFiles(a.historyDir, "HISTORY")...)

	if a.projRoot == "" {
		return files
	}

	err := filepath.WalkDir(a.projRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.Println(err)
			return nil
		}
		if d.IsDir() {
			if len(d.Name()) == 0 || d.Name()[0] == '.' || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if len(d.Name()) == 0 || d.Name()[0] == '.' {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			filename := strings.Replace(path, a.projRoot, "$PROJ", 1)
			files = append(files, filename)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return files
}

func listFiles(path string, varName string) (files []string) {
	glob, err := filepath.Glob(fmt.Sprintf("%s/*.md", path))
	if err != nil {
		log.Println(err)
		return
	}
	replace := fmt.Sprintf("$%s/", varName)
	for i := range glob {
		files = append(files, strings.Replace(glob[i], path, replace, 1))
	}
	return
}

func (a *WApp) AddToFileList(filename string) {
	var files []string

	cache.Read(cache.NS_NOTESW_FILES, a.usrNotesDir, &files)
	defer cache.Write(cache.NS_NOTESW_FILES, a.usrNotesDir, &files, cache.Days(365))

	if slices.Contains(files, filename) {
		return
	}

	files = append([]string{filename}, files...)
}

func (a *WApp) expandMappingFunc(s string) string {
	switch s {
	case "PROJ":
		return a.projRoot
	case "NOTES":
		return a.usrNotesDir
	case "HOME":
		return a.homeDir
	case "GLOBAL":
		return a.globalNotes
	case "HISTORY":
		return a.historyDir
	default:
		return "error"
	}
}

func (a WApp) filePath(filename string) string {
	filename = os.Expand(filename, a.expandMappingFunc)
	if filepath.IsLocal(filename) {
		filename = a.usrNotesDir + string(filepath.Separator) + filename
	}
	return filename
}

func (a *WApp) SaveFile(filename, contents string) error {
	filename = a.filePath(filename)
	return os.WriteFile(filename, []byte(contents), 0644)
}

func (a *WApp) RenameFile(oldPath, newPath string) error {
	oldPath = a.filePath(oldPath)
	newPath = a.filePath(newPath)
	return os.Rename(oldPath, newPath)
}

func (a *WApp) DeleteFile(filename string) error {
	filename = a.filePath(filename)
	return os.Remove(filename)
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

// --------------------

func (a *WApp) ipcRespFunc(msg *dispatcher.IpcMessageT) {
	a.msgPipe <- msg
}

func (a *WApp) startup(ctx context.Context) {
	a.ctx = ctx

	runtime.WindowSetPosition(ctx, int(a.payload.Window.Pos.X), int(a.payload.Window.Pos.Y))
}

func (a *WApp) domReady(ctx context.Context) {
	go func() {
		for msg := range a.msgPipe {
			switch {
			case msg.Error != nil:
				runtime.EventsEmit(a.ctx, "error", msg.Error)
			case msg.EventName == "focus":
				//runtime.WindowShow(ctx)
				a.WindowShow()
			case msg.EventName == "notesToggleShowHide":
				a.WindowShowHide()
			case msg.EventName == "notesFocus":
				a.WindowShow()
				fallthrough
			case msg.EventName == "notesUpdatePaths":
				a.projRoot = msg.Parameters["projectRoot"]
				a.usrNotesDir = msg.Parameters["userNotes"]
				runtime.EventsEmit(a.ctx, "updateTitle", msg.Parameters["title"])
				runtime.WindowExecJS(a.ctx, `window.refreshFiles();`)
			default:
				runtime.EventsEmit(a.ctx, msg.EventName, msg.Parameters)
			}
		}
	}()

	switch a.window {
	case dispatcher.WindowTerminal:
		go a.startTerminalWindow()
	case dispatcher.WindowHistory:
		err := a.ipc.Send(&dispatcher.IpcMessageT{EventName: "focus"})
		if err != nil {
			log.Println(err)
		}
	case dispatcher.WindowNotes:
		runtime.EventsEmit(a.ctx, "updateTitle", a.ps.notes.Title)
	}
}

func (a *WApp) beforeClose(ctx context.Context) bool {
	switch a.window {
	case dispatcher.WindowHistory:
		err := a.ipc.Send(&dispatcher.IpcMessageT{EventName: "closeMenu"})
		if err != nil {
			log.Println(err)
		}
	case dispatcher.WindowNotes:
		a.WindowHide()
		return true
	}

	return false
}

func startWails(window dispatcher.WindowTypeT) {
	payload := new(dispatcher.PayloadT)
	pStruct := &pStructT{}
	switch window {
	case dispatcher.WindowInputBox:
		pStruct.inputBox = new(dispatcher.PInputBoxT)
		payload.Parameters = pStruct.inputBox
	case dispatcher.WindowMarkdown:
		pStruct.markdown = new(dispatcher.PMarkdownT)
		payload.Parameters = pStruct.markdown
	case dispatcher.WindowPreview:
		pStruct.preview = new(dispatcher.PPreviewT)
		payload.Parameters = pStruct.preview
	case dispatcher.WindowNotes:
		pStruct.notes = new(dispatcher.PNotesT)
		payload.Parameters = pStruct.notes
	default:
		//payload.Parameters = make(map[string]string)
	}

	err := dispatcher.GetPayload(payload)
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}

	// Create an instance of the app structure
	app := NewWailsApp(window, payload, pStruct)

	ipc, err := dispatcher.ClientConnect(app.ipcRespFunc)
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	app.ipc = ipc

	// Create application with options
	err = wails.Run(&options.App{
		Title:             fmt.Sprintf("%s: %s", ttyphoon.Name, payload.Window.Title),
		Width:             int(payload.Window.Size.X),
		Height:            int(payload.Window.Size.Y),
		Frameless:         payload.Window.Frameless,
		AlwaysOnTop:       payload.Window.AlwaysOnTop,
		StartHidden:       payload.Window.StartHidden,
		HideWindowOnClose: payload.Window.HideOnClose,
		AssetServer: &assetserver.Options{
			Assets: wailsAssets,
		},
		BackgroundColour: &options.RGBA{
			R: payload.Window.Colours.Fg.Red,
			G: payload.Window.Colours.Fg.Green,
			B: payload.Window.Colours.Fg.Blue,
			A: 255, //uint8(config.Config.Window.Opacity/100) * 255,
		},
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
		OnBeforeClose: app.beforeClose,
		Bind:          []interface{}{app},
		EnumBind:      []interface{}{WWindowTsBindings},
		/*Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
		},

		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},*/

		//BindingsAllowedOrigins: "*",
	})

	if err != nil {
		panic(err)
	}
}
