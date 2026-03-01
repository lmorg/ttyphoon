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
	"strings"

	ttyphoon "github.com/lmorg/ttyphoon/app"
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
	ipc     *dispatcher.IpcT
	msgPipe chan *dispatcher.IpcMessageT
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
		msgPipe: make(chan *dispatcher.IpcMessageT),
	}
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

func (a *WApp) SendIpc(eventName string, parameters map[string]string) {
	err := a.ipc.Send(&dispatcher.IpcMessageT{
		EventName:  eventName,
		Parameters: parameters,
	})
	if err != nil {
		log.Println(err)
	}
}

func (a *WApp) VisualInputBox(value string) {
	err := a.ipc.Send(&dispatcher.IpcMessageT{
		EventName:  "ok",
		Parameters: map[string]string{"value": value},
	})
	if err != nil {
		log.Println(err.Error())
	}

	runtime.Quit(a.ctx)
}

func (a *WApp) GetMarkdown(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return err.Error()
	}

	b, err := io.ReadAll(f)
	if err != nil {
		log.Println(err)
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

	if path[0] != '/' {
		// TODO: this isn't Windows compatible
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

	return fmt.Sprintf("data:%s;base64,%s", imageMime(ext), base64)
}

func imageMime(ext string) string {
	if ext == ".svg" {
		return "image/svg+xml"
	}
	return "image/" + ext[1:]
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
				runtime.Show(ctx)
			default:
				runtime.EventsEmit(a.ctx, msg.EventName, msg.Parameters)
			}
		}
	}()

	switch a.window {
	case dispatcher.WindowHistory:
		err := a.ipc.Send(&dispatcher.IpcMessageT{EventName: "focus"})
		if err != nil {
			log.Println(err)
		}
	case dispatcher.WindowInputBox:
		//runtime.EventsEmit(a.ctx, "autoGrow")
	}
}

func (a *WApp) beforeClose(ctx context.Context) bool {
	switch a.window {
	case dispatcher.WindowHistory:
		err := a.ipc.Send(&dispatcher.IpcMessageT{EventName: "closeMenu"})
		if err != nil {
			log.Println(err)
		}
	}

	return false
}

func startWails(window dispatcher.WindowTypeT) {
	payload := &dispatcher.PayloadT{}

	// Create an instance of the app structure
	app := NewWailsApp(window, payload)

	ipc, err := dispatcher.ClientConnect(app.ipcRespFunc)
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	app.ipc = ipc

	err = dispatcher.GetPayload(payload)
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:       fmt.Sprintf("%s: %s", ttyphoon.Name, payload.Window.Title),
		Width:       int(payload.Window.Size.X),
		Height:      int(payload.Window.Size.Y),
		Frameless:   payload.Window.Frameless,
		AlwaysOnTop: payload.Window.AlwaysOnTop,
		StartHidden: payload.Window.StartHidden,
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
