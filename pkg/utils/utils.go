package utils

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ClearMsg struct{}

// ParseTableNames is a helper function that extracts and deduplicates table names from a raw input string.
func ParseTableNames(input string) []string {
	var tables []string
	seen := make(map[string]bool)

	// Split the input string by common delimiters like comma, space, tab, or newline.
	for _, table := range strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	}) {
		trimmedTable := strings.TrimSpace(table)
		if trimmedTable != "" && !seen[trimmedTable] {
			tables = append(tables, trimmedTable)
			seen[trimmedTable] = true
		}
	}
	return tables
}

// ClearAfter returns a command that triggers a notification clear after a specified duration.
func ClearAfter(duration time.Duration) tea.Cmd {
	return tea.Tick(
		duration,
		func(t time.Time) tea.Msg {
			return ClearMsg{}
		},
	)
}

func Dispatch(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func Duration(duration time.Duration) string {
	switch {
	case duration < time.Millisecond:
		return fmt.Sprintf("%dµs", duration.Microseconds())
	case duration < time.Second:
		return fmt.Sprintf("%dms", duration.Milliseconds())
	default:
		return fmt.Sprintf("%.3fs", duration.Seconds())
	}
}
