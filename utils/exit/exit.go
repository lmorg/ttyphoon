package exit

import (
	"os"

	"github.com/lmorg/ttyphoon/debug/pprof"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func Exit(code int) {
	pprof.CleanUp()
	dispatcher.CleanUp()
	os.Exit(code)
}
