package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/pkg/browser"
	"github.com/ionut-t/perp/pkg/update"
)

func (m model) checkForUpdates() tea.Cmd {
	return func() tea.Msg {
		if !m.config.AutoUpdateEnabled() {
			return nil
		}

		checker := update.New(version.Version(), m.config.Storage(), m.config.UpdateCheckIntervalHours())
		release, err := checker.CheckForUpdate()
		if err != nil {
			return nil
		}

		return updateAvailableMsg{release: release}
	}
}

func (m *model) openReleaseNotes() tea.Cmd {
	if m.latestRelease == nil {
		return nil
	}

	if err := browser.Open(m.latestRelease.ReleaseURL); err != nil {
		return m.errorNotification(err)
	}

	return nil
}

func (m *model) dismissUpdate() tea.Cmd {
	if m.latestRelease == nil {
		return nil
	}

	checker := update.New("", m.config.Storage(), 0)
	_ = checker.DismissUpdate()
	m.content.SetLatestReleaseInfo(nil)
	m.latestRelease = nil

	return nil
}
