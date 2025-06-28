package vertexai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/llm"
	"google.golang.org/genai"
)

type vertexAI struct {
	model                string
	instructions         string
	dbSchemaInstructions string
	client               *genai.Client
	ctx                  context.Context
}

func New(projectID, location, model, instructions string) (llm.LLM, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})

	if err != nil {
		return nil, err
	}

	return &vertexAI{
		model:        model,
		instructions: instructions,
		client:       client,
		ctx:          ctx,
	}, nil
}

func (v *vertexAI) Ask(prompt string, cmd llm.Command) (*llm.Response, error) {
	if v.model == "" {
		return nil, errors.New("no Vertex AI model specified")
	}

	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(v.ctx, timeout)
	defer cancel()

	instructions := v.getInstructions() + "\n" + "Prompt: \n" + prompt
	result, err := v.client.Models.GenerateContent(
		ctx,
		v.model,
		genai.Text(instructions),
		nil,
	)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request timed out after %v for Vertex AI", timeout)
		}

		if errors.As(err, &genai.APIError{}) {
			apiErr := err.(genai.APIError)
			return nil, errors.New(apiErr.Message)
		}

		return nil, err
	}

	if result == nil {
		return nil, errors.New("received nil response from Vertex AI")
	}

	text := result.Text()

	if cmd != llm.Explain {
		text = llm.SanitiseResponse(text)
	}

	return &llm.Response{
		Response: text,
		Time:     time.Now(),
		Command:  cmd,
	}, nil
}

func (v *vertexAI) AppendInstructions(instructions string) {
	v.dbSchemaInstructions = instructions
}

func (v *vertexAI) ResetInstructions() {
	v.dbSchemaInstructions = ""
}

func (v *vertexAI) getInstructions() string {
	return strings.TrimSpace(v.instructions + "\n" + v.dbSchemaInstructions)
}

func (v *vertexAI) SetModel(model string) {
	v.model = model
}
