package ai

import (
	"context"
	"fmt"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func OpenAI(cmdLine, termOutput, userPrompt string) (string, error) {
	query := getPrompt(cmdLine, termOutput, userPrompt)

	// Initialize OpenAI LLM and embedder
	llm, err := openai.New(openai.WithModel(_OPENAI_MODEL))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI client: %v", err)
	}

	enc, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		return "", fmt.Errorf("unable to create token encoder: %v", err)
	}

	tokens := enc.Encode(query, nil, nil)
	if len(tokens) >= 200_000 {
		return "", fmt.Errorf("output contains too many tokens")
	}

	// Get answer from LLM
	answer, err := llm.Call(context.Background(), query, llms.WithTemperature(1))
	if err != nil {
		return "", fmt.Errorf("failed to get response from LLM: %v", err)
	}

	return answer, nil
}
