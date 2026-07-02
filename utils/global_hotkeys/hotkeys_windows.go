//go:build windows
// +build windows

package globalhotkeys

import (
	"log"

	"golang.design/x/hotkey"
)

func registerHotkey(hks ...*hotkeyFuncT) {
	for _, hk := range hks {
		hk.hk = hotkey.New(hk.Mod, hk.Key)
		err := hk.hk.Register()
		if err != nil {
			log.Printf("[error] hotkey failed: %v", err)
			continue
		}

		go func() {
			for range hk.hk.Keydown() {
				event <- hk
			}
		}()
	}
}
