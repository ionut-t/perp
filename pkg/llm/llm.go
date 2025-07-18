package llm

import (
	"strings"
	"time"
)

type Command int

const (
	Ask Command = iota
	Explain
	Optimise
	Fix
	Info
)

var LLMKeywords = [...]string{
	"/ask",
	"/add",
	"/remove",
}

type Response struct {
	Response string
	Time     time.Time
	Command  Command
}

type LLM interface {
	Ask(prompt string, cmd Command) (*Response, error)
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

func ExtractQuery(text string) string {
	startIndex := strings.Index(strings.ToLower(text), "```sql")
	if startIndex == -1 {
		startIndex = strings.Index(text, "```")
	}

	if startIndex != -1 {
		startIndex += len("```sql")
	} else {
		return text
	}

	endIndex := strings.Index(text[startIndex:], "```")
	if endIndex != -1 {
		endIndex += startIndex
	} else {
		endIndex = len(text)
	}

	query := text[startIndex:endIndex]
	query = strings.TrimSpace(query)

	return strings.TrimSpace(query)
}

func IsAskCommand(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(text, "/ask")
}

func IsExplainCommand(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	return strings.Contains(text, "-- explain") || strings.Contains(text, "--explain")
}

func IsOptimiseCommand(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	return strings.Contains(text, "-- optimise") ||
		strings.Contains(text, "--optimise") ||
		strings.Contains(text, "-- optimize") ||
		strings.Contains(text, "--optimize")
}

func IsFixCommand(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	return strings.Contains(text, "-- fix") || strings.Contains(text, "--fix")
}
