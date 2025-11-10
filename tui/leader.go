package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/internal/leader"
	"github.com/ionut-t/perp/internal/whichkey"
)

// Leader key and which-key handlers

// updateMenuContext builds the current context from app state and updates the registry
func (m *model) updateMenuContext() {
	ctx := &whichkey.MenuContext{
		// Connection state
		IsConnected: m.server.Name != "",
		ServerName:  m.server.Name,

		// View state
		InServersView:   m.view == viewServers,
		InExportView:    m.view == viewExportData,
		InMainView:      m.view == viewMain,
		InHistoryView:   m.view == viewHistory,
		FocusedOnTable:  m.focused == focusedContent,
		FocusedOnEditor: m.focused == focusedEditor,
		IsFullScreen:    m.fullScreen,
		IsHelpVisible:   m.view == viewHelp,

		// Data state
		HasQueryResults: len(m.content.GetQueryResults()) > 0,
		HasHistory:      len(m.historyLogs) > 0,
		ResultCount:     len(m.content.GetQueryResults()),
		HistoryCount:    len(m.historyLogs),

		// Feature availability
		LLMEnabled:      m.llm != nil,
		LLMSchemaShared: m.server.ShareDatabaseSchemaLLM,
	}

	m.menuRegistry.UpdateContext(ctx)
	m.whichKeyMenu.SetContext(ctx)
}

func (m model) handleLeaderKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Update menu context before showing menu
	m.updateMenuContext()

	state, cmd := m.leaderMgr.HandleKey(msg.String())

	if state == leader.StateWaiting {
		// Just activated leader mode, wait for next key or timeout
		return m, cmd
	}

	return m, cmd
}

func (m model) handleLeaderSequence(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.leaderMgr.Reset()
		return m, nil
	}

	// When a leader sequence is active, always start from the root menu.
	// The root menu is context-aware and will return appropriate items.
	// Submenu navigation is handled by the whichKeyMenu component itself.
	currentMenu := m.menuRegistry.GetRootMenu()
	items := currentMenu.GetItems()

	// Check if the pressed key matches any menu item
	for _, item := range items {
		if item.Key == msg.String() {
			m.leaderMgr.Reset()
			if item.Action.IsSubmenu() {
				// Show submenu immediately
				submenu := item.Action.Execute().(whichkey.ShowSubmenuMsg)
				m.whichKeyMenu.SetMenu(submenu.Menu)
				m.whichKeyMenu.Show()
				m.showingMenu = true
				return m, nil
			} else {
				// Execute direct action
				return m.Update(item.Action.Execute())
			}
		}
	}

	// Key not found in menu - ignore and reset
	m.leaderMgr.Reset()
	return m, nil
}

func (m model) handleWhichKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.whichKeyMenu, cmd = m.whichKeyMenu.Update(msg)
	return m, cmd
}

func (m model) canTriggerLeaderKey() bool {
	switch m.view {
	case viewMain:
		return (!m.editor.IsFocused() || m.editor.IsNormalMode()) && !m.isPromptActive
	case viewServers:
		return m.serverSelection.CanTriggerLeaderKey()
	case viewExportData:
		return m.exportData.CanTriggerLeaderKey()
	case viewHelp:
		return true
	case viewHistory:
		return m.history.CanTriggerLeaderKey()
	default:
		return true
	}
}
