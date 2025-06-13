package utils

import (
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ClearNotificationMsg struct{}

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

// ClearNotification returns a command that triggers a notification clear after a specified duration.
func ClearNotification() tea.Cmd {
	return tea.Tick(
		time.Second*2,
		func(t time.Time) tea.Msg {
			return ClearNotificationMsg{}
		},
	)
}

// HandleDataExport processes query results and selected rows for export.
func HandleDataExport(queryResults []map[string]any, rows []int, all bool) (any, error) {
	if queryResults != nil {
		var data any
		if len(rows) > 1 {
			data = make([]map[string]any, 0)

			for _, rowIdx := range rows {
				idx := rowIdx - 1
				if idx >= 0 && idx < len(queryResults) {
					data = append(data.([]map[string]any), queryResults[idx])
				}
			}
		} else if len(rows) == 1 {
			idx := rows[0] - 1
			if idx >= 0 && idx < len(queryResults) {
				data = queryResults[idx]
			}
		}

		if all {
			data = make([]map[string]any, 0)
			data = append(data.([]map[string]any), queryResults...)
		}

		return data, nil
	}

	return nil, errors.New("no query results to export")
}

func Dispatch(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}
