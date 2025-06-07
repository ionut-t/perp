package gemini

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/llm"
)

// --- Gemini API Request and Response Structs ---
// part represents a part of the content (e.g., text)
type part struct {
	Text string `json:"text"`
}

// content represents a piece of content (a list of parts)
type content struct {
	Parts []part `json:"parts"`
}

// generateContentRequest is the structure for the request body to Gemini
type generateContentRequest struct {
	Contents []content `json:"contents"`
}

// Candidate represents a generated response candidate
type candidate struct {
	Content content `json:"content"`
}

// generateContentResponse is the structure for the response body from Gemini
type generateContentResponse struct {
	Candidates []candidate `json:"candidates"`
}

type Gemini struct {
	apiKey       string
	instructions string
}

func New(apiKey string) *Gemini {
	return &Gemini{
		apiKey:       apiKey,
		instructions: llm.BaseInstructions,
	}
}

func (g *Gemini) Ask(prompt string) (*llm.Response, error) {
	apiKey := g.apiKey

	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

	requestBody := generateContentRequest{
		Contents: []content{
			{
				Parts: []part{
					{Text: g.instructions + "\n" + prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for error details
		return nil, fmt.Errorf("API returned non-200 status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var geminiResponse generateContentResponse
	err = json.Unmarshal(body, &geminiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w - Body: %s", err, string(body))
	}

	text := geminiResponse.Candidates[0].Content.Parts[0].Text

	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "SQL: ")
	text = strings.TrimPrefix(text, "sql: ")
	text = strings.TrimPrefix(text, "Sql: ")
	text = strings.TrimPrefix(text, "```sql")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSuffix(text, "```sql")
	text = strings.TrimSpace(text)

	return &llm.Response{
		Response: text,
		Time:     time.Now(),
	}, nil
}

func (g *Gemini) AppendInstructions(instructions string) {
	g.instructions = llm.BaseInstructions + "\n" + instructions
}
