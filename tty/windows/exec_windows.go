//go:build ignore
// +build ignore

package tty_windows

import (
	"context"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	conpty "github.com/qsocket/conpty-go"
)

//

func (p *Pty) ExecuteShell(exit func()) {
	var defaultErr, fallbackErr error
	if len(config.Config.Shell.Default) == 0 {
		goto fallback
	}

	defaultErr = p._exec(config.Config.Shell.Default)
	if defaultErr == nil {
		// success, no need to run fallback shell
		exit()
		return
	}

fallback:
	fallbackErr = p._exec(config.Config.Shell.Fallback)
	if fallbackErr == nil {
		// success, no need to run fallback shell
		exit()
		return
	}

	panic(fmt.Sprintf(
		"Cannot launch either shell: (Default) %s: (Fallback) %s",
		defaultErr, fallbackErr))
}

func (p *Pty) _exec(command []string) (err error) {
	cmd := strings.Join(command, " ")
	p.conPty, err = conpty.Start(cmd, conpty.ConPtyDimensions(p.w, p.h))
	if err != nil {
		return err
	}

	go p.read()

	_, err = p.conPty.Wait(context.Background())
	return err
}
