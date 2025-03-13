package virtualterm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/utils/getshell"
)

func init() {
	for _, env := range config.UnsetEnv {
		err := os.Unsetenv(env)
		if err != nil {
			debug.Log(err)
		}
	}
}

func (term *Term) exec() {
	if term.Pty.File() == nil {
		return
	}

	var defaultErr, fallbackErr error
	if len(config.Config.Shell.Default) == 0 {
		goto fallback
	}

	defaultErr = _exec(term.Pty.File(), config.Config.Shell.Default, &term.process)
	if defaultErr == nil {
		// success, no need to run fallback shell
		term.renderer.TriggerQuit()
		return
	}

fallback:
	fallbackErr = _exec(term.Pty.File(), config.Config.Shell.Fallback, &term.process)
	if fallbackErr == nil {
		// success, no need to run fallback shell
		term.renderer.TriggerQuit()
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
