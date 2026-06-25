//go:build darwin
// +build darwin

package globalhotkeys

import (
	"github.com/lmorg/ttyphoon/utils/global_hotkeys/macos"
)

func registerHotkey(hks ...*hotkeyFuncT) {
	for _, hk := range hks {
		var mod uint32
		for _, m := range hk.Mod {
			mod |= uint32(m)
		}
		macos.Register(uint32(hk.Key), mod, hk.Func)
	}
}
