package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/perp/pkg/utils"
)

// successNotification displays a success message
func (m *model) successNotification(msg string) tea.Cmd {
	m.notification = m.styles.Success.Render(msg)
	return utils.ClearAfter(NotificationDuration)
}

// errorNotification displays an error message
func (m *model) errorNotification(err error) tea.Cmd {
	m.notification = m.styles.Error.Render(err.Error())
	return utils.ClearAfter(NotificationDuration)
}
