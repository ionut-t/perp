package snippets

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/store/snippets"
	"github.com/ionut-t/perp/tui/common/splitview"
)

// Wrapper to implement splitview.Item interface for Snippet
type snippetItem struct {
	*snippets.Snippet
}

func (s snippetItem) GetName() string     { return s.Name }
func (s snippetItem) GetContent() string  { return s.Content } // Show full content with metadata
func (s snippetItem) SetContent(c string) { s.Content = c }

// Adapter store to work with splitview.Store interface
type storeAdapter struct {
	snippets.Store
}

func (s *storeAdapter) Load() ([]snippetItem, error) {
	snips, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]snippetItem, len(snips))
	for i := range snips {
		items[i] = snippetItem{&snips[i]}
	}
	return items, nil
}

func (s *storeAdapter) Update(item snippetItem) error {
	return s.Store.Update(*item.Snippet)
}

func (s *storeAdapter) Delete(item snippetItem) error {
	return s.Store.Delete(*item.Snippet)
}

func (s *storeAdapter) Rename(item *snippetItem, newName string) error {
	return s.Store.Rename(item.Snippet, newName)
}

func (s *storeAdapter) GetCurrent() snippetItem {
	snip := s.GetCurrentSnippet()
	return snippetItem{&snip}
}

func (s *storeAdapter) SetCurrentName(name string) {
	s.SetCurrentSnippetName(name)
}

func (s *storeAdapter) GetPath(item snippetItem) string {
	return s.Store.GetPath(*item.Snippet)
}

type SelectedMsg struct {
	Snippet snippets.Snippet
}

type DeleteMsg struct {
	Snippet snippets.Snippet
}

type Model struct {
	*splitview.Model[snippetItem, *storeAdapter]
	server server.Server
}

type item struct {
	snippet snippets.Snippet
}

func (i item) Title() string {
	// Add scope indicator
	prefix := "󰖟 " // Global
	if i.snippet.Scope == snippets.ScopeServer {
		prefix = "󰒋 " // Server-specific
	}

	return prefix + strings.TrimSuffix(i.snippet.Name, ".sql")
}

func (i item) Description() string {
	if i.snippet.Description != "" {
		return i.snippet.Description
	}
	// Show first line of query as description if no description exists
	lines := strings.Split(i.snippet.Query, "\n")
	if len(lines) > 0 {
		desc := strings.TrimSpace(lines[0])
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		return desc
	}
	return ""
}

func (i item) FilterValue() string {
	// Allow filtering by name, description, tags, and query
	return i.snippet.Name + " " + i.snippet.Description + " " + strings.Join(i.snippet.Tags, " ") + " " + i.snippet.Query
}

type itemDelegate struct {
	styles list.DefaultItemStyles
}

func (d itemDelegate) Height() int                             { return 2 }
func (d itemDelegate) Spacing() int                            { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	title := i.Title()
	desc := i.Description()

	fn := d.styles.NormalTitle.Render
	descFn := d.styles.NormalDesc.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.SelectedTitle.Render(s...)
		}
		descFn = func(s ...string) string {
			return d.styles.SelectedDesc.Render(s...)
		}
	}

	_, _ = io.WriteString(w, fn(title)+"\n"+descFn(desc))
}

