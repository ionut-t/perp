package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/tui/content"
	historyView "github.com/ionut-t/perp/tui/history"
	"github.com/ionut-t/perp/tui/servers"
)

// Key bindings for the TUI application
var (
	yankCell = key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yank selected cell"),
	)

	yankRow = key.NewBinding(
		key.WithKeys("Y"),
		key.WithHelp("Y", "yank selected row (copies selected row as JSON to clipboard)"),
	)

	previousCell = key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("← / h", "previous cell"),
	)

	nextCell = key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→ / l", "next cell"),
	)

	changeFocused = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "change focus between editor and main content"),
	)

	executeQuery = key.NewBinding(
		key.WithKeys("alt+enter", "ctrl+s"),
		key.WithHelp("alt+enter/ctrl+s", "execute query (no matter the editor mode)"),
	)

	previousHistory = key.NewBinding(
		key.WithKeys("shift+up"),
		key.WithHelp("shift+↑", "previous history log"),
	)

	nextHistory = key.NewBinding(
		key.WithKeys("shift+down"),
		key.WithHelp("shift+↓", "next history log"),
	)

	enterCommand = key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "enter command mode (available when the editor is not focused)"),
	)

	viewHistoryEntries = key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "view history logs"),
	)

	openRelease = key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "open release notes in browser"),
	)

	dismissUpdate = key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "dismiss release"),
	)
)

// tryHandleKeyPress processes keyboard input in the main view
// Returns (model, cmd, handled) - handled=true if key was processed
func (m model) tryHandleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	var updatedModel tea.Model

	switch {
	case key.Matches(msg, keymap.Quit):
		updatedModel, cmd = m.handleQuitKey()
		return updatedModel, cmd, true

	case key.Matches(msg, changeFocused):
		updatedModel, cmd = m.handleChangeFocusKey()
		return updatedModel, cmd, true

	case key.Matches(msg, enterCommand):
		updatedModel, cmd = m.handleEnterCommandKey()
		return updatedModel, cmd, true

	case key.Matches(msg, keymap.Insert):
		updatedModel, cmd = m.handleInsertKey()
		return updatedModel, cmd, false

	case key.Matches(msg, keymap.Submit):
		updatedModel, cmd = m.handleSubmitKey()
		return updatedModel, cmd, true

	case key.Matches(msg, executeQuery):
		updatedModel, cmd = m.handleExecuteQueryKey()
		return updatedModel, cmd, true

	case key.Matches(msg, keymap.Cancel):
		updatedModel, cmd = m.handleCancelKey()
		return updatedModel, cmd, true

	case key.Matches(msg, previousHistory):
		updatedModel, cmd = m.handlePreviousHistoryKey()
		return updatedModel, cmd, false

	case key.Matches(msg, nextHistory):
		updatedModel, cmd = m.handleNextHistoryKey()
		return updatedModel, cmd, false

	case key.Matches(msg, viewHistoryEntries):
		updatedModel, cmd = m.handleViewHistoryKey()
		return updatedModel, cmd, true

	case key.Matches(msg, openRelease):
		updatedModel, cmd = m.openReleaseNotes()
		return updatedModel, cmd, true

	case key.Matches(msg, dismissUpdate):
		updatedModel, cmd = m.dismissUpdate()
		return updatedModel, cmd, true
	}

	// Key not handled - let it fall through to component updates
	return m, nil, false
}

// handleQuitKey processes the quit key
func (m model) handleQuitKey() (tea.Model, tea.Cmd) {
	if m.error != nil {
		m.serverSelection = servers.New(m.config.Storage())
		_, cmd := m.serverSelection.Update(nil)

		m.view = viewServers
		m.error = nil
		return m, cmd
	}

	if m.focused == focusedContent && m.content.IsViewChangeRequired() ||
		m.editor.IsNormalMode() && m.fullScreen {
		m.fullScreen = false

		m.updateSize()
		contentModel, cmd := m.content.Update(content.ResizeMsg{})
		m.content = contentModel.(content.Model)

		return m, cmd
	}

	if m.view == viewHelp {
		m.view = viewMain
		m.focused = focusedEditor
		m.editor.Focus()
		return m, nil
	}

	return m, nil
}

