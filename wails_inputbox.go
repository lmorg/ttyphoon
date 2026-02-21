package main

import (
	"context"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// App struct
type WInputBox struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewWInputBox() *WInputBox {
	return &WInputBox{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *WInputBox) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *WInputBox) Greet(name string) string {
	/*state := struct {
		Cancel bool
		Value  string
	}{
		Value: name,
	}*/

	//wailswindow.Close(&state)
	return name
}

// --------------------

func wInputBox() {
	// Create an instance of the app structure
	app := NewWInputBox()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "TTYphoon",
		Width:  1024,
		Height: 300,
		AssetServer: &assetserver.Options{
			Assets: wailsAssets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}

}
