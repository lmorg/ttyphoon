package mainwindow

import (
	"context"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/utils/wailsruntime"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type visibilityTestWindow struct {
	application.Window
	calls []string
}

func (w *visibilityTestWindow) Show() application.Window {
	w.calls = append(w.calls, "show")
	return w
}
func (w *visibilityTestWindow) Focus() { w.calls = append(w.calls, "focus") }
func (w *visibilityTestWindow) Hide() application.Window {
	w.calls = append(w.calls, "hide")
	return w
}

func TestWindowShowHideFocusesOnlyWhenShowing(t *testing.T) {
	window := &visibilityTestWindow{}
	desktop := &wailsDesktopRuntime{window: window}
	app := &WApp{ctx: wailsruntime.WithRuntime(context.Background(), desktop)}

	app.WindowShowHide()
	if !app.visible || !reflect.DeepEqual(window.calls, []string{"show", "focus"}) {
		t.Fatalf("show toggle: visible=%v, calls=%v", app.visible, window.calls)
	}
	window.calls = nil
	app.WindowShowHide()
	if app.visible || !reflect.DeepEqual(window.calls, []string{"hide"}) {
		t.Fatalf("hide toggle: visible=%v, calls=%v", app.visible, window.calls)
	}
}
