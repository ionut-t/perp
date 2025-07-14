package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/pkg/browser"
	"github.com/ionut-t/perp/pkg/update"
)

type updateAvailableMsg struct {
	release *update.LatestReleaseInfo
}

func (m model) checkForUpdates() tea.Cmd {
	return func() tea.Msg {
		if !m.config.AutoUpdateEnabled() {
			return nil
		}

		checker := update.New(version.Version(), m.config.Storage())
		release, err := checker.CheckForUpdate()

		if err != nil {
			return nil
		}

		return updateAvailableMsg{release: release}
	}
}

func (m *model) openReleaseNotes() (*model, tea.Cmd) {
	release, ok := m.content.GetLatestReleaseInfo()

	if !ok {
		return m, nil
	}

	if err := browser.Open(release.ReleaseURL); err != nil {
		return m, m.errorNotification(err)
	}

	return m, nil
}

func (m *model) dismissUpdate() (*model, tea.Cmd) {
	if _, ok := m.content.GetLatestReleaseInfo(); !ok {
		return m, nil
	}

	checker := update.New("", m.config.Storage())
	_ = checker.DismissUpdate()
	m.content.SetLatestReleaseInfo(nil)

	return m, nil
}
