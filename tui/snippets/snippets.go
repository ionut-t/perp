package snippets

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/store/snippets"
	"github.com/ionut-t/perp/ui/help"
)

var (
	splitViewSeparator      = " "
	splitViewSeparatorWidth = lipgloss.Width(splitViewSeparator)
	minListWidth            = 50
)

type clearMsg struct{}

type editorClosedMsg struct{}

func clearMessages() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return clearMsg{}
	})
}

type view int

const (
	viewSplit view = iota
	viewList
	viewSnippet
	viewHelp
	viewPlaceholder
)

type focusedView int

const (
	focusedViewList focusedView = iota
	focusedViewSnippet
)

type SelectedMsg struct {
	Snippet snippets.Snippet
}

type DeleteMsg struct {
	Snippet snippets.Snippet
}

type Model struct {
	store snippets.Store

	width, height int

	server server.Server

	view           view
	focusedView    focusedView
	error          error
	list           list.Model
	editor         editor.Model
	successMessage string
	help           help.Model
}

type item struct {
	snippet snippets.Snippet
}

func (i item) Title() string {
	// Add scope indicator
	prefix := "🌍 " // Global
	if i.snippet.Scope == snippets.ScopeServer {
		prefix = "🖥️  " // Server-specific
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

	_, _ = fmt.Fprintf(w, "%s\n%s", fn(title), descFn(desc))
}

func New(store snippets.Store, server server.Server, width, height int) Model {
	snippetList, err := store.Load()

	delegate := itemDelegate{
		styles: styles.ListItemStyles(),
	}

	textEditor := editor.New(80, 20)
	textEditor.WithTheme(styles.EditorTheme())
	textEditor.SetLanguage("postgres", styles.EditorLanguageTheme())

	if len(snippetList) > 0 {
		textEditor.SetContent(snippetList[0].Content)
	}

	items := processSnippets(snippetList)

	ls := list.New(items, delegate, 80, 20)

	ls.Styles = styles.ListStyles()

	ls.FilterInput.PromptStyle = styles.Accent
	ls.FilterInput.Cursor.Style = styles.Accent

	ls.InfiniteScrolling = true
	ls.SetShowHelp(false)
	ls.SetShowTitle(false)
	ls.DisableQuitKeybindings()

	view := viewSplit
	if len(items) == 0 {
		view = viewPlaceholder
	}

	m := Model{
		store:  store,
		error:  err,
		list:   ls,
		editor: textEditor,
		help:   help.New(),
		view:   view,
		server: server,
	}

	m.handleWindowSize(tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	})

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keymap.Quit), key.Matches(msg, keymap.Cancel):
			if m.view == viewHelp {
				m.view = viewSplit
				return m, nil
			}

		case key.Matches(msg, keymap.Insert):
			if m.view == viewHelp || m.view == viewPlaceholder {
				break
			}

			if m.focusedView == focusedViewList {
				m.focusedView = focusedViewSnippet
				m.editor.Focus()
				_ = m.editor.SetCursorPosition(0, 0)
				m.editor.SetInsertMode()
				return m, nil
			}

		case key.Matches(msg, changeFocused):
			if m.view == viewHelp || m.view == viewPlaceholder {
				break
			}

			if m.view == viewSplit && !m.editor.IsInsertMode() {
				if m.focusedView == focusedViewList {
					m.focusedView = focusedViewSnippet
					m.editor.Focus()
					_ = m.editor.SetCursorPosition(0, 0)
					m.editor.SetNormalMode()
				} else {
					m.focusedView = focusedViewList
					m.editor.Blur()
				}
			}

		case key.Matches(msg, keymap.Editor):
			return m, utils.Dispatch(whichkey.SnippetEditorCmd())

		case key.Matches(msg, keymap.Submit):
			if m.view == viewHelp || m.view == viewPlaceholder {
				break
			}

			if m.focusedView == focusedViewList {
				selected := m.list.SelectedItem()
				if selected != nil {
					if item, ok := selected.(item); ok {
						return m, utils.Dispatch(SelectedMsg{
							Snippet: item.snippet,
						})
					}
				}
			}
		}

	case whichkey.SnippetEditorMsg:
		if m.view == viewPlaceholder {
			break
		}

		return m.openInExternalEditor()

	case editorClosedMsg:
		return m.handleEditorClose()

	case editor.SaveMsg:
		current := m.store.GetCurrentSnippet()
		current.Content = string(msg.Content)
		err := m.store.Update(current)
		if err != nil {
			m.error = fmt.Errorf("failed to save snippet: %w", err)
		} else {
			m.error = nil
			snippetList, err := m.store.Load()
			m.error = err

			if err == nil {
				items := processSnippets(snippetList)
				m.list.SetItems(items)
			}
		}

	case editor.QuitMsg:
		return m, utils.Dispatch(whichkey.CloseSnippetsCmd())

	case editor.DeleteFileMsg:
		current := m.store.GetCurrentSnippet()

		var cmd tea.Cmd

		if err := m.store.Delete(current); err == nil {
			m.error = nil
			m.successMessage = "Snippet deleted successfully."
			snippetList, err := m.store.Load()
			m.error = err

			if err == nil {
				items := processSnippets(snippetList)
				m.list.SetItems(items)
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					m.store.SetCurrentSnippetName(selectedItem.snippet.Name)
				}
			}

			if len(snippetList) > 0 {
				current = m.store.GetCurrentSnippet()
				m.editor.SetContent(current.Content)
			} else {
				m.editor.SetContent("")
				m.view = viewPlaceholder
			}

			var textEditor tea.Model
			textEditor, cmd = m.editor.Update(nil)
			m.editor = textEditor.(editor.Model)
			m.editor.SetLanguage("postgres", styles.EditorLanguageTheme())

		} else {
			m.error = fmt.Errorf("failed to delete snippet: %w", err)
		}

		return m, tea.Batch(
			cmd,
			clearMessages(),
		)

	case editor.RenameMsg:
		current := m.store.GetCurrentSnippet()

		oldSnippetName := current.Name
		newName := msg.FileName

		if newName == oldSnippetName {
			return m, nil
		}

		if err := m.store.Rename(&current, newName); err == nil {
			m.successMessage = "Snippet renamed successfully."

			return m, tea.Batch(
				m.list.SetItem(m.list.Index(), item{
					snippet: current,
				}),
				clearMessages(),
			)

		} else {
			m.error = fmt.Errorf("failed to rename snippet: %w", err)
		}

		return m, clearMessages()

	case clearMsg:
		m.successMessage = ""
		m.error = nil
	}

	if m.view == viewHelp {
		hp, cmd := m.help.Update(msg)
		m.help = hp.(help.Model)

		return m, cmd
	}

	var cmds []tea.Cmd

	if m.focusedView == focusedViewList {
		ls, cmd := m.list.Update(msg)
		m.list = ls
		if selectedItem, ok := m.list.SelectedItem().(item); ok {
			m.store.SetCurrentSnippetName(selectedItem.snippet.Name)
		}
		current := m.store.GetCurrentSnippet()
		m.editor.SetContent(current.Content)

		cmds = append(cmds, cmd)
	}

	em, cmd := m.editor.Update(msg)
	m.editor = em.(editor.Model)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.error != nil {
		return styles.Error.Render(m.error.Error())
	}

	switch m.view {
	case viewList:
		return styles.ViewPadding.Render(m.list.View()) + "\n" + m.statusBarView()

	case viewSnippet:
		return m.editor.View()

	case viewSplit:
		return m.getSplitView()

	case viewHelp:
		return m.help.View()

	case viewPlaceholder:
		return styles.ViewPadding.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.Primary.Render("No snippets available."),
				"\n",
				styles.Subtext0.Render("Press '<leader>ns' to save a snippet or '<leader>c' to go back."),
			),
		)

	default:
		return ""
	}
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.help.SetSize(msg.Width, msg.Height)
	m.help.SetContent(m.renderHelp())

	if msg.Width < 2*minListWidth {
		switch m.view {
		case viewSplit:
			m.view = viewList
		case viewList:
			m.view = viewSplit
		}
	}

	m.width, m.height = msg.Width, msg.Height

	availableWidth, availableHeight := m.getAvailableSizes()

	if m.view == viewList {
		m.list.SetSize(availableWidth, availableHeight)
	}

	if m.view == viewSnippet {
		m.editor.SetSize(msg.Width, msg.Height)
	}

	if m.view == viewSplit {
		listWidth := min(availableWidth/2, minListWidth)

		m.list.SetHeight(availableHeight)
		m.list.SetWidth(listWidth)

		m.editor.SetSize(availableWidth-listWidth, availableHeight)
	}
}

