package main

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	ttyphoon "github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var wailsAssets embed.FS

// App struct
type WApp struct {
	ctx     context.Context
	payload *dispatcher.PayloadT
	window  dispatcher.WindowTypeT
	dir     string
}

var WWindowTsBindings = []struct {
	Value  dispatcher.WindowTypeT
	TSName string
}{
	{dispatcher.WindowSdl, "sdl"},
	{dispatcher.WindowInputBox, "inputBox"},
	{dispatcher.WindowMarkdown, "markdown"},
}

// NewApp creates a new App application struct
func NewWailsApp(window dispatcher.WindowTypeT, payload *dispatcher.PayloadT) *WApp {
	return &WApp{
		window:  window,
		payload: payload,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *WApp) startup(ctx context.Context) {
	a.ctx = ctx

	runtime.WindowSetPosition(ctx, int(a.payload.Window.Pos.X), int(a.payload.Window.Pos.Y))
}

func (a *WApp) GetWindowType() string {
	return string(a.window)
}

func (a *WApp) GetPayload() string {
	return os.Getenv(dispatcher.ENV_PARAMETERS)
}

func (a *WApp) GetWindowStyle() dispatcher.WindowStyleT {
	return a.payload.Window
}

func (a *WApp) GetParameters() any {
	return a.payload.Parameters
}

func (a *WApp) VisualInputBox(name string) string {
	response := &dispatcher.RInputBoxT{Value: name}
	err := dispatcher.Response(response)
	if err != nil {
		return err.Error()
	}

	runtime.Quit(a.ctx)
	return ""
}

func (a *WApp) CallOpen(path string) string {
	cmd := exec.Command("open", path)
	err := cmd.Start()
	if err != nil {
		return err.Error()
	}
	return ""
}

func (a *WApp) GetMarkdown(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return err.Error()
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return err.Error()
	}

	a.dir = filepath.Dir(path)

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
	ext = ext[1:]

	if path[0] != '/' {
		// warning, this isn't Windows compatible
		path = a.dir + "/" + path
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

	return fmt.Sprintf("data:image/%s;base64,%s", ext, base64)
}

// --------------------

func startWails(window dispatcher.WindowTypeT) {
	payload := &dispatcher.PayloadT{}

	// Create an instance of the app structure
	app := NewWailsApp(window, payload)

	err := dispatcher.GetPayload(payload)
	if err != nil {
		panic(err)
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:       fmt.Sprintf("%s: %s", ttyphoon.Name, payload.Window.Title),
		Width:       int(payload.Window.Size.X),
		Height:      int(payload.Window.Size.Y),
		Frameless:   payload.Window.Frameless,
		AlwaysOnTop: payload.Window.AlwaysOnTop,
		AssetServer: &assetserver.Options{
			Assets: wailsAssets,
		},
		BackgroundColour: &options.RGBA{
			R: payload.Window.Colours.Fg.Red,
			G: payload.Window.Colours.Fg.Green,
			B: payload.Window.Colours.Fg.Blue,
			A: uint8(config.Config.Window.Opacity/100) * 254,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		EnumBind: []interface{}{
			WWindowTsBindings,
		},
	})

	if err != nil {
		panic(err)
	}

}
