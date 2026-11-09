package mainwindow

import (
	"context"
	"log"

	"github.com/lmorg/ttyphoon/hotkeys"
	"github.com/lmorg/ttyphoon/utils/file/watcher"
	globalhotkeys "github.com/lmorg/ttyphoon/utils/global_hotkeys"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func (a *WApp) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = runtime.WithRuntime(ctx, a.uiRuntime)

	// Set up log capture to emit events
	lw := &logWriter{ctx: a.ctx}
	log.SetOutput(lw)

	hotkeys.SetTerminalFocusFn(a.FocusTerminalPane)
	watcher.SetEventCallback(func(filename string) {
		if a.ctx == nil {
			return
		}
		runtime.EventsEmit(a.ctx, "notesFileChanged", filename)
	})

	globalhotkeys.Register(func(key string) {
		switch key {
		case "F12":
			a.WindowShowHide()
		}
	})
	return nil
}

func Shutdown(a *WApp) {
	a.stopGitStatusWatch()
	a.notesListMu.Lock()
	a.notesIndexClosed = true
	if a.notesListCancel != nil {
		a.notesListCancel()
	}
	index := a.notesFileIndex
	unsubscribe := a.notesIndexUnsubscribe
	a.notesListMu.Unlock()
	if unsubscribe != nil {
		unsubscribe()
	}
	if index != nil {
		index.Close()
	}
	watcher.Close()
}
