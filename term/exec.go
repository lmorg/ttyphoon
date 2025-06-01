package virtualterm

import (
	"os"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
)

func init() {
	for _, env := range config.UnsetEnv {
		err := os.Unsetenv(env)
		if err != nil {
			debug.Log(err)
		}
	}
}
