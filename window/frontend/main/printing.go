package mainwindow

import (
	"github.com/lmorg/ttyphoon/utils/printing"
)

func (a *WApp) WindowPrint() error {
	return printing.Print(a.ctx)
}
