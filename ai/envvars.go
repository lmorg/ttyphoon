package ai

import (
	"os"

	"github.com/lmorg/mxtty/types"
)

const (
	_ANTHROPIC_ENV_VAR = "ANTHROPIC_API_KEY"
	_OPENAI_ENV_VAR    = "OPENAI_API_KEY"
)

func EnvOpenAi(renderer types.Renderer, callback func()) {
	renderer.DisplayInputBox("OpenAI (ChatGPT) API Key", "", func(s string) {
		_ = os.Setenv(_OPENAI_ENV_VAR, s)
	})
}

func EnvAnthropic(renderer types.Renderer, callback func()) {
	renderer.DisplayInputBox("Anthropic (Claude) API Key", "", func(s string) {
		_ = os.Setenv(_ANTHROPIC_ENV_VAR, s)
	})
}