func New(store snippets.Store, server server.Server, width, height int, s styles.Styles, isDark bool) Model {
	adapter := &storeAdapter{Store: store}

	config := splitview.Config{
		EditorLanguage:      "postgres",
		PlaceholderTitle:    "No snippets available.",
		PlaceholderSubtitle: "Press '<leader>ns' to save a snippet or '<leader>c' to go back.",
		SuccessDeleteMsg:    "Snippet deleted successfully.",
		SuccessRenameMsg:    "Snippet renamed successfully.",
	}

	baseModel := splitview.New(
		adapter,
		config,
		processSnippets,
		func(m *splitview.Model[snippetItem, *storeAdapter], width int) string {
			return renderStatusBar(m, server, width)
		},
		func(m *splitview.Model[snippetItem, *storeAdapter]) string {
			return renderHelp(m)
		},
		func() tea.Msg {
			return whichkey.CloseSnippetsCmd()
		},
		width,
		height,
		s,
		isDark,
	)

	// Override list delegate for custom rendering
	items, _ := adapter.Load()
	listItems := processSnippets(items)
	delegate := itemDelegate{
		styles: styles.ListItemStyles(s, isDark),
	}

	// Get current list dimensions from base model
	currentList := baseModel.GetList()
	listWidth := currentList.Width()
	listHeight := currentList.Height()

	// Create new list with custom delegate but preserve dimensions
	listModel := list.New(listItems, delegate, listWidth, listHeight)
	listModel.Styles = styles.ListStyles(s, isDark)
	listModel.InfiniteScrolling = true
	listModel.SetShowHelp(false)
	listModel.SetShowTitle(false)
	listModel.DisableQuitKeybindings()
	*baseModel.GetList() = listModel

	// Set the custom list selection handler
	baseModel.OnListSelection = func(m *splitview.Model[snippetItem, *storeAdapter], listItem list.Item) {
		if i, ok := listItem.(item); ok {
			m.GetStore().SetCurrentName(i.snippet.Name)
		}

		// Update editor with current content
		current := m.GetStore().GetCurrent()
		m.GetEditor().SetContent(current.GetContent())
	}

	m := Model{
		Model:  baseModel,
		server: server,
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return m.Model.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case whichkey.SnippetEditorMsg:
		updatedBase, cmd := m.OpenInExternalEditor()
		m.Model = &updatedBase
		return m, cmd

	case tea.KeyMsg:
		if m.GetList().FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keymap.Editor):
			if m.GetEditor().IsNormalMode() {
				return m, utils.Dispatch(whichkey.SnippetEditorCmd())
			}

		case key.Matches(msg, keymap.Submit):
			// Handle Enter key to select snippet
			selected := m.GetList().SelectedItem()
			if selected != nil {
				if item, ok := selected.(item); ok {
					return m, utils.Dispatch(SelectedMsg{
						Snippet: item.snippet,
					})
				}
			}
		}
	}

	// Delegate to base model
	updatedModel, cmd := m.Model.Update(msg)
	m.Model = &updatedModel
	return m, cmd
}

func (m Model) View() string {
	return m.Model.View()
}

func processSnippets(snippets []snippetItem) []list.Item {
	items := make([]list.Item, 0, len(snippets))

	for _, snippet := range snippets {
		items = append(items, item{
			snippet: *snippet.Snippet,
		})
	}

	return items
}

func renderStatusBar(m *splitview.Model[snippetItem, *storeAdapter], server server.Server, width int) string {
	bg := m.Styles.Surface0.GetBackground()

	separator := m.Styles.Surface0.Render(" | ")

	current := m.GetStore().GetCurrent()
	scopeLabel := "Global"
	if current.Scope == snippets.ScopeServer {
		scopeLabel = server.Name
	}

	scope := m.Styles.Primary.Background(bg).Render(scopeLabel)

	snippetName := m.Styles.Accent.Background(bg).Render(strings.TrimSuffix(current.Name, ".sql"))

	left := scope + separator + snippetName

	leftInfo := m.Styles.Surface0.Padding(0, 1).Render(left)

	helpText := m.Styles.Info.Background(bg).PaddingRight(1).Render("<leader>? Help")

	displayedInfoWidth := width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := m.Styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return m.Styles.Surface0.Width(width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}

func renderHelp(m *splitview.Model[snippetItem, *storeAdapter]) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		renderUsefulHelp(m),
		splitview.RenderCommonListHelp(m.Styles, m.GetWidth(), *m.GetList()),
		splitview.RenderCommonEditorHelp(m.Styles, m.GetWidth()),
	)
}

func renderUsefulHelp(m *splitview.Model[snippetItem, *storeAdapter]) string {
	bindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("<leader>c"),
			key.WithHelp("leader>c", "go back to the main view"),
		),
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select snippet"),
		),
		keymap.ForceQuit,
		splitview.ChangeFocused,
		keymap.Editor,
	}

	return splitview.RenderCommonUsefulHelp(m.Styles, m.GetWidth(), bindings)
}

func (m Model) CanTriggerLeaderKey() bool {
	return m.Model.CanTriggerLeaderKey()
}

func (m *Model) HandleHelpToggle() {
	m.Model.HandleHelpToggle()
}

func (m *Model) GetCurrentSnippet() *snippets.Snippet {
	selected := m.GetList().SelectedItem()
	if selected != nil {
		if item, ok := selected.(item); ok {
			return &item.snippet
		}
	}
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.Model.Update(tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	})
}
