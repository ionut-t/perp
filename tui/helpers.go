package tui

import "fmt"

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

// toggleStatus returns "ON" or "OFF" based on boolean value
func toggleStatus(enabled bool) string {
	if enabled {
		return "ON"
	}
	return "OFF"
}
