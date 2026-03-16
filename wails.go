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

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	ttyphoon "github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	globalhotkeys "github.com/lmorg/ttyphoon/utils/global_hotkeys"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	"github.com/lmorg/ttyphoon/window/backend"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	mac "github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var wailsAssets embed.FS

// App struct
type WApp struct {
	ctx         context.Context
	mdBaseDir   string
	projRoot    string
	usrNotesDir string
	homeDir     string
	globalNotes string
	historyDir  string
	visible     bool
	notesKills  map[string]func()
}

func docsDir(function string) string {
	path := fmt.Sprintf("%s/Documents/%s/%s/", xdg.Home, app.DirName, function)

	/*err :=*/
	_ = os.MkdirAll(path, 0700)
	/*if err != nil {
		return err
	}*/

	return path
}

func findProjectRoot(cwd string) string {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	pwd := cwd
	home, _ := os.UserHomeDir()
	for {
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			return pwd
		}
		parent := filepath.Dir(cwd)
		if parent == pwd || parent == home {
			return ""
		}
		pwd = parent
	}
}

// NewApp creates a new App application struct
func NewWailsApp() *WApp {
	a := &WApp{
		visible:    true,
		notesKills: map[string]func(){},
		homeDir:    xdg.Home,
		projRoot:   findProjectRoot(""),
		//usrNotesDir: userDocs(),
		globalNotes: docsDir("notes"),
	}

	return a
}

type WindowStyleT struct {
	Colours          *ColoursT `json:"colors"`
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
			Selection:     *types.COLOR_SELECTION,
			Link:          *types.SGR_COLOR_BLUE,
			Error:         *types.COLOR_ERROR,
		},
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

func (a *WApp) TerminalRequestRedraw() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.TriggerRedraw()
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

func (a *WApp) TerminalInputBoxSubmit(id int64, value string, isOk bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.InputBoxSubmit(id, value, isOk)
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
	// todo
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

func (a *WApp) GetAppName() string {
	return app.Name
}

// --------------------

func (a *WApp) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *WApp) domReady(ctx context.Context) {
	/*go func() {
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
	}()*/

	go a.startTerminalWindow()
}

func startWails() {
	app := NewWailsApp()

	/*dispatcherCallback := func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			_, _ = os.Stderr.WriteString(msg.Error.Error())
		}

		switch msg.EventName {
		case "started":
			log.Println("Global hotkeys registered successfully")
		case "F12":
			app.WindowShowHide()
		}
	}
	var closeHotkeys func()
	_, closeHotkeys = dispatcher.StartApp(dispatcher.AppGlobalHotkeys, dispatcherCallback)
	*/

	hotkeyCallback := func(key string) {
		switch key {
		case "F12":
			app.WindowShowHide()
		}
	}

	globalhotkeys.Register(hotkeyCallback)

	// Create application with options
	err := wails.Run(&options.App{
		Title: ttyphoon.Name,
		//Width:             int(payload.Window.Size.X),
		//Height:            int(payload.Window.Size.Y),
		AlwaysOnTop:       true,
		HideWindowOnClose: true,
		WindowStartState:  options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: wailsAssets,
		},
		/*BackgroundColour: &options.RGBA{
			R: payload.Window.Colours.Fg.Red,
			G: payload.Window.Colours.Fg.Green,
			B: payload.Window.Colours.Fg.Blue,
			A: 255, //uint8(config.Config.Window.Opacity/100) * 255,
		},*/
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		Bind:             []any{app},
		BackgroundColour: &options.RGBA{0, 0, 0, 0},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			//WindowIsTranslucent:  true,
			//WebviewIsTransparent: true,
		},
		/*Linux: &linux.Options{
			WindowIsTranslucent: true,
		},

		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},*/

		//BindingsAllowedOrigins: "*",
	})

	if err != nil {
		//closeHotkeys()
		panic(err)
	}
}
