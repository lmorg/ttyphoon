//go:build ignore
// +build ignore

package tty

import (
	tty_windows "github.com/lmorg/mxtty/tty/windows"
	"github.com/lmorg/mxtty/types"
)

func NewPty(size *types.XY) (types.Pty, error) {
	return tty_windows.NewPty(size)
}