// handleChangeFocusKey switches focus between editor and content
func (m model) handleChangeFocusKey() (tea.Model, tea.Cmd) {
	if m.view == viewMain && !m.editor.IsInsertMode() {
		switch m.focused {
		case focusedEditor:
			m.focused = focusedContent
			m.editor.Blur()
		case focusedContent:
			m.focused = focusedEditor
			m.editor.Focus()
		}

		if m.fullScreen {
			m.updateSize()
		}

		_, cmd := m.content.Update(nil)

		return m, tea.Batch(
			cmd,
			m.editor.CursorBlink(),
			utils.Dispatch(content.ResizeMsg{}),
		)
	}

	return m, nil
}

// handleEnterCommandKey enters command mode
func (m model) handleEnterCommandKey() (tea.Model, tea.Cmd) {
	if m.view == viewMain && m.editor.IsNormalMode() {
		m.focused = focusedCommand
		m.editor.Blur()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			m.command.Focus(),
			cmd,
		)
	}

	return m, nil
}

// handleInsertKey switches to insert mode
func (m model) handleInsertKey() (tea.Model, tea.Cmd) {
	if m.view == viewMain && m.focused == focusedContent {
		m.focused = focusedEditor
		m.editor.Focus()
		m.editor.SetInsertMode()

		_, cmd := m.editor.Update(nil)

		return m, tea.Batch(
			cmd,
			m.editor.CursorBlink(),
		)
	}

	return m, nil
}

// handleSubmitKey submits the current query (normal mode only)
func (m model) handleSubmitKey() (tea.Model, tea.Cmd) {
	if m.editor.IsNormalMode() {
		content := m.editor.GetCurrentContent()

		if content == "" {
			return m, nil
		}

		if !m.loading {
			m.loading = true
			m.resetHistory()
			m.addToHistory()
			m.fullScreen = false
			m.updateSize()

			return m, m.sendQueryCmd()
		}
	}

	return m, nil
}

// handleExecuteQueryKey executes query regardless of editor mode
func (m model) handleExecuteQueryKey() (tea.Model, tea.Cmd) {
	if !m.loading {
		m.loading = true
		m.resetHistory()
		m.addToHistory()
		m.fullScreen = false
		m.updateSize()

		return m, m.sendQueryCmd()
	}

	return m, nil
}

// handleCancelKey cancels current operation
func (m model) handleCancelKey() (tea.Model, tea.Cmd) {
	if m.view == viewMain && m.focused == focusedEditor {
		m.resetHistory()

		if m.editor.IsNormalMode() {
			if m.editor.IsFocused() {
				m.focused = focusedContent
				m.editor.Blur()
			}
		}
	}

	return m, nil
}

// handlePreviousHistoryKey navigates to previous history entry
func (m model) handlePreviousHistoryKey() (tea.Model, tea.Cmd) {
	if m.editor.IsFocused() && len(m.historyLogs) > 0 {
		m.previousHistory()
	}

	return m, nil
}

// handleNextHistoryKey navigates to next history entry
func (m model) handleNextHistoryKey() (tea.Model, tea.Cmd) {
	if m.editor.IsFocused() && m.historyNavigating {
		m.nextHistory()
	}

	return m, nil
}

// handleViewHistoryKey opens the history view
func (m model) handleViewHistoryKey() (tea.Model, tea.Cmd) {
	if entries, err := history.Get(m.config.Storage()); err != nil {
		m.content.SetError(err)
	} else {
		m.view = viewHistory
		m.focused = focusedHistory
		m.editor.Blur()
		m.historyLogs = entries

		m.history = historyView.New(entries, m.width, m.height)
	}

	return m, nil
}
