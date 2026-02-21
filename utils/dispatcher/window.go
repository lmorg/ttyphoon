package dispatcher

import (
	"fmt"
	"os"
	"os/exec"
)

type WindowNameT string

const (
	WindowSDL      WindowNameT = "sdl"
	WindowInputBox WindowNameT = "inputBox"
)

func DisplayWindow(windowName WindowNameT, parameters any) error {
	/*params, err := json.Marshal(parameters)
	if err != nil {
		return err
	}*/

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe) //, "-window", string(windowName), "-meta", string(params))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", ENV_WINDOW, windowName))

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func startSdl() {
	err := DisplayWindow(WindowSDL, nil)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
