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

func (g *gemini) Ask(prompt string, cmd llm.Command) (*llm.Response, error) {
	timeout := 30 * time.Second

	if g.model == "" {
		return nil, errors.New("no Gemini model specified")
	}

	ctx, cancel := context.WithTimeout(g.ctx, timeout)
	defer cancel()

	instructions := g.getInstructions() + "\n" + "Prompt: \n" + prompt

	result, err := g.client.Models.GenerateContent(
		ctx,
		g.model,
		genai.Text(instructions),
		nil,
	)

	if err != nil {

		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request timed out after %v for Gemini", timeout)
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

func (g *gemini) AppendInstructions(instructions string) {
	g.dbSchemaInstructions = instructions
}

func (g *gemini) ResetInstructions() {
	g.dbSchemaInstructions = ""
}

func (g *gemini) getInstructions() string {
	return strings.TrimSpace(g.instructions + "\n" + g.dbSchemaInstructions)
}

func (g *gemini) SetModel(model string) {
	g.model = model
}
