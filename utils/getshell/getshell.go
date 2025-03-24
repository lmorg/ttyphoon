package getshell

import (
	"runtime"

	"github.com/lmorg/mxtty/debug"
)

const (
	_OS_MACOS = "darwin"
	_OS_LINUX = "linux"
)

func GetShell() string {
	switch runtime.GOOS {
	case _OS_MACOS:
		shell, err := dscl()
		if err != nil {
			debug.Log(err)
			return "/bin/zsh"
		}

		return shell

	case _OS_LINUX:
		shell, err := getent()
		if err != nil {
			debug.Log(err)
			return "/bin/bash"
		}

		return shell

	default:
		return "/bin/sh"
	}
}
