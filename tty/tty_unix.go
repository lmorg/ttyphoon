//go:build !windows
// +build !windows

package tty

import (
	tty_unix "github.com/lmorg/ttyphoon/tty/unix"
	"github.com/lmorg/ttyphoon/types"
)

func NewPty(size *types.XY) (types.Pty, error) {
	return tty_unix.NewPty(size)
}
