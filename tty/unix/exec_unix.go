//go:build !windows
// +build !windows

package tty_unix

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/getshell"
)

func (p *Pty) ExecuteShell(exit func()) {
	var defaultErr, fallbackErr error
	if len(config.Config.Shell.Default) == 0 {
		goto fallback
	}

	defaultErr = _exec(p.primary, config.Config.Shell.Default, &p.process)
	if defaultErr == nil {
		// success, no need to run fallback shell
		exit()
		return
	}

fallback:
	fallbackErr = _exec(p.primary, config.Config.Shell.Fallback, &p.process)
	if fallbackErr == nil {
		// success, no need to run fallback shell
		exit()
		return
	}

	panic(fmt.Sprintf(
		"Cannot launch either shell: (Default) %s: (Fallback) %s",
		defaultErr, fallbackErr))
}

func _exec(tty *os.File, command []string, proc **os.Process) error {
	if len(command) == 0 || command[0] == "" {
		command = []string{getshell.GetShell()}
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = config.SetEnv()
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Noctty:  false,
		Setctty: true,
		//Ctty:    int(term.Pty.File().Fd()),
		//Setpgid: true,
		Setsid: true,
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	//cmd.SysProcAttr.Ctty = cmd.Process.Pid
	cmd.SysProcAttr.Pgid = cmd.Process.Pid

	*proc = cmd.Process

	err = cmd.Wait()
	if err != nil && strings.HasPrefix(err.Error(), "Signal") {
		return err
	}

	return nil
}
