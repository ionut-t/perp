package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Wrap(width int, text string) string {
	var sb strings.Builder

	lines := strings.SplitSeq(text, "\n")

	for line := range lines {
		if lipgloss.Width(line) <= width {
			sb.WriteString(line)
			continue
		}

		// Should split the line into multiple lines without breaking words
		words := strings.Fields(line)
		var currentLine strings.Builder
		for _, word := range words {
			if lipgloss.Width(currentLine.String()+" "+word) <= width {
				if currentLine.Len() > 0 {
					currentLine.WriteString(" ")
				}
				currentLine.WriteString(word)
			} else {
				if currentLine.Len() > 0 {
					sb.WriteString(currentLine.String() + "\n")
				}
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}

		if currentLine.Len() > 0 {
			sb.WriteString(currentLine.String())
		}
	}

	return sb.String()
}
