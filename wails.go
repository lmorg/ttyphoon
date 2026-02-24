package main

import (
	"context"
	"embed"
	"os"

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
	Payload *dispatcher.PayloadT
	//window  dispatcher.WindowNameT
}

// NewApp creates a new App application struct
func NewWailsApp(window dispatcher.WindowNameT, payload *dispatcher.PayloadT) *WApp {
	return &WApp{
		//Window:  window,
		Payload: payload,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *WApp) startup(ctx context.Context) {
	a.ctx = ctx

	runtime.WindowSetPosition(ctx, int(a.Payload.Window.Pos.X), int(a.Payload.Window.Pos.Y))
}

//func (a *WApp) shutdown(ctx context.Context) { os.Exit(0) }

func (a *WApp) GetPayload() string {
	return os.Getenv(dispatcher.ENV_PARAMETERS)
}

func (a *WApp) GetWindowStyle() dispatcher.WindowStyleT {
	return a.Payload.Window
}

func (a *WApp) GetParameters() any {
	return a.Payload.Parameters
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

// --------------------

func startWails(window dispatcher.WindowNameT) {
	payload := &dispatcher.PayloadT{}

	switch window {
	case dispatcher.WindowInputBox:
		//payload.Parameters = dispatcher.PInputBoxT{}
	default:
		//payload.Parameters = "undef"
	}

	err := dispatcher.GetPayload(payload)
	if err != nil {
		panic(err)
	}

	// Create an instance of the app structure
	app := NewWailsApp(window, payload)

	// Create application with options
	err = wails.Run(&options.App{
		Title:       "TTYphoon",
		Width:       int(payload.Window.Size.X),
		Height:      int(payload.Window.Size.Y),
		Frameless:   payload.Window.Frameless,
		AlwaysOnTop: payload.Window.AlwaysOnTop,
		AssetServer: &assetserver.Options{
			Assets: wailsAssets,
		},
		BackgroundColour: &options.RGBA{
			R: payload.Window.Fg.Red,
			G: payload.Window.Fg.Green,
			B: payload.Window.Fg.Blue,
			A: uint8(config.Config.Window.Opacity/100) * 254,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		panic(err)
	}

}