func (m *Model) getAvailableSizes() (int, int) {
	h, v := styles.ViewPadding.GetFrameSize()

	statusBarHeight := lipgloss.Height(m.statusBarView())

	availableHeight := m.height - v - statusBarHeight - styles.ActiveBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	return availableWidth, availableHeight
}

func (m *Model) getSplitView() string {
	horizontalFrameSize := styles.ViewPadding.GetHorizontalFrameSize()
	horizontalFrameBorderSize := styles.ActiveBorder.GetHorizontalFrameSize()

	availableWidth := m.width - horizontalFrameSize

	listWidth := min(minListWidth, availableWidth/2) - horizontalFrameBorderSize*2 - splitViewSeparatorWidth
	snippetWidth := availableWidth - listWidth - horizontalFrameBorderSize*2 - splitViewSeparatorWidth

	var joinedContent string

	if m.focusedView == focusedViewList {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.ActiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			styles.InactiveBorder.
				Width(snippetWidth).
				Height(m.list.Height()).
				Render(m.editor.View()),
		)
	} else {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.InactiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			styles.ActiveBorder.
				Width(snippetWidth).
				Height(m.list.Height()).
				Render(m.editor.View()),
		)
	}

	renderedView := styles.ViewPadding.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		joinedContent,
	))

	return renderedView + "\n" + m.statusBarView()
}

