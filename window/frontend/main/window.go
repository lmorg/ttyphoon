package mainwindow

import (
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func NewWindow(wailsApp *application.App, wapp *WApp) *application.WebviewWindow {
	wailsApp.Event.On("ttyphoon:frontend-ready", func(*application.CustomEvent) {
		go wapp.startTerminalWindow()
	})
	wailsApp.Event.On(terminalVisibleEvent, func(*application.CustomEvent) {
		setTerminalRenderingPaused(false)
	})
	wailsApp.Event.On(renderwebkit.RedrawAckEvent, func(event *application.CustomEvent) {
		batchID, ok := event.Data.(float64)
		if !ok || batchID < 1 || batchID > math.MaxInt64 || math.Trunc(batchID) != batchID {
			log.Printf("[warn] renderer: invalid terminal redraw acknowledgement: %T", event.Data)
			return
		}
		if renderer, ok := renderwebkit.CurrentRenderer(); ok {
			renderer.AcknowledgeRedraw(uint64(batchID))
		}
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
	registerRenderPauseEvents(window)
	return window
}

const terminalVisibleEvent = "ttyphoon:terminal-visible"

// On macOS, Common.WindowHide/WindowShow also fire when the window's occlusion
// state changes, so a minimised, app-hidden, or fully covered window pauses
// rendering. Focus and the frontend's terminalVisibleEvent resume rendering if
// a show event is ever missed.
func registerRenderPauseEvents(window interface {
	OnWindowEvent(events.WindowEventType, func(*application.WindowEvent)) func()
}) {
	for _, event := range []events.WindowEventType{
		events.Common.WindowHide,
		events.Common.WindowMinimise,
	} {
		window.OnWindowEvent(event, func(*application.WindowEvent) {
			setTerminalRenderingPaused(true)
		})
	}
	for _, event := range []events.WindowEventType{
		events.Common.WindowShow,
		events.Common.WindowUnMinimise,
		events.Common.WindowRestore,
		events.Common.WindowFocus,
	} {
		window.OnWindowEvent(event, func(*application.WindowEvent) {
			setTerminalRenderingPaused(false)
		})
	}
}

func setTerminalRenderingPaused(paused bool) {
	if renderer, ok := renderwebkit.CurrentRenderer(); ok {
		renderer.SetRenderingPaused(paused)
	}
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
