package wailsruntime

import (
	"context"
	"errors"

	"github.com/lmorg/ttyphoon/types"
)

type runtimeContextKey struct{}

var ErrUnavailable = errors.New("Wails application runtime is unavailable")

type FileFilter = types.FileFilter
type SaveDialogOptions = types.SaveDialogOptions

func WithRuntime(ctx context.Context, runtime types.DesktopRuntime) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, runtime)
}

func getRuntime(ctx context.Context) types.DesktopRuntime {
	if ctx == nil {
		return nil
	}
	runtime, _ := ctx.Value(runtimeContextKey{}).(types.DesktopRuntime)
	return runtime
}

func EventsEmit(ctx context.Context, event string, data ...any) {
	if runtime := getRuntime(ctx); runtime != nil {
		runtime.Emit(event, data...)
	}
}

func BrowserOpenURL(ctx context.Context, url string) {
	if runtime := getRuntime(ctx); runtime != nil {
		runtime.OpenURL(url)
	}
}

func WindowShow(ctx context.Context) {
	if runtime := getRuntime(ctx); runtime != nil {
		runtime.ShowWindow()
	}
}

func WindowHide(ctx context.Context) {
	if runtime := getRuntime(ctx); runtime != nil {
		runtime.HideWindow()
	}
}

func WindowSetTitle(ctx context.Context, title string) {
	if runtime := getRuntime(ctx); runtime != nil {	
		runtime.SetWindowTitle(title)
	}
}

func WindowSetSize(ctx context.Context, width, height int) {
	if runtime := getRuntime(ctx); runtime != nil {
		runtime.SetWindowSize(width, height)
	}
}

func WindowSetAlwaysOnTop(ctx context.Context, alwaysOnTop bool) {
	if runtime := getRuntime(ctx); runtime != nil {
		runtime.SetAlwaysOnTop(alwaysOnTop)
	}
}

func WindowPrint(ctx context.Context) error {
	runtime := getRuntime(ctx)
	if runtime == nil {
		return ErrUnavailable
	}
	return runtime.PrintWindow()
}

func SaveFileDialog(ctx context.Context, options SaveDialogOptions) (string, error) {
	runtime := getRuntime(ctx)
	if runtime == nil {
		return "", ErrUnavailable
	}
	return runtime.SaveFileDialog(options)
}