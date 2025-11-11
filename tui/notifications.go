package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/pkg/utils"
)

// successNotification displays a success message
func (m *model) successNotification(msg string) tea.Cmd {
	m.notification = styles.Success.Render(msg)
	return utils.ClearAfter(NotificationDuration)
}

// errorNotification displays an error message
func (m *model) errorNotification(err error) tea.Cmd {
	m.notification = styles.Error.Render(err.Error())
	return utils.ClearAfter(NotificationDuration)
}
