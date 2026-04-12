package tui

import (
	"fmt"
	"strconv"
)

// requireLLM validates that the LLM is properly initialized
func (m *model) requireLLM() error {
	if m.llmError != nil {
		return fmt.Errorf("LLM is not configured: %w", m.llmError)
	}
	if m.llm == nil {
		return fmt.Errorf("LLM instance is not initialized")
	}
	return nil
}

// mustParsePort converts a port string to int, defaulting to 5432 on failure.
func mustParsePort(s string) int {
	p, err := strconv.Atoi(s)
	if err != nil {
		return 5432
	}
	return p
}

// toggleStatus returns "ON" or "OFF" based on boolean value
func toggleStatus(enabled bool) string {
	if enabled {
		return "ON"
	}
	return "OFF"
}
