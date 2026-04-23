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
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	globalhotkeys "github.com/lmorg/ttyphoon/utils/global_hotkeys"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	menuhyperlink "github.com/lmorg/ttyphoon/utils/menu_hyperlink"
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
	usrNotesDir   string
	homeDir       string
	globalNotes   string
	historyDir    string
	visible       bool
	notesKills    map[string]func()
	notesStickies map[string]types.Notification
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
		visible:       true,
		notesKills:    map[string]func(){},
		notesStickies: map[string]types.Notification{},
		homeDir:       xdg.Home,
		projRoot:      findProjectRoot(""),
		//usrNotesDir: userDocs(),
		globalNotes: docsDir("notes"),
	}

	return a
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
		if existing, ok := a.notesStickies[id]; ok {
			existing.Close()
			delete(a.notesStickies, id)
		}
		sticky := renderer.DisplaySticky(notifType, message, func() {})
		a.notesStickies[id] = sticky

	case "terminal-sticky-update":
		id := strings.TrimSpace(parameters["id"])
		message := strings.TrimSpace(parameters["message"])
		if id == "" || message == "" {
			return
		}
		if sticky, ok := a.notesStickies[id]; ok {
			sticky.SetMessage(message)
		}

	case "terminal-sticky-close":
		id := strings.TrimSpace(parameters["id"])
		if id == "" {
			return
		}
		if sticky, ok := a.notesStickies[id]; ok {
			sticky.Close()
			delete(a.notesStickies, id)
		}
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

	path = string(filepath.Separator) + path
	if _, err := os.Stat(path); err != nil {
		path = a.mdBaseDir + path
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
	files := []string{}

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return files
	}

	tile := renderer.ActiveTile()
	if tile == nil {
		return files
	}

	a.projRoot = findProjectRoot(tile.Pwd())
	a.globalNotes = docsDir("notes")
	a.usrNotesDir = a.globalNotes + tile.GroupName() + "/"

	cache.Read(cache.NS_NOTESW_FILES, a.usrNotesDir, &files)
	cache.Write(cache.NS_NOTESW_FILES, a.usrNotesDir, &files, cache.Days(365))

	files = append(files, listFiles(a.globalNotes, "GLOBAL")...)
	files = append(files, listFiles(a.usrNotesDir, "NOTES")...)
	//files = append(files, listFiles(a.historyDir, "HISTORY")...)

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

		//if strings.HasSuffix(strings.ToLower(d.Name()), ".md") || strings.HasSuffix(strings.ToLower(d.Name()), ".json") ||
		//	strings.HasSuffix(strings.ToLower(d.Name()), ".yml") || strings.HasSuffix(strings.ToLower(d.Name()), ".yaml") {
		filename := strings.Replace(path, a.projRoot, "$PROJECT", 1)
		files = append(files, filename)
		//}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return files
}

func listFiles(path string, varName string) (files []string) {
	files = listFilesWithGlob(path, varName, "*.md")
	files = append(files, listFilesWithGlob(path, varName, "*.json")...)
	files = append(files, listFilesWithGlob(path, varName, "*.yaml")...)
	files = append(files, listFilesWithGlob(path, varName, "*.yml")...)
	slices.Sort(files)
	return files
}

func listFilesWithGlob(path string, varName string, pattern string) (files []string) {
	if path == "" {
		return []string{}
	}

	glob, err := filepath.Glob(fmt.Sprintf("%s/%s", path, pattern))
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
	case "PROJECT":
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

func (a *WApp) ResolveFilePath(filename string) string {
	return a.filePath(filename)
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

func (a *WApp) SaveFile(filename, contents string) error {
	filename = a.filePath(filename)
	return os.WriteFile(filename, []byte(contents), 0644)
}

func (a *WApp) SaveBinaryFile(filename, base64Contents string) error {
	filename = a.filePath(filename)

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

// GetClipboardData returns clipboard data as either text or a base64-encoded PNG image.
func (a *WApp) GetClipboardData() ClipboardData {
	b := clipboard.Read(clipboard.FmtImage)
	if len(b) != 0 {
		return ClipboardData{Image: base64.StdEncoding.EncodeToString(b)}
	}

	return ClipboardData{Text: string(clipboard.Read(clipboard.FmtText))}
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
		for _, tab := range renderer.TerminalPaneTabs() {
			if tab.ID == "notes" {
				renderer.ActivateTerminalPaneTab("notes")
				break
			}
		}

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

// --------------------

func (a *WApp) startup(ctx context.Context) {
	a.ctx = ctx

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
