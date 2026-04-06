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
	SetModel(model string) error
}

func ExtractQuery(text string) string {
	const sqlFence = "```sql"
	const fence = "```"

	var startIndex int
	if idx := strings.Index(strings.ToLower(text), sqlFence); idx != -1 {
		startIndex = idx + len(sqlFence)
	} else if idx := strings.Index(text, fence); idx != -1 {
		startIndex = idx + len(fence)
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
