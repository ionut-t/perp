package llm_factory

import (
	"errors"
	"strings"

	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/gemini"
	"github.com/ionut-t/perp/pkg/llm/vertexai"
)

func New(config config.Config, instructions string) (llm.LLM, error) {
	provider, err := config.GetLLMProvider()

	provider = strings.ToLower(provider)

	if err != nil {
		provider = "gemini"
	}

	llmModel, _ := config.GetLLMModel()

	switch provider {
	case "gemini":
		return gemini.New(llmModel, instructions)
	case "vertexai":
		return vertexai.New(llmModel, instructions)
	default:
		return nil, errors.New("unsupported LLM provider: " + provider)
	}
}
