//go:build !windows
// +build !windows

package tty

import (
	tty_unix "github.com/lmorg/mxtty/tty/unix"
	"github.com/lmorg/mxtty/types"
)

func NewPty(size *types.XY) (types.Pty, error) {
	return tty_unix.NewPty(size)
}