func (m *Model) statusBarView() string {
	if m.error != nil {
		return styles.Error.Margin(0, 2).Render(m.error.Error())
	}

	if m.successMessage != "" {
		return styles.Success.Margin(0, 2).Render(m.successMessage)
	}

	return lipgloss.NewStyle().Margin(0, 1).
		Render(m.renderStatusBar())
}

func processSnippets(snippets []snippets.Snippet) []list.Item {
	items := make([]list.Item, 0, len(snippets))

	for _, snippet := range snippets {
		items = append(items, item{
			snippet: snippet,
		})
	}

	return items
}

func (m *Model) renderStatusBar() string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	current := m.store.GetCurrentSnippet()
	scopeLabel := "Global"
	if current.Scope == snippets.ScopeServer {
		scopeLabel = m.server.Name
	}

	scope := styles.Primary.Background(bg).Render(scopeLabel)

	snippetName := styles.Accent.Background(bg).Render(strings.TrimSuffix(current.Name, ".sql"))

	left := scope + separator + snippetName

	leftInfo := styles.Surface0.Padding(0, 1).Render(left)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("<leader>? Help")

	displayedInfoWidth := m.width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}

func (m Model) CanTriggerLeaderKey() bool {
	return m.editor.IsNormalMode() && m.list.FilterState() != list.Filtering
}

func (m *Model) HandleHelpToggle() {
	if !m.editor.IsNormalMode() ||
		m.view == viewPlaceholder {
		return
	}

	switch m.view {
	case viewSplit:
		m.view = viewHelp
		m.editor.Blur()
	case viewHelp:
		m.view = viewSplit
	}
}

func (m Model) openInExternalEditor() (tea.Model, tea.Cmd) {
	path := m.store.GetCurrentSnippet().Path
	execCmd := tea.ExecProcess(exec.Command(m.store.Editor(), path), func(error) tea.Msg {
		return editorClosedMsg{}
	})
	return m, execCmd
}

func (m Model) handleEditorClose() (Model, tea.Cmd) {
	snippetList, err := m.store.Load()
	if err != nil {
		m.error = err
		return m, nil
	}

	m.list.SetItems(processSnippets(snippetList))

	current := m.store.GetCurrentSnippet()
	m.editor.SetContent(current.Content)

	m.list.ResetFilter()

	textEditor, cmd := m.editor.Update(nil)
	m.editor = textEditor.(editor.Model)

	return m, cmd
}

func (m *Model) GetCurrentSnippet() *snippets.Snippet {
	selected := m.list.SelectedItem()
	if selected != nil {
		if item, ok := selected.(item); ok {
			return &item.snippet
		}
	}
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	w, h := m.getAvailableSizes()
	lsWidth := max(50, w/3)

	m.list.SetSize(lsWidth, h)

	vpW := max(1, w-lsWidth-styles.ViewPadding.GetHorizontalFrameSize()-2)
	m.editor.SetSize(vpW, max(1, h))
}
