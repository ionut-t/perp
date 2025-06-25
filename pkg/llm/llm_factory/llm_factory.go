package llm_factory

import (
	"errors"

	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/gemini"
	"github.com/ionut-t/perp/pkg/llm/vertexai"
)

func New(config config.Config, instructions string) (llm.LLM, error) {
	provider, err := config.GetLLMProvider()

	if err != nil {
		provider = "gemini"
	}

	llmModel, _ := config.GetLLMModel()

	switch provider {
	case "gemini":
		llmApiKey, _ := config.GetLLMApiKey()
		return gemini.New(llmApiKey, llmModel, instructions)
	case "vertexai":
		vertexAIProjectID, _ := config.GetVertexAIProjectID()
		vertexAILocation, _ := config.GetVertexAILocation()
		return vertexai.New(vertexAIProjectID, vertexAILocation, llmModel, instructions)
	default:
		return nil, errors.New("unsupported LLM provider: " + provider)
	}
}
