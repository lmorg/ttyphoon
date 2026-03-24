package globalhotkeys

import (
	"os"

	"golang.design/x/hotkey"
)

type hotkeyFuncT struct {
	Key  hotkey.Key
	Mod  []hotkey.Modifier
	Func func()
	hk   *hotkey.Hotkey
}

var event = make(chan *hotkeyFuncT)

func Register(callbackFunc func(string)) {
	hotkeyCallback := func(key string) func() {
		return func() { callbackFunc(key) }
	}

	registerHotkey(
		&hotkeyFuncT{
			Key:  hotkey.KeyF12,
			Func: hotkeyCallback("F12"),
		},
	/*&hotkeyFuncT{
		Key:  hotkey.KeyF10,
		Func: sr.toggleNotes,
	},*/
	)

	go func() {
		for hk := range event {
			hk.Func()
		}
	}()
}

func registerHotkey(hks ...*hotkeyFuncT) {
	for _, hk := range hks {
		hk.hk = hotkey.New(hk.Mod, hk.Key)
		err := hk.hk.Register()
		if err != nil {
			os.Stderr.WriteString(err.Error())
			continue
		}

		go func() {
			for range hk.hk.Keydown() {
				event <- hk
			}
		}()
	}
}
