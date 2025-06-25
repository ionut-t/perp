package vertexai

import (
	"context"
	"errors"
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

func New(projectID, location, model, instructions string) (*vertexAI, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})

	if err != nil {
		return &vertexAI{}, err
	}

	return &vertexAI{
		model:        model,
		instructions: instructions,
		client:       client,
		ctx:          ctx,
	}, nil
}

func (v *vertexAI) Ask(prompt string) (*llm.Response, error) {
	if v.model == "" {
		return nil, errors.New("vertex AI model is required")
	}

	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(v.ctx, timeout)
	defer cancel()

	instructions := v.getInstructions() + "\n" + prompt
	result, err := v.client.Models.GenerateContent(
		ctx,
		v.model,
		genai.Text(instructions),
		nil,
	)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.New("request timed out after " + timeout.String())
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

	text := sanitise(result.Text())

	return &llm.Response{
		Response: text,
		Time:     time.Now(),
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

func sanitise(text string) string {
	text = strings.TrimSpace(text)

	sqlPrefixes := []string{"SQL: ", "sql: ", "Sql: ", "```sql", "```"}
	for _, prefix := range sqlPrefixes {
		text = strings.TrimPrefix(text, prefix)
	}

	sqlSuffixes := []string{"```", "```sql"}
	for _, suffix := range sqlSuffixes {
		text = strings.TrimSuffix(text, suffix)
	}

	return strings.TrimSpace(text)
}
