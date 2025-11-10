package vertexai

import (
	"context"

	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/genai"
	go_genai "google.golang.org/genai"
)

type vertexAI struct {
	genai.GenAI
}

func New(ctx context.Context, model, project, location, instructions string) (llm.LLM, error) {
	client, err := go_genai.NewClient(ctx, &go_genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  go_genai.BackendVertexAI,
	})
	if err != nil {
		return nil, err
	}

	return &vertexAI{
		GenAI: genai.GenAI{
			Model:        model,
			Instructions: instructions,
			Client:       client,
			Ctx:          ctx,
		},
	}, nil
}

func (v *vertexAI) Ask(prompt string, cmd llm.Command) (*llm.Response, error) {
	return v.GenAI.Ask(prompt, cmd, "Vertex AI")
}

func (v *vertexAI) SetModel(model string) error {
	return v.GenAI.SetModel(model, "Vertex AI")
}
