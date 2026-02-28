package rendersdl

import (
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) DisplayInputBoxW(title, prefill string, history []string, okFunc, keyPressFunc types.InputBoxCallbackT) {
	sr.cancelWInputBox()

	pos := new(types.XY)
	pos.X, pos.Y = sr.window.GetPosition()
	displayIndex, err := sr.window.GetDisplayIndex()
	if err == nil {
		bounds, _ := sdl.GetDisplayBounds(displayIndex)
		if bounds.X < pos.X {
			pos.X -= bounds.X
		}
		if bounds.Y < pos.Y {
			pos.Y -= bounds.Y
		}
	}
	w, _ := sr.window.GetSize()
	size := &types.XY{X: w, Y: 300}

	windowStyle := dispatcher.NewWindowStyle()
	windowStyle.Pos = *pos
	windowStyle.Size = *size
	windowStyle.AlwaysOnTop = true
	windowStyle.Frameless = true

	// get history

	var cacheKey string
	// get caller
	if history == nil {
		pc, _, _, ok := runtime.Caller(2)
		if !ok {
			cacheKey = "DisplayInputBoxW()"
		} else {
			fn := runtime.FuncForPC(pc)
			cacheKey = strings.Replace(fn.Name(), app.ProjectSourcePath, "", 1)
		}
		cacheKey += title
		cache.Read(cache.NS_INPUTBOXW_HISTORY, cacheKey, &history)
	}

	// display

	parameters := &dispatcher.PInputBoxT{
		Title:   strings.Title(title),
		Prefill: prefill,
		History: history,
	}

	_, sr._cancelWInputBox = dispatcher.DisplayWindow(dispatcher.WindowInputBox, windowStyle, parameters, func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
			return
		}
		var value string
		if msg.Parameters != nil {
			value = strings.TrimSpace(msg.Parameters["value"])
		}
		switch msg.EventName {
		case "ok":
			if value != "" {
				history = prependHistory(value, history)
				if cacheKey != "" {
					cache.Write(cache.NS_INPUTBOXW_HISTORY, cacheKey, &history, time.Now().Add(time.Hour*24*365))
				}
			}
			okFunc(value)
		case "keyPress":
			if keyPressFunc != nil {
				keyPressFunc(value)
			}
		}
	})
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
