package prompts

import (
	"io"
	"log"
	"os"

	"github.com/lmorg/ttyphoon/utils/file"
)

func AgentsMd() string {
	var (
		prompt string
		files  = file.GetConfigGlob("AGENTS.md")
	)

	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Printf("ERROR: cannot open %s: %v", files[i], err)
			continue
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			log.Printf("ERROR: cannot read %s: %v", files[i], err)
			continue
		}

		prompt = prompt + "\n\n" + string(b)
	}

	if prompt == "" {
		return prompt
	}

	return "\n\n# AGENTS.md\n\n%s" + prompt
}
