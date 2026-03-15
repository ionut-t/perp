package splitview

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// ClearMsg is sent after a delay to clear success/error messages
type ClearMsg struct{}

// EditorClosedMsg is sent when an external editor closes
type EditorClosedMsg struct{}

// ClearMessages returns a command that sends ClearMsg after 2 seconds
func ClearMessages() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return ClearMsg{}
	})
}
