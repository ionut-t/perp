package gemini

import (
	"context"

	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/genai"
	go_genai "google.golang.org/genai"
)

type gemini struct {
	genai.GenAI
}

func New(ctx context.Context, model, apiKey, instructions string) (llm.LLM, error) {
	client, err := go_genai.NewClient(ctx, &go_genai.ClientConfig{
		APIKey:  apiKey,
		Backend: go_genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
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

func (g *gemini) SetModel(model string) error {
	return g.GenAI.SetModel(model, "Gemini")
}
