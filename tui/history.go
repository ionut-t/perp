package tui

import "github.com/ionut-t/perp/pkg/history"

func (m *model) addToHistory() {
	if logs, err := history.Add(m.editor.GetCurrentContent(), m.config.Storage()); err == nil {
		m.historyLogs = logs
	}
}

func (m *model) resetHistory() {
	m.historyNavigating = false
	m.currentHistoryIndex = -1
	m.originalEditorContent = ""
}

func (m *model) previousHistory() {
	if !m.historyNavigating {
		m.originalEditorContent = m.editor.GetCurrentContent()
		m.currentHistoryIndex = -1
		m.historyNavigating = true
	}

	// Move to older entry
	if m.currentHistoryIndex < len(m.historyLogs)-1 {
		m.currentHistoryIndex++
		m.editor.SetContent(m.historyLogs[m.currentHistoryIndex].Query)
		_ = m.editor.SetCursorPositionEnd()
	}
}

func (m *model) nextHistory() {
	// Move to newer entry
	if m.currentHistoryIndex > 0 {
		m.currentHistoryIndex--
		m.editor.SetContent(m.historyLogs[m.currentHistoryIndex].Query)
		_ = m.editor.SetCursorPositionEnd()
	} else if m.currentHistoryIndex == 0 {
		m.currentHistoryIndex = -1
		m.editor.SetContent(m.originalEditorContent)
		_ = m.editor.SetCursorPositionEnd()
		m.historyNavigating = false
	}
}
