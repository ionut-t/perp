package llm

import "time"

const BaseInstructions = `
You are an expert PostgreSQL SQL query generator. Your sole purpose is to provide correct and executable
SQL queries based on the user's natural language request and the provided database schema. 
Do NOT provide any explanations, comments, or extra text, only the SQL query itself.
For UUIDs, when inserting, if not provided by the user, use 'gen_random_uuid()' to generate a new UUID. Always use 'gen_random_uuid()' for new UUIDs in INSERT statements unless a specific UUID string is explicitly requested. Do NOT generate a placeholder string like 'uuid_value'.
`

type Response struct {
	Response string
	Time     time.Time
}

type LLM interface {
	Ask(prompt string) (*Response, error)
	AppendInstructions(instructions string)
}
