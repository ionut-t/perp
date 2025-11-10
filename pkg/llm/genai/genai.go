package genai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/llm"
	"google.golang.org/genai"
)

type GenAI struct {
	Model                string
	Instructions         string
	DBSchemaInstructions string
	Client               *genai.Client
	Ctx                  context.Context
}

func (g *GenAI) Ask(prompt string, cmd llm.Command, providerName string) (*llm.Response, error) {
	timeout := 30 * time.Second

	if g.Model == "" {
		return nil, fmt.Errorf("no %s model specified", providerName)
	}

	ctx, cancel := context.WithTimeout(g.Ctx, timeout)
	defer cancel()

	instructions := g.getInstructions() + "\n" + "Prompt: \n" + prompt

	result, err := g.Client.Models.GenerateContent(
		ctx,
		g.Model,
		genai.Text(instructions),
		nil,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request timed out after %v for %s", timeout, providerName)
		}

		if errors.As(err, &genai.APIError{}) {
			apiErr := err.(genai.APIError)
			return nil, errors.New(apiErr.Message)
		}

		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("received nil response from %s", providerName)
	}

	text := result.Text()

	if strings.HasPrefix(text, "INFO:") {
		cmd = llm.Info
	}

	if cmd == llm.Ask {
		text = llm.SanitiseResponse(text)
	}

	return &llm.Response{
		Response: text,
		Time:     time.Now(),
		Command:  cmd,
	}, nil
}

func (g *GenAI) AppendInstructions(instructions string) {
	g.DBSchemaInstructions = instructions
}

func (g *GenAI) ResetInstructions() {
	g.DBSchemaInstructions = ""
}

func (g *GenAI) getInstructions() string {
	return strings.TrimSpace(g.Instructions + "\n" + g.DBSchemaInstructions)
}

func (g *GenAI) SetModel(model, providerName string) error {
	timeout := 30 * time.Second

	if model == "" {
		return fmt.Errorf("no %s model specified", providerName)
	}

	ctx, cancel := context.WithTimeout(g.Ctx, timeout)
	defer cancel()

	_, err := g.Client.Models.GenerateContent(
		ctx,
		model,
		genai.Text("Test, say nothing."),
		nil,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("request timed out after %v for %s", timeout, providerName)
		}

		if errors.As(err, &genai.APIError{}) {
			apiErr := err.(genai.APIError)
			return errors.New(apiErr.Message)
		}

		return err
	}

	g.Model = model

	return nil
}
