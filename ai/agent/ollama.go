package agent

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

func Init(renderer types.Renderer) {
	go addServiceOllama(renderer)
}

func addServiceOllama(renderer types.Renderer) {
	sticky := renderer.DisplaySticky(types.NOTIFY_INFO, "Querying Ollama....", func() {})
	defer sticky.Close()

	ollamaModels := ollamaModels()
	if len(ollamaModels) > 0 {
		go func() {
			if len(models) > 0 {
				models[LLM_OLLAMA] = ollamaModels
				return
			}
			time.Sleep(100 * time.Millisecond)
		}()
	}
}

func ollamaModels() []string {
	var (
		buf    bytes.Buffer
		models []string
		err    error
	)

	defer debug.Log(err)
	defer debug.Log(models)

	cmd := exec.Command("ollama", "list")
	cmd.Env = os.Environ()
	cmd.Stdout = &buf

	err = cmd.Start()
	if err != nil {
		return nil
	}

	err = cmd.Wait()
	if err != nil {
		return nil
	}

	lines := strings.Split(buf.String(), "\n")
	if len(lines) < 2 {
		return nil
	}

	for i := 1; i < len(lines); i++ {
		split := strings.SplitN(lines[i], " ", 2)
		if len(split) != 2 {
			continue
		}
		models = append(models, split[0])
	}
	return models
}
