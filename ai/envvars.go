package ai

import (
	"os"

	"github.com/lmorg/ttyphoon/types"
)

const (
	_ANTHROPIC_ENV_VAR = "ANTHROPIC_API_KEY"
	_OPENAI_ENV_VAR    = "OPENAI_API_KEY"
)

func EnvOpenAi(renderer types.Renderer, callback func()) {
	renderer.DisplayInputBox("OpenAI (ChatGPT) API Key", "", func(v *types.InputBoxCallbackResultT) {
		_ = os.Setenv(_OPENAI_ENV_VAR, v.String())
	}, nil)
}

func EnvAnthropic(renderer types.Renderer, callback func()) {
	renderer.DisplayInputBox("Anthropic (Claude) API Key", "", func(v *types.InputBoxCallbackResultT) {
		_ = os.Setenv(_ANTHROPIC_ENV_VAR, v.String())
	}, nil)
}
