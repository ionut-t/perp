package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// resetEditor clears the editor content and resets its state
func (m *model) resetEditor() tea.Cmd {
	m.editor.SetContent("")
	ed, cmd := m.editor.Update(nil)
	m.editor = ed
	return cmd
}

// applyQueryToEditor sets a query in the editor and prepares it for editing
func (m *model) applyQueryToEditor(query string) tea.Cmd {
	m.editor.SetContent(query)
	m.editor.Focus()
	_ = m.editor.SetCursorPositionEnd()
	m.view = viewMain
	m.focused = focusedEditor
	ed, cmd := m.editor.Update(nil)
	m.editor = ed
	return tea.Batch(cmd, m.editor.CursorBlink())
}

// focusEditor changes focus to the editor component
func (m *model) focusEditor() {
	m.focused = focusedEditor
	m.editor.Focus()
}

// finishQueryExecution sets common state after query execution
func (m *model) finishQueryExecution() {
	m.loading = false
	m.focused = focusedContent
	m.editor.Blur()
	m.editor.SetNormalMode()
}

// setHighlightedKeywords determines which keywords to highlight based on editor content
func (m model) setHighlightedKeywords() map[string]lipgloss.Style {
	content := m.editor.GetCurrentContent()

	if strings.HasPrefix(content, "/") {
		return m.llmKeywords
	}

	if strings.HasPrefix(content, "\\") {
		return m.psqlCommands
	}

	return nil
}
