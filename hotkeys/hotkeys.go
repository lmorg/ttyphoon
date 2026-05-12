package hotkeys

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/config"
)

type HotkeyFn func()
type TerminalFocusFn func()

var (
	prefixes           = map[codes.KeyName]*hotkeysT{}
	prefixFound        = func() {}
	setTerminalFocusFn TerminalFocusFn
)

func SetTerminalFocusFn(fn TerminalFocusFn) {
	setTerminalFocusFn = fn
}

func TerminalFocus() {
	if setTerminalFocusFn != nil {
		setTerminalFocusFn()
	}
}

type hotkeysT struct {
	fnTable   [codes.MOD_META << 1]map[codes.KeyCode]*hotKeyT
	prefixMod codes.Modifier
	prefixKey codes.KeyCode
	prefixTtl time.Time
	//mutex   sync.Mutex
}

type hotKeyT struct {
	fn   HotkeyFn
	name codes.KeyName
	desc string
	icon rune
	term bool
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

func (hk *hotkeysT) Add(key codes.KeyName, fn HotkeyFn, desc string, icon rune, term bool) {
	code, mod := key.Code()
	hk.fnTable[mod][code] = &hotKeyT{
		fn:   fn,
		name: key,
		desc: desc,
		icon: icon,
		term: term,
	}
}

func (hk *hotkeysT) KeyPress(key codes.KeyCode, mod codes.Modifier) HotkeyFn {
	fn, consume, _ := hk.KeyPressEx(key, mod)
	if !consume {
		return nil
	}

	if fn == nil {
		return prefixFound
	}

	return fn
}

func (hk *hotkeysT) KeyPressEx(key codes.KeyCode, mod codes.Modifier) (HotkeyFn, bool, bool) {
	if hk.prefixKey == 0 && hk.prefixMod == codes.MOD_NONE {
		fn := hk.fnTable[mod][key]
		if fn == nil {
			return nil, false, false
		}
		return fn.fn, true, false
	}

	if hk.prefixTtl.After(time.Now()) {
		// still within prefix time limit
		fn := hk.fnTable[mod][key]
		if fn == nil {
			// Consume the next key once a prefix is active even when no mapping exists.
			// This prevents accidental text insertion in editors while a prefix sequence is in progress.
			hk.prefixTtl = time.Time{}
			return nil, true, false
		}

		// a valid hotkey so lets extend the timeout to allow multiple hotkey presses
		hk.prefixTtl = time.Now().Add(time.Duration(config.Config.Hotkeys.RepeatTtl) * time.Millisecond)
		return fn.fn, true, true
	}

	if key == hk.prefixKey && mod == hk.prefixMod {
		// prefix pressed, lets add a wait for any subsequent hotkeys
		hk.prefixTtl = time.Now().Add(time.Duration(config.Config.Hotkeys.PrefixTtl) * time.Millisecond)
		return nil, true, true
	}

	// not a hotkey. Nothing to see here!
	return nil, false, false
}

// Add appends the hotkey DB with the values included.
// If no icon is 0, then hotkey will not be included in command palette
func Add(prefix codes.KeyName, hotkey codes.KeyName, fn HotkeyFn, desc string, icon rune, term bool) {
	hk := prefixes[prefix]
	if hk == nil {
		hk = newPrefix(prefix)
	}

	if term {
		hk.Add(hotkey, func() {
			fn()
			TerminalFocus()
		}, desc, icon, term)
		return
	}

	hk.Add(hotkey, fn, desc, icon, term)
}

func KeyPress(key codes.KeyCode, mod codes.Modifier) HotkeyFn {
	for _, prefix := range prefixes {
		fn := prefix.KeyPress(key, mod)
		if fn != nil {
			return fn
		}
	}

	return nil
}

func KeyPressEx(key codes.KeyCode, mod codes.Modifier) (HotkeyFn, bool, bool) {
	for _, prefix := range prefixes {
		fn, consume, prefixActive := prefix.KeyPressEx(key, mod)
		if consume {
			return fn, true, prefixActive
		}
	}

	return nil, false, false
}

func KeyPressWithPrefix(prefix codes.KeyName, hotkey codes.KeyName) error {
	hotkeys := prefixes[prefix]
	if hotkeys == nil {
		return fmt.Errorf("hotkey prefix not found: %s", prefix)
	}

	code, mod := hotkey.Code()
	codes := hotkeys.fnTable[mod]
	if codes == nil {
		return fmt.Errorf("hotkey not found with modifier: %s", hotkey)
	}
	hk := codes[code]
	if hk == nil {
		return fmt.Errorf("hotkey not found with key code: %s", hotkey)
	}

	hk.fn()
	return nil
}

type HotKeyListItemT struct {
	Prefix      codes.KeyName
	Hotkey      codes.KeyName
	Description string
	Icon        rune
}

func (hk *hotKeyT) Description() string {
	if hk.term {
		return "Terminal: " + hk.desc
	}
	return hk.desc
}

func List() []*HotKeyListItemT {
	var list []*HotKeyListItemT

	for prefix := range prefixes {
		for _, codes := range prefixes[prefix].fnTable {
			for _, hk := range codes {
				list = append(list, &HotKeyListItemT{
					Prefix:      prefix,
					Hotkey:      hk.name,
					Description: hk.Description(),
					Icon:        hk.icon,
				})
			}
		}

	}

	slices.SortFunc(list, func(a, b *HotKeyListItemT) int {
		n := strings.Compare(a.Description, b.Description)
		if n != 0 {
			return n
		}

		n = strings.Compare(string(a.Prefix), string(b.Prefix))
		if n != 0 {
			return n
		}

		return strings.Compare(string(a.Hotkey), string(b.Hotkey))
	})

	return list
}
