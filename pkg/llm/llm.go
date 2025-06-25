package llm

import (
	"strings"
	"time"
)

type Response struct {
	Response string
	Time     time.Time
}

type LLM interface {
	Ask(prompt string) (*Response, error)
	AppendInstructions(instructions string)
	ResetInstructions()
	SetModel(model string)
}

func SanitiseResponse(text string) string {
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
