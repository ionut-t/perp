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

func New(projectID, location, model, instructions string) (llm.LLM, error) {
	ctx := context.Background()

	client, err := go_genai.NewClient(ctx, &go_genai.ClientConfig{
		Project:  projectID,
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
