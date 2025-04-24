package rendersdl

import (
	"sync"

	"github.com/lmorg/mxtty/codes"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/readline/v4"
	"github.com/veandco/go-sdl2/sdl"
)

type widgetReadlineT struct {
	_instance      *readline.Instance
	_callback      chan *readline.NoTtyCallbackT
	_curPos        int32
	_value         string
	_defaultStatus string
	_mutex         sync.RWMutex
}

func (sr *sdlRender) NewReadline(defaultValue, defaultStatus string) *widgetReadlineT {
	rl := &widgetReadlineT{
		_instance:      readline.NewInstance(),
		_defaultStatus: defaultStatus,
	}

	rl._callback = rl._instance.MakeNoTtyChan()
	rl._instance.History.Write(defaultValue)
	rl._instance.HintText = func([]rune, int) []rune { return []rune(defaultStatus) }

	go rl._reader(sr)

	return rl
}

func (rl *widgetReadlineT) _reader(sr *sdlRender) {
	for {
		cb, ok := <-rl._callback
		if !ok {
			return
		}

		rl._mutex.Lock()

		rl._value = cb.Line.String()
		rl._curPos = int32(cb.Line.CellPos()) * (sr.glyphSize.X + dropShadowOffset)
		if cb.Hint == "" {
			sr.footerText = rl._defaultStatus
		} else {
			sr.footerText = cb.Hint
		}

		rl._mutex.Unlock()

		sr.TriggerRedraw()
	}
}

func (rl *widgetReadlineT) eventTextInput(sr *sdlRender, evt *sdl.TextInputEvent) {
	rl._instance.KeyPress([]byte(evt.GetText()))
}

func (rl *widgetReadlineT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	switch evt.Keysym.Sym {
	case sdl.K_LSHIFT, sdl.K_RSHIFT, sdl.K_LALT, sdl.K_RALT,
		sdl.K_LCTRL, sdl.K_RCTRL, sdl.K_LGUI, sdl.K_RGUI,
		sdl.K_CAPSLOCK, sdl.K_NUMLOCKCLEAR, sdl.K_SCROLLLOCK, sdl.K_SPACE:
		// modifier keys pressed on their own shouldn't trigger anything
		return
	}

	keyCode := sr.keyCodeLookup(evt.Keysym.Sym)
	mod := keyEventModToCodesModifier(evt.Keysym.Mod)

	if (evt.Keysym.Sym > ' ' && evt.Keysym.Sym < 127) &&
		(mod == 0 || mod == codes.MOD_SHIFT) {
		// lets let eventTextInput() handle this so we don't need to think about
		// keyboard layouts and shift chars like whether shift+'2' == '@' or '"'
		return
	}

	b := codes.GetAnsiEscSeq(types.KeysNormal, keyCode, mod)
	rl._instance.KeyPress(b)
}

func (rl *widgetReadlineT) CursorPosition() int32 {
	rl._mutex.RLock()
	i := rl._curPos
	rl._mutex.RUnlock()

	return i
}

func (rl *widgetReadlineT) Value() string {
	rl._mutex.RLock()
	s := rl._value
	rl._mutex.RUnlock()

	return s
}

// Readline method here is a little crazy, but it's so we remember to use non-blocking
// APIs such as goroutines for instance.Readline() and the deallocStack for callbacks
func (rl *widgetReadlineT) Readline(sr *sdlRender, callback func(string, error)) {
	go func() {
		s, err := rl._instance.Readline()
		sr._deallocStack <- func() {
			callback(s, err)
		}
	}()
}
