package virtualterm

import (
	"os"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/debug"
)

func init() {
	for _, env := range config.UnsetEnv {
		err := os.Unsetenv(env)
		if err != nil {
			debug.Log(err)
		}
	}
}
