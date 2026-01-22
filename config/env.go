package config

import (
	"fmt"
	"os"

	"github.com/lmorg/ttyphoon/app"
)

var UnsetEnv = []string{
	"TMUX",
	"TERM",
	"TERM_PROGRAM",
}

var export = map[string]string{
	"MXTTY":         "true",
	"MXTTY_VERSION": app.Version(),
	"TERM":          "xterm-256color",
	"TERM_PROGRAM":  "mxtty",
}

func SetEnv() []string {
	envvars := os.Environ()

	for env, value := range export {
		envvars = append(envvars, fmt.Sprintf("%s=%s", env, value))
	}

	if Config.Tmux.Enabled {
		envvars = append(envvars, "MXTTY_TMUX=true")
	}

	return envvars
}
