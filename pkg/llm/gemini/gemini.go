package gemini

import (
	"context"
	"errors"
	"os"

	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/genai"
	go_genai "google.golang.org/genai"
)

type gemini struct {
	genai.GenAI
}

func New(model, instructions string) (llm.LLM, error) {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY environment variable not set")
	}

	client, err := go_genai.NewClient(ctx, &go_genai.ClientConfig{
		APIKey:  apiKey,
		Backend: go_genai.BackendGeminiAPI,
	})

	if err != nil {
		return nil, errors.New("failed to create Gemini client: " + err.Error())
	}

	return &gemini{
		GenAI: genai.GenAI{
			Model:        model,
			Instructions: instructions,
			Client:       client,
			Ctx:          ctx,
		},
	}, nil
}

func (g *gemini) Ask(prompt string, cmd llm.Command) (*llm.Response, error) {
	return g.GenAI.Ask(prompt, cmd, "Gemini")
}
