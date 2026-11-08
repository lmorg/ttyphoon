package window

import (
	"io/fs"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/window/frontend/browser"
	mainwindow "github.com/lmorg/ttyphoon/window/frontend/main"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func StartWails(wailsAssets fs.FS, appIcon []byte) {
	if err := browser.InitializeUserAgent(); err != nil {
		panic(err)
	}
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

	menu := application.NewMenu()
	menu.AddSubmenu(app.Name()).AddRole(application.About)
	wailsApp.Menu.SetApplicationMenu(menu)

	mainwindow.NewWindow(wailsApp, wapp)
	if err := wailsApp.Run(); err != nil {
		panic(err)
	}
}
