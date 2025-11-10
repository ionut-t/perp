package llm_factory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/gemini"
	"github.com/ionut-t/perp/pkg/llm/vertexai"
)

var (
	ErrNoProviderConfigured = errors.New("no LLM provider configured")
	ErrInvalidProvider      = errors.New("unsupported LLM provider")
	ErrMissingCredentials   = errors.New("missing provider credentials")
)

// providerCredentials holds the environment variable values for different providers
type providerCredentials struct {
	geminiAPIKey      string
	vertexAIProjectID string
	vertexAILocation  string
	hasGemini         bool
	hasVertexAI       bool
}

// loadCredentials reads and validates environment variables
func loadCredentials() *providerCredentials {
	creds := &providerCredentials{
		geminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		vertexAIProjectID: os.Getenv("VERTEXAI_PROJECT_ID"),
		vertexAILocation:  os.Getenv("VERTEXAI_LOCATION"),
	}

	creds.hasGemini = creds.geminiAPIKey != ""
	creds.hasVertexAI = creds.vertexAIProjectID != "" && creds.vertexAILocation != ""

	return creds
}

// detectProvider automatically detects which provider to use based on available credentials
func (c *providerCredentials) detectProvider() (string, error) {
	if c.hasGemini {
		return "gemini", nil
	}
	if c.hasVertexAI {
		return "vertexai", nil
	}
	return "", fmt.Errorf("%w: set GEMINI_API_KEY or both VERTEXAI_PROJECT_ID and VERTEXAI_LOCATION", ErrNoProviderConfigured)
}

// validateProvider checks if credentials exist for the specified provider
func (c *providerCredentials) validateProvider(provider string) error {
	switch provider {
	case "gemini":
		if !c.hasGemini {
			return fmt.Errorf("%w for Gemini: GEMINI_API_KEY not set", ErrMissingCredentials)
		}
	case "vertexai":
		if !c.hasVertexAI {
			missing := []string{}
			if c.vertexAIProjectID == "" {
				missing = append(missing, "VERTEXAI_PROJECT_ID")
			}
			if c.vertexAILocation == "" {
				missing = append(missing, "VERTEXAI_LOCATION")
			}
			return fmt.Errorf("%w for Vertex AI: %s not set", ErrMissingCredentials, strings.Join(missing, " and "))
		}
	default:
		return fmt.Errorf("%w: %s (supported: gemini, vertexai)", ErrInvalidProvider, provider)
	}

	return nil
}

func New(ctx context.Context, cfg config.Config, instructions string) (llm.LLM, error) {
	creds := loadCredentials()

	provider, err := cfg.GetLLMProvider()
	if err != nil || provider == "" {
		provider, err = creds.detectProvider()
		if err != nil {
			return nil, err
		}
	}

	provider = strings.ToLower(strings.TrimSpace(provider))

	if err := creds.validateProvider(provider); err != nil {
		return nil, err
	}

	model, err := cfg.GetLLMModel()
	if err != nil {
		return nil, err
	}

	switch provider {
	case "gemini":
		return gemini.New(ctx, model, creds.geminiAPIKey, instructions)
	case "vertexai":
		return vertexai.New(ctx, model, creds.vertexAIProjectID, creds.vertexAILocation, instructions)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, provider)
	}
}
