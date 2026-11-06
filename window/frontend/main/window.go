package mainwindow

import (
	"fmt"
	"sync"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func NewWindow(wailsApp *application.App, wapp *WApp) *application.WebviewWindow {
	wailsApp.Event.On("ttyphoon:frontend-ready", func(*application.CustomEvent) {
		go wapp.startTerminalWindow()
	})
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       appTitle(),
		AlwaysOnTop: config.Config.Window.AlwaysOnTop,
		StartState:  application.WindowStateMaximised,
		BackgroundColour: application.NewRGBA(
			types.SGR_DEFAULT.Bg.Red,
			types.SGR_DEFAULT.Bg.Green,
			types.SGR_DEFAULT.Bg.Blue,
			255,
		),
		Mac: application.MacWindow{TitleBar: application.MacTitleBarHidden},
	})
	wapp.uiRuntime = &wailsDesktopRuntime{app: wailsApp, window: window}
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		wapp.setVisible(false)
		window.Hide()
	})
	var maximizeOnFirstShow sync.Once
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		maximizeOnFirstShow.Do(func() {
			application.InvokeAsync(func() {
				if !window.IsMaximised() {
					window.Maximise()
				}
			})
		})
	})
	return window
}

func (a *WApp) WindowShow() {
	a.setVisible(true)
	runtime.WindowShow(a.ctx)
}

func (a *WApp) WindowHide() {
	a.setVisible(false)
	runtime.WindowHide(a.ctx)
}

func (a *WApp) WindowShowHide() {
	a.visibleMu.Lock()
	a.visible = !a.visible
	visible := a.visible
	a.visibleMu.Unlock()

	if visible {
		runtime.WindowShow(a.ctx)
	} else {
		runtime.WindowHide(a.ctx)
	}
}

func (a *WApp) setVisible(visible bool) {
	a.visibleMu.Lock()
	a.visible = visible
	a.visibleMu.Unlock()
}

func (a *WApp) GetAppTitle() string { return appTitle() }

func appTitle() string {
	return fmt.Sprintf("%s: %s", app.Name(), app.TagLine())
}
