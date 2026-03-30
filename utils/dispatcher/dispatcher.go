package dispatcher

import (
	"fmt"
	"os"
	"os/exec"
)

const ENV_APP = "MXTTY_APP"

type AppTypeT string

const (
	AppGlobalHotkeys AppTypeT = "globalHotkeys"
)

func StartApp(app AppTypeT, callback RespFunc) (*IpcT, func()) {
	exe, err := os.Executable()
	if err != nil {
		callback(&IpcMessageT{Error: err})
		return nil, nil
	}

	cmd := exec.Command(exe)
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%s", ENV_APP, app),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	ipc := &IpcT{
		r:        stdout,
		w:        stdin,
		respFunc: callback,
	}

	err = cmd.Start()
	if err != nil {
		callback(&IpcMessageT{Error: err})
		return nil, nil
	}

	cleanUp := func() { _ = cmd.Process.Kill() }
	cleanUpFuncs = append(cleanUpFuncs, cleanUp)

	go ipc.listen()

	return ipc, cleanUp
}

func GetIpc(callback RespFunc) *IpcT {
	ipc := &IpcT{
		r:        os.Stdin,
		w:        os.Stdout,
		respFunc: callback,
	}

	go ipc.listen()

	return ipc
}
