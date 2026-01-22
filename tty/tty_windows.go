//go:build ignore
// +build ignore

package tty

import (
	tty_windows "github.com/lmorg/ttyphoon/tty/windows"
	"github.com/lmorg/ttyphoon/types"
)

func NewPty(size *types.XY) (types.Pty, error) {
	return tty_windows.NewPty(size)
}
