package main

import (
	"embed"

	"github.com/lmorg/ttyphoon/app"
	mainwindow "github.com/lmorg/ttyphoon/window/frontend/main"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var wailsAssets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func startWails() {
	wapp := mainwindow.NewWailsApp()
	wailsApp := application.New(application.Options{
		Name:        app.Name(),
		Description: app.TagLine(),
		Icon:        appIcon,
		Services:    []application.Service{application.NewService(wapp)},
		Assets: application.AssetOptions{
			Handler: mainwindow.AssetHandler(wapp, application.AssetFileServerFS(wailsAssets)),
		},
		Linux:      application.LinuxOptions{ProgramName: app.Name()},
		Windows:    application.WindowsOptions{WndClass: app.Name()},
		OnShutdown: func() { mainwindow.Shutdown(wapp) },
	})
	mainwindow.NewWindow(wailsApp, wapp)
	if err := wailsApp.Run(); err != nil {
		panic(err)
	}
}
