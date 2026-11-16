package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/app"
)

var unsetEnv = []string{
	"TMUX",
	"TERM",
	"TERM_PROGRAM",
	"GITHUB_TOKEN",
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"OLLAMA_HOST",
}

var exportVars = map[string]string{
	"MXTTY":         "true",
	"MXTTY_VERSION": app.Version(),
	"TERM":          "xterm-256color",
	"TERM_PROGRAM":  "mxtty",
}

func SetEnv() []string {
	var err error
	envvars := os.Environ()

	for _, env := range unsetEnv {
		env += "="
		for i := range envvars {
			if strings.HasPrefix(envvars[i], env) {
				envvars, err = lists.RemoveUnordered(envvars, i)
				if err != nil {
					panic(err)
				}
				break // we don't expect duplicates vars with the same name
			}
		}
	}

	for env, value := range exportVars {
		envvars = append(envvars, fmt.Sprintf("%s=%s", env, value))
	}

	if Config.Tmux.Enabled {
		envvars = append(envvars, "MXTTY_TMUX=true")
	}

	return envvars
}
