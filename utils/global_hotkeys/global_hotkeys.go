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

var (
	event = make(chan *hotkeyFuncT)
	//ipc   *dispatcher.IpcT
)

func Register(callbackFunc func(string)) {
	/*dispatcherCallback := func(_ *dispatcher.IpcMessageT) {}
	ipc = dispatcher.GetIpc(dispatcherCallback)*/
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

	/*ipc.Send(&dispatcher.IpcMessageT{
		EventName: "started",
	})*/

	go func() {
		for hk := range event {
			hk.Func()
		}
	}()
}

func _registerHotkey(hks ...*hotkeyFuncT) {
	for _, hk := range hks {
		hk.hk = hotkey.New(hk.Mod, hk.Key)
		//os.Stderr.WriteString("regestering...\n")
		err := hk.hk.Register()
		if err != nil {
			os.Stderr.WriteString(err.Error())
			/*ipc.Send(&dispatcher.IpcMessageT{
				Error: fmt.Errorf("unable to set hotkey %s: %s", hk.hk.String(), err.Error()),
			})*/
			continue
		}

		go func() {
			for range hk.hk.Keydown() {
				event <- hk
			}
		}()
	}
}
