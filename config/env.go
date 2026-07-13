package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/utils/file"
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

func ReadEnvConfig() {
	files := file.GetConfigFiles("/", ".env")
	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Print(err)
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			split := strings.SplitN(scanner.Text(), "=", 2)
			if len(split) != 2 {
				split = []string{files[i], ""}
			}
			debug.Log(fmt.Sprintf(`%s: "%s" = "%s"`, files[i], split[0], split[1]))
			if len(split[0]) == 0 {
				continue
			}
			if split[0][0] != '!' { // exclude any env vars from child PIDs, if the name does not have `!` prefix
				unsetEnv = append(unsetEnv, split[0])
			} else {
				split[0] = split[0][1:]
			}
			os.Setenv(split[0], split[1])
		}
	}
}
