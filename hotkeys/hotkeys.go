package hotkeys

import (
	"time"

	"github.com/lmorg/mxtty/codes"
	"github.com/lmorg/mxtty/config"
)

type HotKeyFn func() error

var (
	prefixes    = map[codes.KeyName]*hotkeysT{}
	prefixFound = func() error { return nil }
)

type hotkeysT struct {
	fnTable   [codes.MOD_META << 1]map[codes.KeyCode]*hotKeyT
	prefixMod codes.Modifier
	prefixKey codes.KeyCode
	prefixTtl time.Time
	//mutex   sync.Mutex
}

type hotKeyT struct {
	fn   HotKeyFn
	desc string
}

func newPrefix(prefix codes.KeyName) *hotkeysT {
	key, mod := prefix.Code()
	hk := &hotkeysT{
		prefixKey: key,
		prefixMod: mod,
		prefixTtl: time.Now(),
	}

	for i := range hk.fnTable {
		hk.fnTable[i] = make(map[codes.KeyCode]*hotKeyT)
	}

	prefixes[prefix] = hk
	return hk
}

func (hk *hotkeysT) Add(key codes.KeyCode, mod codes.Modifier, fn HotKeyFn, desc string) {
	hk.fnTable[mod][key] = &hotKeyT{fn, desc}
}

func (hk *hotkeysT) KeyPress(key codes.KeyCode, mod codes.Modifier) HotKeyFn {
	if hk.prefixTtl.After(time.Now()) {
		// still within prefix time limit
		fn := hk.fnTable[mod][key]
		if fn == nil {
			return nil
		}

		// a valid hotkey so lets extend the timeout to allow multiple hotkey presses
		hk.prefixTtl = time.Now().Add(time.Duration(config.Config.Hotkeys.RepeatTtl) * time.Millisecond)
		return fn.fn
	}

	if key == hk.prefixKey && mod == hk.prefixMod {
		// prefix pressed, lets add a wait for any subsequent hotkeys
		hk.prefixTtl = time.Now().Add(time.Duration(config.Config.Hotkeys.PrefixTtl) * time.Millisecond)
		return prefixFound
	}

	// not a hotkey. Nothing to see here!
	return nil
}

func Add(prefix codes.KeyName, hotkey codes.KeyName, fn HotKeyFn, desc string) {
	hk := prefixes[prefix]
	if hk == nil {
		hk = newPrefix(prefix)
	}

	key, mod := hotkey.Code()
	hk.Add(key, mod, fn, desc)
}

func KeyPress(key codes.KeyCode, mod codes.Modifier) HotKeyFn {
	for _, prefix := range prefixes {
		fn := prefix.KeyPress(key, mod)
		if fn != nil {
			return fn
		}
	}

	return nil
}
