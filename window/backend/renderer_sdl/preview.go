package rendersdl

import (
	"strconv"
	"sync"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
	"github.com/veandco/go-sdl2/sdl"
)

var preview struct {
	ipc   *dispatcher.IpcT
	mutex sync.Mutex
}

func startPreviewer(sr *sdlRender) {
	if !preview.mutex.TryLock() {
		// only run once
		return
	}
	defer preview.mutex.Unlock()

	windowStyle := &dispatcher.WindowStyleT{
		StartHidden: true,
		Frameless:   true,
		AlwaysOnTop: true,
		Size:        types.XY{320, 240},
	}
	parameters := &dispatcher.PPreviewT{}
	preview.ipc, _ = dispatcher.DisplayWindow(dispatcher.WindowPreview, windowStyle, parameters, sr.previewIpcCallback)
}

func (sr *sdlRender) ShowPreview(url string) {
	if preview.ipc == nil {
		startPreviewer(sr)
	}

	x, y, _ := sdl.GetGlobalMouseState()

	preview.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "previewOpen",
		Parameters: map[string]string{
			"url": url,
			"x":   strconv.Itoa(int(x)),
			"y":   strconv.Itoa(int(y)),
		},
	})
}

func (sr *sdlRender) HidePreview() {
	if preview.ipc == nil {
		return
	}

	preview.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "previewHide",
	})
}

func (sr *sdlRender) previewIpcCallback(msg *dispatcher.IpcMessageT) {
	if msg.Error != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
	} else {
		switch msg.EventName {
		case "focus":
			sr.TriggerDeallocation(sr.window.Raise)
		}
	}
}
