package vertexai

import (
	"context"
	"errors"
	"os"

	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/genai"
	go_genai "google.golang.org/genai"
)

type vertexAI struct {
	genai.GenAI
}

func New(model, instructions string) (llm.LLM, error) {
	ctx := context.Background()

	projectID := os.Getenv("VERTEXAI_PROJECT_ID")
	if projectID == "" {
		return nil, errors.New("VERTEXAI_PROJECT_ID environment variable not set")
	}

	location := os.Getenv("VERTEXAI_LOCATION")
	if location == "" {
		return nil, errors.New("VERTEXAI_LOCATION environment variable not set")
	}

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
