package rendererwebkit

import (
	"sync/atomic"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

type keyboardModeT struct {
	keyboardMode int32
}

func (km *keyboardModeT) Set(mode types.KeyboardMode) {
	if config.Config.Tmux.Enabled {
		mode = types.KeysTmuxClient // override keyboard mode if in tmux control mode
	}
	atomic.StoreInt32(&km.keyboardMode, int32(mode))
}
func (km *keyboardModeT) Get() types.KeyboardMode {
	return types.KeyboardMode(atomic.LoadInt32(&km.keyboardMode))
}

func (wr *webkitRender) SetKeyboardFnMode(code types.KeyboardMode) {
	wr.keyboardMode.Set(code)
}
