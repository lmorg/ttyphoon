package main

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type wailsDesktopRuntime struct {
	app    *application.App
	window application.Window
}

func (runtime *wailsDesktopRuntime) Emit(event string, data ...any) {
	runtime.app.Event.Emit(event, data...)
}

func (runtime *wailsDesktopRuntime) OpenURL(url string) {
	_ = runtime.app.Browser.OpenURL(url)
}

func (runtime *wailsDesktopRuntime) ShowWindow() {
	runtime.window.Show()
}

func (runtime *wailsDesktopRuntime) HideWindow() {
	runtime.window.Hide()
}

func (runtime *wailsDesktopRuntime) SetWindowTitle(title string) {
	runtime.window.SetTitle(title)
}

func (runtime *wailsDesktopRuntime) SetWindowSize(width, height int) {
	runtime.window.SetSize(width, height)
}

func (runtime *wailsDesktopRuntime) SetAlwaysOnTop(alwaysOnTop bool) {
	runtime.window.SetAlwaysOnTop(alwaysOnTop)
}

func (runtime *wailsDesktopRuntime) SaveFileDialog(options types.SaveDialogOptions) (string, error) {
	dialogOptions := &application.SaveFileDialogOptions{
		Title:     options.Title,
		Directory: options.DefaultDirectory,
		Filename:  options.DefaultFilename,
	}
	for _, filter := range options.Filters {
		dialogOptions.Filters = append(dialogOptions.Filters, application.FileFilter{
			DisplayName: filter.DisplayName,
			Pattern:     filter.Pattern,
		})
	}
	return runtime.app.Dialog.SaveFileWithOptions(dialogOptions).PromptForSingleSelection()
}

func (runtime *wailsDesktopRuntime) PrintWindow() error {
	return runtime.window.Print()
}
