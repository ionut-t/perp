package llm

import "time"

type Response struct {
	Response string
	Time     time.Time
}

type LLM interface {
	Ask(prompt string) (*Response, error)
	AppendInstructions(instructions string)
	ResetInstructions()
}
