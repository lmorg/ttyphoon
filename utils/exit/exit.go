package exit

import (
	"os"

	"github.com/lmorg/ttyphoon/debug/pprof"
)

func Exit(code int) {
	pprof.CleanUp()
	os.Exit(code)
}
