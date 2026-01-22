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

var (
	prefixes    = map[codes.KeyName]*hotkeysT{}
	prefixFound = func() {}
)

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

func (hk *hotkeysT) Add(key codes.KeyName, fn HotkeyFn, desc string) {
	code, mod := key.Code()
	hk.fnTable[mod][code] = &hotKeyT{
		fn:   fn,
		name: key,
		desc: desc,
	}
}

func (hk *hotkeysT) KeyPress(key codes.KeyCode, mod codes.Modifier) HotkeyFn {
	if hk.prefixKey == 0 && hk.prefixMod == codes.MOD_NONE {
		fn := hk.fnTable[mod][key]
		if fn == nil {
			return nil
		}
		return fn.fn
	}

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

func Add(prefix codes.KeyName, hotkey codes.KeyName, fn HotkeyFn, desc string) {
	hk := prefixes[prefix]
	if hk == nil {
		hk = newPrefix(prefix)
	}

	hk.Add(hotkey, fn, desc)
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
}

func List() []*HotKeyListItemT {
	var list []*HotKeyListItemT

	for prefix := range prefixes {
		for _, codes := range prefixes[prefix].fnTable {
			for _, hk := range codes {
				list = append(list, &HotKeyListItemT{
					Prefix:      prefix,
					Hotkey:      hk.name,
					Description: hk.desc,
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
