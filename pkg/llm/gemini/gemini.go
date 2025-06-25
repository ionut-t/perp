package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/llm"
	"google.golang.org/genai"
)

type gemini struct {
	apiKey               string
	model                string
	instructions         string
	dbSchemaInstructions string
	client               *genai.Client
	ctx                  context.Context
}

func New(apiKey, model, instructions string) (llm.LLM, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		return nil, errors.New("failed to create Gemini client: " + err.Error())
	}

	return &gemini{
		apiKey:       apiKey,
		model:        model,
		instructions: instructions,
		client:       client,
		ctx:          ctx,
	}, nil
}

func (g *gemini) Ask(prompt string) (*llm.Response, error) {
	timeout := 30 * time.Second

	ctx, cancel := context.WithTimeout(g.ctx, timeout)
	defer cancel()

	if g.model == "" {
		return nil, errors.New("a Gemini model is required")
	}

	if g.apiKey == "" {
		return nil, errors.New("API key for Gemini is required")
	}

	instructions := g.getInstructions() + "\n" + prompt

	result, err := g.client.Models.GenerateContent(
		ctx,
		g.model,
		genai.Text(instructions),
		nil,
	)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request timed out after %v", timeout)
		}

		if errors.As(err, &genai.APIError{}) {
			apiErr := err.(genai.APIError)
			return nil, errors.New(apiErr.Message)
		}

		return nil, err
	}

	if result == nil {
		return nil, errors.New("received nil response from Gemini")
	}

	text := sanitise(result.Text())

	return &llm.Response{
		Response: text,
		Time:     time.Now(),
	}, nil
}

func (g *gemini) AppendInstructions(instructions string) {
	g.dbSchemaInstructions = instructions
}

func (g *gemini) ResetInstructions() {
	g.dbSchemaInstructions = ""
}

func (g *gemini) getInstructions() string {
	return strings.TrimSpace(g.instructions + "\n" + g.dbSchemaInstructions)
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
