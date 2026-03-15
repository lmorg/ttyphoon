package rendererwebkit

import (
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type inputBoxCallbacksT struct {
	ok     types.InputBoxCallbackT
	cancel types.InputBoxCallbackT
}

type inputBoxesT struct {
	mu        sync.Mutex
	callbacks map[int64]inputBoxCallbacksT
}

func (ib *inputBoxesT) store(ok, cancel types.InputBoxCallbackT) int64 {
	id := time.Now().UnixMicro()
	ib.mu.Lock()
	if ib.callbacks == nil {
		ib.callbacks = make(map[int64]inputBoxCallbacksT)
	}
	ib.callbacks[id] = inputBoxCallbacksT{ok: ok, cancel: cancel}
	ib.mu.Unlock()
	return id
}

func (ib *inputBoxesT) pop(id int64) (inputBoxCallbacksT, bool) {
	ib.mu.Lock()
	cbs, ok := ib.callbacks[id]
	if ok {
		delete(ib.callbacks, id)
	}
	ib.mu.Unlock()
	return cbs, ok
}

type inputBoxPayload struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	DefaultValue string `json:"defaultValue"`
}

func (wr *webkitRender) DisplayInputBox(title, defaultValue string, ok, cancel types.InputBoxCallbackT) {
	if ok == nil {
		ok = func(string) {}
	}
	if cancel == nil {
		cancel = func(string) {}
	}

	id := wr.inputBoxes.store(ok, cancel)

	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalInputBox", inputBoxPayload{
			ID:           id,
			Title:        title,
			DefaultValue: defaultValue,
		})
	}
}

// InputBoxSubmit is called from wails.go when JS submits the input box.
func (wr *webkitRender) InputBoxSubmit(id int64, value string, isOk bool) {
	cbs, ok := wr.inputBoxes.pop(id)
	if !ok {
		return
	}
	if isOk {
		cbs.ok(value)
	} else {
		cbs.cancel(value)
	}
}
