package rendererwebkit

import (
	goruntime "runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type inputBoxCallbacksT struct {
	ok       types.InputBoxCallbackT
	cancel   types.InputBoxCallbackT
	history  []string
	cacheKey string
}

type inputBoxesT struct {
	mu        sync.Mutex
	callbacks map[int64]inputBoxCallbacksT
}

func (ib *inputBoxesT) store(ok, cancel types.InputBoxCallbackT, cacheKey string, history []string) int64 {
	id := time.Now().UnixMicro()
	ib.mu.Lock()
	if ib.callbacks == nil {
		ib.callbacks = make(map[int64]inputBoxCallbacksT)
	}
	ib.callbacks[id] = inputBoxCallbacksT{
		ok:       ok,
		cancel:   cancel,
		history:  history,
		cacheKey: cacheKey,
	}
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

type DisplayInputBoxWT struct {
	Options    DisplayInputBoxWTOptions
	OkFunc     types.InputBoxCallbackT
	CancelFunc types.InputBoxCallbackT
}

type DisplayInputBoxWTOptions struct {
	Title       string   `json:"title"`
	Prefill     string   `json:"prefill"`
	Placeholder string   `json:"placeholder"`
	History     []string `json:"history"`
	Multiline   bool     `json:"multiline"`
}

type inputBoxPayload struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	DefaultValue string   `json:"defaultValue"`
	Placeholder  string   `json:"placeholder"`
	History      []string `json:"history"`
	Multiline    bool     `json:"multiline"`
}

// DisplayInputBoxW displays an input box with options, supporting multiline input.
func (wr *webkitRender) DisplayInputBoxW(parameters *DisplayInputBoxWT) {
	if parameters == nil {
		return
	}
	ok := parameters.OkFunc
	if ok == nil {
		ok = func(string) {}
	}
	cancel := parameters.CancelFunc
	if cancel == nil {
		cancel = func(string) {}
	}

	// get history

	var cacheKey string
	if len(parameters.Options.History) == 0 {
		// get caller
		pc, _, _, ok := goruntime.Caller(2)
		if !ok {
			cacheKey = "DisplayInputBoxW()"
		} else {
			fn := goruntime.FuncForPC(pc)
			cacheKey = strings.Replace(fn.Name(), app.ProjectSourcePath, "", 1)
		}
		cacheKey += parameters.Options.Title
		cache.Read(cache.NS_INPUTBOXW_HISTORY, cacheKey, &parameters.Options.History)
	}

	id := wr.inputBoxes.store(ok, cancel, cacheKey, parameters.Options.History)

	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalInputBox", inputBoxPayload{
			ID:           id,
			Title:        parameters.Options.Title,
			DefaultValue: parameters.Options.Prefill,
			Placeholder:  parameters.Options.Placeholder,
			History:      parameters.Options.History,
			Multiline:    parameters.Options.Multiline,
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
		if cbs.cacheKey != "" {
			cbs.history = prependHistory(value, cbs.history)
			cache.Write(cache.NS_INPUTBOXW_HISTORY, cbs.cacheKey, &cbs.history, cache.Days(365))
		}
		cbs.ok(value)
	} else {
		cbs.cancel(value)
	}
}

func (wr *webkitRender) DisplayInputBox(title, defaultValue string, ok, cancel types.InputBoxCallbackT) {
	params := &DisplayInputBoxWT{
		Options: DisplayInputBoxWTOptions{
			Title:   title,
			Prefill: defaultValue,
		},
		OkFunc:     ok,
		CancelFunc: cancel,
	}
	wr.DisplayInputBoxW(params)
}

func prependHistory(item string, slice []string) []string {
	i := slices.Index(slice, item)
	switch i {
	case -1:
		return append([]string{item}, slice...)
	case 0:
		return slice
	default:
		slice, _ = lists.RemoveOrdered(slice, i)
		return append([]string{item}, slice...)
	}
}
