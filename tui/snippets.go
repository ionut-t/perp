package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	pkgSnippets "github.com/ionut-t/perp/pkg/snippets"
	snippetsStore "github.com/ionut-t/perp/store/snippets"
	snippetsView "github.com/ionut-t/perp/tui/snippets"
)

func (m model) applySnippet(msg snippetsView.SelectedMsg) (tea.Model, tea.Cmd) {
	m.view = viewMain
	m.focusEditor()
	return m, m.applyQueryToEditor(msg.Snippet.Query)
}

func (m model) saveSnippet(name string) (tea.Model, tea.Cmd) {
	m.isPromptActive = false

	query := m.editor.GetCurrentContent()
	if strings.TrimSpace(query) == "" {
		return m, m.errorNotification(fmt.Errorf("cannot save empty query as snippet"))
	}

	scope := snippetsStore.ScopeServer

	globalSnippetsPath := pkgSnippets.GetGlobalSnippetsPath(m.config.Storage())
	serverSnippetsPath := pkgSnippets.GetServerSnippetsPath(m.config.Storage(), m.server.Name)
	m.snippetsStore = snippetsStore.New(globalSnippetsPath, serverSnippetsPath, m.config.Editor())

	if err := m.snippetsStore.Create(name, query, scope); err != nil {
		return m, m.errorNotification(err)
	}

	m.focusEditor()

	return m, m.successNotification("Snippet saved")
}

func (m *model) listSnippets() {
	globalSnippetsPath := pkgSnippets.GetGlobalSnippetsPath(m.config.Storage())
	serverSnippetsPath := pkgSnippets.GetServerSnippetsPath(m.config.Storage(), m.server.Name)
	m.snippetsStore = snippetsStore.New(globalSnippetsPath, serverSnippetsPath, m.config.Editor())

	m.view = viewSnippets
	m.focused = focusedSnippets
	m.editor.Blur()
	m.snippets = snippetsView.New(m.snippetsStore, m.server, m.width, m.height)
}
