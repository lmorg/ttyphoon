package rendersdl

import (
	"errors"
	"sync"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/readline/v4"
	"github.com/veandco/go-sdl2/sdl"
	"golang.design/x/clipboard"
)

type widgetReadlineT struct {
	_instance      *readline.Instance
	_termWidth     int
	_callback      chan *readline.NoTtyCallbackT
	_curPos        *types.XY
	_value         string
	_defaultStatus string
	_mutex         sync.RWMutex
	Hook           func()
}

func (sr *sdlRender) NewReadline(termWidth int32, historyKey, defaultValue, defaultStatus string) *widgetReadlineT {
	rl := &widgetReadlineT{
		_instance:      readline.NewInstance(),
		_defaultStatus: defaultStatus,
		_curPos:        &types.XY{},
		_termWidth:     int(termWidth),
	}

	rl._callback = rl._instance.MakeNoTtyChan(int(termWidth))
	rl._instance.History = newReadlineHistory(historyKey)
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
		x, y := readline.LineWrappedCellPos(0, cb.Line.Runes()[:cb.Line.RunePos()], rl._termWidth)
		rl._curPos = &types.XY{X: int32(x) * sr.glyphSize.X, Y: int32(y) * sr.glyphSize.Y}
		if cb.Hint == "" {
			sr.footerText = rl._defaultStatus
		} else {
			sr.footerText = cb.Hint
		}

		rl._mutex.Unlock()

		if rl.Hook != nil {
			rl.Hook()
		}
		sr.TriggerRedraw()
	}
}

func (rl *widgetReadlineT) eventTextInput(_ *sdlRender, evt *sdl.TextInputEvent) {
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
		(mod == codes.MOD_NONE || mod == codes.MOD_SHIFT) {
		// lets let eventTextInput() handle this so we don't need to think about
		// keyboard layouts and shift chars like whether shift+'2' == '@' or '"'
		return
	}

	if evt.Keysym.Sym == 'v' && (mod == codes.MOD_META || mod == codes.MOD_CTRL|codes.MOD_SHIFT) {
		b := clipboard.Read(clipboard.FmtText)
		if len(b) > 0 {
			for i := range b {
				if b[i] < ' ' {
					b[i] = ' '
				}
			}
			rl._instance.KeyPress(b)
		}
		return
	}

	b := codes.GetAnsiEscSeq(types.KeysNormal, keyCode, mod)
	rl._instance.KeyPress(b)
}

func (rl *widgetReadlineT) CursorPosition() *types.XY {
	rl._mutex.RLock()
	ptr := rl._curPos
	rl._mutex.RUnlock()

	return ptr
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

/*
	HISTORY
*/

var readlineHistoryCache = map[string][]string{}

// readlineHistory is an example of a LineHistory interface:
type readlineHistory struct {
	items []string
}

func newReadlineHistory(key string) *readlineHistory {
	h := new(readlineHistory)

	if key == "" {
		return h
	}

	s, ok := readlineHistoryCache[key]
	if !ok {
		s = []string{}
	}

	h.items = make([]string, len(s))
	copy(h.items, s)

	return h
}

// Write to history
func (h *readlineHistory) Write(s string) (int, error) {
	h.items = append(h.items, s)
	return len(h.items), nil
}

// GetLine returns a line from history
func (h *readlineHistory) GetLine(i int) (string, error) {
	switch {
	case i < 0:
		return "", errors.New("requested history item out of bounds: < 0")
	case i > h.Len()-1:
		return "", errors.New("requested history item out of bounds: > Len()")
	default:
		return h.items[i], nil
	}
}

// Len returns the number of lines in history
func (h *readlineHistory) Len() int {
	return len(h.items)
}

// Dump returns the entire history
func (h *readlineHistory) Dump() interface{} {
	return h.items
}
